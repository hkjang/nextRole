package server

import (
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
	"time"
)

func (a *App) public(w http.ResponseWriter, r *http.Request) {
	s, e := a.settings(r.Context())
	if !a.good(w, e) {
		return
	}
	ps := []map[string]string{}
	providers, e := a.providers(r)
	if !a.good(w, e) {
		return
	}
	for _, p := range providers {
		if p.Enabled {
			ps = append(ps, map[string]string{"id": p.ID, "name": p.Name, "type": p.Type})
		}
	}
	respond(w, 200, map[string]any{"name": s.General.Name, "version": a.version, "registrationEnabled": s.General.RegistrationEnabled, "approvalEnabled": s.General.ApprovalEnabled, "demoEnabled": s.General.DemoEnabled, "providers": ps})
}
func (a *App) session(w http.ResponseWriter, r *http.Request, u User) bool {
	s, e := a.settings(r.Context())
	if !a.good(w, e) {
		return false
	}
	v := token()
	duration := time.Duration(s.Security.SessionHours) * time.Hour
	_, e = a.db.Exec(r.Context(), "INSERT INTO nr_sessions(hash,user_id,expires_at) VALUES($1,$2,$3)", digest(v), u.ID, time.Now().Add(duration))
	if !a.good(w, e) {
		return false
	}
	http.SetCookie(w, &http.Cookie{Name: "nr_session", Value: v, Path: "/", HttpOnly: true, Secure: strings.HasPrefix(s.General.BaseURL, "https://") || r.TLS != nil, SameSite: http.SameSiteLaxMode, MaxAge: int(duration.Seconds())})
	_, _ = a.db.Exec(r.Context(), "DELETE FROM nr_sessions WHERE expires_at<now()")
	return true
}
func (a *App) login(w http.ResponseWriter, r *http.Request) {
	if !a.rate(r) {
		fail(w, 429, "로그인 요청이 많습니다. 1분 뒤 다시 시도하세요")
		return
	}
	var in struct{ Email, Password string }
	if !readJSON(w, r, &in) {
		return
	}
	var uid, hash string
	err := a.db.QueryRow(r.Context(), "SELECT id,password_hash FROM nr_users WHERE email=$1 AND disabled=false", strings.ToLower(strings.TrimSpace(in.Email))).Scan(&uid, &hash)
	if err != nil {
		hash = "$2a$10$7EqJtq98hPqEX7fNZaFWoOhiSCp/zpxk.Vs/q6LiUd52tkFCU79bC"
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil || err != nil {
		fail(w, 401, "계정 또는 비밀번호를 확인하세요")
		return
	}
	u, e := a.user(r.Context(), uid)
	if !a.good(w, e) {
		return
	}
	if !a.session(w, r, u) {
		return
	}
	a.audit(r.Context(), u.ID, "auth.login", u.ID)
	respond(w, 200, map[string]any{"user": u})
}
func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if c, e := r.Cookie("nr_session"); e == nil {
		_, _ = a.db.Exec(r.Context(), "DELETE FROM nr_sessions WHERE hash=$1", digest(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: "nr_session", Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	ok(w)
}
func (a *App) register(w http.ResponseWriter, r *http.Request) {
	if !a.rate(r) {
		fail(w, 429, "요청이 많습니다. 잠시 뒤 시도하세요")
		return
	}
	s, e := a.settings(r.Context())
	if !a.good(w, e) {
		return
	}
	if !s.General.RegistrationEnabled {
		fail(w, 403, "관리자가 회원가입을 허용하지 않았습니다")
		return
	}
	var in struct{ Email, Password, Name string }
	if !readJSON(w, r, &in) {
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if len(in.Email) > 254 || !strings.Contains(in.Email, "@") || strings.TrimSpace(in.Name) == "" || len(in.Name) > 200 || len(in.Password) < 12 || len(in.Password) > 72 {
		fail(w, 400, "이메일·이름과 12~72바이트 비밀번호를 입력하세요")
		return
	}
	h, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	uid := id()
	_, e = a.db.Exec(r.Context(), "INSERT INTO nr_users(id,email,name,role,password_hash) VALUES($1,$2,$3,'user',$4)", uid, in.Email, in.Name, string(h))
	if e != nil {
		fail(w, 409, "이미 등록된 계정이거나 등록할 수 없는 값입니다")
		return
	}
	u, e := a.user(r.Context(), uid)
	if !a.good(w, e) {
		return
	}
	if a.session(w, r, u) {
		a.audit(r.Context(), uid, "auth.register", uid)
		respond(w, 201, map[string]any{"user": u})
	}
}
func (a *App) me(w http.ResponseWriter, r *http.Request) { respond(w, 200, who(r).User) }
func (a *App) updateMe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name        string         `json:"name"`
		Preferences map[string]any `json:"preferences"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 200 {
		fail(w, 400, "이름을 확인하세요")
		return
	}
	b, _ := json.Marshal(in.Preferences)
	if len(b) > 10000 {
		fail(w, 400, "개인 설정 크기를 초과했습니다")
		return
	}
	_, e := a.db.Exec(r.Context(), "UPDATE nr_users SET name=$1,preferences=$2 WHERE id=$3", in.Name, b, who(r).User.ID)
	if !a.good(w, e) {
		return
	}
	u, e := a.user(r.Context(), who(r).User.ID)
	if a.good(w, e) {
		respond(w, 200, u)
	}
}
func (a *App) password(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if len(in.NewPassword) < 12 || len(in.NewPassword) > 72 {
		fail(w, 400, "새 비밀번호는 12~72바이트로 입력하세요")
		return
	}
	uid := who(r).User.ID
	var hash string
	if a.db.QueryRow(r.Context(), "SELECT password_hash FROM nr_users WHERE id=$1", uid).Scan(&hash) != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.CurrentPassword)) != nil {
		fail(w, 400, "현재 비밀번호가 일치하지 않습니다")
		return
	}
	h, _ := bcrypt.GenerateFromPassword([]byte(in.NewPassword), bcrypt.DefaultCost)
	tx, e := a.db.Begin(r.Context())
	if !a.good(w, e) {
		return
	}
	defer tx.Rollback(r.Context())
	_, e = tx.Exec(r.Context(), "UPDATE nr_users SET password_hash=$1 WHERE id=$2", string(h), uid)
	if e == nil {
		_, e = tx.Exec(r.Context(), "DELETE FROM nr_sessions WHERE user_id=$1", uid)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if !a.good(w, e) {
		return
	}
	if !a.session(w, r, who(r).User) {
		return
	}
	a.audit(r.Context(), uid, "password.change", uid)
	ok(w)
}
