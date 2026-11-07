package agent

import (
	"context"

	"github.com/kocoro/edu-cli/internal/client"
)

// ToolInfo is the schema sent to the model.
type ToolInfo struct {
	Name        string
	Description string
	Parameters  map[string]any
	Required    []string
}

// ToolResult is the text returned to the model after a tool runs.
type ToolResult struct {
	Content string
	IsError bool
}

// Tool is one local or MCP-backed function.
type Tool interface {
	Info() ToolInfo
	Run(ctx context.Context, args string) (ToolResult, error)
	RequiresApproval() bool
}

// ToolRegistry keeps registration order. Local tools are registered first so
// they win when an MCP tool reuses the same name.
type ToolRegistry struct {
	tools map[string]Tool
	order []string
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: map[string]Tool{}}
}

func (r *ToolRegistry) Register(t Tool) {
	if r == nil || t == nil {
		return
	}
	name := t.Info().Name
	if _, exists := r.tools[name]; !exists {
		r.order = append(r.order, name)
	}
	r.tools[name] = t
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	if r == nil {
		return nil, false
	}
	t, ok := r.tools[name]
	return t, ok
}

func (r *ToolRegistry) Names() []string {
	if r == nil {
		return nil
	}
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

func (r *ToolRegistry) Defs() []client.ToolDef {
	if r == nil {
		return nil
	}
	out := make([]client.ToolDef, 0, len(r.order))
	for _, name := range r.order {
		info := r.tools[name].Info()
		params := info.Parameters
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		if len(info.Required) > 0 {
			params["required"] = info.Required
		}
		out = append(out, client.ToolDef{
			Name:        info.Name,
			Description: info.Description,
			Parameters:  params,
		})
	}
	return out
}
