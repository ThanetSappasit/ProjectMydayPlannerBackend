package model

import "time"

type Board struct {
	BoardID   string    `firestore:"boardid,omitempty"`
	BoardName string    `firestore:"boardname,omitempty"`
	BoardType string    `firestore:"type,omitempty"`
	DeepLink  string    `firestore:"link,omitempty"`
	CreatedAt time.Time `firestore:"createdat,omitempty"`
	CreatedBy string    `firestore:"createdby,omitempty"`
	UpdatedAt time.Time `firestore:"updatedat,omitempty"`
}
