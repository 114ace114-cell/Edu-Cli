package client

import "fmt"

// Scripted returns canned assistant messages. Tests and `shan demo` use it
// so the loop, tools, daemon, and session store run without a network call.
type Scripted struct {
	Replies []Message
	n       int
}

func (s *Scripted) Complete(Request) (Message, error) {
	if s.n >= len(s.Replies) {
		return Message{}, fmt.Errorf("scripted model ran out of replies at step %d", s.n+1)
	}
	msg := s.Replies[s.n]
	s.n++
	if msg.Role == "" {
		msg.Role = "assistant"
	}
	return msg, nil
}

// DemoScript is the offline coding task used by `shan demo`.
func DemoScript() *Scripted {
	return &Scripted{Replies: []Message{
		{ToolCalls: []ToolCall{{
			ID:        "c1",
			Name:      "file_write",
			Arguments: `{"path":"hello.go","content":"package main\n\nfunc greet() string { return \"hi\" }\n"}`,
		}}},
		{ToolCalls: []ToolCall{{
			ID:        "c2",
			Name:      "file_edit",
			Arguments: `{"path":"hello.go","old_string":"return \"hi\"","new_string":"return \"hello from skeleton\""}`,
		}}},
		{ToolCalls: []ToolCall{{
			ID:        "c3",
			Name:      "file_read",
			Arguments: `{"path":"hello.go"}`,
		}}},
		{ToolCalls: []ToolCall{{
			ID:        "c4",
			Name:      "bash",
			Arguments: `{"command":"echo skeleton-ok> proof.txt"}`,
		}}},
		{Content: "updated greet and wrote proof.txt"},
	}}
}
