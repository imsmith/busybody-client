package main

import (
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
	// PID 1 (init) is running but this test process opened the listener above only
	// in the other test; a fresh sub-scope with no listener should report none.
	// Use a very unlikely PID that has no sockets.
	if _, ok := discoverPort(999999); ok {
		t.Error("expected ok=false for a pid with no listening sockets")
	}
}
