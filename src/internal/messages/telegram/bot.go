package telegram

import (
	"github.com/Formula-SAE/discord/src/internal/messages"

	tapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type TelegramBot struct {
	providerName string

	bot *tapi.BotAPI
}

func NewTelegramBot(token string) (*TelegramBot, error) {
	bot, err := tapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &TelegramBot{
		bot:          bot,
		providerName: "telegram",
	}, nil
}

func (t *TelegramBot) SendMessage(configs ...messages.MessageConfig) error {
	for _, c := range configs {
		if c.Provider != t.providerName {
			continue
		}

		chatID, ok := c.Channel.(int64)
		if !ok {
			continue
		}

		msg := tapi.NewMessage(chatID, c.Text)
		_, err := t.bot.Send(msg)
		return err
	}

	return nil
}
