package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_ValidConfig(t *testing.T) {
	path := writeConfig(t, `
server:
  port: "9554"
auth:
  enabled: true
  username: user
  password: pass
streams:
  cam1:
    url: https://example.com/stream.m3u8
metrics:
  enabled: false
  interval: 10s
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Port != "9554" {
		t.Errorf("port = %q, want %q", cfg.Server.Port, "9554")
	}
	if !cfg.Auth.Enabled {
		t.Error("auth.enabled = false, want true")
	}
	if cfg.Auth.Username != "user" {
		t.Errorf("username = %q, want %q", cfg.Auth.Username, "user")
	}
	if cfg.Auth.Password != "pass" {
		t.Errorf("password = %q, want %q", cfg.Auth.Password, "pass")
	}
	if len(cfg.Streams) != 1 {
		t.Fatalf("streams count = %d, want 1", len(cfg.Streams))
	}
	if cfg.Streams["cam1"].URL != "https://example.com/stream.m3u8" {
		t.Errorf("stream url = %q", cfg.Streams["cam1"].URL)
	}
	if cfg.Metrics.Enabled {
		t.Error("metrics.enabled = true, want false")
	}
	if cfg.Metrics.Interval != 10*time.Second {
		t.Errorf("metrics.interval = %v, want 10s", cfg.Metrics.Interval)
	}
}

func TestLoad_Defaults(t *testing.T) {
	path := writeConfig(t, `
streams:
  cam1:
    url: https://example.com/stream.m3u8
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Port != "8554" {
		t.Errorf("default port = %q, want %q", cfg.Server.Port, "8554")
	}
	if !cfg.Metrics.Enabled {
		t.Error("default metrics.enabled = false, want true")
	}
	if cfg.Metrics.Interval != 30*time.Second {
		t.Errorf("default metrics.interval = %v, want 30s", cfg.Metrics.Interval)
	}
}

func TestLoad_NoStreams(t *testing.T) {
	path := writeConfig(t, `
server:
  port: "8554"
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for no streams")
	}
}

func TestLoad_EmptyURL(t *testing.T) {
	path := writeConfig(t, `
streams:
  cam1:
    url: ""
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeConfig(t, `{{{invalid`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoad_MultipleStreams(t *testing.T) {
	path := writeConfig(t, `
streams:
  cam1:
    url: https://example.com/stream1.m3u8
  cam2:
    url: https://example.com/stream2.m3u8
  cam3:
    url: https://example.com/stream3.m3u8
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Streams) != 3 {
		t.Errorf("streams count = %d, want 3", len(cfg.Streams))
	}
}
