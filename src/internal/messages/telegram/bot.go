package telegram

import (
	"log"
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

	if enabled {
		log.Println("Telegram enabled")
	} else {
		log.Println("Telegram not enabled")
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
			log.Printf("Chat ID %s not valid: %s\n", c.Channel, err.Error())
			continue
		}

		log.Printf("ChatID: %d\nMessage: %s\n", chatID, c.Text)
		msg := tapi.NewMessage(int64(chatID), c.Text)
		msg.ParseMode = "HTML"

		_, err = t.bot.Send(msg)
		if err != nil {
			log.Printf("Error sending message to chat %d: %s\n", chatID, err.Error())
		} else {
			log.Printf("Sent message to chat %d\n", chatID)
		}
	}

	return nil
}

func (t *TelegramBot) IsEnabled() bool {
	return t.enabled
}
