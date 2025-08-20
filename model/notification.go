package model

import (
	"time"
)

type Notification struct {
	NotificationID   string     `firestore:"notificationid,omitempty"`
	TaskID           string     `firestore:"taskid,omitempty"`
	DueDate          *time.Time `firestore:"duedate,omitempty"`
	BeforeDueDate    *time.Time `firestore:"beforeduedate,omitempty"`
	RecurringPattern *string    `firestore:"pattern,omitempty"`
	Snooze           *time.Time `firestore:"snooze,omitempty"`
	Send             string     `firestore:"send,omitempty"`
	Updatedat        time.Time  `firestore:"updatedat,omitempty"`
}
