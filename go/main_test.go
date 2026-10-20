package main

import (
	"os"
	"testing"
)

func TestParseRunArgs_Defaults(t *testing.T) {
	cfg, err := parseRunArgs([]string{"--name", "foo", "--", "./server", "-x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "foo" {
		t.Errorf("Name = %q, want foo", cfg.Name)
	}
	if cfg.Port != 0 {
		t.Errorf("Port = %d, want 0 (auto-discover)", cfg.Port)
	}
	if cfg.RegistryURL != "http://localhost:5150" {
		t.Errorf("RegistryURL = %q, want default", cfg.RegistryURL)
	}
	host, _ := os.Hostname()
	if cfg.Host != host {
		t.Errorf("Host = %q, want %q", cfg.Host, host)
	}
	if len(cfg.Command) != 2 || cfg.Command[0] != "./server" || cfg.Command[1] != "-x" {
		t.Errorf("Command = %v, want [./server -x]", cfg.Command)
	}
}

func TestParseRunArgs_Explicit(t *testing.T) {
	cfg, err := parseRunArgs([]string{
		"--name", "bar", "--port", "4000", "--host", "devbox", "--group", "infra",
		"--registry-url", "http://reg:9999", "--", "run.sh",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 4000 || cfg.Host != "devbox" || cfg.Group != "infra" || cfg.RegistryURL != "http://reg:9999" {
		t.Errorf("bad explicit parse: %+v", cfg)
	}
}

func TestParseRunArgs_Errors(t *testing.T) {
	if _, err := parseRunArgs([]string{"--", "cmd"}); err == nil {
		t.Error("expected error when --name missing")
	}
	if _, err := parseRunArgs([]string{"--name", "foo"}); err == nil {
		t.Error("expected error when no command after --")
	}
}
