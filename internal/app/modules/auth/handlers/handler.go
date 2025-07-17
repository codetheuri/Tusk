package handlers
import (
   //	"context"
	//"net/http"
	authServices "github.com/codetheuri/todolist/internal/app/modules/auth/services"
	"github.com/codetheuri/todolist/pkg/logger"
	//"github.com/codetheuri/todolist/pkg/web"
	//"github.com/codetheuri/todolist/internal/app/modules/auth/models"
)

type AuthHandler struct {
       authService authServices.AuthService
	   log logger.Logger
}

// constructor for AuthHandler
func NewAuthHandler(authService authServices.AuthService, log logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		log: log,
	}
}

//example handler method

// func (h *AuthHandler) GetAuthByID(w http.ResponseWriter, r *http.Request) {
// 	h.log.Info("GetAuthByID handler invoked")
// 	// For example, to decode a request body into a model from this module:
// 	// var item models.Auth
// 	// if err := json.NewDecoder(r.Body).Decode(&item); err != nil { /* handle error */ }
// 	web.RespondJSON(w, http.StatusOK, map[string]string{"message": "Hello from Auth handler!"})
// }
