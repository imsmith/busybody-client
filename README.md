# busybody-client

Standalone wrappers that register any local HTTP dev app with a
[busybody](../busybody) directory and heartbeat every 60 seconds — the
same behavior as the Elixir `Busybody.Client`, for apps in any language.

Three artifacts: an Elixir library for Phoenix apps, and two
process wrappers with one CLI contract for everything else.

## Elixir: `busybody_client`

A dependency-free Mix library exposing `Busybody.Client`, a GenServer
for a Phoenix supervision tree. It resolves the bound port from your
endpoint, registers, and heartbeats. It is deliberately **not** the
busybody app: no Phoenix, no endpoint, nothing starts unless you put
it in your tree. The `go/` and `sh/` directories are ignored by Mix.

### Do not pick a port

**The app must not choose its own port.** Busybody exists so that no
one has to. Bind `port: 0` and let the OS assign one; the client reads
the bound port from the endpoint and reports it. A fixed port defeats
the directory, and the client logs a warning when it sees one.

The port is set in exactly one place, `config/runtime.exs`, and it is
one line:

```elixir
config :my_app, MyAppWeb.Endpoint,
  http: [ip: {0, 0, 0, 0}, port: String.to_integer(System.get_env("PORT", "0"))]
```

Remove the `http:` lines `phx.new` put in `config/dev.exs` and the prod
block of `runtime.exs`; a second `http:` anywhere silently wins and you
are back on 4000. Check with:

```sh
grep -n 'http:' config/*.exs
```

Likewise never write `server: false` into config: an explicit `:server`
key overrides `mix phx.server`, the endpoint never binds, and there is
no port to report. Set `server: true` only, and only for releases.

```elixir
# mix.exs
{:busybody_client, github: "imsmith/busybody-client", only: :dev}

# application.ex, after your endpoint
{Busybody.Client, name: "my_app", endpoint: MyAppWeb.Endpoint}
```

Add `group: "infra"` to that child spec to show the app under a group in
the directory. Same meaning as `--group` on the wrappers below. Omit it
and the app is ungrouped.

Guard the child with `Code.ensure_loaded?(Busybody.Client)` if the same
`application.ex` must boot in prod and test, where the dep is absent.
If the endpoint has no bound port (`mix run`, Mix tasks, tests) the client
idles and retries each heartbeat; it never crashes the app.
Test the library with `mix test`.

## Process wrappers

Two artifacts, one CLI contract. **Both are Linux-only**: port discovery
reads `/proc` (binary) or runs `ss` (script). The Go binary does not
build for other platforms.

- **`go/` → `busybody` binary** — a single static Linux binary for
  containers (Alpine) and hosts with no interpreter. `busybody run -- CMD`.
- **`sh/busybody-run`** — a POSIX-sh sidecar for the Linux dev box. Needs
  `curl` and `ss` (iproute2). `busybody-run -- CMD`.

Both launch your app as a child, discover its listening port (or take
`--port`), register with the child's PID, heartbeat, forward
`SIGTERM`/`SIGINT`, and exit with the child's exit code. Every registry
call is best-effort: if busybody is down, your app still runs.

## Contract

```text
busybody run   --name NAME [--port PORT] [--host HOST] [--group GROUP] [--registry-url URL] -- CMD [ARGS...]   # binary
busybody-run   --name NAME [--port PORT] [--host HOST] [--group GROUP] [--registry-url URL] -- CMD [ARGS...]   # script
```

| Flag | Default | Meaning |
|------|---------|---------|
| `--name` | *(required)* | Name shown in the directory |
| `--port` | *(auto-discover)* | Explicit port; skips discovery |
| `--host` | system hostname | Host shown and health-checked |
| `--group` | *(none)* | Optional group membership shown on the card |
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
