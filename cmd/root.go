package cmd

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kocoro/skeleton/internal/client"
	"github.com/kocoro/skeleton/internal/config"
	"github.com/kocoro/skeleton/internal/daemon"
	"github.com/kocoro/skeleton/internal/session"
	"github.com/kocoro/skeleton/internal/tui"
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
	case "run":
		task := strings.TrimSpace(strings.Join(args[1:], " "))
		if task == "" {
			fail(fmt.Errorf("usage: edu run <task>"))
		}
		if err := runTask(task); err != nil {
			fail(err)
		}
	case "", "repl":
		if err := runRepl(); err != nil {
			fail(err)
		}
	case "-h", "--help", "help":
		usage()
	default:
		task := strings.TrimSpace(strings.Join(args, " "))
		if err := runTask(task); err != nil {
			fail(err)
		}
	}
}

func usage() {
	fmt.Println(`Edu CLI

  edu                      full-screen session (same as edu repl)
  edu run <task>           one-shot task with the configured model
  edu daemon [--addr h:p]  HTTP API on 127.0.0.1:8787
  edu repl                 full-screen session

Set OPENAI_API_KEY, or api_key in data/config.json, for run, daemon, and repl.
A local endpoint (localhost or port 11434) does not need a key.
EDU_DATA holds sessions and config.`)
}

func runTask(task string) error {
	runner, err := liveRunner()
	if err != nil {
		return err
	}
	result, err := runner.RunAgent(daemon.Request{Text: task})
	if err != nil {
		return err
	}
	fmt.Printf("session: %s\n%s\n", result.SessionID, result.Reply)
	return nil
}

func runDaemon(addr string) error {
	runner, err := liveRunner()
	if err != nil {
		return err
	}
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
	if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		if cfg.APIKey == "" && !localEndpoint(cfg.BaseURL) {
			return fmt.Errorf("set OPENAI_API_KEY or api_key in data/config.json")
		}
		return lineRepl(runner)
	}
	recent, _ := runner.Sessions.Recent(1)
	activity := make([]tui.Activity, 0, len(recent))
	for _, row := range recent {
		activity = append(activity, tui.Activity{Title: row.Title, When: row.UpdatedAt, Msgs: row.MsgCount})
	}
	var sessionID string
	return tui.Start(tui.Options{
		Agent:     "default",
		Model:     cfg.Model,
		Endpoint:  cfg.BaseURL,
		CWD:       cfg.Workspace,
		Version:   "dev",
		Connected: endpointReachable(cfg.BaseURL),
		Activity:  activity,
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
			result, err := runner.RunAgent(daemon.Request{Text: text, SessionID: sessionID})
			if result.SessionID != "" {
				sessionID = result.SessionID
			}
			if err != nil {
				return "", err
			}
			return result.Reply, nil
		},
	})
}

func lineRepl(runner *daemon.Runner) error {
	var sessionID string
	return tui.Run(os.Stdin, os.Stdout, func(text string) (string, error) {
		result, err := runner.RunAgent(daemon.Request{Text: text, SessionID: sessionID})
		if result.SessionID != "" {
			sessionID = result.SessionID
		}
		if err != nil {
			return "", err
		}
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
	return &daemon.Runner{
		Cfg:      cfg,
		LLM:      llm,
		Sessions: store,
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
