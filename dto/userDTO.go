package dto

type UserResponse struct {
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Profile   string `json:"profile"`
	Role      string `json:"role"`
	IsVerify  string `json:"is_verify"`
	IsActive  string `json:"is_active"`
	CreatedAt string `json:"created_at"`
}

type EmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}
type SearchEmailRequest struct {
	Email string `json:"email"`
}

type UpdateProfileRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Profile  string `json:"profile"`
}
