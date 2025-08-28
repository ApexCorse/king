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

func (d *DB) CreateTaskWithUserDiscordID(task *Task, authorID string, assigneeIDs []string) error {
	// Create channels to receive results from goroutines
	authorChan := make(chan *User, 1)
	assigneeChans := make([]chan *User, len(assigneeIDs))
	errorChan := make(chan error, 1+len(assigneeIDs))

	// Run author query
	go func() {
		author, err := d.GetUserByDiscordID(authorID)
		if err != nil {
			errorChan <- err
			return
		}
		authorChan <- author
	}()

	// Run assignee queries in parallel
	for i, assigneeID := range assigneeIDs {
		assigneeChans[i] = make(chan *User, 1)
		go func(idx int, id string) {
			assignee, err := d.GetUserByDiscordID(id)
			if err != nil {
				errorChan <- err
				return
			}
			assigneeChans[idx] <- assignee
		}(i, assigneeID)
	}

	// Wait for author result
	var author *User
	select {
	case err := <-errorChan:
		return err
	case author = <-authorChan:
	}

	// Wait for all assignee results
	assignees := make([]*User, len(assigneeIDs))
	for i, ch := range assigneeChans {
		select {
		case err := <-errorChan:
			return err
		case assignees[i] = <-ch:
		}
	}

	task.AuthorID = author.ID

	// Create the task first
	if err := d.db.Create(task).Error; err != nil {
		return err
	}

	// Create task assignments for each assignee
	for _, assignee := range assignees {
		assignment := &TaskAssignment{
			TaskID:         task.ID,
			AssignedUserID: assignee.ID,
		}
		if err := d.db.Create(assignment).Error; err != nil {
			return err
		}
	}

	return nil
}

func (d *DB) GetAssignedTasksByUserDiscordID(userID string) ([]Task, error) {
	tasks := make([]Task, 0)

	user, err := d.GetUserByDiscordID(userID)
	if err != nil {
		return nil, err
	}

	if err := d.db.Preload("Author").Preload("Assignments.AssignedUser").
		Joins("JOIN task_assignments ON tasks.id = task_assignments.task_id").
		Where("task_assignments.assigned_user_id = ? AND tasks.status != ?", user.ID, TASK_COMPLETED).
		Find(&tasks).Error; err != nil {
		return nil, err
	}

	return tasks, nil
}

func (d *DB) GetTaskByID(id int64) (*Task, error) {
	task := &Task{}

	if err := d.db.Preload("Author").Preload("Assignments.AssignedUser").First(task, id).Error; err != nil {
		return nil, err
	}

	return task, nil
}

func (d *DB) GetCompletedTasksByRole(role string) ([]Task, error) {
	tasks := make([]Task, 0)
	if role == "" {
		return nil, fmt.Errorf("role cannot be empty")
	}
	if err := d.db.Preload("Author").Preload("Assignments.AssignedUser").
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
	if err := d.db.Preload("Author").Preload("Assignments.AssignedUser").
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
	if err := d.db.Preload("Author").Preload("Assignments.AssignedUser").
		Where("role = ? AND status != ? AND id NOT IN (SELECT DISTINCT task_id FROM task_assignments)", role, TASK_COMPLETED).
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (d *DB) AssignTask(taskID int64, userID int64) error {
	// Check if assignment already exists
	var existingAssignment TaskAssignment
	err := d.db.Where("task_id = ? AND assigned_user_id = ?", taskID, userID).First(&existingAssignment).Error
	if err == nil {
		return fmt.Errorf("task is already assigned to this user")
	}

	assignment := &TaskAssignment{
		TaskID:         uint(taskID),
		AssignedUserID: uint(userID),
	}

	if err := d.db.Create(assignment).Error; err != nil {
		return fmt.Errorf("failed to assign task: %w", err)
	}

	return nil
}

func (d *DB) UnassignTask(taskID int64, userID int64) error {
	if err := d.db.Where("task_id = ? AND assigned_user_id = ?", taskID, userID).Delete(&TaskAssignment{}).Error; err != nil {
		return fmt.Errorf("failed to unassign task: %w", err)
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
