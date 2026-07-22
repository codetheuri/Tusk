package main

import (
	"os"

	"github.com/codetheuri/tusk/config"
	"github.com/codetheuri/tusk/internal/app"
	"github.com/codetheuri/tusk/pkg/logger"
)

func main() {
	log := logger.NewConsoleLogger()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration", err)
		os.Exit(1)
	}

	application, err := app.New(cfg, log)
	if err != nil {
		log.Fatal("Application setup failed", err)
		os.Exit(1)
	}

	if err := application.Run(); err != nil {
		log.Fatal("Server exited with error", err)
		os.Exit(1)
	}
}
