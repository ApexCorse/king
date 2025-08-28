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

	TaskAssignments []TaskAssignment `gorm:"foreignKey:AssignedUserID"`
	CreatedTasks    []Task           `gorm:"foreignKey:AuthorID"`
}

type Task struct {
	gorm.Model

	Title       string
	Description string
	Role        string
	Status      string `gorm:"default:Not Started"`

	AuthorID uint
	Author   User

	// Many-to-many relationship with users through TaskAssignment
	Assignments []TaskAssignment

	Comments []TaskComment
}

// TaskAssignment represents the many-to-many relationship between tasks and assigned users
type TaskAssignment struct {
	gorm.Model

	TaskID         uint
	Task           Task
	AssignedUserID uint
	AssignedUser   User
}

type TaskComment struct {
	gorm.Model

	Text string

	TaskID uint
}
