package config

import (
	"strings"
	"testing"
	"time"
)

// validSecret is long enough to satisfy minJWTSecretLength.
const validSecret = "test-secret-key-that-is-long-enough-for-hs256"

// baseEnv sets the minimum required configuration for a successful load.
// t.Setenv restores the previous value automatically when the test ends, which
// is why these tests do not manage cleanup by hand.
func baseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", validSecret)
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_NAME", "tusk_test")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_PASS", "postgres")
}

func TestLoadConfig_Defaults(t *testing.T) {
	baseEnv(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected successful config load, got err: %v", err)
	}

	if cfg.ServerPort != 8080 {
		t.Errorf("expected default ServerPort 8080, got %d", cfg.ServerPort)
	}
	if cfg.DBDriver != "postgres" {
		t.Errorf("expected DBDriver postgres, got %s", cfg.DBDriver)
	}
	if cfg.ReadTimeout != 30*time.Second {
		t.Errorf("expected default ReadTimeout 30s, got %s", cfg.ReadTimeout)
	}
	if cfg.MaxRequestBodyBytes != 10<<20 {
		t.Errorf("expected default body limit 10MiB, got %d", cfg.MaxRequestBodyBytes)
	}
	if cfg.RateLimitRPS != 10 || cfg.RateLimitBurst != 30 {
		t.Errorf("expected default rate limit 10rps/30burst, got %v/%v", cfg.RateLimitRPS, cfg.RateLimitBurst)
	}
}

// A short signing key is forgeable, not merely weak — loading must fail rather
// than start a server that will happily accept minted tokens.
func TestLoadConfig_RejectsShortJWTSecret(t *testing.T) {
	baseEnv(t)
	t.Setenv("JWT_SECRET", "too-short")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected short JWT_SECRET to be rejected, got nil error")
	}
	if !strings.Contains(err.Error(), "at least") {
		t.Errorf("expected error to state the minimum length, got: %v", err)
	}
}

func TestLoadConfig_RejectsMissingJWTSecret(t *testing.T) {
	baseEnv(t)
	t.Setenv("JWT_SECRET", "")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected missing JWT_SECRET to be rejected, got nil error")
	}
}

func TestIsProduction(t *testing.T) {
	cases := map[string]bool{
		"production": true,
		"prod":       true,
		"PRODUCTION": true,
		" prod ":     true,
		"dev":        false,
		"staging":    false,
		"":           false,
	}
	for mode, want := range cases {
		cfg := &Config{AppMode: mode}
		if got := cfg.IsProduction(); got != want {
			t.Errorf("IsProduction(%q) = %v, want %v", mode, got, want)
		}
	}
}

// Documentation exposes the full route and schema map, so it defaults off in
// production and on everywhere else — unless explicitly overridden.
func TestDocsEnabled_DefaultsByMode(t *testing.T) {
	t.Run("off in production", func(t *testing.T) {
		baseEnv(t)
		t.Setenv("APP_MODE", "production")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("load failed: %v", err)
		}
		if cfg.DocsEnabled {
			t.Error("expected docs disabled by default in production")
		}
	})

	t.Run("on in development", func(t *testing.T) {
		baseEnv(t)
		t.Setenv("APP_MODE", "dev")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("load failed: %v", err)
		}
		if !cfg.DocsEnabled {
			t.Error("expected docs enabled by default outside production")
		}
	})

	t.Run("explicit override wins in production", func(t *testing.T) {
		baseEnv(t)
		t.Setenv("APP_MODE", "production")
		t.Setenv("DOCS_ENABLED", "true")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("load failed: %v", err)
		}
		if !cfg.DocsEnabled {
			t.Error("expected explicit DOCS_ENABLED=true to override the production default")
		}
	})
}

// Database traffic must not silently fall back to an unencrypted connection in
// production just because DB_SSLMODE was left unset.
func TestDBSSLMode_DefaultsToRequireInProduction(t *testing.T) {
	baseEnv(t)
	t.Setenv("APP_MODE", "production")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.DBSSLMode != "require" {
		t.Errorf("expected DBSSLMode 'require' in production, got %q", cfg.DBSSLMode)
	}
}

func TestEnvHelpers_FallBackOnGarbage(t *testing.T) {
	t.Setenv("TUSK_TEST_INT", "not-a-number")
	if got := envInt("TUSK_TEST_INT", 42); got != 42 {
		t.Errorf("envInt fallback = %d, want 42", got)
	}

	t.Setenv("TUSK_TEST_DUR", "not-a-duration")
	if got := envDuration("TUSK_TEST_DUR", time.Minute); got != time.Minute {
		t.Errorf("envDuration fallback = %s, want 1m", got)
	}

	t.Setenv("TUSK_TEST_BOOL", "maybe")
	if got := envBool("TUSK_TEST_BOOL", true); !got {
		t.Error("envBool fallback = false, want true")
	}
}

// Unsupported drivers must be refused at startup, not at the first query.
// MySQL and SQLite were dropped when primary keys became UUIDs — neither has a
// native UUID type, so their schema can no longer describe the same rows as the
// Go models. A process that connects but cannot read its own tables is worse
// than one that refuses to start.
func TestLoadConfig_RejectsUnsupportedDrivers(t *testing.T) {
	for _, driver := range []string{"mysql", "sqlite", "mssql", ""} {
		t.Run(driver, func(t *testing.T) {
			baseEnv(t)
			t.Setenv("DB_DRIVER", driver)

			if _, err := LoadConfig(); err == nil {
				t.Fatalf("expected driver %q to be rejected", driver)
			}
		})
	}
}

func TestLoadConfig_AcceptsPgsqlAlias(t *testing.T) {
	baseEnv(t)
	t.Setenv("DB_DRIVER", "pgsql")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected pgsql to be accepted as an alias: %v", err)
	}
	if !strings.Contains(cfg.DbURL, "dbname=tusk_test") {
		t.Errorf("DSN does not name the database: %s", cfg.DbURL)
	}
}

// A configuration error should name what is wrong. The generic form of this
// message cost a real debugging round-trip when DB_PASSWORD was set instead of
// DB_PASS: every key looked present, and the error pointed at nothing.
func TestLoadConfig_NamesMissingDatabaseKeys(t *testing.T) {
	baseEnv(t)
	t.Setenv("DB_PASS", "")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected missing DB_PASS to be rejected")
	}
	if !strings.Contains(err.Error(), "DB_PASS") {
		t.Errorf("error must name the missing key, got: %v", err)
	}
}

func TestLoadConfig_NamesAllMissingKeys(t *testing.T) {
	baseEnv(t)
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASS", "")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected missing keys to be rejected")
	}
	for _, key := range []string{"DB_USER", "DB_PASS"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error should list %s, got: %v", key, err)
		}
	}
}
