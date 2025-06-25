package discord

import (
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

		_, err := b.bot.ChannelMessageSend(c.Channel, c.Text)
		if err != nil {
			return err
		}
	}

	return discordgo.ErrNilState
}

func (d *DiscordBot) IsEnabled() bool {
	return d.enabled
}
