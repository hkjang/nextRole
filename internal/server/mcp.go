package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func (a *App) mcp(w http.ResponseWriter, r *http.Request) {
	var req rpcRequest
	if !readJSON(w, r, &req) {
		return
	}
	rpcErr := func(code int, message string) {
		respond(w, 200, map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": code, "message": message}})
	}
	result := func(v any) { respond(w, 200, map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": v}) }
	if req.JSONRPC != "2.0" || req.Method == "" {
		rpcErr(-32600, "잘못된 JSON-RPC 요청입니다")
		return
	}
	if len(req.ID) == 0 {
		if strings.HasPrefix(req.Method, "notifications/") {
			w.WriteHeader(202)
			return
		}
		rpcErr(-32600, "요청 ID가 필요합니다")
		return
	}
	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &p)
		version := p.ProtocolVersion
		if !contains([]string{"2025-03-26", "2025-06-18", "2025-11-25"}, version) {
			version = "2025-11-25"
		}
		result(map[string]any{"protocolVersion": version, "capabilities": map[string]any{"tools": map[string]any{"listChanged": false}}, "serverInfo": map[string]string{"name": "NextRole", "version": a.version}, "instructions": "경력 점수는 취업 확률이 아닙니다. synthetic=true 자료는 합성 예시입니다. 사용자별 API 키와 최소 범위 권한을 사용하세요."})
	case "ping":
		result(map[string]any{})
	case "tools/list":
		tools := []map[string]any{}
		for _, t := range mcpTools() {
			if a.mcpScope(r, t["scope"].(string)) {
				delete(t, "scope")
				tools = append(tools, t)
			}
		}
		result(map[string]any{"tools": tools})
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if json.Unmarshal(req.Params, &p) != nil {
			rpcErr(-32602, "도구 인자를 확인하세요")
			return
		}
		scope := ""
		for _, t := range mcpTools() {
			if t["name"] == p.Name {
				scope = t["scope"].(string)
			}
		}
		if scope == "" {
			rpcErr(-32602, "지원하지 않는 도구입니다")
			return
		}
		if !a.mcpScope(r, scope) {
			rpcErr(-32003, "도구에 필요한 API 키 권한이 없습니다")
			return
		}
		var v any
		var e error
		switch p.Name {
		case "career_profile":
			v, e = a.loadProfile(r)
		case "job_search":
			var in struct {
				Query string `json:"query"`
			}
			if len(p.Arguments) > 0 && json.Unmarshal(p.Arguments, &in) != nil {
				rpcErr(-32602, "query 문자열을 확인하세요")
				return
			}
			jobs, err := a.jobs(r.Context())
			e = err
			matches := []any{}
			for _, j := range jobs {
				b, _ := json.Marshal(j)
				if in.Query == "" || strings.Contains(strings.ToLower(string(b)), strings.ToLower(in.Query)) {
					matches = append(matches, j)
				}
			}
			v = matches
		case "career_simulate":
			var in simulationInput
			if json.Unmarshal(p.Arguments, &in) != nil {
				rpcErr(-32602, "시뮬레이션 인자를 확인하세요")
				return
			}
			v, e = a.calculate(r, in)
		case "career_opportunities":
			var in struct {
				JobID string `json:"jobId"`
			}
			if json.Unmarshal(p.Arguments, &in) != nil {
				rpcErr(-32602, "jobId를 입력하세요")
				return
			}
			v, e = a.opportunities(r, in.JobID)
		}
		if e != nil {
			result(map[string]any{"isError": true, "content": []map[string]string{{"type": "text", "text": "데이터를 조회할 수 없습니다. 목표 직무와 프로필을 확인하세요."}}})
			return
		}
		b, _ := json.Marshal(v)
		result(map[string]any{"content": []map[string]string{{"type": "text", "text": string(b)}}})
	default:
		rpcErr(-32601, "지원하지 않는 메서드입니다")
	}
}
func (a *App) mcpScope(r *http.Request, scope string) bool {
	ident := who(r)
	if ident.KeyID == "" {
		return true
	}
	s, e := a.settings(r.Context())
	return e == nil && contains(ident.Scopes, scope) && contains(s.Security.AllowedKeyScopes, scope)
}
func mcpTools() []map[string]any {
	obj := func(props map[string]any, required []string) map[string]any {
		return map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false}
	}
	str := map[string]string{"type": "string"}
	annotation := map[string]any{"readOnlyHint": true, "destructiveHint": false, "openWorldHint": false}
	return []map[string]any{
		{"name": "career_profile", "description": "현재 인증 사용자의 경력 프로필 조회", "scope": "profile:read", "inputSchema": obj(map[string]any{}, []string{}), "annotations": annotation},
		{"name": "job_search", "description": "출처가 있는 직무 카탈로그 검색 (합성 여부 포함)", "scope": "jobs:read", "inputSchema": obj(map[string]any{"query": str}, []string{}), "annotations": annotation},
		{"name": "career_simulate", "description": "역량 추가 가정의 적합도·격차·ROI·경로 계산. 프로필을 변경하거나 결과를 저장하지 않습니다.", "scope": "simulate:write", "inputSchema": obj(map[string]any{"jobId": str, "months": map[string]any{"type": "integer", "enum": []int{3, 6, 12}}, "addedSkills": map[string]any{"type": "array", "items": obj(map[string]any{"name": str, "level": map[string]any{"type": "number", "minimum": 0, "maximum": 5}}, []string{"name", "level"})}}, []string{"jobId"}), "annotations": annotation},
		{"name": "career_opportunities", "description": "목표 직무와 연결된 채용·훈련 및 실제 공고 역량 빈도 조회", "scope": "jobs:read", "inputSchema": obj(map[string]any{"jobId": str}, []string{"jobId"}), "annotations": annotation},
	}
}
func (a *App) openapi(w http.ResponseWriter, r *http.Request) {
	paths := map[string]any{}
	add := func(path, method, summary string, schema map[string]any) {
		op := map[string]any{"summary": summary, "responses": map[string]any{"200": map[string]any{"description": "성공"}, "400": map[string]any{"description": "입력 오류"}, "401": map[string]any{"description": "인증 필요"}, "403": map[string]any{"description": "권한 부족"}}, "security": []map[string]any{{"bearerAuth": []string{}}, {"sessionCookie": []string{}}}}
		if schema != nil {
			op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": schema}}}
		}
		if paths[path] == nil {
			paths[path] = map[string]any{}
		}
		paths[path].(map[string]any)[method] = op
	}
	for _, item := range []struct{ path, method, summary string }{{"/profile", "get", "내 경력 조회 (profile:read)"}, {"/jobs", "get", "직무 검색 (jobs:read)"}, {"/recommendations", "get", "경력 기반 직무 추천 (profile:read)"}, {"/simulations", "get", "저장한 시뮬레이션 조회 (profile:read)"}, {"/opportunities", "get", "채용 및 훈련 조회, jobId 필요 (jobs:read)"}, {"/roadmap", "get", "로드맵 진척 조회 (profile:read)"}, {"/market", "get", "실제 공고 표본의 지역·역량·급여·수집 추세 (jobs:read)"}} {
		add(item.path, item.method, item.summary, nil)
	}
	skill := map[string]any{"type": "object", "required": []string{"name", "level"}, "properties": map[string]any{"name": map[string]string{"type": "string"}, "level": map[string]any{"type": "number", "minimum": 0, "maximum": 5}, "years": map[string]any{"type": "number", "minimum": 0, "maximum": 80}, "confidence": map[string]any{"type": "string", "enum": []string{"explicit", "inferred", "review"}}}}
	add("/simulate", "post", "What-if 시뮬레이션 (simulate:write)", map[string]any{"type": "object", "required": []string{"jobId"}, "properties": map[string]any{"jobId": map[string]string{"type": "string"}, "months": map[string]any{"type": "integer", "enum": []int{3, 6, 12}}, "save": map[string]string{"type": "boolean"}, "addedSkills": map[string]any{"type": "array", "items": skill}}})
	add("/profile", "put", "내 경력 저장 (profile:write)", map[string]any{"type": "object", "properties": map[string]any{"name": map[string]string{"type": "string"}, "currentRole": map[string]string{"type": "string"}, "yearsExperience": map[string]string{"type": "number"}, "skills": map[string]any{"type": "array", "items": skill}, "narrative": map[string]string{"type": "string"}}})
	add("/ai/stream", "post", "AI SSE 스트림 (ai:use); delta, profile, done, error 이벤트", map[string]any{"type": "object", "required": []string{"task"}, "properties": map[string]any{"task": map[string]any{"type": "string", "enum": []string{"parse", "explain", "plan"}}, "text": map[string]string{"type": "string"}, "jobId": map[string]string{"type": "string"}}})
	add("/profile/parse", "post", "오프라인 경력 파서 (profile:write)", map[string]any{"type": "object", "required": []string{"text"}, "properties": map[string]any{"text": map[string]string{"type": "string"}}})
	for _, p := range []string{"/jobs", "/opportunities"} {
		op := paths[p].(map[string]any)["get"].(map[string]any)
		name := "q"
		required := false
		if p == "/opportunities" {
			name = "jobId"
			required = true
		}
		op["parameters"] = []map[string]any{{"name": name, "in": "query", "required": required, "schema": map[string]string{"type": "string"}}}
	}
	respond(w, 200, map[string]any{"openapi": "3.1.0", "info": map[string]string{"title": "NextRole API", "version": a.version, "description": "개인 API는 Bearer 키 scope로 제한됩니다. 사용자·키·관리자 설정은 브라우저 세션 전용입니다. MCP endpoint /mcp (Streamable HTTP JSON response). 전체 관리자 계약은 docs/architecture.md 참조."}, "servers": []map[string]string{{"url": "/api/v1"}}, "paths": paths, "components": map[string]any{"securitySchemes": map[string]any{"bearerAuth": map[string]string{"type": "http", "scheme": "bearer"}, "sessionCookie": map[string]string{"type": "apiKey", "in": "cookie", "name": "nr_session"}}}})
}
