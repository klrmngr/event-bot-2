package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func registerEventEditing(s *discordgo.Session, guildID string) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "change_name",
		Description: "Change the name of the event in the current channel",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "new_name",
				Description: "New name of event",
				Required:    true,
			},
		},
	}
	_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd)
	if err != nil {
		log.Printf("Cannot create '/change_name' command: %v", err)
	}
}

// Register a command to change the event date/time in the current channel
func registerChangeDate(s *discordgo.Session, guildID string) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "change_date",
		Description: "Change the date/time of the event in the current channel",
		Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "new_date",
					Description: "New date/time of event (flexible formats like YYYY-MM-DD HH:MM:SS)",
					Required:    true,
				},
			},
	}
	_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd)
	if err != nil {
		log.Printf("Cannot create '/change_date' command: %v", err)
	}
}

func handleChangeNameCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	if i.ApplicationCommandData().Name != "change_name" {
		return
	}
	var newName string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "new_name" {
			newName = opt.StringValue()
		}
	}
	channelID := i.ChannelID

	// Update the channel name (sanitize to a valid channel name)
	sanitized := strings.ReplaceAll(strings.ToLower(newName), " ", "-")
	if _, err := s.ChannelEdit(channelID, &discordgo.ChannelEdit{Name: sanitized}); err != nil {
		log.Printf("Failed to edit channel name: %v", err)
	}

	// update DB
	if err := UpdateEventFieldByChannel(channelID, "title", newName); err != nil {
		log.Printf("Failed to update event title in DB: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Failed to update event in DB.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	// re-render and edit the event message
	if ev, err := GetEventByChannel(channelID); err == nil && ev.MessageID != "" {
		if rendered, rerr := RenderEventMessage(channelID); rerr == nil {
			_, _ = s.ChannelMessageEdit(channelID, ev.MessageID, rendered)
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("Event name changed to '%s'!", newName), Flags: discordgo.MessageFlagsEphemeral},
	})
}

func handleChangeDateCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	if i.ApplicationCommandData().Name != "change_date" {
		return
	}
	var newDate string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "new_date" {
			newDate = opt.StringValue()
		}
	}
	channelID := i.ChannelID

	// parse flexible input
	t, perr := ParseFlexibleTime(newDate)
	if perr != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Please provide a valid time (formats like YYYY-MM-DD HH:MM:SS).", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	newDate = t.Format(time.RFC3339)

	// update DB: store as text in "date" column
	if err := UpdateEventFieldByChannel(channelID, "date", newDate); err != nil {
		log.Printf("Failed to update event date in DB: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Failed to update event date in DB.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	if ev, err := GetEventByChannel(channelID); err == nil && ev.MessageID != "" {
		if rendered, rerr := RenderEventMessage(channelID); rerr == nil {
			_, _ = s.ChannelMessageEdit(channelID, ev.MessageID, rendered)
		}
	}

	// respond with Discord relative timestamp format
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("Event date changed to  <t:%d:R>!", t.Unix()), Flags: discordgo.MessageFlagsEphemeral},
	})
}

// Register and handle change_location
func registerChangeLocation(s *discordgo.Session, guildID string) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "change_location",
		Description: "Change the location of the event in the current channel",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "new_location",
				Description: "New location of the event",
				Required:    true,
			},
		},
	}
	_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd)
	if err != nil {
		log.Printf("Cannot create '/change_location' command: %v", err)
	}
}

func handleChangeLocationCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	if i.ApplicationCommandData().Name != "change_location" {
		return
	}
	var newLocation string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "new_location" {
			newLocation = opt.StringValue()
		}
	}
	channelID := i.ChannelID

	if err := UpdateEventFieldByChannel(channelID, "location", newLocation); err != nil {
		log.Printf("Failed to update event location in DB: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Failed to update event location in DB.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	if ev, err := GetEventByChannel(channelID); err == nil && ev.MessageID != "" {
		if rendered, rerr := RenderEventMessage(channelID); rerr == nil {
			_, _ = s.ChannelMessageEdit(channelID, ev.MessageID, rendered)
		}
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("Location updated: %s", newLocation), Flags: discordgo.MessageFlagsEphemeral},
	})
}

// Register and handle change_price
func registerChangePrice(s *discordgo.Session, guildID string) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "change_price",
		Description: "Change the price of the event in the current channel",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "new_price",
				Description: "New price of the event",
				Required:    true,
			},
		},
	}
	_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd)
	if err != nil {
		log.Printf("Cannot create '/change_price' command: %v", err)
	}
}

func handleChangePriceCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	if i.ApplicationCommandData().Name != "change_price" {
		return
	}
	var newPrice string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "new_price" {
			newPrice = opt.StringValue()
		}
	}
	channelID := i.ChannelID

	if err := UpdateEventFieldByChannel(channelID, "price", newPrice); err != nil {
		log.Printf("Failed to update event price in DB: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Failed to update event price in DB.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	if ev, err := GetEventByChannel(channelID); err == nil && ev.MessageID != "" {
		if rendered, rerr := RenderEventMessage(channelID); rerr == nil {
			_, _ = s.ChannelMessageEdit(channelID, ev.MessageID, rendered)
		}
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("Price updated: %s", newPrice), Flags: discordgo.MessageFlagsEphemeral},
	})
}

// Register and handle change_notes (minimal DM flow — simplified: ask user for notes in channel)
func registerChangeNotes(s *discordgo.Session, guildID string) {
	// create a slash command without options; we'll show a modal to collect notes
	cmd := &discordgo.ApplicationCommand{
		Name:        "change_notes",
		Description: "Change the notes for the event in the current channel",
	}
	_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd)
	if err != nil {
		log.Printf("Cannot create '/change_notes' command: %v", err)
	}
}

func handleChangeNotesCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// If this is a modal submit for our change_notes modal, handle the save/update flow
	if i.Type == discordgo.InteractionModalSubmit {
		if i.ModalSubmitData().CustomID == "change_notes_modal" {
			channelID := i.ChannelID
			// extract text input value from modal components
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

			if err := UpdateEventFieldByChannel(channelID, "description", notes); err != nil {
				log.Printf("Failed to update event notes in DB: %v", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Content: "Failed to update event notes in DB.", Flags: discordgo.MessageFlagsEphemeral},
				})
				return
			}

			if ev, err := GetEventByChannel(channelID); err == nil && ev.MessageID != "" {
				if rendered, rerr := RenderEventMessage(channelID); rerr == nil {
					_, _ = s.ChannelMessageEdit(channelID, ev.MessageID, rendered)
				}
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "Notes updated.", Flags: discordgo.MessageFlagsEphemeral},
			})
			return
		}
		// not our modal -> ignore
		return
	}

	// Otherwise, if this is the slash command invocation, open a modal
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	if i.ApplicationCommandData().Name != "change_notes" {
		return
	}

	// Respond with a modal asking for notes/description
	modal := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "change_notes_modal",
			Title:    "Change event notes",
			Components: []discordgo.MessageComponent{
				&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					&discordgo.TextInput{
						CustomID:    "notes_input",
						Label:       "Notes / Description",
						Style:       discordgo.TextInputParagraph,
						Required:    false,
						Placeholder: "Add or edit notes for this event...",
						MaxLength:   2000,
					},
				}},
			},
		},
	}
	if err := s.InteractionRespond(i.Interaction, modal); err != nil {
		log.Printf("failed to open modal: %v", err)
	}
}

// Register and handle change_emoji
func registerChangeEmoji(s *discordgo.Session, guildID string) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "change_emoji",
		Description: "Change the emoji for the event in the current channel",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "new_emoji",
				Description: "New emoji for the event (e.g., :tada:, :calendar:)",
				Required:    true,
			},
		},
	}
	_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd)
	if err != nil {
		log.Printf("Cannot create '/change_emoji' command: %v", err)
	}
}

func handleChangeEmojiCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	if i.ApplicationCommandData().Name != "change_emoji" {
		return
	}
	var newEmoji string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "new_emoji" {
			newEmoji = opt.StringValue()
		}
	}
	channelID := i.ChannelID

	if err := UpdateEventFieldByChannel(channelID, "emoji", newEmoji); err != nil {
		log.Printf("Failed to update event emoji in DB: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Failed to update event emoji in DB.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	if ev, err := GetEventByChannel(channelID); err == nil && ev.MessageID != "" {
		if rendered, rerr := RenderEventMessage(channelID); rerr == nil {
			_, _ = s.ChannelMessageEdit(channelID, ev.MessageID, rendered)
		}
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("Emoji updated to %s", newEmoji), Flags: discordgo.MessageFlagsEphemeral},
	})
}
