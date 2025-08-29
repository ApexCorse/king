package discord

import (
	"fmt"
	"net/http"

	"github.com/Formula-SAE/discord/internal/db"
	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/mux"
)

type DiscordBot struct {
	session *discordgo.Session
	db      *db.DB
	appID   string
	guildID string

	router *mux.Router
}

func NewDiscordBot(s *discordgo.Session, db *db.DB, appID string, guildID string, router *mux.Router) *DiscordBot {
	return &DiscordBot{
		session: s,
		db:      db,
		appID:   appID,
		guildID: guildID,
		router:  router,
	}
}

func (b *DiscordBot) Start() (func() error, error) {
	b.router.HandleFunc("/push", b.onPushWebhook).Methods("POST")

	err := b.session.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open Discord session: %v", err)
	}

	fmt.Printf("[bot] Discord session opened successfully\n")
	b.session.AddHandler(b.createTaskCommand)
	b.session.AddHandler(b.getAssignedTasksCommand)
	b.session.AddHandler(b.getTaskCommand)
	b.session.AddHandler(b.getTasksByRoleCommand)
	b.session.AddHandler(b.getUnassignedTasksByRoleCommand)
	b.session.AddHandler(b.assignTaskCommand)
	b.session.AddHandler(b.updateTaskStatusCommand)
	b.session.AddHandler(b.getCompletedTasksByRoleCommand)
	b.session.AddHandler(b.subscribeChannelToPushWebhookCommand)
	b.session.AddHandler(b.unsubscribeChannelFromPushWebhookCommand)
	fmt.Printf("[bot] Command handlers registered\n")

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "create-task",
			Description: "Create a new task to assign to a member",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "title",
					Description: "The title of the task",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "description",
					Description: "The description of the task (optional)",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "assignee",
					Description: "The user to assign the task to (optional)",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionRole,
					Name:        "role",
					Description: "The role to assign the task to (optional)",
					Required:    false,
				},
			},
		},
		{
			Name:        "assigned-tasks",
			Description: "Get all tasks assigned to the current user",
		},
		{
			Name:        "get-task",
			Description: "Get a task by its ID",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "id",
					Description: "The ID of the task to get",
					Required:    true,
				},
			},
		},
		{
			Name:        "get-tasks-by-role",
			Description: "Get all tasks for a specific role",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionRole,
					Name:        "role",
					Description: "The role to get tasks for",
					Required:    true,
				},
			},
		},
		{
			Name:        "unassigned-tasks-by-role",
			Description: "Get all unassigned tasks for a specific role",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionRole,
					Name:        "role",
					Description: "The role to get unassigned tasks for",
					Required:    true,
				},
			},
		},
		{
			Name:        "assign-task",
			Description: "Assign a task to a user",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "task-id",
					Description: "The ID of the task to assign",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user-id",
					Description: "The ID of the user to assign the task to",
					Required:    true,
				},
			},
		},
		{
			Name:        "update-task-status",
			Description: "Update the status of a task",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "task-id",
					Description: "The ID of the task to update",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "status",
					Description: "The new status (Not Started, In Progress, Completed)",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "Not Started",
							Value: db.TASK_NOT_STARTED,
						},
						{
							Name:  "In Progress",
							Value: db.TASK_IN_PROGRESS,
						},
						{
							Name:  "Completed",
							Value: db.TASK_COMPLETED,
						},
					},
				},
			},
		},
		{
			Name:        "completed-tasks-by-role",
			Description: "Get all completed tasks for a specific role",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionRole,
					Name:        "role",
					Description: "The role to get completed tasks for",
					Required:    true,
				},
			},
		},
		{
			Name:        "subscribe-channel-to-push",
			Description: "Subscribe a channel to push webhook for a repository",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "repository",
					Description: "The repository to subscribe to",
					Required:    true,
				},
			},
		},
		{
			Name:        "unsubscribe-channel-from-push",
			Description: "Unsubscribe a channel from push webhook for a repository",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "repository",
					Description: "The repository to unsubscribe from",
					Required:    true,
				},
			},
		},
	}

	for _, command := range commands {
		_, err = b.session.ApplicationCommandCreate(b.appID, "", command)
		if err != nil {
			return nil, fmt.Errorf("failed to create application command: %v", err)
		}
		fmt.Printf("[bot] Created command: %s\n", command.Name)
	}

	go http.ListenAndServe(":8080", b.router)

	return b.session.Close, nil
}
