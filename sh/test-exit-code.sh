#!/bin/sh
# Verify busybody-run propagates the child's exit code, including when the
# wrapper itself is signaled — the path the happy-path smoke test misses.
# Uses an unreachable registry so best-effort registration cannot affect exit.
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
BR="$HERE/busybody-run"
fail=0

# Case 1: child exits 7 on its own.
"$BR" --name exit7 --port 4001 --registry-url http://127.0.0.1:1 -- sh -c 'exit 7'
code=$?
if [ "$code" -eq 7 ]; then echo "PASS: clean child exit code propagated (7)";
else echo "FAIL: clean exit got $code want 7"; fail=1; fi

# Case 2: wrapper is TERM'd; child traps TERM and exits 42; wrapper must report 42.
"$BR" --name term42 --port 4002 --registry-url http://127.0.0.1:1 \
    -- sh -c 'trap "exit 42" TERM; while :; do sleep 1; done' &
wp=$!
sleep 1
kill -TERM "$wp"
wait "$wp"
code=$?
if [ "$code" -eq 42 ]; then echo "PASS: signaled-wrapper propagates child exit code (42)";
else echo "FAIL: signaled exit got $code want 42"; fail=1; fi

exit "$fail"
