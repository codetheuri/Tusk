package router

import (
	"net/http"

	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/go-chi/chi"
)

func NewRouter(log logger.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("tusk is healthy"))
	})
	log.Info("Base HTTP router initialized. ")
	return r

}
