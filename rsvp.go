package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func registerRSVP(s *discordgo.Session, guildID string) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "rsvp",
		Description: "RSVP for the event by choosing yes, no, or maybe",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "response",
				Description: "Your RSVP response (yes, no, maybe)",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "Optional: The user to RSVP for",
				Required:    false,
			},
		},
	}
	_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd)
	if err != nil {
		log.Printf("Cannot create '/rsvp' command: %v", err)
	}
}

func handleRSVPCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ApplicationCommandData().Name != "rsvp" {
		return
	}
	var response, userID string
	for _, opt := range i.ApplicationCommandData().Options {
		switch opt.Name {
		case "response":
			response = strings.ToLower(opt.StringValue())
		case "user":
			userID = opt.UserValue(nil).ID
		}
	}
	if response != "yes" && response != "no" && response != "maybe" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Invalid response. Please use yes, no, or maybe.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	if userID == "" {
		userID = i.Member.User.ID
	}
	userMention := fmt.Sprintf("<@%s>", userID)

	// Get the first message in the channel
	msgs, err := s.ChannelMessages(i.ChannelID, 1, "", "", "")
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
	content := msg.Content

	// Define section markers
	sections := map[string]string{
		"yes":   "**:white_check_mark: Going:**",
		"maybe": "**:question: Maybe:**",
		"no":    "**:x: Can't make it:**",
	}

	// Update RSVP section
	updatedContent := updateRSVPSection(content, sections[response], userMention)
	_, err = s.ChannelMessageEdit(i.ChannelID, msg.ID, updatedContent)
	if err != nil {
		log.Printf("Failed to update RSVP: %v", err)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("RSVP updated for %s: %s", userMention, response),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// Helper to update RSVP section in message
func updateRSVPSection(content, sectionMarker, userMention string) string {
	lines := strings.Split(content, "\n")
	var newLines []string
	foundSection := false
	for _, line := range lines {
		if strings.HasPrefix(line, sectionMarker) {
			foundSection = true
			if !strings.Contains(line, userMention) {
				line += " " + userMention
			}
		}
		newLines = append(newLines, line)
	}
	if !foundSection {
		newLines = append(newLines, sectionMarker+" "+userMention)
	}
	return strings.Join(newLines, "\n")
}
