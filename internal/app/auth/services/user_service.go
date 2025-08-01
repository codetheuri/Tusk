package services

import (
	"errors"

	"github.com/codetheuri/todolist/internal/app/auth/models"
	"github.com/codetheuri/todolist/internal/app/auth/repositories"
	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/validators"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"
	"gorm.io/gorm"
)

type UserService interface {
	RegisterUser(ctx context.Context, email, password, role string) (*models.User, error)
	GetUserByID(ctx context.Context, id uint) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUsers(ctx context.Context, offset, limit int) ([]*models.User, int64, error)
	UpdateUser(ctx context.Context, user *models.User) error
	ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error
	DeleteUser(ctx context.Context, id uint) error
	RestoreUser(ctx context.Context, id uint) error
}

type userService struct {
	userRepo  repositories.UserRepository
	log       logger.Logger
	validator *validators.Validator
}

func NewUserService(userRepo repositories.UserRepository, validator *validators.Validator, log logger.Logger) UserService {
	return &userService{
		userRepo:  userRepo,
		log:       log,
		validator: validator,
	}
}

func (s *userService) RegisterUser(ctx context.Context, email, password, role string) (*models.User, error) {
	s.log.Info("Registering new user", "email", email)

	newUser := models.User{
		Email:    email,
		Password: password,
		Role:     role,
	}
	validationErros := s.validator.Struct(newUser)
	if validationErros != nil {
		s.log.Warn("Validation failed for user registration", "err", validationErros)
		return nil, appErrors.ValidationError("validation failed for user registration", nil, validationErros)
	}
	existingUser, err := s.userRepo.GetUserByEmail(ctx, email)

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.log.Error("Failed to check for existing user", err, "email", email)
		return nil, appErrors.DatabaseError("failed to check for existing user", err)
	}
	if existingUser != nil {
		s.log.Warn("User with this email already exists", "email", email)
		// return nil, appErrors.AuthError("User with this email already exists", nil)
		return nil, appErrors.ConflictError("user with this email already exists", nil)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("Failed to hash password", err, "email", email)
		return nil, appErrors.InternalServerError("failed to hash password", err)
	}
	newUser.Password = string(hashedPassword)
	if err := s.userRepo.CreateUser(ctx, &newUser); err != nil {
		s.log.Error("Failed to create user in database", err, "email", email)
		return nil, appErrors.DatabaseError("failed to create user in database", err)
	}
	s.log.Info("User registered successfully", "userID", newUser.ID, "email", newUser.Email)
	return &newUser, nil
}

func (s *userService) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	s.log.Debug("Getting user by ID in service", "id", id)
	user, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.log.Warn("User not found by ID", "id", id)
			return nil, appErrors.NotFoundError("user not found", err)
		}
		s.log.Error("Failed to get user by ID from repository", err, "id", id)
		return nil, appErrors.DatabaseError("failed to retrieve user", err)
	}
	return user, nil
}
func (s *userService) GetUserByEmail(ctx context.Context, email string) (*models.User, error) { // <--- Added error return
	s.log.Debug("Getting user by email in service", "email", email)
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.log.Warn("User not found by email", "email", email)
			return nil, appErrors.NotFoundError("user not found", err)
		}
		s.log.Error("Failed to get user by email from repository", err, "email", email)
		return nil, appErrors.DatabaseError("failed to retrieve user", err)
	}
	return user, nil
}

func (s *userService) UpdateUser(ctx context.Context, user *models.User) error {
	s.log.Info("Updating user in service", "id", user.ID, "email", user.Email)
	//  Add validation for the user struct before updating
	validationErrors := s.validator.Struct(user)
	if validationErrors != nil {
		s.log.Warn("Validation failed for user update", "err", validationErrors)
		return appErrors.ValidationError("validation failed for user update", nil, validationErrors)
	}
	_, err := s.userRepo.GetUserByID(ctx, user.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.log.Warn("Service: User to update not found", "id", user.ID)
			return appErrors.NotFoundError("user to update not found", err)
		}
		s.log.Error("Service: Failed to retrieve user for update check", err, "id", user.ID)
		return appErrors.DatabaseError("failed to retrieve user for update", err)
	}
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		s.log.Error("Failed to update user in database", err, "id", user.ID)
		return appErrors.DatabaseError("failed to update user", err)
	}
	return nil
}
func (s *userService) ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	s.log.Info("Changing password for user", "userID", userID)

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.log.Warn("Service: User not found for password change", "userID", userID)
			return appErrors.NotFoundError("user not found", err)
		}
		s.log.Error("Service: Failed to retrieve user for password change from repository", err, "userID", userID)
		return appErrors.DatabaseError("failed to retrieve user for password change", err)

	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		s.log.Warn("Old password mismatch", "err", err, "userID", userID)
		return appErrors.AuthError("invalid old password", nil) // AuthError for credential mismatch
	}
	//not yet done

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("Failed to hash new password", err, "userID", userID)
		return appErrors.InternalServerError("failed to hash new password", err)
	}
	user.Password = string(hashedPassword)

	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		s.log.Error("Failed to update user password in database", err, "userID", userID)
		return appErrors.DatabaseError("failed to update user password", err)
	}

	//not yet done
	s.log.Info("Password changed successfully", "userID", userID)
	return nil
}

func (s *userService) DeleteUser(ctx context.Context, id uint) error {
	s.log.Info("Deleting user (soft) in service", "id", id)

	_, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {

		return err
	}
	//not done

	if err := s.userRepo.DeleteUser(ctx, id); err != nil {
		s.log.Error("Failed to soft delete user in database", err, "id", id)
		return appErrors.DatabaseError("failed to delete user", err)
	}
	s.log.Info("User soft-deleted successfully", "id", id)
	return nil
}

func (s *userService) RestoreUser(ctx context.Context, id uint) error {
	s.log.Info("Restoring user ", "id", id)
	_, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {

		return err
	}
	if err := s.userRepo.RestoreUser(ctx, id); err != nil {
		s.log.Error("Failed to restore user in database", err, "id", id)
		return appErrors.DatabaseError("failed to restore user", err)
	}

	return nil
}
func (s *userService) GetUsers(ctx context.Context, offset, limit int) ([]*models.User, int64, error) {
	s.log.Info("Service: Getting users with pagination params", "offset", offset, "limit", limit)
	users, totalCount, err := s.userRepo.GetUsers(ctx, offset, limit)
	if err != nil {
		s.log.Error("Service: Failed to get users from repository", err)
		return nil, 0, appErrors.DatabaseError("failed to retrieve users", err)
	}
	s.log.Info("Service: Successfully retrieved paginated users (models)", "count", len(users), "total", totalCount)
	return users, totalCount, nil
}
