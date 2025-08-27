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

	fmt.Printf("[create-task] Command executed by user %s\n", i.Member.User.Username)

	if len(i.ApplicationCommandData().Options) != 3 {
		fmt.Printf("[create-task] Invalid number of options provided: %d\n", len(i.ApplicationCommandData().Options))
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
		fmt.Printf("[create-task] Invalid option names provided\n")
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
		fmt.Printf("[create-task] Invalid option types provided\n")
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

	fmt.Printf("[create-task] Creating task with title: %s, description: %s, assignee: %s\n", title, description, assigneeDiscordID)

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
		fmt.Printf("[create-task] Failed to create task: %v\n", err)
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

	fmt.Printf("[create-task] Task created successfully with ID: %d\n", task.ID)
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
		fmt.Printf("[assigned-tasks] User is not a member of the server\n")
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You must be a member of a server to use this command",
			},
		})
		return
	}

	fmt.Printf("[assigned-tasks] Command executed by user %s\n", i.Member.User.Username)
	userDiscordID := i.Member.User.ID
	tasks, err := b.db.GetAssignedTasksByUserDiscordID(userDiscordID)
	if err != nil {
		fmt.Printf("[assigned-tasks] Failed to get assigned tasks: %v\n", err)
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

	fmt.Printf("[assigned-tasks] Retrieved %d tasks for user %s\n", len(tasks), userDiscordID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: respContent,
		},
	})
}
