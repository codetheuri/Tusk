package dto

type AuthResponse struct {
	UserID    uint   `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Token     string `json:"token"`      
	ExpiresAt int64  `json:"expires_at"`
}
type GetUserProfileResponse struct {
    UserID uint   `json:"user_id"`
    Email  string `json:"email"`
    Role   string `json:"role"`
}
type SuccessResponse struct {
    Message string `json:"message"`
}