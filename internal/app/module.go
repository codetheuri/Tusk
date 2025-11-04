package app

// import "github.com/go-chi/chi"
import "github.com/codetheuri/todolist/internal/app/routers"


type Module interface {
	RegisterRoutes(r router.Router)
}