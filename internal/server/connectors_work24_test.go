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
	"time"

	"github.com/hkjang/nextRole/internal/career"
	"github.com/hkjang/nextRole/internal/connectors"
)

func TestWork24ImportPersistsOnlySelectedRawAndDerivedFields(t *testing.T) {
	app, _ := isolatedSSOApp(t)
	r, _ := http.NewRequest(http.MethodPost, "http://nextrole.test/api/v1/admin/import", nil)
	input := []map[string]any{{
		"id": "fake-top-level", "title": "잘못된 가공 제목", "salary": "선택하지 않은 급여", "description": "선택하지 않은 설명",
		"skills": []string{"Python"}, "skillsOrigin": "source_field",
		"source": map[string]any{"dataset": connectors.Work24Jobs, "fields": []string{"wantedAuthNo", "title", "company", "wantedInfoUrl"}},
		"raw":    map[string]any{"wantedAuthNo": "J1", "title": "Python 공고", "company": "테스트 기업", "wantedInfoUrl": "https://example.test/jobs/J1", "sal": "선택하지 않은 급여", "unexpected": "저장 금지"},
	}}
	for range 2 {
		if err := app.importRecords(r, "jobs", input, "고용24 채용정보", "work24-test"); err != nil {
			t.Fatal(err)
		}
	}
	if input[0]["id"] != "fake-top-level" || input[0]["title"] != "잘못된 가공 제목" {
		t.Fatal("import mutated caller data; retry cannot be safe")
	}
	var o career.Opportunity
	if err := app.getRecord(context.Background(), "jobs", "", "work24-test:J1", &o); err != nil {
		t.Fatal(err)
	}
	if o.Title != "Python 공고" || o.Salary != "" || o.Description != "" || len(o.Skills) != 0 || o.SkillsOrigin != "" || len(o.PublicData) != 4 || o.PublicData["sal"] != nil || o.PublicData["unexpected"] != nil {
		t.Fatalf("unselected/invented fields persisted: %+v", o)
	}
	if o.Source.Kind != "public_api" || o.Source.RecordID != "J1" || len(o.Source.Fields) != 4 || strings.Contains(o.Source.URL, "example.test") {
		t.Fatalf("incorrect official provenance: %+v", o.Source)
	}
	var count int
	if err := app.db.QueryRow(context.Background(), "SELECT count(*) FROM nr_records WHERE kind='jobs'").Scan(&count); err != nil || count != 1 {
		t.Fatal("same source record was duplicated on retry")
	}
}

func TestWork24SourceDatasetsAndTrainingRoundsRemainSeparate(t *testing.T) {
	app, _ := isolatedSSOApp(t)
	r, _ := http.NewRequest(http.MethodPost, "http://nextrole.test/api/v1/admin/import", nil)
	fixtures := []struct {
		preset, dataset, id string
		raw                 map[string]any
	}{
		{connectors.Work24Occupation, "occupation_details", "133301", map[string]any{"jobCd": "133301", "jobSmclNm": "개발자", "jobSum": "공식 하는 일", "jobAbil": "공식 역량 문자열"}},
		{connectors.Work24NCS, "ncs_units", "2001020201_23v1", map[string]any{"ablt_unit": "2001020201_23v1", "job_sdvn": "서버 구현", "ablt_def": "공식 정의", "job_sdvn_cd": "20010202"}},
	}
	for _, fixture := range fixtures {
		rows, err := connectors.Normalize([]map[string]any{fixture.raw}, connectors.Config{PresetID: fixture.preset})
		if err != nil {
			t.Fatal(err)
		}
		if err := app.importRecords(r, fixture.dataset, rows, "원천 자료", "source"); err != nil {
			t.Fatal(err)
		}
		var original career.SourceRecord
		if err := app.getRecord(context.Background(), fixture.dataset, "", "source:"+fixture.id, &original); err != nil || original.Source.RecordID != fixture.id || len(original.Raw) == 0 || original.OfficialLevel != "" {
			t.Fatalf("original source was lost or given an invented level: %+v %v", original, err)
		}
	}
	var occupations int
	if err := app.db.QueryRow(context.Background(), "SELECT count(*) FROM nr_records WHERE kind='occupations'").Scan(&occupations); err != nil || occupations != 0 {
		t.Fatal("unreviewed public source was inserted into scoring occupations")
	}
	training := []map[string]any{}
	for _, round := range []string{"1", "2"} {
		rows, err := connectors.Normalize([]map[string]any{{"trprId": "COURSE", "trprDegr": round, "title": "교육", "subTitle": "기관", "titleLink": "https://example.test/course/" + round, "courseMan": "100000", "ncsCd": "20010202", "traStartDate": "2026-10-01", "traEndDate": "2026-12-31"}}, connectors.Config{PresetID: connectors.Work24Training})
		if err != nil {
			t.Fatal(err)
		}
		training = append(training, rows[0])
	}
	if err := app.importRecords(r, "training", training, "훈련", "training-test"); err != nil {
		t.Fatal(err)
	}
	for _, round := range []string{"1", "2"} {
		var stored career.Opportunity
		if err := app.getRecord(context.Background(), "training", "", "training-test:COURSE:"+round, &stored); err != nil || stored.CourseRound != round || stored.NCSCode != "20010202" || stored.StartDate == "" || stored.EndDate == "" || stored.Cost != 0 || !strings.Contains(stored.TuitionReference, "확정 본인부담금 아님") {
			t.Fatalf("training round identity or tuition semantics lost: %+v %v", stored, err)
		}
	}
	duplicate := []map[string]any{training[0], training[0]}
	if err := app.importRecords(r, "training", duplicate, "훈련", "training-test"); err == nil {
		t.Fatal("duplicate course round must fail atomically")
	}
}

func TestWork24EmptySyncPreservesExistingRecordsAndMasksKey(t *testing.T) {
	app, ts := isolatedSSOApp(t)
	client := newSSOBrowser(t)
	login, err := client.Post(ts.URL+"/api/v1/auth/login", "application/json", strings.NewReader(`{"email":"admin@example.test","password":"NextRole-Test-Password42"}`))
	if err != nil {
		t.Fatal(err)
	}
	login.Body.Close()
	if login.StatusCode != http.StatusOK {
		t.Fatal("fixture login failed")
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("authKey") != "private-work24-test-key" {
			t.Error("API key was not sent through the dedicated parameter")
		}
		_, _ = io.WriteString(w, `<wantedRoot><total>0</total></wantedRoot>`)
	}))
	defer upstream.Close()
	var config connectors.Config
	for _, preset := range connectors.Presets() {
		if preset.PresetID == connectors.Work24Jobs {
			config = preset
		}
	}
	config.ID, config.Endpoint, config.Enabled, config.APIKey = "work24-empty", upstream.URL, true, "private-work24-test-key"
	if err := app.setConfig(context.Background(), "connectors", []connectors.Config{config}); err != nil {
		t.Fatal(err)
	}
	if err := app.putRecord(context.Background(), "jobs", "", "work24-empty:existing", map[string]any{"id": "work24-empty:existing", "title": "기존 공고"}); err != nil {
		t.Fatal(err)
	}
	res, err := client.Post(ts.URL+"/api/v1/admin/connectors/work24-empty/sync", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(res.Body)
	res.Body.Close()
	var result map[string]any
	_ = json.Unmarshal(data, &result)
	if res.StatusCode != 200 || result["count"] != float64(0) || bytes.Contains(data, []byte(config.APIKey)) {
		t.Fatalf("empty official response must be a successful no-op sync: %d %s", res.StatusCode, data)
	}
	var existing map[string]any
	if err := app.getRecord(context.Background(), "jobs", "", "work24-empty:existing", &existing); err != nil || existing["title"] != "기존 공고" {
		t.Fatal("empty sync deleted or changed existing data")
	}
	var saved []connectors.Config
	if err := app.config(context.Background(), "connectors", &saved); err != nil || saved[0].LastError != "" || saved[0].LastSync == "" {
		t.Fatal("empty sync was recorded as an error")
	}
	if _, err := time.Parse(time.RFC3339, saved[0].LastSync); err != nil {
		t.Fatal(err)
	}
	res, err = client.Get(ts.URL + "/api/v1/admin/connectors")
	if err != nil {
		t.Fatal(err)
	}
	data, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || bytes.Contains(data, []byte(config.APIKey)) {
		t.Fatal("administrator list leaked API key")
	}
}

func TestImportNoTitleBasedSkillInferenceAndLegacyWork24Correction(t *testing.T) {
	app, _ := isolatedSSOApp(t)
	r, _ := http.NewRequest(http.MethodPost, "http://nextrole.test/api/v1/admin/import", nil)
	for _, fixture := range []struct {
		id, source string
		skills     []string
		want       int
	}{
		{"plain", "직접 반입", nil, 0},
		{"explicit", "직접 반입", []string{"Go"}, 1},
		{"legacy", "고용24 채용정보", []string{"Python", "Kubernetes"}, 0},
	} {
		input := []map[string]any{{"id": fixture.id, "title": "Python Kubernetes 엔지니어", "description": "Go Linux 개발", "organization": "합성 테스트 기관", "url": "https://example.test/" + fixture.id, "skills": fixture.skills, "source": map[string]any{"name": fixture.source}}}
		if err := app.importRecords(r, "jobs", input, fixture.source, "test"); err != nil {
			t.Fatal(err)
		}
		var stored career.Opportunity
		if err := app.getRecord(context.Background(), "jobs", "", "test:"+fixture.id, &stored); err != nil || len(stored.Skills) != fixture.want {
			t.Fatalf("skill provenance failed: %+v %v", stored, err)
		}
		if fixture.want > 0 && stored.SkillsOrigin != "admin_reviewed" {
			t.Fatal("explicit administrator skills lost their distinct provenance")
		}
	}
}
