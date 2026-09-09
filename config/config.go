package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // the one PostgreSQL driver, matching gorm.io/driver/postgres
	"github.com/joho/godotenv"
)

type Config struct {
	DBUser            string
	DBPass            string
	DBHost            string
	DBPort            string
	DBName            string
	DBDriver          string
	ServerPort        int
	LOG_LEVEL         string
	JWTSecret         string
	AccessTokenTTL    time.Duration
	AppName           string
	AppVersion        string
	AppMode           string
	DbURL             string
	DBSSLMode         string
	DBMaxIdleConns    int
	DBMaxOpenConns    int
	DBConnMaxLifetime int
	CORSOrigins       []string

	// HTTP server limits.
	// ReadTimeout must accommodate the largest request a client may legitimately
	// send over a slow connection — a mobile upload on 2G, for example. Set it too
	// low and such requests are severed mid-flight with no useful error.
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	ShutdownTimeout     time.Duration
	MaxRequestBodyBytes int64

	// Rate limiting, per client IP.
	RateLimitBurst float64 // bucket capacity — how large a spike is tolerated
	RateLimitRPS   float64 // sustained refill rate, requests per second

	// DocsEnabled controls whether /docs and /openapi.json are served.
	// Defaults to false in production. API endpoints are unaffected either way.
	DocsEnabled bool

	//mailer config
	MailerHost     string
	MailerPort     int
	MailerUsername string
	MailerPassword string
	MailerSender   string
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load(".env")
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}
	cfg := &Config{
		DBUser:            os.Getenv("DB_USER"),
		DBPass:            os.Getenv("DB_PASS"),
		DBHost:            os.Getenv("DB_HOST"),
		DBName:            os.Getenv("DB_NAME"),
		DBDriver:          os.Getenv("DB_DRIVER"),
		LOG_LEVEL:         os.Getenv("LOG_LEVEL"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		AppName:           os.Getenv("APP_NAME"),
		AppVersion:        os.Getenv("APP_VERSION"),
		AppMode:           os.Getenv("APP_MODE"),
		DBMaxIdleConns:    10,
		DBMaxOpenConns:    100,
		DBConnMaxLifetime: 60, // default value in seconds

		// Mailer configuration
		MailerHost:     os.Getenv("MAIL_HOST"),
		MailerUsername: os.Getenv("MAIL_USERNAME"),
		MailerPassword: os.Getenv("MAIL_PASSWORD"),
		MailerSender:   os.Getenv("MAIL_SENDER"),
	}
	// A short secret is not a weak secret in the way a short password is — it is a
	// forgeable one. HS256 signatures are only as strong as the key, so anything
	// brute-forceable lets an attacker mint valid tokens for any user. 32 bytes is
	// the practical floor for HMAC-SHA256.
	if len(cfg.JWTSecret) < minJWTSecretLength {
		if cfg.JWTSecret == "" {
			return nil, fmt.Errorf("JWT_SECRET not set in .env")
		}
		return nil, fmt.Errorf("JWT_SECRET must be at least %d characters (got %d)", minJWTSecretLength, len(cfg.JWTSecret))
	}
	accessTokenTTLStr := os.Getenv("ACCESS_TOKEN_TTL")
	if accessTokenTTLStr == "" {

		accessTokenTTLStr = "24h"
	}
	// Parse the duration string (e.g., "3600s", "1h", "24h")
	parsedTTL, err := time.ParseDuration(accessTokenTTLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid ACCESS_TOKEN_TTL value: %s, error: %w", accessTokenTTLStr, err)
	}
	cfg.AccessTokenTTL = parsedTTL

	if cfg.DBDriver == "" {
		return nil, fmt.Errorf("DB_DRIVER not set in .env")
	}
	dbPortStr := os.Getenv("DB_PORT")
	if dbPortStr == "" && cfg.DBDriver != "sqlite" {
		return nil, fmt.Errorf("DB_PORT not set in .env for non-sqlite driver")
	}
	if cfg.DBDriver != "sqlite" {
		dbPort, err := strconv.Atoi(dbPortStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_PORT value in .env: %w", err)
		}
		cfg.DBPort = strconv.Itoa(dbPort)
	}

	//server port
	serverPortStr := os.Getenv("SERVER_PORT")
	if serverPortStr == "" {
		serverPortStr = "8080" // default port
	}
	serverPort, err := strconv.Atoi(serverPortStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_PORT value %s: %w", serverPortStr, err)
	}
	cfg.ServerPort = serverPort
	//mail port
	mailerPortStr := os.Getenv("MAIL_PORT")
	if mailerPortStr != "" {
		mailPort, err := strconv.Atoi(mailerPortStr)
		if err != nil {
			return nil, fmt.Errorf("invalid MAIL_PORT value %s: %w", mailerPortStr, err)
		}
		cfg.MailerPort = mailPort
	}

	// Name the variables that are actually missing.
	//
	// An error like "missing required database configuration" is true but useless:
	// it sends the reader to re-read a whole file looking for which of four keys is
	// blank. Listing them turns a hunt into a fix — and this is exactly where a
	// typo in a key name (DB_PASSWORD instead of DB_PASS) surfaces.
	required := map[string]string{
		"DB_HOST": cfg.DBHost,
		"DB_NAME": cfg.DBName,
		"DB_USER": cfg.DBUser,
		"DB_PASS": cfg.DBPass,
	}
	var missing []string
	for _, key := range []string{"DB_HOST", "DB_NAME", "DB_USER", "DB_PASS"} {
		if strings.TrimSpace(required[key]) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required database configuration: %s", strings.Join(missing, ", "))
	}
	if val := os.Getenv("DB_MAX_IDLE_CONNS"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.DBMaxIdleConns = i
		}
	}
	if val := os.Getenv("DB_MAX_OPEN_CONNS"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.DBMaxOpenConns = i
		}
	}
	if val := os.Getenv("DB_CONN_MAX_LIFETIME"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.DBConnMaxLifetime = i
		}
	}

	corsOriginStr := os.Getenv("ALLOWED_ORIGINS")
	if corsOriginStr != "" {
		cfg.CORSOrigins = strings.Split(corsOriginStr, ",")
	} else {
		cfg.CORSOrigins = []string{}
	}

	// --- HTTP server limits ---
	cfg.ReadTimeout = envDuration("READ_TIMEOUT", 30*time.Second)
	cfg.WriteTimeout = envDuration("WRITE_TIMEOUT", 60*time.Second)
	cfg.IdleTimeout = envDuration("IDLE_TIMEOUT", 120*time.Second)
	cfg.ShutdownTimeout = envDuration("SHUTDOWN_TIMEOUT", 30*time.Second)
	cfg.MaxRequestBodyBytes = int64(envInt("MAX_REQUEST_BODY_BYTES", 10<<20)) // 10 MiB

	// --- Rate limiting ---
	cfg.RateLimitBurst = float64(envInt("RATE_LIMIT_BURST", 30))
	cfg.RateLimitRPS = float64(envInt("RATE_LIMIT_RPS", 10))

	// --- API documentation ---
	// Off by default in production; on by default everywhere else. An explicit
	// DOCS_ENABLED always wins, so production docs remain possible deliberately.
	cfg.DocsEnabled = envBool("DOCS_ENABLED", !cfg.IsProduction())

	// --- Database TLS ---
	// Defaults to require in production: a hardcoded "disable" silently sends
	// credentials and data over an unencrypted socket with no way to turn it on.
	defaultSSL := "disable"
	if cfg.IsProduction() {
		defaultSSL = "require"
	}
	cfg.DBSSLMode = getEnvOr("DB_SSLMODE", defaultSSL)
	// DSN. PostgreSQL only — see the default case for why.
	switch cfg.DBDriver {
	case "postgres", "pgsql":
		cfg.DbURL = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
			cfg.DBHost,
			cfg.DBUser,
			cfg.DBPass,
			cfg.DBName,
			cfg.DBPort,
			cfg.DBSSLMode,
		)
	default:
		// Fail here rather than at the first query. MySQL and SQLite support was
		// dropped when primary keys became UUIDs: neither has a native UUID type,
		// so their schema could no longer describe the same rows as the Go models.
		// A driver that connects but cannot read its own tables is worse than one
		// that refuses to start.
		return nil, fmt.Errorf(
			"unsupported DB_DRIVER %q: Tusk targets PostgreSQL only. Set DB_DRIVER=postgres",
			cfg.DBDriver)
	}
	return cfg, nil

}

// minJWTSecretLength is the practical floor for an HS256 signing key.
const minJWTSecretLength = 32

// IsProduction reports whether the application is running in production mode.
// Behaviour that should differ between environments — documentation exposure,
// log formatting, error verbosity — keys off this rather than inspecting
// APP_MODE in scattered places.
func (c *Config) IsProduction() bool {
	mode := strings.ToLower(strings.TrimSpace(c.AppMode))
	return mode == "production" || mode == "prod"
}

// envInt reads an integer environment variable, falling back to def when unset
// or unparseable. Configuration should never fail closed on a typo in an
// optional tuning knob; required values are validated explicitly instead.
func envInt(key string, def int) int {
	if raw := os.Getenv(key); raw != "" {
		if v, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
			return v
		}
	}
	return def
}

// envDuration reads a Go duration string such as "30s" or "2m".
func envDuration(key string, def time.Duration) time.Duration {
	if raw := os.Getenv(key); raw != "" {
		if v, err := time.ParseDuration(strings.TrimSpace(raw)); err == nil {
			return v
		}
	}
	return def
}

// envBool reads a boolean environment variable, accepting the forms
// strconv.ParseBool understands ("1", "true", "TRUE", "0", "false", …).
func envBool(key string, def bool) bool {
	if raw := os.Getenv(key); raw != "" {
		if v, err := strconv.ParseBool(strings.TrimSpace(raw)); err == nil {
			return v
		}
	}
	return def
}

// getEnvOr returns the environment variable value, or def when unset or empty.
func getEnvOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
