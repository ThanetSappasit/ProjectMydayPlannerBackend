package model

import (
	"time"
)

type Task struct {
	TaskID      int       `gorm:"column:task_id;primaryKey;autoIncrement"`
	BoardID     int       `gorm:"column:board_id;not null"`
	TaskName    string    `gorm:"column:task_name;type:varchar(255);not null"`
	Description string    `gorm:"column:description;type:text"`
	Status      string    `gorm:"column:status;type:enum('0','1','2');default:'0'"`
	Priority    string    `gorm:"column:priority;type:enum('1','2','3')"`
	CreatedBy   *int      `gorm:"column:create_by"`
	AssignedTo  int       `gorm:"column:assigned_to;not null"`
	CreatedAt   time.Time `gorm:"column:create_at;autoCreateTime"`

	// Relations
	Board    Board `gorm:"foreignKey:BoardID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE"`
	Creator  User  `gorm:"foreignKey:CreatedBy;references:UserID"`
	Assignee User  `gorm:"foreignKey:AssignedTo;references:UserID"`
}

func (Task) TableName() string {
	return "tasks"
}
