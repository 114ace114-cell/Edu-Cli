package tools

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kocoro/edu-cli/internal/agent"
	"github.com/kocoro/edu-cli/internal/config"
	"github.com/kocoro/edu-cli/internal/mcp"
)

// RegisterLocalTools registers the local tools. They are registered before MCP
// tools so a remote tool cannot replace a local one.
func RegisterLocalTools(workspace string) *agent.ToolRegistry {
	reg := agent.NewToolRegistry()
	reg.Register(&FileReadTool{Root: workspace})
	reg.Register(&FileWriteTool{Root: workspace})
	reg.Register(&FileEditTool{Root: workspace})
	reg.Register(&GlobTool{Root: workspace})
	reg.Register(&GrepTool{Root: workspace})
	reg.Register(&BashTool{Root: workspace})
	reg.Register(&DirectoryListTool{Root: workspace})
	reg.Register(&SystemInfoTool{})
	return reg
}

// RegisterAll registers local tools, then connects MCP servers and appends
// their tools. The cleanup function closes those servers.
func RegisterAll(cfg config.Config) (*agent.ToolRegistry, func(), error) {
	reg := RegisterLocalTools(cfg.Workspace)
	var mgr *mcp.ClientManager
	if len(cfg.MCPServers) > 0 {
		mgr = mcp.NewClientManager()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		remote, err := mgr.ConnectAll(ctx, cfg.MCPServers)
		cancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: MCP servers: %v\n", err)
		}
		added := 0
		for _, t := range remote {
			if _, exists := reg.Get(t.Tool.Name); exists {
				continue
			}
			reg.Register(NewMCPTool(t.ServerName, t.Tool, mgr))
			added++
		}
		if added > 0 {
			fmt.Fprintf(os.Stderr, "MCP: %d tools from %d servers\n", added, len(cfg.MCPServers))
		}
	}
	cleanup := func() {
		if mgr != nil {
			mgr.Close()
		}
	}
	return reg, cleanup, nil
}
