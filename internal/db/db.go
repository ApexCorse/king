package db

import (
	"fmt"
	"slices"

	"gorm.io/gorm"
)

type UserRetrieveOptions struct {
	WithAssignedTasks bool
	WithCreatedTasks  bool
}

type DB struct {
	db *gorm.DB
}

func NewDB(db *gorm.DB) *DB {
	return &DB{db: db}
}

func (d *DB) CreateUser(user *User) error {
	return d.db.Create(user).Error
}

func (d *DB) GetUserByID(id uint) (*User, error) {
	user := &User{}
	if err := d.db.First(user, id).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (d *DB) GetUserByDiscordID(discordID string, options *UserRetrieveOptions) (*User, error) {
	user := &User{}
	query := d.db
	if options != nil {
		if options.WithAssignedTasks {
			query = query.Preload("AssignedTasks.Author")
		}
		if options.WithCreatedTasks {
			query = query.Preload("CreatedTasks.Author")
		}
	}

	if err := query.
		Where("discord_id = ?", discordID).
		First(user).Error; err != nil {
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
		author, err := d.GetUserByDiscordID(authorID, nil)
		if err != nil {
			errorChan <- err
			return
		}
		authorChan <- author
	}()

	// Run assignee query in parallel
	if assigneeID != "" {
		go func() {
			assignee, err := d.GetUserByDiscordID(assigneeID, nil)
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
		task.AssignedUsers = append(task.AssignedUsers, *assignee)
	}

	return d.db.Create(task).Error
}

func (d *DB) GetTaskByID(id uint) (*Task, error) {
	task := &Task{}

	if err := d.db.Preload("Author").Preload("AssignedUsers").First(task, id).Error; err != nil {
		return nil, err
	}

	return task, nil
}

func (d *DB) GetCompletedTasksByRole(role string) ([]Task, error) {
	tasks := make([]Task, 0)
	if role == "" {
		return nil, fmt.Errorf("role cannot be empty")
	}
	if err := d.db.Preload("Author").Preload("AssignedUsers").
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
	if err := d.db.Preload("Author").Preload("AssignedUsers").
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
	if err := d.db.Preload("Author").Preload("AssignedUsers").
		Where("role = ? AND status != ?", role, TASK_COMPLETED).
		Find(&tasks).Error; err != nil {
		return nil, err
	}

	unassignedTasks := make([]Task, 0)
	for _, task := range tasks {
		if len(task.AssignedUsers) == 0 {
			unassignedTasks = append(unassignedTasks, task)
		}
	}

	return unassignedTasks, nil
}

func (d *DB) AssignTask(taskID uint, userID uint) error {
	task := &Task{}
	task.ID = taskID

	user, err := d.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if err := d.db.Model(task).Association("AssignedUsers").Append(user); err != nil {
		return fmt.Errorf("failed to assign task: %w", err)
	}

	return nil
}

func (d *DB) UpdateTaskStatus(taskID uint, status string) (*Task, error) {
	validStatuses := []string{TASK_NOT_STARTED, TASK_IN_PROGRESS, TASK_COMPLETED}
	isValid := slices.Contains(validStatuses, status)

	if !isValid {
		return nil, fmt.Errorf("invalid status '%s'. Valid statuses are: %s, %s, %s",
			status, TASK_NOT_STARTED, TASK_IN_PROGRESS, TASK_COMPLETED)
	}

	task, err := d.GetTaskByID(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	task.Status = status

	if err := d.db.Save(task).Error; err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	return task, nil
}

func (d *DB) DeleteTask(taskID uint) error {
	return d.db.Model(&Task{}).Where("id = ?", taskID).Delete(&Task{}).Error
}

func (d *DB) GetWebhookSubscriptionsByRepository(repoName string) ([]WebhookSubscription, error) {
	subscriptions := make([]WebhookSubscription, 0)

	if err := d.db.Preload("Repository").
		Joins("JOIN repositories ON repositories.id = webhook_subscriptions.repository_id").
		Where("repositories.name = ?", repoName).
		Find(&subscriptions).Error; err != nil {
		return nil, err
	}

	return subscriptions, nil
}

func (d *DB) CreateWebhookSubscription(repoName string, channelID string) (*WebhookSubscription, error) {
	repo := &Repository{}

	if err := d.db.Where("name = ?", repoName).First(repo).Error; err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	subscription := &WebhookSubscription{
		RepositoryID: repo.ID,
		ChannelID:    channelID,
	}

	if err := d.db.Create(subscription).Error; err != nil {
		return nil, err
	}

	return subscription, nil
}

func (d *DB) DeleteWebhookSubscription(repoName string, channelID string) error {
	webhookSubscription := &WebhookSubscription{}

	if err := d.db.Joins("JOIN repositories ON repositories.id = webhook_subscriptions.repository_id").
		Where("repositories.name = ? AND webhook_subscriptions.channel_id = ?", repoName, channelID).
		First(webhookSubscription).Error; err != nil {
		return fmt.Errorf("failed to get webhook subscription: %w", err)
	}

	return d.db.Unscoped().Delete(webhookSubscription).Error
}

func (d *DB) CreateRepository(repo *Repository) error {
	return d.db.Create(repo).Error
}

func (d *DB) GetAllRepositories() ([]Repository, error) {
	repositories := make([]Repository, 0)

	if err := d.db.Order("name").Find(&repositories).Error; err != nil {
		return nil, err
	}

	return repositories, nil
}
