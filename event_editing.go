package main

import (
	"fmt"
	"log"
	"strings"

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
				Description: "New date/time of event",
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

	// Get the first message in the channel
	msgs, err := s.ChannelMessages(channelID, 1, "", "", "")
	if err != nil || len(msgs) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Could not find the event message.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	msg := msgs[0]

	// Update the event message (replace first bold section with new name)
	content := msg.Content
	firstBoldStart := strings.Index(content, "**")
	if firstBoldStart == -1 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Could not parse the event message format.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	firstBoldEndRel := strings.Index(content[firstBoldStart+2:], "**")
	if firstBoldEndRel == -1 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Could not parse the event message format.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	firstBoldEnd := firstBoldStart + 2 + firstBoldEndRel
	updatedContent := content[:firstBoldStart] + "**" + newName + "**" + content[firstBoldEnd+2:]

	// Edit the message
	_, err = s.ChannelMessageEdit(channelID, msg.ID, updatedContent)
	if err != nil {
		log.Printf("Failed to edit event message: %v", err)
	}

	// Update the channel name (sanitize to a valid channel name)
	sanitized := strings.ReplaceAll(strings.ToLower(newName), " ", "-")
	_, err = s.ChannelEdit(channelID, &discordgo.ChannelEdit{
		Name: sanitized,
	})
	if err != nil {
		log.Printf("Failed to edit channel name: %v", err)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Event name changed to '%s'!", newName),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func handleChangeDateCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	// Get the first message in the channel
	msgs, err := s.ChannelMessages(channelID, 1, "", "", "")
	if err != nil || len(msgs) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Could not find the event message.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	msg := msgs[0]

	// Find the "Time:" line and replace it
	content := msg.Content
	lines := strings.Split(content, "\n")
	found := false
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Time:") {
			lines[idx] = fmt.Sprintf("Time: %s", newDate)
			found = true
			break
		}
	}
	if !found {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Could not find a Time: line in the event message.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	updatedContent := strings.Join(lines, "\n")

	// Edit the message
	_, err = s.ChannelMessageEdit(channelID, msg.ID, updatedContent)
	if err != nil {
		log.Printf("Failed to edit event message: %v", err)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Event date changed to '%s'!", newDate),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
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

	msgs, err := s.ChannelMessages(channelID, 1, "", "", "")
	if err != nil || len(msgs) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Could not find the event message.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	msg := msgs[0]
	content := msg.Content
	start := strings.Index(content, "**:round_pushpin: Location:**")
	if start == -1 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Could not find a Location: line.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	start += len("**:round_pushpin: Location:**")
	end := strings.Index(content[start:], "**")
	if end == -1 {
		end = len(content)
	} else {
		end = start + end
	}
	updated := content[:start] + " " + newLocation + content[end:]
	_, err = s.ChannelMessageEdit(channelID, msg.ID, updated)
	if err != nil {
		log.Printf("Failed to edit event message: %v", err)
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

	msgs, err := s.ChannelMessages(channelID, 1, "", "", "")
	if err != nil || len(msgs) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Could not find the event message.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	msg := msgs[0]
	content := msg.Content
	start := strings.Index(content, "**:dollar: Price:**")
	if start == -1 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Could not find a Price: line.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	start += len("**:dollar: Price:**")
	// find next blank line or end
	next := strings.Index(content[start:], "\n\n")
	if next == -1 {
		next = len(content)
	} else {
		next = start + next
	}
	updated := content[:start] + " " + newPrice + content[next:]
	_, err = s.ChannelMessageEdit(channelID, msg.ID, updated)
	if err != nil {
		log.Printf("Failed to edit event message: %v", err)
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("Price updated: %s", newPrice), Flags: discordgo.MessageFlagsEphemeral},
	})
}

// Register and handle change_notes (minimal DM flow — simplified: ask user for notes in channel)
func registerChangeNotes(s *discordgo.Session, guildID string) {
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
	if i.ApplicationCommandData().Name != "change_notes" {
		return
	}
	// For simplicity (no async DM flow here), respond ephemerally instructing the user how to update notes manually.
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: "Please DM the bot with the new notes (feature not implemented in Go).", Flags: discordgo.MessageFlagsEphemeral},
	})
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

	msgs, err := s.ChannelMessages(channelID, 1, "", "", "")
	if err != nil || len(msgs) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Could not find the event message.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	msg := msgs[0]
	content := msg.Content
	firstBoldStart := strings.Index(content, "**")
	if firstBoldStart == -1 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Could not parse event title format.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	firstBoldEndRel := strings.Index(content[firstBoldStart+2:], "**")
	if firstBoldEndRel == -1 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Could not parse event title format.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	firstBoldEnd := firstBoldStart + 2 + firstBoldEndRel
	boldContent := content[firstBoldStart+2 : firstBoldEnd]
	// find space between emoji and name
	spaceIdx := strings.Index(boldContent, " ")
	if spaceIdx == -1 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Could not parse event title format.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	eventName := boldContent[spaceIdx+1:]
	updated := content[:firstBoldStart] + "**" + newEmoji + " " + eventName + "**" + content[firstBoldEnd+2:]
	_, err = s.ChannelMessageEdit(channelID, msg.ID, updated)
	if err != nil {
		log.Printf("Failed to edit event message: %v", err)
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("Emoji updated to %s", newEmoji), Flags: discordgo.MessageFlagsEphemeral},
	})
}
