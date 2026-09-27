package server

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type App struct {
	db      *pgxpool.Pool
	aead    cipher.AEAD
	version string
	mu      sync.Mutex
	limits  map[string]rateEntry
}
type rateEntry struct {
	count int
	until time.Time
}
type User struct {
	ID          string         `json:"id"`
	Email       string         `json:"email"`
	Name        string         `json:"name"`
	Role        string         `json:"role"`
	Disabled    bool           `json:"disabled"`
	Preferences map[string]any `json:"preferences"`
	Version     string         `json:"version,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
}
type Settings struct {
	General struct {
		Name                string `json:"name"`
		BaseURL             string `json:"baseUrl"`
		RegistrationEnabled bool   `json:"registrationEnabled"`
		DemoEnabled         bool   `json:"demoEnabled"`
		ApprovalEnabled     bool   `json:"approvalEnabled"`
	} `json:"general"`
	AI struct {
		TimeoutSeconds  int     `json:"timeoutSeconds"`
		TokenParameter  string  `json:"tokenParameter"`
		SendTemperature bool    `json:"sendTemperature"`
		Enabled         bool    `json:"enabled"`
		BaseURL         string  `json:"baseUrl"`
		Model           string  `json:"model"`
		APIKey          string  `json:"apiKey"`
		HasSecret       bool    `json:"hasSecret"`
		MaxTokens       int     `json:"maxTokens"`
		Temperature     float64 `json:"temperature"`
		ContextWindow   int     `json:"contextWindow"`
	} `json:"ai"`
	Security struct {
		SessionHours     int      `json:"sessionHours"`
		KeyMaxDays       int      `json:"keyMaxDays"`
		AllowedKeyScopes []string `json:"allowedKeyScopes"`
	} `json:"security"`
	Scoring struct {
		Skill      float64 `json:"skill"`
		Transfer   float64 `json:"transfer"`
		Experience float64 `json:"experience"`
		Domain     float64 `json:"domain"`
		Education  float64 `json:"education"`
		Preference float64 `json:"preference"`
	} `json:"scoring"`
}

var allScopes = []string{"profile:read", "profile:write", "jobs:read", "simulate:write", "ai:use", "mcp:use"}

func defaults() Settings {
	var s Settings
	s.General.Name = "NextRole"
	s.General.DemoEnabled = true
	s.AI.BaseURL = "http://localhost:11434/v1"
	s.AI.Model = "qwen3"
	s.AI.MaxTokens = 4096
	s.AI.TimeoutSeconds = 3600
	s.AI.ContextWindow = 262144
	s.AI.Temperature = 0.3
	s.AI.TokenParameter = "max_tokens"
	s.AI.SendTemperature = true
	s.Security.SessionHours = 12
	s.Security.KeyMaxDays = 90
	s.Security.AllowedKeyScopes = allScopes
	s.Scoring.Skill = 40
	s.Scoring.Transfer = 20
	s.Scoring.Experience = 15
	s.Scoring.Domain = 10
	s.Scoring.Education = 5
	s.Scoring.Preference = 10
	return s
}
func New(ctx context.Context, version string) (*App, error) {
	dsn, admin, password, key := os.Getenv("POSTGRES_DSN"), os.Getenv("BOOTSTRAP_ADMIN"), os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"), os.Getenv("ENCRYPTION_KEY")
	if dsn == "" || admin == "" || password == "" || key == "" {
		return nil, errors.New("POSTGRES_DSN, BOOTSTRAP_ADMIN, BOOTSTRAP_ADMIN_PASSWORD, ENCRYPTION_KEY 환경변수가 필요합니다")
	}
	if len(password) < 12 || len(password) > 72 {
		return nil, errors.New("초기 관리자 암호는 12~72바이트여야 합니다")
	}
	raw, err := base64.StdEncoding.DecodeString(key)
	if err != nil || len(raw) != 32 {
		return nil, errors.New("ENCRYPTION_KEY는 32바이트를 base64 인코딩한 값이어야 합니다")
	}
	block, _ := aes.NewCipher(raw)
	aead, _ := cipher.NewGCM(block)
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, errors.New("PostgreSQL DSN 형식이 올바르지 않습니다")
	}
	cfg.MaxConns = 16
	cfg.ConnConfig.ConnectTimeout = 10 * time.Second
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, errors.New("PostgreSQL 연결을 만들 수 없습니다")
	}
	app := &App{db: db, aead: aead, version: version, limits: map[string]rateEntry{}}
	if err = db.Ping(ctx); err != nil {
		db.Close()
		return nil, errors.New("PostgreSQL 연결에 실패했습니다. 서버 주소와 인증을 확인하세요")
	}
	if _, err = db.Exec(ctx, schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("DB 스키마 초기화: %w", err)
	}
	// Advisory lock makes first-boot initialization safe across replicas.
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(472091)"); err != nil {
		return nil, err
	}
	var exists bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM nr_config WHERE id='settings')").Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		blob, _ := json.Marshal(defaults())
		enc := app.encrypt(blob)
		_, err = tx.Exec(ctx, "INSERT INTO nr_config(id,data) VALUES('settings',$1)", enc)
		if err != nil {
			return nil, err
		}
	}
	var count int
	if err = tx.QueryRow(ctx, "SELECT count(*) FROM nr_users").Scan(&count); err != nil {
		return nil, err
	}
	if count == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		_, err = tx.Exec(ctx, "INSERT INTO nr_users(id,email,name,role,password_hash) VALUES($1,$2,'관리자','admin',$3)", id(), strings.ToLower(strings.TrimSpace(admin)), string(hash))
		if err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	if _, err = app.settings(ctx); err != nil {
		return nil, errors.New("설정 복호화 실패: 기존 ENCRYPTION_KEY를 사용하세요")
	}
	return app, nil
}
func (a *App) Close() { a.db.Close() }
func id() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func token() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func digest(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func (a *App) encrypt(b []byte) []byte {
	nonce := make([]byte, a.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		panic(err)
	}
	return a.aead.Seal(nonce, nonce, b, []byte("nextrole:v1"))
}
func (a *App) decrypt(b []byte) ([]byte, error) {
	n := a.aead.NonceSize()
	if len(b) < n {
		return nil, errors.New("invalid encrypted value")
	}
	return a.aead.Open(nil, b[:n], b[n:], []byte("nextrole:v1"))
}
func (a *App) config(ctx context.Context, key string, dst any) error {
	var b []byte
	if err := a.db.QueryRow(ctx, "SELECT data FROM nr_config WHERE id=$1", key).Scan(&b); err != nil {
		return err
	}
	b, err := a.decrypt(b)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}
func (a *App) setConfig(ctx context.Context, key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = a.db.Exec(ctx, "INSERT INTO nr_config(id,data) VALUES($1,$2) ON CONFLICT(id) DO UPDATE SET data=excluded.data", key, a.encrypt(b))
	return err
}
func (a *App) settings(ctx context.Context) (Settings, error) {
	s := defaults()
	err := a.config(ctx, "settings", &s)
	return s, err
}
func (a *App) getRecord(ctx context.Context, kind, owner, key string, dst any) error {
	var b []byte
	err := a.db.QueryRow(ctx, "SELECT data FROM nr_records WHERE kind=$1 AND owner_id=$2 AND id=$3", kind, owner, key).Scan(&b)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}
func (a *App) putRecord(ctx context.Context, kind, owner, key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if contains([]string{"profile", "simulation", "roadmap", "feedback", "approval"}, kind) {
		return a.putPrivateCareerRecord(ctx, kind, owner, key, b)
	}
	_, err = a.db.Exec(ctx, "INSERT INTO nr_records(kind,owner_id,id,data) VALUES($1,$2,$3,$4) ON CONFLICT(kind,owner_id,id) DO UPDATE SET data=excluded.data,updated_at=now()", kind, owner, key, b)
	return err
}
func (a *App) records(ctx context.Context, kind, owner string) ([]json.RawMessage, error) {
	rows, err := a.db.Query(ctx, "SELECT data FROM nr_records WHERE kind=$1 AND owner_id=$2 ORDER BY updated_at DESC LIMIT 5000", kind, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []json.RawMessage{}
	for rows.Next() {
		var b []byte
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		out = append(out, json.RawMessage(b))
	}
	return out, rows.Err()
}
func (a *App) audit(ctx context.Context, actor, action, target string) {
	_, _ = a.db.Exec(ctx, "INSERT INTO nr_audit(id,actor,action,target) VALUES($1,$2,$3,$4)", id(), actor, action, target)
}

const schema = `
CREATE TABLE IF NOT EXISTS nr_users(id text PRIMARY KEY,email text UNIQUE NOT NULL,name text NOT NULL,role text NOT NULL DEFAULT 'user' CHECK(role IN ('admin','user','reviewer')),disabled boolean NOT NULL DEFAULT false,password_hash text NOT NULL DEFAULT '',preferences jsonb NOT NULL DEFAULT '{}',created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS nr_config(id text PRIMARY KEY,data bytea NOT NULL);
CREATE TABLE IF NOT EXISTS nr_sessions(hash text PRIMARY KEY,user_id text NOT NULL REFERENCES nr_users(id) ON DELETE CASCADE,expires_at timestamptz NOT NULL);
CREATE INDEX IF NOT EXISTS nr_sessions_user_idx ON nr_sessions(user_id);
CREATE TABLE IF NOT EXISTS nr_keys(id text PRIMARY KEY,user_id text NOT NULL REFERENCES nr_users(id) ON DELETE CASCADE,name text NOT NULL,prefix text NOT NULL,hash text UNIQUE NOT NULL,scopes jsonb NOT NULL,created_at timestamptz NOT NULL DEFAULT now(),expires_at timestamptz NOT NULL,last_used_at timestamptz,revoked_at timestamptz);
CREATE TABLE IF NOT EXISTS nr_records(kind text NOT NULL,owner_id text NOT NULL,id text NOT NULL,data jsonb NOT NULL,updated_at timestamptz NOT NULL DEFAULT now(),PRIMARY KEY(kind,owner_id,id));
CREATE TABLE IF NOT EXISTS nr_audit(id text PRIMARY KEY,actor text NOT NULL,action text NOT NULL,target text NOT NULL,created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS nr_oauth_states(hash text PRIMARY KEY,data bytea NOT NULL,expires_at timestamptz NOT NULL);
CREATE TABLE IF NOT EXISTS nr_identities(provider text NOT NULL,subject text NOT NULL,user_id text NOT NULL REFERENCES nr_users(id),PRIMARY KEY(provider,subject));
`
