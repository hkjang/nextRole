package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type authKey struct{}
type identity struct {
	User   User
	Scopes []string
	KeyID  string
}

func who(r *http.Request) identity { v, _ := r.Context().Value(authKey{}).(identity); return v }
func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}
func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	d := json.NewDecoder(r.Body)
	if err := d.Decode(v); err != nil {
		fail(w, 400, "입력 JSON 형식이나 크기를 확인하세요")
		return false
	}
	if d.Decode(&struct{}{}) != io.EOF {
		fail(w, 400, "JSON 객체 하나만 입력하세요")
		return false
	}
	return true
}
func ok(w http.ResponseWriter) { respond(w, 200, map[string]bool{"ok": true}) }
func (a *App) good(w http.ResponseWriter, err error) bool {
	if err != nil {
		log.Printf("database operation failed: %T", err)
		fail(w, 500, "저장 또는 조회에 실패했습니다. 잠시 후 다시 시도하세요")
		return false
	}
	return true
}
func contains(a []string, v string) bool {
	for _, s := range a {
		if s == v {
			return true
		}
	}
	return false
}
func (a *App) Handler() http.Handler {
	m := http.NewServeMux()
	p := "/api/v1"
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if a.db.Ping(ctx) != nil {
			fail(w, 503, "database unavailable")
			return
		}
		respond(w, 200, map[string]string{"status": "ok", "version": a.version})
	})
	m.HandleFunc("GET "+p+"/public", a.public)
	m.HandleFunc("POST "+p+"/auth/login", a.login)
	m.HandleFunc("POST "+p+"/auth/register", a.register)
	m.HandleFunc("POST "+p+"/auth/logout", a.logout)
	m.HandleFunc("GET "+p+"/auth/sso/{id}/start", a.ssoStart)
	m.HandleFunc("GET "+p+"/auth/sso/{id}/callback", a.ssoCallback)
	route := func(pattern, scope string, fn http.HandlerFunc) { m.Handle(pattern, a.auth(scope, false, fn)) }
	admin := func(pattern string, fn http.HandlerFunc) { m.Handle(pattern, a.auth("", true, fn)) }
	route("GET "+p+"/me", "", a.me)
	route("PUT "+p+"/me", "", a.updateMe)
	route("POST "+p+"/me/password", "", a.password)
	route("GET "+p+"/profile", "profile:read", a.profileGet)
	route("PUT "+p+"/profile", "profile:write", a.profilePut)
	route("POST "+p+"/profile/parse", "profile:write", a.profileParse)
	route("POST "+p+"/profile/upload", "profile:write", a.profileUpload)
	route("GET "+p+"/jobs", "jobs:read", a.jobsGet)
	route("GET "+p+"/recommendations", "profile:read", a.recommendations)
	route("POST "+p+"/simulate", "simulate:write", a.simulate)
	route("GET "+p+"/simulations", "profile:read", a.simulations)
	route("DELETE "+p+"/simulations/{id}", "simulate:write", a.simulationDelete)
	route("GET "+p+"/opportunities", "jobs:read", a.opportunitiesGet)
	route("GET "+p+"/market", "jobs:read", a.marketGet)
	route("POST "+p+"/ai/stream", "ai:use", a.aiStream)
	route("POST "+p+"/feedback", "profile:write", a.feedback)
	route("GET "+p+"/roadmap", "profile:read", a.roadmapGet)
	route("PUT "+p+"/roadmap", "profile:write", a.roadmapPut)
	route("GET "+p+"/keys", "", a.keysList)
	route("POST "+p+"/keys", "", a.keysCreate)
	route("PUT "+p+"/keys/{id}", "", a.keysUpdate)
	route("POST "+p+"/keys/{id}/rotate", "", a.keysRotate)
	route("DELETE "+p+"/keys/{id}", "", a.keysDelete)
	route("GET "+p+"/key-scopes", "", a.keyScopes)
	route("GET "+p+"/approvals", "", a.approvalsList)
	route("POST "+p+"/approvals", "", a.approvalsCreate)
	route("PUT "+p+"/approvals/{id}", "", a.approvalsUpdate)
	admin("GET "+p+"/admin/settings", a.settingsGet)
	admin("PUT "+p+"/admin/settings", a.settingsPut)
	admin("GET "+p+"/admin/users", a.usersList)
	admin("POST "+p+"/admin/users", a.usersCreate)
	admin("PUT "+p+"/admin/users/{id}", a.usersUpdate)
	admin("GET "+p+"/admin/providers", a.providersList)
	admin("POST "+p+"/admin/providers", a.providersSave)
	admin("PUT "+p+"/admin/providers/{id}", a.providersSave)
	admin("DELETE "+p+"/admin/providers/{id}", a.providersDelete)
	admin("GET "+p+"/admin/connectors", a.connectorsList)
	admin("POST "+p+"/admin/connectors", a.connectorsSave)
	admin("PUT "+p+"/admin/connectors/{id}", a.connectorsSave)
	admin("DELETE "+p+"/admin/connectors/{id}", a.connectorsDelete)
	admin("POST "+p+"/admin/connectors/{id}/test", a.connectorsTest)
	admin("POST "+p+"/admin/connectors/{id}/sync", a.connectorsSync)
	admin("POST "+p+"/admin/import", a.dataImport)
	admin("GET "+p+"/admin/audit", a.auditList)
	admin("GET "+p+"/admin/status", a.status)
	admin("POST "+p+"/admin/ai/test", a.aiTest)
	route("POST /mcp", "mcp:use", a.mcp)
	route("POST "+p+"/mcp", "mcp:use", a.mcp)

	for _, endpoint := range []string{"/mcp", p + "/mcp"} {
		for _, method := range []string{"GET", "DELETE"} {
			route(method+" "+endpoint, "mcp:use", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Allow", "POST")
				fail(w, http.StatusMethodNotAllowed, "이 MCP 서버는 상태 없는 POST 전송을 지원합니다")
			})
		}
	}
	m.HandleFunc("GET "+p+"/openapi.json", a.openapi)
	m.HandleFunc("/api/v1/", func(w http.ResponseWriter, r *http.Request) { fail(w, 404, "API를 찾을 수 없습니다") })
	m.HandleFunc("/", a.frontend)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/mcp" {
			w.Header().Set("Cache-Control", "no-store")
		}
		defer func() {
			if e := recover(); e != nil {
				log.Printf("request panic: %T", e)
				fail(w, 500, "요청을 처리할 수 없습니다")
			}
		}()
		// Cookies require same-origin writes. Bearer requests remain subject to Origin checks when supplied.
		if (r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS") || r.URL.Path == "/mcp" || r.URL.Path == "/api/v1/mcp" {
			origin := r.Header.Get("Origin")
			if origin != "" {
				u, err := url.Parse(origin)
				s, _ := a.settings(r.Context())
				scheme := "http"
				if r.TLS != nil {
					scheme = "https"
				}
				expected := scheme + "://" + r.Host
				if s.General.BaseURL != "" {
					if base, parseErr := url.Parse(s.General.BaseURL); parseErr == nil {
						expected = base.Scheme + "://" + base.Host
					}
				}
				valid := err == nil && u.User == nil && u.Path == "" && u.RawQuery == "" && u.Fragment == "" && origin == expected
				if !valid {
					fail(w, 403, "다른 출처의 요청은 허용되지 않습니다")
					return
				}
			} else if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				fail(w, 403, "다른 출처의 요청은 허용되지 않습니다")
				return
			}
		}
		m.ServeHTTP(w, r)
	})
}
func (a *App) frontend(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		fail(w, 405, "허용되지 않은 메서드입니다")
		return
	}
	rel := filepath.Clean("/" + r.URL.Path)
	path := filepath.Join("web/dist", rel)
	info, err := os.Stat(path)
	if err == nil && !info.IsDir() {
		http.ServeFile(w, r, path)
		return
	}
	if strings.Contains(filepath.Base(rel), ".") {
		http.NotFound(w, r)
		return
	}
	if _, err := os.Stat("web/dist/index.html"); err != nil {
		fail(w, 503, "프런트엔드 번들을 먼저 빌드하세요")
		return
	}
	http.ServeFile(w, r, "web/dist/index.html")
}
func (a *App) auth(scope string, admin bool, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ident identity
		var uid string
		bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if bearer != "" && bearer != r.Header.Get("Authorization") {
			var b []byte
			err := a.db.QueryRow(r.Context(), "SELECT id,user_id,scopes FROM nr_keys WHERE hash=$1 AND revoked_at IS NULL AND expires_at>now()", digest(bearer)).Scan(&ident.KeyID, &uid, &b)
			if err != nil {
				fail(w, 401, "API 키가 만료되었거나 유효하지 않습니다")
				return
			}
			_ = json.Unmarshal(b, &ident.Scopes)
			s, err := a.settings(r.Context())
			if err != nil {
				fail(w, 503, "설정을 읽을 수 없습니다")
				return
			}
			if admin || scope == "" || !contains(ident.Scopes, scope) || !contains(s.Security.AllowedKeyScopes, scope) {
				fail(w, 403, "API 키 권한이 부족합니다")
				return
			}
			_, _ = a.db.Exec(r.Context(), "UPDATE nr_keys SET last_used_at=now() WHERE id=$1", ident.KeyID)
		} else {
			c, err := r.Cookie("nr_session")
			if err != nil {
				fail(w, 401, "로그인이 필요합니다")
				return
			}
			if a.db.QueryRow(r.Context(), "SELECT user_id FROM nr_sessions WHERE hash=$1 AND expires_at>now()", digest(c.Value)).Scan(&uid) != nil {
				fail(w, 401, "세션이 만료되었습니다")
				return
			}
		}
		u, err := a.user(r.Context(), uid)
		if err != nil || u.Disabled {
			fail(w, 401, "사용할 수 없는 계정입니다")
			return
		}
		if admin && u.Role != "admin" {
			fail(w, 403, "관리자 권한이 필요합니다")
			return
		}
		ident.User = u
		next(w, r.WithContext(context.WithValue(r.Context(), authKey{}, ident)))
	})
}
func (a *App) user(ctx context.Context, uid string) (User, error) {
	var u User
	var b []byte
	err := a.db.QueryRow(ctx, "SELECT id,email,name,role,disabled,preferences,created_at FROM nr_users WHERE id=$1", uid).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Disabled, &b, &u.CreatedAt)
	if err == nil {
		_ = json.Unmarshal(b, &u.Preferences)
	}
	u.Version = a.version
	return u, err
}
func (a *App) rate(r *http.Request) bool {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	key := host + ":" + r.URL.Path
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	if len(a.limits) > 5000 {
		for k, v := range a.limits {
			if now.After(v.until) {
				delete(a.limits, k)
			}
		}
	}
	e := a.limits[key]
	if now.After(e.until) {
		e = rateEntry{until: now.Add(time.Minute)}
	}
	e.count++
	a.limits[key] = e
	return e.count <= 15
}
func validURL(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" && u.User == nil && u.Fragment == ""
}

var errInvalid = errors.New("invalid input")
