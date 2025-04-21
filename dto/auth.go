package dto

type EmailOTPRequest struct {
	Email string `json:"email"`
}

type VerifyRequest struct {
	Email     string `json:"email" binding:"required"`
	Reference string `json:"ref" binding:"required"`
	OTP       string `json:"otp" binding:"required"`
}

type SigninRequest struct {
	Email          string `json:"email" binding:"required"`
	HashedPassword string `json:"hashed_password"`
}

type SignoutRequest struct {
	Email string `json:"email" binding:"required"`
}

type GoogleSignInRequest struct {
	Email   string `json:"email" binding:"required"`
	Name    string `json:"name"`
	Profile string `json:"profile"`
}
