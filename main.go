package main

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	// Load from .env (if present) and then from the environment
	if err := godotenv.Load(); err != nil {
		log.Printf(".env not found or failed to load: %v", err)
	} else {
		log.Printf("Loaded .env file")
	}

	// Load environment variables
	token := os.Getenv("DISCORD_TOKEN")
	guildID := os.Getenv("GUILD_ID")
	missing := []string{}
	if token == "" {
		missing = append(missing, "DISCORD_TOKEN")
	}
	if guildID == "" {
		missing = append(missing, "GUILD_ID")
	}
	if len(missing) > 0 {
		log.Fatalf("Missing required environment variables: %s", strings.Join(missing, ", "))
	}

	// Create and run the bot
	if err := runBot(token, guildID); err != nil {
		log.Fatalf("Bot error: %v", err)
	}
}
