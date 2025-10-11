package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type ModuleData struct {
	ModuleName            string
	CapitalizedModuleName string
	ProjectRoot           string
}

const projectGoModulePath = "github.com/codetheuri/todolist"

const moduleGoTemplate = `package {{.ModuleName}}

import (
	{{.ModuleName}}Handlers "{{.ProjectRoot}}/internal/app/{{.ModuleName}}/handlers"
	{{.ModuleName}}Repositories "{{.ProjectRoot}}/internal/app/{{.ModuleName}}/repositories"
	{{.ModuleName}}Services "{{.ProjectRoot}}/internal/app/{{.ModuleName}}/services"
	"{{.ProjectRoot}}/pkg/logger"
	"{{.ProjectRoot}}/pkg/validators"
	"github.com/go-chi/chi"
	"gorm.io/gorm"
	"net/http"
)

// Module represents the {{.CapitalizedModuleName}} module.
type Module struct {
	Handlers *{{.ModuleName}}Handlers.{{.CapitalizedModuleName}}Handler
}

// NewModule initializes {{.CapitalizedModuleName}} module.
func NewModule(db *gorm.DB, log logger.Logger, validator *validators.Validator) *Module {
	repo := {{.ModuleName}}Repositories.New{{.CapitalizedModuleName}}Repository(db, log)
	service := {{.ModuleName}}Services.New{{.CapitalizedModuleName}}Service(repo, validator, log)
	handler := {{.ModuleName}}Handlers.New{{.CapitalizedModuleName}}Handler(service, log)

	return &Module{
		Handlers: handler,
	}
}

// RegisterRoutes registers the routes for the {{.CapitalizedModuleName}} module.
func (m *Module) RegisterRoutes(r chi.Router) {
	r.Route("/{{.ModuleName}}", func(r chi.Router) {
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Module {{.ModuleName}} is working!"))
		})
	})
}
`

const handlersGoTemplate = `package handlers
import (
	{{.ModuleName}}Services "{{.ProjectRoot}}/internal/app/{{.ModuleName}}/services"
	"{{.ProjectRoot}}/pkg/logger"
)

type {{.CapitalizedModuleName}}Handler struct {
	{{.ModuleName}}Service {{.ModuleName}}Services.{{.CapitalizedModuleName}}Service
	log logger.Logger
}

// New{{.CapitalizedModuleName}}Handler constructor for {{.CapitalizedModuleName}}Handler
func New{{.CapitalizedModuleName}}Handler({{.ModuleName}}Service {{.ModuleName}}Services.{{.CapitalizedModuleName}}Service, log logger.Logger) *{{.CapitalizedModuleName}}Handler {
	return &{{.CapitalizedModuleName}}Handler{
		{{.ModuleName}}Service: {{.ModuleName}}Service,
		log: log,
	}
}
`

const servicesGoTemplate = `package services
import (
	{{.ModuleName}}Repositories "{{.ProjectRoot}}/internal/app/{{.ModuleName}}/repositories"
	"{{.ProjectRoot}}/pkg/logger"
	"{{.ProjectRoot}}/pkg/validators"
)

type {{.CapitalizedModuleName}}Service interface {
}

type {{.ModuleName}}Service struct {
	Repo      {{.ModuleName}}Repositories.{{.CapitalizedModuleName}}Repository
	validator *validators.Validator
	log       logger.Logger
}

// New{{.CapitalizedModuleName}}Service service constructor
func New{{.CapitalizedModuleName}}Service(Repo {{.ModuleName}}Repositories.{{.CapitalizedModuleName}}Repository, validator *validators.Validator, log logger.Logger) {{.CapitalizedModuleName}}Service {
	return &{{.ModuleName}}Service{
		Repo:      Repo,
		validator: validator,
		log:       log,
	}
}
`

const repositoriesGoTemplate = `package repositories
import (
	"{{.ProjectRoot}}/pkg/logger"
	"gorm.io/gorm"
)

type {{.CapitalizedModuleName}}Repository interface {
}

type {{.ModuleName}}Repository struct {
	db  *gorm.DB
	log logger.Logger
}

// New{{.CapitalizedModuleName}}Repository repo constructor
func New{{.CapitalizedModuleName}}Repository(db *gorm.DB, log logger.Logger) {{.CapitalizedModuleName}}Repository {
	return &{{.ModuleName}}Repository{
		db:  db,
		log: log,
	}
}
`

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./cmd/genmodule <module_name>")
		os.Exit(1)
	}

	moduleName := strings.ToLower(os.Args[1])
	capitalizedModuleName := strings.ToUpper(moduleName[:1]) + moduleName[1:]

	if strings.Contains(moduleName, "-") || strings.Contains(moduleName, "_") {
		fmt.Println("Module name cannot contain '-' or '_' characters. Please use a valid module name.")
		os.Exit(1)
	}
	if moduleName == "main" || moduleName == "cmd" || moduleName == "internal" || moduleName == "pkg" {
		fmt.Printf("Module name '%s' is reserved. Please choose a different name.\n", moduleName)
		os.Exit(1)
	}

	moduleData := ModuleData{
		ModuleName:            moduleName,
		CapitalizedModuleName: capitalizedModuleName,
		ProjectRoot:           projectGoModulePath,
	}
	basePath := filepath.Join("internal", "app", moduleName)

	if _, err := os.Stat(basePath); !os.IsNotExist(err) {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Module '%s' already exists at %s. Overwrite? (Y/N): ", moduleName, basePath)

		response, _ := reader.ReadString('\n')
		response = strings.ToLower(strings.TrimSpace(response))

		if response != "y" {
			fmt.Println("Operation cancelled")
			os.Exit(0)
		}
		fmt.Printf("Proceeding to overwrite module '%s'....\n", moduleName)
	}

	dirs := []string{
		basePath,
		filepath.Join(basePath, "handlers"),
		filepath.Join(basePath, "handlers", "dto"), // Added dto folder
		filepath.Join(basePath, "services"),
		filepath.Join(basePath, "repositories"),
		filepath.Join(basePath, "models"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("Error creating directory %s: %v\n", dir, err)
			os.Exit(1)
		}
	}

	filesToGenerate := []struct {
		path     string
		template string
	}{
		{filepath.Join(basePath, "module.go"), moduleGoTemplate},
		{filepath.Join(basePath, "handlers", "handler.go"), handlersGoTemplate},
		{filepath.Join(basePath, "services", "service.go"), servicesGoTemplate},
		{filepath.Join(basePath, "repositories", "repository.go"), repositoriesGoTemplate},
	}

	for _, file := range filesToGenerate {
		tmpl, err := template.New(filepath.Base(file.path)).Parse(file.template)
		if err != nil {
			fmt.Printf("Error parsing template for %s: %v\n", file.path, err)
			os.Exit(1)
		}

		f, err := os.Create(file.path)
		if err != nil {
			fmt.Printf("Error creating file %s: %v\n", file.path, err)
			os.Exit(1)
		}
		defer f.Close()

		if err := tmpl.Execute(f, moduleData); err != nil {
			fmt.Printf("Error executing template for %s: %v\n", file.path, err)
			os.Exit(1)
		}
		fmt.Printf("Generated: %s\n", file.path)
	}

	fmt.Printf("\nModule '%s' generated successfully. Remember to:\n", moduleName)
	fmt.Println("- Add it to 'internal/bootstrap/app.go' in the appModules slice.")
	fmt.Printf("- **Crucially, create your model file(s) in 'internal/app/%s/models/'.**\n", moduleName)
	fmt.Println("- Adjust generated boilerplate code (e.g., imports, methods, model usage).")
	fmt.Println("- Run 'go mod tidy' after adding to app.go.")
}