package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMustLoad_Success(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := []byte("env: local\naccess_token_ttl: 2h\nrefresh_token_ttl: 4h\njwt_secret: test_secret\ngrpc:\n  port: 12345\n  timeout: 1h\npostgres:\n  host: localhost\n  port: 5432\n  user: u\n  password: p\n  dbname: db\n  sslmode: disable\n  max_conns: 5\n  min_conns: 1\n  connect_timeout: 5s\n")
	if err := os.WriteFile(cfgPath, content, 0o644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	t.Setenv("CONFIG_PATH", cfgPath)

	cfg := MustLoad()
	if cfg.GRPC.Port != 12345 || cfg.JwtSecret != "test_secret" || cfg.AccessTokenTTL.Hours() != 2 {
		t.Fatalf("config not parsed correctly: %+v", cfg)
	}
}

func TestMustLoad_PanicWhenPathEmpty(t *testing.T) {
	// Ensure no env variable set
	t.Setenv("CONFIG_PATH", "")
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic when CONFIG_PATH empty")
		}
	}()
	MustLoad()
}

func TestMustLoad_PanicWhenFileMissing(t *testing.T) {
	t.Setenv("CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.yaml"))
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic when config file missing")
		}
	}()
	MustLoad()
}
