package db

import (
	"database/sql"

	"gorm.io/gorm"
)

const (
	TASK_NOT_STARTED = "Not Started"
	TASK_IN_PROGRESS = "In Progress"
	TASK_COMPLETED   = "Completed"
)

type User struct {
	gorm.Model

	Username  string `gorm:"unique"`
	DiscordID string `gorm:"unique"`

	AssignedTasks []Task `gorm:"foreignKey:AssignedUserID"`
	CreatedTasks  []Task `gorm:"foreignKey:AuthorID"`
}

type Task struct {
	gorm.Model

	Title       string
	Description string
	Role        string
	Status      string `gorm:"default:Not Started"`

	AuthorID uint
	Author   User

	AssignedUserID sql.NullInt64
	AssignedUser   User

	Comments []TaskComment
}

type TaskComment struct {
	gorm.Model

	Text string

	TaskID uint
}

type WebhookSubscriptions struct {
	gorm.Model

	Repository string
	ChannelID  string
}
