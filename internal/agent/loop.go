package agent

import (
	"fmt"
	"strings"

	"github.com/kocoro/skeleton/internal/audit"
	"github.com/kocoro/skeleton/internal/client"
	"github.com/kocoro/skeleton/internal/hooks"
	"github.com/kocoro/skeleton/internal/permissions"
)

// Events receives tool lifecycle signals. The daemon SSE handler implements this.
// The full loop uses agent.EventHandler (OnStreamDelta, tool status, approvals).
type Events interface {
	OnToolStatus(name, phase string)
}

// Loop is one agent run. Stand-in for AgentLoop.run in internal/agent/loop.go.
type Loop struct {
	LLM       client.LLMClient
	Model     string
	Tools     *Registry
	System    string
	Workspace string
	Denied    []string
	Hooks     *hooks.Runner
	Audit     *audit.Logger
	MaxIter   int
	Messages  []client.Message
}

// Run appends userText to history and calls the model until it stops asking for tools.
func (l *Loop) Run(userText string, history []client.Message, ev Events) (string, error) {
	if l.MaxIter <= 0 {
		l.MaxIter = 8
	}
	if len(history) == 0 || history[0].Role != "system" {
		l.Messages = []client.Message{{Role: "system", Content: l.System}}
		l.Messages = append(l.Messages, history...)
	} else {
		l.Messages = append([]client.Message{}, history...)
	}
	l.Messages = append(l.Messages, client.Message{Role: "user", Content: userText})

	for step := 1; step <= l.MaxIter; step++ {
		assistant, err := l.LLM.Complete(client.Request{
			Model:    l.Model,
			Messages: l.Messages,
			Tools:    l.Tools.Defs(),
		})
		if err != nil {
			return "", err
		}
		l.Messages = append(l.Messages, assistant)
		if len(assistant.ToolCalls) == 0 {
			return strings.TrimSpace(assistant.Content), nil
		}
		for _, call := range assistant.ToolCalls {
			if ev != nil {
				ev.OnToolStatus(call.Name, "running")
			}
			l.Hooks.BeforeTool(call.Name)
			out := l.exec(call)
			l.Hooks.AfterTool(call.Name)
			ok := !strings.HasPrefix(out, "error:")
			if l.Audit != nil {
				_ = l.Audit.Record(call.Name, out, ok)
			}
			if ev != nil {
				ev.OnToolStatus(call.Name, "completed")
			}
			l.Messages = append(l.Messages, client.Message{
				Role:       "tool",
				ToolCallID: call.ID,
				Content:    out,
			})
		}
	}
	return "", fmt.Errorf("stopped after %d steps", l.MaxIter)
}

func (l *Loop) exec(call client.ToolCall) string {
	if err := permissions.Admit(l.Workspace, call.Name, call.Arguments, l.Denied); err != nil {
		return "error: " + err.Error()
	}
	tool, ok := l.Tools.Get(call.Name)
	if !ok {
		return "error: unknown tool " + call.Name
	}
	return tool.Run(call.Arguments)
}
