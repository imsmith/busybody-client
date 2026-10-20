package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

const usage = `busybody — register a local dev app with a busybody directory

Usage:
  busybody run --name NAME [--port PORT] [--host HOST] [--group GROUP] [--registry-url URL] -- CMD [ARGS...]

The command after -- is launched as a child. busybody registers it, heartbeats
every 60s, forwards SIGTERM/SIGINT to it, and exits with its exit code.`

func main() {
	if len(os.Args) < 2 || os.Args[1] != "run" {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	cfg, err := parseRunArgs(os.Args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "busybody: %v\n\n%s\n", err, usage)
		os.Exit(2)
	}
	os.Exit(run(cfg))
}

// parseRunArgs parses the arguments after the `run` subcommand.
func parseRunArgs(args []string) (Config, error) {
	cfg := defaultConfig()
	i := 0
	for i < len(args) {
		a := args[i]
		if a == "--" {
			cfg.Command = args[i+1:]
			break
		}
		if i+1 >= len(args) {
			return cfg, fmt.Errorf("flag %q needs a value", a)
		}
		val := args[i+1]
		switch a {
		case "--name":
			cfg.Name = val
		case "--port":
			p, err := strconv.Atoi(val)
			if err != nil || p < 1 || p > 65535 {
				return cfg, fmt.Errorf("--port must be an integer 1-65535, got %q", val)
			}
			cfg.Port = p
		case "--host":
			cfg.Host = val
		case "--group":
			if val == "" {
				return cfg, errors.New("--group must not be empty")
			}
			cfg.Group = val
		case "--registry-url":
			cfg.RegistryURL = val
		default:
			return cfg, fmt.Errorf("unknown flag %q", a)
		}
		i += 2
	}
	if cfg.Name == "" {
		return cfg, errors.New("--name is required")
	}
	if len(cfg.Command) == 0 {
		return cfg, errors.New("no command given after --")
	}
	return cfg, nil
}
