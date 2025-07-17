package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/codetheuri/todolist/internal/app/modules/auth/models"
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
	r.log.Info("CreateUser repository")
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		r.log.Error("Failed to create user", err)
		return err
	}
	return nil
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	r.log.Info("GetUserByEmail repository")
	var user models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		r.log.Error("Failed to get user by email", err)
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	r.log.Info("GetUserByID repository")
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		r.log.Error("Failed to get user by ID", err)
		return nil, err
	}
	return &user, nil
}
func (r *userRepository) UpdateUser(ctx context.Context, user *models.User) error {
	r.log.Info("UpdateUser repository")
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		r.log.Error("Failed to update user", err, "id", user.ID)
		return err
	}
	return nil
}
func (r *userRepository) DeleteUser(ctx context.Context, id uint) error {
	r.log.Info("DeleteUser repository")
	if err := r.db.WithContext(ctx).Delete(&models.User{}, id).Error; err != nil {
		r.log.Error("Failed to delete user", err, "id", id)
		return err
	}
	return nil
}
func (r *userRepository) RestoreUser(ctx context.Context, id uint) error {
	r.log.Info("RestoreUser repository")
	var user models.User
	if err := r.db.WithContext(ctx).Unscoped().First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("User not found for restore ", err, "id", id)
			
			return err
		}
		r.log.Error("Failed to find user for restore", err, "id", id)
		return err
	}


	if user.DeletedAt.Valid { 
		user.DeletedAt.Valid = false
		user.DeletedAt.Time = time.Time{} 

		if err := r.db.WithContext(ctx).Save(&user).Error; err != nil {
			r.log.Error("Failed to restore user", err, "id", id)
			return err
		}
	} else {
		
		r.log.Warn("Attempted to restore a user failed", "id", id)
		
	}

	return nil
}
