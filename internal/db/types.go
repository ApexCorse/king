package db

import "gorm.io/gorm"

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

	AuthorID uint
	Author   User

	AssignedUserID uint
	AssignedUser   User

	Comments []TaskComment
}

type TaskComment struct {
	gorm.Model

	Text string

	TaskID uint
}
