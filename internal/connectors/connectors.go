// Package connectors reads administrator-configured data sources. It never logs
// credentials, URLs, response bodies or database errors; callers may safely show
// its errors in the administration UI. Only administrators may configure or run
// a connector, since intranet addresses are intentionally supported.
package connectors

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	MaxResponseBytes = 10 << 20
	MaxRecords       = 5000
	FetchTimeout     = 20 * time.Second
)

// Config is persisted encrypted by the server. APIKey and DSN are write-only in
// management responses; Mask returns a response-safe copy. Params and Body must
// not contain credentials: use APIKeyParam or AuthHeader instead.
type Config struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Enabled     bool              `json:"enabled"`
	Endpoint    string            `json:"endpoint"`
	APIKey      string            `json:"apiKey,omitempty"`
	AuthHeader  string            `json:"authHeader,omitempty"`
	APIKeyParam string            `json:"apiKeyParam,omitempty"`
	Params      map[string]string `json:"params,omitempty"`
	Method      string            `json:"method,omitempty"`
	Body        string            `json:"body,omitempty"`
	DSN         string            `json:"dsn,omitempty"`
	Query       string            `json:"query,omitempty"`
	RootPath    string            `json:"rootPath"`
	Mapping     map[string]string `json:"mapping"`
	Dataset     string            `json:"dataset"`
	LastSync    string            `json:"lastSync,omitempty"`
	LastError   string            `json:"lastError,omitempty"`
	HasAPIKey   bool              `json:"hasApiKey,omitempty"`
	HasDSN      bool              `json:"hasDsn,omitempty"`
}

func (c Config) Mask() Config {
	c.HasAPIKey = c.APIKey != ""
	c.HasDSN = c.DSN != ""
	c.APIKey, c.DSN = "", ""
	return c
}

// Validate checks the configuration without making a network connection. Fetch
// may run a disabled connector so administrators can test it before enabling it.
func Validate(c Config) error {
	switch c.Dataset {
	case "jobs", "training", "occupations":
	default:
		return errors.New("데이터 종류는 jobs, training, occupations 중 선택하세요")
	}
	if len(c.Mapping) > 200 {
		return errors.New("필드 매핑은 200개까지 설정할 수 있습니다")
	}
	for target, path := range c.Mapping {
		if _, err := targetSegments(target); err != nil {
			return err
		}
		if err := validateSourceExpression(path); err != nil {
			return err
		}
	}
	if _, err := pathSegments(c.RootPath); err != nil {
		return err
	}
	switch c.Type {
	case "json", "xml", "csv":
		if _, err := endpointURL(c.Endpoint); err != nil {
			return err
		}
		method := strings.ToUpper(c.Method)
		if method != "" && method != "GET" && method != "POST" {
			return errors.New("API 요청 방식은 GET 또는 POST만 지원합니다")
		}
		if len(c.Body) > 1<<20 {
			return errors.New("API 요청 본문은 1 MiB 이하여야 합니다")
		}
		if c.AuthHeader != "" && !validHeaderName(c.AuthHeader) {
			return errors.New("인증 헤더 이름이 올바르지 않습니다")
		}
		if strings.ContainsAny(c.APIKey, "\r\n") {
			return errors.New("API 키에 줄바꿈을 사용할 수 없습니다")
		}
		if len(c.Params) > 100 {
			return errors.New("API 요청 매개변수는 100개까지 설정할 수 있습니다")
		}
	case "postgres", "mysql":
		if c.DSN == "" {
			return errors.New("데이터베이스 연결 문자열을 입력하세요")
		}
		return ValidateQuery(c.Query)
	default:
		return errors.New("지원하지 않는 연동 형식입니다")
	}
	return nil
}

// Fetch reads one configured page or query and maps it to the internal schema.
// It deliberately does not guess pagination or invent records on an API error.
func Fetch(ctx context.Context, c Config) ([]map[string]any, error) {
	if err := Validate(c); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, FetchTimeout)
	defer cancel()
	var records []map[string]any
	var err error
	if c.Type == "postgres" || c.Type == "mysql" {
		records, err = fetchSQL(ctx, c)
	} else {
		records, err = fetchHTTP(ctx, c)
	}
	if err != nil {
		return nil, err
	}
	return Normalize(records, c)
}

func endpointURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" {
		return nil, errors.New("연동 주소는 사용자 인증정보와 fragment가 없는 HTTP(S) URL이어야 합니다")
	}
	return u, nil
}

func validHeaderName(s string) bool {
	if len(s) == 0 || len(s) > 100 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", c)) {
			return false
		}
	}
	return true
}

func fetchHTTP(ctx context.Context, c Config) ([]map[string]any, error) {
	u, _ := endpointURL(c.Endpoint) // validated by Fetch
	q := u.Query()
	for key, value := range c.Params {
		q.Set(key, value)
	}
	if c.APIKey != "" && c.APIKeyParam != "" {
		q.Set(c.APIKeyParam, c.APIKey)
	}
	u.RawQuery = q.Encode()
	method := strings.ToUpper(c.Method)
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), strings.NewReader(c.Body))
	if err != nil {
		return nil, errors.New("API 요청을 만들 수 없습니다")
	}
	req.Header.Set("Accept", "application/json, application/xml, text/xml, text/csv")
	req.Header.Set("User-Agent", "NextRole/1.0")
	if method == http.MethodPost {
		contentType := "application/json"
		if c.Type == "xml" {
			contentType = "application/xml"
		} else if c.Type == "csv" {
			contentType = "text/csv"
		}
		req.Header.Set("Content-Type", contentType+"; charset=utf-8")
	}
	if c.APIKey != "" && c.APIKeyParam == "" {
		header := c.AuthHeader
		if header == "" {
			header = "Authorization"
		}
		req.Header.Set(header, c.APIKey)
	}
	// Do not follow redirects: custom authentication headers and query credentials
	// must never be forwarded to an unreviewed destination, including a redirect
	// on the same origin. Administrators should configure the canonical URL.
	client := &http.Client{Timeout: FetchTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, errors.New("API 요청이 취소되었거나 제한 시간을 초과했습니다")
		}
		return nil, errors.New("API 연결에 실패했습니다. 주소, 인증정보와 네트워크를 확인하세요")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API가 HTTP %d 상태를 반환했습니다", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))
	if err != nil {
		return nil, errors.New("API 응답을 읽을 수 없습니다")
	}
	if len(data) > MaxResponseBytes {
		return nil, errors.New("API 응답이 10 MiB 제한을 초과했습니다. 조회 범위를 줄이세요")
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	if c.Type == "csv" {
		if c.RootPath != "" && c.RootPath != "$" {
			return nil, errors.New("CSV에는 목록 경로를 지정하지 마세요")
		}
		return parseCSV(data)
	}
	var root any
	if c.Type == "json" {
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		if err := decoder.Decode(&root); err != nil {
			return nil, errors.New("JSON 응답 형식이 올바르지 않습니다")
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			return nil, errors.New("JSON 응답에는 하나의 문서만 허용합니다")
		}
	} else {
		root, err = parseXML(data)
		if err != nil {
			return nil, err
		}
	}
	if err := upstreamError(root); err != nil {
		return nil, err
	}
	path, _ := pathSegments(c.RootPath)
	selected, ok := lookup(root, path)
	if !ok {
		return nil, errors.New("목록 경로에서 데이터를 찾지 못했습니다. 응답 구조와 rootPath를 확인하세요")
	}
	return objectRecords(selected)
}

func objectRecords(selected any) ([]map[string]any, error) {
	if selected == nil || selected == "" {
		return []map[string]any{}, nil
	}
	if record, ok := selected.(map[string]any); ok {
		return []map[string]any{record}, nil
	}
	items, ok := selected.([]any)
	if !ok {
		return nil, errors.New("목록 경로는 객체 또는 객체 배열을 가리켜야 합니다")
	}
	if len(items) > MaxRecords {
		return nil, errors.New("한 번에 5,000개 레코드까지만 가져올 수 있습니다")
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		record, ok := item.(map[string]any)
		if !ok {
			return nil, errors.New("목록에 객체가 아닌 항목이 포함되어 있습니다")
		}
		result = append(result, record)
	}
	return result, nil
}

func parseCSV(data []byte) ([]map[string]any, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	headers, err := reader.Read()
	if err != nil {
		return nil, errors.New("CSV 헤더를 읽을 수 없습니다")
	}
	if len(headers) > 200 {
		return nil, errors.New("CSV 열은 200개까지 지원합니다")
	}
	seen := map[string]bool{}
	for i, header := range headers {
		header = strings.TrimSpace(header)
		if header == "" || seen[header] {
			return nil, errors.New("CSV 헤더는 비어 있거나 중복될 수 없습니다")
		}
		headers[i], seen[header] = header, true
	}
	result := []map[string]any{}
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.New("CSV 행 형식 또는 열 수가 올바르지 않습니다")
		}
		if len(result) >= MaxRecords {
			return nil, errors.New("한 번에 5,000개 레코드까지만 가져올 수 있습니다")
		}
		record := make(map[string]any, len(headers))
		for i, name := range headers {
			record[name] = row[i]
		}
		result = append(result, record)
	}
	return result, nil
}

// Reject common error envelopes without exposing the provider's message (which
// can contain the submitted authentication key). Schema validation still belongs
// to the importing service and must run before any records are persisted.
func upstreamError(root any) error {
	m, ok := root.(map[string]any)
	if !ok {
		return nil
	}
	for key, value := range m {
		switch strings.ToLower(key) {
		case "error", "errors", "errorreport":
			if value != nil && value != "" {
				return errors.New("연동 API가 오류 응답을 반환했습니다. 인증키와 요청 조건을 확인하세요")
			}
		case "resultcode", "errorcode":
			code := fmt.Sprint(value)
			if code != "" && code != "0" && code != "00" && code != "0000" && code != "200" && code != "NORMAL_SERVICE" {
				return errors.New("연동 API가 실패 코드를 반환했습니다. 인증키와 요청 조건을 확인하세요")
			}
		}
		if nested, ok := value.(map[string]any); ok {
			if err := upstreamError(nested); err != nil {
				return err
			}
		}
	}
	return nil
}
