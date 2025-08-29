package db

import (
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

	AssignedTasks []Task `gorm:"many2many:task_assignments"`
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

	AssignedUsers []User `gorm:"many2many:task_assignments"`

	Comments []TaskComment
}

type TaskComment struct {
	gorm.Model

	Text string

	TaskID uint
}

type WebhookSubscription struct {
	gorm.Model

	ChannelID string

	RepositoryID uint
	Repository   Repository
}

type Repository struct {
	gorm.Model

	Name          string `gorm:"unique"`
	Subscriptions []WebhookSubscription
}
