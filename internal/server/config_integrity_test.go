package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestInvalidConfigurationCannotBeOverwritten(t *testing.T) {
	app, ts := isolatedSSOApp(t)
	client := newSSOBrowser(t)
	res, err := client.Post(ts.URL+"/api/v1/auth/login", "application/json", strings.NewReader(`{"email":"admin@example.test","password":"NextRole-Test-Password42"}`))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatal("test administrator login failed")
	}
	for _, kind := range []string{"providers", "connectors"} {
		t.Run(kind, func(t *testing.T) {
			for _, invalid := range [][]byte{[]byte("corrupt-secret-configuration"), app.encrypt([]byte("not valid JSON"))} {
				_, err := app.db.Exec(context.Background(), "INSERT INTO nr_config(id,data) VALUES($1,$2) ON CONFLICT(id) DO UPDATE SET data=excluded.data", kind, invalid)
				if err != nil {
					t.Fatal(err)
				}
				body := map[string]any{"name": "새 설정", "type": "oidc", "enabled": false}
				if kind == "connectors" {
					body["type"], body["dataset"], body["endpoint"] = "json", "jobs", "https://example.test/jobs"
				}
				encoded, _ := json.Marshal(body)
				for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
					path := ts.URL + "/api/v1/admin/" + kind
					if method == http.MethodDelete {
						path += "/some-id"
					}
					req, _ := http.NewRequest(method, path, bytes.NewReader(encoded))
					req.Header.Set("Content-Type", "application/json")
					response, err := client.Do(req)
					if err != nil {
						t.Fatal(err)
					}
					data, _ := io.ReadAll(response.Body)
					response.Body.Close()
					if response.StatusCode != 500 || bytes.Contains(data, invalid) {
						t.Fatalf("invalid config was hidden or exposed for %s %s: %d", method, kind, response.StatusCode)
					}
				}
				var after []byte
				if err := app.db.QueryRow(context.Background(), "SELECT data FROM nr_config WHERE id=$1", kind).Scan(&after); err != nil || !bytes.Equal(after, invalid) {
					t.Fatal("failed config read caused existing configuration data loss")
				}
			}
			_, _ = app.db.Exec(context.Background(), "DELETE FROM nr_config WHERE id=$1", kind)
		})
	}
}

func TestDuplicateImportIdentifiersAreAtomic(t *testing.T) {
	app, _ := isolatedSSOApp(t)
	req, _ := http.NewRequest(http.MethodPost, "http://nextrole.test/api/v1/admin/import", nil)
	fixture := func(title string) map[string]any {
		return map[string]any{"id": "same-id", "title": title, "organization": "예제 기업", "url": "https://jobs.example.test/same-id", "deadline": "채용시까지", "skills": []string{"Go", "Linux"}}
	}
	if err := app.importRecords(req, "jobs", []map[string]any{fixture("기존 공고")}, "승인 데이터", "manual"); err != nil {
		t.Fatal(err)
	}
	err := app.importRecords(req, "jobs", []map[string]any{fixture("변경 1"), fixture("변경 2")}, "승인 데이터", "manual")
	if err == nil || !strings.Contains(err.Error(), "중복") {
		t.Fatal("duplicate IDs silently overwrote records")
	}
	var stored map[string]any
	if err := app.getRecord(context.Background(), "jobs", "", "manual:same-id", &stored); err != nil || stored["title"] != "기존 공고" {
		t.Fatal("duplicate batch partially changed existing data")
	}
	var count int
	if err := app.db.QueryRow(context.Background(), "SELECT count(*) FROM nr_records WHERE kind='jobs'").Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate batch changed record count")
	}
}
