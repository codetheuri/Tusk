package main

import (
	"github.com/codetheuri/todolist/config"
	"github.com/codetheuri/todolist/internal/bootstrap"
	"github.com/codetheuri/todolist/pkg/logger"
	// "github.com/codetheuri/todolist/pkg/mailer"
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

	//test mail
	// mailerService := mailer.NewMailerService(cfg, log)
	// log.Info("--- Attempting to send test email ---")
	// testRecipient := "theurij113@gmail.com" 
	// testSubject := "Tusk Mailer Test from CLI"
	// testBody := "Hello from Tusk! This is a test email sent using Go's net/smtp. If you see this, the mailer is working!"
	// err = mailerService.SendEmail([]string{testRecipient}, testSubject, testBody)
    // if err != nil {
	// 	log.Error("Failed to send test email", err)
	// } else {
	// 	log.Info("Test email sent successfully! Check your inbox for " + testRecipient)
	// }
	// log.Info("--- Test email attempt finished ---")


	log.Info("Configuration loaded successfully")

	log.Info("Starting application...")
	if err := bootstrap.Run(cfg, log); err != nil {
		log.Fatal("Application failed to start", err)
	}

}
