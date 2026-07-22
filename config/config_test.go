package config

import (
	"os"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-12345")
	os.Setenv("DB_DRIVER", "sqlite")
	os.Setenv("DB_NAME", ":memory:")
	defer func() {
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("DB_DRIVER")
		os.Unsetenv("DB_NAME")
	}()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected successful config load, got err: %v", err)
	}

	if cfg.ServerPort != 8080 {
		t.Errorf("expected default ServerPort 8080, got %d", cfg.ServerPort)
	}
	if cfg.DBDriver != "sqlite" {
		t.Errorf("expected DBDriver sqlite, got %s", cfg.DBDriver)
	}
}
