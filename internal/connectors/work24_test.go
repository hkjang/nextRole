package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func work24TestConfig(t *testing.T, preset, payload string) Config {
	t.Helper()
	for _, c := range Work24Presets(time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)) {
		if c.PresetID != preset {
			continue
		}
		c.Endpoint = serve(t, payload, nil)
		c.APIKey = "test-only-secret"
		if preset == Work24Occupation {
			c.Params["jobCd"] = "133301"
		}
		if preset == Work24NCS {
			c.Params["jobCont"] = "합성 수행직무 테스트"
		}
		return c
	}
	t.Fatal("unknown fixture preset")
	return Config{}
}

func assertWork24Source(t *testing.T, record map[string]any, preset string) {
	t.Helper()
	source := record["source"].(map[string]any)
	if source["kind"] != "public_api" || source["provider"] != "한국고용정보원" || source["dataset"] != preset || source["synthetic"] != false || source["recordId"] != record["id"] {
		t.Fatalf("provenance missing: %#v", source)
	}
	if _, err := time.Parse(time.RFC3339, source["retrievedAt"].(string)); err != nil {
		t.Fatal(err)
	}
	m, _ := work24Metadata(preset)
	if source["url"] != m.SpecURL || strings.Contains(fmt.Sprint(source), "test-only-secret") {
		t.Fatalf("source must refer to official specification, not credential endpoint: %#v", source)
	}
	fields := source["fields"].([]string)
	raw := record["raw"].(map[string]any)
	if len(fields) != len(raw) {
		t.Fatal("provenance fields must list only retained fields")
	}
	for _, field := range fields {
		if _, ok := raw[field]; !ok {
			t.Fatal("provenance contains a missing field")
		}
	}
}

func TestWork24OccupationProjectsSelectedOfficialFields(t *testing.T) {
	c := work24TestConfig(t, Work24Occupation, `<jobSum><jobCd>133301</jobCd><jobSmclNm>응용 소프트웨어 개발자</jobSmclNm><jobSum>응용 프로그램을 개발합니다.</jobSum><sal>선택하지 않은 임금</sal><privatePayload>보관 금지</privatePayload><relMajorList><majorCd>001</majorCd><majorNm>전산학</majorNm><credential>보관 금지</credential></relMajorList><relMajorList><majorCd>002</majorCd><majorNm>컴퓨터공학</majorNm></relMajorList><relCertList><certNm>정보처리기사</certNm><extra>보관 금지</extra></relCertList></jobSum>`)
	c.SelectedFields = []string{"jobCd", "jobSmclNm", "jobSum", "relMajorList", "relCertList", "way"}
	records, err := Fetch(context.Background(), c)
	if err != nil || len(records) != 1 {
		t.Fatalf("occupation fetch: %v %v", records, err)
	}
	r := records[0]
	if r["id"] != "133301" || r["title"] != "응용 소프트웨어 개발자" || r["description"] != "응용 프로그램을 개발합니다." || r["occupationCode"] != "133301" {
		t.Fatalf("incorrect occupation normalization: %#v", r)
	}
	raw := r["raw"].(map[string]any)
	if raw["sal"] != nil || raw["privatePayload"] != nil || raw["way"] != nil || r["skills"] != nil {
		t.Fatal("unselected, missing or invented fields were retained")
	}
	majors := raw["relMajorList"].([]any)
	if len(majors) != 2 || len(majors[0].(map[string]any)) != 2 || len(raw["relCertList"].([]any)[0].(map[string]any)) != 1 {
		t.Fatal("nested groups must preserve only documented children")
	}
	assertWork24Source(t, r, Work24Occupation)
}

func TestWork24NCSObjectValuesAndNumericCodes(t *testing.T) {
	c := work24TestConfig(t, Work24NCS, `{"result":{"서버프로그램 구현":{"job_sdvn":"서버프로그램 구현","ablt_unit":900719925474099312345,"ablt_def":"서버 프로그램을 구현한다.","job_sdvn_cd":"20010202","knwg_tchn_attd":"공식 지식·기술·태도","inventedLevel":5,"rawSecret":"discard"},"데이터 처리":{"job_sdvn":"데이터 처리","ablt_unit":"2001020206_23v1","ablt_def":"데이터 처리 정의","job_sdvn_cd":"20010202"}}}`)
	records, err := Fetch(context.Background(), c)
	if err != nil || len(records) != 2 {
		t.Fatalf("dictionary result must yield two units: %#v %v", records, err)
	}
	foundLarge := false
	for _, r := range records {
		assertWork24Source(t, r, Work24NCS)
		if r["id"] == "900719925474099312345" {
			foundLarge = true
		}
		raw := r["raw"].(map[string]any)
		if r["ncsCode"] != r["id"] || r["officialLevel"] != nil || r["skills"] != nil || raw["inventedLevel"] != nil || raw["rawSecret"] != nil {
			t.Fatal("NCS codes must remain original; no internal proficiency may be inferred")
		}
	}
	if !foundLarge {
		t.Fatal("JSON identifier lost numeric precision")
	}
}

func TestWork24JobsNeverInferRequirements(t *testing.T) {
	c := work24TestConfig(t, Work24Jobs, `<wantedRoot><total>1</total><wanted><wantedAuthNo>J100</wantedAuthNo><title>Python Kubernetes 개발자</title><company>테스트 회사</company><wantedInfoUrl>https://example.org/jobs/J100</wantedInfoUrl><jobsCd>1333</jobsCd><region>서울</region><sal>연 5000만원</sal><closeDt>2026-12-31</closeDt><skills>Python, Kubernetes</skills><unselected>비공개 보관 금지</unselected></wanted></wantedRoot>`)
	c.Mapping = map[string]string{"skills": "skills"} // old/custom mapping cannot override the official projection
	records, err := Fetch(context.Background(), c)
	if err != nil || len(records) != 1 {
		t.Fatal(err)
	}
	r := records[0]
	if !reflect.DeepEqual(r["skills"], []string{}) || r["occupationCode"] != "1333" || r["salary"] != "연 5000만원" || r["deadline"] != "2026-12-31" {
		t.Fatalf("job mapping or no-inference rule failed: %#v", r)
	}
	if r["raw"].(map[string]any)["skills"] != nil {
		t.Fatal("unofficial skill list was stored")
	}
	assertWork24Source(t, r, Work24Jobs)
}

func TestWork24TrainingRoundsDatesAndTuitionReferences(t *testing.T) {
	body := `<HRDNet><scn_cnt>2</scn_cnt><srchList>`
	for _, round := range []string{"1", "2"} {
		body += `<scn_list><trprId>C100</trprId><trprDegr>` + round + `</trprDegr><title>데이터 과정</title><subTitle>합성 훈련기관</subTitle><titleLink>https://example.org/course/` + round + `</titleLink><ncsCd>20010202</ncsCd><traStartDate>2026-10-01</traStartDate><traEndDate>2026-12-30</traEndDate><courseMan>100,000</courseMan><realMan>300,000</realMan><eiEmplCnt3Gt10>Null</eiEmplCnt3Gt10><credential>discard</credential></scn_list>`
	}
	body += `</srchList></HRDNet>`
	c := work24TestConfig(t, Work24Training, body)
	records, err := Fetch(context.Background(), c)
	if err != nil || len(records) != 2 || records[0]["id"] != "C100:1" || records[1]["id"] != "C100:2" {
		t.Fatalf("training rounds collapsed: %#v %v", records, err)
	}
	for _, r := range records {
		if r["code"] != "C100" || r["courseId"] != "C100" || r["ncsCode"] != "20010202" || r["startDate"] != "2026-10-01" || r["endDate"] != "2026-12-30" || r["cost"] != nil || !strings.Contains(r["tuitionReference"].(string), "확정 본인부담금 아님") {
			t.Fatalf("training official fields incorrectly transformed: %#v", r)
		}
		raw := r["raw"].(map[string]any)
		if raw["eiEmplCnt3Gt10"] != nil || raw["credential"] != nil {
			t.Fatal("retired/null-only or unknown field retained")
		}
		assertWork24Source(t, r, Work24Training)
	}
}

func TestWork24EmptyResponsesDifferFromErrors(t *testing.T) {
	cases := []struct {
		preset, body string
		failure      bool
	}{
		{Work24Jobs, `<wantedRoot><total>0</total></wantedRoot>`, false},
		{Work24Training, `<HRDNet><scn_cnt>0</scn_cnt><srchList/></HRDNet>`, false},
		{Work24NCS, `{"result":{}}`, false},
		{Work24Jobs, `<wantedRoot><total>3</total></wantedRoot>`, true},
		{Work24Jobs, `<wrongRoot/>`, true},
		{Work24Training, `<HRDNet><errorCode>401</errorCode><scn_cnt>0</scn_cnt><message>test-only-secret</message></HRDNet>`, true},
		{Work24NCS, `{"message_cd":"INVALID_KEY","message":"test-only-secret"}`, true},
		{Work24NCS, `{"message_cd":"FAIL","result":{}}`, true},
		{Work24NCS, `{"result":[]}`, true},
		{Work24NCS, `{"result":null}`, true},
		{Work24NCS, `{"result":{"name":"not an object"}}`, true},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			c := work24TestConfig(t, tc.preset, tc.body)
			records, err := Fetch(context.Background(), c)
			if (err != nil) != tc.failure || !tc.failure && len(records) != 0 {
				t.Fatalf("incorrect empty/error distinction: records=%v err=%v", records, err)
			}
			if err != nil && strings.Contains(err.Error(), "test-only-secret") {
				t.Fatal("upstream credential echo leaked into error")
			}
		})
	}
}

func TestWork24SelectionsAndRequiredParameters(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Config)
	}{
		{"unknown field", func(c *Config) { c.SelectedFields = append(c.SelectedFields, "secret") }},
		{"duplicate field", func(c *Config) { c.SelectedFields = append(c.SelectedFields, "title") }},
		{"no identity", func(c *Config) { c.SelectedFields = []string{} }},
		{"wrong response format", func(c *Config) { c.Params["returnType"] = "JSON" }},
		{"detail instead of list", func(c *Config) { c.Params["callTp"] = "D" }},
		{"too many records", func(c *Config) { c.Params["display"] = "101" }},
		{"too many pages", func(c *Config) { c.Params["startPage"] = "1001" }},
		{"query credential", func(c *Config) { c.Params["authKey"] = "secret" }},
		{"url credential", func(c *Config) { c.Endpoint += "?AUTHKEY=secret" }},
		{"wrong credential key", func(c *Config) { c.APIKeyParam = "apiKey" }},
		{"wrong root", func(c *Config) { c.RootPath = "wantedRoot" }},
		{"wrong dataset", func(c *Config) { c.Dataset = "occupations" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := work24TestConfig(t, Work24Jobs, "")
			tc.edit(&c)
			if err := Validate(c); err == nil {
				t.Fatal("invalid Work24 config accepted")
			}
		})
	}
	training := work24TestConfig(t, Work24Training, "")
	training.SelectedFields = append(training.SelectedFields, "eiEmplCnt3Gt10")
	if Validate(training) == nil {
		t.Fatal("retired compatibility field must not be selectable")
	}
	for _, preset := range []string{Work24Occupation, Work24NCS} {
		c := work24TestConfig(t, preset, "")
		delete(c.Params, map[string]string{Work24Occupation: "jobCd", Work24NCS: "jobCont"}[preset])
		if Validate(c) == nil {
			t.Fatal("missing user query accepted")
		}
	}
}

func TestWork24NoAPIKeyNoRequestAndLegacyMasking(t *testing.T) {
	c := work24TestConfig(t, Work24Jobs, "")
	var requests atomic.Int32
	c.Endpoint = serve(t, "", func(w http.ResponseWriter, r *http.Request) { requests.Add(1) })
	c.APIKey = ""
	if _, err := Fetch(context.Background(), c); err == nil || !strings.Contains(err.Error(), "실제 API 조회를 실행하지 않았습니다") || requests.Load() != 0 {
		t.Fatalf("missing key should fail before any API request: %v", err)
	}
	c.Endpoint = "https://www.work24.go.kr/cm/openApi/call/wk/callOpenApiSvcInfo210L01.do?authKey=legacy-secret&region=11"
	c.PresetID = ""
	c.Params["AUTHKEY"] = "legacy-secret"
	masked, _ := json.Marshal(c.Mask())
	if strings.Contains(string(masked), "legacy-secret") || c.Params["AUTHKEY"] != "legacy-secret" {
		t.Fatal("legacy authKey masking leaked or mutated encrypted input")
	}
	c.PresetID = ""
	c.SelectedFields = nil
	upgraded := UpgradeWork24Config(c)
	if upgraded.PresetID != Work24Jobs || len(upgraded.SelectedFields) == 0 {
		t.Fatal("legacy Work24 connector was not recognized")
	}
}

func TestWork24SelectionDoesNotMutateSourceAndNullsAreOmitted(t *testing.T) {
	c := work24TestConfig(t, Work24NCS, "")
	raw := map[string]any{"ablt_unit": "U1", "job_sdvn": "이름", "ablt_def": nil, "unknown": "discard"}
	records, err := Normalize([]map[string]any{raw}, c)
	if err != nil || len(records) != 1 {
		t.Fatal(err)
	}
	projected := records[0]["raw"].(map[string]any)
	projected["job_sdvn"] = "changed"
	if raw["job_sdvn"] != "이름" || projected["ablt_def"] != nil || len(projected) != 2 {
		t.Fatal("projection retained nulls or changed original response")
	}
	delete(raw, "ablt_unit")
	if _, err = Normalize([]map[string]any{raw}, c); err == nil {
		t.Fatal("record missing identity must fail atomically")
	}
}

func TestWork24OptionalEmptyQueryValuesAreOmitted(t *testing.T) {
	c := work24TestConfig(t, Work24Training, "")
	c.Params["srchTraArea1"], c.Params["srchTraArea2"], c.Params["srchNcs1"] = "", "", ""
	c.Endpoint = serve(t, "", func(w http.ResponseWriter, r *http.Request) {
		for _, key := range []string{"srchTraArea1", "srchTraArea2", "srchNcs1"} {
			if r.URL.Query().Has(key) {
				t.Error("empty optional query parameter must be omitted")
			}
		}
		if r.URL.Query().Get("returnType") != "XML" || r.URL.Query().Get("authKey") == "" {
			t.Error("mandatory transport values were lost")
		}
		_, _ = w.Write([]byte(`<HRDNet><scn_cnt>0</scn_cnt></HRDNet>`))
	})
	if _, err := Fetch(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	c.Params["srchTraArea2"] = "11010"
	if Validate(c) == nil {
		t.Fatal("child region without a parent was accepted")
	}
	c.Params["srchTraArea2"] = ""
	c.Params["srchNcs3"] = "200102"
	if Validate(c) == nil {
		t.Fatal("child NCS classification without a parent was accepted")
	}
}
