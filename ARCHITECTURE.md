# Architecture

Package layout matches the full Kocoro repo. Each file names the production symbol it stands in for.

```
cmd/shan                         cmd/root.go, cmd/daemon.go
internal/config                  internal/config
internal/client                  internal/client/gateway.go   Complete
internal/prompt                  internal/prompt/builder.go
internal/skills                  internal/skills              SKILL.md loader
internal/session                 internal/session             JSON sessions
internal/permissions             internal/permissions         admit before Run
internal/hooks                   internal/hooks               before/after tool
internal/audit                   internal/audit               JSONL tool log
internal/agent                   internal/agent/loop.go       AgentLoop.run
internal/tools                   internal/tools/register.go   local tools
internal/daemon                  internal/daemon/server.go    HTTP + runner
internal/tui                     internal/tui                 home screen + composer
internal/mcp                     internal/mcp                 doc only
internal/memory                  internal/memory              doc only
internal/schedule                internal/schedule            doc only
```

## Request path

1. `POST /message` decodes `{text, session_id}` (`daemon/server.go`, full repo `handleMessage`).
2. `router.RouteKey` pins a resume to `session:<id>`. A missing id mints a new session (`daemon/router.go`, full repo `ComputeRouteKey`).
3. `runner.RunAgent` loads the session, registers tools, builds one loop, and saves the transcript (`daemon/runner.go`, full repo `RunAgent`).
4. `agent.Loop.Run` calls the model until the reply has no tool call (`agent/loop.go`, full repo `AgentLoop.run`).
5. Each tool call is admitted by `permissions.Admit`, then `Tool.Run`. Failures are tool-result text, and the loop continues.
6. Hooks fire around the call. Audit appends one JSONL line.
7. The HTTP handler returns JSON, or SSE when the client asks for `text/event-stream`.

One-shot `shan run` and `shan demo` call the same `runner.RunAgent`. They do not open a second code path.

## Loop

```
messages = system + history + new user text
for step in 1..max:
    reply = llm.Complete(messages, tool schemas)
    if reply has no tool call:
        return reply text
    for each call:
        admit -> hook before -> tool.Run -> hook after -> audit
        append a role=tool message
```

The model sees numbered `file_read` output. `file_edit` matches raw file bytes, and `old_string` must occur once. `bash` is `cmd /c` on Windows and `sh -c` elsewhere, with the working directory set to the workspace. `rm`, `del`, `format`, `shutdown`, `mkfs`, and `dd` are denied before the process starts.

## What stays in the full repo

These exist there as real subsystems. This tree only records where they sit:

- `internal/mcp` — external tool servers, deferred loading, reconnect
- `internal/memory` — sidecar recall over a local socket
- `internal/schedule` — cron runs that re-enter the same runner
- voice (`internal/koe`), computer use, marketplace, Cloud channels, Desktop RPC

A new feature in the full repo is another tool, another HTTP route, or another branch inside `AgentLoop.run`. The spine above does not change.
