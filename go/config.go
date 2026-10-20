package main

import "os"

// Config holds the resolved settings for one `run` invocation.
type Config struct {
	Name        string
	Port        int // 0 means auto-discover from the child's listening socket
	Host        string
	Group       string // optional; omitted from the payload when empty
	RegistryURL string
	Command     []string
}

func defaultConfig() Config {
	host, _ := os.Hostname()
	return Config{
		Host:        host,
		RegistryURL: "http://localhost:5150",
	}
}
