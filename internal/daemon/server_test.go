package daemon

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kocoro/skeleton/internal/audit"
	"github.com/kocoro/skeleton/internal/client"
	"github.com/kocoro/skeleton/internal/config"
	"github.com/kocoro/skeleton/internal/hooks"
	"github.com/kocoro/skeleton/internal/session"
	"github.com/kocoro/skeleton/internal/skills"
)

func testServer(t *testing.T, llm client.LLMClient) (*httptest.Server, string) {
	t.Helper()
	workspace := t.TempDir()
	data := t.TempDir()
	store, err := session.NewStore(filepath.Join(data, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	logger, err := audit.New(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(data, "skills", "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "skills", "demo", "SKILL.md"), []byte("be brief"), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := skills.Load(data)
	if err != nil {
		t.Fatal(err)
	}
	runner := &Runner{
		Cfg: config.Config{
			Model:          "script",
			MaxIterations:  8,
			Workspace:      workspace,
			DataDir:        data,
			DeniedCommands: nil,
		},
		LLM:      llm,
		Sessions: store,
		Skills:   loaded,
		Hooks:    &hooks.Runner{},
		Audit:    logger,
	}
	return httptest.NewServer(NewServer(runner).Handler()), workspace
}

func TestMessageWritesFileAndResumesSession(t *testing.T) {
	llm := &client.Scripted{Replies: []client.Message{
		{ToolCalls: []client.ToolCall{{ID: "w", Name: "file_write", Arguments: `{"path":"note.txt","content":"alpha"}`}}},
		{Content: "wrote note"},
		{ToolCalls: []client.ToolCall{{ID: "s", Name: "use_skill", Arguments: `{"name":"demo"}`}}},
		{Content: "skill says be brief"},
	}}
	srv, workspace := testServer(t, llm)
	defer srv.Close()

	res := postMessage(t, srv, `{"text":"create a note"}`)
	if res.Reply != "wrote note" || res.SessionID == "" {
		t.Fatalf("%+v", res)
	}
	body, err := os.ReadFile(filepath.Join(workspace, "note.txt"))
	if err != nil || string(body) != "alpha" {
		t.Fatalf("note %q %v", body, err)
	}

	res2 := postMessage(t, srv, `{"text":"load the skill","session_id":"`+res.SessionID+`"}`)
	if res2.Reply != "skill says be brief" || res2.SessionID != res.SessionID {
		t.Fatalf("%+v", res2)
	}
	if res2.RouteKey != "session:"+res.SessionID {
		t.Fatalf("route %s", res2.RouteKey)
	}

	got, err := http.Get(srv.URL + "/sessions/" + res.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	defer got.Body.Close()
	var sess session.Session
	if err := json.NewDecoder(got.Body).Decode(&sess); err != nil {
		t.Fatal(err)
	}
	var users int
	for _, m := range sess.Messages {
		if m.Role == "user" {
			users++
		}
	}
	if users != 2 {
		t.Fatalf("users = %d", users)
	}
}

func TestSSEAndHealth(t *testing.T) {
	llm := &client.Scripted{Replies: []client.Message{
		{ToolCalls: []client.ToolCall{{ID: "g", Name: "grep", Arguments: `{"pattern":"alpha","path":"note.txt"}`}}},
		{Content: "found"},
	}}
	srv, workspace := testServer(t, llm)
	defer srv.Close()
	if err := os.WriteFile(filepath.Join(workspace, "note.txt"), []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}

	health, err := http.Get(srv.URL + "/health")
	if err != nil || health.StatusCode != 200 {
		t.Fatalf("health %v", err)
	}
	health.Body.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/message", strings.NewReader(`{"text":"search"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw := new(bytes.Buffer)
	_, _ = raw.ReadFrom(resp.Body)
	text := raw.String()
	if !strings.Contains(text, "event: tool_status") || !strings.Contains(text, "event: done") || !strings.Contains(text, "found") {
		t.Fatalf("sse:\n%s", text)
	}
}

func TestEmptyTextRejected(t *testing.T) {
	srv, _ := testServer(t, &client.Scripted{})
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/message", "application/json", strings.NewReader(`{"text":"  "}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func httpGet(t *testing.T, url string) (string, error) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	return string(raw), err
}

func postMessage(t *testing.T, srv *httptest.Server, body string) Result {
	t.Helper()
	resp, err := http.Post(srv.URL+"/message", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var res Result
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}
	return res
}
