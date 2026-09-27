package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func serve(t *testing.T, payload string, handler func(http.ResponseWriter, *http.Request)) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler != nil {
			handler(w, r)
			return
		}
		_, _ = w.Write([]byte(payload))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func TestJSONFetchPreservesParametersAndHidesCredentials(t *testing.T) {
	endpoint := serve(t, "", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("existing") != "keep" || r.URL.Query().Get("authKey") != "secret & + = value" || r.URL.Query().Get("region") != "서울" {
			t.Errorf("query parameters were not preserved/encoded")
		}
		_, _ = w.Write([]byte(`{"data":{"items":[{"key":12,"name":"플랫폼 엔지니어","company":{"name":"기업"},"tags":[{"label":"Go"},{"label":"Linux"}]}]}}`))
	})
	c := Config{Name: "내부 채용", Type: "json", Dataset: "jobs", Endpoint: endpoint + "?existing=keep", APIKey: "secret & + = value", APIKeyParam: "authKey", Params: map[string]string{"region": "서울"}, RootPath: "$.data.items", Mapping: map[string]string{"id": "key", "title": "name", "organization": "company.name", "skills": "tags[].label"}}
	records, err := Fetch(context.Background(), c)
	if err != nil || len(records) != 1 {
		t.Fatalf("fetch: %v %v", records, err)
	}
	if records[0]["id"] != "12" || records[0]["organization"] != "기업" || !reflect.DeepEqual(records[0]["skills"], []any{"Go", "Linux"}) {
		t.Fatalf("incorrect mapping: %#v", records[0])
	}
	source := records[0]["source"].(map[string]any)
	if source["url"] != endpoint || source["synthetic"] != false {
		t.Fatalf("unsafe source attribution: %#v", source)
	}
	masked, _ := json.Marshal(c.Mask())
	if strings.Contains(string(masked), "secret") || !c.Mask().HasAPIKey {
		t.Fatal("credential masking failed")
	}
}

func TestXMLRepeatedAndSingletonRecords(t *testing.T) {
	for _, count := range []int{1, 2} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			payload := "<?xml version=\"1.0\" encoding=\"UTF-8\"?><wantedRoot>"
			for i := 0; i < count; i++ {
				payload += fmt.Sprintf(`<wanted code="%d"><title>개발자 &amp; 운영자</title><tags><tag>Go</tag><tag>SQL</tag></tags></wanted>`, i)
			}
			payload += "</wantedRoot>"
			c := Config{Type: "xml", Dataset: "jobs", Endpoint: serve(t, payload, nil), RootPath: "wantedRoot.wanted", Mapping: map[string]string{"id": "@code", "title": "title", "skills": "tags.tag[]"}}
			records, err := Fetch(context.Background(), c)
			if err != nil || len(records) != count || records[0]["title"] != "개발자 & 운영자" {
				t.Fatalf("XML fetch: %#v, %v", records, err)
			}
		})
	}
}

func TestCSVMappingAndNumbers(t *testing.T) {
	c := Config{Type: "csv", Dataset: "training", Endpoint: serve(t, "\ufeff코드,과정,기술,비용\r\nC1,\"실전, 데이터\",Python;SQL,12000\r\n", nil), Mapping: map[string]string{"id": "코드", "title": "과정", "skills": "기술", "cost": "비용"}}
	records, err := Fetch(context.Background(), c)
	if err != nil || len(records) != 1 || records[0]["cost"] != float64(12000) || records[0]["title"] != "실전, 데이터" {
		t.Fatalf("CSV fetch: %#v, %v", records, err)
	}
	if !reflect.DeepEqual(records[0]["skills"], []any{"Python", "SQL"}) {
		t.Fatal("CSV list mapping failed")
	}
}

func TestNestedArrayObjectMapping(t *testing.T) {
	c := Config{Type: "json", Dataset: "occupations", Endpoint: serve(t, `[{"code":"platform","requirements":[{"label":"Go","rank":"4","importance":"2"},{"label":"Linux","rank":"3","importance":"1"}]}]`, nil), Mapping: map[string]string{"id": "code", "title": "=플랫폼 엔지니어", "skills[].name": "requirements[].label", "skills[].level": "requirements[].rank", "skills[].weight": "requirements[].importance", "source.name": "=사내 직무 사전"}}
	records, err := Fetch(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	skills := records[0]["skills"].([]any)
	if len(skills) != 2 || skills[0].(map[string]any)["level"] != float64(4) || skills[1].(map[string]any)["name"] != "Linux" {
		t.Fatalf("nested mapping: %#v", records[0])
	}
}

func TestAuthHeaderAndPOST(t *testing.T) {
	endpoint := serve(t, "", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("X-Api-Key") != "secret-token" || !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			t.Error("request configuration not applied")
		}
		_, _ = w.Write([]byte(`[]`))
	})
	_, err := Fetch(context.Background(), Config{Type: "json", Dataset: "jobs", Endpoint: endpoint, Method: "POST", Body: `{"page":1}`, AuthHeader: "X-Api-Key", APIKey: "secret-token"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRedirectDoesNotForwardSecrets(t *testing.T) {
	var visited atomic.Bool
	destination := serve(t, "", func(w http.ResponseWriter, r *http.Request) { visited.Store(true) })
	origin := serve(t, "", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination, http.StatusFound) })
	_, err := Fetch(context.Background(), Config{Type: "json", Dataset: "jobs", Endpoint: origin, APIKey: "test-credential", AuthHeader: "X-Api-Key"})
	if err == nil || visited.Load() || strings.Contains(err.Error(), "test-credential") {
		t.Fatalf("redirect credentials were not protected: %v", err)
	}
}

func TestMalformedResponsesAndLimits(t *testing.T) {
	cases := []struct{ name, kind, body, root string }{
		{"json parse", "json", "secret-invalid-json", ""},
		{"json trailing", "json", `[] []`, ""},
		{"upstream json error", "json", `{"error":"secret-invalid-json"}`, ""},
		{"upstream xml error", "xml", `<errorReport><message>secret-invalid-json</message></errorReport>`, ""},
		{"xml dtd", "xml", `<!DOCTYPE root [<!ENTITY xxe SYSTEM "file:///etc/passwd">]><root>&xxe;</root>`, "root"},
		{"xml roots", "xml", `<root/><another/>`, ""},
		{"xml excessive depth", "xml", strings.Repeat("<a>", 65) + strings.Repeat("</a>", 65), ""},
		{"missing root", "json", `{"items":[]}`, "missing"},
		{"scalar record", "json", `[1]`, ""},
		{"too many records", "json", "[" + strings.Repeat(`{},`, MaxRecords) + "{}]", ""},
		{"oversized", "json", strings.Repeat(" ", MaxResponseBytes+1), ""},
		{"csv duplicate", "csv", "id,id\na,b\n", ""},
		{"csv malformed row", "csv", "id,title\na\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Fetch(context.Background(), Config{Type: tc.kind, Dataset: "jobs", Endpoint: serve(t, tc.body, nil), RootPath: tc.root})
			if err == nil {
				t.Fatal("malformed response accepted")
			}
			if strings.Contains(err.Error(), "secret-invalid-json") {
				t.Fatal("upstream response leaked into error")
			}
		})
	}
}

func TestMappingErrorsAreAtomicAndDoNotLeakValues(t *testing.T) {
	records := []map[string]any{{"key": "one"}, {"other": "secret-value"}}
	out, err := Normalize(records, Config{Mapping: map[string]string{"id": "key"}})
	if err == nil || out != nil || strings.Contains(err.Error(), "secret-value") {
		t.Fatalf("partial results or leaked data: %v %v", out, err)
	}
	_, err = Normalize(records[:1], Config{Mapping: map[string]string{"source": "=text", "source.name": "=name"}})
	if err == nil {
		t.Fatal("conflicting target mapping accepted")
	}
}

func TestURLValidation(t *testing.T) {
	for _, endpoint := range []string{"file:///etc/passwd", "ftp://intranet.local/feed", "http://user:secret@localhost/feed", "http://host/#secret", "//localhost/feed", "http://"} {
		err := Validate(Config{Type: "json", Dataset: "jobs", Endpoint: endpoint})
		if err == nil || strings.Contains(err.Error(), "secret") {
			t.Errorf("unsafe URL validation: %q %v", endpoint, err)
		}
	}
	if err := Validate(Config{Type: "json", Dataset: "jobs", Endpoint: "http://10.10.0.1/feed"}); err != nil {
		t.Fatal("intentional intranet URL rejected")
	}
}

func TestQueryGuard(t *testing.T) {
	valid := []string{
		"SELECT id, title FROM public.career_jobs LIMIT 100",
		"WITH visible AS (SELECT id, title FROM career_jobs WHERE region = '서울') SELECT * FROM visible LIMIT 100;",
		"SELECT 'DROP TABLE x; SELECT 1' AS title, 'it''s safe' AS description",
	}
	for _, query := range valid {
		if err := ValidateQuery(query); err != nil {
			t.Errorf("valid query rejected: %s: %v", query, err)
		}
	}
	invalid := []string{
		"DELETE FROM jobs", "SELECT 1; DELETE FROM jobs", "WITH removed AS (DELETE FROM jobs RETURNING *) SELECT * FROM removed",
		"SELECT * INTO OUTFILE '/tmp/exfil' FROM jobs", "SELECT pg_read_file('/etc/passwd')", `SELECT "pg_read_file"('/etc/passwd')`,
		"SELECT LOAD_FILE('/etc/passwd')", "SELECT 1 /*!50000 INTO OUTFILE '/tmp/x' */", "SELECT 1 -- comment",
		"SELECT SLEEP(100)", "SELECT pg_sleep(100)", "SELECT dblink_exec('dsn','DELETE FROM jobs')", "SELECT set_config('role','superuser',false)",
		"SELECT $$literal$$", "SELECT E'\\'; DROP TABLE jobs; --'", "SELECT 'unterminated", "SELECT * FROM jobs FOR UPDATE",
	}
	for _, query := range invalid {
		if err := ValidateQuery(query); err == nil {
			t.Errorf("unsafe query accepted: %s", query)
		}
	}
}

func TestCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Fetch(ctx, Config{Type: "json", Dataset: "jobs", Endpoint: serve(t, "[]", nil)})
	if err == nil {
		t.Fatal("cancelled fetch succeeded")
	}
}

func TestWork24Presets(t *testing.T) {
	for _, config := range Work24Presets(time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)) {
		if err := Validate(config); err != nil || config.Enabled || config.APIKey != "" {
			t.Fatalf("invalid starter config: %v", err)
		}
	}
}

func TestCompoundIdentifierPreservesTrainingRounds(t *testing.T) {
	config := Config{Dataset: "training", Mapping: map[string]string{"id": "concat:trprId,trprDegr", "title": "title"}}
	records, err := Normalize([]map[string]any{
		{"trprId": "C100", "trprDegr": "1", "title": "같은 과정"},
		{"trprId": "C100", "trprDegr": "2", "title": "같은 과정"},
	}, config)
	if err != nil || records[0]["id"] != "C100:1" || records[1]["id"] != "C100:2" {
		t.Fatalf("training rounds collapsed: %+v %v", records, err)
	}
	left, _ := concatenateFields(map[string]any{"a": "A:B", "b": "C"}, "concat:a,b")
	right, _ := concatenateFields(map[string]any{"a": "A", "b": "B:C"}, "concat:a,b")
	if left == right {
		t.Fatal("compound delimiter collision")
	}
	for _, source := range []string{"concat:a", "concat:a,", "concat:items[],other", "concat:=literal,a"} {
		if err := validateSourceExpression(source); err == nil {
			t.Errorf("invalid concat expression accepted: %s", source)
		}
	}
	if _, err := concatenateFields(map[string]any{"a": "", "b": "secret-raw-value"}, "concat:a,b"); err == nil || strings.Contains(err.Error(), "secret-raw-value") {
		t.Fatal("empty compound field accepted or source data leaked")
	}
}
