package repositories

import (
	"github.com/codetheuri/todolist/pkg/logger"
	"gorm.io/gorm"
)

// define interface
type AuthRepository interface {
	// Example method:
	// CreateAuth(ctx context.Context, auth *models.Auth) error
}
type gormAuthRepository struct {
	db  *gorm.DB
	log logger.Logger
}

//repos methods
//eg.
// func (r *gormAuthRepository) CreateAuth(ctx context.Context, auth *models.Auth) error {
// 	r.log.Info("CreateAuth repository ")
// 	// Placeholder for actual database logic
// 	return nil
// }
