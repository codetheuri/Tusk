// Package app assembles the pieces every Tusk service needs: a router, the
// middleware chain in the right order, health endpoints, a configured Huma API,
// and graceful shutdown.
//
// It deliberately registers no routes of its own. An application supplies its
// own modules against the returned API:
//
//	a, err := app.New(app.Options{
//	    Config: cfg, Logger: log,
//	    Title:  "Salio API",
//	    Tags:   []*huma.Tag{{Name: "Customers"}},
//	})
//	customer.RegisterRoutes(a.API(), a.DB(), cfg, log)
//	ledger.RegisterRoutes(a.API(), a.DB(), cfg, log)
//	a.Run()
//
// Tusk's own auth module is registered the same way, by cmd/api, rather than
// from inside here — so a service that wants different authentication is not
// fighting the framework to get it.
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/codetheuri/tusk/config"
	"github.com/codetheuri/tusk/database"
	"github.com/codetheuri/tusk/pkg/logger"
	"github.com/codetheuri/tusk/pkg/middleware"
	"github.com/codetheuri/tusk/pkg/response"
)

// TagGroup collects related OpenAPI tags under one heading in the docs sidebar.
type TagGroup struct {
	Name string
	Tags []string
}

// Options configures an App.
type Options struct {
	// Config and Logger are required.
	Config *config.Config
	Logger logger.Logger

	// DB is optional. When nil, New opens a connection from Config — which is
	// what most services want. Supplying one is for tests, and for a service
	// that needs to configure the pool itself.
	DB *gorm.DB

	// Title and Version name the API in its documentation. They default to the
	// application name and version from Config.
	Title   string
	Version string

	// Description is rendered above the endpoint list. Markdown.
	Description string

	Contact *huma.Contact

	// Tags describe the endpoint groups. TagGroups optionally nests them under
	// headings in the sidebar.
	Tags      []*huma.Tag
	TagGroups []TagGroup
}

// App is a configured but not yet running service.
type App struct {
	cfg    *config.Config
	log    logger.Logger
	db     *gorm.DB
	router *chi.Mux
	api    huma.API
}

// New builds the service. Register routes on the returned App, then call Run.
func New(opts Options) (*App, error) {
	if opts.Config == nil {
		return nil, fmt.Errorf("app: Options.Config is required")
	}
	if opts.Logger == nil {
		return nil, fmt.Errorf("app: Options.Logger is required")
	}
	cfg, log := opts.Config, opts.Logger

	db := opts.DB
	if db == nil {
		var err error
		if db, err = database.Connect(cfg, log); err != nil {
			return nil, fmt.Errorf("database connection failed: %w", err)
		}
	}

	r := chi.NewRouter()

	// Middleware order is significant — each layer wraps everything below it.
	// RequestID first so every later log line and error carries a correlation ID;
	// Recovery above the handlers so a panic becomes a 500 rather than killing the
	// process; rate limiting and the body cap before any handler allocates memory
	// on behalf of an unauthenticated caller.
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitBurst, cfg.RateLimitRPS, log)

	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(log))
	r.Use(middleware.Recovery(log))
	r.Use(middleware.CORS(cfg.CORSOrigins, log))
	r.Use(middleware.SecurityHeaders)
	r.Use(rateLimiter.Limit())
	r.Use(middleware.MaxBodyBytes(cfg.MaxRequestBodyBytes))

	// Liveness: is the process up? Deliberately does not touch the database — a
	// liveness probe that fails on a database blip gets the container killed and
	// restarted, which does nothing to fix the database and drops live traffic.
	r.Get("/health", healthHandler(cfg))
	r.Get("/live", healthHandler(cfg))

	// Readiness: can this instance serve traffic right now? This one does check
	// the database, because an instance that cannot reach it should be removed
	// from the load balancer rather than restarted.
	r.Get("/ready", readyHandler(db))

	response.SetupHuma()

	title := opts.Title
	if title == "" {
		title = cfg.AppName
	}
	version := opts.Version
	if version == "" {
		version = cfg.AppVersion
	}

	humaConfig := huma.DefaultConfig(title, version)
	humaConfig.Info.Description = opts.Description
	humaConfig.Info.Contact = opts.Contact
	humaConfig.Tags = opts.Tags

	// Add Bearer JWT Security Scheme to OpenAPI docs
	humaConfig.Components = &huma.Components{
		SecuritySchemes: map[string]*huma.SecurityScheme{
			"bearerAuth": {
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
				Description:  "Enter token as: Bearer <your_jwt_token>",
			},
		},
	}

	// Disable $schema from showing up in OpenAPI docs/responses
	humaConfig.CreateHooks = []func(huma.Config) huma.Config{
		func(c huma.Config) huma.Config {
			c.SchemasPath = ""
			return c
		},
	}

	// Documentation gating.
	//
	// Blanking these paths removes the routes entirely rather than hiding them, so
	// there is no endpoint left to probe. API endpoints are untouched — only the
	// human-facing docs UI and the machine-readable spec disappear.
	//
	// The spec describes every route, parameter and schema in the system, which is
	// a convenient map for anyone looking for a way in. Publishing it is a choice,
	// and in production the default answer is no.
	if !cfg.DocsEnabled {
		humaConfig.DocsPath = ""
		humaConfig.OpenAPIPath = ""
	}

	api := humachi.New(r, humaConfig)

	if len(opts.TagGroups) > 0 {
		openAPI := api.OpenAPI()
		if openAPI.Extensions == nil {
			openAPI.Extensions = map[string]any{}
		}
		groups := make([]map[string]any, 0, len(opts.TagGroups))
		for _, g := range opts.TagGroups {
			groups = append(groups, map[string]any{"name": g.Name, "tags": g.Tags})
		}
		openAPI.Extensions["x-tagGroups"] = groups
	}

	// Registered before any route, so every operation inherits it.
	api.UseMiddleware(middleware.HumaAuthenticate(api, cfg.JWTSecret))

	return &App{cfg: cfg, log: log, db: db, router: r, api: api}, nil
}

// API returns the Huma API to register operations against.
func (a *App) API() huma.API { return a.api }

// Router returns the underlying chi router, for routes that are not part of the
// API — a server-rendered admin console, or static assets.
func (a *App) Router() *chi.Mux { return a.router }

// DB returns the database handle the App is using.
func (a *App) DB() *gorm.DB { return a.db }

// Config returns the configuration the App was built with.
func (a *App) Config() *config.Config { return a.cfg }

func (a *App) Run() error {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", a.cfg.ServerPort),
		Handler: a.router,

		// ReadHeaderTimeout is the defence against Slowloris: a client that opens a
		// connection and dribbles headers forever holds a goroutine hostage. It must
		// be set even when ReadTimeout is, because ReadTimeout alone permits a slow
		// header phase to consume the entire budget.
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       a.cfg.ReadTimeout,
		WriteTimeout:      a.cfg.WriteTimeout,
		IdleTimeout:       a.cfg.IdleTimeout,
		MaxHeaderBytes:    1 << 20, // 1 MiB
	}

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}

	actualAddr := ln.Addr().(*net.TCPAddr)
	a.log.Info(fmt.Sprintf("Server is listening on port %d", actualAddr.Port))
	if a.cfg.DocsEnabled {
		a.log.Info(fmt.Sprintf("API documentation at http://localhost:%d/docs", actualAddr.Port))
	} else {
		a.log.Info("API documentation is disabled (set DOCS_ENABLED=true to serve /docs and /openapi.json)")
	}

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			a.log.Fatal("Server failed to listen or serve", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	a.log.Warn("Received shutdown signal", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
	defer cancel()

	a.log.Info("Attempting to shut down gracefully...")
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	a.log.Info("Server shut down gracefully.")
	return nil
}

// healthHandler reports process liveness. It deliberately performs no dependency
// checks — see the route registration for why.
func healthHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "ok",
			"app_name": cfg.AppName,
			"version":  cfg.AppVersion,
			"time":     time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// readyHandler reports whether this instance can serve traffic, which means
// checking that the database is actually reachable.
func readyHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		sqlDB, err := db.DB()
		if err != nil || sqlDB.PingContext(ctx) != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status":   "unavailable",
				"database": "disconnected",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "ready",
			"database": "connected",
		})
	}
}

// writeJSON encodes a response body. Hand-formatting JSON with fmt.Sprintf works
// until a value contains a quote or a backslash, at which point it silently emits
// malformed JSON — encoding/json escapes correctly by construction.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
