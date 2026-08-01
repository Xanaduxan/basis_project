package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadReadsConfigFromYAML(t *testing.T) {
	yamlData := []byte(`
app:
  host: "127.0.0.1"
  port: 9090
  shutdown_timeout: "15s"

mysql:
  dsn: "user:password@tcp(localhost:3306)/tasks"
  max_open_connections: 20
  max_idle_connections: 5
  connection_max_lifetime: "3m"

redis:
  address: "localhost:6379"
  password: ""
  database: 1
  task_ttl: "5m"

jwt:
  secret: "test-secret"
  lifetime: "1h"

email:
  base_url: "http://localhost:8081"
  timeout: "3s"

rate_limit:
  requests: 100
  window: "1m"
`)

	configPath := filepath.Join(t.TempDir(), "config.yaml")

	if err := os.WriteFile(configPath, yamlData, 0o600); err != nil {
		t.Fatalf("write temporary config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	if cfg.App.Host != "127.0.0.1" {
		t.Errorf("App.Host = %q, want %q", cfg.App.Host, "127.0.0.1")
	}

	if cfg.App.Port != 9090 {
		t.Errorf("App.Port = %d, want %d", cfg.App.Port, 9090)
	}

	if cfg.App.ShutdownTimeout != 15*time.Second {
		t.Errorf(
			"App.ShutdownTimeout = %s, want %s",
			cfg.App.ShutdownTimeout,
			15*time.Second,
		)
	}

	if cfg.MySQL.MaxOpenConnections != 20 {
		t.Errorf(
			"MySQL.MaxOpenConnections = %d, want %d",
			cfg.MySQL.MaxOpenConnections,
			20,
		)
	}

	if cfg.Redis.Database != 1 {
		t.Errorf("Redis.Database = %d, want %d", cfg.Redis.Database, 1)
	}

	if cfg.Redis.TaskTTL != 5*time.Minute {
		t.Errorf(
			"Redis.TaskTTL = %s, want %s",
			cfg.Redis.TaskTTL,
			5*time.Minute,
		)
	}

	if cfg.JWT.Secret != "test-secret" {
		t.Errorf("JWT.Secret = %q, want %q", cfg.JWT.Secret, "test-secret")
	}

	if cfg.Email.BaseURL != "http://localhost:8081" {
		t.Errorf(
			"Email.BaseURL = %q, want %q",
			cfg.Email.BaseURL,
			"http://localhost:8081",
		)
	}

	if cfg.RateLimit.Requests != 100 {
		t.Errorf(
			"RateLimit.Requests = %d, want %d",
			cfg.RateLimit.Requests,
			100,
		)
	}
}
