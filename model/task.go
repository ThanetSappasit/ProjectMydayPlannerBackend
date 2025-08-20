package model

import (
	"time"
)

type Tasks struct {
	TaskID      string    `firestore:"taskid,omitempty"`
	BoardID     string    `firestore:"boardid,omitempty"`
	TaskName    string    `firestore:"taskname,omitempty"`
	Description string    `firestore:"description,omitempty"`
	Status      string    `firestore:"status,omitempty"`   // "0" = pending, "1" = in progress, "2" = completed
	Priority    string    `firestore:"priority,omitempty"` // "1" = low, "2" = medium, "3" = high
	CreatedBy   string    `firestore:"createdby,omitempty"`
	UpdatedAt   time.Time `firestore:"updatedat,omitempty"`
}
