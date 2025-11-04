package bootstrap

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	// for _, module := range appModules {
	// 	module.RegisterRoutes(router)
	// }
   mainRouter.Route("/api", func(r router.Router) {
		// Register routes from all modules onto this sub-router.
		for _, module := range appModules {
			module.RegisterRoutes(r)
		}
	})
	//middleware
	var handler http.Handler = mainRouter
	handler = middleware.CORS(cfg.CORSOrigins, log)(handler)
	handler = middleware.SecurityHeaders(handler)
	handler = middleware.Logger(log)(handler)
	handler = middleware.Recovery(log)(handler)
	handler = middleware.RequestID()(handler)

	// Setup HTTP Server with Timeouts
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.ServerPort),
		Handler:      handler,
		ReadTimeout:  5 * time.Second, // Timeouts prevent slowloris attacks and resource hangs
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 1. Create Listener and check port availability early
	ln, err := net.Listen("tcp", srv.Addr) // Use srv.Addr for consistent port definition
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}

	actualAddr := ln.Addr().(*net.TCPAddr)
	log.Info(fmt.Sprintf("Server is listening on port %d", actualAddr.Port))

	// 2. Start the Server in a Goroutine (Non-blocking)
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			// Log fatal error if the server fails to start, but ignore http.ErrServerClosed (which is expected during graceful shutdown)
			log.Fatal("Server failed to listen or serve", err)
		}
	}()

	// 3. Graceful Shutdown Listener
	// Create a channel to listen for OS interrupt and termination signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-quit
	log.Warn("Received shutdown signal", "signal", sig.String())

	// 4. Shut Down Server
	// Create a context with a 30-second timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Info("Attempting to shut down gracefully...")
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server shutdown failed (forcing close)", err)
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Info("Server shut down gracefully.")
	return nil

}
