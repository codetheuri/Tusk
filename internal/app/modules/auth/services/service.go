package services
import (
	//"context"
	authRepositories "github.com/codetheuri/todolist/internal/app/modules/auth/repositories"
	//"github.com/codetheuri/todolist/internal/app/modules/auth/models"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/validators"
)
	//define methods
type AuthService interface {
	// Define mthods here : eg
	//GetAuthByID(ctx context.Context, id int) (*models.Auth, error)
}

type authService struct {
	Repo authRepositories.AuthRepository
	validator *validators.Validator
	log logger.Logger
}

//service constructor
func NewAuthService(Repo authRepositories.AuthRepository, validator *validators.Validator, log logger.Logger) AuthService {
	return &authService{
		Repo: Repo,
		validator: validator,
		log: log,
	}
}

//methods
// func (s *authService) GetAuthByID(ctx context.Context, id uint) (*models.Auth, error) {
// 	s.log.Info("GetAuthByID service invoked")
// 	// Placeholder for actual logic
// 	return nil, nil
// }
