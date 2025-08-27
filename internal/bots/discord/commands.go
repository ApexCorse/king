package discord

import (
	"fmt"
	"sync"

	"github.com/Formula-SAE/discord/internal/db"
	"github.com/bwmarrin/discordgo"
)

func (b *DiscordBot) createTaskCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != "create-task" {
		return
	}

	if len(i.ApplicationCommandData().Options) != 3 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You must provide a title, description and assignee",
			},
		})
		return
	}

	if i.ApplicationCommandData().Options[0].Name != "title" ||
		i.ApplicationCommandData().Options[1].Name != "description" ||
		i.ApplicationCommandData().Options[2].Name != "assignee" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You must provide a title, description and assignee",
			},
		})
		return
	}

	if i.ApplicationCommandData().Options[0].Type != discordgo.ApplicationCommandOptionString ||
		i.ApplicationCommandData().Options[1].Type != discordgo.ApplicationCommandOptionString ||
		i.ApplicationCommandData().Options[2].Type != discordgo.ApplicationCommandOptionUser {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You must provide a title, description and assignee",
			},
		})
		return
	}

	if i.Member == nil || i.Member.User == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You must be a member of a server to use this command",
			},
		})
		return
	}

	userDiscordID := i.Member.User.ID
	title := i.ApplicationCommandData().Options[0].StringValue()
	description := i.ApplicationCommandData().Options[1].StringValue()
	assigneeDiscordID := i.ApplicationCommandData().Options[2].UserValue(s).ID

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		b.db.CreateUser(&db.User{
			Username:  i.Member.User.Username,
			DiscordID: userDiscordID,
		})
		wg.Done()
	}()

	go func() {
		b.db.CreateUser(&db.User{
			Username:  i.ApplicationCommandData().Options[2].UserValue(s).Username,
			DiscordID: assigneeDiscordID,
		})
		wg.Done()
	}()

	wg.Wait()

	task := &db.Task{
		Title:       title,
		Description: description,
	}

	err := b.db.CreateTaskWithUserDiscordID(
		task,
		userDiscordID,
		assigneeDiscordID,
	)

	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to create task",
			},
		})
		return
	}

	respContent := fmt.Sprintf(`Task created successfully:
	*Title*: %s
	*Description*: %s
	*Assignee*: <@%s>`,
		title, description, assigneeDiscordID)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: respContent,
		},
	})
}

func (b *DiscordBot) getAssignedTasks(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != "assigned-tasks" {
		return
	}

	if i.Member == nil || i.Member.User == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You must be a member of a server to use this command",
			},
		})
		return
	}

	userDiscordID := i.Member.User.ID
	tasks, err := b.db.GetAssignedTasksByUserDiscordID(userDiscordID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to get assigned tasks",
			},
		})
		return
	}

	respContent := "Current tasks:\n"
	for _, task := range tasks {
		respContent += fmt.Sprintf(`
	*Title*: %s
	*Description*: %s
	*Author*: <@%s>
		`, task.Title, task.Description, task.Author.DiscordID)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: respContent,
		},
	})
}
