package db

import (
	"database/sql"
	"fmt"
	"slices"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DB struct {
	db *gorm.DB
}

func NewDB(db *gorm.DB) *DB {
	return &DB{db: db}
}

func (d *DB) CreateUser(user *User) error {
	return d.db.Create(user).Error
}

func (d *DB) GetUserByDiscordID(discordID string) (*User, error) {
	user := &User{}

	if err := d.db.Where("discord_id = ?", discordID).First(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (d *DB) CreateTaskWithUserDiscordID(task *Task, authorID string, assigneeID string) error {
	// Create channels to receive results from goroutines
	authorChan := make(chan *User, 1)
	assigneeChan := make(chan *User, 1)
	errorChan := make(chan error, 2)

	// Run author query in parallel
	go func() {
		author, err := d.GetUserByDiscordID(authorID)
		if err != nil {
			errorChan <- err
			return
		}
		authorChan <- author
	}()

	// Run assignee query in parallel
	if assigneeID != "" {
		go func() {
			assignee, err := d.GetUserByDiscordID(assigneeID)
			if err != nil {
				errorChan <- err
				return
			}
			assigneeChan <- assignee
		}()
	}

	// Wait for both results
	var author, assignee *User
	numChannels := 1
	if assigneeID != "" {
		numChannels++
	}
	for range numChannels {
		select {
		case err := <-errorChan:
			return err
		case author = <-authorChan:
		case assignee = <-assigneeChan:
		}
	}

	task.AuthorID = author.ID
	if assigneeID != "" {
		task.AssignedUserID = sql.NullInt64{
			Int64: int64(assignee.ID),
			Valid: true,
		}
	}

	return d.db.Create(task).Error
}

func (d *DB) GetAssignedTasksByUserDiscordID(userID string) ([]Task, error) {
	tasks := make([]Task, 0)

	user, err := d.GetUserByDiscordID(userID)
	if err != nil {
		return nil, err
	}

	if err := d.db.Preload("Author").Preload("AssignedUser").
		Where("assigned_user_id = ? AND status != ?", int64(user.ID), TASK_COMPLETED).
		Find(&tasks).Error; err != nil {
		return nil, err
	}

	return tasks, nil
}

func (d *DB) GetTaskByID(id int64) (*Task, error) {
	task := &Task{}

	if err := d.db.Preload("Author").Preload("AssignedUser").First(task, id).Error; err != nil {
		return nil, err
	}

	return task, nil
}

func (d *DB) GetCompletedTasksByRole(role string) ([]Task, error) {
	tasks := make([]Task, 0)
	if role == "" {
		return nil, fmt.Errorf("role cannot be empty")
	}
	if err := d.db.Preload("Author").Preload("AssignedUser").
		Where("role = ? AND status = ?", role, TASK_COMPLETED).
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (d *DB) GetTasksByRole(role string) ([]Task, error) {
	tasks := make([]Task, 0)
	if role == "" {
		return nil, fmt.Errorf("role cannot be empty")
	}
	if err := d.db.Preload("Author").Preload("AssignedUser").
		Where("role = ? AND status != ?", role, TASK_COMPLETED).
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (d *DB) GetUnassignedTasksByRole(role string) ([]Task, error) {
	tasks := make([]Task, 0)
	if role == "" {
		return nil, fmt.Errorf("role cannot be empty")
	}
	if err := d.db.Preload("Author").Preload("AssignedUser").
		Where("role = ? AND assigned_user_id IS NULL AND status != ?", role, TASK_COMPLETED).
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (d *DB) AssignTask(taskID int64, userID int64) error {
	if err := d.db.Model(&Task{}).Where("id = ?", taskID).Update("assigned_user_id", userID).Error; err != nil {
		return fmt.Errorf("failed to assign task: %w", err)
	}

	return nil
}

func (d *DB) UpdateTaskStatus(taskID int64, status string) (*Task, error) {
	validStatuses := []string{TASK_NOT_STARTED, TASK_IN_PROGRESS, TASK_COMPLETED}
	isValid := slices.Contains(validStatuses, status)

	if !isValid {
		return nil, fmt.Errorf("invalid status '%s'. Valid statuses are: %s, %s, %s",
			status, TASK_NOT_STARTED, TASK_IN_PROGRESS, TASK_COMPLETED)
	}

	task := &Task{}

	if err := d.db.Model(task).
		Clauses(clause.Returning{}).
		Where("id = ?", taskID).
		Update("status", status).Error; err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	return task, nil
}
