package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hkjang/nextRole/internal/career"
)

func TestIdentifierMinimization(t *testing.T) {
	p := analysisProfile(career.Profile{Name: "홍길동", Narrative: "성명: 홍길동\n연락처: 010-1234-5678\n메일 user@example.test / 900101-1234567\n홍길동은 Python 3, Java 17, NCS 20010202 경험 10년", Skills: []career.Skill{{Name: "Python", Level: 3}}, WeeklyHours: 10})
	b, _ := json.Marshal(p)
	for _, s := range []string{"홍길동", "010-1234-5678", "user@example.test", "900101-1234567"} {
		if bytes.Contains(b, []byte(s)) {
			t.Fatalf("identifier retained: %s", s)
		}
	}
	for _, s := range []string{"Python 3", "Java 17", "20010202", "10년"} {
		if !strings.Contains(p.Narrative, s) {
			t.Fatalf("career evidence removed: %s", s)
		}
	}
	if p.Name != "" || p.Source.Kind != "user_input" || p.Source.Synthetic {
		t.Fatal("user source metadata incorrect")
	}
}

func TestWork24CannotClaimRequiredSkills(t *testing.T) {
	for _, origin := range []string{"source_field", "admin_reviewed", ""} {
		o := career.Opportunity{Source: career.Source{Kind: "public_api", Dataset: "work24-jobs"}, Skills: []string{"Python"}, SkillsOrigin: origin}
		if len(factualOpportunitySkills(o)) != 0 {
			t.Fatalf("Work24 fabricated requirement accepted: %s", origin)
		}
	}
}

func privacyRequest(t *testing.T, client *http.Client, ts *httptest.Server, method, path string, body any, want int) map[string]any {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, ts.URL+"/api/v1"+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", ts.URL)
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != want {
		t.Fatalf("%s %s =%d want %d: %s", method, path, res.StatusCode, want, raw)
	}
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out
}
func TestCareerConsentVersionWithdrawalAndIsolation(t *testing.T) {
	app, ts := isolatedSSOApp(t)
	client := newSSOBrowser(t)
	call := func(method, path string, body any, want int) map[string]any {
		return privacyRequest(t, client, ts, method, path, body, want)
	}
	login := call("POST", "/auth/login", map[string]any{"email": "admin@example.test", "password": "NextRole-Test-Password42"}, 200)
	uid := login["user"].(map[string]any)["id"].(string)
	call("PUT", "/profile", map[string]any{}, 403)
	call("POST", "/profile/parse", map[string]any{"text": "Java 경험"}, 403)
	call("POST", "/ai/stream", map[string]any{"task": "parse", "text": "경력"}, 403)
	call("GET", "/recommendations", nil, 403)
	call("GET", "/admin/data-policy", nil, 200)
	p := call("GET", "/privacy", nil, 200)
	if p["consent"] != nil {
		t.Fatal("implicit consent")
	}
	call("POST", "/privacy/consent", map[string]any{"version": p["version"], "accepted": false}, 409)
	call("POST", "/privacy/consent", map[string]any{"version": "old", "accepted": true}, 409)
	call("POST", "/privacy/consent", map[string]any{"version": p["version"], "accepted": true}, 200)
	profile := call("PUT", "/profile", map[string]any{"name": "홍길동", "narrative": "Python 3년 user@example.test 010-1234-5678", "skills": []map[string]any{{"name": "Python", "level": 3, "confidence": "explicit"}}, "weeklyHours": 10}, 200)
	if profile["name"] != "" || strings.Contains(profile["narrative"].(string), "user@example.test") {
		t.Fatal("identifier stored")
	}
	call("POST", "/simulate", map[string]any{"jobId": "ai-platform", "save": true}, 200)
	policy := call("GET", "/admin/data-policy", nil, 200)
	policy["notice"] = policy["notice"].(string) + "\n변경된 수집 안내입니다."
	call("PUT", "/admin/data-policy", policy, 400)
	policy["version"] = "2"
	call("PUT", "/admin/data-policy", policy, 200)
	call("PUT", "/profile", map[string]any{}, 403)
	call("POST", "/privacy/consent", map[string]any{"version": "2", "accepted": true}, 200)
	otherID := "other-user-fixture"
	if err := app.putRecord(context.Background(), "jobs", "", "retain-public", map[string]any{"id": "retain-public", "title": "공공자료"}); err != nil {
		t.Fatal(err)
	}
	_, err := app.db.Exec(context.Background(), "INSERT INTO nr_records(kind,owner_id,id,data) VALUES('profile',$1,'current','{}'),('approval','','own-request',jsonb_build_object('userId',$2::text)),('approval','','other-request',jsonb_build_object('userId',$1::text))", otherID, uid)
	if err != nil {
		t.Fatal(err)
	}
	call("DELETE", "/privacy/consent", nil, 200)
	var ownCount, keepCount int
	if err = app.db.QueryRow(context.Background(), "SELECT count(*) FROM nr_records WHERE owner_id=$1 OR (kind='approval' AND data->>'userId'=$1)", uid).Scan(&ownCount); err != nil || ownCount != 0 {
		t.Fatalf("withdrawal retained own data: %d %v", ownCount, err)
	}
	if err = app.db.QueryRow(context.Background(), "SELECT count(*) FROM nr_records WHERE owner_id=$1 OR id IN ('other-request','retain-public')", otherID).Scan(&keepCount); err != nil || keepCount != 3 {
		t.Fatalf("withdrawal affected unrelated data: %d %v", keepCount, err)
	}
	call("GET", "/me", nil, 200)
	call("PUT", "/profile", map[string]any{}, 403)
	if err := app.putRecord(context.Background(), "profile", uid, "current", career.Profile{}); err != errConsent {
		t.Fatalf("late write after withdrawal: %v", err)
	}
}

func TestAIPromptExcludesIdentifiers(t *testing.T) {
	app, ts := isolatedSSOApp(t)
	client := newSSOBrowser(t)
	privacyRequest(t, client, ts, "POST", "/auth/login", map[string]any{"email": "admin@example.test", "password": "NextRole-Test-Password42"}, 200)
	policy := privacyRequest(t, client, ts, "GET", "/privacy", nil, 200)
	privacyRequest(t, client, ts, "POST", "/privacy/consent", map[string]any{"version": policy["version"], "accepted": true}, 200)
	seen := make(chan []byte, 1)
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		seen <- raw
		w.Header().Set("Content-Type", "text/event-stream")
		payload := map[string]any{"choices": []map[string]any{{"delta": map[string]string{"content": `{"currentRole":"개발자","skills":[],"weeklyHours":10}`}}}}
		b, _ := json.Marshal(payload)
		_, _ = w.Write([]byte("data: " + string(b) + "\n\ndata: [DONE]\n\n"))
	}))
	defer model.Close()
	s, err := app.settings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	s.AI.Enabled = true
	s.AI.BaseURL = model.URL
	s.AI.Model = "mock"
	if err = app.setConfig(context.Background(), "settings", s); err != nil {
		t.Fatal(err)
	}
	body := `{"task":"parse","text":"성명: 홍길동\nJava 개발 10년, user@example.test, 010-1234-5678"}`
	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/ai/stream", strings.NewReader(body))
	req.Header.Set("Origin", ts.URL)
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !bytes.Contains(raw, []byte("event: profile")) {
		t.Fatalf("stream failed: %d %s", res.StatusCode, raw)
	}
	select {
	case prompt := <-seen:
		for _, v := range []string{"홍길동", "user@example.test", "010-1234-5678"} {
			if bytes.Contains(prompt, []byte(v)) {
				t.Fatalf("identifier sent to model: %s", v)
			}
		}
	default:
		t.Fatal("model was not called")
	}
}
