package agent

import (
	"github.com/kocoro/skeleton/internal/client"
)

// Tool is the local tool port. The full interface is agent.Tool in internal/agent/tools.go.
type Tool interface {
	Info() Info
	Run(args string) string
}

// Info is the model-facing schema plus the required-field names.
type Info struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// Registry is the tool map. Production ordering is local > MCP > gateway.
// This skeleton registers local tools only.
type Registry struct {
	order []string
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}}
}

func (r *Registry) Add(t Tool) {
	name := t.Info().Name
	if _, ok := r.tools[name]; !ok {
		r.order = append(r.order, name)
	}
	r.tools[name] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) Defs() []client.ToolDef {
	out := make([]client.ToolDef, 0, len(r.order))
	for _, name := range r.order {
		info := r.tools[name].Info()
		out = append(out, client.ToolDef{
			Name:        info.Name,
			Description: info.Description,
			Parameters:  info.Parameters,
		})
	}
	return out
}
