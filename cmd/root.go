package cmd

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kocoro/edu-cli/internal/client"
	"github.com/kocoro/edu-cli/internal/config"
	"github.com/kocoro/edu-cli/internal/daemon"
	"github.com/kocoro/edu-cli/internal/mcp"
	"github.com/kocoro/edu-cli/internal/session"
	"github.com/kocoro/edu-cli/internal/tools"
	"github.com/kocoro/edu-cli/internal/tui"
	"github.com/mattn/go-isatty"
)

// Execute is the CLI entry. The root main.go calls this and nothing else.
func Execute() {
	args := os.Args[1:]
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	switch name {
	case "daemon":
		addr := flagValue(args[1:], "addr", "127.0.0.1:8787")
		if err := runDaemon(addr); err != nil {
			fail(err)
		}
	case "", "repl":
		if err := runRepl(); err != nil {
			fail(err)
		}
	case "-h", "--help", "help":
		usage()
	default:
		fail(fmt.Errorf("unknown command %q (try edu --help)", name))
	}
}

func usage() {
	fmt.Println(`Edu CLI

  edu                      full-screen session (same as edu repl)
  edu daemon [--addr h:p]  HTTP API on 127.0.0.1:8787
  edu repl                 full-screen session

Set OPENAI_API_KEY, or api_key in data/config.json, for daemon and repl.
A local endpoint (localhost or port 11434) does not need a key.
EDU_DATA holds sessions and config.`)
}

func runDaemon(addr string) error {
	runner, err := liveRunner()
	if err != nil {
		return err
	}
	defer runner.Close()
	fmt.Fprintf(os.Stderr, "listening on http://%s\n", addr)
	return http.ListenAndServe(addr, daemon.NewServer(runner).Handler())
}

func runRepl() error {
	cfg, err := config.Load("", "")
	if err != nil {
		return err
	}
	llm := &client.OpenAI{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
	}
	runner, err := buildRunner(cfg.Workspace, cfg.DataDir, llm)
	if err != nil {
		return err
	}
	defer runner.Close()
	if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		if cfg.APIKey == "" && !localEndpoint(cfg.BaseURL) {
			return fmt.Errorf("set OPENAI_API_KEY or api_key in data/config.json")
		}
		return lineRepl(runner, cfg)
	}
	mgr := session.NewManager(runner.Sessions)
	if _, err := mgr.NewSession(); err != nil {
		return err
	}
	recent, _ := runner.Sessions.Recent(1)
	activity := make([]tui.Activity, 0, len(recent))
	for _, row := range recent {
		activity = append(activity, tui.Activity{Title: row.Title, When: row.UpdatedAt, Msgs: row.MsgCount})
	}
	return tui.Start(tui.Options{
		Agent:     "default",
		Model:     cfg.Model,
		Endpoint:  cfg.BaseURL,
		CWD:       cfg.Workspace,
		Version:   "dev",
		Connected: endpointReachable(cfg.BaseURL),
		Activity:  activity,
		Sessions:  mgr,
		Models: func() []tui.ModelChoice {
			rows := make([]tui.ModelChoice, 0, 8)
			for _, m := range client.DeepSeekModels() {
				rows = append(rows, tui.ModelChoice{Name: m.Name, Detail: m.Detail})
			}
			for _, m := range client.OllamaModels("http://127.0.0.1:11434") {
				rows = append(rows, tui.ModelChoice{Name: m.Name, Detail: m.Detail})
			}
			return rows
		},
		ApplyModel: func(name string) (string, error) {
			return llm.Use(name), nil
		},
		Submit: func(text string) (string, error) {
			if cfg.APIKey == "" && !localEndpoint(cfg.BaseURL) {
				return "", fmt.Errorf("set OPENAI_API_KEY or api_key in data/config.json")
			}
			id := ""
			if cur := mgr.Current(); cur != nil {
				id = cur.ID
			}
			result, err := runner.RunAgent(daemon.Request{Text: text, SessionID: id})
			if err != nil {
				return "", err
			}
			return result.Reply, nil
		},
	})
}

func lineRepl(runner *daemon.Runner, cfg config.Config) error {
	mgr := session.NewManager(runner.Sessions)
	if _, err := mgr.NewSession(); err != nil {
		return err
	}
	return tui.Run(os.Stdin, os.Stdout, func(text string) (string, error) {
		if strings.HasPrefix(text, "/") {
			switch strings.Fields(text)[0] {
			case "/session":
				parts := strings.Fields(text)
				if len(parts) >= 2 && parts[1] == "new" {
					if _, err := mgr.NewSession(); err != nil {
						return "", err
					}
					return "Started new session", nil
				}
				if len(parts) >= 3 && parts[1] == "resume" {
					target := parts[2]
					var sess *session.Session
					var err error
					if n, perr := strconv.Atoi(target); perr == nil {
						sess, err = mgr.ResumeByListIndex(n)
					} else {
						sess, err = mgr.Resume(target)
					}
					if err != nil {
						return "", err
					}
					return fmt.Sprintf("Resumed session %s (%d messages)", session.DisplayTitle(sess), len(sess.Messages)), nil
				}
				return "Usage: /session new | /session resume <# or id>", nil
			}
		}
		id := ""
		if cur := mgr.Current(); cur != nil {
			id = cur.ID
		}
		result, err := runner.RunAgent(daemon.Request{Text: text, SessionID: id})
		if err != nil {
			return "", err
		}
		if cur := mgr.Current(); cur != nil && cur.Title == "New session" {
			if t := session.TitleFromUserLine(text); t != "" {
				cur.Title = t
			}
		}
		_ = mgr.RefreshCurrent()
		if cur := mgr.Current(); cur != nil && cur.Title == "New session" {
			if t := session.TitleFromUserLine(text); t != "" {
				cur.Title = t
			}
		}
		_ = mgr.Save()
		return result.Reply, nil
	})
}

func localEndpoint(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || u.Port() == "11434"
}

func endpointReachable(raw string) bool {
	addr, err := endpointAddr(raw)
	if err != nil {
		return false
	}
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// endpointAddr is host:port. HTTPS and HTTP without a port use 443 and 80.
// net.Dial rejects a host that has no port, which made every cloud URL look down.
func endpointAddr(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return "", fmt.Errorf("bad endpoint")
	}
	if u.Port() != "" {
		return u.Host, nil
	}
	port := "443"
	if strings.EqualFold(u.Scheme, "http") {
		port = "80"
	}
	return net.JoinHostPort(u.Hostname(), port), nil
}

func liveRunner() (*daemon.Runner, error) {
	cfg, err := config.Load("", "")
	if err != nil {
		return nil, err
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("set OPENAI_API_KEY or api_key in data/config.json")
	}
	return buildRunner(cfg.Workspace, cfg.DataDir, &client.OpenAI{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
	})
}

func buildRunner(workspace, data string, llm client.LLMClient) (*daemon.Runner, error) {
	cfg, err := config.Load(data, workspace)
	if err != nil {
		return nil, err
	}
	cfg.Workspace = workspace
	cfg.DataDir = data
	store, err := session.NewStore(filepath.Join(data, "sessions"))
	if err != nil {
		return nil, err
	}
	reg, cleanup, err := tools.RegisterAll(cfg)
	if err != nil {
		return nil, err
	}
	return &daemon.Runner{
		Cfg:        cfg,
		LLM:        llm,
		Sessions:   store,
		Tools:      reg,
		MCPContext: mcp.BuildContext(cfg.MCPServers),
		Cleanup:    cleanup,
	}, nil
}

func flagValue(args []string, name, fallback string) string {
	prefix := "--" + name
	for i, a := range args {
		if a == prefix && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(a, prefix+"=") {
			return strings.TrimPrefix(a, prefix+"=")
		}
	}
	return fallback
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
