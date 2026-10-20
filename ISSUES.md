# Issues

Tracked here until they migrate to a real issue tracker. Newest first.

## The Go binary is Linux-only; the README presents it as broadly portable

**Filed:** 2026-08-04
**Resolved:** 2026-10-15, option 1 — README now states Linux-only
**Reported from:** as-built architecture review

### Symptom

`README.md:9-10` positions the binary as the portable artifact:

```markdown
- **`go/` → `busybody` binary** — a single static binary for containers
  (Alpine) and hosts with no interpreter. `busybody run -- CMD`.
```

It does not build off Linux. `discoverPort` is defined only in
`go/discover_linux.go:14`, and the `_linux` filename suffix is an
implicit build constraint — that file is excluded from any non-Linux
build. `go/run.go:103` calls it unconditionally:

```go
			if port, ok := discoverPort(pid); ok {
```

Verified:

```text
$ GOOS=darwin GOARCH=arm64 go build ./...
# busybody-client
./run.go:103:19: undefined: discoverPort
```

This is a compile failure, not a graceful degradation. There is no
`discover_darwin.go`, no `discover_windows.go`, and no fallback
`discover_other.go` behind a negated build tag.

### Why it happens

Port discovery is implemented by parsing `/proc` —
`go/discover_linux.go:19` reads `/proc/net/tcp` and `/proc/net/tcp6`,
and line 30 walks `/proc/<pid>/fd` to match sockets to the child. That
is a Linux-only interface, correctly isolated into a `_linux` file. What
is missing is any build for the platforms that file excludes.

The shell wrapper is Linux-only in fact too, for a different reason:
`sh/busybody-run:38` discovers the port with `ss -Htlnp` (iproute2),
which the README does disclose as a dependency at line 11. So both
artifacts are Linux-only; only the Go one is described otherwise.

### What should change

Pick one, and say so in the README either way:

1. **Document the constraint.** Change "a single static binary for
   containers (Alpine) and hosts with no interpreter" to say Linux
   explicitly. This is the honest one-line fix and is probably right for
   a dev-box tool.
2. **Make it build everywhere.** Add a `discover_other.go` behind
   `//go:build !linux` returning `(0, false)`, so non-Linux builds
   compile and require `--port`. The `--port` flag already exists and
   already bypasses discovery entirely, so the degraded path is real
   and usable — but it changes a compile error into a runtime "no
   listening port after ~10s" message, which is worse feedback unless
   the non-Linux path also errors early and clearly.
3. **Implement discovery per-platform.** Most work, only worth it if
   someone actually runs this on macOS.

---

## Verified accurate: the wire contract holds across all three client implementations

**Filed:** 2026-08-04
**Reported from:** as-built architecture review

### Symptom

None. Recorded as a verified result so nobody re-audits this — three
independent implementations of the same protocol is exactly the shape
that invites a drift hunt.

The server, `BusybodyWeb.ApiController.register/2` in the busybody
repo, guards `pid` as a string and `port` as an integer:

```elixir
      when is_binary(name) and byte_size(name) > 0
      and is_integer(port) and port in 1..65_535
      and is_binary(pid) and is_binary(host) and byte_size(host) > 0 do
```

All three clients satisfy that:

- **Go** — `go/register.go:13-14` types the struct as
  `Port int \`json:"port"\`` / `Pid string \`json:"pid"\``, and line 22
  converts with `Pid: strconv.Itoa(pid)`, carrying the comment
  `// server requires pid as a string`. Port serializes unquoted.
- **sh** — `sh/busybody-run:72` emits
  `{\"name\":\"$_name\",\"port\":$1,\"pid\":\"$CHILD_PID\",\"host\":\"$_host\"}`
  — port unquoted, pid quoted, with `name` and `host` passed through the
  `json_escape` helper at line 63.
- **Elixir** — `Busybody.Client.build_payload/1` sends
  `pid: System.pid() |> to_string()` and an integer `port` from
  `resolve_port/1`.

Defaults agree as well: `go/config.go:18` and `sh/busybody-run:12` both
default `RegistryURL` to `http://localhost:5150`, matching busybody's
own default port, and both heartbeat at 60s
(`go/run.go:15 heartbeatInterval = 60 * time.Second`;
`sh/busybody-run:83 sleep 60`). All three POST `/api/register`.

The one asymmetry worth knowing is not a contract violation: the server
accepts an empty `pid` (only `is_binary/1`, no `byte_size` check),
tracked in `busybody/ISSUES.md`. No client sends one, so the looseness
is currently unexercised.
