package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kocoro/edu-cli/internal/agent"
	"github.com/kocoro/edu-cli/internal/mcp"
	mcpproto "github.com/mark3labs/mcp-go/mcp"
)

// MCPTool wraps one remote MCP tool. Local tools with the same name stay in front.
type MCPTool struct {
	serverName string
	tool       mcpproto.Tool
	manager    *mcp.ClientManager
}

func NewMCPTool(serverName string, tool mcpproto.Tool, manager *mcp.ClientManager) *MCPTool {
	return &MCPTool{serverName: serverName, tool: tool, manager: manager}
}

func (t *MCPTool) Info() agent.ToolInfo {
	name := strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, t.tool.Name)
	desc := t.tool.Description
	if desc == "" {
		desc = "MCP tool from " + t.serverName
	}
	if len(desc) > 500 {
		desc = desc[:500] + "..."
	}
	params := map[string]any{"type": "object", "properties": map[string]any{}}
	if t.tool.InputSchema.Properties != nil {
		params["properties"] = t.tool.InputSchema.Properties
	}
	return agent.ToolInfo{
		Name:        name,
		Description: fmt.Sprintf("[%s] %s", t.serverName, desc),
		Parameters:  params,
		Required:    append([]string{}, t.tool.InputSchema.Required...),
	}
}

func (t *MCPTool) RequiresApproval() bool { return true }

func (t *MCPTool) Run(ctx context.Context, argsJSON string) (agent.ToolResult, error) {
	var args map[string]any
	if strings.TrimSpace(argsJSON) != "" && argsJSON != "{}" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return agent.ToolResult{Content: "invalid arguments: " + err.Error(), IsError: true}, nil
		}
	}
	if args == nil {
		args = map[string]any{}
	}
	content, isError, err := t.manager.CallTool(ctx, t.serverName, t.tool.Name, args)
	if err != nil {
		return agent.ToolResult{Content: err.Error(), IsError: true}, nil
	}
	return agent.ToolResult{Content: content, IsError: isError}, nil
}
