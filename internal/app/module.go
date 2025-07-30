package app

import "github.com/go-chi/chi"


type Module interface {
	RegisterRoutes(r chi.Router	)
}