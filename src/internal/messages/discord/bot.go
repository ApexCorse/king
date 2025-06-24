package discord

import (
	"github.com/Formula-SAE/discord/src/internal/messages"
	"github.com/bwmarrin/discordgo"
)

type DiscordBot struct {
	providerName string

	bot *discordgo.Session
}

func NewDiscordBot(token string) (*DiscordBot, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	return &DiscordBot{
		providerName: "discord",
		bot:          dg,
	}, nil
}

func (b *DiscordBot) SendMessage(configs ...messages.MessageConfig) error {
	for _, c := range configs {
		if c.Provider != b.providerName {
			continue
		}

		channel, ok := c.Channel.(string)
		if !ok {
			continue
		}

		_, err := b.bot.ChannelMessageSend(channel, c.Text)
		if err != nil {
			return err
		}
	}

	return discordgo.ErrNilState
}
