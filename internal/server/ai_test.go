package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStreamingCompletion(t *testing.T) {
	for _, tc := range []struct {
		name, body, contentType string
		wantErr                 bool
	}{
		{"success", "data: {\"choices\":[{\"delta\":{\"content\":\"안녕\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"하세요\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n", "text/event-stream", false},
		{"disconnect", "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n", "text/event-stream", true},
		{"length", "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"},\"finish_reason\":\"length\"}]}\n\n", "text/event-stream", true},
		{"non streaming", "{\"choices\":[]}", "application/json", true},
		{"malformed", "data: broken\n\n", "text/event-stream", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var v map[string]any
				_ = json.NewDecoder(r.Body).Decode(&v)
				if v["stream"] != true || v["max_completion_tokens"] != float64(2048) {
					t.Error("stream/token limit not forwarded")
				}
				if r.Header.Get("Authorization") != "Bearer secret" {
					t.Error("auth missing")
				}
				w.Header().Set("Content-Type", tc.contentType)
				fmt.Fprint(w, tc.body)
			}))
			defer upstream.Close()
			s := defaults()
			s.AI.BaseURL = upstream.URL
			s.AI.APIKey = "secret"
			s.AI.TokenParameter = "max_completion_tokens"
			var out strings.Builder
			err := streamCompletion(context.Background(), s, "system", "input", 2048, func(v string) error { out.WriteString(v); return nil })
			if (err != nil) != tc.wantErr {
				t.Fatalf("error=%v", err)
			}
			if !tc.wantErr && out.String() != "안녕하세요" {
				t.Fatalf("output=%s", out.String())
			}
		})
	}
}
func TestAIRejectsContextOverflow(t *testing.T) {
	s := defaults()
	s.AI.ContextWindow = 1024
	err := streamCompletion(context.Background(), s, "system", "prompt", 1024, func(string) error { return nil })
	if err == nil {
		t.Fatal("budget overflow allowed")
	}
}
func TestProviderDefaults(t *testing.T) {
	for _, typ := range []string{"google", "linkedin", "kakao", "naver"} {
		p := preset(Provider{Type: typ})
		if typ == "naver" {
			if p.SubjectField != "response.id" || p.TokenURL == "" {
				t.Fatal("naver defaults")
			}
		} else if !validURL(p.Issuer) || !strings.Contains(p.Scopes, "openid") {
			t.Fatalf("%s defaults", typ)
		}
	}
	if preset(Provider{Type: "linkedin"}).Issuer != "https://www.linkedin.com" {
		t.Fatal("LinkedIn issuer must match signed tokens")
	}
}
