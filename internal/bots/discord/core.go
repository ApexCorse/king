package discord

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Formula-SAE/discord/internal/db"
	"github.com/bwmarrin/discordgo"
	"github.com/google/go-github/v74/github"
	"github.com/gorilla/mux"
)

type DiscordBot struct {
	session *discordgo.Session
	db      *db.DB
	appID   string
	guildID string

	router *mux.Router
	gc     *github.Client
}

func NewDiscordBot(s *discordgo.Session, db *db.DB, appID string, guildID string, router *mux.Router, gc *github.Client) *DiscordBot {
	return &DiscordBot{
		session: s,
		db:      db,
		appID:   appID,
		guildID: guildID,
		router:  router,
		gc:      gc,
	}
}

func (b *DiscordBot) Start(ctx context.Context) (func() error, error) {
	b.initWebhookHandlers()

	err := b.session.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open Discord session: %v", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	githubRepoNames, err := b.updateRepositoriesInDB(ctx)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[bot] Discord session opened successfully\n")
	b.initCommandHandlers()
	fmt.Printf("[bot] Command handlers registered\n")

	if err := b.initCommands(githubRepoNames); err != nil {
		return nil, err
	}

	go http.ListenAndServe(":8080", b.router)

	return b.session.Close, nil
}

func (b *DiscordBot) initCommandHandlers() {
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
	b.session.AddHandler(b.deleteTaskCommand)
}

func (b *DiscordBot) initCommands(githubRepoNames []string) error {
	repoOptions := make([]*discordgo.ApplicationCommandOptionChoice, len(githubRepoNames))
	for i, repoName := range githubRepoNames {
		repoOptions[i] = &discordgo.ApplicationCommandOptionChoice{
			Name:  repoName,
			Value: repoName,
		}
	}

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
					Choices:     repoOptions,
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
					Choices:     repoOptions,
				},
			},
		},
		{
			Name:        "delete-task",
			Description: "Delete a task by its ID",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "id",
					Description: "The ID of the task to delete",
					Required:    true,
				},
			},
		},
	}

	errChan := make(chan error, len(commands))
	wg := sync.WaitGroup{}
	for _, command := range commands {
		wg.Add(1)
		go func(command *discordgo.ApplicationCommand) {
			defer wg.Done()
			_, err := b.session.ApplicationCommandCreate(b.appID, "", command)
			if err != nil {
				fmt.Printf("[bot] Failed to create application command: %v\n", err)
				errChan <- err
			}
			fmt.Printf("[bot] Created command: %s\n", command.Name)
		}(command)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		return err
	}

	return nil
}

func (b *DiscordBot) initWebhookHandlers() {
	b.router.HandleFunc("/push", b.onPushWebhook).Methods("POST")
}
