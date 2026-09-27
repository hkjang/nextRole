package connectors

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"
)

type xmlNode struct {
	name     string
	children map[string]any
	text     strings.Builder
}

// parseXML uses Go's non-expanding XML parser and additionally rejects DTDs and
// declarations. No external entities, external resources or filesystem reads.
func parseXML(data []byte) (any, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	stack := []*xmlNode{}
	result := map[string]any{}
	tokens := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.New("XML 응답 형식이 올바르지 않습니다 (UTF-8 필요)")
		}
		tokens++
		if tokens > 500000 {
			return nil, errors.New("XML 응답의 요소 수가 제한을 초과했습니다")
		}
		switch t := token.(type) {
		case xml.Directive:
			return nil, errors.New("XML DTD와 외부 엔티티는 지원하지 않습니다")
		case xml.StartElement:
			if len(stack) >= 64 {
				return nil, errors.New("XML 중첩 깊이가 제한을 초과했습니다")
			}
			node := &xmlNode{name: t.Name.Local, children: map[string]any{}}
			for _, attr := range t.Attr {
				node.children["@"+attr.Name.Local] = attr.Value
			}
			stack = append(stack, node)
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].text.Write([]byte(t))
			} else if strings.TrimSpace(string(t)) != "" {
				return nil, errors.New("XML 루트 밖에 텍스트가 있습니다")
			}
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, errors.New("XML 응답 형식이 올바르지 않습니다")
			}
			node := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			var value any = node.children
			text := strings.TrimSpace(node.text.String())
			if len(node.children) == 0 {
				value = text
			} else if text != "" {
				node.children["#text"] = text
			}
			parent := result
			if len(stack) > 0 {
				parent = stack[len(stack)-1].children
			} else if len(result) != 0 {
				return nil, errors.New("XML 응답에는 하나의 루트만 허용합니다")
			}
			if previous, ok := parent[node.name]; ok {
				if list, ok := previous.([]any); ok {
					parent[node.name] = append(list, value)
				} else {
					parent[node.name] = []any{previous, value}
				}
			} else {
				parent[node.name] = value
			}
		}
	}
	if len(result) == 0 || len(stack) > 0 {
		return nil, errors.New("XML 응답이 비어 있거나 완전하지 않습니다")
	}
	return result, nil
}
