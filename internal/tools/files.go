package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kocoro/edu-cli/internal/agent"
)

type FileReadTool struct{ Root string }
type FileWriteTool struct{ Root string }
type FileEditTool struct{ Root string }
type DirectoryListTool struct{ Root string }

func resolvePath(root, p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("path is required")
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	return abs, nil
}

func (t *FileReadTool) Info() agent.ToolInfo {
	return agent.ToolInfo{
		Name:        "file_read",
		Description: "Read a file with line numbers. offset is the 0-based start line. limit is the max lines.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":   map[string]any{"type": "string", "description": "File path, absolute or relative to the workspace"},
				"offset": map[string]any{"type": "integer"},
				"limit":  map[string]any{"type": "integer"},
			},
		},
		Required: []string{"path"},
	}
}

func (t *FileReadTool) RequiresApproval() bool { return false }

func (t *FileReadTool) Run(_ context.Context, argsJSON string) (agent.ToolResult, error) {
	var args struct {
		Path   string `json:"path"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return agent.ToolResult{Content: "invalid arguments: " + err.Error(), IsError: true}, nil
	}
	path, err := resolvePath(t.Root, args.Path)
	if err != nil {
		return agent.ToolResult{Content: err.Error(), IsError: true}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return agent.ToolResult{Content: "error reading file: " + err.Error(), IsError: true}, nil
	}
	lines := strings.Split(string(data), "\n")
	start := args.Offset
	if start < 0 {
		start = 0
	}
	if start > len(lines) {
		start = len(lines)
	}
	end := len(lines)
	if args.Limit > 0 && start+args.Limit < end {
		end = start + args.Limit
	}
	var sb strings.Builder
	for i := start; i < end; i++ {
		fmt.Fprintf(&sb, "%4d | %s\n", i+1, lines[i])
	}
	return agent.ToolResult{Content: sb.String()}, nil
}

func (t *FileWriteTool) Info() agent.ToolInfo {
	return agent.ToolInfo{
		Name:        "file_write",
		Description: "Create or overwrite a file with the full contents. Parent directories are created.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":    map[string]any{"type": "string"},
				"content": map[string]any{"type": "string"},
			},
		},
		Required: []string{"path", "content"},
	}
}

func (t *FileWriteTool) RequiresApproval() bool { return true }

func (t *FileWriteTool) Run(_ context.Context, argsJSON string) (agent.ToolResult, error) {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return agent.ToolResult{Content: "invalid arguments: " + err.Error(), IsError: true}, nil
	}
	path, err := resolvePath(t.Root, args.Path)
	if err != nil {
		return agent.ToolResult{Content: err.Error(), IsError: true}, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return agent.ToolResult{Content: err.Error(), IsError: true}, nil
	}
	if err := os.WriteFile(path, []byte(args.Content), 0o644); err != nil {
		return agent.ToolResult{Content: err.Error(), IsError: true}, nil
	}
	return agent.ToolResult{Content: "wrote " + path}, nil
}

func (t *FileEditTool) Info() agent.ToolInfo {
	return agent.ToolInfo{
		Name:        "file_edit",
		Description: "Replace one exact occurrence of old_string. It must appear once in the file.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":       map[string]any{"type": "string"},
				"old_string": map[string]any{"type": "string"},
				"new_string": map[string]any{"type": "string"},
			},
		},
		Required: []string{"path", "old_string", "new_string"},
	}
}

func (t *FileEditTool) RequiresApproval() bool { return true }

func (t *FileEditTool) Run(_ context.Context, argsJSON string) (agent.ToolResult, error) {
	var args struct {
		Path      string `json:"path"`
		OldString string `json:"old_string"`
		NewString string `json:"new_string"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return agent.ToolResult{Content: "invalid arguments: " + err.Error(), IsError: true}, nil
	}
	path, err := resolvePath(t.Root, args.Path)
	if err != nil {
		return agent.ToolResult{Content: err.Error(), IsError: true}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return agent.ToolResult{Content: err.Error(), IsError: true}, nil
	}
	text := string(data)
	n := strings.Count(text, args.OldString)
	if n == 0 {
		return agent.ToolResult{Content: "old_string not found", IsError: true}, nil
	}
	if n > 1 {
		return agent.ToolResult{Content: "old_string is not unique", IsError: true}, nil
	}
	text = strings.Replace(text, args.OldString, args.NewString, 1)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return agent.ToolResult{Content: err.Error(), IsError: true}, nil
	}
	return agent.ToolResult{Content: "edited " + path}, nil
}

func (t *DirectoryListTool) Info() agent.ToolInfo {
	return agent.ToolInfo{
		Name:        "directory_list",
		Description: "List files and directories in a path.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string", "description": "Directory path. Defaults to the workspace."},
			},
		},
	}
}

func (t *DirectoryListTool) RequiresApproval() bool { return false }

func (t *DirectoryListTool) Run(_ context.Context, argsJSON string) (agent.ToolResult, error) {
	var args struct {
		Path string `json:"path"`
	}
	if strings.TrimSpace(argsJSON) != "" {
		_ = json.Unmarshal([]byte(argsJSON), &args)
	}
	if args.Path == "" {
		args.Path = t.Root
	}
	path, err := resolvePath(t.Root, args.Path)
	if err != nil {
		return agent.ToolResult{Content: err.Error(), IsError: true}, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return agent.ToolResult{Content: err.Error(), IsError: true}, nil
	}
	var sb strings.Builder
	for _, e := range entries {
		kind := "file"
		if e.IsDir() {
			kind = "dir"
		}
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		fmt.Fprintf(&sb, "%s\t%s\t%d\n", kind, e.Name(), size)
	}
	return agent.ToolResult{Content: sb.String()}, nil
}
