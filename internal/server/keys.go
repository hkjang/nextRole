package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type APIKey struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Scopes     []string   `json:"scopes"`
	CreatedAt  time.Time  `json:"createdAt"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	RevokedAt  *time.Time `json:"revokedAt"`
}

func (a *App) keysList(w http.ResponseWriter, r *http.Request) {
	rows, e := a.db.Query(r.Context(), "SELECT id,name,prefix,scopes,created_at,expires_at,last_used_at,revoked_at FROM nr_keys WHERE user_id=$1 ORDER BY created_at DESC", who(r).User.ID)
	if !a.good(w, e) {
		return
	}
	defer rows.Close()
	out := []APIKey{}
	for rows.Next() {
		var k APIKey
		var b []byte
		if !a.good(w, rows.Scan(&k.ID, &k.Name, &k.Prefix, &b, &k.CreatedAt, &k.ExpiresAt, &k.LastUsedAt, &k.RevokedAt)) {
			return
		}
		_ = json.Unmarshal(b, &k.Scopes)
		out = append(out, k)
	}
	respond(w, 200, out)
}
func (a *App) keyScopes(w http.ResponseWriter, r *http.Request) {
	s, e := a.settings(r.Context())
	if a.good(w, e) {
		respond(w, 200, map[string]any{"available": allScopes, "allowed": s.Security.AllowedKeyScopes, "maxDays": s.Security.KeyMaxDays})
	}
}
func (a *App) validScopes(r *http.Request, scopes []string) bool {
	s, e := a.settings(r.Context())
	if e != nil || len(scopes) == 0 || len(scopes) > len(allScopes) {
		return false
	}
	for _, v := range scopes {
		if !contains(allScopes, v) || !contains(s.Security.AllowedKeyScopes, v) {
			return false
		}
	}
	return true
}
func (a *App) keysCreate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name          string   `json:"name"`
		Scopes        []string `json:"scopes"`
		ExpiresInDays int      `json:"expiresInDays"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	s, e := a.settings(r.Context())
	if !a.good(w, e) {
		return
	}
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 100 || !a.validScopes(r, in.Scopes) || in.ExpiresInDays < 1 || in.ExpiresInDays > s.Security.KeyMaxDays {
		fail(w, 400, "키 이름·권한·유효기간을 확인하세요")
		return
	}
	key := "nr_" + token()
	k := APIKey{ID: id(), Name: in.Name, Prefix: key[:11], Scopes: in.Scopes, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Duration(in.ExpiresInDays) * 24 * time.Hour)}
	b, _ := json.Marshal(k.Scopes)
	_, e = a.db.Exec(r.Context(), "INSERT INTO nr_keys(id,user_id,name,prefix,hash,scopes,created_at,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)", k.ID, who(r).User.ID, k.Name, k.Prefix, digest(key), b, k.CreatedAt, k.ExpiresAt)
	if !a.good(w, e) {
		return
	}
	a.audit(r.Context(), who(r).User.ID, "key.create", k.ID)
	respond(w, 201, map[string]any{"key": key, "record": k})
}
func (a *App) keysUpdate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 100 || !a.validScopes(r, in.Scopes) {
		fail(w, 400, "키 이름과 권한을 확인하세요")
		return
	}
	b, _ := json.Marshal(in.Scopes)
	k := APIKey{ID: r.PathValue("id"), Name: in.Name, Scopes: in.Scopes}
	e := a.db.QueryRow(r.Context(), "UPDATE nr_keys SET name=$1,scopes=$2 WHERE id=$3 AND user_id=$4 AND revoked_at IS NULL AND expires_at>now() RETURNING prefix,created_at,expires_at,last_used_at", in.Name, b, k.ID, who(r).User.ID).Scan(&k.Prefix, &k.CreatedAt, &k.ExpiresAt, &k.LastUsedAt)
	if e != nil {
		fail(w, 404, "활성 API 키를 찾을 수 없습니다")
		return
	}
	a.audit(r.Context(), who(r).User.ID, "key.update", k.ID)
	respond(w, 200, k)
}
func (a *App) keysRotate(w http.ResponseWriter, r *http.Request) {
	key := "nr_" + token()
	var k APIKey
	var b []byte
	k.ID = r.PathValue("id")
	e := a.db.QueryRow(r.Context(), "UPDATE nr_keys SET hash=$1,prefix=$2,last_used_at=NULL WHERE id=$3 AND user_id=$4 AND revoked_at IS NULL AND expires_at>now() RETURNING name,scopes,created_at,expires_at", digest(key), key[:11], k.ID, who(r).User.ID).Scan(&k.Name, &b, &k.CreatedAt, &k.ExpiresAt)
	if e != nil {
		fail(w, 404, "활성 API 키를 찾을 수 없습니다")
		return
	}
	k.Prefix = key[:11]
	_ = json.Unmarshal(b, &k.Scopes)
	a.audit(r.Context(), who(r).User.ID, "key.rotate", k.ID)
	respond(w, 200, map[string]any{"key": key, "record": k})
}
func (a *App) keysDelete(w http.ResponseWriter, r *http.Request) {
	tag, e := a.db.Exec(r.Context(), "UPDATE nr_keys SET revoked_at=now() WHERE id=$1 AND user_id=$2 AND revoked_at IS NULL", r.PathValue("id"), who(r).User.ID)
	if !a.good(w, e) {
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 404, "키를 찾을 수 없습니다")
		return
	}
	a.audit(r.Context(), who(r).User.ID, "key.revoke", r.PathValue("id"))
	ok(w)
}

type Approval struct {
	Simulation   map[string]any `json:"simulation"`
	ID           string         `json:"id"`
	UserID       string         `json:"userId"`
	UserName     string         `json:"userName"`
	SimulationID string         `json:"simulationId"`
	JobTitle     string         `json:"jobTitle"`
	Note         string         `json:"note"`
	Status       string         `json:"status"`
	ReviewerID   string         `json:"reviewerId,omitempty"`
	ReviewNote   string         `json:"reviewNote,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

func (a *App) approvalEnabled(w http.ResponseWriter, r *http.Request) bool {
	s, e := a.settings(r.Context())
	if !a.good(w, e) {
		return false
	}
	if !s.General.ApprovalEnabled {
		fail(w, 404, "검토·승인 프로세스가 비활성화되어 있습니다")
		return false
	}
	return true
}
func (a *App) approvalsList(w http.ResponseWriter, r *http.Request) {
	if !a.approvalEnabled(w, r) {
		return
	}
	rs, e := a.records(r.Context(), "approval", "")
	if !a.good(w, e) {
		return
	}
	out := []Approval{}
	for _, b := range rs {
		var v Approval
		_ = json.Unmarshal(b, &v)
		if who(r).User.Role != "user" || v.UserID == who(r).User.ID {
			out = append(out, v)
		}
	}
	respond(w, 200, out)
}
func (a *App) approvalsCreate(w http.ResponseWriter, r *http.Request) {
	if !a.approvalEnabled(w, r) {
		return
	}
	var in struct {
		SimulationID string `json:"simulationId"`
		Note         string `json:"note"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if len(in.Note) > 4000 {
		fail(w, 400, "검토 요청 내용이 너무 깁니다")
		return
	}
	var sim map[string]any
	if a.getRecord(r.Context(), "simulation", who(r).User.ID, in.SimulationID, &sim) != nil {
		fail(w, 404, "내가 저장한 시뮬레이션을 선택하세요")
		return
	}
	title := claimString(sim, "job.title")
	v := Approval{Simulation: sim, ID: id(), UserID: who(r).User.ID, UserName: who(r).User.Name, SimulationID: in.SimulationID, JobTitle: title, Note: in.Note, Status: "pending", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if a.good(w, a.putRecord(r.Context(), "approval", "", v.ID, v)) {
		a.audit(r.Context(), v.UserID, "approval.request", v.ID)
		respond(w, 201, v)
	}
}
func (a *App) approvalsUpdate(w http.ResponseWriter, r *http.Request) {
	if !a.approvalEnabled(w, r) {
		return
	}
	if who(r).User.Role == "user" {
		fail(w, 403, "검토자 권한이 필요합니다")
		return
	}
	var in struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if !contains([]string{"approved", "rejected"}, in.Status) || len(in.Note) > 4000 {
		fail(w, 400, "승인 또는 반려 상태와 검토 내용을 확인하세요")
		return
	}
	tx, e := a.db.Begin(r.Context())
	if !a.good(w, e) {
		return
	}
	defer tx.Rollback(r.Context())
	var b []byte
	e = tx.QueryRow(r.Context(), "SELECT data FROM nr_records WHERE kind='approval' AND owner_id='' AND id=$1 FOR UPDATE", r.PathValue("id")).Scan(&b)
	if e != nil {
		fail(w, 404, "검토 요청을 찾을 수 없습니다")
		return
	}
	var v Approval
	_ = json.Unmarshal(b, &v)
	if v.UserID == who(r).User.ID {
		fail(w, 403, "자신의 요청을 검토할 수 없습니다")
		return
	}
	if v.Status != "pending" {
		fail(w, 409, "이미 검토된 요청입니다")
		return
	}
	v.Status = in.Status
	v.ReviewNote = in.Note
	v.ReviewerID = who(r).User.ID
	v.UpdatedAt = time.Now().UTC()
	b, _ = json.Marshal(v)
	_, e = tx.Exec(r.Context(), "UPDATE nr_records SET data=$1,updated_at=now() WHERE kind='approval' AND owner_id='' AND id=$2", b, v.ID)
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if a.good(w, e) {
		a.audit(r.Context(), who(r).User.ID, "approval."+v.Status, v.ID)
		respond(w, 200, v)
	}
}
