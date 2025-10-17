package bootstrap

import (
	"fmt"
	"net"
	"net/http"

	"github.com/codetheuri/todolist/config"
	modules "github.com/codetheuri/todolist/internal/app"

	authModule "github.com/codetheuri/todolist/internal/app/auth"
	router "github.com/codetheuri/todolist/internal/app/routers"
	todoModule "github.com/codetheuri/todolist/internal/app/todo"
	"github.com/codetheuri/todolist/internal/platform/database"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/middleware"
	"github.com/codetheuri/todolist/pkg/validators"
	// "github.com/codetheuri/todolist/pkg/validators"
)

// initiliazes and start the application
func Run(cfg *config.Config, log logger.Logger) error {
	//db
	db, err := database.NewGoRMDB(cfg, log)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	//initialize the router

	//initilialize app components
	appValidator := validators.NewValidator()

	//application modules
	var appModules []modules.Module
	authMod := authModule.NewModule(db, log, appValidator, cfg)
	// Example of adding a new module))
	appModules = append(appModules, authModule.NewModule(db, log, appValidator, cfg)) // Example of adding a new module
	appModules = append(appModules, todoModule.NewModule(db, log, appValidator, authMod.TokenService))
	//register routes from all modules
	mainRouter := router.NewRouter(log)
	for _, module := range appModules {
		module.RegisterRoutes(mainRouter)
	}

	//middleware
	var handler http.Handler = mainRouter
	handler = middleware.Logger(log)(handler)
	handler = middleware.Recovery(log)(handler)
	handler = middleware.RequestID()(handler)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.ServerPort))
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}

	// get the actual address assigned (useful if port was 0)
	actualAddr := ln.Addr().(*net.TCPAddr)
	log.Info(fmt.Sprintf("Server is listening on port %d", actualAddr.Port))

	if err := http.Serve(ln, handler); err != nil {
		return fmt.Errorf("server failed to start: %w", err)
	}

	return nil

}
