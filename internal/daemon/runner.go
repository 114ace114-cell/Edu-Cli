package daemon

import (
	"github.com/kocoro/skeleton/internal/agent"
	"github.com/kocoro/skeleton/internal/client"
	"github.com/kocoro/skeleton/internal/config"
	"github.com/kocoro/skeleton/internal/session"
)

// Request is the body of POST /message.
type Request struct {
	Text      string `json:"text"`
	SessionID string `json:"session_id,omitempty"`
}

// Result is the JSON reply.
type Result struct {
	Reply     string `json:"reply"`
	SessionID string `json:"session_id"`
	RouteKey  string `json:"route_key"`
}

// Runner executes one request and persists the transcript.
type Runner struct {
	Cfg      config.Config
	LLM      client.LLMClient
	Sessions *session.Store
}

func (r *Runner) RunAgent(req Request) (Result, error) {
	route := RouteKey(req.SessionID)
	sess, err := r.Sessions.LoadOrCreate(req.SessionID, route)
	if err != nil {
		return Result{}, err
	}
	loop := &agent.Loop{
		LLM:   r.LLM,
		Model: r.Cfg.Model,
	}
	reply, err := loop.Run(req.Text, sess.Messages)
	sess.Messages = loop.Messages
	sess.RouteKey = route
	if saveErr := r.Sessions.Save(sess); saveErr != nil && err == nil {
		err = saveErr
	}
	if err != nil {
		return Result{SessionID: sess.ID, RouteKey: route}, err
	}
	return Result{Reply: reply, SessionID: sess.ID, RouteKey: route}, nil
}
