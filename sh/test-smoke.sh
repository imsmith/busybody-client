#!/bin/sh
# Smoke test: busybody-run must POST a registration for a child that binds a port.
# Requires: python3 (test-only stub registry), a free port, curl, ss.
set -eu

HERE=$(cd "$(dirname "$0")" && pwd)
STUB_PORT=5199
CHILD_PORT=4399
CAPTURE=$(mktemp)

# Minimal stub registry: record each POST body to $CAPTURE, always 200.
python3 - "$STUB_PORT" "$CAPTURE" <<'PY' &
import sys, http.server
port, cap = int(sys.argv[1]), sys.argv[2]
class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get("Content-Length", 0))
        open(cap, "ab").write(self.rfile.read(n) + b"\n")
        self.send_response(200); self.end_headers(); self.wfile.write(b'{"status":"ok"}')
    def log_message(self, *a): pass
http.server.HTTPServer(("127.0.0.1", port), H).serve_forever()
PY
STUB_PID=$!
trap 'kill "$STUB_PID" 2>/dev/null || true; rm -f "$CAPTURE"' EXIT
sleep 0.5

# Run the wrapper around a child that binds CHILD_PORT and holds it open.
"$HERE/busybody-run" --name shsmoke --registry-url "http://127.0.0.1:$STUB_PORT" \
  -- sh -c "exec python3 -m http.server $CHILD_PORT" &
WRAP_PID=$!
sleep 2
kill "$WRAP_PID" 2>/dev/null || true
sleep 0.5

if grep -q '"name":"shsmoke"' "$CAPTURE" && grep -q "\"port\":$CHILD_PORT" "$CAPTURE"; then
    echo "PASS: registration captured with discovered port"
    exit 0
fi
echo "FAIL: expected registration not captured. Captured:"; cat "$CAPTURE"
exit 1
