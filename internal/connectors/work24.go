package connectors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Work24PresetID also recognizes pre-preset configurations at official legacy
// endpoints. An explicit presetId permits an approved intranet proxy endpoint.
func Work24PresetID(c Config) string {
	if c.PresetID != "" {
		return c.PresetID
	}
	u, err := url.Parse(c.Endpoint)
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if host != "www.work24.go.kr" && host != "work24.go.kr" && host != "openapi.work.go.kr" && host != "www.hrd.go.kr" {
		return ""
	}
	switch {
	case strings.HasSuffix(u.Path, "/callOpenApiSvcInfo212D01.do"):
		return Work24Occupation
	case strings.HasSuffix(u.Path, "/callOpenApiSvcInfo215L01.do"):
		return Work24NCS
	case strings.HasSuffix(u.Path, "/callOpenApiSvcInfo210L01.do"), strings.HasSuffix(u.Path, "/wantedApi.do"):
		return Work24Jobs
	case strings.HasSuffix(u.Path, "/callOpenApiSvcInfo310L01.do"), strings.HasSuffix(u.Path, "/HRD4J/HRD4J010/HRD4J010_1.jsp"):
		return Work24Training
	}
	return ""
}

func work24Metadata(id string) (PresetMetadata, bool) {
	for _, m := range Work24Metadata() {
		if m.ID == id {
			return m, true
		}
	}
	return PresetMetadata{}, false
}

// UpgradeWork24Config makes legacy preset identity explicit without adding any
// fields to an administrator's selection. Unknown presets fail in Validate.
func UpgradeWork24Config(c Config) Config {
	if p := Work24PresetID(c); p != "" {
		c.PresetID = p
		if m, ok := work24Metadata(p); ok && c.SelectedFields == nil {
			c.SelectedFields = append([]string{}, m.DefaultSelectedFields...)
		}
	}
	return c
}

func selectedWork24Fields(c Config, m PresetMetadata) ([]string, error) {
	selected := c.SelectedFields
	if selected == nil {
		selected = m.DefaultSelectedFields
	}
	allowed := map[string]FieldMetadata{}
	for _, f := range m.Fields {
		allowed[f.Name] = f
	}
	seen := map[string]bool{}
	for _, name := range selected {
		if _, ok := allowed[name]; !ok || seen[name] {
			return nil, errors.New("고용24 선택 필드에 미지원 항목 또는 중복 항목이 있습니다")
		}
		seen[name] = true
	}
	for _, f := range m.Fields {
		if f.Required && !seen[f.Name] {
			return nil, fmt.Errorf("고용24 필수 식별·표시 필드는 선택 해제할 수 없습니다: %s", f.Name)
		}
	}
	return selected, nil
}

func validateWork24Config(c Config) error {
	preset := Work24PresetID(c)
	if preset == "" {
		if len(c.SelectedFields) > 0 {
			return errors.New("선택 필드는 고용24 프리셋에서만 사용합니다. 일반 연동에는 필드 매핑을 설정하세요")
		}
		return nil
	}
	m, ok := work24Metadata(preset)
	if !ok {
		return errors.New("지원하지 않는 고용24 프리셋입니다")
	}
	if c.Dataset != m.Dataset || c.Type != m.Format || c.RootPath != m.RootPath {
		return errors.New("고용24 프리셋의 데이터 종류·응답 형식·목록 경로는 공식 규격을 유지해야 합니다")
	}
	if c.APIKeyParam != "authKey" || (c.Method != "" && c.Method != "GET") || c.Body != "" {
		return errors.New("고용24 요청은 GET과 authKey 인증키 전용 필드를 사용합니다")
	}
	if _, err := selectedWork24Fields(c, m); err != nil {
		return err
	}
	for key := range c.Params {
		if strings.EqualFold(key, "authKey") {
			return errors.New("authKey는 요청 매개변수에 저장할 수 없습니다. 관리자 API 키 필드만 사용하세요")
		}
	}
	if u, err := url.Parse(c.Endpoint); err == nil {
		for key := range u.Query() {
			if strings.EqualFold(key, "authKey") {
				return errors.New("authKey는 URL에 저장할 수 없습니다. 관리자 API 키 필드만 사용하세요")
			}
		}
	}
	for _, key := range m.RequiredParams {
		if strings.TrimSpace(c.Params[key]) == "" {
			return fmt.Errorf("고용24 필수 요청 조건을 입력하세요: %s", key)
		}
	}
	fixed := map[string]string{"returnType": strings.ToUpper(m.Format)}
	switch preset {
	case Work24Occupation:
		fixed["target"], fixed["jobGb"], fixed["dtlGb"] = "JOBDTL", "1", "1"
	case Work24Jobs:
		fixed["callTp"] = "L"
		if !boundedInt(c.Params["startPage"], 1, 1000) || !boundedInt(c.Params["display"], 1, 100) {
			return errors.New("채용 startPage는 1~1000, display는 1~100 범위의 정수여야 합니다")
		}
	case Work24NCS:
		if c.Params["limit"] != "" && !boundedInt(c.Params["limit"], 1, MaxRecords) {
			return errors.New("NCS limit은 1~5000 범위의 정수여야 합니다")
		}
	case Work24Training:
		fixed["outType"] = "1"
		if c.Params["srchTraArea2"] != "" && c.Params["srchTraArea1"] == "" {
			return errors.New("훈련 지역 중분류를 조회하려면 지역 대분류도 입력하세요")
		}
		for level := 2; level <= 4; level++ {
			if c.Params[fmt.Sprintf("srchNcs%d", level)] != "" && c.Params[fmt.Sprintf("srchNcs%d", level-1)] == "" {
				return errors.New("훈련 NCS 분류를 조회하려면 상위 분류도 입력하세요")
			}
		}
		if !boundedInt(c.Params["pageNum"], 1, 1000) || !boundedInt(c.Params["pageSize"], 1, 100) {
			return errors.New("훈련 pageNum은 1~1000, pageSize는 1~100 범위의 정수여야 합니다")
		}
		from, e1 := time.Parse("20060102", c.Params["srchTraStDt"])
		to, e2 := time.Parse("20060102", c.Params["srchTraEndDt"])
		if e1 != nil || e2 != nil || from.After(to) {
			return errors.New("훈련 시작일 검색 범위는 YYYYMMDD 형식이며 시작일이 종료일보다 늦을 수 없습니다")
		}
		if c.Params["sort"] != "ASC" && c.Params["sort"] != "DESC" {
			return errors.New("훈련 sort는 ASC 또는 DESC로 지정하세요")
		}
		if !strings.Contains("|1|2|3|5|", "|"+c.Params["sortCol"]+"|") {
			return errors.New("훈련 sortCol은 1, 2, 3, 5 중 선택하세요")
		}
	}
	for key, value := range fixed {
		if c.Params[key] != value {
			return fmt.Errorf("고용24 공식 요청 조건을 유지하세요: %s=%s", key, value)
		}
	}
	return nil
}

func boundedInt(raw string, min, max int) bool {
	n, err := strconv.Atoi(raw)
	return err == nil && n >= min && n <= max
}

func work24Records(root any, preset string) ([]map[string]any, error) {
	m, ok := work24Metadata(preset)
	if !ok {
		return nil, errors.New("지원하지 않는 고용24 프리셋입니다")
	}
	parts, _ := pathSegments(m.RootPath)
	selected, found := lookup(root, parts)
	if preset == Work24NCS {
		groups, ok := selected.(map[string]any)
		if !found || !ok {
			return nil, errors.New("NCS 결과는 result 안의 능력단위명별 객체여야 합니다")
		}
		if len(groups) > MaxRecords {
			return nil, errors.New("한 번에 5,000개 레코드까지만 가져올 수 있습니다")
		}
		names := make([]string, 0, len(groups))
		for name := range groups {
			names = append(names, name)
		}
		sort.Strings(names)
		records := make([]map[string]any, 0, len(groups))
		for _, name := range names {
			record, ok := groups[name].(map[string]any)
			if !ok {
				return nil, errors.New("NCS 능력단위 결과 항목은 객체여야 합니다")
			}
			records = append(records, record)
		}
		return records, nil
	}
	if !found {
		countPath := ""
		if preset == Work24Jobs {
			countPath = "wantedRoot.total"
		} else if preset == Work24Training {
			countPath = "HRDNet.scn_cnt"
		}
		parts, _ := pathSegments(countPath)
		if count, exists := lookup(root, parts); countPath != "" && exists && fmt.Sprint(count) == "0" {
			return []map[string]any{}, nil
		}
		return nil, errors.New("고용24 공식 응답의 목록 경로가 없습니다. API 오류 또는 요청 조건을 확인하세요")
	}
	return objectRecords(selected)
}

func scalarText(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case json.Number, float64, int, int64:
		return fmt.Sprint(v), true
	case nil:
		return "", true
	}
	return "", false
}

// selectOfficialGroup preserves only documented child fields, never an
// unfiltered nested object. XML singleton groups become uniform JSON arrays.
func selectOfficialGroup(value any, children []string) ([]any, error) {
	if value == nil || value == "" {
		return []any{}, nil
	}
	items, ok := value.([]any)
	if !ok {
		items = []any{value}
	}
	if len(items) > MaxRecords {
		return nil, errors.New("고용24 관련 항목 수가 제한을 초과했습니다")
	}
	out := []any{}
	for _, item := range items {
		record, ok := item.(map[string]any)
		if !ok {
			return nil, errors.New("고용24 관련 항목의 객체 형식이 올바르지 않습니다")
		}
		projected := map[string]any{}
		for _, child := range children {
			if v, ok := record[child]; ok && v != nil {
				text, scalar := scalarText(v)
				if !scalar {
					return nil, errors.New("고용24 공식 필드에 지원하지 않는 중첩 구조가 있습니다")
				}
				projected[child] = text
			}
		}
		out = append(out, projected)
	}
	return out, nil
}

func normalizeWork24(records []map[string]any, c Config, preset string) ([]map[string]any, error) {
	m, ok := work24Metadata(preset)
	if !ok {
		return nil, errors.New("지원하지 않는 고용24 프리셋입니다")
	}
	selected, err := selectedWork24Fields(c, m)
	if err != nil {
		return nil, err
	}
	required := map[string]bool{}
	for _, f := range m.Fields {
		required[f.Name] = f.Required
	}
	groups := map[string][]string{"relMajorList": {"majorCd", "majorNm"}, "relCertList": {"certNm"}, "relJobList": {"jobCd", "jobNm"}}
	results := make([]map[string]any, 0, len(records))
	for i, record := range records {
		raw := map[string]any{}
		present := []string{}
		for _, key := range selected {
			value, exists := record[key]
			if !exists || value == nil {
				if required[key] {
					return nil, fmt.Errorf("%d번째 고용24 레코드에 필수 식별·표시 필드가 없습니다: %s", i+1, key)
				}
				continue
			}
			if children, nested := groups[key]; nested {
				value, err = selectOfficialGroup(value, children)
				if err != nil {
					return nil, err
				}
			} else {
				text, scalar := scalarText(value)
				if !scalar {
					return nil, fmt.Errorf("%d번째 고용24 레코드의 공식 필드 형식이 올바르지 않습니다: %s", i+1, key)
				}
				if required[key] && strings.TrimSpace(text) == "" {
					return nil, fmt.Errorf("%d번째 고용24 레코드의 필수 식별·표시 필드가 비어 있습니다: %s", i+1, key)
				}
				value = text
			}
			raw[key] = value
			present = append(present, key)
		}
		out := map[string]any{"raw": raw}
		copyFields := func(mapping map[string]string) {
			for dst, src := range mapping {
				if value, exists := raw[src]; exists {
					out[dst] = value
				}
			}
		}
		switch preset {
		case Work24Occupation:
			copyFields(map[string]string{"id": "jobCd", "title": "jobSmclNm", "occupationCode": "jobCd", "description": "jobSum"})
		case Work24NCS:
			copyFields(map[string]string{"id": "ablt_unit", "title": "job_sdvn", "ncsCode": "ablt_unit", "description": "ablt_def"})
		case Work24Jobs:
			copyFields(map[string]string{"id": "wantedAuthNo", "title": "title", "organization": "company", "url": "wantedInfoUrl", "occupationCode": "jobsCd", "region": "region", "salary": "sal", "deadline": "closeDt"})
			out["skills"] = []string{}
		case Work24Training:
			copyFields(map[string]string{"title": "title", "organization": "subTitle", "url": "titleLink", "region": "address", "description": "contents", "code": "trprId", "courseId": "trprId", "courseRound": "trprDegr", "ncsCode": "ncsCd", "startDate": "traStartDate", "endDate": "traEndDate"})
			out["id"], err = concatenateFields(raw, "concat:trprId,trprDegr")
			if err != nil {
				return nil, err
			}
			tuition := []string{}
			for _, fee := range []struct{ key, label string }{{"courseMan", "수강비"}, {"realMan", "실제 훈련비"}} {
				if value, exists := raw[fee.key]; exists && value != "" {
					tuition = append(tuition, fee.label+": "+fmt.Sprint(value))
				}
			}
			if len(tuition) > 0 {
				out["tuitionReference"] = strings.Join(tuition, " / ") + " (원문 참고·개인별 확정 본인부담금 아님)"
			}
			out["skills"] = []string{}
		}
		out["source"] = map[string]any{"name": m.Name, "kind": "public_api", "provider": "한국고용정보원", "dataset": preset, "url": m.SpecURL, "recordId": out["id"], "retrievedAt": time.Now().UTC().Format(time.RFC3339), "fields": present, "synthetic": false}
		results = append(results, out)
	}
	return results, nil
}
