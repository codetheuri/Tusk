package config

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"

	"github.com/codetheuri/todolist/pkg/errors"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/joho/godotenv"
)

var DB *sql.DB

type Config struct {
	DBUser     string
	DBPass     string
	DBHost     string
	DBPort     string
	DBName     string
	ServerPort int
	LOG_LEVEL  string
	JWTSecret  string
	AppName    string
	AppVersion string
	AppMode    string
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load(".env")
	if err != nil && !os.IsNotExist(err) {
		return nil, errors.ConfigError("Error loading .env file", err)
	}
	cfg := &Config{
		DBUser: os.Getenv("DB_USER"),
		DBPass: os.Getenv("DB_PASS"),
		DBHost: os.Getenv("DB_HOST"),
		// DBPort: os.Getenv("DB_PORT"),
		DBName:     os.Getenv("DB_NAME"),
		LOG_LEVEL:  os.Getenv("LOG_LEVEL"),
		JWTSecret:  os.Getenv("JWT_SECRET"),
		AppName:    os.Getenv("APP_NAME"),
		AppVersion: os.Getenv("APP_VERSION"),
		AppMode:    os.Getenv("APP_MODE"),
	}
	dbPortStr := os.Getenv("DB_PORT")
	if dbPortStr == "" {
		return nil, errors.ConfigError("DB_PORT not set in .env", nil)
	}
	dbPort, err := strconv.Atoi(dbPortStr)
	if err != nil {
		return nil, errors.ConfigError("Invalid DB_PORT value in .env", err)
	}
	cfg.DBPort = strconv.Itoa(dbPort)
	//server port
	serverPortStr := os.Getenv("SERVER_PORT")
	if serverPortStr == "" {
		serverPortStr = "8080" // default port
	}
	serverPort, err := strconv.Atoi(serverPortStr)
	if err != nil {
		return nil, errors.ConfigError(fmt.Sprintf("Invalid SERVER_PORT value : %s", serverPortStr), err)
	}
	cfg.ServerPort = serverPort

	//basic validation
	if cfg.DBUser == "" || cfg.DBPass == "" || cfg.DBHost == "" || cfg.DBName == "" {
		return nil, errors.ConfigError("Missing required database configuration", nil)
	}
	return cfg, nil
	
}

// func InitDb() {
// 	err := godotenv.Load(".env")
// 	if err != nil {
// 		log.Fatal("Error loading .env file")
// 	}

// 	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
// 		os.Getenv("DB_USER"),
// 		os.Getenv("DB_PASS"),
// 		os.Getenv("DB_HOST"),
// 		os.Getenv("DB_PORT"),
// 		os.Getenv("DB_NAME"),
// 	)

// 	DB, err = sql.Open("mysql", dsn)
// 	if err != nil {
// 		log.Fatalf("Error connecting to the database: %v", err)
// 	}
// 	if err = DB.Ping(); err != nil {
// 		log.Fatal("Database unreachable:", err)
// 	}

// 	log.Println("Database connected ✅")
// }
