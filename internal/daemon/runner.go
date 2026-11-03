package daemon

import (
	"github.com/kocoro/skeleton/internal/agent"
	"github.com/kocoro/skeleton/internal/audit"
	"github.com/kocoro/skeleton/internal/client"
	"github.com/kocoro/skeleton/internal/config"
	"github.com/kocoro/skeleton/internal/hooks"
	"github.com/kocoro/skeleton/internal/prompt"
	"github.com/kocoro/skeleton/internal/session"
	"github.com/kocoro/skeleton/internal/skills"
	"github.com/kocoro/skeleton/internal/tools"
)

// Request is the body of POST /message.
// The full type is daemon.RunAgentRequest.
type Request struct {
	Text      string `json:"text"`
	SessionID string `json:"session_id,omitempty"`
}

// Result is the JSON reply. The full type is daemon.RunAgentResult.
type Result struct {
	Reply     string `json:"reply"`
	SessionID string `json:"session_id"`
	RouteKey  string `json:"route_key"`
}

// Runner executes one request on a fresh loop and persists the transcript.
// Stand-in for daemon.RunAgent in internal/daemon/runner.go.
type Runner struct {
	Cfg      config.Config
	LLM      client.LLMClient
	Sessions *session.Store
	Skills   []skills.Skill
	Hooks    *hooks.Runner
	Audit    *audit.Logger
}

func (r *Runner) RunAgent(req Request, ev agent.Events) (Result, error) {
	route := RouteKey(req.SessionID)
	sess, err := r.Sessions.LoadOrCreate(req.SessionID, route)
	if err != nil {
		return Result{}, err
	}
	loop := &agent.Loop{
		LLM:       r.LLM,
		Model:     r.Cfg.Model,
		Tools:     tools.Register(r.Cfg.Workspace, r.Skills),
		System:    prompt.Build(r.Cfg.Workspace, r.Skills),
		Workspace: r.Cfg.Workspace,
		Denied:    r.Cfg.DeniedCommands,
		Hooks:     r.Hooks,
		Audit:     r.Audit,
		MaxIter:   r.Cfg.MaxIterations,
	}
	reply, err := loop.Run(req.Text, sess.Messages, ev)
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
