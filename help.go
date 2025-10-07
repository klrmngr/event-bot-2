package main

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

func registerHelp(s *discordgo.Session, guildID string) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "help",
		Description: "Get a list of available commands.",
	}
	_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd)
	if err != nil {
		log.Printf("Cannot create '/help' command: %v", err)
	}
}

func handleHelpCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	if i.ApplicationCommandData().Name != "help" {
		return
	}
	helpMessage := "**Available Commands:**\n" +
		"1. `/help` - Get a list of available commands.\n" +
		"2. `/event [name] [time] [location] [emoji] [price]` - Announce an event in the current channel.\n" +
		"3. `/rsvp [yes/no/maybe] (@user optional) - RSVP to an event; you can RSVP for others by mentioning them (e.g. <@123...>).\n" +
		"4. `/change_name [name]` - Change the name of the event.\n" +
		"5. `/change_date [new_date]` - Change the event's date/time in the current channel.\n" +
		"6. `/change_location [new_location]` - Change the event location.\n" +
		"7. `/change_price [new_price]` - Change the event price.\n" +
		"8. `/change_notes` - Start an interactive notes update (DM flow).\n" +
		"9. `/change_emoji [new_emoji]` - Change the event emoji.\n"

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: helpMessage,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Failed to send help message: %v", err)
	}
}
