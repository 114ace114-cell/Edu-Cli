package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAI is a chat-completions client. Stand-in for internal/client/gateway.go.
type OpenAI struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

func (o *OpenAI) Complete(req Request) (Message, error) {
	base := strings.TrimRight(o.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	model := req.Model
	if model == "" {
		model = o.Model
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	payload := map[string]any{
		"model":    model,
		"messages": wireMessages(req.Messages),
	}
	if len(req.Tools) > 0 {
		payload["tools"] = wireTools(req.Tools)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Message{}, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Message{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if o.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.APIKey)
	}
	hc := o.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 90 * time.Second}
	}
	resp, err := hc.Do(httpReq)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return Message{}, err
	}
	if resp.StatusCode != http.StatusOK {
		text := string(raw)
		if len(text) > 400 {
			text = text[:400]
		}
		return Message{}, fmt.Errorf("llm %s: %s", resp.Status, text)
	}
	return parseCompletion(raw)
}

func parseCompletion(raw []byte) (Message, error) {
	var parsed struct {
		Choices []struct {
			Message struct {
				Content   *string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Message{}, err
	}
	if len(parsed.Choices) == 0 {
		return Message{}, fmt.Errorf("empty llm response")
	}
	in := parsed.Choices[0].Message
	out := Message{Role: "assistant"}
	if in.Content != nil {
		out.Content = *in.Content
	}
	for _, call := range in.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:        call.ID,
			Name:      call.Function.Name,
			Arguments: call.Function.Arguments,
		})
	}
	return out, nil
}

func wireMessages(messages []Message) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		item := map[string]any{"role": m.Role}
		if m.Role == "tool" {
			item["tool_call_id"] = m.ToolCallID
			item["content"] = m.Content
			out = append(out, item)
			continue
		}
		if m.Content != "" || len(m.ToolCalls) == 0 {
			item["content"] = m.Content
		}
		if len(m.ToolCalls) > 0 {
			calls := make([]map[string]any, 0, len(m.ToolCalls))
			for _, c := range m.ToolCalls {
				calls = append(calls, map[string]any{
					"id":   c.ID,
					"type": "function",
					"function": map[string]any{
						"name":      c.Name,
						"arguments": c.Arguments,
					},
				})
			}
			item["tool_calls"] = calls
		}
		out = append(out, item)
	}
	return out
}

func wireTools(tools []ToolDef) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		out = append(out, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			},
		})
	}
	return out
}
