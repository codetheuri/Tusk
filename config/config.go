package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"


	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"     // PostgreSQL driver
	_ "gorm.io/driver/sqlite" // SQLite driver
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
	DBMaxIdleConns    int
	DBMaxOpenConns    int
	DBConnMaxLifetime int
	CORSOrigins    []string

	//mailer config
	MailerHost     string
	MailerPort     int
	MailerUsername     string
	MailerPassword string
	MailerSender   string
	
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load(".env")
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}
	cfg := &Config{
		DBUser: os.Getenv("DB_USER"),
		DBPass: os.Getenv("DB_PASS"),
		DBHost: os.Getenv("DB_HOST"),
		// DBPort: os.Getenv("DB_PORT"),
		DBName:            os.Getenv("DB_NAME"),
		DBDriver:          os.Getenv("DB_DRIVER"),
		LOG_LEVEL:         os.Getenv("LOG_LEVEL"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		// AccessTokenTTL:    os.Getenv("ACCESS_TOKEN_TTL"),
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
	JWTSecret :=   os.Getenv("JWT_SECRET")
	if JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET not set in .env")
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

	//basic validation
	if cfg.DBDriver != "sqlite" && (cfg.DBUser == "" || cfg.DBPass == "" || cfg.DBHost == "" || cfg.DBName == "") {
		return nil, fmt.Errorf("missing required database configuration")
	}
	//sqlite
	if cfg.DBDriver == "sqlite" && cfg.DBName == "" {
		return nil, fmt.Errorf("DB_NAME not set for sqlite driver (should be file path)")
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
	//dsn based on DB driver
	switch cfg.DBDriver {
	case "mysql":
		cfg.DbURL = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.DBUser,
			cfg.DBPass,
			cfg.DBHost,
			cfg.DBPort,
			cfg.DBName,
		)
	case "postgres", "pgsql":
		cfg.DbURL = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
			cfg.DBHost,
			cfg.DBUser,
			cfg.DBPass,
			cfg.DBName,
			cfg.DBPort,
		)
	case "sqlite":
		cfg.DbURL = cfg.DBName

	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER: %s", cfg.DBDriver)
	}
	return cfg, nil

}
