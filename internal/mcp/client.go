package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	mcpclient "github.com/mark3labs/mcp-go/client"
	mcpproto "github.com/mark3labs/mcp-go/mcp"
)

// ServerConfig describes how to connect to an MCP server.
type ServerConfig struct {
	Command  string            `json:"command"`
	Args     []string          `json:"args"`
	Env      map[string]string `json:"env"`
	Type     string            `json:"type"`
	URL      string            `json:"url"`
	Disabled bool              `json:"disabled"`
	Context  string            `json:"context"`
}

// RemoteTool is one tool discovered from an MCP server.
type RemoteTool struct {
	ServerName string
	Tool       mcpproto.Tool
}

// ClientManager keeps connections to MCP servers for the life of the process.
type ClientManager struct {
	mu      sync.Mutex
	clients map[string]mcpclient.MCPClient
}

func NewClientManager() *ClientManager {
	return &ClientManager{clients: map[string]mcpclient.MCPClient{}}
}

// ConnectAll connects every enabled server and returns the tools it listed.
// A failed server is skipped. An error is returned only when every server fails.
func (m *ClientManager) ConnectAll(ctx context.Context, servers map[string]ServerConfig) ([]RemoteTool, error) {
	var all []RemoteTool
	var errs []string
	for name, cfg := range servers {
		if cfg.Disabled {
			continue
		}
		tools, err := m.connect(ctx, name, cfg)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		all = append(all, tools...)
	}
	if len(errs) > 0 && len(all) == 0 {
		return nil, fmt.Errorf("all MCP servers failed: %s", stringsJoin(errs))
	}
	return all, nil
}

func (m *ClientManager) connect(ctx context.Context, name string, cfg ServerConfig) ([]RemoteTool, error) {
	var c mcpclient.MCPClient
	var err error
	switch cfg.Type {
	case "http":
		if cfg.URL == "" {
			return nil, fmt.Errorf("http MCP server requires url")
		}
		c, err = mcpclient.NewStreamableHttpClient(cfg.URL)
		if err != nil {
			return nil, err
		}
		if starter, ok := c.(interface{ Start(context.Context) error }); ok {
			if err := starter.Start(ctx); err != nil {
				return nil, err
			}
		}
	default:
		if cfg.Command == "" {
			return nil, fmt.Errorf("stdio MCP server requires command")
		}
		c, err = mcpclient.NewStdioMCPClient(cfg.Command, envSlice(cfg.Env), cfg.Args...)
		if err != nil {
			return nil, fmt.Errorf("failed to start %q: %w", cfg.Command, err)
		}
	}

	_, err = c.Initialize(ctx, mcpproto.InitializeRequest{
		Params: mcpproto.InitializeParams{
			ProtocolVersion: mcpproto.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcpproto.Implementation{Name: "edu-cli", Version: "dev"},
		},
	})
	if err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("initialize failed: %w", err)
	}
	listed, err := c.ListTools(ctx, mcpproto.ListToolsRequest{})
	if err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("tools/list failed: %w", err)
	}
	m.mu.Lock()
	m.clients[name] = c
	m.mu.Unlock()

	out := make([]RemoteTool, 0, len(listed.Tools))
	for _, tool := range listed.Tools {
		out = append(out, RemoteTool{ServerName: name, Tool: tool})
	}
	return out, nil
}

// CallTool runs one tool on a connected server.
func (m *ClientManager) CallTool(ctx context.Context, serverName, toolName string, args map[string]any) (string, bool, error) {
	m.mu.Lock()
	c, ok := m.clients[serverName]
	m.mu.Unlock()
	if !ok {
		return "", true, fmt.Errorf("MCP server %q not connected", serverName)
	}
	result, err := c.CallTool(ctx, mcpproto.CallToolRequest{
		Params: mcpproto.CallToolParams{Name: toolName, Arguments: args},
	})
	if err != nil {
		return "", true, err
	}
	var texts []string
	for _, block := range result.Content {
		if text, ok := block.(mcpproto.TextContent); ok {
			texts = append(texts, text.Text)
			continue
		}
		raw, _ := json.Marshal(block)
		texts = append(texts, string(raw))
	}
	return stringsJoin(texts), result.IsError, nil
}

func (m *ClientManager) Close() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, c := range m.clients {
		_ = c.Close()
		delete(m.clients, name)
	}
}

// BuildContext joins the context hint from each enabled server.
func BuildContext(servers map[string]ServerConfig) string {
	var parts []string
	for name, cfg := range servers {
		if cfg.Disabled || cfg.Context == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("[%s] %s", name, cfg.Context))
	}
	return stringsJoin(parts)
}

func envSlice(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}

func stringsJoin(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "\n"
		}
		out += p
	}
	return out
}
