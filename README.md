# Edu CLI

Local CLI and HTTP daemon: chat with an OpenAI-compatible model, run tools and MCP, and save transcripts under `data/`.

```
edu run / edu daemon
  -> config
  -> daemon router + runner
  -> agent loop (tools + MCP)
  -> session store
```

## Run

```bash
go test ./...
```

`go run .` and `go run . repl` open the full-screen home view: brand mark, tips, recent sessions, a connected line, and the composer. A pipe falls back to a line reader. `?` lists commands. Ctrl+C twice quits. The root `main.go` only calls `cmd.Execute`.

Live model:

```bash
set OPENAI_API_KEY=sk-...
go run . run "summarize README.md"
go run . daemon
```

Daemon listens on `127.0.0.1:8787`.

```bash
curl -s http://127.0.0.1:8787/health
curl -s http://127.0.0.1:8787/message -H "Content-Type: application/json" -d "{\"text\":\"hello\"}"
```

`Accept: text/event-stream` on `POST /message` returns a final `done` event.

Environment:

| Variable | Role |
|---|---|
| `OPENAI_API_KEY` | bearer token; overrides `api_key` in `data/config.json` |
| `OPENAI_BASE_URL` | overrides `base_url`; default `https://api.openai.com/v1` |
| `OPENAI_MODEL` | overrides `model`; default `gpt-4o-mini` |
| `/model` | lists DeepSeek (`deepseek-chat`, `deepseek-reasoner`, `deepseek-flash`) and local Ollama tags. `/model deepseek-chat` switches the next turn to DeepSeek |
| `EDU_WORKSPACE` | process workspace; default is the working directory |
| `EDU_DATA` | sessions and config directory (default `<workspace>/data`) |

Copy `config.example.json` to `data/config.json` to set `model`, `base_url`, `api_key`, and optional `mcp_servers`. `data/` is gitignored. The checked-in example targets DeepSeek (`deepseek-flash` at `https://api.deepseek.com`). Environment variables still override the file.

## Git remote

This project’s Go module is `github.com/kocoro/edu-cli`. Host the git repo as **edu-cli** (or any name you prefer) and point `origin` at it:

```bash
git remote add origin <your-clone-url>
git push -u origin master
```

On Cursor-hosted git (`origin.cursor.com`), use the [origin CLI](https://cursor.com/codebase) from macOS/Linux or WSL: `origin repo create edu-cli`, then add the printed URL as `origin`.
