package main

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// killHelperProcess finds and kills any TestHelperProcess child bound to
// childPort. The helper blocks forever (select{}) by design, so nothing in
// run() ever terminates it; the parent-side run() goroutine in
// TestRun_RegistersHeartbeatsAndPropagatesExit is intentionally left running
// too (see comment there). Without this cleanup the orphaned child keeps a
// dup of the test binary's stdout pipe open, which makes `go test` hang
// until its WaitDelay expires and report a spurious top-level FAIL even
// though every subtest passed. This is process hygiene only — it does not
// touch any assertion.
func killHelperProcess(t *testing.T, childPort int) {
	t.Helper()
	marker := []byte("BUSYBODY_TEST_PORT=" + strconv.Itoa(childPort))
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return
	}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		environ, err := os.ReadFile("/proc/" + e.Name() + "/environ")
		if err != nil {
			continue
		}
		if !bytes.Contains(environ, marker) {
			continue
		}
		cmdline, err := os.ReadFile("/proc/" + e.Name() + "/cmdline")
		if err != nil || !strings.Contains(string(cmdline), "TestHelperProcess") {
			continue
		}
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
}

// TestHelperProcess is re-executed as the child app: it binds the port given in
// BUSYBODY_TEST_PORT and blocks until signaled.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("BUSYBODY_TEST_HELPER") != "1" {
		return
	}
	ln, err := net.Listen("tcp", "127.0.0.1:"+os.Getenv("BUSYBODY_TEST_PORT"))
	if err != nil {
		os.Exit(3)
	}
	defer ln.Close()
	select {} // block until the parent signals us
}

func TestRun_RegistersHeartbeatsAndPropagatesExit(t *testing.T) {
	// Pick a free port for the helper child to bind.
	tmp, _ := net.Listen("tcp", "127.0.0.1:0")
	childPort := tmp.Addr().(*net.TCPAddr).Port
	tmp.Close()

	var count int32
	var mu sync.Mutex
	var lastBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		mu.Lock()
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		lastBody = string(b)
		mu.Unlock()
		w.WriteHeader(200)
	}))
	defer srv.Close()

	// Speed the loops up for the test.
	heartbeatInterval = 150 * time.Millisecond
	discoverInterval = 20 * time.Millisecond
	discoverTimeout = 3 * time.Second
	defer func() {
		heartbeatInterval = 60 * time.Second
		discoverInterval = 200 * time.Millisecond
		discoverTimeout = 10 * time.Second
	}()

	cfg := Config{
		Name:        "helper",
		Host:        "testhost",
		RegistryURL: srv.URL,
		Command: []string{
			os.Args[0], "-test.run", "TestHelperProcess",
		},
	}
	// Pass helper env through the child by wrapping the command in the current env.
	os.Setenv("BUSYBODY_TEST_HELPER", "1")
	os.Setenv("BUSYBODY_TEST_PORT", strconv.Itoa(childPort))
	defer os.Unsetenv("BUSYBODY_TEST_HELPER")
	defer os.Unsetenv("BUSYBODY_TEST_PORT")

	defer killHelperProcess(t, childPort)

	done := make(chan int, 1)
	go func() { done <- run(cfg) }()

	// Wait for at least two registrations (initial + one heartbeat).
	deadline := time.After(4 * time.Second)
	for atomic.LoadInt32(&count) < 2 {
		select {
		case <-deadline:
			t.Fatalf("only %d registrations before timeout", atomic.LoadInt32(&count))
		case <-time.After(20 * time.Millisecond):
		}
	}

	mu.Lock()
	body := lastBody
	mu.Unlock()
	if !contains(body, `"port":`+strconv.Itoa(childPort)) {
		t.Errorf("registered body %q missing discovered port %d", body, childPort)
	}

	// Terminate the child so the wrapper's run() returns; then wait for it to
	// fully return (joining its goroutines) before this test's deferred var
	// restores fire — otherwise the restore races the still-running registerLoop.
	killHelperProcess(t, childPort)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("run() did not return after the child was killed")
	}
}

// TestRun_ExitCodePropagates checks that a child exiting non-zero yields that code.
func TestRun_ExitCodePropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	discoverTimeout = 300 * time.Millisecond
	discoverInterval = 20 * time.Millisecond
	defer func() {
		discoverTimeout = 10 * time.Second
		discoverInterval = 200 * time.Millisecond
	}()

	cfg := Config{
		Name:        "exiter",
		Host:        "h",
		RegistryURL: srv.URL,
		Command:     []string{"sh", "-c", "exit 7"},
	}
	code := run(cfg)
	if code != 7 {
		t.Errorf("exit code = %d, want 7", code)
	}
}

// ensure exec import is used even if the helper path changes
var _ = exec.Command
