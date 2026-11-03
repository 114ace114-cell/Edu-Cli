package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIToolRoundTrip(t *testing.T) {
	var sawTool bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer k" {
			http.Error(w, "auth", http.StatusUnauthorized)
			return
		}
		sawTool = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok","tool_calls":[{"id":"call_1","type":"function","function":{"name":"file_read","arguments":"{\"path\":\"a.txt\"}"}}]}}]}`))
	}))
	defer srv.Close()

	msg, err := (&OpenAI{BaseURL: srv.URL, APIKey: "k", HTTP: srv.Client()}).Complete(Request{
		Model:    "fake",
		Messages: []Message{{Role: "user", Content: "read"}},
		Tools:    []ToolDef{{Name: "file_read", Description: "read", Parameters: map[string]any{"type": "object"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !sawTool || msg.Content != "ok" || len(msg.ToolCalls) != 1 || msg.ToolCalls[0].Name != "file_read" {
		t.Fatalf("%+v", msg)
	}
	if !strings.Contains(msg.ToolCalls[0].Arguments, "a.txt") {
		t.Fatalf("args %s", msg.ToolCalls[0].Arguments)
	}
}
