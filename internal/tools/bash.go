package tools

import (
	"context"
	"encoding/json"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/kocoro/edu-cli/internal/agent"
)

type BashTool struct{ Root string }

func (t *BashTool) Info() agent.ToolInfo {
	shell := "sh -c"
	if runtime.GOOS == "windows" {
		shell = "powershell -NoProfile -Command"
	}
	return agent.ToolInfo{
		Name:        "bash",
		Description: "Run a shell command in the workspace. On Windows the command is PowerShell. Shell: " + shell,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{"type": "string"},
				"timeout": map[string]any{"type": "integer", "description": "Seconds, default 120"},
			},
		},
		Required: []string{"command"},
	}
}

func (t *BashTool) RequiresApproval() bool { return true }

func (t *BashTool) Run(ctx context.Context, argsJSON string) (agent.ToolResult, error) {
	var args struct {
		Command string `json:"command"`
		Timeout int    `json:"timeout"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return agent.ToolResult{Content: "invalid arguments: " + err.Error(), IsError: true}, nil
	}
	command := strings.TrimSpace(args.Command)
	if command == "" {
		return agent.ToolResult{Content: "command is required", IsError: true}, nil
	}
	lower := strings.ToLower(command)
	for _, denied := range []string{"format ", "shutdown", "mkfs", "del /s", "rm -rf /", "rm -rf \\"} {
		if strings.Contains(lower, denied) {
			return agent.ToolResult{Content: "command denied", IsError: true}, nil
		}
	}
	timeout := 120 * time.Second
	if args.Timeout > 0 {
		timeout = time.Duration(args.Timeout) * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}
	if t.Root != "" {
		cmd.Dir = t.Root
	}
	out, err := cmd.CombinedOutput()
	text := string(out)
	if len(text) > 30000 {
		text = text[:30000] + "\n... (truncated)"
	}
	if err != nil {
		return agent.ToolResult{Content: err.Error() + "\n" + text, IsError: true}, nil
	}
	return agent.ToolResult{Content: text}, nil
}
