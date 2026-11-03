package permissions

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// DefaultDenied is the hard-block list for this skeleton.
// The full engine is internal/permissions.CheckCommand:
// hard-block, denied_commands, compound split, always-ask, allowlist.
var DefaultDenied = []string{"rm", "del", "format", "shutdown", "mkfs", "dd"}

// Admit decides whether a tool call may run.
// The agent loop calls this before Tool.Run, matching checkPermissionAndApproval.
func Admit(workspace, toolName, argsJSON string, denied []string) error {
	if len(denied) == 0 {
		denied = DefaultDenied
	}
	switch toolName {
	case "bash":
		var args struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || strings.TrimSpace(args.Command) == "" {
			return fmt.Errorf("bash: command is required")
		}
		return checkCommand(args.Command, denied)
	case "file_read", "file_write", "file_edit", "grep":
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Errorf("%s: invalid arguments", toolName)
		}
		if toolName != "grep" && strings.TrimSpace(args.Path) == "" {
			return fmt.Errorf("%s: path is required", toolName)
		}
		if args.Path == "" {
			return nil
		}
		return checkPath(workspace, args.Path)
	case "use_skill":
		return nil
	default:
		return fmt.Errorf("unknown tool %s", toolName)
	}
}

func checkCommand(command string, denied []string) error {
	for _, tok := range strings.Fields(command) {
		base := strings.ToLower(strings.Trim(tok, `"'`))
		base = strings.TrimSuffix(base, ".exe")
		for _, d := range denied {
			if base == strings.ToLower(d) {
				return fmt.Errorf("command %q is denied", d)
			}
		}
	}
	return nil
}

func checkPath(workspace, p string) error {
	abs := p
	if !filepath.IsAbs(p) {
		abs = filepath.Join(workspace, p)
	}
	abs = filepath.Clean(abs)
	root := filepath.Clean(workspace)
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path outside workspace: %s", p)
	}
	return nil
}
