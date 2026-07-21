package main

import (
	"fmt"
	"net"
	"os"
	"testing"
)

func TestDiscoverPort_FindsOwnListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	wantPort := ln.Addr().(*net.TCPAddr).Port

	got, ok := discoverPort(os.Getpid())
	if !ok {
		t.Fatal("discoverPort returned ok=false, expected to find our listener")
	}
	if got != wantPort {
		t.Errorf("discoverPort = %d, want %d", got, wantPort)
	}
}

func TestDiscoverPort_NoListener(t *testing.T) {
	pid := unusedPID(t)
	if _, ok := discoverPort(pid); ok {
		t.Errorf("expected ok=false for pid %d which has no /proc entry", pid)
	}
}

// unusedPID returns a PID that currently has no /proc entry, so discoverPort
// must find no sockets for it. Deterministic: scans upward for the first gap.
func unusedPID(t *testing.T) int {
	t.Helper()
	for pid := 2; pid < 1<<22; pid++ {
		if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); os.IsNotExist(err) {
			return pid
		}
	}
	t.Skip("no unused pid found in /proc")
	return 0
}
