package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

// Loop timings — package vars so tests can shrink them.
var (
	heartbeatInterval = 60 * time.Second
	discoverInterval  = 200 * time.Millisecond
	discoverTimeout   = 10 * time.Second
)

// run launches the child, registers/heartbeats best-effort, forwards
// SIGTERM/SIGINT, and returns the child's exit code.
func run(cfg Config) int {
	cmd := exec.Command(cfg.Command[0], cfg.Command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		log.Printf("busybody: failed to start child: %v", err)
		return 1
	}
	pid := cmd.Process.Pid

	client := &http.Client{Timeout: 3 * time.Second}
	stop := make(chan struct{})

	// Forward termination signals to the child.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		for sig := range sigCh {
			_ = cmd.Process.Signal(sig)
		}
	}()

	// Resolve the port, then register + heartbeat until the child exits.
	regDone := make(chan struct{})
	go func() {
		registerLoop(cfg, client, pid, stop)
		close(regDone)
	}()

	err := cmd.Wait()
	close(stop)
	<-regDone // wait for registerLoop to observe stop and return before we do
	signal.Stop(sigCh)
	close(sigCh)
	return exitCode(err)
}

// registerLoop resolves the port (flag or discovery), registers once, then
// re-registers every heartbeatInterval until stop is closed.
func registerLoop(cfg Config, client *http.Client, pid int, stop <-chan struct{}) {
	port := cfg.Port
	if port == 0 {
		port = waitForPort(pid, stop)
		if port == 0 {
			return // stopped before a port appeared
		}
	}
	p := buildPayload(cfg, port, pid)

	register := func() {
		if err := postRegister(client, cfg.RegistryURL, p); err != nil {
			log.Printf("busybody: register failed (will retry): %v", err)
		}
	}
	register()

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			register()
		}
	}
}

// waitForPort polls discoverPort until it finds a listening port, the timeout
// elapses (after which it keeps trying), or stop is closed. Returns 0 if stopped.
func waitForPort(pid int, stop <-chan struct{}) int {
	ticker := time.NewTicker(discoverInterval)
	defer ticker.Stop()
	deadline := time.Now().Add(discoverTimeout)
	warned := false
	for {
		select {
		case <-stop:
			return 0
		case <-ticker.C:
			if port, ok := discoverPort(pid); ok {
				return port
			}
			if !warned && time.Now().After(deadline) {
				log.Printf("busybody: no listening port after %s; still watching", discoverTimeout)
				warned = true
			}
		}
	}
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	return 1
}
