package dto

import "time"

type AuthResponse struct {
	UserID    uint   `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Token     string `json:"token"` 
	// CreatedAt string `json:"created_at"`     
	ExpiresAt int64  `json:"expires_at"`
}
type GetUserProfileResponse struct {
    UserID uint   `json:"user_id"`
    Email  string `json:"email"`
	CreatedAt *time.Time `json:"created_at"`
    // Role   string `json:"role"`
}
type SuccessResponse struct {
    Message string `json:"message"`
}