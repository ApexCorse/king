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
		assert.True(t, task.AssignedUserID.Valid)
		assert.Equal(t, int64(assignee.ID), task.AssignedUserID.Int64)

		// Verify task was saved to database
		dbTask := &Task{}
		err = gormDB.Preload("Author").Preload("AssignedUser").First(dbTask, task.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, task.Title, dbTask.Title)
		assert.Equal(t, task.Description, dbTask.Description)
		assert.Equal(t, author.ID, dbTask.AuthorID)
		assert.True(t, dbTask.AssignedUserID.Valid)
		assert.Equal(t, int64(assignee.ID), dbTask.AssignedUserID.Int64)
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
		assert.True(t, task.AssignedUserID.Valid)
		assert.Equal(t, int64(user.ID), task.AssignedUserID.Int64)
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
		assert.False(t, task.AssignedUserID.Valid) // Should be invalid when no assignee

		// Verify task was saved to database
		dbTask := &Task{}
		err = gormDB.Preload("Author").Preload("AssignedUser").First(dbTask, task.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, task.Title, dbTask.Title)
		assert.Equal(t, task.Description, dbTask.Description)
		assert.Equal(t, author.ID, dbTask.AuthorID)
		assert.False(t, dbTask.AssignedUserID.Valid)
		assert.Equal(t, author.Username, dbTask.Author.Username)
		assert.Zero(t, dbTask.AssignedUser.ID) // Should be zero when no assignee
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
		retrievedTask, err := db.GetTaskByID(int64(task.ID))
		assert.NoError(t, err)
		assert.NotNil(t, retrievedTask)
		assert.Equal(t, task.ID, retrievedTask.ID)
		assert.Equal(t, task.Title, retrievedTask.Title)
		assert.Equal(t, task.Description, retrievedTask.Description)
		assert.Equal(t, author.ID, retrievedTask.AuthorID)
		assert.True(t, retrievedTask.AssignedUserID.Valid)
		assert.Equal(t, int64(assignee.ID), retrievedTask.AssignedUserID.Int64)
		assert.Equal(t, author.Username, retrievedTask.Author.Username)
		assert.Equal(t, assignee.Username, retrievedTask.AssignedUser.Username)
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
		retrievedTask, err := db.GetTaskByID(int64(task.ID))
		assert.NoError(t, err)
		assert.NotNil(t, retrievedTask)
		assert.Equal(t, task.ID, retrievedTask.ID)
		assert.Equal(t, task.Title, retrievedTask.Title)
		assert.Equal(t, task.Description, retrievedTask.Description)
		assert.Equal(t, author.ID, retrievedTask.AuthorID)
		assert.False(t, retrievedTask.AssignedUserID.Valid)
		assert.Equal(t, author.Username, retrievedTask.Author.Username)
		assert.Zero(t, retrievedTask.AssignedUser.ID) // Should be zero when no assignee
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

	t.Run("negative ID", func(t *testing.T) {
		gormDB := CreateTestDB()
		db := NewDB(gormDB)

		// Try to retrieve task with negative ID
		task, err := db.GetTaskByID(-1)
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
			assert.False(t, task.AssignedUserID.Valid)
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
			assert.False(t, task.AssignedUserID.Valid)
		}

		// Test designer role
		designerTasks, err := db.GetUnassignedTasksByRole("designer")
		assert.NoError(t, err)
		assert.Len(t, designerTasks, 1)
		assert.Equal(t, "designer", designerTasks[0].Role)
		assert.False(t, designerTasks[0].AssignedUserID.Valid)

		// Test QA role
		qaTasks, err := db.GetUnassignedTasksByRole("qa")
		assert.NoError(t, err)
		assert.Len(t, qaTasks, 1)
		assert.Equal(t, "qa", qaTasks[0].Role)
		assert.False(t, qaTasks[0].AssignedUserID.Valid)
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
		assert.Equal(t, assignee.Username, retrievedTask.AssignedUser.Username)
		assert.True(t, retrievedTask.AssignedUserID.Valid)
		assert.Equal(t, int64(assignee.ID), retrievedTask.AssignedUserID.Int64)
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
		retrievedTask, err := db.GetTaskByID(int64(task.ID))
		require.NoError(t, err)
		assert.False(t, retrievedTask.AssignedUserID.Valid)

		// Assign task to user
		err = db.AssignTask(int64(task.ID), int64(assignee.ID))
		assert.NoError(t, err)

		// Verify task is now assigned
		retrievedTask, err = db.GetTaskByID(int64(task.ID))
		require.NoError(t, err)
		assert.True(t, retrievedTask.AssignedUserID.Valid)
		assert.Equal(t, int64(assignee.ID), retrievedTask.AssignedUserID.Int64)
		assert.Equal(t, assignee.Username, retrievedTask.AssignedUser.Username)
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
		retrievedTask, err := db.GetTaskByID(int64(task.ID))
		require.NoError(t, err)
		assert.True(t, retrievedTask.AssignedUserID.Valid)
		assert.Equal(t, int64(assignee1.ID), retrievedTask.AssignedUserID.Int64)

		// Reassign task to second user
		err = db.AssignTask(int64(task.ID), int64(assignee2.ID))
		assert.NoError(t, err)

		// Verify task is now assigned to second user
		retrievedTask, err = db.GetTaskByID(int64(task.ID))
		require.NoError(t, err)
		assert.True(t, retrievedTask.AssignedUserID.Valid)
		assert.Equal(t, int64(assignee2.ID), retrievedTask.AssignedUserID.Int64)
		assert.Equal(t, assignee2.Username, retrievedTask.AssignedUser.Username)
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
		err = db.AssignTask(int64(task.ID), 999)
		assert.NoError(t, err) // The function doesn't validate user existence

		// Verify task is assigned to the non-existent user ID
		retrievedTask, err := db.GetTaskByID(int64(task.ID))
		require.NoError(t, err)
		assert.True(t, retrievedTask.AssignedUserID.Valid)
		assert.Equal(t, int64(999), retrievedTask.AssignedUserID.Int64)
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
		err = db.AssignTask(999, int64(user.ID))
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
		err = db.AssignTask(int64(task.ID), 0)
		assert.NoError(t, err)

		// Verify task is assigned to user ID 0
		retrievedTask, err := db.GetTaskByID(int64(task.ID))
		require.NoError(t, err)
		assert.True(t, retrievedTask.AssignedUserID.Valid)
		assert.Equal(t, int64(0), retrievedTask.AssignedUserID.Int64)
	})

	t.Run("assign task with negative user ID", func(t *testing.T) {
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

		// Try to assign task with negative user ID
		err = db.AssignTask(int64(task.ID), -1)
		assert.NoError(t, err)

		// Verify task is assigned to negative user ID
		retrievedTask, err := db.GetTaskByID(int64(task.ID))
		require.NoError(t, err)
		assert.True(t, retrievedTask.AssignedUserID.Valid)
		assert.Equal(t, int64(-1), retrievedTask.AssignedUserID.Int64)
	})
}
