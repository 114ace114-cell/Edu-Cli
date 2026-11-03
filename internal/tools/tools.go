package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kocoro/skeleton/internal/agent"
	"github.com/kocoro/skeleton/internal/skills"
)

// maxGrepFiles bounds a workspace walk.
// Workload: a local repo of a few hundred files.
// Symptom: the result ends with "stopped early" and later matches are missing.
// Override: raise the const in this file.
const maxGrepFiles = 500

// Register builds the local tool set. Stand-in for tools.RegisterLocalTools.
func Register(workspace string, loaded []skills.Skill) *agent.Registry {
	abs, err := filepath.Abs(workspace)
	if err != nil {
		abs = workspace
	}
	reg := agent.NewRegistry()
	reg.Add(&fileTool{root: abs, name: "file_read"})
	reg.Add(&fileTool{root: abs, name: "file_write"})
	reg.Add(&fileTool{root: abs, name: "file_edit"})
	reg.Add(&bashTool{root: abs})
	reg.Add(&grepTool{root: abs})
	reg.Add(&skillTool{skills: loaded})
	return reg
}

type fileTool struct {
	root string
	name string
}

func (t *fileTool) Info() agent.Info {
	props := map[string]any{
		"path": map[string]any{"type": "string", "description": "Path relative to the workspace"},
	}
	required := []string{"path"}
	desc := "Read a UTF-8 text file. Lines are numbered."
	switch t.name {
	case "file_write":
		desc = "Create or overwrite a file with the full contents."
		props["content"] = map[string]any{"type": "string"}
		required = []string{"path", "content"}
	case "file_edit":
		desc = "Replace one exact occurrence of old_string. It must be unique."
		props["old_string"] = map[string]any{"type": "string"}
		props["new_string"] = map[string]any{"type": "string"}
		required = []string{"path", "old_string", "new_string"}
	}
	return agent.Info{Name: t.name, Description: desc, Parameters: objSchema(required, props)}
}

func (t *fileTool) Run(argsJSON string) string {
	switch t.name {
	case "file_read":
		return t.read(argsJSON)
	case "file_write":
		return t.write(argsJSON)
	default:
		return t.edit(argsJSON)
	}
}

func (t *fileTool) read(argsJSON string) string {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || args.Path == "" {
		return "error: file_read: path is required"
	}
	path, err := resolve(t.root, args.Path)
	if err != nil {
		return "error: " + err.Error()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "error: " + err.Error()
	}
	if len(data) == 0 {
		return "(empty file)"
	}
	if len(data) > 32*1024 {
		data = data[:32*1024]
	}
	var b strings.Builder
	for i, row := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		fmt.Fprintf(&b, "%4d|%s\n", i+1, row)
	}
	return b.String()
}

func (t *fileTool) write(argsJSON string) string {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	// A missing string unmarshals as "". Reject that so an empty call cannot wipe a file.
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || args.Path == "" || args.Content == "" {
		return "error: file_write: path and content are required"
	}
	path, err := resolve(t.root, args.Path)
	if err != nil {
		return "error: " + err.Error()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "error: " + err.Error()
	}
	if err := os.WriteFile(path, []byte(args.Content), 0o644); err != nil {
		return "error: " + err.Error()
	}
	return fmt.Sprintf("wrote %s (%d bytes)", args.Path, len(args.Content))
}

func (t *fileTool) edit(argsJSON string) string {
	var args struct {
		Path      string `json:"path"`
		OldString string `json:"old_string"`
		NewString string `json:"new_string"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || args.Path == "" || args.OldString == "" {
		return "error: file_edit: path and old_string are required"
	}
	path, err := resolve(t.root, args.Path)
	if err != nil {
		return "error: " + err.Error()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "error: " + err.Error()
	}
	content := string(data)
	count := strings.Count(content, args.OldString)
	if count == 0 {
		return "error: old_string not found"
	}
	if count != 1 {
		return fmt.Sprintf("error: old_string found %d times; it must be unique", count)
	}
	next := strings.Replace(content, args.OldString, args.NewString, 1)
	if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
		return "error: " + err.Error()
	}
	return "edited " + args.Path
}

type bashTool struct{ root string }

func (t *bashTool) Info() agent.Info {
	return agent.Info{
		Name:        "bash",
		Description: "Run a shell command in the workspace. Returns the combined output and exit code.",
		Parameters: objSchema([]string{"command"}, map[string]any{
			"command": map[string]any{"type": "string"},
		}),
	}
}

func (t *bashTool) Run(argsJSON string) string {
	var args struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || strings.TrimSpace(args.Command) == "" {
		return "error: bash: command is required"
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", args.Command)
	} else {
		cmd = exec.Command("sh", "-c", args.Command)
	}
	cmd.Dir = t.root
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	timer := time.AfterFunc(15*time.Second, func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})
	defer timer.Stop()
	err := cmd.Run()
	exit := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exit = ee.ExitCode()
		} else {
			return "error: " + err.Error()
		}
	}
	out := buf.String()
	if len(out) > 8*1024 {
		out = out[:8*1024] + "\n... truncated"
	}
	return fmt.Sprintf("exit %d\n%s", exit, out)
}

type grepTool struct{ root string }

func (t *grepTool) Info() agent.Info {
	return agent.Info{
		Name:        "grep",
		Description: "Literal substring search under a file or directory in the workspace.",
		Parameters: objSchema([]string{"pattern"}, map[string]any{
			"pattern": map[string]any{"type": "string"},
			"path":    map[string]any{"type": "string"},
		}),
	}
}

func (t *grepTool) Run(argsJSON string) string {
	var args struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || args.Pattern == "" {
		return "error: grep: pattern is required"
	}
	start := t.root
	if args.Path != "" {
		var err error
		start, err = resolve(t.root, args.Path)
		if err != nil {
			return "error: " + err.Error()
		}
	}
	var hits []string
	seen := 0
	stopped := false
	err := filepath.WalkDir(start, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		seen++
		if seen > maxGrepFiles {
			stopped = true
			return fs.SkipAll
		}
		info, err := d.Info()
		if err != nil || info.Size() > 256*1024 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(t.root, path)
		for i, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, args.Pattern) {
				hits = append(hits, fmt.Sprintf("%s:%d:%s", filepath.ToSlash(rel), i+1, line))
				if len(hits) >= 50 {
					stopped = true
					return fs.SkipAll
				}
			}
		}
		return nil
	})
	if err != nil {
		return "error: " + err.Error()
	}
	if len(hits) == 0 {
		return "no matches"
	}
	out := strings.Join(hits, "\n")
	if stopped {
		out += "\nstopped early"
	}
	return out
}

type skillTool struct{ skills []skills.Skill }

func (t *skillTool) Info() agent.Info {
	return agent.Info{
		Name:        "use_skill",
		Description: "Load the body of a skill by name.",
		Parameters: objSchema([]string{"name"}, map[string]any{
			"name": map[string]any{"type": "string"},
		}),
	}
}

func (t *skillTool) Run(argsJSON string) string {
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || args.Name == "" {
		return "error: use_skill: name is required"
	}
	s, ok := skills.Find(t.skills, args.Name)
	if !ok {
		return "error: unknown skill " + args.Name
	}
	return s.Body
}

func objSchema(required []string, props map[string]any) map[string]any {
	return map[string]any{
		"type":       "object",
		"required":   required,
		"properties": props,
	}
}

func resolve(root, p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("empty path")
	}
	abs := p
	if !filepath.IsAbs(p) {
		abs = filepath.Join(root, p)
	}
	abs = filepath.Clean(abs)
	rel, err := filepath.Rel(filepath.Clean(root), abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path outside workspace: %s", p)
	}
	return abs, nil
}
