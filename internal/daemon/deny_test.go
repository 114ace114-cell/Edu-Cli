package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kocoro/skeleton/internal/client"
)

func TestDeniedCommandDoesNotRun(t *testing.T) {
	llm := &client.Scripted{Replies: []client.Message{
		{ToolCalls: []client.ToolCall{{ID: "d", Name: "bash", Arguments: `{"command":"rm keep.txt"}`}}},
		{Content: "blocked"},
	}}
	srv, workspace := testServer(t, llm)
	defer srv.Close()
	if err := os.WriteFile(filepath.Join(workspace, "keep.txt"), []byte("safe"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := postMessage(t, srv, `{"text":"delete keep.txt"}`)
	if res.Reply != "blocked" {
		t.Fatal(res.Reply)
	}
	body, err := os.ReadFile(filepath.Join(workspace, "keep.txt"))
	if err != nil || string(body) != "safe" {
		t.Fatalf("file %q %v", body, err)
	}
	sessResp, err := httpGet(t, srv.URL+"/sessions/"+res.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sessResp, "denied") {
		t.Fatalf("session did not record the denial: %s", sessResp)
	}
}
