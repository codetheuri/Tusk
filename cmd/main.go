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

	// //initialize the db using GORM
	// db, err := database.NewGoRMDB(cfg, log)
	// if err != nil {
	// 	log.Fatal("Failed to connect to database", err)
	// }
	// _ = db // Use db as needed, e.g., for migrations or initial data setup

	// router := http.NewServeMux()
	// router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	fmt.Fprintln(w, "Tusk is running!")
	// })

	// serverAddr := fmt.Sprintf(":%d", cfg.ServerPort)
	// log.Info(fmt.Sprintf("Server starting on %s", serverAddr))
	// if err := http.ListenAndServe(serverAddr, router); err != nil {
	// 	log.Fatal("Server failed to start", err)
	// }

	// config.LoadConfig()

	// router.SetupRouter()
	// log.Info("Server running on :8081")
	// // log.Fatal(http.ListenAndServe(":8081", nil))
	// serverAddr := fmt.Sprintf(":%d", cfg.ServerPort)
	// log.Info(fmt.Sprintf("Server starting on %s", serverAddr))
	// if err := http.ListenAndServe(serverAddr, nil); err != nil {
	// 	log.Fatal("Server failed to start", err)
	// }

}
