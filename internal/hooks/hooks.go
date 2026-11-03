package hooks

// Runner is the before/after tool seam. The full type is internal/hooks.HookRunner.
type Runner struct {
	BeforeN int
	AfterN  int
}

func (r *Runner) BeforeTool(string) {
	if r == nil {
		return
	}
	r.BeforeN++
}

func (r *Runner) AfterTool(string) {
	if r == nil {
		return
	}
	r.AfterN++
}
