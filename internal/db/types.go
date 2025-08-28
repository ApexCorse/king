package db

import (
	"database/sql"

	"gorm.io/gorm"
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
