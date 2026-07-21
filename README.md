# busybody-client

Standalone wrappers that register any local HTTP dev app with a
[busybody](../busybody) directory and heartbeat every 60 seconds — the
same behavior as the Elixir `Busybody.Client`, for apps in any language.

Two artifacts, one CLI contract:

- **`go/` → `busybody` binary** — a single static binary for containers
  (Alpine) and hosts with no interpreter. `busybody run -- CMD`.
- **`sh/busybody-run`** — a POSIX-sh sidecar for the dev box. Needs
  `curl` and `ss` (iproute2). `busybody-run -- CMD`.

Both launch your app as a child, discover its listening port (or take
`--port`), register with the child's PID, heartbeat, forward
`SIGTERM`/`SIGINT`, and exit with the child's exit code. Every registry
call is best-effort: if busybody is down, your app still runs.

## Contract

```text
busybody run   --name NAME [--port PORT] [--host HOST] [--registry-url URL] -- CMD [ARGS...]   # binary
busybody-run   --name NAME [--port PORT] [--host HOST] [--registry-url URL] -- CMD [ARGS...]   # script
```

| Flag | Default | Meaning |
|------|---------|---------|
| `--name` | *(required)* | Name shown in the directory |
| `--port` | *(auto-discover)* | Explicit port; skips discovery |
| `--host` | system hostname | Host shown and health-checked |
| `--registry-url` | `http://localhost:5150` | Where busybody runs |

If your app binds several ports or forks a separate listener process,
pass `--port` — discovery matches only the direct child PID.

## Examples

```bash
# Go binary, auto-discovered port
busybody run --name paperclip -- ./paperclip-server

# Shell script, explicit port
busybody-run --name docs --port 8080 -- python3 -m http.server 8080
```

## Build

```bash
make build      # produces ./busybody
make test       # Go unit + integration tests
make test-sh    # shell smoke test (needs python3)
```

## License

AGPL-3.0
