package telegram

import (
	"strconv"

	"github.com/Formula-SAE/discord/src/internal/messages"

	tapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type TelegramBot struct {
	providerName string

	bot     *tapi.BotAPI
	enabled bool
}

func NewTelegramBot(token string, enabled bool) (*TelegramBot, error) {
	bot, err := tapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &TelegramBot{
		bot:          bot,
		providerName: "telegram",
		enabled:      enabled,
	}, nil
}

func (t *TelegramBot) SendMessage(configs ...messages.MessageConfig) error {
	for _, c := range configs {
		if c.Provider != t.providerName {
			continue
		}

		chatID, err := strconv.Atoi(c.Channel)
		if err != nil {
			continue
		}

		msg := tapi.NewMessage(int64(chatID), c.Text)
		_, err = t.bot.Send(msg)
		return err
	}

	return nil
}

func (t *TelegramBot) IsEnabled() bool {
	return t.enabled
}
