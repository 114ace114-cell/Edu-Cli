package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestStartupHeaderMatchesHomeFrame(t *testing.T) {
	view := renderStartupHeader(headerTotalFrames-1, 90, "dev", "ollama/qwen3-vl:8b", "http://localhost:11434", `C:\work`, []Activity{{
		Title: "fadsfasd hello",
		When:  time.Now().Add(-72 * time.Hour),
		Msgs:  20,
	}}, 0, "default")
	for _, want := range []string{
		"Edu CLI",
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

func TestPageUpShowsEarlierLines(t *testing.T) {
	m := newModel(Options{
		Agent:     "default",
		Model:     "deepseek-flash",
		Endpoint:  "https://api.deepseek.com",
		CWD:       `C:\work`,
		Version:   "dev",
		Connected: true,
	})
	m.width = 40
	m.height = 16
	m.finishHeader()
	var lines []string
	for i := 0; i < 40; i++ {
		lines = append(lines, fmt.Sprintf("scroll-line-%02d", i))
	}
	m.append(strings.Join(lines, "\n"))
	if !strings.Contains(m.View(), "scroll-line-39") {
		t.Fatalf("bottom missing:\n%s", m.View())
	}
	if strings.Contains(m.View(), "scroll-line-00") {
		t.Fatal("viewport already shows the first line")
	}
	before := m.viewport.YOffset
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	next := updated.(*Model)
	if next.viewport.YOffset >= before {
		t.Fatalf("page up left offset at %d", next.viewport.YOffset)
	}
	for i := 0; i < 20 && !strings.Contains(next.View(), "scroll-line-00"); i++ {
		updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyPgUp})
		next = updated.(*Model)
	}
	if !strings.Contains(next.View(), "scroll-line-00") {
		t.Fatalf("never reached the top:\n%s", next.View())
	}
	before = next.viewport.YOffset
	updated, _ = next.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
	if updated.(*Model).viewport.YOffset <= before {
		t.Fatal("wheel did not scroll")
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
		"Edu CLI",
		"Connected to http://localhost:11434",
		"Ask Edu anything",
		"? for commands",
		"ollama/qwen3-vl:8b",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q\n%s", want, view)
		}
	}
}

func TestModelCommandListsDeepSeek(t *testing.T) {
	m := newModel(Options{
		Agent:    "default",
		Model:    "qwen3-vl:8b",
		Endpoint: "http://localhost:11434",
		CWD:      `C:\work`,
		Version:  "dev",
		Models: func() []ModelChoice {
			return []ModelChoice{{Name: "deepseek-chat"}, {Name: "qwen3-vl:8b", Detail: "6.2 GB"}}
		},
		ApplyModel: func(name string) (string, error) {
			if name == "deepseek-chat" {
				return "https://api.deepseek.com", nil
			}
			return "", fmt.Errorf("unknown model %s", name)
		},
	})
	m.width = 80
	m.height = 28
	m.finishHeader()
	m.textarea.SetValue("/model")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view := updated.View()
	if !strings.Contains(view, "deepseek-chat") || !strings.Contains(view, "→ qwen3-vl:8b") {
		t.Fatalf("model list:\n%s", view)
	}
	next := updated.(*Model)
	next.textarea.SetValue("/model deepseek-chat")
	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view = updated.View()
	if !strings.Contains(view, "deepseek/deepseek-chat") {
		t.Fatalf("switch:\n%s", view)
	}
}
