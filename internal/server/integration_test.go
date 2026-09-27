package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

// Each integration run has a private schema, so tests never mutate service data.
func TestIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is not set")
	}
	ctx := context.Background()
	base, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer base.Close()
	schema := "test_" + id()
	if _, e = base.Exec(ctx, "CREATE SCHEMA "+schema); e != nil {
		t.Fatal(e)
	}
	defer base.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
	u, e := url.Parse(dsn)
	if e != nil {
		t.Fatal(e)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	t.Setenv("POSTGRES_DSN", u.String())
	t.Setenv("BOOTSTRAP_ADMIN", "admin@example.test")
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "Test-NextRole-Password42")
	t.Setenv("ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{42}, 32)))
	app, e := New(ctx, "test")
	if e != nil {
		t.Fatal(e)
	}
	defer app.Close()
	ts := httptest.NewServer(app.Handler())
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	call := func(method, path string, body any, key string, want int) map[string]any {
		t.Helper()
		var b []byte
		if body != nil {
			b, _ = json.Marshal(body)
		}
		req, _ := http.NewRequest(method, ts.URL+path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		if key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
		res, e := client.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		if res.StatusCode != want {
			t.Fatalf("%s %s: %d want %d %s", method, path, res.StatusCode, want, raw)
		}
		var out map[string]any
		_ = json.Unmarshal(raw, &out)
		return out
	}
	call("GET", "/api/v1/me", nil, "", 401)
	login := call("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Test-NextRole-Password42"}, "", 200)
	uid := login["user"].(map[string]any)["id"].(string)
	call("GET", "/api/v1/admin/status", nil, "", 200)
	req, _ := http.NewRequest("PUT", ts.URL+"/api/v1/me", strings.NewReader(`{"name":"attack"}`))
	req.Header.Set("Origin", "https://other.example")
	res, e := client.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	res.Body.Close()
	if res.StatusCode != 403 {
		t.Fatal("cross origin write accepted")
	}
	call("PUT", "/api/v1/profile", map[string]any{"name": "테스트", "yearsExperience": 10, "weeklyHours": 10, "skills": []map[string]any{{"name": "Java", "level": 4, "confidence": "explicit"}, {"name": "Docker", "level": 3, "confidence": "explicit"}}}, "", 200)
	sim := call("POST", "/api/v1/simulate", map[string]any{"jobId": "ai-platform", "months": 6, "addedSkills": []map[string]any{{"name": "Python", "level": 4}}, "save": true}, "", 200)
	if sim["score"].(float64) < sim["baselineScore"].(float64) {
		t.Fatal("what if score decreased")
	}
	simID := sim["id"].(string)
	keyResponse := call("POST", "/api/v1/keys", map[string]any{"name": "테스트 키", "scopes": []string{"jobs:read", "mcp:use"}, "expiresInDays": 30}, "", 201)
	key := keyResponse["key"].(string)
	keyID := keyResponse["record"].(map[string]any)["id"].(string)
	call("GET", "/api/v1/jobs", nil, key, 200)
	call("GET", "/api/v1/profile", nil, key, 403)
	call("GET", "/api/v1/admin/settings", nil, key, 403)
	call("POST", "/api/v1/keys", map[string]any{}, key, 403)
	m := call("POST", "/mcp", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}, key, 200)
	tools := m["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 2 {
		t.Fatalf("tools must respect scopes: %v", tools)
	}
	call("POST", "/api/v1/keys/"+keyID+"/rotate", map[string]any{}, "", 200)
	call("GET", "/api/v1/jobs", nil, key, 401)
	call("GET", "/api/v1/approvals", nil, "", 404)
	s := call("GET", "/api/v1/admin/settings", nil, "", 200)
	s["general"].(map[string]any)["approvalEnabled"] = true
	s["ai"].(map[string]any)["apiKey"] = "super-secret-token"
	call("PUT", "/api/v1/admin/settings", s, "", 200)
	redacted := call("GET", "/api/v1/admin/settings", nil, "", 200)
	if redacted["ai"].(map[string]any)["apiKey"] != "" {
		t.Fatal("secret leaked")
	}
	var cipher []byte
	if e = app.db.QueryRow(ctx, "SELECT data FROM nr_config WHERE id='settings'").Scan(&cipher); e != nil {
		t.Fatal(e)
	}
	if bytes.Contains(cipher, []byte("super-secret-token")) {
		t.Fatal("secret stored plaintext")
	}
	approval := call("POST", "/api/v1/approvals", map[string]any{"simulationId": simID, "note": "검토"}, "", 201)
	call("PUT", "/api/v1/approvals/"+approval["id"].(string), map[string]any{"status": "approved"}, "", 403)
	call("PUT", "/api/v1/admin/users/"+uid, map[string]any{"name": "관리자", "role": "user"}, "", 400)
	user := call("POST", "/api/v1/admin/users", map[string]any{"email": "user@example.test", "name": "일반 사용자", "role": "user", "password": "Test-NextRole-Password42"}, "", 201)
	call("POST", "/api/v1/auth/logout", map[string]any{}, "", 200)
	call("POST", "/api/v1/auth/login", map[string]any{"email": "user@example.test", "password": "Test-NextRole-Password42"}, "", 200)
	call("GET", "/api/v1/admin/users", nil, "", 403)
	call("DELETE", "/api/v1/simulations/"+simID, nil, "", 404)
	p := call("GET", "/api/v1/profile", nil, "", 200)
	if p["name"] != "일반 사용자" {
		t.Fatal("profile isolation failed")
	}
	if user["role"] != "user" {
		t.Fatal("role mismatch")
	}
}
