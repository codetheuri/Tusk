package main

import (
	"github.com/codetheuri/todolist/config"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/internal/bootstrap"
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

	log.Info("Starting application...")
	if err := bootstrap.Run(cfg, log); err != nil {
		log.Fatal("Application failed to start", err)
	}

	

}
