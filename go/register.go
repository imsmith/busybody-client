package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type payload struct {
	Name  string `json:"name"`
	Port  int    `json:"port"`
	Pid   string `json:"pid"`
	Host  string `json:"host"`
	Group string `json:"group,omitempty"`
}

func buildPayload(cfg Config, port, pid int) payload {
	return payload{
		Name:  cfg.Name,
		Port:  port,
		Pid:   strconv.Itoa(pid), // server requires pid as a string
		Host:  cfg.Host,
		Group: cfg.Group,
	}
}

func postRegister(client *http.Client, registryURL string, p payload) error {
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}
	resp, err := client.Post(registryURL+"/api/register", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("registry returned %d", resp.StatusCode)
	}
	return nil
}
