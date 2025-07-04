package main

import (
	"fmt"
	"net/http"

	"github.com/codetheuri/todolist/config"
	"github.com/codetheuri/todolist/internal/platform/database"
	"github.com/codetheuri/todolist/pkg/logger"
)

func main() {
	// config.InitDb()
	//initialize logger
	log := logger.NewConsoleLogger()
	logger.SetGlobalLogger(log)

	// load configs
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration", err)
	}
	log.Info("Configuration loaded successfully")

	//initialize the db using GORM
	db, err := database.NewGoRMDB(cfg, log)
	if err != nil {
		log.Fatal("Failed to connect to database", err)
	}
	_ = db // Use db as needed, e.g., for migrations or initial data setup
	
	router := http.NewServeMux()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Tusk is running!")
	})

	// config.LoadConfig()
	// router.SetupRouter()
	serverAddr := fmt.Sprintf(":%d", cfg.ServerPort)
	log.Info(fmt.Sprintf("Server starting on %s", serverAddr))
	if err := http.ListenAndServe(serverAddr, router); err != nil {
		log.Fatal("Server failed to start", err)
	}

}
