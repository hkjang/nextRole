package connectors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// concat is intentionally a small, non-executable mapping operation for stable
// compound identifiers. Escaping each value prevents delimiter collisions.
func validateSourceExpression(source string) error {
	if strings.HasPrefix(source, "concat:") {
		paths := strings.Split(strings.TrimPrefix(source, "concat:"), ",")
		if len(paths) < 2 || len(paths) > 8 {
			return errors.New("concat 매핑은 2~8개 원본 필드 경로를 쉼표로 구분하세요")
		}
		for _, path := range paths {
			segments, err := pathSegments(path)
			if err != nil || len(segments) == 0 || strings.Contains(path, "*") || strings.Contains(path, "[]") {
				return errors.New("concat 매핑에는 값 하나를 가리키는 원본 필드 경로만 허용합니다")
			}
		}
		return nil
	}
	_, err := pathSegments(source)
	return err
}

func concatenateFields(record map[string]any, source string) (string, error) {
	if err := validateSourceExpression(source); err != nil {
		return "", err
	}
	parts := []string{}
	for _, path := range strings.Split(strings.TrimPrefix(source, "concat:"), ",") {
		segments, _ := pathSegments(path)
		value, found := lookup(record, segments)
		if !found || value == nil {
			return "", errors.New("concat 매핑의 원본 필드가 없습니다")
		}
		switch value.(type) {
		case string, json.Number, float64, int, int64:
			if strings.TrimSpace(fmt.Sprint(value)) == "" {
				return "", errors.New("concat 매핑의 원본 값이 비어 있습니다")
			}
			parts = append(parts, url.QueryEscape(fmt.Sprint(value)))
		default:
			return "", errors.New("concat 매핑은 문자열 또는 숫자 필드만 사용할 수 있습니다")
		}
	}
	return strings.Join(parts, ":"), nil
}

func pathSegments(path string) ([]string, error) {
	path = strings.TrimSpace(path)
	if len(path) > 1000 {
		return nil, errors.New("필드 경로가 너무 깁니다")
	}
	if strings.HasPrefix(path, "=") {
		return nil, nil // mapping constant, interpreted by Normalize
	}
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")
	path = strings.ReplaceAll(path, "[]", ".*")
	path = strings.ReplaceAll(path, "[*]", ".*")
	path = strings.ReplaceAll(path, "[", ".")
	path = strings.ReplaceAll(path, "]", "")
	if path == "" {
		return nil, nil
	}
	parts := strings.Split(path, ".")
	if len(parts) > 32 {
		return nil, errors.New("필드 경로는 32단계까지 지원합니다")
	}
	for _, part := range parts {
		if part == "" || part == "__proto__" || part == "constructor" || part == "prototype" {
			return nil, errors.New("필드 경로가 올바르지 않습니다")
		}
	}
	return parts, nil
}

func targetSegments(path string) ([]string, error) {
	if path == "" || strings.HasPrefix(path, "=") {
		return nil, errors.New("대상 필드 이름을 입력하세요")
	}
	parts, err := pathSegments(path)
	if err != nil || len(parts) == 0 {
		return nil, errors.New("대상 필드 경로가 올바르지 않습니다")
	}
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err == nil {
			return nil, errors.New("대상 배열에는 숫자 인덱스 대신 []를 사용하세요")
		}
	}
	return parts, nil
}

func lookup(value any, path []string) (any, bool) {
	if len(path) == 0 {
		return value, true
	}
	if path[0] == "*" {
		items, ok := value.([]any)
		if !ok {
			// XML represents a single repeated element as an object. Treat it as
			// a singleton when the administrator explicitly requests an array.
			if value == nil {
				return []any{}, true
			}
			items = []any{value}
		}
		result := make([]any, 0, len(items))
		for _, item := range items {
			v, found := lookup(item, path[1:])
			if !found {
				return nil, false
			}
			result = append(result, v)
		}
		return result, true
	}
	switch current := value.(type) {
	case map[string]any:
		next, ok := current[path[0]]
		if !ok {
			return nil, false
		}
		return lookup(next, path[1:])
	case []any:
		i, err := strconv.Atoi(path[0])
		if err != nil || i < 0 || i >= len(current) {
			return nil, false
		}
		return lookup(current[i], path[1:])
	default:
		return nil, false
	}
}

func assign(existing any, path []string, value any) (any, error) {
	if len(path) == 0 {
		if existing != nil {
			return nil, errors.New("대상 필드 매핑 경로가 중복됩니다")
		}
		return value, nil
	}
	if path[0] == "*" {
		values, ok := value.([]any)
		if !ok {
			return nil, errors.New("배열 대상 필드에는 배열 원본을 매핑하세요")
		}
		arr, ok := existing.([]any)
		if existing == nil {
			arr = make([]any, len(values))
		} else if !ok || len(arr) != len(values) {
			return nil, errors.New("배열 매핑의 항목 수 또는 구조가 일치하지 않습니다")
		}
		for i, item := range values {
			var err error
			arr[i], err = assign(arr[i], path[1:], item)
			if err != nil {
				return nil, err
			}
		}
		return arr, nil
	}
	m, ok := existing.(map[string]any)
	if existing == nil {
		m = map[string]any{}
	} else if !ok {
		return nil, errors.New("대상 필드 매핑 경로가 중복됩니다")
	}
	v, err := assign(m[path[0]], path[1:], value)
	if err != nil {
		return nil, err
	}
	m[path[0]] = v
	return m, nil
}

// Normalize projects source records onto the application schema. Mapping keys
// are target paths and values are source paths. A value prefixed with '=' is a
// constant (JSON literals or plain strings). Missing mapped fields fail the whole
// batch rather than silently overwriting existing data with partial records.
func Normalize(records []map[string]any, c Config) ([]map[string]any, error) {
	if len(records) > MaxRecords {
		return nil, errors.New("한 번에 5,000개 레코드까지만 가져올 수 있습니다")
	}
	keys := make([]string, 0, len(c.Mapping))
	for key := range c.Mapping {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]map[string]any, 0, len(records))
	for index, record := range records {
		out := map[string]any{}
		if len(c.Mapping) == 0 {
			// Deep copy: normalization must not mutate the caller's source data.
			data, err := json.Marshal(record)
			if err != nil || json.Unmarshal(data, &out) != nil {
				return nil, errors.New("연동 레코드를 JSON 객체로 변환할 수 없습니다")
			}
		} else {
			for _, target := range keys {
				source := c.Mapping[target]
				var value any
				if strings.HasPrefix(source, "concat:") {
					var err error
					value, err = concatenateFields(record, source)
					if err != nil {
						return nil, fmt.Errorf("%d번째 레코드: %w", index+1, err)
					}
				} else if strings.HasPrefix(source, "=") {
					literal := strings.TrimPrefix(source, "=")
					if err := json.Unmarshal([]byte(literal), &value); err != nil {
						value = literal
					}
				} else {
					path, err := pathSegments(source)
					if err != nil {
						return nil, err
					}
					var ok bool
					value, ok = lookup(record, path)
					if !ok {
						return nil, fmt.Errorf("%d번째 레코드에 매핑한 원본 필드가 없습니다", index+1)
					}
				}
				parts, err := targetSegments(target)
				if err != nil {
					return nil, err
				}
				_, err = assign(out, parts, value)
				if err != nil {
					return nil, err
				}
			}
		}
		if err := coerceKnownFields(out, c.Dataset); err != nil {
			return nil, fmt.Errorf("%d번째 레코드: %w", index+1, err)
		}
		source, _ := out["source"].(map[string]any)
		if source == nil {
			source = map[string]any{}
		}
		if source["name"] == nil || source["name"] == "" {
			source["name"] = c.Name
		}
		if source["name"] == "" {
			source["name"] = "외부 연동 데이터"
		}
		if source["url"] == nil {
			// Never expose endpoint query parameters (which may contain secrets).
			if u, err := endpointURL(c.Endpoint); err == nil {
				u.RawQuery, u.Fragment = "", ""
				source["url"] = u.String()
			} else {
				source["url"] = ""
			}
		}
		if source["synthetic"] == nil {
			source["synthetic"] = false
		}
		out["source"] = source
		result = append(result, out)
	}
	return result, nil
}

func coerceKnownFields(out map[string]any, dataset string) error {
	for _, key := range []string{"id", "title", "description", "organization", "region", "url", "salary", "deadline", "category", "domain", "education", "salaryRange"} {
		if value, ok := out[key]; ok && value != nil {
			switch value.(type) {
			case string:
			case json.Number, float64, int, int64:
				out[key] = fmt.Sprint(value)
			}
		}
	}
	for _, key := range []string{"minExperience", "cost"} {
		if err := coerceNumber(out, key); err != nil {
			return err
		}
	}
	for _, key := range []string{"skills", "regions", "bridgeIds"} {
		if key == "skills" && dataset == "occupations" {
			continue
		}
		if value, ok := out[key].(string); ok {
			items := []any{}
			// JSON arrays in CSV / SQL JSON columns are accepted; plain values
			// use semicolons or pipes to avoid splitting skill names on commas.
			if strings.HasPrefix(strings.TrimSpace(value), "[") {
				if json.Unmarshal([]byte(value), &items) != nil {
					return errors.New("목록 필드의 JSON 배열이 올바르지 않습니다")
				}
			} else {
				for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == ';' || r == '|' }) {
					if part = strings.TrimSpace(part); part != "" {
						items = append(items, part)
					}
				}
			}
			out[key] = items
		}
	}
	if dataset == "occupations" {
		if raw, ok := out["skills"].(string); ok {
			var skills []any
			if err := json.Unmarshal([]byte(raw), &skills); err != nil {
				return errors.New("직무 역량은 name, level, weight를 가진 JSON 배열이어야 합니다")
			}
			out["skills"] = skills
		}
		if skills, ok := out["skills"].([]any); ok {
			for _, raw := range skills {
				if skill, ok := raw.(map[string]any); ok {
					for _, key := range []string{"level", "weight"} {
						if err := coerceNumber(skill, key); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

func coerceNumber(m map[string]any, key string) error {
	if s, ok := m[key].(string); ok {
		if strings.TrimSpace(s) == "" {
			delete(m, key)
			return nil
		}
		// JSON numeric parsing rejects NaN and infinities accepted by ParseFloat.
		var number json.Number
		if err := json.Unmarshal([]byte(strings.TrimSpace(s)), &number); err != nil {
			return errors.New("숫자 필드에 숫자가 아닌 값이 포함되어 있습니다")
		}
		value, err := number.Float64()
		if err != nil {
			return errors.New("숫자 필드 값이 지원 범위를 벗어났습니다")
		}
		m[key] = value
	}
	return nil
}
