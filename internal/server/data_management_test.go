package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hkjang/nextRole/internal/career"
)

func mappingFixture() SkillMapping {
	return SkillMapping{Title: "검토된 데이터 직무", OccupationRecordID: "public:occupation", NCSRecordIDs: []string{"public:ncs"}, Basis: "원문 업무 내용과 능력단위 지식을 관리자 검토", Skills: []MappingSkill{{InternalSkillID: "skill:sql", Name: "SQL", Aliases: []string{"질의언어"}, Level: 3, Weight: 1, SourceRecordIDs: []string{"public:ncs"}, Basis: "데이터 질의 지식의 적용: 내부 수준 3"}}}
}
func TestMappingRequirementsAreExplicitAndAliasesUnambiguous(t *testing.T) {
	if err := cleanMapping(ptrMapping(mappingFixture())); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*SkillMapping){
		func(m *SkillMapping) { m.Skills[0].Level = 0 },
		func(m *SkillMapping) { m.Skills[0].Weight = 0 },
		func(m *SkillMapping) { m.Skills[0].SourceRecordIDs = nil },
		func(m *SkillMapping) { m.Skills[0].Basis = "" },
		func(m *SkillMapping) {
			m.Skills = append(m.Skills, MappingSkill{InternalSkillID: "python", Name: "Python", Aliases: []string{"질의언어"}, Level: 3, Weight: 1, SourceRecordIDs: []string{"public:ncs"}, Basis: "근거"})
		},
	} {
		m := mappingFixture()
		change(&m)
		if cleanMapping(&m) == nil {
			t.Fatalf("invalid mapping accepted: %+v", m)
		}
	}
}
func ptrMapping(m SkillMapping) *SkillMapping { return &m }

func mappingRequest(method, key string, body any) *http.Request {
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(method, "/api/v1/admin/mappings", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.WithValue(r.Context(), authKey{}, identity{User: User{ID: "mapping-reviewer", Role: "admin"}}))
	if key != "" {
		r.SetPathValue("id", key)
	}
	return r
}
func mappingResponse(t *testing.T, w *httptest.ResponseRecorder) SkillMapping {
	t.Helper()
	if w.Code != 200 {
		t.Fatalf("mapping status %d: %s", w.Code, w.Body.String())
	}
	var m SkillMapping
	if json.Unmarshal(w.Body.Bytes(), &m) != nil {
		t.Fatal(w.Body.String())
	}
	return m
}
func seedMappingSources(t *testing.T, a *App) {
	t.Helper()
	for dataset, v := range map[string]career.SourceRecord{
		"occupation_details": {ID: "public:occupation", Title: "데이터 직무", OccupationCode: "1234", Description: "데이터를 처리합니다", Source: career.Source{Kind: "public_api", Provider: "한국고용정보원", RecordID: "1234"}, Raw: map[string]any{"jobAbil": "데이터 처리"}},
		"ncs_units":          {ID: "public:ncs", Title: "질의 관리", NCSCode: "2001020201_16v3", OfficialLevel: "6", Source: career.Source{Kind: "public_api", Provider: "한국고용정보원", RecordID: "ncs1", Fields: []string{"ablt_unit", "knwg_tchn_attd"}}, Raw: map[string]any{"knwg_tchn_attd": "데이터 질의", "job_sdvn_cd": "20010202"}},
	} {
		if e := a.putRecord(context.Background(), dataset, "", v.ID, v); e != nil {
			t.Fatal(e)
		}
	}
}
func TestMappingPublishVersionHistoryAndChangedSourceFailClosed(t *testing.T) {
	a, _ := isolatedSSOApp(t)
	ctx := context.Background()
	seedMappingSources(t, a)
	s, e := a.settings(ctx)
	if e != nil {
		t.Fatal(e)
	}
	s.General.DemoEnabled = false
	if e = a.setConfig(ctx, "settings", s); e != nil {
		t.Fatal(e)
	}
	jobs, e := a.jobs(ctx)
	if e != nil || len(jobs) != 0 {
		t.Fatalf("disabled demo leaked catalog: %d %v", len(jobs), e)
	}
	w := httptest.NewRecorder()
	a.mappingsSave(w, mappingRequest("POST", "", mappingFixture()))
	draft := mappingResponse(t, w)
	if draft.Version != 1 || draft.Status != "draft" {
		t.Fatal(draft)
	}
	jobs, e = a.jobs(ctx)
	if e != nil || len(jobs) != 0 {
		t.Fatal("draft mapping was used in scoring")
	}
	w = httptest.NewRecorder()
	a.mappingPublish(w, mappingRequest("POST", draft.ID, map[string]int{"version": draft.Version}))
	published := mappingResponse(t, w)
	if published.Version != 2 || !published.Valid || published.ReviewedBy != "mapping-reviewer" {
		t.Fatal(published)
	}
	jobs, e = a.jobs(ctx)
	if e != nil || len(jobs) != 1 || jobs[0].Source.Kind != "derived" || jobs[0].Skills[0].Level != 3 {
		t.Fatalf("published mapping did not preserve internal level: %+v %v", jobs, e)
	}
	if jobs[0].NCSCodes[1] != "20010202" || !trainingMatches(career.Opportunity{NCSCode: "20010202", Source: career.Source{Kind: "public_api"}}, jobs[0]) {
		t.Fatal("reviewed NCS subdivision was not mapped to training")
	}
	// Internal scale must not be copied from the source's official level 6.
	sim := career.Simulate(career.Profile{Skills: []career.Skill{{Name: "질의언어", Level: 3, Confidence: "explicit"}}}, jobs[0], 6, nil)
	if len(sim.Gaps) != 1 || sim.Gaps[0].Current != 3 || sim.Gaps[0].Required != 3 {
		t.Fatal("reviewed aliases were not applied or official NCS level leaked")
	}
	w = httptest.NewRecorder()
	a.mappingsSave(w, mappingRequest("PUT", draft.ID, draft))
	if w.Code != 409 {
		t.Fatal("stale mapping overwrote published version")
	}
	var raw career.SourceRecord
	if e = a.getRecord(ctx, "ncs_units", "", "public:ncs", &raw); e != nil {
		t.Fatal(e)
	}
	raw.Source.RetrievedAt = time.Now().UTC().Format(time.RFC3339)
	raw.Source.Fields = []string{"knwg_tchn_attd", "ablt_unit"}
	if e = a.putRecord(ctx, "ncs_units", "", raw.ID, raw); e != nil {
		t.Fatal(e)
	}
	jobs, e = a.jobs(ctx)
	if e != nil || len(jobs) != 1 {
		t.Fatal("unchanged re-fetch invalidated mapping")
	}
	raw.Raw["knwg_tchn_attd"] = "변경된 원문"
	if e = a.putRecord(ctx, "ncs_units", "", raw.ID, raw); e != nil {
		t.Fatal(e)
	}
	jobs, e = a.jobs(ctx)
	if e != nil || len(jobs) != 0 {
		t.Fatal("changed source was silently used with old mapping")
	}
	w = httptest.NewRecorder()
	a.mappingPublish(w, mappingRequest("POST", published.ID, map[string]int{"version": published.Version}))
	published = mappingResponse(t, w)
	w = httptest.NewRecorder()
	a.mappingHistory(w, mappingRequest("GET", published.ID, nil))
	var history []SkillMapping
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &history) != nil || len(history) != 3 {
		t.Fatal("mapping revision history not preserved")
	}
	deleteReq := mappingRequest("DELETE", "public:ncs", nil)
	deleteReq.SetPathValue("dataset", "ncs_units")
	w = httptest.NewRecorder()
	a.sourceRecordDelete(w, deleteReq)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	jobs, e = a.jobs(ctx)
	if e != nil || len(jobs) != 0 {
		t.Fatal("mapping survived deleted source")
	}
	w = httptest.NewRecorder()
	a.mappingPublish(w, mappingRequest("POST", published.ID, map[string]int{"version": published.Version}))
	if w.Code != 400 {
		t.Fatal("dangling source published")
	}
	w = httptest.NewRecorder()
	a.mappingsDelete(w, mappingRequest("DELETE", published.ID, nil))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	w = httptest.NewRecorder()
	a.mappingHistory(w, mappingRequest("GET", published.ID, nil))
	if json.Unmarshal(w.Body.Bytes(), &history) != nil || len(history) != 4 || history[0].Status != "deleted" {
		t.Fatal("deleted mapping lost auditable history")
	}
}

func TestTrainingNeedsReviewedCodeAndMarketIgnoresHistoricalInferences(t *testing.T) {
	job := career.Job{Title: "데이터 엔지니어", OccupationCode: "123", NCSCodes: []string{"456"}, Source: career.Source{Kind: "derived"}, Skills: []career.Requirement{{Name: "SQL"}, {Name: "Python"}}}
	if trainingMatches(career.Opportunity{Title: job.Title, Source: career.Source{Kind: "public_api"}, Skills: []string{"SQL", "Python"}}, job) {
		t.Fatal("public training guessed from title or fabricated skills")
	}
	if trainingMatches(career.Opportunity{OccupationCode: "123", Source: career.Source{Kind: "public_api"}}, job) {
		t.Fatal("different occupation code systems were equated")
	}
	if marketMatches(career.Opportunity{Title: "채용", OccupationCode: "123"}, &job) {
		t.Fatal("unreviewed recruitment crosswalk guessed")
	}
	job.RecruitmentCodes = []string{"987"}
	if !marketMatches(career.Opportunity{Title: "채용", OccupationCode: "987"}, &job) {
		t.Fatal("reviewed recruitment crosswalk ignored")
	}
	now := time.Now()
	legacy := marketFixture("legacy", "데이터 엔지니어", "서울", "상시채용", []string{"SQL", "Python"}, false)
	legacy.Opportunity.SkillsOrigin = ""
	inferred := legacy
	inferred.Opportunity.ID = "inferred"
	inferred.Opportunity.URL += "/inferred"
	inferred.Opportunity.SkillsOrigin = "inferred"
	synthetic := legacy
	synthetic.Opportunity.Source.Kind = "synthetic"
	summary := summarizeMarket([]marketRecord{legacy, inferred, synthetic}, nil, now)
	if summary.TotalJobs != 2 || len(summary.SkillFrequency) != 0 {
		t.Fatalf("guessed or synthetic skills leaked into demand: %+v", summary)
	}
	otherJob := job
	otherJob.Title = "다른 직무"
	if marketMatches(legacy.Opportunity, &otherJob) {
		t.Fatal("inferred historic skills used for job relevance")
	}
}

func TestAdminSourceRecordsExcludePrivateDataAndDemoCatalog(t *testing.T) {
	a, _ := isolatedSSOApp(t)
	ctx := context.Background()
	seedMappingSources(t, a)
	// Even if malformed private rows carry a whitelisted dataset name, the
	// administrative source browser must never include another owner's data.
	if err := a.putRecord(ctx, "occupation_details", "private-owner", "secret", career.SourceRecord{ID: "secret", Title: "개인 비공개 자료"}); err != nil {
		t.Fatal(err)
	}
	for _, job := range []career.Job{
		{ID: "synthetic-old", Title: "과거 합성", Source: career.Source{Synthetic: true}, Skills: []career.Requirement{{Name: "SQL", Level: 3, Weight: 1}}},
		{ID: "forged-public-level", Title: "원문에 없는 수준", Source: career.Source{Kind: "public_api"}, Skills: []career.Requirement{{Name: "SQL", Level: 5, Weight: 1}}},
		{ID: "forged-derived", Title: "매핑 우회", Source: career.Source{Kind: "derived"}, Skills: []career.Requirement{{Name: "SQL", Level: 5, Weight: 1}}},
	} {
		if err := a.putRecord(ctx, "occupations", "", job.ID, job); err != nil {
			t.Fatal(err)
		}
	}
	s, _ := a.settings(ctx)
	s.General.DemoEnabled = false
	if err := a.setConfig(ctx, "settings", s); err != nil {
		t.Fatal(err)
	}
	jobs, err := a.jobs(ctx)
	if err != nil || len(jobs) != 0 {
		t.Fatal("synthetic or unreviewed source became an available scored job")
	}
	r := httptest.NewRequest("GET", "/api/v1/admin/source-records?dataset=occupation_details&limit=1", nil)
	w := httptest.NewRecorder()
	a.sourceRecords(w, r)
	var out struct {
		Records []sourceDataRow `json:"records"`
		Total   int             `json:"total"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &out) != nil || out.Total != 1 || len(out.Records) != 1 || out.Records[0].ID == "secret" {
		t.Fatalf("source pagination or privacy isolation failed: %s", w.Body.String())
	}
	r = httptest.NewRequest("GET", "/api/v1/admin/source-records?dataset=profile", nil)
	w = httptest.NewRecorder()
	a.sourceRecords(w, r)
	if w.Code != 400 {
		t.Fatal("unapproved private dataset browsed")
	}
	w = httptest.NewRecorder()
	a.dataOverview(w, httptest.NewRequest("GET", "/api/v1/admin/data", nil))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
}
