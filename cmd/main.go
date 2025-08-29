package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Formula-SAE/discord/internal/bots/discord"
	"github.com/Formula-SAE/discord/internal/db"
	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/mux"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func main() {
	log.Println("=== Starting King Discord Bot ===")

	// env vars
	discordToken := os.Getenv("DISCORD_TOKEN")
	address := os.Getenv("ADDRESS")
	dbUrl := os.Getenv("DB_URL")
	appID := os.Getenv("APPLICATION_ID")
	guildID := os.Getenv("GUILD_ID")

	log.Println("Checking environment variables...")
	if discordToken == "" || dbUrl == "" || appID == "" || guildID == "" {
		log.Fatalf("Missing required environment variables: DISCORD_TOKEN=%t, DB_URL=%t, APP_ID=%t, GUILD_ID=%t",
			discordToken != "", dbUrl != "", appID != "", guildID != "")
	}
	log.Println("Environment variables validated successfully")

	if address == "" {
		log.Println("Address not specified, setting to :8080")
		address = ":8080"
	} else {
		log.Printf("Using address: %s", address)
	}

	log.Println("Initializing database connection...")
	gormDB, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: "libsql",
		DSN:        dbUrl,
	}))
	if err != nil {
		log.Fatalf("Failed to create database connection: %v", err)
	}
	log.Println("Database connection established successfully")

	log.Println("Running database migrations...")
	err = gormDB.AutoMigrate(&db.User{}, &db.Task{}, &db.TaskComment{}, &db.WebhookSubscriptions{})
	if err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}
	log.Println("Database migrations completed successfully")

	DB := db.NewDB(gormDB)
	log.Println("Database wrapper initialized")

	log.Println("Initializing Discord session...")
	session, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		log.Fatalf("Failed to create Discord session: %v", err)
	}
	log.Println("Discord session created successfully")

	router := mux.NewRouter()

	log.Println("Creating Discord bot instance...")
	discordBot := discord.NewDiscordBot(session, DB, appID, guildID, router)

	log.Println("Starting Discord bot...")
	close, err := discordBot.Start()
	if err != nil {
		panic(err)
	}
	defer close()
	log.Println("Discord bot started successfully")

	log.Println("Setting up signal handlers for graceful shutdown...")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Bot is running. Press Ctrl+C to stop.")
	<-stop

	log.Println("Received shutdown signal, stopping bot...")
	log.Println("=== King Discord Bot stopped ===")
}
