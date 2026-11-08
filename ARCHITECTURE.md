# Edu CLI — Architecture

```
main.go                          calls cmd.Execute
cmd/root.go                      CLI: repl, daemon
internal/config                  JSON config plus environment overrides
internal/client                  OpenAI-compatible chat completions
internal/agent                   one completion per turn
internal/session                 JSON sessions; Manager for TUI harness
internal/daemon                  HTTP API and runner
internal/tui                     home screen and composer
```

## Request path

1. `POST /message` decodes `{text, session_id}`.
2. `router.RouteKey` pins a resume to `session:<id>`. A missing id mints a new session.
3. `runner.RunAgent` loads the session, calls the model once, and saves the transcript.
4. The HTTP handler returns JSON, or SSE when the client asks for `text/event-stream`.

The TUI holds a `session.Manager` (new/resume/picker); each turn calls `runner.RunAgent` with `current` id, same as HTTP.
