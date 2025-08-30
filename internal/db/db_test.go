package db

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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
		retrievedUser, err := db.GetUserByDiscordID("1234567890", nil)
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
		user, err := db.GetUserByDiscordID("nonexistent", nil)
		assert.Error(t, err)
		assert.Nil(t, user)
	})

	t.Run("empty discord ID", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		user, err := db.GetUserByDiscordID("", nil)
		assert.Error(t, err)
		assert.Nil(t, user)
	})

	t.Run("retrieve user with assigned tasks", func(t *testing.T) {
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
			Description: "First assigned task",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task1, "author123", "assignee456")
		require.NoError(t, err)

		task2 := &Task{
			Title:       "Task 2",
			Description: "Second assigned task",
			Role:        "designer",
		}
		err = db.CreateTaskWithUserDiscordID(task2, "author123", "assignee456")
		require.NoError(t, err)

		// Retrieve user with assigned tasks
		retrievedUser, err := db.GetUserByDiscordID("assignee456", &UserRetrieveOptions{
			WithAssignedTasks: true,
		})
		assert.NoError(t, err)
		assert.NotNil(t, retrievedUser)
		assert.Equal(t, assignee.ID, retrievedUser.ID)
		assert.Equal(t, assignee.Username, retrievedUser.Username)
		assert.Len(t, retrievedUser.AssignedTasks, 2)

		// Verify assigned tasks are loaded with their authors
		taskIDs := make(map[uint]bool)
		for _, task := range retrievedUser.AssignedTasks {
			taskIDs[task.ID] = true
			assert.Equal(t, author.ID, task.AuthorID)
			assert.Equal(t, author.Username, task.Author.Username)
			assert.NotEmpty(t, task.Title)
			assert.NotEmpty(t, task.Description)
		}
		assert.True(t, taskIDs[task1.ID])
		assert.True(t, taskIDs[task2.ID])
	})

	t.Run("retrieve user with created tasks", func(t *testing.T) {
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

		// Create tasks created by the author
		task1 := &Task{
			Title:       "Created Task 1",
			Description: "First created task",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task1, "author123", "assignee456")
		require.NoError(t, err)

		task2 := &Task{
			Title:       "Created Task 2",
			Description: "Second created task",
			Role:        "designer",
		}
		err = db.CreateTaskWithUserDiscordID(task2, "author123", "")
		require.NoError(t, err)

		// Retrieve user with created tasks
		retrievedUser, err := db.GetUserByDiscordID("author123", &UserRetrieveOptions{
			WithCreatedTasks: true,
		})
		assert.NoError(t, err)
		assert.NotNil(t, retrievedUser)
		assert.Equal(t, author.ID, retrievedUser.ID)
		assert.Equal(t, author.Username, retrievedUser.Username)
		assert.Len(t, retrievedUser.CreatedTasks, 2)

		// Verify created tasks are loaded with their authors (should be the same user)
		taskIDs := make(map[uint]bool)
		for _, task := range retrievedUser.CreatedTasks {
			taskIDs[task.ID] = true
			assert.Equal(t, author.ID, task.AuthorID)
			assert.Equal(t, author.Username, task.Author.Username)
			assert.NotEmpty(t, task.Title)
			assert.NotEmpty(t, task.Description)
		}
		assert.True(t, taskIDs[task1.ID])
		assert.True(t, taskIDs[task2.ID])
	})

	t.Run("retrieve user with both assigned and created tasks", func(t *testing.T) {
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

		// Create tasks where author creates and assignee is assigned
		task1 := &Task{
			Title:       "Shared Task 1",
			Description: "Task created by author, assigned to assignee",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task1, "author123", "assignee456")
		require.NoError(t, err)

		task2 := &Task{
			Title:       "Author Only Task",
			Description: "Task created by author, no assignee",
			Role:        "designer",
		}
		err = db.CreateTaskWithUserDiscordID(task2, "author123", "")
		require.NoError(t, err)

		// Retrieve author with both created and assigned tasks
		retrievedAuthor, err := db.GetUserByDiscordID("author123", &UserRetrieveOptions{
			WithCreatedTasks:  true,
			WithAssignedTasks: true,
		})
		assert.NoError(t, err)
		assert.NotNil(t, retrievedAuthor)
		assert.Equal(t, author.ID, retrievedAuthor.ID)
		assert.Len(t, retrievedAuthor.CreatedTasks, 2)  // Should have 2 created tasks
		assert.Len(t, retrievedAuthor.AssignedTasks, 0) // Should have 0 assigned tasks

		// Retrieve assignee with both created and assigned tasks
		retrievedAssignee, err := db.GetUserByDiscordID("assignee456", &UserRetrieveOptions{
			WithCreatedTasks:  true,
			WithAssignedTasks: true,
		})
		assert.NoError(t, err)
		assert.NotNil(t, retrievedAssignee)
		assert.Equal(t, assignee.ID, retrievedAssignee.ID)
		assert.Len(t, retrievedAssignee.CreatedTasks, 0)  // Should have 0 created tasks
		assert.Len(t, retrievedAssignee.AssignedTasks, 1) // Should have 1 assigned task

		// Verify the shared task appears in both relationships
		assert.Equal(t, task1.ID, retrievedAssignee.AssignedTasks[0].ID)
		assert.Equal(t, task1.ID, retrievedAuthor.CreatedTasks[0].ID)
	})

	t.Run("retrieve user with no tasks", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create user with no tasks
		user := &User{
			Username:  "NoTasksUser",
			DiscordID: "notasks123",
		}
		err := db.CreateUser(user)
		require.NoError(t, err)

		// Retrieve user with assigned tasks option
		retrievedUser, err := db.GetUserByDiscordID("notasks123", &UserRetrieveOptions{
			WithAssignedTasks: true,
		})
		assert.NoError(t, err)
		assert.NotNil(t, retrievedUser)
		assert.Equal(t, user.ID, retrievedUser.ID)
		assert.Empty(t, retrievedUser.AssignedTasks)

		// Retrieve user with created tasks option
		retrievedUser, err = db.GetUserByDiscordID("notasks123", &UserRetrieveOptions{
			WithCreatedTasks: true,
		})
		assert.NoError(t, err)
		assert.NotNil(t, retrievedUser)
		assert.Equal(t, user.ID, retrievedUser.ID)
		assert.Empty(t, retrievedUser.CreatedTasks)

		// Retrieve user with both options
		retrievedUser, err = db.GetUserByDiscordID("notasks123", &UserRetrieveOptions{
			WithAssignedTasks: true,
			WithCreatedTasks:  true,
		})
		assert.NoError(t, err)
		assert.NotNil(t, retrievedUser)
		assert.Equal(t, user.ID, retrievedUser.ID)
		assert.Empty(t, retrievedUser.AssignedTasks)
		assert.Empty(t, retrievedUser.CreatedTasks)
	})

	t.Run("retrieve user with complex task relationships", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create multiple users
		user1 := &User{
			Username:  "User1",
			DiscordID: "user1",
		}
		err := db.CreateUser(user1)
		require.NoError(t, err)

		user2 := &User{
			Username:  "User2",
			DiscordID: "user2",
		}
		err = db.CreateUser(user2)
		require.NoError(t, err)

		user3 := &User{
			Username:  "User3",
			DiscordID: "user3",
		}
		err = db.CreateUser(user3)
		require.NoError(t, err)

		// Create complex task relationships:
		// - User1 creates Task1, assigned to User2
		// - User2 creates Task2, assigned to User1
		// - User1 creates Task3, assigned to User3
		// - User3 creates Task4, assigned to User1

		task1 := &Task{
			Title:       "Task 1",
			Description: "Created by User1, assigned to User2",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task1, "user1", "user2")
		require.NoError(t, err)

		task2 := &Task{
			Title:       "Task 2",
			Description: "Created by User2, assigned to User1",
			Role:        "designer",
		}
		err = db.CreateTaskWithUserDiscordID(task2, "user2", "user1")
		require.NoError(t, err)

		task3 := &Task{
			Title:       "Task 3",
			Description: "Created by User1, assigned to User3",
			Role:        "qa",
		}
		err = db.CreateTaskWithUserDiscordID(task3, "user1", "user3")
		require.NoError(t, err)

		task4 := &Task{
			Title:       "Task 4",
			Description: "Created by User3, assigned to User1",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task4, "user3", "user1")
		require.NoError(t, err)

		// Test User1: should have 2 created tasks and 2 assigned tasks
		retrievedUser1, err := db.GetUserByDiscordID("user1", &UserRetrieveOptions{
			WithCreatedTasks:  true,
			WithAssignedTasks: true,
		})
		assert.NoError(t, err)
		assert.Len(t, retrievedUser1.CreatedTasks, 2)  // Task1, Task3
		assert.Len(t, retrievedUser1.AssignedTasks, 2) // Task2, Task4

		// Verify created tasks
		createdTaskIDs := make(map[uint]bool)
		for _, task := range retrievedUser1.CreatedTasks {
			createdTaskIDs[task.ID] = true
		}
		assert.True(t, createdTaskIDs[task1.ID])
		assert.True(t, createdTaskIDs[task3.ID])

		// Verify assigned tasks
		assignedTaskIDs := make(map[uint]bool)
		for _, task := range retrievedUser1.AssignedTasks {
			assignedTaskIDs[task.ID] = true
		}
		assert.True(t, assignedTaskIDs[task2.ID])
		assert.True(t, assignedTaskIDs[task4.ID])

		// Test User2: should have 1 created task and 1 assigned task
		retrievedUser2, err := db.GetUserByDiscordID("user2", &UserRetrieveOptions{
			WithCreatedTasks:  true,
			WithAssignedTasks: true,
		})
		assert.NoError(t, err)
		assert.Len(t, retrievedUser2.CreatedTasks, 1)  // Task2
		assert.Len(t, retrievedUser2.AssignedTasks, 1) // Task1

		// Test User3: should have 1 created task and 1 assigned task
		retrievedUser3, err := db.GetUserByDiscordID("user3", &UserRetrieveOptions{
			WithCreatedTasks:  true,
			WithAssignedTasks: true,
		})
		assert.NoError(t, err)
		assert.Len(t, retrievedUser3.CreatedTasks, 1)  // Task4
		assert.Len(t, retrievedUser3.AssignedTasks, 1) // Task3
	})

	t.Run("retrieve user with tasks having different statuses", func(t *testing.T) {
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

		// Create tasks with different statuses
		task1 := &Task{
			Title:       "Not Started Task",
			Description: "Task with default status",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task1, "author123", "assignee456")
		require.NoError(t, err)

		task2 := &Task{
			Title:       "In Progress Task",
			Description: "Task in progress",
			Role:        "designer",
		}
		err = db.CreateTaskWithUserDiscordID(task2, "author123", "assignee456")
		require.NoError(t, err)
		_, err = db.UpdateTaskStatus(task2.ID, TASK_IN_PROGRESS)
		require.NoError(t, err)

		task3 := &Task{
			Title:       "Completed Task",
			Description: "Completed task",
			Role:        "qa",
		}
		err = db.CreateTaskWithUserDiscordID(task3, "author123", "assignee456")
		require.NoError(t, err)
		_, err = db.UpdateTaskStatus(task3.ID, TASK_COMPLETED)
		require.NoError(t, err)

		// Retrieve assignee with assigned tasks
		retrievedUser, err := db.GetUserByDiscordID("assignee456", &UserRetrieveOptions{
			WithAssignedTasks: true,
		})
		assert.NoError(t, err)
		assert.Len(t, retrievedUser.AssignedTasks, 3)

		// Verify all tasks are loaded regardless of status
		taskIDs := make(map[uint]bool)
		statuses := make(map[string]bool)
		for _, task := range retrievedUser.AssignedTasks {
			taskIDs[task.ID] = true
			statuses[task.Status] = true
		}
		assert.True(t, taskIDs[task1.ID])
		assert.True(t, taskIDs[task2.ID])
		assert.True(t, taskIDs[task3.ID])
		assert.True(t, statuses[TASK_NOT_STARTED])
		assert.True(t, statuses[TASK_IN_PROGRESS])
		assert.True(t, statuses[TASK_COMPLETED])
	})

	t.Run("retrieve user with tasks having different roles", func(t *testing.T) {
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

		// Create tasks with different roles
		roles := []string{"developer", "designer", "qa", "devops", "product"}
		tasks := make([]*Task, len(roles))

		for i, role := range roles {
			task := &Task{
				Title:       fmt.Sprintf("Task %d", i+1),
				Description: fmt.Sprintf("Task with role %s", role),
				Role:        role,
			}
			err = db.CreateTaskWithUserDiscordID(task, "author123", "assignee456")
			require.NoError(t, err)
			tasks[i] = task
		}

		// Retrieve assignee with assigned tasks
		retrievedUser, err := db.GetUserByDiscordID("assignee456", &UserRetrieveOptions{
			WithAssignedTasks: true,
		})
		assert.NoError(t, err)
		assert.Len(t, retrievedUser.AssignedTasks, len(roles))

		// Verify all tasks are loaded with correct roles
		taskIDs := make(map[uint]bool)
		loadedRoles := make(map[string]bool)
		for _, task := range retrievedUser.AssignedTasks {
			taskIDs[task.ID] = true
			loadedRoles[task.Role] = true
		}

		for _, task := range tasks {
			assert.True(t, taskIDs[task.ID])
		}

		for _, role := range roles {
			assert.True(t, loadedRoles[role])
		}
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
		assert.Len(t, task.AssignedUsers, 1)
		assert.Equal(t, assignee.ID, task.AssignedUsers[0].ID)

		// Verify task was saved to database
		dbTask := &Task{}
		err = gormDB.Preload("Author").Preload("AssignedUsers").First(dbTask, task.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, task.Title, dbTask.Title)
		assert.Equal(t, task.Description, dbTask.Description)
		assert.Equal(t, author.ID, dbTask.AuthorID)
		assert.Len(t, dbTask.AssignedUsers, 1)
		assert.Equal(t, assignee.ID, dbTask.AssignedUsers[0].ID)
		assert.Equal(t, author.Username, dbTask.Author.Username)
		assert.Equal(t, assignee.Username, dbTask.AssignedUsers[0].Username)
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
		assert.Len(t, task.AssignedUsers, 1)
		assert.Equal(t, user.ID, task.AssignedUsers[0].ID)
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

	t.Run("task creation with no assigned user", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create only author user
		author := &User{
			Username:  "Author",
			DiscordID: "author123",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)

		// Create task with no assigned user (empty assignee ID)
		task := &Task{
			Title:       "Unassigned Task",
			Description: "Task with no assigned user",
		}

		err = db.CreateTaskWithUserDiscordID(task, "author123", "")
		assert.NoError(t, err)
		assert.NotZero(t, task.ID)
		assert.Equal(t, author.ID, task.AuthorID)
		assert.Len(t, task.AssignedUsers, 0) // Should be invalid when no assignee

		// Verify task was saved to database
		dbTask := &Task{}
		err = gormDB.Preload("Author").Preload("AssignedUsers").First(dbTask, task.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, task.Title, dbTask.Title)
		assert.Equal(t, task.Description, dbTask.Description)
		assert.Equal(t, author.ID, dbTask.AuthorID)
		assert.Len(t, dbTask.AssignedUsers, 0)
		assert.Equal(t, author.Username, dbTask.Author.Username)
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

func TestGetTaskByID(t *testing.T) {
	t.Run("successful task retrieval with assignee", func(t *testing.T) {
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

		// Create task
		task := &Task{
			Title:       "Test Task",
			Description: "Test Description",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "assignee456")
		require.NoError(t, err)

		// Retrieve task by ID
		retrievedTask, err := db.GetTaskByID(task.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedTask)
		assert.Equal(t, task.ID, retrievedTask.ID)
		assert.Equal(t, task.Title, retrievedTask.Title)
		assert.Equal(t, task.Description, retrievedTask.Description)
		assert.Equal(t, author.ID, retrievedTask.AuthorID)
		assert.Len(t, retrievedTask.AssignedUsers, 1)
		assert.Equal(t, assignee.ID, retrievedTask.AssignedUsers[0].ID)
		assert.Equal(t, author.Username, retrievedTask.Author.Username)
		assert.Equal(t, assignee.Username, retrievedTask.AssignedUsers[0].Username)
	})

	t.Run("successful task retrieval without assignee", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create only author
		author := &User{
			Username:  "Author",
			DiscordID: "author123",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)

		// Create task without assignee
		task := &Task{
			Title:       "Unassigned Task",
			Description: "Task with no assignee",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "")
		require.NoError(t, err)

		// Retrieve task by ID
		retrievedTask, err := db.GetTaskByID(task.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedTask)
		assert.Equal(t, task.ID, retrievedTask.ID)
		assert.Equal(t, task.Title, retrievedTask.Title)
		assert.Equal(t, task.Description, retrievedTask.Description)
		assert.Equal(t, author.ID, retrievedTask.AuthorID)
		assert.Len(t, retrievedTask.AssignedUsers, 0)
		assert.Equal(t, author.Username, retrievedTask.Author.Username)
	})

	t.Run("task not found", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Try to retrieve non-existent task
		task, err := db.GetTaskByID(999)
		assert.Error(t, err)
		assert.Nil(t, task)
	})

	t.Run("zero ID", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Try to retrieve task with zero ID
		task, err := db.GetTaskByID(0)
		assert.Error(t, err)
		assert.Nil(t, task)
	})
}

func TestGetUnassignedTasksByRole(t *testing.T) {
	t.Run("successful retrieval of unassigned tasks by role", func(t *testing.T) {
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

		// Create unassigned tasks with "developer" role
		task1 := &Task{
			Title:       "Unassigned Task 1",
			Description: "First unassigned task",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task1, "author123", "")
		require.NoError(t, err)

		task2 := &Task{
			Title:       "Unassigned Task 2",
			Description: "Second unassigned task",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task2, "author123", "")
		require.NoError(t, err)

		// Create assigned task with "developer" role
		task3 := &Task{
			Title:       "Assigned Task",
			Description: "Assigned task",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task3, "author123", "assignee456")
		require.NoError(t, err)

		// Create unassigned task with different role
		task4 := &Task{
			Title:       "Designer Task",
			Description: "Designer unassigned task",
			Role:        "designer",
		}
		err = db.CreateTaskWithUserDiscordID(task4, "author123", "")
		require.NoError(t, err)

		// Retrieve unassigned tasks for "developer" role
		tasks, err := db.GetUnassignedTasksByRole("developer")
		assert.NoError(t, err)
		assert.Len(t, tasks, 2)

		// Verify the tasks are the correct unassigned ones
		taskIDs := make(map[uint]bool)
		for _, task := range tasks {
			taskIDs[task.ID] = true
			assert.Equal(t, "developer", task.Role)
			assert.Len(t, task.AssignedUsers, 0)
			assert.Equal(t, author.Username, task.Author.Username)
		}
		assert.True(t, taskIDs[task1.ID])
		assert.True(t, taskIDs[task2.ID])
		assert.False(t, taskIDs[task3.ID]) // Should not be included (assigned)
		assert.False(t, taskIDs[task4.ID]) // Should not be included (different role)
	})

	t.Run("no unassigned tasks for role", func(t *testing.T) {
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

		// Create only assigned tasks with "developer" role
		task1 := &Task{
			Title:       "Assigned Task 1",
			Description: "First assigned task",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task1, "author123", "assignee456")
		require.NoError(t, err)

		task2 := &Task{
			Title:       "Assigned Task 2",
			Description: "Second assigned task",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task2, "author123", "assignee456")
		require.NoError(t, err)

		// Retrieve unassigned tasks for "developer" role
		tasks, err := db.GetUnassignedTasksByRole("developer")
		assert.NoError(t, err)
		assert.Empty(t, tasks)
	})

	t.Run("no tasks for role at all", func(t *testing.T) {
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

		// Create tasks with different role
		task1 := &Task{
			Title:       "Designer Task",
			Description: "Designer task",
			Role:        "designer",
		}
		err = db.CreateTaskWithUserDiscordID(task1, "author123", "")
		require.NoError(t, err)

		// Retrieve unassigned tasks for "developer" role
		tasks, err := db.GetUnassignedTasksByRole("developer")
		assert.NoError(t, err)
		assert.Empty(t, tasks)
	})

	t.Run("empty role should return error", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Try to retrieve tasks with empty role
		tasks, err := db.GetUnassignedTasksByRole("")
		assert.Error(t, err)
		assert.Nil(t, tasks)
		assert.Contains(t, err.Error(), "role cannot be empty")
	})

	t.Run("mixed roles with unassigned tasks", func(t *testing.T) {
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

		// Create tasks with different roles and assignment status
		tasks := []struct {
			title       string
			description string
			role        string
			assigneeID  string
		}{
			{"Dev Unassigned 1", "Developer unassigned task", "developer", ""},
			{"Dev Unassigned 2", "Developer unassigned task", "developer", ""},
			{"Dev Assigned", "Developer assigned task", "developer", "assignee456"},
			{"Designer Unassigned", "Designer unassigned task", "designer", ""},
			{"Designer Assigned", "Designer assigned task", "designer", "assignee456"},
			{"QA Unassigned", "QA unassigned task", "qa", ""},
		}

		createdTasks := make([]*Task, len(tasks))
		for i, taskData := range tasks {
			task := &Task{
				Title:       taskData.title,
				Description: taskData.description,
				Role:        taskData.role,
			}
			err = db.CreateTaskWithUserDiscordID(task, "author123", taskData.assigneeID)
			require.NoError(t, err)
			createdTasks[i] = task
		}

		// Test developer role
		devTasks, err := db.GetUnassignedTasksByRole("developer")
		assert.NoError(t, err)
		assert.Len(t, devTasks, 2)
		for _, task := range devTasks {
			assert.Equal(t, "developer", task.Role)
			assert.Len(t, task.AssignedUsers, 0)
		}

		// Test designer role
		designerTasks, err := db.GetUnassignedTasksByRole("designer")
		assert.NoError(t, err)
		assert.Len(t, designerTasks, 1)
		assert.Equal(t, "designer", designerTasks[0].Role)
		assert.Len(t, designerTasks[0].AssignedUsers, 0)

		// Test QA role
		qaTasks, err := db.GetUnassignedTasksByRole("qa")
		assert.NoError(t, err)
		assert.Len(t, qaTasks, 1)
		assert.Equal(t, "qa", qaTasks[0].Role)
		assert.Len(t, qaTasks[0].AssignedUsers, 0)
	})

	t.Run("case sensitive role matching", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create users
		author := &User{
			Username:  "Author",
			DiscordID: "author123",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)

		// Create task with "Developer" role (capitalized)
		task := &Task{
			Title:       "Developer Task",
			Description: "Developer task",
			Role:        "Developer",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "")
		require.NoError(t, err)

		// Try to retrieve with "developer" (lowercase)
		tasks, err := db.GetUnassignedTasksByRole("developer")
		assert.NoError(t, err)
		assert.Empty(t, tasks) // Should not match due to case sensitivity

		// Try to retrieve with "Developer" (correct case)
		tasks, err = db.GetUnassignedTasksByRole("Developer")
		assert.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Developer", tasks[0].Role)
	})
}

func TestGetTasksByRole(t *testing.T) {
	t.Run("successful retrieval of tasks by role", func(t *testing.T) {
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

		// Create tasks with "developer" role (both assigned and unassigned)
		task1 := &Task{
			Title:       "Unassigned Task 1",
			Description: "First unassigned task",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task1, "author123", "")
		require.NoError(t, err)

		task2 := &Task{
			Title:       "Assigned Task",
			Description: "Assigned task",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task2, "author123", "assignee456")
		require.NoError(t, err)

		task3 := &Task{
			Title:       "Unassigned Task 2",
			Description: "Second unassigned task",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task3, "author123", "")
		require.NoError(t, err)

		// Create task with different role
		task4 := &Task{
			Title:       "Designer Task",
			Description: "Designer task",
			Role:        "designer",
		}
		err = db.CreateTaskWithUserDiscordID(task4, "author123", "")
		require.NoError(t, err)

		// Retrieve all tasks for "developer" role
		tasks, err := db.GetTasksByRole("developer")
		assert.NoError(t, err)
		assert.Len(t, tasks, 3)

		// Verify the tasks are the correct ones
		taskIDs := make(map[uint]bool)
		for _, task := range tasks {
			taskIDs[task.ID] = true
			assert.Equal(t, "developer", task.Role)
			assert.Equal(t, author.Username, task.Author.Username)
		}
		assert.True(t, taskIDs[task1.ID])
		assert.True(t, taskIDs[task2.ID])
		assert.True(t, taskIDs[task3.ID])
		assert.False(t, taskIDs[task4.ID]) // Should not be included (different role)
	})

	t.Run("no tasks for role", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create users
		author := &User{
			Username:  "Author",
			DiscordID: "author123",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)

		// Create tasks with different role
		task1 := &Task{
			Title:       "Designer Task",
			Description: "Designer task",
			Role:        "designer",
		}
		err = db.CreateTaskWithUserDiscordID(task1, "author123", "")
		require.NoError(t, err)

		// Retrieve tasks for "developer" role
		tasks, err := db.GetTasksByRole("developer")
		assert.NoError(t, err)
		assert.Empty(t, tasks)
	})

	t.Run("empty role should return error", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Try to retrieve tasks with empty role
		tasks, err := db.GetTasksByRole("")
		assert.Error(t, err)
		assert.Nil(t, tasks)
		assert.Contains(t, err.Error(), "role cannot be empty")
	})

	t.Run("mixed roles with various assignment statuses", func(t *testing.T) {
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

		// Create tasks with different roles and assignment status
		tasks := []struct {
			title       string
			description string
			role        string
			assigneeID  string
		}{
			{"Dev Unassigned 1", "Developer unassigned task", "developer", ""},
			{"Dev Assigned", "Developer assigned task", "developer", "assignee456"},
			{"Dev Unassigned 2", "Developer unassigned task", "developer", ""},
			{"Designer Unassigned", "Designer unassigned task", "designer", ""},
			{"Designer Assigned", "Designer assigned task", "designer", "assignee456"},
			{"QA Unassigned", "QA unassigned task", "qa", ""},
		}

		createdTasks := make([]*Task, len(tasks))
		for i, taskData := range tasks {
			task := &Task{
				Title:       taskData.title,
				Description: taskData.description,
				Role:        taskData.role,
			}
			err = db.CreateTaskWithUserDiscordID(task, "author123", taskData.assigneeID)
			require.NoError(t, err)
			createdTasks[i] = task
		}

		// Test developer role (should get all 3 developer tasks)
		devTasks, err := db.GetTasksByRole("developer")
		assert.NoError(t, err)
		assert.Len(t, devTasks, 3)
		for _, task := range devTasks {
			assert.Equal(t, "developer", task.Role)
		}

		// Test designer role (should get all 2 designer tasks)
		designerTasks, err := db.GetTasksByRole("designer")
		assert.NoError(t, err)
		assert.Len(t, designerTasks, 2)
		for _, task := range designerTasks {
			assert.Equal(t, "designer", task.Role)
		}

		// Test QA role (should get 1 QA task)
		qaTasks, err := db.GetTasksByRole("qa")
		assert.NoError(t, err)
		assert.Len(t, qaTasks, 1)
		assert.Equal(t, "qa", qaTasks[0].Role)
	})

	t.Run("case sensitive role matching", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create users
		author := &User{
			Username:  "Author",
			DiscordID: "author123",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)

		// Create task with "Developer" role (capitalized)
		task := &Task{
			Title:       "Developer Task",
			Description: "Developer task",
			Role:        "Developer",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "")
		require.NoError(t, err)

		// Try to retrieve with "developer" (lowercase)
		tasks, err := db.GetTasksByRole("developer")
		assert.NoError(t, err)
		assert.Empty(t, tasks) // Should not match due to case sensitivity

		// Try to retrieve with "Developer" (correct case)
		tasks, err = db.GetTasksByRole("Developer")
		assert.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Developer", tasks[0].Role)
	})

	t.Run("tasks with preloaded relationships", func(t *testing.T) {
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

		// Create assigned task
		task := &Task{
			Title:       "Assigned Task",
			Description: "Task with assignee",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "assignee456")
		require.NoError(t, err)

		// Retrieve tasks and verify relationships are loaded
		tasks, err := db.GetTasksByRole("developer")
		assert.NoError(t, err)
		assert.Len(t, tasks, 1)

		retrievedTask := tasks[0]
		assert.Equal(t, task.ID, retrievedTask.ID)
		assert.Equal(t, author.Username, retrievedTask.Author.Username)
		assert.Len(t, retrievedTask.AssignedUsers, 1)
		assert.Equal(t, assignee.Username, retrievedTask.AssignedUsers[0].Username)
		assert.Equal(t, assignee.ID, retrievedTask.AssignedUsers[0].ID)
	})
}

func TestAssignTask(t *testing.T) {
	t.Run("successful task assignment", func(t *testing.T) {
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

		// Create task without assignee
		task := &Task{
			Title:       "Unassigned Task",
			Description: "Task to be assigned",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "")
		require.NoError(t, err)

		// Verify task is initially unassigned
		retrievedTask, err := db.GetTaskByID(task.ID)
		require.NoError(t, err)
		assert.Len(t, retrievedTask.AssignedUsers, 0)

		// Assign task to user
		err = db.AssignTask(task.ID, assignee.ID)
		assert.NoError(t, err)

		// Verify task is now assigned
		retrievedTask, err = db.GetTaskByID(task.ID)
		require.NoError(t, err)
		assert.Len(t, retrievedTask.AssignedUsers, 1)
		assert.Equal(t, assignee.ID, retrievedTask.AssignedUsers[0].ID)
		assert.Equal(t, assignee.Username, retrievedTask.AssignedUsers[0].Username)
	})

	t.Run("assign task that is already assigned", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create users
		author := &User{
			Username:  "Author",
			DiscordID: "author123",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)

		assignee1 := &User{
			Username:  "Assignee1",
			DiscordID: "assignee1",
		}
		err = db.CreateUser(assignee1)
		require.NoError(t, err)

		assignee2 := &User{
			Username:  "Assignee2",
			DiscordID: "assignee2",
		}
		err = db.CreateUser(assignee2)
		require.NoError(t, err)

		// Create task assigned to first user
		task := &Task{
			Title:       "Assigned Task",
			Description: "Task already assigned",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "assignee1")
		require.NoError(t, err)

		// Verify task is initially assigned to first user
		retrievedTask, err := db.GetTaskByID(task.ID)
		require.NoError(t, err)
		assert.Len(t, retrievedTask.AssignedUsers, 1)
		assert.Equal(t, assignee1.ID, retrievedTask.AssignedUsers[0].ID)

		// Reassign task to second user
		err = db.AssignTask(task.ID, assignee2.ID)
		assert.NoError(t, err)

		// Verify task is now assigned to second user too
		retrievedTask, err = db.GetTaskByID(task.ID)
		require.NoError(t, err)
		assert.Len(t, retrievedTask.AssignedUsers, 2)
		assert.Equal(t, assignee2.ID, retrievedTask.AssignedUsers[1].ID)
		assert.Equal(t, assignee2.Username, retrievedTask.AssignedUsers[1].Username)
	})

	t.Run("assign task to non-existent user ID", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create author
		author := &User{
			Username:  "Author",
			DiscordID: "author123",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)

		// Create task without assignee
		task := &Task{
			Title:       "Unassigned Task",
			Description: "Task to be assigned",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "")
		require.NoError(t, err)

		// Try to assign task to non-existent user ID
		err = db.AssignTask(task.ID, 999)
		assert.Error(t, err)

		// Verify task is not assigned
		retrievedTask, err := db.GetTaskByID(task.ID)
		require.NoError(t, err)
		assert.Len(t, retrievedTask.AssignedUsers, 0)
	})

	t.Run("assign non-existent task", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create user
		user := &User{
			Username:  "User",
			DiscordID: "user123",
		}
		err := db.CreateUser(user)
		require.NoError(t, err)

		// Try to assign non-existent task
		err = db.AssignTask(999, user.ID)
		assert.NoError(t, err) // The function doesn't validate task existence
	})

	t.Run("assign task with zero user ID", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create author
		author := &User{
			Username:  "Author",
			DiscordID: "author123",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)

		// Create task without assignee
		task := &Task{
			Title:       "Unassigned Task",
			Description: "Task to be assigned",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "")
		require.NoError(t, err)

		// Try to assign task with zero user ID
		err = db.AssignTask(task.ID, 0)
		assert.Error(t, err)

		// Verify task is not assigned
		retrievedTask, err := db.GetTaskByID(task.ID)
		require.NoError(t, err)
		assert.Len(t, retrievedTask.AssignedUsers, 0)
	})
}

func TestUpdateTaskStatus(t *testing.T) {
	t.Run("successful status update to Not Started", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create user
		author := &User{
			Username:  "Author",
			DiscordID: "author123",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)

		// Create task with default status
		task := &Task{
			Title:       "Test Task",
			Description: "Test Description",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "")
		require.NoError(t, err)

		// Update status to "Not Started"
		updatedTask, err := db.UpdateTaskStatus(task.ID, TASK_NOT_STARTED)
		assert.NoError(t, err)
		assert.NotNil(t, updatedTask)
		assert.Equal(t, TASK_NOT_STARTED, updatedTask.Status)

		// Verify the update in database
		retrievedTask, err := db.GetTaskByID(task.ID)
		require.NoError(t, err)
		assert.Equal(t, TASK_NOT_STARTED, retrievedTask.Status)
	})

	t.Run("successful status update to In Progress", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create user
		author := &User{
			Username:  "Author",
			DiscordID: "author123",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)

		// Create task
		task := &Task{
			Title:       "Test Task",
			Description: "Test Description",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "")
		require.NoError(t, err)

		// Update status to "In Progress"
		updatedTask, err := db.UpdateTaskStatus(task.ID, TASK_IN_PROGRESS)
		assert.NoError(t, err)
		assert.NotNil(t, updatedTask)
		assert.Equal(t, TASK_IN_PROGRESS, updatedTask.Status)

		// Verify the update in database
		retrievedTask, err := db.GetTaskByID(task.ID)
		require.NoError(t, err)
		assert.Equal(t, TASK_IN_PROGRESS, retrievedTask.Status)
	})

	t.Run("successful status update to Completed", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create user
		author := &User{
			Username:  "Author",
			DiscordID: "author123",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)

		// Create task
		task := &Task{
			Title:       "Test Task",
			Description: "Test Description",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "")
		require.NoError(t, err)

		// Update status to "Completed"
		updatedTask, err := db.UpdateTaskStatus(task.ID, TASK_COMPLETED)
		assert.NoError(t, err)
		assert.NotNil(t, updatedTask)
		assert.Equal(t, TASK_COMPLETED, updatedTask.Status)

		// Verify the update in database
		retrievedTask, err := db.GetTaskByID(task.ID)
		require.NoError(t, err)
		assert.Equal(t, TASK_COMPLETED, retrievedTask.Status)
	})

	t.Run("status update preserves other task fields", func(t *testing.T) {
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

		// Create task with all fields populated
		task := &Task{
			Title:       "Complex Task",
			Description: "Complex Description",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "assignee456")
		require.NoError(t, err)
		t.Logf("task: %+v", task)

		// Update status
		updatedTask, err := db.UpdateTaskStatus(task.ID, TASK_IN_PROGRESS)
		assert.NoError(t, err)
		assert.NotNil(t, updatedTask)
		t.Logf("updatedTask: %+v", updatedTask)

		// Verify all other fields are preserved
		assert.Equal(t, task.Title, updatedTask.Title)
		assert.Equal(t, task.Description, updatedTask.Description)
		assert.Equal(t, task.Role, updatedTask.Role)
		assert.Equal(t, task.AuthorID, updatedTask.AuthorID)
		assert.Len(t, updatedTask.AssignedUsers, 1)
		assert.Equal(t, assignee.ID, updatedTask.AssignedUsers[0].ID)
		assert.Equal(t, TASK_IN_PROGRESS, updatedTask.Status)
	})

	t.Run("invalid status should return error", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create user
		author := &User{
			Username:  "Author",
			DiscordID: "author123",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)

		// Create task
		task := &Task{
			Title:       "Test Task",
			Description: "Test Description",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "")
		require.NoError(t, err)

		// Try to update with invalid status
		updatedTask, err := db.UpdateTaskStatus(task.ID, "Invalid Status")
		assert.Error(t, err)
		assert.Nil(t, updatedTask)
		assert.Contains(t, err.Error(), "invalid status")
		assert.Contains(t, err.Error(), TASK_NOT_STARTED)
		assert.Contains(t, err.Error(), TASK_IN_PROGRESS)
		assert.Contains(t, err.Error(), TASK_COMPLETED)

		// Verify task status was not changed
		retrievedTask, err := db.GetTaskByID(task.ID)
		require.NoError(t, err)
		assert.Equal(t, TASK_NOT_STARTED, retrievedTask.Status) // Default status
	})

	t.Run("case sensitive status validation", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create user
		author := &User{
			Username:  "Author",
			DiscordID: "author123",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)

		// Create task
		task := &Task{
			Title:       "Test Task",
			Description: "Test Description",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "")
		require.NoError(t, err)

		// Try to update with lowercase status
		updatedTask, err := db.UpdateTaskStatus(task.ID, "not started")
		assert.Error(t, err)
		assert.Nil(t, updatedTask)
		assert.Contains(t, err.Error(), "invalid status")

		// Try to update with mixed case status
		updatedTask, err = db.UpdateTaskStatus(task.ID, "in progress")
		assert.Error(t, err)
		assert.Nil(t, updatedTask)
		assert.Contains(t, err.Error(), "invalid status")

		// Verify task status was not changed
		retrievedTask, err := db.GetTaskByID(task.ID)
		require.NoError(t, err)
		assert.Equal(t, TASK_NOT_STARTED, retrievedTask.Status) // Default status
	})

	t.Run("non-existent task should return empty task object", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Try to update non-existent task
		updatedTask, err := db.UpdateTaskStatus(999, TASK_IN_PROGRESS)
		assert.Error(t, err)
		assert.Nil(t, updatedTask)
	})

	t.Run("zero task ID should return empty task object", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Try to update task with zero ID
		updatedTask, err := db.UpdateTaskStatus(0, TASK_IN_PROGRESS)
		assert.Error(t, err)
		assert.Nil(t, updatedTask)
	})

	t.Run("empty status should return error", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create user
		author := &User{
			Username:  "Author",
			DiscordID: "author123",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)

		// Create task
		task := &Task{
			Title:       "Test Task",
			Description: "Test Description",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "")
		require.NoError(t, err)

		// Try to update with empty status
		updatedTask, err := db.UpdateTaskStatus(task.ID, "")
		assert.Error(t, err)
		assert.Nil(t, updatedTask)
		assert.Contains(t, err.Error(), "invalid status")
	})

	t.Run("multiple status updates on same task", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create user
		author := &User{
			Username:  "Author",
			DiscordID: "author123",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)

		// Create task
		task := &Task{
			Title:       "Test Task",
			Description: "Test Description",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "")
		require.NoError(t, err)

		// Update status multiple times
		updatedTask, err := db.UpdateTaskStatus(task.ID, TASK_IN_PROGRESS)
		assert.NoError(t, err)
		assert.Equal(t, TASK_IN_PROGRESS, updatedTask.Status)

		updatedTask, err = db.UpdateTaskStatus(task.ID, TASK_COMPLETED)
		assert.NoError(t, err)
		assert.Equal(t, TASK_COMPLETED, updatedTask.Status)

		updatedTask, err = db.UpdateTaskStatus(task.ID, TASK_NOT_STARTED)
		assert.NoError(t, err)
		assert.Equal(t, TASK_NOT_STARTED, updatedTask.Status)

		// Verify final status in database
		retrievedTask, err := db.GetTaskByID(task.ID)
		require.NoError(t, err)
		assert.Equal(t, TASK_NOT_STARTED, retrievedTask.Status)
	})
}

func TestGetCompletedTasksByRole(t *testing.T) {
	t.Run("successful retrieval of completed tasks by role", func(t *testing.T) {
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

		// Create completed tasks with "developer" role
		task1 := &Task{
			Title:       "Completed Task 1",
			Description: "First completed task",
			Role:        "developer",
			Status:      TASK_COMPLETED,
		}
		err = db.CreateTaskWithUserDiscordID(task1, "author123", "assignee456")
		require.NoError(t, err)
		// Update status to completed
		_, err = db.UpdateTaskStatus(task1.ID, TASK_COMPLETED)
		require.NoError(t, err)

		task2 := &Task{
			Title:       "Completed Task 2",
			Description: "Second completed task",
			Role:        "developer",
			Status:      TASK_COMPLETED,
		}
		err = db.CreateTaskWithUserDiscordID(task2, "author123", "assignee456")
		require.NoError(t, err)
		// Update status to completed
		_, err = db.UpdateTaskStatus(task2.ID, TASK_COMPLETED)
		require.NoError(t, err)

		// Create non-completed task with "developer" role
		task3 := &Task{
			Title:       "In Progress Task",
			Description: "Task in progress",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task3, "author123", "assignee456")
		require.NoError(t, err)
		// Update status to in progress
		_, err = db.UpdateTaskStatus(task3.ID, TASK_IN_PROGRESS)
		require.NoError(t, err)

		// Create completed task with different role
		task4 := &Task{
			Title:       "Designer Completed Task",
			Description: "Designer completed task",
			Role:        "designer",
		}
		err = db.CreateTaskWithUserDiscordID(task4, "author123", "assignee456")
		require.NoError(t, err)
		// Update status to completed
		_, err = db.UpdateTaskStatus(task4.ID, TASK_COMPLETED)
		require.NoError(t, err)

		// Retrieve completed tasks for "developer" role
		tasks, err := db.GetCompletedTasksByRole("developer")
		assert.NoError(t, err)
		assert.Len(t, tasks, 2)

		// Verify the tasks are the correct completed ones
		taskIDs := make(map[uint]bool)
		for _, task := range tasks {
			taskIDs[task.ID] = true
			assert.Equal(t, "developer", task.Role)
			assert.Equal(t, TASK_COMPLETED, task.Status)
			assert.Equal(t, author.Username, task.Author.Username)
		}
		assert.True(t, taskIDs[task1.ID])
		assert.True(t, taskIDs[task2.ID])
		assert.False(t, taskIDs[task3.ID]) // Should not be included (not completed)
		assert.False(t, taskIDs[task4.ID]) // Should not be included (different role)
	})

	t.Run("no completed tasks for role", func(t *testing.T) {
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

		// Create only non-completed tasks with "developer" role
		task1 := &Task{
			Title:       "In Progress Task 1",
			Description: "First in progress task",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task1, "author123", "assignee456")
		require.NoError(t, err)
		_, err = db.UpdateTaskStatus(task1.ID, TASK_IN_PROGRESS)
		require.NoError(t, err)

		task2 := &Task{
			Title:       "Not Started Task",
			Description: "Not started task",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task2, "author123", "assignee456")
		require.NoError(t, err)
		// Keep default status (Not Started)

		// Retrieve completed tasks for "developer" role
		tasks, err := db.GetCompletedTasksByRole("developer")
		assert.NoError(t, err)
		assert.Empty(t, tasks)
	})

	t.Run("no tasks for role at all", func(t *testing.T) {
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

		// Create completed tasks with different role
		task1 := &Task{
			Title:       "Designer Completed Task",
			Description: "Designer completed task",
			Role:        "designer",
		}
		err = db.CreateTaskWithUserDiscordID(task1, "author123", "assignee456")
		require.NoError(t, err)
		_, err = db.UpdateTaskStatus(task1.ID, TASK_COMPLETED)
		require.NoError(t, err)

		// Retrieve completed tasks for "developer" role
		tasks, err := db.GetCompletedTasksByRole("developer")
		assert.NoError(t, err)
		assert.Empty(t, tasks)
	})

	t.Run("empty role should return error", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Try to retrieve tasks with empty role
		tasks, err := db.GetCompletedTasksByRole("")
		assert.Error(t, err)
		assert.Nil(t, tasks)
		assert.Contains(t, err.Error(), "role cannot be empty")
	})

	t.Run("mixed roles with completed tasks", func(t *testing.T) {
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

		// Create tasks with different roles and completion status
		tasks := []struct {
			title       string
			description string
			role        string
			status      string
		}{
			{"Dev Completed 1", "Developer completed task", "developer", TASK_COMPLETED},
			{"Dev Completed 2", "Developer completed task", "developer", TASK_COMPLETED},
			{"Dev In Progress", "Developer in progress task", "developer", TASK_IN_PROGRESS},
			{"Designer Completed", "Designer completed task", "designer", TASK_COMPLETED},
			{"Designer In Progress", "Designer in progress task", "designer", TASK_IN_PROGRESS},
			{"QA Completed", "QA completed task", "qa", TASK_COMPLETED},
		}

		createdTasks := make([]*Task, len(tasks))
		for i, taskData := range tasks {
			task := &Task{
				Title:       taskData.title,
				Description: taskData.description,
				Role:        taskData.role,
			}
			err = db.CreateTaskWithUserDiscordID(task, "author123", "assignee456")
			require.NoError(t, err)
			_, err = db.UpdateTaskStatus(task.ID, taskData.status)
			require.NoError(t, err)
			createdTasks[i] = task
		}

		// Test developer role (should get 2 completed tasks)
		devTasks, err := db.GetCompletedTasksByRole("developer")
		assert.NoError(t, err)
		assert.Len(t, devTasks, 2)
		for _, task := range devTasks {
			assert.Equal(t, "developer", task.Role)
			assert.Equal(t, TASK_COMPLETED, task.Status)
		}

		// Test designer role (should get 1 completed task)
		designerTasks, err := db.GetCompletedTasksByRole("designer")
		assert.NoError(t, err)
		assert.Len(t, designerTasks, 1)
		assert.Equal(t, "designer", designerTasks[0].Role)
		assert.Equal(t, TASK_COMPLETED, designerTasks[0].Status)

		// Test QA role (should get 1 completed task)
		qaTasks, err := db.GetCompletedTasksByRole("qa")
		assert.NoError(t, err)
		assert.Len(t, qaTasks, 1)
		assert.Equal(t, "qa", qaTasks[0].Role)
		assert.Equal(t, TASK_COMPLETED, qaTasks[0].Status)
	})

	t.Run("case sensitive role matching", func(t *testing.T) {
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

		// Create completed task with "Developer" role (capitalized)
		task := &Task{
			Title:       "Developer Completed Task",
			Description: "Developer completed task",
			Role:        "Developer",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "assignee456")
		require.NoError(t, err)
		_, err = db.UpdateTaskStatus(task.ID, TASK_COMPLETED)
		require.NoError(t, err)

		// Try to retrieve with "developer" (lowercase)
		tasks, err := db.GetCompletedTasksByRole("developer")
		assert.NoError(t, err)
		assert.Empty(t, tasks) // Should not match due to case sensitivity

		// Try to retrieve with "Developer" (correct case)
		tasks, err = db.GetCompletedTasksByRole("Developer")
		assert.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Developer", tasks[0].Role)
		assert.Equal(t, TASK_COMPLETED, tasks[0].Status)
	})

	t.Run("completed tasks with preloaded relationships", func(t *testing.T) {
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

		// Create completed task
		task := &Task{
			Title:       "Completed Task with Assignee",
			Description: "Task with assignee",
			Role:        "developer",
		}
		err = db.CreateTaskWithUserDiscordID(task, "author123", "assignee456")
		require.NoError(t, err)
		_, err = db.UpdateTaskStatus(task.ID, TASK_COMPLETED)
		require.NoError(t, err)

		// Retrieve completed tasks and verify relationships are loaded
		tasks, err := db.GetCompletedTasksByRole("developer")
		assert.NoError(t, err)
		assert.Len(t, tasks, 1)

		retrievedTask := tasks[0]
		assert.Equal(t, task.ID, retrievedTask.ID)
		assert.Equal(t, author.Username, retrievedTask.Author.Username)
		assert.Len(t, retrievedTask.AssignedUsers, 1)
		assert.Equal(t, assignee.Username, retrievedTask.AssignedUsers[0].Username)
		assert.Equal(t, assignee.ID, retrievedTask.AssignedUsers[0].ID)
		assert.Equal(t, TASK_COMPLETED, retrievedTask.Status)
	})
}

func TestGetWebhookSubscriptionsByRepository(t *testing.T) {
	t.Run("successful retrieval of webhook subscriptions by repository", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create repositories first
		repo1 := &Repository{Name: "test-repo"}
		err := gormDB.Create(repo1).Error
		require.NoError(t, err)

		repo2 := &Repository{Name: "other-repo"}
		err = gormDB.Create(repo2).Error
		require.NoError(t, err)

		// Create webhook subscriptions for the same repository
		_, err = db.CreateWebhookSubscription("test-repo", "channel123")
		require.NoError(t, err)

		_, err = db.CreateWebhookSubscription("test-repo", "channel456")
		require.NoError(t, err)

		// Create subscription for different repository
		_, err = db.CreateWebhookSubscription("other-repo", "channel789")
		require.NoError(t, err)

		// Retrieve subscriptions for "test-repo"
		subscriptions, err := db.GetWebhookSubscriptionsByRepository("test-repo")
		assert.NoError(t, err)
		assert.Len(t, subscriptions, 2)

		// Verify the subscriptions are the correct ones
		channelIDs := make(map[string]bool)
		for _, sub := range subscriptions {
			channelIDs[sub.ChannelID] = true
			assert.Equal(t, "test-repo", sub.Repository.Name)
			assert.NotZero(t, sub.ID)
		}
		assert.True(t, channelIDs["channel123"])
		assert.True(t, channelIDs["channel456"])
		assert.False(t, channelIDs["channel789"]) // Should not be included (different repo)
	})

	t.Run("no subscriptions for repository", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create repository and subscription
		repo := &Repository{Name: "existing-repo"}
		err := gormDB.Create(repo).Error
		require.NoError(t, err)

		_, err = db.CreateWebhookSubscription("existing-repo", "channel123")
		require.NoError(t, err)

		// Retrieve subscriptions for non-existent repository
		subscriptions, err := db.GetWebhookSubscriptionsByRepository("nonexistent-repo")
		assert.NoError(t, err)
		assert.Empty(t, subscriptions)
	})

	t.Run("empty repository name", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Try to retrieve subscriptions with empty repository name
		subscriptions, err := db.GetWebhookSubscriptionsByRepository("")
		assert.NoError(t, err)
		assert.Empty(t, subscriptions)
	})

	t.Run("case sensitive repository matching", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create repository with "Test-Repo" (capitalized)
		repo := &Repository{Name: "Test-Repo"}
		err := gormDB.Create(repo).Error
		require.NoError(t, err)

		// Create subscription
		_, err = db.CreateWebhookSubscription("Test-Repo", "channel123")
		require.NoError(t, err)

		// Try to retrieve with "test-repo" (lowercase)
		subscriptions, err := db.GetWebhookSubscriptionsByRepository("test-repo")
		assert.NoError(t, err)
		assert.Empty(t, subscriptions) // Should not match due to case sensitivity

		// Try to retrieve with "Test-Repo" (correct case)
		subscriptions, err = db.GetWebhookSubscriptionsByRepository("Test-Repo")
		assert.NoError(t, err)
		assert.Len(t, subscriptions, 1)
		assert.Equal(t, "Test-Repo", subscriptions[0].Repository.Name)
		assert.Equal(t, "channel123", subscriptions[0].ChannelID)
	})

	t.Run("multiple repositories with subscriptions", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create repositories first
		repos := []string{"repo1", "repo2", "repo3"}
		for _, repoName := range repos {
			repo := &Repository{Name: repoName}
			err := gormDB.Create(repo).Error
			require.NoError(t, err)
		}

		// Create subscriptions for different repositories
		subscriptions := []struct {
			repository string
			channelID  string
		}{
			{"repo1", "channel1"},
			{"repo1", "channel2"},
			{"repo2", "channel3"},
			{"repo2", "channel4"},
			{"repo2", "channel5"},
			{"repo3", "channel6"},
		}

		for _, subData := range subscriptions {
			_, err := db.CreateWebhookSubscription(subData.repository, subData.channelID)
			require.NoError(t, err)
		}

		// Test repo1 (should get 2 subscriptions)
		repo1Subs, err := db.GetWebhookSubscriptionsByRepository("repo1")
		assert.NoError(t, err)
		assert.Len(t, repo1Subs, 2)
		for _, sub := range repo1Subs {
			assert.Equal(t, "repo1", sub.Repository.Name)
		}

		// Test repo2 (should get 3 subscriptions)
		repo2Subs, err := db.GetWebhookSubscriptionsByRepository("repo2")
		assert.NoError(t, err)
		assert.Len(t, repo2Subs, 3)
		for _, sub := range repo2Subs {
			assert.Equal(t, "repo2", sub.Repository.Name)
		}

		// Test repo3 (should get 1 subscription)
		repo3Subs, err := db.GetWebhookSubscriptionsByRepository("repo3")
		assert.NoError(t, err)
		assert.Len(t, repo3Subs, 1)
		assert.Equal(t, "repo3", repo3Subs[0].Repository.Name)
		assert.Equal(t, "channel6", repo3Subs[0].ChannelID)
	})
}

func TestCreateWebhookSubscription(t *testing.T) {
	t.Run("successful webhook subscription creation", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create repository first
		repo := &Repository{Name: "test-repo"}
		err := gormDB.Create(repo).Error
		require.NoError(t, err)

		subscription, err := db.CreateWebhookSubscription("test-repo", "channel123")
		assert.NoError(t, err)
		assert.NotZero(t, subscription.ID)
		assert.Equal(t, "channel123", subscription.ChannelID)
		assert.Equal(t, repo.ID, subscription.RepositoryID)

		// Verify subscription was saved to database
		dbSubscription := &WebhookSubscription{}
		err = gormDB.First(dbSubscription, subscription.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, subscription.ChannelID, dbSubscription.ChannelID)
		assert.Equal(t, subscription.RepositoryID, dbSubscription.RepositoryID)
	})

	t.Run("create multiple subscriptions for same repository", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create repository first
		repo := &Repository{Name: "test-repo"}
		err := gormDB.Create(repo).Error
		require.NoError(t, err)

		// Create first subscription
		subscription1, err := db.CreateWebhookSubscription("test-repo", "channel123")
		assert.NoError(t, err)
		assert.NotZero(t, subscription1.ID)

		// Create second subscription for same repository
		subscription2, err := db.CreateWebhookSubscription("test-repo", "channel456")
		assert.NoError(t, err)
		assert.NotZero(t, subscription2.ID)

		// Verify both subscriptions exist
		subscriptions, err := db.GetWebhookSubscriptionsByRepository("test-repo")
		assert.NoError(t, err)
		assert.Len(t, subscriptions, 2)
	})

	t.Run("create subscription for non-existent repository", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Try to create subscription for non-existent repository
		subscription, err := db.CreateWebhookSubscription("non-existent-repo", "channel123")
		assert.Error(t, err)
		assert.Nil(t, subscription)
		assert.Contains(t, err.Error(), "failed to get repository")
	})

	t.Run("create subscription with empty repository name", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Try to create subscription with empty repository name
		subscription, err := db.CreateWebhookSubscription("", "channel123")
		assert.Error(t, err)
		assert.Nil(t, subscription)
		assert.Contains(t, err.Error(), "failed to get repository")
	})

	t.Run("create subscription with special characters", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create repository with special characters
		repo := &Repository{Name: "test-repo/with-special-chars_123"}
		err := gormDB.Create(repo).Error
		require.NoError(t, err)

		subscription, err := db.CreateWebhookSubscription("test-repo/with-special-chars_123", "channel-with-special-chars_123")
		assert.NoError(t, err)
		assert.NotZero(t, subscription.ID)

		// Verify subscription was saved correctly
		dbSubscription := &WebhookSubscription{}
		err = gormDB.First(dbSubscription, subscription.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "channel-with-special-chars_123", dbSubscription.ChannelID)
		assert.Equal(t, repo.ID, dbSubscription.RepositoryID)
	})

	t.Run("create subscription with very long values", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		longRepoName := "very-long-repository-name-that-might-exceed-normal-lengths-and-test-the-database-limits"
		longChannel := "very-long-channel-id-that-might-exceed-normal-lengths-and-test-the-database-limits"

		// Create repository with long name
		repo := &Repository{Name: longRepoName}
		err := gormDB.Create(repo).Error
		require.NoError(t, err)

		subscription, err := db.CreateWebhookSubscription(longRepoName, longChannel)
		assert.NoError(t, err)
		assert.NotZero(t, subscription.ID)

		// Verify subscription was saved correctly
		dbSubscription := &WebhookSubscription{}
		err = gormDB.First(dbSubscription, subscription.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, longChannel, dbSubscription.ChannelID)
		assert.Equal(t, repo.ID, dbSubscription.RepositoryID)
	})
}

func TestDeleteWebhookSubscription(t *testing.T) {
	t.Run("successful webhook subscription deletion", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create repository first
		repo := &Repository{Name: "test-repo"}
		err := gormDB.Create(repo).Error
		require.NoError(t, err)

		// Create subscription
		subscription, err := db.CreateWebhookSubscription("test-repo", "channel123")
		require.NoError(t, err)
		require.NotZero(t, subscription.ID)

		// Verify subscription exists
		subscriptions, err := db.GetWebhookSubscriptionsByRepository("test-repo")
		assert.NoError(t, err)
		assert.Len(t, subscriptions, 1)

		// Delete subscription
		err = db.DeleteWebhookSubscription("test-repo", "channel123")
		assert.NoError(t, err)

		// Verify subscription was deleted
		subscriptions, err = db.GetWebhookSubscriptionsByRepository("test-repo")
		assert.NoError(t, err)
		assert.Empty(t, subscriptions)
	})

	t.Run("delete subscription that doesn't exist", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create repository and subscription
		repo := &Repository{Name: "test-repo"}
		err := gormDB.Create(repo).Error
		require.NoError(t, err)

		subscription1, err := db.CreateWebhookSubscription("test-repo", "channel123")
		require.NoError(t, err)

		// Try to delete a non-existent subscription
		err = db.DeleteWebhookSubscription("test-repo", "channel456")
		assert.Error(t, err)

		// Verify original subscription still exists
		subscriptions, err := db.GetWebhookSubscriptionsByRepository("test-repo")
		assert.NoError(t, err)
		assert.Len(t, subscriptions, 1)
		assert.Equal(t, subscription1.ID, subscriptions[0].ID)
	})

	t.Run("delete one subscription from multiple", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create repository first
		repo := &Repository{Name: "test-repo"}
		err := gormDB.Create(repo).Error
		require.NoError(t, err)

		// Create multiple subscriptions for same repository
		_, err = db.CreateWebhookSubscription("test-repo", "channel123")
		require.NoError(t, err)

		_, err = db.CreateWebhookSubscription("test-repo", "channel456")
		require.NoError(t, err)

		_, err = db.CreateWebhookSubscription("test-repo", "channel789")
		require.NoError(t, err)

		// Verify all subscriptions exist
		subscriptions, err := db.GetWebhookSubscriptionsByRepository("test-repo")
		assert.NoError(t, err)
		assert.Len(t, subscriptions, 3)

		// Delete only one subscription
		err = db.DeleteWebhookSubscription("test-repo", "channel456")
		assert.NoError(t, err)

		// Verify only the deleted subscription was removed
		subscriptions, err = db.GetWebhookSubscriptionsByRepository("test-repo")
		assert.NoError(t, err)
		assert.Len(t, subscriptions, 2)

		// Verify the remaining subscriptions are the correct ones
		channelIDs := make(map[string]bool)
		for _, sub := range subscriptions {
			channelIDs[sub.ChannelID] = true
		}
		assert.True(t, channelIDs["channel123"])
		assert.False(t, channelIDs["channel456"]) // Should be deleted
		assert.True(t, channelIDs["channel789"])
	})

	t.Run("delete subscription from different repository", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create repositories first
		repo1 := &Repository{Name: "repo1"}
		err := gormDB.Create(repo1).Error
		require.NoError(t, err)

		repo2 := &Repository{Name: "repo2"}
		err = gormDB.Create(repo2).Error
		require.NoError(t, err)

		// Create subscriptions for different repositories
		_, err = db.CreateWebhookSubscription("repo1", "channel123")
		require.NoError(t, err)

		subscription2, err := db.CreateWebhookSubscription("repo2", "channel456")
		require.NoError(t, err)

		// Delete subscription from repo1
		err = db.DeleteWebhookSubscription("repo1", "channel123")
		assert.NoError(t, err)

		// Verify repo1 has no subscriptions
		repo1Subs, err := db.GetWebhookSubscriptionsByRepository("repo1")
		assert.NoError(t, err)
		assert.Empty(t, repo1Subs)

		// Verify repo2 still has its subscription
		repo2Subs, err := db.GetWebhookSubscriptionsByRepository("repo2")
		assert.NoError(t, err)
		assert.Len(t, repo2Subs, 1)
		assert.Equal(t, subscription2.ID, repo2Subs[0].ID)
	})

	t.Run("delete and recreate subscription", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create repository first
		repo := &Repository{Name: "test-repo"}
		err := gormDB.Create(repo).Error
		require.NoError(t, err)

		// Create subscription
		subscription, err := db.CreateWebhookSubscription("test-repo", "channel123")
		require.NoError(t, err)
		originalID := subscription.ID

		// Delete subscription
		err = db.DeleteWebhookSubscription("test-repo", "channel123")
		assert.NoError(t, err)

		// Verify subscription was deleted
		subscriptions, err := db.GetWebhookSubscriptionsByRepository("test-repo")
		assert.NoError(t, err)
		assert.Empty(t, subscriptions)

		// Recreate subscription with same data
		newSubscription, err := db.CreateWebhookSubscription("test-repo", "channel123")
		assert.NoError(t, err)
		assert.NotZero(t, newSubscription.ID)
		assert.NotEqual(t, originalID, newSubscription.ID) // Should have different ID

		// Verify new subscription exists
		subscriptions, err = db.GetWebhookSubscriptionsByRepository("test-repo")
		assert.NoError(t, err)
		assert.Len(t, subscriptions, 1)
		assert.Equal(t, newSubscription.ID, subscriptions[0].ID)
	})
}

func TestCreateRepository(t *testing.T) {
	t.Run("successful repository creation", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		repo := &Repository{
			Name: "test-repository",
		}

		err := db.CreateRepository(repo)
		assert.NoError(t, err)
		assert.NotZero(t, repo.ID)

		// Verify repository was saved to database
		dbRepo := &Repository{}
		err = gormDB.First(dbRepo, repo.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, repo.Name, dbRepo.Name)
	})

	t.Run("duplicate repository name should fail", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		repo1 := &Repository{
			Name: "test-repository",
		}
		err := db.CreateRepository(repo1)
		assert.NoError(t, err)

		repo2 := &Repository{
			Name: "test-repository", // Same name
		}
		err = db.CreateRepository(repo2)
		assert.Error(t, err)
	})

	t.Run("create repository with special characters", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		repo := &Repository{
			Name: "test-repo/with-special-chars_123",
		}

		err := db.CreateRepository(repo)
		assert.NoError(t, err)
		assert.NotZero(t, repo.ID)

		// Verify repository was saved correctly
		dbRepo := &Repository{}
		err = gormDB.First(dbRepo, repo.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "test-repo/with-special-chars_123", dbRepo.Name)
	})

	t.Run("create repository with very long name", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		longName := "very-long-repository-name-that-might-exceed-normal-lengths-and-test-the-database-limits"
		repo := &Repository{
			Name: longName,
		}

		err := db.CreateRepository(repo)
		assert.NoError(t, err)
		assert.NotZero(t, repo.ID)

		// Verify repository was saved correctly
		dbRepo := &Repository{}
		err = gormDB.First(dbRepo, repo.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, longName, dbRepo.Name)
	})

	t.Run("create repository with empty name", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		repo := &Repository{
			Name: "",
		}

		err := db.CreateRepository(repo)
		assert.NoError(t, err) // The method doesn't validate empty names
		assert.NotZero(t, repo.ID)

		// Verify repository was saved
		dbRepo := &Repository{}
		err = gormDB.First(dbRepo, repo.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "", dbRepo.Name)
	})

	t.Run("create multiple repositories", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		repos := []*Repository{
			{Name: "repo1"},
			{Name: "repo2"},
			{Name: "repo3"},
		}

		for _, repo := range repos {
			err := db.CreateRepository(repo)
			assert.NoError(t, err)
			assert.NotZero(t, repo.ID)
		}

		// Verify all repositories were saved
		var count int64
		gormDB.Model(&Repository{}).Count(&count)
		assert.Equal(t, int64(3), count)
	})
}

func TestGetAllRepositories(t *testing.T) {
	t.Run("successful retrieval of all repositories", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create repositories
		repos := []*Repository{
			{Name: "repo-c"},
			{Name: "repo-a"},
			{Name: "repo-b"},
		}

		for _, repo := range repos {
			err := db.CreateRepository(repo)
			require.NoError(t, err)
		}

		// Retrieve all repositories
		retrievedRepos, err := db.GetAllRepositories()
		assert.NoError(t, err)
		assert.Len(t, retrievedRepos, 3)

		// Verify repositories are ordered by name
		assert.Equal(t, "repo-a", retrievedRepos[0].Name)
		assert.Equal(t, "repo-b", retrievedRepos[1].Name)
		assert.Equal(t, "repo-c", retrievedRepos[2].Name)
	})

	t.Run("no repositories in database", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Retrieve all repositories when none exist
		repos, err := db.GetAllRepositories()
		assert.NoError(t, err)
		assert.Empty(t, repos)
	})

	t.Run("single repository", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create single repository
		repo := &Repository{Name: "single-repo"}
		err := db.CreateRepository(repo)
		require.NoError(t, err)

		// Retrieve all repositories
		retrievedRepos, err := db.GetAllRepositories()
		assert.NoError(t, err)
		assert.Len(t, retrievedRepos, 1)
		assert.Equal(t, "single-repo", retrievedRepos[0].Name)
		assert.Equal(t, repo.ID, retrievedRepos[0].ID)
	})

	t.Run("repositories with special characters", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create repositories with special characters
		repos := []*Repository{
			{Name: "repo-with-dashes"},
			{Name: "repo_with_underscores"},
			{Name: "repo/with/slashes"},
			{Name: "repo.with.dots"},
		}

		for _, repo := range repos {
			err := db.CreateRepository(repo)
			require.NoError(t, err)
		}

		// Retrieve all repositories
		retrievedRepos, err := db.GetAllRepositories()
		assert.NoError(t, err)
		assert.Len(t, retrievedRepos, 4)

		// Verify all repositories are returned
		repoNames := make(map[string]bool)
		for _, repo := range retrievedRepos {
			repoNames[repo.Name] = true
		}
		assert.True(t, repoNames["repo-with-dashes"])
		assert.True(t, repoNames["repo_with_underscores"])
		assert.True(t, repoNames["repo/with/slashes"])
		assert.True(t, repoNames["repo.with.dots"])
	})

	t.Run("repositories with numbers", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create repositories with numbers
		repos := []*Repository{
			{Name: "repo1"},
			{Name: "repo10"},
			{Name: "repo2"},
		}

		for _, repo := range repos {
			err := db.CreateRepository(repo)
			require.NoError(t, err)
		}

		// Retrieve all repositories
		retrievedRepos, err := db.GetAllRepositories()
		assert.NoError(t, err)
		assert.Len(t, retrievedRepos, 3)

		// Verify repositories are ordered by name (lexicographic order)
		assert.Equal(t, "repo1", retrievedRepos[0].Name)
		assert.Equal(t, "repo10", retrievedRepos[1].Name)
		assert.Equal(t, "repo2", retrievedRepos[2].Name)
	})

	t.Run("repositories with mixed case", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create repositories with mixed case
		repos := []*Repository{
			{Name: "Repo-C"},
			{Name: "repo-a"},
			{Name: "REPO-B"},
		}

		for _, repo := range repos {
			err := db.CreateRepository(repo)
			require.NoError(t, err)
		}

		// Retrieve all repositories
		retrievedRepos, err := db.GetAllRepositories()
		assert.NoError(t, err)
		assert.Len(t, retrievedRepos, 3)

		// Verify repositories are ordered by name (case sensitive)
		assert.Equal(t, "REPO-B", retrievedRepos[0].Name)
		assert.Equal(t, "Repo-C", retrievedRepos[1].Name)
		assert.Equal(t, "repo-a", retrievedRepos[2].Name)
	})

	t.Run("large number of repositories", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create many repositories
		numRepos := 50
		for i := range numRepos {
			repo := &Repository{Name: fmt.Sprintf("repo-%03d", i)}
			err := db.CreateRepository(repo)
			require.NoError(t, err)
		}

		// Retrieve all repositories
		retrievedRepos, err := db.GetAllRepositories()
		assert.NoError(t, err)
		assert.Len(t, retrievedRepos, numRepos)

		// Verify repositories are ordered
		for i := range numRepos {
			expectedName := fmt.Sprintf("repo-%03d", i)
			assert.Equal(t, expectedName, retrievedRepos[i].Name)
		}
	})
}

func TestDeleteTask(t *testing.T) {
	t.Run("successful task deletion", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create a user first
		user := &User{
			Username:  "TestUser",
			DiscordID: "1234567890",
		}
		err := db.CreateUser(user)
		require.NoError(t, err)

		// Create a task
		task := &Task{
			Title:       "Test Task",
			Description: "Test Description",
			Role:        "Developer",
			Status:      TASK_NOT_STARTED,
			AuthorID:    user.ID,
		}
		err = gormDB.Create(task).Error
		require.NoError(t, err)
		require.NotZero(t, task.ID)

		// Delete the task
		err = db.DeleteTask(task.ID)
		assert.NoError(t, err)

		// Verify task was deleted
		var deletedTask Task
		err = gormDB.First(&deletedTask, task.ID).Error
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("delete non-existent task", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Try to delete a task that doesn't exist
		nonExistentID := uint(999)
		err := db.DeleteTask(nonExistentID)
		assert.NoError(t, err) // GORM doesn't return error for deleting non-existent records
	})

	t.Run("delete task with ID 0", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Try to delete a task with ID 0
		err := db.DeleteTask(0)
		assert.NoError(t, err) // GORM doesn't return error for deleting with ID 0
	})

	t.Run("delete task with assigned users", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create users
		author := &User{
			Username:  "Author",
			DiscordID: "1111111111",
		}
		assignee := &User{
			Username:  "Assignee",
			DiscordID: "2222222222",
		}
		err := db.CreateUser(author)
		require.NoError(t, err)
		err = db.CreateUser(assignee)
		require.NoError(t, err)

		// Create a task
		task := &Task{
			Title:       "Task with Assignee",
			Description: "Test Description",
			Role:        "Developer",
			Status:      TASK_NOT_STARTED,
			AuthorID:    author.ID,
		}
		err = gormDB.Create(task).Error
		require.NoError(t, err)

		// Assign the task to a user
		err = db.AssignTask(task.ID, assignee.ID)
		require.NoError(t, err)

		// Verify task has assigned user
		retrievedTask, err := db.GetTaskByID(task.ID)
		require.NoError(t, err)
		require.Len(t, retrievedTask.AssignedUsers, 1)
		assert.Equal(t, assignee.ID, retrievedTask.AssignedUsers[0].ID)

		// Delete the task
		err = db.DeleteTask(task.ID)
		assert.NoError(t, err)

		// Verify task was deleted
		var deletedTask Task
		err = gormDB.First(&deletedTask, task.ID).Error
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)

		// Verify assigned user still exists (cascade delete should not affect users)
		var userExists User
		err = gormDB.First(&userExists, assignee.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, assignee.Username, userExists.Username)
	})

	t.Run("delete task with comments", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create a user
		user := &User{
			Username:  "TestUser",
			DiscordID: "1234567890",
		}
		err := db.CreateUser(user)
		require.NoError(t, err)

		// Create a task
		task := &Task{
			Title:       "Task with Comments",
			Description: "Test Description",
			Role:        "Developer",
			Status:      TASK_NOT_STARTED,
			AuthorID:    user.ID,
		}
		err = gormDB.Create(task).Error
		require.NoError(t, err)

		// Add comments to the task
		comment1 := &TaskComment{
			Text:   "First comment",
			TaskID: task.ID,
		}
		comment2 := &TaskComment{
			Text:   "Second comment",
			TaskID: task.ID,
		}
		err = gormDB.Create(comment1).Error
		require.NoError(t, err)
		err = gormDB.Create(comment2).Error
		require.NoError(t, err)

		// Verify comments exist
		var comments []TaskComment
		err = gormDB.Where("task_id = ?", task.ID).Find(&comments).Error
		require.NoError(t, err)
		require.Len(t, comments, 2)

		// Delete the task
		err = db.DeleteTask(task.ID)
		assert.NoError(t, err)

		// Verify task was deleted
		var deletedTask Task
		err = gormDB.First(&deletedTask, task.ID).Error
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)

		// Verify comments still exist (no cascade delete configured)
		var remainingComments []TaskComment
		err = gormDB.Where("task_id = ?", task.ID).Find(&remainingComments).Error
		require.NoError(t, err)
		assert.Len(t, remainingComments, 2) // Comments remain as orphaned records
	})

	t.Run("delete multiple tasks", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Create a user
		user := &User{
			Username:  "TestUser",
			DiscordID: "1234567890",
		}
		err := db.CreateUser(user)
		require.NoError(t, err)

		// Create multiple tasks
		tasks := []*Task{
			{
				Title:       "Task 1",
				Description: "Description 1",
				Role:        "Developer",
				Status:      TASK_NOT_STARTED,
				AuthorID:    user.ID,
			},
			{
				Title:       "Task 2",
				Description: "Description 2",
				Role:        "Designer",
				Status:      TASK_IN_PROGRESS,
				AuthorID:    user.ID,
			},
			{
				Title:       "Task 3",
				Description: "Description 3",
				Role:        "Tester",
				Status:      TASK_COMPLETED,
				AuthorID:    user.ID,
			},
		}

		for _, task := range tasks {
			err = gormDB.Create(task).Error
			require.NoError(t, err)
		}

		// Verify all tasks exist
		var allTasks []Task
		err = gormDB.Find(&allTasks).Error
		require.NoError(t, err)
		assert.Len(t, allTasks, 3)

		// Delete the first task
		err = db.DeleteTask(tasks[0].ID)
		assert.NoError(t, err)

		// Verify only the first task was deleted
		var remainingTasks []Task
		err = gormDB.Find(&remainingTasks).Error
		require.NoError(t, err)
		assert.Len(t, remainingTasks, 2)

		// Verify the correct tasks remain
		remainingIDs := make([]uint, len(remainingTasks))
		for i, task := range remainingTasks {
			remainingIDs[i] = task.ID
		}
		assert.Contains(t, remainingIDs, tasks[1].ID)
		assert.Contains(t, remainingIDs, tasks[2].ID)
		assert.NotContains(t, remainingIDs, tasks[0].ID)
	})
}
