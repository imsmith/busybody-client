package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildPayload_PidIsString(t *testing.T) {
	p := buildPayload(Config{Name: "foo", Host: "devbox"}, 4000, 12345)
	b, _ := json.Marshal(p)
	var raw map[string]any
	_ = json.Unmarshal(b, &raw)
	if _, ok := raw["pid"].(string); !ok {
		t.Errorf("pid must serialize as JSON string, got %T (%v)", raw["pid"], raw["pid"])
	}
	if _, ok := raw["port"].(float64); !ok { // JSON numbers decode to float64
		t.Errorf("port must serialize as JSON number, got %T", raw["port"])
	}
	if raw["pid"] != "12345" {
		t.Errorf("pid = %v, want \"12345\"", raw["pid"])
	}
}

func TestPostRegister_SendsCorrectRequest(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	p := buildPayload(Config{Name: "foo", Host: "devbox"}, 4000, 12345)
	if err := postRegister(srv.Client(), srv.URL, p); err != nil {
		t.Fatalf("postRegister error: %v", err)
	}
	if gotPath != "/api/register" {
		t.Errorf("path = %q, want /api/register", gotPath)
	}
	if want := `"name":"foo"`; !contains(gotBody, want) {
		t.Errorf("body %q missing %q", gotBody, want)
	}
	if want := `"pid":"12345"`; !contains(gotBody, want) {
		t.Errorf("body %q missing %q", gotBody, want)
	}
}

func TestPostRegister_Non2xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()
	err := postRegister(srv.Client(), srv.URL, buildPayload(Config{Name: "x", Host: "h"}, 1, 2))
	if err == nil {
		t.Error("expected error on 503")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
