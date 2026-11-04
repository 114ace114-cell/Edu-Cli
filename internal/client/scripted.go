package client

import "fmt"

// Scripted returns canned assistant messages so tests can run the daemon
// and session store without a network call.
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
