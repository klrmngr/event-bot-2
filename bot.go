package main

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

func runBot(token, guildID string) error {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return err
	}

	dg.AddHandler(onReady)
	dg.AddHandler(onMessageCreate)
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		handleEventCommand(s, i)
		handleChangeNameCommand(s, i)
		handleChangeDateCommand(s, i)
		handleChangeLocationCommand(s, i)
		handleChangePriceCommand(s, i)
		handleChangeNotesCommand(s, i)
		handleChangeEmojiCommand(s, i)
		handleRSVPCommand(s, i)
		handleHelpCommand(s, i)
	})

	// Open a websocket connection to Discord
	if err := dg.Open(); err != nil {
		return err
	}
	defer dg.Close()

	// Register slash commands (after opening so s.State is available)
	registerEventCreation(dg, guildID)
	registerEventEditing(dg, guildID)
	registerChangeDate(dg, guildID)
	registerChangeLocation(dg, guildID)
	registerChangePrice(dg, guildID)
	registerChangeNotes(dg, guildID)
	registerChangeEmoji(dg, guildID)
	registerRSVP(dg, guildID)
	registerHelp(dg, guildID)

	log.Println("Bot is now running. Press CTRL+C to exit.")
	select {} // Block forever
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("%s has connected to Discord!", s.State.User.String())
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	log.Printf("[%s] %s: %s", m.ChannelID, m.Author.Username, m.Content)
}
