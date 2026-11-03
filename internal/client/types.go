package client

// Message is one transcript entry. Tool results use Role "tool".
// Same roles as internal/client messages in the full repo.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall is one function call the model asked for.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolDef is the schema sent on the completion request.
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// Request is one completion. The full repo type is client.CompletionRequest.
type Request struct {
	Model    string
	Messages []Message
	Tools    []ToolDef
}

// LLMClient is the model port. GatewayClient and OllamaClient implement the
// production version of this (Complete / CompleteStream).
type LLMClient interface {
	Complete(req Request) (Message, error)
}
