package discord

import (
	"fmt"

	"github.com/Formula-SAE/discord/internal/db"
	"github.com/bwmarrin/discordgo"
)

type DiscordBot struct {
	session *discordgo.Session
	db      *db.DB
	appID   string
	guildID string
}

func NewDiscordBot(s *discordgo.Session, db *db.DB, appID string, guildID string) *DiscordBot {
	return &DiscordBot{
		session: s,
		db:      db,
		appID:   appID,
		guildID: guildID,
	}
}

func (b *DiscordBot) Start() (func() error, error) {
	err := b.session.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open Discord session: %v", err)
	}

	b.session.AddHandler(b.createTaskCommand)
	b.session.AddHandler(b.getAssignedTasks)

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
					Description: "The description of the task",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "assignee",
					Description: "The user to assign the task to",
					Required:    true,
				},
			},
		},
		{
			Name:        "assigned-tasks",
			Description: "Get all tasks assigned to the current user",
		},
	}

	for _, command := range commands {
		_, err = b.session.ApplicationCommandCreate(b.appID, "", command)
		if err != nil {
			return nil, fmt.Errorf("failed to create application command: %v", err)
		}
	}

	return b.session.Close, nil
}
