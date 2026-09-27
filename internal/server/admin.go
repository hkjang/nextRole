package server

import (
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
	"time"
)

func (a *App) settingsGet(w http.ResponseWriter, r *http.Request) {
	s, e := a.settings(r.Context())
	if !a.good(w, e) {
		return
	}
	s.AI.HasSecret = s.AI.APIKey != ""
	s.AI.APIKey = ""
	respond(w, 200, s)
}
func (a *App) settingsPut(w http.ResponseWriter, r *http.Request) {
	old, e := a.settings(r.Context())
	if !a.good(w, e) {
		return
	}
	s := old
	if !readJSON(w, r, &s) {
		return
	}
	if s.AI.APIKey == "" {
		s.AI.APIKey = old.AI.APIKey
	}
	s.General.BaseURL = strings.TrimRight(s.General.BaseURL, "/")
	sum := s.Scoring.Skill + s.Scoring.Transfer + s.Scoring.Experience + s.Scoring.Domain + s.Scoring.Education + s.Scoring.Preference
	if strings.TrimSpace(s.General.Name) == "" || len(s.General.Name) > 100 || (s.General.BaseURL != "" && !validURL(s.General.BaseURL)) || s.Security.SessionHours < 1 || s.Security.SessionHours > 168 || s.Security.KeyMaxDays < 1 || s.Security.KeyMaxDays > 365 || s.AI.MaxTokens < 1 || s.AI.MaxTokens > 262144 || s.AI.ContextWindow < 1024 || s.AI.ContextWindow > 2097152 || s.AI.MaxTokens > s.AI.ContextWindow || s.AI.Temperature < 0 || s.AI.Temperature > 2 || sum < 99.99 || sum > 100.01 || s.Scoring.Skill < 0 || s.Scoring.Transfer < 0 || s.Scoring.Experience < 0 || s.Scoring.Domain < 0 || s.Scoring.Education < 0 || s.Scoring.Preference < 0 {
		fail(w, 400, "설정 범위를 확인하세요. 가중치 합계는 100, 최대 출력 토큰은 1~262144이며 컨텍스트 이하여야 합니다")
		return
	}
	if s.AI.TimeoutSeconds < 30 || s.AI.TimeoutSeconds > 14400 {
		fail(w, 400, "AI 제한 시간은 30~14400초로 입력하세요")
		return
	}
	if !contains([]string{"max_tokens", "max_completion_tokens"}, s.AI.TokenParameter) {
		fail(w, 400, "AI 출력 토큰 매개변수를 확인하세요")
		return
	}
	if s.AI.Enabled && (!validURL(s.AI.BaseURL) || s.AI.Model == "") {
		fail(w, 400, "AI 주소와 모델을 입력하세요")
		return
	}
	for _, scope := range s.Security.AllowedKeyScopes {
		if !contains(allScopes, scope) {
			fail(w, 400, "지원하지 않는 키 권한입니다")
			return
		}
	}
	if !a.good(w, a.setConfig(r.Context(), "settings", s)) {
		return
	}
	a.audit(r.Context(), who(r).User.ID, "settings.update", "settings")
	s.AI.HasSecret = s.AI.APIKey != ""
	s.AI.APIKey = ""
	respond(w, 200, s)
}
func (a *App) usersList(w http.ResponseWriter, r *http.Request) {
	rows, e := a.db.Query(r.Context(), "SELECT id FROM nr_users ORDER BY created_at DESC LIMIT 1000")
	if !a.good(w, e) {
		return
	}
	ids := []string{}
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	out := []User{}
	for _, id := range ids {
		u, e := a.user(r.Context(), id)
		if !a.good(w, e) {
			return
		}
		out = append(out, u)
	}
	respond(w, 200, out)
}
func (a *App) usersCreate(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Name, Password, Role string }
	if !readJSON(w, r, &in) {
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.Role == "" {
		in.Role = "user"
	}
	if in.Email == "" || len(in.Email) > 254 || in.Name == "" || len(in.Name) > 200 || len(in.Password) < 12 || len(in.Password) > 72 || !contains([]string{"user", "admin", "reviewer"}, in.Role) {
		fail(w, 400, "계정 정보를 확인하세요. 비밀번호는 12~72바이트입니다")
		return
	}
	h, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	uid := id()
	_, e := a.db.Exec(r.Context(), "INSERT INTO nr_users(id,email,name,role,password_hash) VALUES($1,$2,$3,$4,$5)", uid, in.Email, in.Name, in.Role, string(h))
	if e != nil {
		fail(w, 409, "이미 등록된 계정입니다")
		return
	}
	a.audit(r.Context(), who(r).User.ID, "user.create", uid)
	u, e := a.user(r.Context(), uid)
	if a.good(w, e) {
		respond(w, 201, u)
	}
}
func (a *App) usersUpdate(w http.ResponseWriter, r *http.Request) {
	uid := r.PathValue("id")
	u, e := a.user(r.Context(), uid)
	if e != nil {
		fail(w, 404, "사용자를 찾을 수 없습니다")
		return
	}
	in := struct {
		Name     string `json:"name"`
		Role     string `json:"role"`
		Disabled bool   `json:"disabled"`
		Password string `json:"password"`
	}{Name: u.Name, Role: u.Role, Disabled: u.Disabled}
	if !readJSON(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 200 || !contains([]string{"user", "admin", "reviewer"}, in.Role) || (in.Password != "" && (len(in.Password) < 12 || len(in.Password) > 72)) {
		fail(w, 400, "사용자 설정을 확인하세요")
		return
	}
	if uid == who(r).User.ID && (in.Role != "admin" || in.Disabled) {
		fail(w, 400, "현재 관리자의 권한을 제거할 수 없습니다")
		return
	}
	tx, e := a.db.Begin(r.Context())
	if !a.good(w, e) {
		return
	}
	defer tx.Rollback(r.Context())
	_, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(472092)")
	if !a.good(w, e) {
		return
	}
	if u.Role == "admin" && (in.Role != "admin" || in.Disabled) {
		var n int
		_ = tx.QueryRow(r.Context(), "SELECT count(*) FROM nr_users WHERE role='admin' AND disabled=false AND id<>$1", uid).Scan(&n)
		if n == 0 {
			fail(w, 400, "최소 한 명의 활성 관리자가 필요합니다")
			return
		}
	}
	_, e = tx.Exec(r.Context(), "UPDATE nr_users SET name=$1,role=$2,disabled=$3 WHERE id=$4", in.Name, in.Role, in.Disabled, uid)
	if e == nil && in.Password != "" {
		h, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		_, e = tx.Exec(r.Context(), "UPDATE nr_users SET password_hash=$1 WHERE id=$2", string(h), uid)
	}
	if e == nil && (in.Disabled || in.Password != "" || in.Role != u.Role) {
		_, e = tx.Exec(r.Context(), "DELETE FROM nr_sessions WHERE user_id=$1", uid)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if !a.good(w, e) {
		return
	}
	a.audit(r.Context(), who(r).User.ID, "user.update", uid)
	u, e = a.user(r.Context(), uid)
	if a.good(w, e) {
		respond(w, 200, u)
	}
}
func (a *App) auditList(w http.ResponseWriter, r *http.Request) {
	rows, e := a.db.Query(r.Context(), "SELECT a.id,COALESCE(u.name,a.actor),a.action,a.target,a.created_at FROM nr_audit a LEFT JOIN nr_users u ON u.id=a.actor ORDER BY a.created_at DESC LIMIT 500")
	if !a.good(w, e) {
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, actor, action, target string
		var t time.Time
		if e = rows.Scan(&id, &actor, &action, &target, &t); !a.good(w, e) {
			return
		}
		out = append(out, map[string]any{"id": id, "actor": actor, "action": action, "target": target, "createdAt": t, "details": ""})
	}
	respond(w, 200, out)
}
func (a *App) status(w http.ResponseWriter, r *http.Request) {
	var users, sims int
	if !a.good(w, a.db.QueryRow(r.Context(), "SELECT count(*) FROM nr_users").Scan(&users)) {
		return
	}
	if !a.good(w, a.db.QueryRow(r.Context(), "SELECT count(*) FROM nr_records WHERE kind='simulation'").Scan(&sims)) {
		return
	}
	s, e := a.settings(r.Context())
	if !a.good(w, e) {
		return
	}
	mode := "offline"
	if s.AI.Enabled {
		mode = "connected"
	}
	connectors, e := a.connectorConfigs(r)
	if !a.good(w, e) {
		return
	}
	respond(w, 200, map[string]any{"version": a.version, "database": "connected", "users": users, "simulations": sims, "connectors": len(connectors), "mode": mode})
}
func decodeMap(v any) map[string]any {
	b, _ := json.Marshal(v)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}
