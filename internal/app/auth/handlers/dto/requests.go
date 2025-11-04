package dto

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email" unique:"users,email"`
	Password string `json:"password" validate:"required,min=8"`
	Role     string `json:"role" validate:"required,oneof=user admin"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type ChangePasswordRequest struct {
    OldPassword string `json:"old_password" validate:"required"`
    NewPassword string `json:"new_password" validate:"required,min=8"`
}
type GetUsersRequest struct {
    Page  int `json:"-" query:"page" validate:"omitempty,min=1"`
    Limit int `json:"-" query:"limit" validate:"omitempty,min=1,max=100"`
}