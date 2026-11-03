# edu coding agent

Runnable spine of a local daemon and CLI coding agent.

Same call chain as the full repo, with the production machinery reduced to the pieces that boot a local agent:

```
shan run / shan daemon
  -> config
  -> daemon router + runner          (one-shot uses the runner directly)
  -> prompt + skills
  -> agent loop
       -> permissions, hooks, audit
       -> tools (read, write, edit, bash, grep, use_skill)
       -> LLM client
  -> session store
```

Cloud channels, voice, computer use, MCP, marketplace, and the Desktop UI are named under `internal/` as package docs. They need Shannon Cloud or Kocoro Desktop, so they are not reimplemented here. See `ARCHITECTURE.md`.

## Run

```bash
go test ./...
go run ./cmd/shan demo
```

`demo` uses a scripted model, writes a file, edits it, and runs a shell command. No API key.

`shan` and `shan repl` open the full-screen home view: brand mark, tips, recent sessions, a connected line, and the composer. A pipe falls back to a line reader. `?` lists commands. Ctrl+C twice quits.

Live model:

```bash
set OPENAI_API_KEY=sk-...
go run ./cmd/shan run "summarize README.md"
go run ./cmd/shan daemon
```

Daemon listens on `127.0.0.1:8787`.

```bash
curl -s http://127.0.0.1:8787/health
curl -s http://127.0.0.1:8787/message -H "Content-Type: application/json" -d "{\"text\":\"read README.md\"}"
```

`Accept: text/event-stream` on `POST /message` streams `tool_status` events and a final `done` event.

Environment:

| Variable | Role |
|---|---|
| `OPENAI_API_KEY` | bearer token; overrides `api_key` in `data/config.json` |
| `OPENAI_BASE_URL` | overrides `base_url`; default `https://api.openai.com/v1` |
| `OPENAI_MODEL` | overrides `model`; default `gpt-4o-mini` |
| `SHAN_WORKSPACE` | tool root, default is the process working directory |
| `SHAN_DATA` | sessions, audit log, config |

Copy `config.example.json` to `data/config.json` to set `model`, `base_url`, `api_key`, `max_iterations`, and `denied_commands`. `data/` is gitignored. The checked-in example targets DeepSeek (`deepseek-flash` at `https://api.deepseek.com`). Environment variables still override the file.
