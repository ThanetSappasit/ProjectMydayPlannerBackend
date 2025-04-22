package dto

type DisableUserRequest struct {
	Email string `json:"email"`
}

type AdminRequest struct {
	Email          string `json:"email"`
	HashedPassword string `json:"hashed_password"`
}
