// Command api is the Tusk HTTP server entrypoint.
package main

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/codetheuri/tusk/v2/config"
	"github.com/codetheuri/tusk/v2/internal/auth"
	"github.com/codetheuri/tusk/v2/pkg/app"
	"github.com/codetheuri/tusk/v2/pkg/logger"
)

func main() {
	// A bootstrap logger, because configuration must be loaded before we know
	// which logger the environment wants — and a configuration failure still
	// needs somewhere to be reported.
	log := logger.NewTextLogger("info")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration", err)
	}

	// Now that the environment is known: JSON in production, readable text
	// locally.
	log = logger.New(cfg.IsProduction(), cfg.LOG_LEVEL)

	application, err := app.New(app.Options{
		Config:      cfg,
		Logger:      log,
		Title:       "Tusk Backend API",
		Version:     "1.0.0",
		Description: "## Official Tusk Enterprise Backend API Documentation\n\nWelcome to the developer documentation for Tusk. Explore Identity, Authentication, and RBAC endpoints below.",
		Contact: &huma.Contact{
			Name:  "API Support",
			Email: "theurij113@gmail.com",
		},
		Tags: []*huma.Tag{
			{Name: "Authentication", Description: "User registration, login, token refresh, logout, and self profile operations"},
			{Name: "Users", Description: "User account management and listing"},
			{Name: "Roles", Description: "Security role management (CRUD)"},
			{Name: "Role Permissions", Description: "Attaching and detaching permission strings to/from roles"},
			{Name: "User Roles", Description: "Assigning and revoking security roles to/from users"},
			{Name: "Permissions", Description: "System permission catalog listing"},
		},
		TagGroups: []app.TagGroup{{
			Name: "IAM",
			Tags: []string{
				"Authentication", "Users", "Roles",
				"Role Permissions", "User Roles", "Permissions",
			},
		}},
	})
	if err != nil {
		log.Fatal("Application setup failed", err)
	}

	// Routes are registered here rather than inside app.New. Tusk's own auth
	// module is just the first consumer of the framework, not a privileged part
	// of it — a service that wants different authentication registers its own
	// module in exactly this place.
	auth.RegisterRoutes(application.API(), application.DB(), cfg, log)

	if err := application.Run(); err != nil {
		log.Fatal("Server exited with error", err)
	}
}
