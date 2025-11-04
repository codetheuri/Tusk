package router

import (
	"net/http"

	_ "github.com/codetheuri/todolist/docs"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type Router interface {
	http.Handler
	Get(pattern string, h http.HandlerFunc)
	Post(pattern string, h http.HandlerFunc)
	Put(pattern string, h http.HandlerFunc)
	Patch(pattern string, h http.HandlerFunc)
	Delete(pattern string, h http.HandlerFunc)
	Group(fn func(r Router))
	Route(pattern string, fn func(r Router))
	Use(middlewares ...func(http.Handler) http.Handler)
	Mount(pattern string, h http.Handler)
}

type chiRouter struct {
	r chi.Router
}

// Implement all the interface methods by calling the underlying chi router.
func (cr *chiRouter) Get(pattern string, h http.HandlerFunc)    { cr.r.Get(pattern, h) }
func (cr *chiRouter) Post(pattern string, h http.HandlerFunc)   { cr.r.Post(pattern, h) }
func (cr *chiRouter) Put(pattern string, h http.HandlerFunc)    { cr.r.Put(pattern, h) }
func (cr *chiRouter) Patch(pattern string, h http.HandlerFunc)  { cr.r.Patch(pattern, h) }
func (cr *chiRouter) Delete(pattern string, h http.HandlerFunc) { cr.r.Delete(pattern, h) }
func (cr *chiRouter) Use(middlewares ...func(http.Handler) http.Handler) {
	cr.r.Use(middlewares...)
}
func (cr *chiRouter) Mount(pattern string, h http.Handler) {
	cr.r.Mount(pattern, h)
}

func (cr *chiRouter) Group(fn func(r Router)) {
	cr.r.Group(func(subRouter chi.Router) {
		fn(&chiRouter{r: subRouter})
	})
}

func (cr *chiRouter) Route(pattern string, fn func(r Router)) {
	cr.r.Route(pattern, func(subRouter chi.Router) {
		fn(&chiRouter{r: subRouter})
	})
}

func (cr *chiRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cr.r.ServeHTTP(w, r)
}

func NewRouter(log logger.Logger) Router {
	// r := chi.NewRouter()
	r := chi.NewMux()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("tusk is healthy"))
	})

	swaggerHandler := httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // URL pointing to API definition
	)

	// Mount the swagger handler
	r.Handle("/swagger/*", swaggerHandler)

	log.Info("Base HTTP router initialized. ")
	return &chiRouter{r: r}

}
