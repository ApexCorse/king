package discord

import (
	"fmt"

	"github.com/Formula-SAE/discord/internal/utils"
	"github.com/bwmarrin/discordgo"
)

func confirmComponent(title, body string) discordgo.MessageComponent {
	return discordgo.TextDisplay{
		Content: fmt.Sprintf("✅: %s\n\n%s", utils.H1(title), body),
	}
}

func infoComponent(title, body string) discordgo.MessageComponent {
	return discordgo.TextDisplay{
		Content: fmt.Sprintf("ℹ️: %s\n\n%s", utils.H1(title), body),
	}
}

func errorComponent(title, body string) discordgo.MessageComponent {
	return discordgo.TextDisplay{
		Content: fmt.Sprintf("❌: %s\n\n%s", utils.H1(title), body),
	}
}

func warningComponent(title, body string) discordgo.MessageComponent {
	return discordgo.TextDisplay{
		Content: fmt.Sprintf("⚠️: %s\n\n%s", utils.H1(title), body),
	}
}

func listComponent(title, body string) discordgo.MessageComponent {
	return discordgo.TextDisplay{
		Content: fmt.Sprintf("📋: %s\n\n%s", utils.H1(title), body),
	}
}