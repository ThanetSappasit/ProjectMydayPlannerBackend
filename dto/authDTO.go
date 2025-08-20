package dto

type SigninRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type SignupRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name" binding:"required"`
}

type IdentityOTPRequest struct {
	Email string `json:"email"`
}

type ResetpasswordOTPRequest struct {
	Email string `json:"email"`
}

type SendemailRequest struct {
	Email     string `json:"email"`
	Reference string `json:"reference"`
	Record    string `json:"record"`
}

type ResendOTPRequest struct {
	Email  string `json:"email"`
	Record string `json:"record"`
}

type VerifyRequest struct {
	Email     string `json:"email" binding:"required"`
	Reference string `json:"ref" binding:"required"`
	OTP       string `json:"otp" binding:"required"`
	Record    string `json:"record"`
}

type CaptchaRequest struct {
	Token  string `json:"token" validate:"required"`
	Action string `json:"action" validate:"required"`
}

type AssessmentResult struct {
	Score   float32
	Action  string
	Reasons []string
}

type ResetPasswordRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
