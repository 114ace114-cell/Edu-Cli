package tui

import (
	"strings"
	"testing"
	"time"
)

func TestStartupHeaderMatchesHomeFrame(t *testing.T) {
	view := renderStartupHeader(headerTotalFrames-1, 90, "dev", "ollama/qwen3-vl:8b", "http://localhost:11434", `C:\work`, []Activity{{
		Title: "fadsfasd hello",
		When:  time.Now().Add(-72 * time.Hour),
		Msgs:  20,
	}}, 0, "default")
	for _, want := range []string{
		"Kocoro CLI",
		"Tips",
		"Try /research for deep analysis",
		"Recent activity",
		"fadsfasd hello",
		"20 msgs",
		"default",
		"ollama/qwen3-vl:8b",
		"http://loc",
		"▀",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q\n%s", want, view)
		}
	}
}

func TestLongReplyStaysInView(t *testing.T) {
	m := newModel(Options{
		Agent:     "default",
		Model:     "deepseek-flash",
		Endpoint:  "https://api.deepseek.com",
		CWD:       `C:\work`,
		Version:   "dev",
		Connected: true,
	})
	m.width = 48
	m.height = 30
	m.finishHeader()
	tail := "exercise the daemon endpoint"
	reply := "I interpreted the request as running the suite. " + tail
	updated, _ := m.Update(doneMsg{reply: reply})
	view := updated.View()
	if !strings.Contains(view, tail) {
		t.Fatalf("reply clipped:\n%s", view)
	}
}

func TestHomeViewHasComposer(t *testing.T) {
	m := newModel(Options{
		Agent:     "default",
		Model:     "qwen3-vl:8b",
		Endpoint:  "http://localhost:11434",
		CWD:       `C:\work`,
		Version:   "dev",
		Connected: true,
	})
	m.width = 90
	m.height = 28
	m.finishHeader()
	view := m.View()
	for _, want := range []string{
		"Kocoro CLI",
		"Connected to http://localhost:11434",
		"Ask Kocoro anything",
		"? for commands",
		"ollama/qwen3-vl:8b",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q\n%s", want, view)
		}
	}
}
