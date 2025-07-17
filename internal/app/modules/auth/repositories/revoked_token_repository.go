package repositories

import (
	"context"
	"time"

	"github.com/codetheuri/todolist/internal/app/modules/auth/models"
	"github.com/codetheuri/todolist/pkg/logger"
	"gorm.io/gorm"
)

type RevokedTokenRepository interface {
	SaveRevokedToken(ctx context.Context, revokedToken *models.RevokedToken) error
	IsTokenRevoked(ctx context.Context, gti string) (bool, error)
	DeleteExpiredRevokedTokens(ctx context.Context, currentTime time.Time) error
}

type revokedTokenRepository struct {
	db  *gorm.DB
	log logger.Logger
}

func NewRevokedTokenRepository(db *gorm.DB, log logger.Logger) RevokedTokenRepository {
	return &revokedTokenRepository{
		db:  db,
		log: log,
	}
}
func (r *revokedTokenRepository) SaveRevokedToken(ctx context.Context, revokedToken *models.RevokedToken) error {
	r.log.Info("Saving revoked token JTI", "jti", revokedToken.JTI, "expires_at", revokedToken.ExpiresAt)
	if err := r.db.WithContext(ctx).Create(revokedToken).Error; err != nil {
		r.log.Error("Failed to save revoked token", err, "jti", revokedToken.JTI)
		return err
	}
	return nil
}

func (r *revokedTokenRepository) IsTokenRevoked(ctx context.Context, jti string) (bool, error) {
	r.log.Debug("Checking if token is revoked", "jti", jti)
	var count int64
	// Check if a token with this JTI exists and is not expired (ExpiresAt in the future)
	err := r.db.WithContext(ctx).Model(&models.RevokedToken{}).
		Where("jti = ?", jti).
		Where("expires_at > ?", time.Now()).
		Count(&count).Error
	if err != nil {
		r.log.Error("Failed to check if token is revoked", err, "jti", jti)
		return false, err
	}
	return count > 0, nil
}

func (r *revokedTokenRepository) DeleteExpiredRevokedTokens(ctx context.Context, currentTime time.Time) error {
	r.log.Info("Deleting expired revoked tokens up to", "current_time", currentTime)

	if err := r.db.WithContext(ctx).Unscoped().Where("expires_at <= ?", currentTime).Delete(&models.RevokedToken{}).Error; err != nil {
		r.log.Error("Failed to delete expired revoked tokens", err)
		return err
	}
	return nil
}
