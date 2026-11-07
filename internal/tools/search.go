package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kocoro/edu-cli/internal/agent"
)

type GlobTool struct{ Root string }
type GrepTool struct{ Root string }
type SystemInfoTool struct{}

func (t *GlobTool) Info() agent.ToolInfo {
	return agent.ToolInfo{
		Name:        "glob",
		Description: "Find files under the workspace whose relative path matches a pattern. * matches one path segment.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{"type": "string"},
			},
		},
		Required: []string{"pattern"},
	}
}

func (t *GlobTool) RequiresApproval() bool { return false }

func (t *GlobTool) Run(_ context.Context, argsJSON string) (agent.ToolResult, error) {
	var args struct {
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return agent.ToolResult{Content: err.Error(), IsError: true}, nil
	}
	var hits []string
	_ = filepath.WalkDir(t.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(t.Root, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		ok, _ := filepath.Match(args.Pattern, rel)
		if !ok {
			ok, _ = filepath.Match(args.Pattern, filepath.Base(rel))
		}
		if ok || (strings.Contains(args.Pattern, "**") && strings.HasSuffix(rel, strings.TrimPrefix(args.Pattern, "**/"))) {
			hits = append(hits, rel)
		}
		if len(hits) >= 200 {
			return fs.SkipAll
		}
		return nil
	})
	return agent.ToolResult{Content: strings.Join(hits, "\n")}, nil
}

func (t *GrepTool) Info() agent.ToolInfo {
	return agent.ToolInfo{
		Name:        "grep",
		Description: "Search file contents under the workspace for a substring.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{"type": "string"},
				"path":    map[string]any{"type": "string", "description": "Optional file or directory"},
			},
		},
		Required: []string{"pattern"},
	}
}

func (t *GrepTool) RequiresApproval() bool { return false }

func (t *GrepTool) Run(_ context.Context, argsJSON string) (agent.ToolResult, error) {
	var args struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return agent.ToolResult{Content: err.Error(), IsError: true}, nil
	}
	root := t.Root
	if args.Path != "" {
		resolved, err := resolvePath(t.Root, args.Path)
		if err != nil {
			return agent.ToolResult{Content: err.Error(), IsError: true}, nil
		}
		root = resolved
	}
	var sb strings.Builder
	n := 0
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || n >= 80 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil || strings.Contains(string(data[:min(len(data), 64)]), "\x00") {
			return nil
		}
		for i, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, args.Pattern) {
				rel, _ := filepath.Rel(t.Root, path)
				fmt.Fprintf(&sb, "%s:%d:%s\n", filepath.ToSlash(rel), i+1, line)
				n++
				if n >= 80 {
					break
				}
			}
		}
		return nil
	})
	return agent.ToolResult{Content: sb.String()}, nil
}

func (t *SystemInfoTool) Info() agent.ToolInfo {
	return agent.ToolInfo{
		Name:        "system_info",
		Description: "Report the operating system, architecture, and hostname.",
		Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
	}
}

func (t *SystemInfoTool) RequiresApproval() bool { return false }

func (t *SystemInfoTool) Run(context.Context, string) (agent.ToolResult, error) {
	host, _ := os.Hostname()
	return agent.ToolResult{Content: fmt.Sprintf("OS: %s\nArch: %s\nHostname: %s\n", runtime.GOOS, runtime.GOARCH, host)}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
