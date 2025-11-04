package docs

import (
	router "github.com/codetheuri/todolist/internal/app/routers"
	"github.com/codetheuri/todolist/pkg/docs"
	"github.com/codetheuri/todolist/pkg/logger"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type Module struct {
	DocService docs.DocService // The shared service instance
	log        logger.Logger
}

func NewModule(log logger.Logger, handlerMap map[string]interface{}) *Module {
	docService := docs.NewDocService(log, handlerMap)
	return &Module{
		DocService: docService,
		log:        log,
	}
}

func (m *Module) RegisterRoutes(r router.Router) {
	m.log.Info("Mounting Docs API and Swagger UI...")
	r.Get("/docs/{module}/doc.json", m.DocService.ServeDocJSON)
	r.Group(func(r router.Router) {
		// Set a default doc URL (e.g., Auth module) for initial load.
		r.Mount("/docs", httpSwagger.WrapHandler(
			httpSwagger.URL("/docs/auth/doc.json"),
			httpSwagger.InstanceName("docs"), // Unique instance name
		))
	})
	swaggerHandler := httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // URL pointing to API definition
	)

	// Mount the swagger handler
	r.Handle("/swagger/*", swaggerHandler)

	m.log.Info("Docs module registered successfully.")

}
