package model

import "time"

type Board struct {
	BoardID   int       `firestore:"boardid,omitempty"`
	BoardName string    `firestore:"boardname,omitempty"`
	CreatedAt time.Time `firestore:"createdat,omitempty"`
	CreatedBy string    `firestore:"createdby,omitempty"`
	UpdatedAt time.Time `firestore:"updatedat,omitempty"`
}
