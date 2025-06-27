package discord

import (
	"log"

	"github.com/Formula-SAE/discord/src/internal/messages"
	"github.com/bwmarrin/discordgo"
)

type DiscordBot struct {
	providerName string

	bot     *discordgo.Session
	enabled bool
}

func NewDiscordBot(token string, enabled bool) (*DiscordBot, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	return &DiscordBot{
		providerName: "discord",
		bot:          dg,
		enabled:      enabled,
	}, nil
}

func (b *DiscordBot) SendMessage(configs ...messages.MessageConfig) error {
	for _, c := range configs {
		if c.Provider != b.providerName {
			continue
		}

		log.Printf("Channel: %s\nMessage: %s\n", c.Channel, c.Text)
		_, err := b.bot.ChannelMessageSend(c.Channel, c.Text)

		if err != nil {
			log.Printf("Error sending message to channel %s: %s\n", c.Channel, err.Error())
		} else {
			log.Printf("Sent message to channel %s\n", c.Channel)
		}
	}

	return nil
}

func (d *DiscordBot) IsEnabled() bool {
	return d.enabled
}
