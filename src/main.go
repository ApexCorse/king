package main

import (
	"log"
	"os"

	"github.com/Formula-SAE/discord/src/internal/api"
	"github.com/Formula-SAE/discord/src/internal/messages"
	d "github.com/Formula-SAE/discord/src/internal/messages/discord"
	t "github.com/Formula-SAE/discord/src/internal/messages/telegram"
	"github.com/gorilla/mux"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func main() {
	// env vars
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	discordToken := os.Getenv("DISCORD_TOKEN")
	telegramEnabled := os.Getenv("TELEGRAM_ENABLED")
	discordEnabled := os.Getenv("DISCORD_ENABLED")
	address := os.Getenv("ADDRESS")
	masterToken := os.Getenv("MASTER_TOKEN")
	dbUrl := os.Getenv("DB_URL")
	if masterToken == "" {
		log.Fatalln("env: MASTER_TOKEN not specified")
	}

	if dbUrl == "" {
		log.Fatalln("env: DB_URL not specified")
	}

	if address == "" {
		log.Println("Address not specified, setting to :8080")
		address = ":8080"
	}

	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: "libsql",
		DSN: dbUrl,
	}))
	if err != nil {
		log.Fatalf("Can't create db connection: %s\n", err.Error())
	}

	log.Println("Auto-migrating db via GORM...")
	db.AutoMigrate(&api.Token{})

	log.Println("Creating router...")
	router := mux.NewRouter()

	telegram, err := t.NewTelegramBot(telegramToken, telegramEnabled == "true")
	if err != nil {
		log.Fatalf("Can't create telegram bot: %s\n", err.Error())
	}

	discord, err := d.NewDiscordBot(discordToken, discordEnabled == "true")
	if err != nil {
		log.Fatalf("Can't create discord bot: %s\n", err.Error())
	}

	providerGroup := messages.NewProviderGroup(telegram, discord)

	log.Printf("Starting server on port %s\n", address)
	api := api.NewAPI(address, router, db, providerGroup, masterToken)

	api.Start()
}
