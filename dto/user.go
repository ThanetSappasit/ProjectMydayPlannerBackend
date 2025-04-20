package dto

type GetUserByEmail struct {
	Email *string `json:"email"`
}

type CreateAccUserRequest struct {
	Name           string `json:"name" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	HashedPassword string `json:"hashed_password"`
}

type DeleteUserRequest struct {
	Email string `json:"email"`
}

type UpdateProfileRequest struct {
	Email       string `json:"email"`
	ProfileData struct {
		Name           string `json:"name"`
		HashedPassword string `json:"hashed_password"`
		Profile        string `json:"profile"`
	} `json:"profileData"`
}

type EmailOTPRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Email          string `json:"email"`
	HashedPassword string `json:"hashed_password"`
}
