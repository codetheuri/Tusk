package repositories

import (
	"context"

	"github.com/codetheuri/todolist/internal/app/auth/models"
	"github.com/codetheuri/todolist/pkg/logger"
	"gorm.io/gorm"
)

// user interface
type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id uint) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, id uint) error
	RestoreUser(ctx context.Context, id uint) error
}
type userRepository struct {
	db  *gorm.DB
	log logger.Logger
}

// repo constructor
func NewUserRepository(db *gorm.DB, log logger.Logger) UserRepository {
	return &userRepository{
		db:  db,
		log: log,
	}
}

func (r *userRepository) CreateUser(ctx context.Context, user *models.User) error {
	r.log.Info("Repository: Creating user in DB", "email", user.Email)
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	r.log.Info("GetUserByEmail repository")
	var user models.User
	 err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	return &user, err
}

func (r *userRepository) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	r.log.Info("GetUserByID repository")
	var user models.User
	 err := r.db.WithContext(ctx).First(&user, id).Error
	return &user, err
}
func (r *userRepository) UpdateUser(ctx context.Context, user *models.User) error {
	r.log.Info("UpdateUser repository")
	return r.db.WithContext(ctx).Save(user).Error
}
func (r *userRepository) DeleteUser(ctx context.Context, id uint) error {
	r.log.Info("DeleteUser repository")
	return r.db.WithContext(ctx).Delete(&models.User{}, id).Error
}
func (r *userRepository) RestoreUser(ctx context.Context, id uint) error {
	r.log.Info("RestoreUser repository")
	return r.db.WithContext(ctx).Unscoped().Model(&models.User{}).Where("id = ?", id).Update("deleted_at", nil).Error
}
