package prompt

import "strings"

const (
	maxMemoryChars       = 2000
	maxInstructionsChars = 16000
	maxContextChars      = 800
)

// BasePrompt is the fixed identity line. Tool rules stay out of this text
// until the process actually registers tools.
const BasePrompt = `You are Edu CLI, a coding assistant in the user's terminal.
Be concise. Do not claim to be Claude.
Do not invent file contents or claim a command ran.


Tool selection rules:
- For LOCAL computer control (opening apps, window management, UI automation): use applescript
- For LOCAL notifications: use notify
- For LOCAL clipboard: use clipboard
- For LOCAL mouse/keyboard control: use computer
- For LOCAL screenshots (screen capture): use screenshot
- For LOCAL browser automation (isolated Chrome): use browser
- For file operations: use file_read, file_write, file_edit, glob, grep, directory_list
- For shell commands, tests, builds: use bash
- For web search and research: use web_search, web_fetch (server-side)
- For server-side browser (remote pages): use browser_navigate, browser_click, etc.
When reading files, always use file_read before editing.`

// PromptOptions configures the system prompt assembly.
type PromptOptions struct {
	BasePrompt   string
	Memory       string
	Instructions string
	ToolNames    []string
	ServerTools  []string
	MCPContext   string
	CWD          string
	SessionInfo  string
}

// BuildSystemPrompt assembles the system prompt in layers.
func BuildSystemPrompt(opts PromptOptions) string {
	var sb strings.Builder

	base := opts.BasePrompt
	if strings.TrimSpace(base) == "" {
		base = BasePrompt
	}
	sb.WriteString(base)

	// memory
	if mem := strings.TrimSpace(opts.Memory); mem != "" {
		sb.WriteString("\n\n## Memory\n")
		sb.WriteString(truncate(mem, maxMemoryChars))
	}

	// instructions
	if inst := strings.TrimSpace(opts.Instructions); inst != "" {
		sb.WriteString("\n\n## Instructions\n")
		sb.WriteString(truncate(inst, maxInstructionsChars))
	}

	// tools
	if len(opts.ToolNames) > 0 || len(opts.ServerTools) > 0 {
		sb.WriteString("\n\n## Available Tools\n")
		if len(opts.ToolNames) > 0 {
			sb.WriteString("You have these local tools: ")
			sb.WriteString(strings.Join(opts.ToolNames, ", "))
			sb.WriteString(".")
		}
		if len(opts.ServerTools) > 0 {
			if len(opts.ToolNames) > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString("You also have server-side tools: ")
			sb.WriteString(strings.Join(opts.ServerTools, ", "))
			sb.WriteString(".")
		}
	}

	// mcp context
	if mcp := strings.TrimSpace(opts.MCPContext); mcp != "" {
		sb.WriteString("\n\n## MCP Server Context\n")
		sb.WriteString(mcp)
	}

	// context
	if contextParts := buildContext(opts.CWD, opts.SessionInfo); contextParts != "" {
		sb.WriteString("\n\n## Context\n")
		sb.WriteString(truncate(contextParts, maxContextChars))
	}
	return sb.String()
}

func buildContext(cwd, sessionInfo string) string {
	var parts []string
	if cwd != "" {
		parts = append(parts, "Working directory: "+cwd)
	}
	if sessionInfo != "" {
		parts = append(parts, sessionInfo)
	}
	return strings.Join(parts, "\n")
}

func truncate(s string, maxChars int) string {
	if len(s) <= maxChars {
		return s
	}
	return s[:maxChars] + "\n[truncated]"
}
