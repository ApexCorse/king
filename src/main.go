package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Formula-SAE/discord/src/internal/api"
	"github.com/Formula-SAE/discord/src/internal/messages"
	d "github.com/Formula-SAE/discord/src/internal/messages/discord"
	t "github.com/Formula-SAE/discord/src/internal/messages/telegram"
	"github.com/gorilla/mux"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// env vars
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	discordToken := os.Getenv("DISCORD_TOKEN")
	telegramEnabled := os.Getenv("TELEGRAM_ENABLED")
	discordEnabled := os.Getenv("DISCORD_ENABLED")
	address := os.Getenv("ADDRESS")
	masterToken := os.Getenv("MASTER_TOKEN")
	if masterToken == "" {
		panic("env: MASTER_TOKEN not specified")
	}

	if address == "" {
		address = ":8080"
	}

	db, err := gorm.Open(sqlite.Open("falkie.db"))
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&api.Token{})

	router := mux.NewRouter()

	telegram, err := t.NewTelegramBot(telegramToken, telegramEnabled == "true")
	if err != nil {
		panic(fmt.Sprintf("can't create telegram bot: %s\n", err.Error()))
	}

	discord, err := d.NewDiscordBot(discordToken, discordEnabled == "true")
	if err != nil {
		panic(fmt.Sprintf("can't create discord bot: %s\n", err.Error()))
	}

	providerGroup := messages.NewProviderGroup(telegram, discord)

	log.Printf("Starting server on port %s\n", address)
	api := api.NewAPI(address, router, db, providerGroup, masterToken)

	api.Start()
}
