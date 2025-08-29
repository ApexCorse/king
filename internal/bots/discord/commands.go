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

	options := i.ApplicationCommandData().Options
	if len(options) < 1 {
		fmt.Printf("[create-task] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Missing title**\n\nYou must provide at least a title for the task.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if i.Member == nil || i.Member.User == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "🚫 **Access denied**\n\nYou must be a member of a server to use this command.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	fmt.Printf("[create-task] Command executed by user %s\n", i.Member.User.Username)

	task := &db.Task{}
	assigneeId := ""
	respContent := ""

	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	title, okTitle := optionMap["title"]
	if !okTitle {
		fmt.Printf("[create-task] Title option not found\n")
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Missing title**\n\nYou must provide a title for the task.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	} else {
		fmt.Printf(
			"[create-task] Creating task with title: %s",
			title.StringValue(),
		)
		task.Title = title.StringValue()
		respContent = fmt.Sprintf("✅ **Task created successfully!**\n\n*Title*: %s", title.StringValue())
	}

	description := optionMap["description"]
	assignee := optionMap["assignee"]
	role := optionMap["role"]
	author := i.Member.User

	if description != nil {
		fmt.Printf(", description: %s", description.StringValue())
		task.Description = description.StringValue()
		respContent += fmt.Sprintf("\n*Description*: %s", description.StringValue())
	}

	if assignee != nil {
		fmt.Printf(", assignee: %s", assignee.UserValue(s).ID)
		assigneeId = assignee.UserValue(s).ID
		respContent += fmt.Sprintf("\n*Assignee*: <@%s>", assigneeId)
	}

	if role != nil {
		fmt.Printf(", role: %s", role.RoleValue(s, b.guildID).Name)
		task.Role = role.RoleValue(s, b.guildID).Name
		respContent += fmt.Sprintf("\n*Role*: %s", role.RoleValue(s, b.guildID).Name)
	}
	fmt.Printf("\n")

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		b.db.CreateUser(&db.User{
			Username:  author.Username,
			DiscordID: author.ID,
		})
		wg.Done()
	}()

	if assignee != nil {
		go func() {
			b.db.CreateUser(&db.User{
				Username:  assignee.UserValue(s).Username,
				DiscordID: assignee.UserValue(s).ID,
			})
			wg.Done()
		}()
	} else {
		wg.Done()
	}

	wg.Wait()

	err := b.db.CreateTaskWithUserDiscordID(
		task,
		author.ID,
		assigneeId,
	)

	if err != nil {
		fmt.Printf("[create-task] Failed to create task: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ **Failed to create task**\n\nAn error occurred while creating the task. Please try again or contact an administrator.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	fmt.Printf("[create-task] Task created successfully with ID: %d\n", task.ID)
	respContent += fmt.Sprintf("\n*Task ID*: `%d`", task.ID)
	respContent += fmt.Sprintf("\n*Status*: %s %s", getStatusIcon(task.Status), task.Status)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: respContent,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *DiscordBot) getAssignedTasksCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
				Flags:   discordgo.MessageFlagsEphemeral,
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
				Content: "❌ **Failed to retrieve tasks**\n\nAn error occurred while fetching your assigned tasks. Please try again or contact an administrator.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if len(tasks) == 0 {
		respContent := "🎉 You have no tasks assigned to you at the moment."
		fmt.Printf("[assigned-tasks] No tasks found for user %s\n", userDiscordID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: respContent,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := "📋 **Your assigned tasks:**\n"
	for _, task := range tasks {
		respContent += fmt.Sprintf("\n*ID*: %d", task.ID)
		respContent += fmt.Sprintf("\n*Title*: %s", task.Title)
		if task.Description != "" {
			respContent += fmt.Sprintf("\n*Description*: %s", task.Description)
		}
		respContent += fmt.Sprintf("\n*Author*: <@%s>", task.Author.DiscordID)
		respContent += fmt.Sprintf("\n*Status*: %s %s\n", getStatusIcon(task.Status), task.Status)
	}

	fmt.Printf("[assigned-tasks] Retrieved %d tasks for user %s\n", len(tasks), userDiscordID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: respContent,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *DiscordBot) getTaskCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != "get-task" {
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) != 1 {
		fmt.Printf("[get-task] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Invalid options**\n\nYou must provide exactly one option (task ID).",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if options[0].Name != "id" && options[0].Type != discordgo.ApplicationCommandOptionInteger {
		fmt.Printf("[get-task] Invalid option provided: %s\n", options[0].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Invalid ID format**\n\nYou must provide a valid integer ID for the task.",
			},
		})
		return
	}

	taskID := options[0].IntValue()
	task, err := b.db.GetTaskByID(taskID)
	if err != nil {
		fmt.Printf("[get-task] Failed to get task: %v\n", err)
		respContent := fmt.Sprintf("❌ **Task not found!**\n\nTask with ID `%d` doesn't exist or has been deleted. Please check the ID and try again.", taskID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: respContent,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := fmt.Sprintf("📋 **Task #%d**\n\n*Title*: %s", task.ID, task.Title)
	if task.Description != "" {
		respContent += fmt.Sprintf("\n*Description*: %s", task.Description)
	}
	if task.Role != "" {
		respContent += fmt.Sprintf("\n*Role*: %s", task.Role)
	}
	if task.AssignedUserID.Valid {
		respContent += fmt.Sprintf("\n*Assigned to*: <@%s>", task.AssignedUser.DiscordID)
	}
	respContent += fmt.Sprintf("\n*Author*: <@%s>", task.Author.DiscordID)
	respContent += fmt.Sprintf("\n*Status*: %s %s", getStatusIcon(task.Status), task.Status)

	fmt.Printf("[get-task] Retrieved task %d: %s\n", task.ID, respContent)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: respContent,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *DiscordBot) getTasksByRoleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != "get-tasks-by-role" {
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) != 1 {
		fmt.Printf("[get-tasks-by-role] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Invalid options**\n\nYou must provide exactly one option (role).",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if options[0].Name != "role" && options[0].Type != discordgo.ApplicationCommandOptionRole {
		fmt.Printf("[get-tasks-by-role] Invalid option provided: %s\n", options[0].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Invalid role**\n\nYou must provide a valid role.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	role := options[0].RoleValue(s, b.guildID).Name
	tasks, err := b.db.GetTasksByRole(role)
	if err != nil {
		fmt.Printf("[get-tasks-by-role] Failed to get tasks: %v\n", err)
		respContent := fmt.Sprintf("❌ **Failed to retrieve tasks for role `%s`**\n\nAn error occurred while fetching tasks. Please try again or contact an administrator.", role)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: respContent,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if len(tasks) == 0 {
		respContent := fmt.Sprintf("🔍 **No tasks found for role `%s`**\n\nThis role currently has no active tasks.", role)
		fmt.Printf("[get-tasks-by-role] No tasks found for role %s\n", role)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: respContent,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := fmt.Sprintf("📋 **Tasks for role `%s`:**\n", role)
	for _, task := range tasks {
		respContent += "--------------------------------\n"
		respContent += fmt.Sprintf("*ID*: %d", task.ID)
		respContent += fmt.Sprintf("\n*Title*: %s", task.Title)
		if task.Description != "" {
			respContent += fmt.Sprintf("\n*Description*: %s", task.Description)
		}
		respContent += fmt.Sprintf("\n*Author*: <@%s>", task.Author.DiscordID)
		if task.AssignedUserID.Valid {
			respContent += fmt.Sprintf("\n*Assigned to*: <@%s>", task.AssignedUser.DiscordID)
		}
		respContent += fmt.Sprintf("\n*Status*: %s %s", getStatusIcon(task.Status), task.Status)
		respContent += "\n"
	}

	fmt.Printf("[get-tasks-by-role] Retrieved %d tasks for role %s\n", len(tasks), role)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: respContent,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *DiscordBot) getUnassignedTasksByRoleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != "unassigned-tasks-by-role" {
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) != 1 {
		fmt.Printf("[unassigned-tasks-by-role] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Invalid options**\n\nYou must provide exactly one option (role).",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if options[0].Name != "role" && options[0].Type != discordgo.ApplicationCommandOptionRole {
		fmt.Printf("[unassigned-tasks-by-role] Invalid option provided: %s\n", options[0].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Invalid role**\n\nYou must provide a valid role.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	role := options[0].RoleValue(s, b.guildID).Name
	tasks, err := b.db.GetUnassignedTasksByRole(role)
	if err != nil {
		fmt.Printf("[unassigned-tasks-by-role] Failed to get tasks: %v\n", err)
		respContent := fmt.Sprintf("❌ **Failed to retrieve tasks for role `%s`**\n\nAn error occurred while fetching tasks. Please try again or contact an administrator.", role)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: respContent,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if len(tasks) == 0 {
		respContent := fmt.Sprintf("🔍 **No unassigned tasks found for role `%s`**\n\nThis role currently has no unassigned tasks.", role)
		fmt.Printf("[unassigned-tasks-by-role] No unassigned tasks found for role %s\n", role)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: respContent,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := fmt.Sprintf("📋 **Unassigned tasks for role `%s`:**\n", role)
	for _, task := range tasks {
		respContent += "--------------------------------\n"
		respContent += fmt.Sprintf("*ID*: %d", task.ID)
		respContent += fmt.Sprintf("\n*Title*: %s", task.Title)
		if task.Description != "" {
			respContent += fmt.Sprintf("\n*Description*: %s", task.Description)
		}
		respContent += fmt.Sprintf("\n*Author*: <@%s>", task.Author.DiscordID)
		respContent += fmt.Sprintf("\n*Status*: %s %s", getStatusIcon(task.Status), task.Status)
		respContent += "\n"
	}

	fmt.Printf("[unassigned-tasks-by-role] Retrieved %d unassigned tasks for role %s\n", len(tasks), role)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: respContent,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *DiscordBot) assignTaskCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != "assign-task" {
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) != 2 {
		fmt.Printf("[assign-task] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Invalid options**\n\nYou must provide exactly two options (task ID and user ID).",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if options[0].Name != "task-id" && options[0].Type != discordgo.ApplicationCommandOptionInteger {
		fmt.Printf("[assign-task] Invalid option provided: %s\n", options[0].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Invalid task ID**\n\nYou must provide a valid integer ID for the task.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	if options[1].Name != "user-id" && options[1].Type != discordgo.ApplicationCommandOptionUser {
		fmt.Printf("[assign-task] Invalid option provided: %s\n", options[1].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Invalid user ID**\n\nYou must provide a valid user ID.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	taskID := options[0].IntValue()
	userDiscordID := options[1].UserValue(s).ID

	user, err := b.db.GetUserByDiscordID(userDiscordID)
	if err != nil {
		fmt.Printf("[assign-task] Failed to get user: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ **Failed to assign task**\n\nAn error occurred while assigning the task. Please try again or contact an administrator.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	err = b.db.AssignTask(taskID, int64(user.ID))
	if err != nil {
		fmt.Printf("[assign-task] Failed to assign task: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ **Failed to assign task**\n\nAn error occurred while assigning the task. Please try again or contact an administrator.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := fmt.Sprintf("✅ **Task assigned successfully!**\n\n*Task ID*: `%d`\n*Assigned to*: <@%s>", taskID, userDiscordID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: respContent,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *DiscordBot) updateTaskStatusCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != "update-task-status" {
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) != 2 {
		fmt.Printf("[update-task-status] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Invalid options**\n\nYou must provide exactly two options (task ID and status).",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if options[0].Name != "task-id" && options[0].Type != discordgo.ApplicationCommandOptionInteger {
		fmt.Printf("[update-task-status] Invalid option provided: %s\n", options[0].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Invalid task ID**\n\nYou must provide a valid integer ID for the task.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if options[1].Name != "status" && options[1].Type != discordgo.ApplicationCommandOptionString {
		fmt.Printf("[update-task-status] Invalid option provided: %s\n", options[1].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Invalid status**\n\nYou must provide a valid status string.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	taskID := options[0].IntValue()
	status := options[1].StringValue()

	task, err := b.db.UpdateTaskStatus(taskID, status)
	if err != nil {
		fmt.Printf("[update-task-status] Failed to update task status: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ **Failed to update task status**\n\n%s", err.Error()),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := fmt.Sprintf("✅ **Task status updated successfully!**\n\n*Task ID*: `%d`\n*Title*: %s\n*New Status*: %s %s", taskID, task.Title, getStatusIcon(status), status)
	fmt.Printf("[update-task-status] Task %d status updated to: %s\n", taskID, status)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: respContent,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *DiscordBot) getCompletedTasksByRoleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != "completed-tasks-by-role" {
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) != 1 {
		fmt.Printf("[completed-tasks-by-role] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Invalid options**\n\nYou must provide exactly one option (role).",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if options[0].Name != "role" && options[0].Type != discordgo.ApplicationCommandOptionRole {
		fmt.Printf("[completed-tasks-by-role] Invalid option provided: %s\n", options[0].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Invalid role**\n\nYou must provide a valid role.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	role := options[0].RoleValue(s, b.guildID).Name
	tasks, err := b.db.GetCompletedTasksByRole(role)
	if err != nil {
		fmt.Printf("[completed-tasks-by-role] Failed to get tasks: %v\n", err)
		respContent := fmt.Sprintf("❌ **Failed to retrieve completed tasks for role `%s`**\n\nAn error occurred while fetching tasks. Please try again or contact an administrator.", role)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: respContent,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if len(tasks) == 0 {
		respContent := fmt.Sprintf("🔍 **No completed tasks found for role `%s`**\n\nThis role currently has no completed tasks.", role)
		fmt.Printf("[completed-tasks-by-role] No completed tasks found for role %s\n", role)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: respContent,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := fmt.Sprintf("✅ **Completed tasks for role `%s`:**\n", role)
	for _, task := range tasks {
		respContent += "--------------------------------\n"
		respContent += fmt.Sprintf("*ID*: %d", task.ID)
		respContent += fmt.Sprintf("\n*Title*: %s", task.Title)
		if task.Description != "" {
			respContent += fmt.Sprintf("\n*Description*: %s", task.Description)
		}
		respContent += fmt.Sprintf("\n*Author*: <@%s>", task.Author.DiscordID)
		if task.AssignedUserID.Valid {
			respContent += fmt.Sprintf("\n*Assigned to*: <@%s>", task.AssignedUser.DiscordID)
		}
		respContent += fmt.Sprintf("\n*Status*: %s %s", getStatusIcon(task.Status), task.Status)
		respContent += "\n"
	}

	fmt.Printf("[completed-tasks-by-role] Retrieved %d completed tasks for role %s\n", len(tasks), role)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: respContent,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *DiscordBot) subscribeChannelToPushWebhookCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != "subscribe-channel-to-push" {
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) != 1 {
		fmt.Printf("[subscribe-channel-to-push] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Invalid options**\n\nYou must provide exactly one option (repository).",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if options[0].Name != "repository" && options[0].Type != discordgo.ApplicationCommandOptionString {
		fmt.Printf("[subscribe-channel-to-push] Invalid option provided: %s\n", options[0].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Invalid repository**\n\nYou must provide a valid repository name.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	repo := options[0].StringValue()

	_, err := b.db.CreateWebhookSubscription(repo, i.ChannelID)
	if err != nil {
		fmt.Printf("[subscribe-channel-to-push] Failed to create webhook subscription: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ **Failed to subscribe channel to push**\n\nAn error occurred while subscribing the channel to the push webhook. Please try again or contact an administrator.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := fmt.Sprintf("✅ **Channel subscribed to push webhook for repository `%s`**\n\n*Repository*: `%s`\n*Channel*: `%s`", repo, repo, i.ChannelID)
	fmt.Printf("[subscribe-channel-to-push] Channel %s subscribed to push webhook for repository %s\n", i.ChannelID, repo)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: respContent,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *DiscordBot) unsubscribeChannelFromPushWebhookCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != "unsubscribe-channel-from-push" {
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) != 1 {
		fmt.Printf("[unsubscribe-channel-from-push] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Invalid options**\n\nYou must provide exactly one option (repository).",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if options[0].Name != "repository" && options[0].Type != discordgo.ApplicationCommandOptionString {
		fmt.Printf("[unsubscribe-channel-from-push] Invalid option provided: %s\n", options[0].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ **Invalid repository**\n\nYou must provide a valid repository name.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	repo := options[0].StringValue()

	err := b.db.DeleteWebhookSubscription(repo, i.ChannelID)
	if err != nil {
		fmt.Printf("[unsubscribe-channel-from-push] Failed to delete webhook subscription: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ **Failed to unsubscribe channel from push**\n\nAn error occurred while unsubscribing the channel from the push webhook. Please try again or contact an administrator.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := fmt.Sprintf("✅ **Channel unsubscribed from push webhook for repository `%s`**\n\n*Repository*: `%s`\n*Channel*: `%s`", repo, repo, i.ChannelID)
	fmt.Printf("[unsubscribe-channel-from-push] Channel %s unsubscribed from push webhook for repository %s\n", i.ChannelID, repo)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: respContent,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// getStatusIcon returns the appropriate emoji icon for a task status
func getStatusIcon(status string) string {
	switch status {
	case db.TASK_NOT_STARTED:
		return "⏳" // Hourglass
	case db.TASK_IN_PROGRESS:
		return "🔄" // Rotating arrows
	case db.TASK_COMPLETED:
		return "✅" // Check mark
	default:
		return "❓" // Question mark for unknown status
	}
}
