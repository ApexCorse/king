package db

import "gorm.io/gorm"

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
	go func() {
		assignee, err := d.GetUserByDiscordID(assigneeID)
		if err != nil {
			errorChan <- err
			return
		}
		assigneeChan <- assignee
	}()

	// Wait for both results
	var author, assignee *User
	for range 2 {
		select {
		case err := <-errorChan:
			return err
		case author = <-authorChan:
		case assignee = <-assigneeChan:
		}
	}

	task.AuthorID = author.ID
	task.AssignedUserID = assignee.ID

	return d.db.Create(task).Error
}

func (d *DB) GetAssignedTasksByUserDiscordID(userID string) ([]Task, error) {
	tasks := make([]Task, 0)

	user, err := d.GetUserByDiscordID(userID)
	if err != nil {
		return nil, err
	}

	if err := d.db.Preload("Author").Preload("AssignedUser").Where("assigned_user_id = ?", user.ID).Find(&tasks).Error; err != nil {
		return nil, err
	}

	return tasks, nil
}
