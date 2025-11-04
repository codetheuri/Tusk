package docs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/codetheuri/todolist/pkg/logger"
	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/web"
	"github.com/go-chi/chi"
)

type DocService interface {
	ServeDocJSON(w http.ResponseWriter, r *http.Request)
}

type docService struct {
	ModuleHandlers map[string]interface{}
	log            logger.Logger
}

func NewDocService(log logger.Logger, moduleHandlers map[string]interface{}) DocService {
	return &docService{
		ModuleHandlers: moduleHandlers,
		log:            log,
	}
}
func (s *docService) ServeDocJSON(w http.ResponseWriter, r *http.Request) {
	// Use Chi's context to get the path parameter, even though this is a generic pkg function.
	// This relies on the router adapter (in the docs module) to set the path parameter correctly.
	moduleName := chi.URLParam(r, "module")

	if moduleName == "" {
		web.RespondError(w, appErrors.NotFoundError("Module name missing in path", nil), http.StatusNotFound)
		return
	}

	// 1. Check if the module handler exists
	_, found := s.ModuleHandlers[moduleName]
	if !found {
		s.log.Warn("Doc request for unknown module", "module", moduleName)
		web.RespondError(w, appErrors.NotFoundError(fmt.Sprintf("Documentation for module '%s' not found.", moduleName), nil), http.StatusNotFound)
		return
	}

	s.log.Info("Serving dynamic doc spec", "module", moduleName)

	// --- CRITICAL: PLACEHOLDER FOR SPEC GENERATION LOGIC ---
	// In a real Code-First app, a reflection library would inspect the handlerInstance
	// (which we retrieve from s.ModuleHandlers[moduleName]) and build the paths array.

	// For now, we serve a valid, minimal, dynamic JSON spec:
	spec := map[string]interface{}{
		"swagger": "2.0",
		"info": map[string]string{
			"title":       fmt.Sprintf("Tusk API: %s Module", strings.Title(moduleName)),
			"version":     "1.0.0",
			"description": fmt.Sprintf("API specification for the %s module, generated dynamically.", strings.Title(moduleName)),
		},
		"host":     "localhost:8080",
		"basePath": fmt.Sprintf("/api/v1/%s", moduleName),
		"paths": map[string]interface{}{
			// Example path that *must* be included for Swagger UI testing
			fmt.Sprintf("/api/v1/%s/test", moduleName): map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Dynamic Test Route",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Success"},
					},
				},
			},
		},
		"securityDefinitions": map[string]interface{}{
			"BearerAuth": map[string]interface{}{
				"type":        "apiKey",
				"name":        "Authorization",
				"in":          "header",
				"description": "JWT Auth",
			},
		},
	}

	// Send the dynamically constructed JSON spec
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(spec); err != nil {
		s.log.Error("Failed to encode doc spec", err)
		web.RespondError(w, appErrors.InternalServerError("Failed to encode spec", err), http.StatusInternalServerError)
	}
}
