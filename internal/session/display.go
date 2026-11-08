package session

import (
	"strings"

	"github.com/kocoro/edu-cli/internal/client"
)

// MessagesForDisplay returns user lines and final assistant replies for the TUI.
func MessagesForDisplay(msgs []client.Message) []client.Message {
	var out []client.Message
	for _, m := range msgs {
		switch m.Role {
		case "user":
			if strings.TrimSpace(m.Content) != "" {
				out = append(out, m)
			}
		case "assistant":
			if len(m.ToolCalls) == 0 && strings.TrimSpace(m.Content) != "" {
				out = append(out, m)
			}
		case "system", "tool":
			continue
		default:
			if strings.TrimSpace(m.Content) != "" {
				out = append(out, m)
			}
		}
	}
	return out
}
