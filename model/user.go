package model

import "time"

type User struct {
	UserID    string    `firestore:"userid,omitempty"`
	Name      string    `firestore:"name,omitempty"`
	Email     string    `firestore:"email,omitempty"`
	Password  string    `firestore:"password,omitempty"`
	Profile   string    `firestore:"profile,omitempty"`
	Role      string    `firestore:"role,omitempty"`   // "user" หรือ "admin"
	Verify    string    `firestore:"verify,omitempty"` // "0" = false, "1" = true
	Active    string    `firestore:"active,omitempty"` // "0" inactive, "1" active, "2" banned
	CreatedAt time.Time `firestore:"createdat,omitempty"`
}
