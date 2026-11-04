# edu coding agent

Local CLI and daemon that send each turn to an OpenAI-compatible model and save the transcript.

```
shan run / shan daemon
  -> config
  -> daemon router + runner
  -> one model call
  -> session store
```

## Run

```bash
go test ./...
```

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
curl -s http://127.0.0.1:8787/message -H "Content-Type: application/json" -d "{\"text\":\"hello\"}"
```

`Accept: text/event-stream` on `POST /message` returns a final `done` event.

Environment:

| Variable | Role |
|---|---|
| `OPENAI_API_KEY` | bearer token; overrides `api_key` in `data/config.json` |
| `OPENAI_BASE_URL` | overrides `base_url`; default `https://api.openai.com/v1` |
| `OPENAI_MODEL` | overrides `model`; default `gpt-4o-mini` |
| `SHAN_WORKSPACE` | process workspace, default is the working directory |
| `SHAN_DATA` | sessions and config |

Copy `config.example.json` to `data/config.json` to set `model`, `base_url`, and `api_key`. `data/` is gitignored. The checked-in example targets DeepSeek (`deepseek-flash` at `https://api.deepseek.com`). Environment variables still override the file.
