package agent

import (
	"os"
	"strings"

	"github.com/kocoro/skeleton/internal/client"
	"github.com/kocoro/skeleton/internal/debuglog"
	"github.com/kocoro/skeleton/internal/instructions"
	"github.com/kocoro/skeleton/internal/prompt"
)

// Loop is one chat turn. It appends the user message, calls the model once,
// and keeps the transcript for the session store.
type Loop struct {
	LLM      client.LLMClient
	Model    string
	Messages []client.Message
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
	cwd, _ := os.Getwd()
	systemPrompt := prompt.BuildSystemPrompt(prompt.PromptOptions{
		Memory:       memory,
		Instructions: loaded,
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
	assistant, err := l.LLM.Complete(client.Request{
		Model:    l.Model,
		Messages: l.Messages,
	})
	if err != nil {
		return "", err
	}
	l.Messages = append(l.Messages, assistant)
	return strings.TrimSpace(assistant.Content), nil
}
