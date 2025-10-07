package main

import (
	"fmt"
	"log"
	"strings"

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
		// Log commands and modal submits for auditing
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			// safe to call ApplicationCommandData
			name := i.ApplicationCommandData().Name
			// build a short command text from options
			var opts []string
			for _, o := range i.ApplicationCommandData().Options {
				if o.Value != nil {
					opts = append(opts, fmt.Sprintf("%s=%v", o.Name, o.Value))
				} else {
					opts = append(opts, o.Name)
				}
			}
			cmdText := name
			if len(opts) > 0 {
				cmdText = cmdText + " " + strings.Join(opts, " ")
			}
			// username is recorded/updated by DB layer; only pass user id and text here
			InsertCommand(i.Member.User.ID, i.Member.User.Username, cmdText)
		case discordgo.InteractionModalSubmit:
			// If this is our change_notes modal, capture the notes text
			if i.ModalSubmitData().CustomID == "change_notes_modal" {
				var notes string
				for _, row := range i.ModalSubmitData().Components {
					if ar, ok := row.(*discordgo.ActionsRow); ok {
						for _, comp := range ar.Components {
							if ti, ok := comp.(*discordgo.TextInput); ok {
								if ti.CustomID == "notes_input" {
									notes = ti.Value
								}
							}
						}
					}
				}
				InsertCommand(i.Member.User.ID, i.Member.User.Username, "change_notes: "+notes)
			}
		}

		// delegate to specific handlers
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
	// persist to DB
	// try to fetch channel name (best-effort)
	channelName := ""
	if ch, cerr := s.Channel(m.ChannelID); cerr == nil && ch != nil {
		channelName = ch.Name
	}
	if err := InsertMessage(m.ID, m.ChannelID, channelName, m.Author.ID, m.Author.Username, m.Content); err != nil {
		log.Printf("failed to insert message into DB: %v", err)
	}
}
