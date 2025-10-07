package main

import (
	"fmt"
	"log"
	"regexp"
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
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
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

	// Persist the response in the DB
	ev, err := GetEventByChannel(i.ChannelID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Could not find the event record.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	if err := UpsertResponse(ev.ID, userID, response); err != nil {
		log.Printf("Failed to persist RSVP: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Failed to save RSVP.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	// Re-render message and edit
	if ev.MessageID != "" {
		if rendered, rerr := RenderEventMessage(i.ChannelID); rerr == nil {
			if _, err := s.ChannelMessageEdit(i.ChannelID, ev.MessageID, rendered); err != nil {
				log.Printf("Failed to update RSVP message: %v", err)
			}
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("RSVP updated for %s: %s", userMention, response),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// handleRSVPMessage parses plain-text messages that start with /rsvp and
// supports the syntax: /rsvp (yes|no|maybe) (@user optional)
// Mentions in message content are like <@715414244270538754> or <@!7154...>
func handleRSVPMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.ID == s.State.User.ID {
		return
	}
	content := strings.TrimSpace(m.Content)
	if !strings.HasPrefix(content, "/rsvp") {
		return
	}
	parts := strings.Fields(content)
	if len(parts) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Usage: /rsvp (yes/no/maybe) (@user optional)")
		return
	}
	response := strings.ToLower(parts[1])
	if response != "yes" && response != "no" && response != "maybe" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Invalid response. Please use yes, no, or maybe.")
		return
	}

	// default to the message author
	userID := m.Author.ID

	// If the message has explicit mentions parsed by Discord, prefer that
	if m.Mentions != nil && len(m.Mentions) > 0 {
		userID = m.Mentions[0].ID
	} else if len(parts) >= 3 {
		// Try to parse a raw mention like <@123456789> or <@!123456789>
		re := regexp.MustCompile(`^<@!?(\d+)>$`)
		if sub := re.FindStringSubmatch(parts[2]); len(sub) == 2 {
			userID = sub[1]
		}
	}

	userMention := fmt.Sprintf("<@%s>", userID)

	ev, err := GetEventByChannel(m.ChannelID)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Could not find the event record.")
		return
	}
	if err := UpsertResponse(ev.ID, userID, response); err != nil {
		log.Printf("Failed to persist RSVP (message): %v", err)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Failed to save RSVP.")
		return
	}

	// Re-render and edit the event message if present
	if ev.MessageID != "" {
		if rendered, rerr := RenderEventMessage(m.ChannelID); rerr == nil {
			if _, err := s.ChannelMessageEdit(m.ChannelID, ev.MessageID, rendered); err != nil {
				log.Printf("Failed to update RSVP message (message): %v", err)
			}
		}
	}

	_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("RSVP updated for %s: %s", userMention, response))
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
