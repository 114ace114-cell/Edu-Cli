package daemon

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kocoro/edu-cli/internal/client"
	"github.com/kocoro/edu-cli/internal/config"
	"github.com/kocoro/edu-cli/internal/session"
)

func testServer(t *testing.T, llm client.LLMClient) *httptest.Server {
	t.Helper()
	data := t.TempDir()
	store, err := session.NewStore(filepath.Join(data, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	runner := &Runner{
		Cfg: config.Config{
			Model:     "script",
			Workspace: t.TempDir(),
			DataDir:   data,
		},
		LLM:      llm,
		Sessions: store,
	}
	return httptest.NewServer(NewServer(runner).Handler())
}

func TestMessageResumesSession(t *testing.T) {
	llm := &client.Scripted{Replies: []client.Message{
		{Content: "first"},
		{Content: "second"},
	}}
	srv := testServer(t, llm)
	defer srv.Close()

	res := postMessage(t, srv, `{"text":"hello"}`)
	if res.Reply != "first" || res.SessionID == "" {
		t.Fatalf("%+v", res)
	}
	res2 := postMessage(t, srv, `{"text":"again","session_id":"`+res.SessionID+`"}`)
	if res2.Reply != "second" || res2.SessionID != res.SessionID {
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
		{Content: "found"},
	}}
	srv := testServer(t, llm)
	defer srv.Close()

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
	if !strings.Contains(text, "event: done") || !strings.Contains(text, "found") {
		t.Fatalf("sse:\n%s", text)
	}
}

func TestEmptyTextRejected(t *testing.T) {
	srv := testServer(t, &client.Scripted{})
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
