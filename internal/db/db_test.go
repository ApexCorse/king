package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateUser(t *testing.T) {
	t.Run("successful user creation", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		user := &User{
			Username:  "Apex",
			DiscordID: "1234567890",
		}

		err := db.CreateUser(user)
		assert.NoError(t, err)
		assert.NotZero(t, user.ID)

		// Verify user was saved to database
		dbUser := &User{}
		err = gormDB.First(dbUser, user.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, user.Username, dbUser.Username)
		assert.Equal(t, user.DiscordID, dbUser.DiscordID)
	})

	t.Run("duplicate username should fail", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		user1 := &User{
			Username:  "Apex",
			DiscordID: "1234567890",
		}
		err := db.CreateUser(user1)
		assert.NoError(t, err)

		user2 := &User{
			Username:  "Apex", // Same username
			DiscordID: "0987654321",
		}
		err = db.CreateUser(user2)
		assert.Error(t, err)
	})

	t.Run("duplicate discord ID should fail", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		user1 := &User{
			Username:  "Apex",
			DiscordID: "1234567890",
		}
		err := db.CreateUser(user1)
		assert.NoError(t, err)

		user2 := &User{
			Username:  "DifferentUser",
			DiscordID: "1234567890", // Same Discord ID
		}
		err = db.CreateUser(user2)
		assert.Error(t, err)
	})
}

func TestGetUserByDiscordID(t *testing.T) {
	t.Run("successful user retrieval", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create a user first
		user := &User{
			Username:  "Apex",
			DiscordID: "1234567890",
		}
		err := db.CreateUser(user)
		require.NoError(t, err)

		// Retrieve user by Discord ID
		retrievedUser, err := db.GetUserByDiscordID("1234567890")
		assert.NoError(t, err)
		assert.NotNil(t, retrievedUser)
		assert.Equal(t, user.ID, retrievedUser.ID)
		assert.Equal(t, user.Username, retrievedUser.Username)
		assert.Equal(t, user.DiscordID, retrievedUser.DiscordID)
	})

	t.Run("user not found", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Try to retrieve non-existent user
		user, err := db.GetUserByDiscordID("nonexistent")
		assert.Error(t, err)
		assert.Nil(t, user)
	})

	t.Run("empty discord ID", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		user, err := db.GetUserByDiscordID("")
		assert.Error(t, err)
		assert.Nil(t, user)
	})
}

func TestCreateTaskWithUserDiscordID(t *testing.T) {
	t.Run("successful task creation with both users", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create author and assignee users
		author := &User{
			Username:  "Author",
			DiscordID: "author123",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)

		assignee := &User{
			Username:  "Assignee",
			DiscordID: "assignee456",
		}
		err = db.CreateUser(assignee)
		require.NoError(t, err)

		// Create task
		task := &Task{
			Title:       "Test Task",
			Description: "Test Description",
		}

		err = db.CreateTaskWithUserDiscordID(task, "author123", "assignee456")
		assert.NoError(t, err)
		assert.NotZero(t, task.ID)
		assert.Equal(t, author.ID, task.AuthorID)
		assert.Equal(t, assignee.ID, task.AssignedUserID)

		// Verify task was saved to database
		dbTask := &Task{}
		err = gormDB.Preload("Author").Preload("AssignedUser").First(dbTask, task.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, task.Title, dbTask.Title)
		assert.Equal(t, task.Description, dbTask.Description)
		assert.Equal(t, author.ID, dbTask.AuthorID)
		assert.Equal(t, assignee.ID, dbTask.AssignedUserID)
		assert.Equal(t, author.Username, dbTask.Author.Username)
		assert.Equal(t, assignee.Username, dbTask.AssignedUser.Username)
	})

	t.Run("author not found", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create only assignee
		assignee := &User{
			Username:  "Assignee",
			DiscordID: "assignee456",
		}
		err := db.CreateUser(assignee)
		require.NoError(t, err)

		task := &Task{
			Title:       "Test Task",
			Description: "Test Description",
		}

		err = db.CreateTaskWithUserDiscordID(task, "nonexistent_author", "assignee456")
		assert.Error(t, err)
		assert.Zero(t, task.ID)
	})

	t.Run("assignee not found", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create only author
		author := &User{
			Username:  "Author",
			DiscordID: "author123",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)

		task := &Task{
			Title:       "Test Task",
			Description: "Test Description",
		}

		err = db.CreateTaskWithUserDiscordID(task, "author123", "nonexistent_assignee")
		assert.Error(t, err)
		assert.Zero(t, task.ID)
	})

	t.Run("both users not found", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		task := &Task{
			Title:       "Test Task",
			Description: "Test Description",
		}

		err := db.CreateTaskWithUserDiscordID(task, "nonexistent_author", "nonexistent_assignee")
		assert.Error(t, err)
		assert.Zero(t, task.ID)
	})

	t.Run("same user as author and assignee", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		user := &User{
			Username:  "SameUser",
			DiscordID: "same123",
		}
		err := db.CreateUser(user)
		require.NoError(t, err)

		task := &Task{
			Title:       "Self-Assigned Task",
			Description: "Task assigned to self",
		}

		err = db.CreateTaskWithUserDiscordID(task, "same123", "same123")
		assert.NoError(t, err)
		assert.NotZero(t, task.ID)
		assert.Equal(t, user.ID, task.AuthorID)
		assert.Equal(t, user.ID, task.AssignedUserID)
	})

	t.Run("empty discord IDs", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		task := &Task{
			Title:       "Test Task",
			Description: "Test Description",
		}

		err := db.CreateTaskWithUserDiscordID(task, "", "")
		assert.Error(t, err)
		assert.Zero(t, task.ID)
	})
}

func TestNewDB(t *testing.T) {
	t.Run("create new DB instance", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		assert.NotNil(t, db)
		assert.Equal(t, gormDB, db.db)
	})
}

func TestGetAssignedTasksByUserDiscordID(t *testing.T) {
	t.Run("successful retrieval of assigned tasks", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create users
		author := &User{
			Username:  "Author",
			DiscordID: "author123",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)

		assignee := &User{
			Username:  "Assignee",
			DiscordID: "assignee456",
		}
		err = db.CreateUser(assignee)
		require.NoError(t, err)

		// Create tasks assigned to the assignee
		task1 := &Task{
			Title:       "Task 1",
			Description: "First task",
		}
		err = db.CreateTaskWithUserDiscordID(task1, "author123", "assignee456")
		require.NoError(t, err)

		task2 := &Task{
			Title:       "Task 2",
			Description: "Second task",
		}
		err = db.CreateTaskWithUserDiscordID(task2, "author123", "assignee456")
		require.NoError(t, err)

		// Create a task assigned to someone else
		otherAssignee := &User{
			Username:  "OtherAssignee",
			DiscordID: "other789",
		}
		err = db.CreateUser(otherAssignee)
		require.NoError(t, err)

		task3 := &Task{
			Title:       "Task 3",
			Description: "Third task",
		}
		err = db.CreateTaskWithUserDiscordID(task3, "author123", "other789")
		require.NoError(t, err)

		// Retrieve tasks assigned to the first assignee
		tasks, err := db.GetAssignedTasksByUserDiscordID("assignee456")
		assert.NoError(t, err)
		assert.Len(t, tasks, 2)

		// Verify the tasks are the correct ones
		taskIDs := make(map[uint]bool)
		for _, task := range tasks {
			taskIDs[task.ID] = true
		}
		assert.True(t, taskIDs[task1.ID])
		assert.True(t, taskIDs[task2.ID])
		assert.False(t, taskIDs[task3.ID])
	})

	t.Run("user with no assigned tasks", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create a user with no tasks
		user := &User{
			Username:  "NoTasksUser",
			DiscordID: "notasks123",
		}
		err := db.CreateUser(user)
		require.NoError(t, err)

		// Retrieve tasks for user with no assignments
		tasks, err := db.GetAssignedTasksByUserDiscordID("notasks123")
		assert.NoError(t, err)
		assert.Empty(t, tasks)
	})

	t.Run("non-existent user discord ID", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Try to retrieve tasks for non-existent user
		tasks, err := db.GetAssignedTasksByUserDiscordID("nonexistent")
		assert.Error(t, err)
		assert.Nil(t, tasks)
	})

	t.Run("empty discord ID", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Try to retrieve tasks with empty discord ID
		tasks, err := db.GetAssignedTasksByUserDiscordID("")
		assert.Error(t, err)
		assert.Nil(t, tasks)
	})
}
