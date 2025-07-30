package repositories
 import (
    //  "context"
	//  "github.com/codetheuri/todolist/internal/app/modules/auth/models"
	  "github.com/codetheuri/todolist/pkg/logger"
	  "gorm.io/gorm"
 )

	type AuthRepository struct {
	 UserRepo         UserRepository
    RevokedTokenRepo RevokedTokenRepository
	}
	// repo constructor
func NewAuthRepository(db *gorm.DB, log logger.Logger) *AuthRepository {	
	return &AuthRepository{
		UserRepo: 	   NewUserRepository(db, log),
		RevokedTokenRepo: NewRevokedTokenRepository(db, log),
	}
	}

