package agent

import (
	"strings"

	"github.com/kocoro/skeleton/internal/client"
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
	l.Messages = append([]client.Message{}, history...)
	l.Messages = append(l.Messages, client.Message{Role: "user", Content: userText})
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
