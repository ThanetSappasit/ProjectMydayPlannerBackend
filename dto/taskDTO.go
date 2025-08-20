package dto

type CreateTaskRequest struct {
	BoardID     string    `json:"boardid" binding:"required"`
	TaskName    string    `json:"taskname" binding:"required"`
	Description string    `json:"description"`
	Status      string    `json:"status" binding:"required"`
	Reminder    *Reminder `json:"reminder"`
	Priority    string    `json:"priority"`
}

type Reminder struct {
	DueDate          string  `json:"duedate"`
	BeforeDueDate    *string `json:"beforeduedate"`
	RecurringPattern string  `json:"pattern,omitempty"`
}
