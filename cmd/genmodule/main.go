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
	ModuleName string
	CapitalizedModuleName string
	ProjectRoot string
}

const projectGoModulePath = "github.com/codetheuri/todolist"

const moduleGoTemplate = `package {{.ModuleName}}

import (
	{{.ModuleName}}Handlers "{{.ProjectRoot}}/internal/app/modules/{{.ModuleName}}/handlers"
	{{.ModuleName}}Repositories "{{.ProjectRoot}}/internal/app/modules/{{.ModuleName}}/repositories"
	{{.ModuleName}}Services "{{.ProjectRoot}}/internal/app/modules/{{.ModuleName}}/services"
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

// NewModule initializes  {{.CapitalizedModuleName}} module.
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
	// Register the routes for the {{.ModuleName}} module
	r.Route("/{{.ModuleName}}", func(r chi.Router) {
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Module {{.ModuleName}} is working!"))
	})
		//r.Post("/", m.Handlers.Create{{.CapitalizedModuleName}})
		//r.Get("/", m.Handlers.GetAll{{.CapitalizedModuleName}}s)
		
	})
}
`

const handlersGoTemplate = `package handlers
import (
   //	"context"
	//"net/http"
	{{.ModuleName}}Services "{{.ProjectRoot}}/internal/app/modules/{{.ModuleName}}/services"
	"{{.ProjectRoot}}/pkg/logger"
	//"{{.ProjectRoot}}/pkg/web"
	//"{{.ProjectRoot}}/internal/app/modules/{{.ModuleName}}/models"
)

type {{.CapitalizedModuleName}}Handler struct {
       {{.ModuleName}}Service {{.ModuleName}}Services.{{.CapitalizedModuleName}}Service
	   log logger.Logger
}

// constructor for {{.CapitalizedModuleName}}Handler
func New{{.CapitalizedModuleName}}Handler({{.ModuleName}}Service {{.ModuleName}}Services.{{.CapitalizedModuleName}}Service, log logger.Logger) *{{.CapitalizedModuleName}}Handler {
	return &{{.CapitalizedModuleName}}Handler{
		{{.ModuleName}}Service: {{.ModuleName}}Service,
		log: log,
	}
}

//example handler method

// func (h *{{.CapitalizedModuleName}}Handler) Get{{.CapitalizedModuleName}}ByID(w http.ResponseWriter, r *http.Request) {
// 	h.log.Info("Get{{.CapitalizedModuleName}}ByID handler invoked")
// 	// For example, to decode a request body into a model from this module:
// 	// var item models.{{.CapitalizedModuleName}}
// 	// if err := json.NewDecoder(r.Body).Decode(&item); err != nil { /* handle error */ }
// 	web.RespondJSON(w, http.StatusOK, map[string]string{"message": "Hello from {{.CapitalizedModuleName}} handler!"})
// }
`

const servicesGoTemplate = `package services
import (
	//"context"
	{{.ModuleName}}Repositories "{{.ProjectRoot}}/internal/app/modules/{{.ModuleName}}/repositories"
	//"{{.ProjectRoot}}/internal/app/modules/{{.ModuleName}}/models"
	"{{.ProjectRoot}}/pkg/logger"
	"{{.ProjectRoot}}/pkg/validators"
)
	//define methods
type {{.CapitalizedModuleName}}Service interface {
	// Define mthods here : eg
	//Get{{.CapitalizedModuleName}}ByID(ctx context.Context, id int) (*models.{{.CapitalizedModuleName}}, error)
}

type {{.ModuleName}}Service struct {
	Repo {{.ModuleName}}Repositories.{{.CapitalizedModuleName}}Repository
	validator *validators.Validator
	log logger.Logger
}

//service constructor
func New{{.CapitalizedModuleName}}Service(Repo {{.ModuleName}}Repositories.{{.CapitalizedModuleName}}Repository, validator *validators.Validator, log logger.Logger) {{.CapitalizedModuleName}}Service {
	return &{{.ModuleName}}Service{
		Repo: Repo,
		validator: validator,
		log: log,
	}
}

//methods
// func (s *{{.ModuleName}}Service) Get{{.CapitalizedModuleName}}ByID(ctx context.Context, id uint) (*models.{{.CapitalizedModuleName}}, error) {
// 	s.log.Info("Get{{.CapitalizedModuleName}}ByID service invoked")
// 	// Placeholder for actual logic
// 	return nil, nil
// }
`
const repositoriesGoTemplate = `package repositories
 import (
    //  "context"
	//  "{{.ProjectRoot}}/internal/app/modules/{{.ModuleName}}/models"
	  "{{.ProjectRoot}}/pkg/logger"
	  "gorm.io/gorm"
 )
//define interface
     type {{.CapitalizedModuleName}}Repository interface {
	// Example method:
	// Create{{.CapitalizedModuleName}}(ctx context.Context, {{.ModuleName}} *models.{{.CapitalizedModuleName}}) error
}
	type {{.ModuleName}}Repository struct {
	db *gorm.DB
	log logger.Logger 
	}
	// repo constructor
func New{{.CapitalizedModuleName}}Repository(db *gorm.DB, log logger.Logger) {{.CapitalizedModuleName}}Repository {	
	return &{{.ModuleName}}Repository{
		db: db,
		log: log,
	}
	}
	//repos methods
	//eg.
	// func (r *gorm{{.CapitalizedModuleName}}Repository) Create{{.CapitalizedModuleName}}(ctx context.Context, {{.ModuleName}} *models.{{.CapitalizedModuleName}}) error {
// 	r.log.Info("Create{{.CapitalizedModuleName}} repository ")
// 	// Placeholder for actual database logic
// 	return nil
// }

`
func main(){
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./cmd/genmodule <module_name> ")
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
		ModuleName: moduleName,
		CapitalizedModuleName: capitalizedModuleName,
		ProjectRoot: projectGoModulePath,
	}
	basePath := filepath.Join("internal", "app", "modules", moduleName)

	if _, err := os.Stat(basePath); !os.IsNotExist(err) {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Module '%s' already exists at %s . Overwrite? (Y/N): ", moduleName, basePath)

		response, _ := reader.ReadString('\n')
		response = strings.ToLower(strings.TrimSpace(response))

		if response != "y"{
			fmt.Println("Operation cancelled")
			os.Exit(0)
		}
		fmt.Printf("proceeding to overwrite module '%s'....\n", moduleName)
	}

	dirs := []string{
		basePath,
		filepath.Join(basePath, "handlers"),
		filepath.Join(basePath, "services"),
		filepath.Join(basePath, "repositories"),
		filepath.Join(basePath, "models"), 
	}

	for _, dir := range dirs{
		if err := os.MkdirAll(dir, 0755); err !=nil{
			fmt.Printf("Error creating directory %s: %v\n", dir, err)
			os.Exit(1)
		}
	}
	filesToGenerate := []struct {
		path string
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
	fmt.Printf("- **Crucially, create your model file(s) in 'internal/app/modules/%s/models/'.**\n",  moduleName)
	fmt.Println("- Adjust generated boilerplate code (e.g., imports, methods, model usage).")
	fmt.Println("- Run 'go mod tidy' after adding to app.go.")

}

