package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kocoro/edu-cli/internal/client"
	"github.com/kocoro/edu-cli/internal/debuglog"
	"github.com/kocoro/edu-cli/internal/instructions"
	"github.com/kocoro/edu-cli/internal/prompt"
)

// Loop is one chat turn. It appends the user message, calls the model once,
// and keeps the transcript for the session store.
type Loop struct {
	LLM        client.LLMClient
	Model      string
	Tools      *ToolRegistry
	MCPContext string
	Workspace  string
	MaxIter    int
	Messages   []client.Message
}

// Run appends userText to history and returns the assistant reply.
func (l *Loop) Run(userText string, history []client.Message) (string, error) {
	memory, err := instructions.LoadMemory(200)
	if err != nil {
		debuglog.Printf("memory: %v", err)
	}
	eduDir := os.Getenv("EDU_DATA")
	if eduDir == "" {
		eduDir = "data"
	}
	loaded, err := instructions.LoadInstructions(eduDir, ".edu", 4000)
	if err != nil {
		debuglog.Printf("instructions: %v", err)
	}
	cwd := l.Workspace
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	systemPrompt := prompt.BuildSystemPrompt(prompt.PromptOptions{
		Memory:       memory,
		Instructions: loaded,
		ToolNames:    l.Tools.Names(),
		MCPContext:   l.MCPContext,
		CWD:          cwd,
	})

	l.Messages = append([]client.Message{}, history...)
	if len(l.Messages) == 0 || l.Messages[0].Role != "system" {
		l.Messages = append([]client.Message{{Role: "system", Content: systemPrompt}}, l.Messages...)
	} else {
		l.Messages[0].Content = systemPrompt
	}
	l.Messages = append(l.Messages, client.Message{Role: "user", Content: userText})
	debuglog.Printf("Messages %v", l.Messages)
	if l.MaxIter <= 0 {
		l.MaxIter = 12
	}
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
			debuglog.Printf("tool %s %s", call.Name, call.Arguments)
			l.Messages = append(l.Messages, client.Message{
				Role:       "tool",
				ToolCallID: call.ID,
				Content:    l.exec(call),
			})
		}
	}
	return "", fmt.Errorf("stopped after %d tool steps", l.MaxIter)
}

func (l *Loop) exec(call client.ToolCall) string {
	if l.Tools == nil {
		return "error: no tools registered"
	}
	tool, ok := l.Tools.Get(call.Name)
	if !ok {
		return "error: unknown tool " + call.Name
	}
	result, err := tool.Run(context.Background(), call.Arguments)
	if err != nil {
		return "error: " + err.Error()
	}
	if result.IsError {
		return "error: " + result.Content
	}
	return result.Content
}
