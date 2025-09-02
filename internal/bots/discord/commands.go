package discord

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Formula-SAE/discord/internal/db"
	"github.com/Formula-SAE/discord/internal/utils"
	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

// Command name constants
const (
	CommandCreateTask                 = "create-task"
	CommandAssignedTasks              = "assigned-tasks"
	CommandGetTask                    = "get-task"
	CommandGetTasksByRole             = "get-tasks-by-role"
	CommandUnassignedTasksByRole      = "unassigned-tasks-by-role"
	CommandAssignTask                 = "assign-task"
	CommandUpdateTaskStatus           = "update-task-status"
	CommandCompletedTasksByRole       = "completed-tasks-by-role"
	CommandDeleteTask                 = "delete-task"
	CommandSubscribeChannelToPush     = "subscribe-channel-to-push"
	CommandUnsubscribeChannelFromPush = "unsubscribe-channel-from-push"
	CommandGetSubscriptionsOfChannel  = "get-subscriptions"
)

func (b *DiscordBot) createTaskCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != CommandCreateTask {
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) < 1 {
		fmt.Printf("[create-task] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					warningComponent("Missing title", "You must provide at least a title for the task."),
				},
			},
		})
		return
	}

	if i.Member == nil || i.Member.User == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					warningComponent("Access denied", "You must be a member of a server to use this command."),
				},
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
				Components: []discordgo.MessageComponent{
					warningComponent("Missing title", "You must provide at least a title for the task."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	} else {
		fmt.Printf(
			"[create-task] Creating task with title: %s",
			title.StringValue(),
		)
		task.Title = title.StringValue()
		respContent = fmt.Sprintf("%s: %s", utils.Italic("Title"), title.StringValue())
	}

	description := optionMap["description"]
	assignee := optionMap["assignee"]
	role := optionMap["role"]
	author := i.Member.User

	if description != nil {
		fmt.Printf(", description: %s", description.StringValue())
		task.Description = description.StringValue()
		respContent += fmt.Sprintf("\n%s: %s", utils.Italic("Description"), description.StringValue())
	}

	if assignee != nil {
		fmt.Printf(", assignee: %s", assignee.UserValue(s).ID)
		assigneeId = assignee.UserValue(s).ID
		respContent += fmt.Sprintf("\n%s: <@%s>", utils.Italic("Assignee"), assigneeId)
	}

	if role != nil {
		fmt.Printf(", role: %s", role.RoleValue(s, b.guildID).Name)
		task.Role = role.RoleValue(s, b.guildID).Name
		respContent += fmt.Sprintf("\n%s: %s", utils.Italic("Role"), role.RoleValue(s, b.guildID).Name)
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
				Flags: discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					errorComponent("Failed to create task", "An error occurred while creating the task. Please try again or contact an administrator."),
				},
			},
		})
		return
	}

	fmt.Printf("[create-task] Task created successfully with ID: %d\n", task.ID)
	respContent += fmt.Sprintf("\n%s: %s", utils.Italic("Task ID"), utils.InlineCode(fmt.Sprintf("%d", task.ID)))
	respContent += fmt.Sprintf("\n%s: %s %s", utils.Italic("Status"), getStatusIcon(task.Status), task.Status)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				confirmComponent("Task created", respContent),
			},
		},
	})

	if assignee != nil {
		message := fmt.Sprintf("👋 Hi <@%s>! You have been assigned a new task:\n\n%s", assigneeId, utils.Bold(task.Title))

		if task.Description != "" {
			message += fmt.Sprintf("\n%s", task.Description)
		}

		message += fmt.Sprintf("\n\n🔗 %s: %s\n\nGood luck! 🚀", utils.Italic("Task ID"), utils.InlineCode(fmt.Sprintf("%d", task.ID)))

		err = b.sendPrivateMessage(assigneeId, message)
		if err != nil {
			fmt.Printf("[create-task] Failed to send private message to assignee: %v\n", err)
		}
	}
}

func (b *DiscordBot) getAssignedTasksCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != CommandAssignedTasks {
		return
	}

	if i.Member == nil || i.Member.User == nil {
		fmt.Printf("[assigned-tasks] User is not a member of the server\n")
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					warningComponent("Access denied", "You must be a member of a server to use this command."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	fmt.Printf("[assigned-tasks] Command executed by user %s\n", i.Member.User.Username)

	userDiscordID := i.Member.User.ID
	user, err := b.getOrCreateUser(userDiscordID, i.Member.User.Username)
	if err != nil {
		fmt.Printf("[assigned-tasks] Failed to get or create user: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					errorComponent("Failed to retrieve tasks", "An error occurred while fetching your assigned tasks. Please try again or contact an administrator."),
				},
			},
		})
		return
	}
	tasks := user.AssignedTasks

	if len(tasks) == 0 {
		fmt.Printf("[assigned-tasks] No tasks found for user %s\n", userDiscordID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					infoComponent("No tasks assigned", "You have no tasks assigned to you at the moment."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := ""
	for _, task := range tasks {
		respContent += fmt.Sprintf("\n%s: %d", utils.Italic("ID"), task.ID)
		respContent += fmt.Sprintf("\n%s: %s", utils.Italic("Title"), task.Title)
		if task.Description != "" {
			respContent += fmt.Sprintf("\n%s: %s", utils.Italic("Description"), task.Description)
		}
		respContent += fmt.Sprintf("\n%s: <@%s>", utils.Italic("Author"), task.Author.DiscordID)
		respContent += fmt.Sprintf("\n%s: %s %s\n", utils.Italic("Status"), getStatusIcon(task.Status), task.Status)
	}

	fmt.Printf("[assigned-tasks] Retrieved %d tasks for user %s\n", len(tasks), userDiscordID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				listComponent("Your assigned tasks", respContent),
			},
		},
	})
}

func (b *DiscordBot) getTaskCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != CommandGetTask {
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) != 1 {
		fmt.Printf("[get-task] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid options", "You must provide exactly one option (task ID)."),
				},
			},
		})
		return
	}

	if options[0].Name != "id" && options[0].Type != discordgo.ApplicationCommandOptionInteger {
		fmt.Printf("[get-task] Invalid option provided: %s\n", options[0].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid ID format", "You must provide a valid integer ID for the task."),
				},
			},
		})
		return
	}

	taskID := options[0].IntValue()
	task, err := b.db.GetTaskByID(uint(taskID))
	if err != nil {
		fmt.Printf("[get-task] Failed to get task: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					errorComponent("Task not found", "Task with ID "+utils.InlineCode(fmt.Sprintf("%d", taskID))+" doesn't exist or has been deleted. Please check the ID and try again."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := fmt.Sprintf("%s: %s", utils.Italic("Title"), task.Title)
	if task.Description != "" {
		respContent += fmt.Sprintf("\n%s: %s", utils.Italic("Description"), task.Description)
	}
	if task.Role != "" {
		respContent += fmt.Sprintf("\n%s: %s", utils.Italic("Role"), task.Role)
	}
	if len(task.AssignedUsers) > 0 {
		respContent += fmt.Sprintf("\n%s:\n", utils.Italic("Assigned to"))
		for _, user := range task.AssignedUsers {
			respContent += fmt.Sprintf("  - <@%s>\n", user.DiscordID)
		}
	}
	respContent += fmt.Sprintf("\n%s: <@%s>", utils.Italic("Author"), task.Author.DiscordID)
	respContent += fmt.Sprintf("\n%s: %s %s", utils.Italic("Status"), getStatusIcon(task.Status), task.Status)

	fmt.Printf("[get-task] Retrieved task %d: %s\n", task.ID, respContent)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Components: []discordgo.MessageComponent{
				infoComponent("Task #"+fmt.Sprintf("%d", task.ID), respContent),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *DiscordBot) getTasksByRoleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != CommandGetTasksByRole {
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) != 1 {
		fmt.Printf("[get-tasks-by-role] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid options", "You must provide exactly one option (role)."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if options[0].Name != "role" && options[0].Type != discordgo.ApplicationCommandOptionRole {
		fmt.Printf("[get-tasks-by-role] Invalid option provided: %s\n", options[0].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid role", "You must provide a valid role."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	role := options[0].RoleValue(s, b.guildID).Name
	tasks, err := b.db.GetTasksByRole(role)
	if err != nil {
		fmt.Printf("[get-tasks-by-role] Failed to get tasks: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					errorComponent("Failed to retrieve tasks", fmt.Sprintf("Failed to retrieve tasks for role `%s`. An error occurred while fetching tasks. Please try again or contact an administrator.", role)),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if len(tasks) == 0 {
		fmt.Printf("[get-tasks-by-role] No tasks found for role %s\n", role)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					infoComponent("No tasks found", fmt.Sprintf("No active tasks found for role %s. This role currently has no active tasks.", utils.InlineCode(role))),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := ""
	for _, task := range tasks {
		respContent += "--------------------------------\n"
		respContent += fmt.Sprintf("%s: %d", utils.Italic("ID"), task.ID)
		respContent += fmt.Sprintf("\n%s: %s", utils.Italic("Title"), task.Title)
		if task.Description != "" {
			respContent += fmt.Sprintf("\n%s: %s", utils.Italic("Description"), task.Description)
		}
		respContent += fmt.Sprintf("\n%s: <@%s>", utils.Italic("Author"), task.Author.DiscordID)
		if len(task.AssignedUsers) > 0 {
			respContent += fmt.Sprintf("\n%s:\n", utils.Italic("Assigned to"))
			for _, user := range task.AssignedUsers {
				respContent += fmt.Sprintf("  - <@%s>\n", user.DiscordID)
			}
		}
		respContent += fmt.Sprintf("\n%s: %s %s", utils.Italic("Status"), getStatusIcon(task.Status), task.Status)
		respContent += "\n"
	}

	fmt.Printf("[get-tasks-by-role] Retrieved %d tasks for role %s\n", len(tasks), role)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Components: []discordgo.MessageComponent{
				listComponent(fmt.Sprintf("Tasks for role %s", utils.InlineCode(role)), respContent),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *DiscordBot) getUnassignedTasksByRoleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != CommandUnassignedTasksByRole {
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) != 1 {
		fmt.Printf("[unassigned-tasks-by-role] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid options", "You must provide exactly one option (role)."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if options[0].Name != "role" && options[0].Type != discordgo.ApplicationCommandOptionRole {
		fmt.Printf("[unassigned-tasks-by-role] Invalid option provided: %s\n", options[0].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid role", "You must provide a valid role."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	role := options[0].RoleValue(s, b.guildID).Name
	tasks, err := b.db.GetUnassignedTasksByRole(role)
	if err != nil {
		fmt.Printf("[unassigned-tasks-by-role] Failed to get tasks: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					errorComponent("Failed to retrieve tasks", fmt.Sprintf("Failed to retrieve tasks for role %s. An error occurred while fetching tasks. Please try again or contact an administrator.", utils.InlineCode(role))),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if len(tasks) == 0 {
		fmt.Printf("[unassigned-tasks-by-role] No unassigned tasks found for role %s\n", role)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					infoComponent("No unassigned tasks found", fmt.Sprintf("No unassigned tasks found for role %s. This role currently has no unassigned tasks.", utils.InlineCode(role))),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := ""
	for _, task := range tasks {
		respContent += "--------------------------------\n"
		respContent += fmt.Sprintf("%s: %d", utils.Italic("ID"), task.ID)
		respContent += fmt.Sprintf("\n%s: %s", utils.Italic("Title"), task.Title)
		if task.Description != "" {
			respContent += fmt.Sprintf("\n%s: %s", utils.Italic("Description"), task.Description)
		}
		respContent += fmt.Sprintf("\n%s: <@%s>", utils.Italic("Author"), task.Author.DiscordID)
		respContent += fmt.Sprintf("\n%s: %s %s", utils.Italic("Status"), getStatusIcon(task.Status), task.Status)
		respContent += "\n"
	}

	fmt.Printf("[unassigned-tasks-by-role] Retrieved %d unassigned tasks for role %s\n", len(tasks), role)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Components: []discordgo.MessageComponent{
				listComponent(fmt.Sprintf("Unassigned tasks for role %s", utils.InlineCode(role)), respContent),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *DiscordBot) assignTaskCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != CommandAssignTask {
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) != 2 {
		fmt.Printf("[assign-task] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid options", "You must provide exactly two options (task ID and user ID)."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if options[0].Name != "task-id" && options[0].Type != discordgo.ApplicationCommandOptionInteger {
		fmt.Printf("[assign-task] Invalid option provided: %s\n", options[0].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid task ID", "You must provide a valid integer ID for the task."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
	}

	if options[1].Name != "user-id" && options[1].Type != discordgo.ApplicationCommandOptionUser {
		fmt.Printf("[assign-task] Invalid option provided: %s\n", options[1].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid user ID", "You must provide a valid user ID."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	taskID := options[0].IntValue()
	userDiscordID := options[1].UserValue(s).ID

	user, err := b.getOrCreateUser(userDiscordID, options[1].UserValue(s).Username)
	if err != nil {
		fmt.Printf("[assign-task] Failed to get user: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					errorComponent("Failed to assign task", "An error occurred while assigning the task. Please try again or contact an administrator."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	err = b.db.AssignTask(uint(taskID), user.ID)
	if err != nil {
		fmt.Printf("[assign-task] Failed to assign task: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					errorComponent("Failed to assign task", "An error occurred while assigning the task. Please try again or contact an administrator."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := fmt.Sprintf("%s: %s\n%s: <@%s>", utils.Italic("Task ID"), utils.InlineCode(fmt.Sprintf("%d", taskID)), utils.Italic("Assigned to"), userDiscordID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Components: []discordgo.MessageComponent{
				confirmComponent("Task assigned successfully!", respContent),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	task, err := b.db.GetTaskByID(uint(taskID))
	if err != nil {
		fmt.Printf("[assign-task] Failed to get task: %v\n", err)
		return
	}
	message := fmt.Sprintf("👋 Hi <@%s>! You have been assigned a new task:\n\n%s", userDiscordID, utils.Bold(task.Title))
	if task.Description != "" {
		message += fmt.Sprintf("\n%s", task.Description)
	}
	message += fmt.Sprintf("\n\n🔗 %s: %s", utils.Italic("Task ID"), utils.InlineCode(fmt.Sprintf("%d", task.ID)))
	err = b.sendPrivateMessage(userDiscordID, message)
	if err != nil {
		fmt.Printf("[create-task] Failed to send private message to assignee: %v\n", err)
	}
}

func (b *DiscordBot) updateTaskStatusCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != CommandUpdateTaskStatus {
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) != 2 {
		fmt.Printf("[update-task-status] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid options", "You must provide exactly two options (task ID and status)."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if options[0].Name != "task-id" && options[0].Type != discordgo.ApplicationCommandOptionInteger {
		fmt.Printf("[update-task-status] Invalid option provided: %s\n", options[0].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid task ID", "You must provide a valid integer ID for the task."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if options[1].Name != "status" && options[1].Type != discordgo.ApplicationCommandOptionString {
		fmt.Printf("[update-task-status] Invalid option provided: %s\n", options[1].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid status", "You must provide a valid status string."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	taskID := options[0].IntValue()
	status := options[1].StringValue()

	task, err := b.db.UpdateTaskStatus(uint(taskID), status)
	if err != nil {
		fmt.Printf("[update-task-status] Failed to update task status: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					errorComponent("Failed to update task status", err.Error()),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := fmt.Sprintf("%s: %s\n%s: %s\n%s: %s %s", utils.Italic("Task ID"), utils.InlineCode(fmt.Sprintf("%d", taskID)), utils.Italic("Title"), task.Title, utils.Italic("New Status"), getStatusIcon(status), status)
	fmt.Printf("[update-task-status] Task %d status updated to: %s\n", taskID, status)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Components: []discordgo.MessageComponent{
				confirmComponent("Task status updated successfully!", respContent),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	allUsers := []db.User{task.Author}
	allUsers = append(allUsers, task.AssignedUsers...)
	// Notify all assigned users about the status update
	for _, user := range allUsers {
		message := fmt.Sprintf(
			"📢 %s\n\nYour task has been updated:\n\n%s\n\n🔗 %s: %s\n📊 %s: %s %s\n\nUpdated by: <@%s>",
			utils.Bold("Task Status Update"),
			utils.Bold(task.Title),
			utils.Italic("Task ID"),
			utils.InlineCode(fmt.Sprintf("%d", task.ID)),
			utils.Italic("New Status"),
			getStatusIcon(status),
			status,
			i.Member.User.ID)

		err = b.sendPrivateMessage(user.DiscordID, message)
		if err != nil {
			fmt.Printf("[update-task-status] Failed to send notification to assignee %s: %v\n", user.DiscordID, err)
		}
	}
}

func (b *DiscordBot) getCompletedTasksByRoleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != CommandCompletedTasksByRole {
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) != 1 {
		fmt.Printf("[completed-tasks-by-role] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid options", "You must provide exactly one option (role)."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if options[0].Name != "role" && options[0].Type != discordgo.ApplicationCommandOptionRole {
		fmt.Printf("[completed-tasks-by-role] Invalid option provided: %s\n", options[0].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid role", "You must provide a valid role."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	role := options[0].RoleValue(s, b.guildID).Name
	tasks, err := b.db.GetCompletedTasksByRole(role)
	if err != nil {
		fmt.Printf("[completed-tasks-by-role] Failed to get tasks: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					errorComponent("Failed to retrieve completed tasks", fmt.Sprintf("Failed to retrieve completed tasks for role %s. An error occurred while fetching tasks. Please try again or contact an administrator.", utils.InlineCode(role))),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if len(tasks) == 0 {
		fmt.Printf("[completed-tasks-by-role] No completed tasks found for role %s\n", role)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					infoComponent("No completed tasks found", fmt.Sprintf("No completed tasks found for role %s. This role currently has no completed tasks.", utils.InlineCode(role))),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := ""
	for _, task := range tasks {
		respContent += "--------------------------------\n"
		respContent += fmt.Sprintf("%s: %d", utils.Italic("ID"), task.ID)
		respContent += fmt.Sprintf("\n%s: %s", utils.Italic("Title"), task.Title)
		if task.Description != "" {
			respContent += fmt.Sprintf("\n%s: %s", utils.Italic("Description"), task.Description)
		}
		respContent += fmt.Sprintf("\n%s: <@%s>", utils.Italic("Author"), task.Author.DiscordID)
		if len(task.AssignedUsers) > 0 {
			respContent += fmt.Sprintf("\n%s:\n", utils.Italic("Assigned to"))
			for _, user := range task.AssignedUsers {
				respContent += fmt.Sprintf("  - <@%s>\n", user.DiscordID)
			}
		}
		respContent += fmt.Sprintf("\n%s: %s %s", utils.Italic("Status"), getStatusIcon(task.Status), task.Status)
		respContent += "\n"
	}

	fmt.Printf("[completed-tasks-by-role] Retrieved %d completed tasks for role %s\n", len(tasks), role)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Components: []discordgo.MessageComponent{
				listComponent(fmt.Sprintf("Completed tasks for role %s", utils.InlineCode(role)), respContent),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *DiscordBot) deleteTaskCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != CommandDeleteTask {
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) != 1 {
		fmt.Printf("[delete-task] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid options", "You must provide exactly one option (task ID)."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	taskID := options[0].IntValue()

	// Get task details before deletion for notifications
	task, err := b.db.GetTaskByID(uint(taskID))
	if err != nil {
		fmt.Printf("[delete-task] Failed to get task details: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					errorComponent("Failed to delete task", "An error occurred while retrieving task details. Please try again or contact an administrator."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	err = b.db.DeleteTask(uint(taskID))
	if err != nil {
		fmt.Printf("[delete-task] Failed to delete task: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					errorComponent("Failed to delete task", "An error occurred while deleting the task. Please try again or contact an administrator."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := fmt.Sprintf("%s: %s", utils.Italic("Task ID"), utils.InlineCode(fmt.Sprintf("%d", taskID)))
	fmt.Printf("[delete-task] Task %d deleted successfully\n", taskID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Components: []discordgo.MessageComponent{
				confirmComponent("Task deleted successfully!", respContent),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	// Notify all users involved about the task deletion
	allUsers := []db.User{task.Author}
	allUsers = append(allUsers, task.AssignedUsers...)

	for _, user := range allUsers {
		message := fmt.Sprintf(
			"🗑️ %s\n\nA task you were involved with has been deleted:\n\n%s\n\n🔗 %s: %s\n\nDeleted by: <@%s>",
			utils.Bold("Task Deleted"),
			utils.Bold(task.Title),
			utils.Italic("Task ID"),
			utils.InlineCode(fmt.Sprintf("%d", task.ID)),
			i.Member.User.ID)

		err = b.sendPrivateMessage(user.DiscordID, message)
		if err != nil {
			fmt.Printf("[delete-task] Failed to send notification to user %s: %v\n", user.DiscordID, err)
		}
	}
}

func (b *DiscordBot) subscribeChannelToPushWebhookCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != CommandSubscribeChannelToPush {
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) != 1 {
		fmt.Printf("[subscribe-channel-to-push] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid options", "You must provide exactly one option (repository)."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if options[0].Name != "repository" && options[0].Type != discordgo.ApplicationCommandOptionString {
		fmt.Printf("[subscribe-channel-to-push] Invalid option provided: %s\n", options[0].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid repository", "You must provide a valid repository name."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	repo := options[0].StringValue()

	_, err := b.db.CreateWebhookSubscription(repo, i.ChannelID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			fmt.Printf("[subscribe-channel-to-push] Channel %s already subscribed to push webhook for repository %s\n", i.ChannelID, repo)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Components: []discordgo.MessageComponent{
						infoComponent("Channel already subscribed", "The channel is already subscribed to the push webhook for the repository."),
					},
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		fmt.Printf("[subscribe-channel-to-push] Failed to create webhook subscription: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					errorComponent("Failed to subscribe channel to push", "An error occurred while subscribing the channel to the push webhook. Please try again or contact an administrator."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := fmt.Sprintf("%s: %s\n%s: %s", utils.Italic("Repository"), utils.InlineCode(repo), utils.Italic("Channel"), utils.InlineCode(i.ChannelID))
	fmt.Printf("[subscribe-channel-to-push] Channel %s subscribed to push webhook for repository %s\n", i.ChannelID, repo)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Components: []discordgo.MessageComponent{
				confirmComponent(fmt.Sprintf("Channel subscribed to push webhook for repository %s", utils.InlineCode(repo)), respContent),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *DiscordBot) unsubscribeChannelFromPushWebhookCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != CommandUnsubscribeChannelFromPush {
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) != 1 {
		fmt.Printf("[unsubscribe-channel-from-push] Invalid number of options provided: %d\n", len(options))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid options", "You must provide exactly one option (repository)."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if options[0].Name != "repository" && options[0].Type != discordgo.ApplicationCommandOptionString {
		fmt.Printf("[unsubscribe-channel-from-push] Invalid option provided: %s\n", options[0].Name)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					warningComponent("Invalid repository", "You must provide a valid repository name."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
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
				Components: []discordgo.MessageComponent{
					errorComponent("Failed to unsubscribe channel from push", "An error occurred while unsubscribing the channel from the push webhook. Please try again or contact an administrator."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := fmt.Sprintf("%s: %s\n%s: %s", utils.Italic("Repository"), utils.InlineCode(repo), utils.Italic("Channel"), utils.InlineCode(i.ChannelID))
	fmt.Printf("[unsubscribe-channel-from-push] Channel %s unsubscribed from push webhook for repository %s\n", i.ChannelID, repo)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Components: []discordgo.MessageComponent{
				confirmComponent(fmt.Sprintf("Channel unsubscribed from push webhook for repository %s", utils.InlineCode(repo)), respContent),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *DiscordBot) getSubscriptionsOfChannelCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != CommandGetSubscriptionsOfChannel {
		return
	}

	subscriptions, err := b.db.GetWebhookSubscriptionsByRepository(i.ChannelID)
	if err != nil {
		fmt.Printf("[get-subscriptions-of-channel] Failed to get webhook subscriptions by repository: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					errorComponent("Failed to get webhook subscriptions", "An error occurred while fetching the webhook subscriptions. Please try again or contact an administrator."),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	respContent := ""
	for _, subscription := range subscriptions {
		respContent += fmt.Sprintf("%s: %s\n", utils.Italic("Repository"), utils.InlineCode(subscription.Repository.Name))
	}

	fmt.Printf("[get-subscriptions-of-channel] Retrieved %d webhook subscriptions for channel %s\n", len(subscriptions), i.ChannelID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Components: []discordgo.MessageComponent{
				listComponent("Subscriptions of channel", respContent),
			},
			Flags: discordgo.MessageFlagsEphemeral,
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

// getOrCreateUser retrieves a user by Discord ID or creates a new one if not found
func (b *DiscordBot) getOrCreateUser(discordID, username string) (*db.User, error) {
	user, err := b.db.GetUserByDiscordID(discordID, &db.UserRetrieveOptions{
		WithAssignedTasks: true,
	})

	if err == nil {
		return user, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// User not found, create new one
	user = &db.User{
		Username:  username,
		DiscordID: discordID,
	}

	err = b.db.CreateUser(user)
	if err != nil {
		return nil, err
	}

	return user, nil
}
