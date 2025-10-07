package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func registerPokerCommands(s *discordgo.Session, guildID string) {
	sessionCmd := &discordgo.ApplicationCommand{
		Name:        "session",
		Description: "Log a poker session: /session [in] [out] (location) (stakes)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionNumber,
				Name:        "in",
				Description: "Buy-in amount (e.g. 100.00)",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionNumber,
				Name:        "out",
				Description: "Cash-out amount (e.g. 250.00)",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "location",
				Description: "Optional location",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "stakes",
				Description: "Optional stakes (e.g. 1/2)",
				Required:    false,
			},
		},
	}
	_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, sessionCmd)
	if err != nil {
		log.Printf("Cannot create '/session' command: %v", err)
	}

	lifetimeCmd := &discordgo.ApplicationCommand{
		Name:        "lifetime",
		Description: "Show lifetime poker stats for a user",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "Optional user to query",
				Required:    false,
			},
		},
	}
	_, err = s.ApplicationCommandCreate(s.State.User.ID, guildID, lifetimeCmd)
	if err != nil {
		log.Printf("Cannot create '/lifetime' command: %v", err)
	}
}

func handlePokerCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	name := i.ApplicationCommandData().Name
	switch name {
	case "session":
		handleSessionCommand(s, i)
	case "lifetime":
		handleLifetimeCommand(s, i)
	}
}

func handleSessionCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var inAmtF, outAmtF float64
	var location, stakes string
	var userID string
	for _, opt := range i.ApplicationCommandData().Options {
		switch opt.Name {
		case "in":
			if opt.Value != nil {
				inAmtF = opt.FloatValue()
			}
		case "out":
			if opt.Value != nil {
				outAmtF = opt.FloatValue()
			}
		case "location":
			location = opt.StringValue()
		case "stakes":
			stakes = opt.StringValue()
		}
	}
	userID = i.Member.User.ID
	if err := CreatePokerSession(userID, inAmtF, outAmtF, location, stakes); err != nil {
		log.Printf("Failed to create poker session: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Failed to save poker session.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	profit := outAmtF - inAmtF
	msg := fmt.Sprintf("Session logged: In=%.2f Out=%.2f Profit=%.2f", inAmtF, outAmtF, profit)
	if location != "" {
		msg += " Location=" + location
	}
	if stakes != "" {
		msg += " Stakes=" + stakes
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: msg, Flags: discordgo.MessageFlagsEphemeral},
	})
}

func handleLifetimeCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var userID string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "user" && opt.UserValue(nil) != nil {
			userID = opt.UserValue(nil).ID
		}
	}
	if userID == "" {
		userID = i.Member.User.ID
	}
	count, net, err := GetPokerLifetime(userID)
	if err != nil {
		log.Printf("Failed to query lifetime: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Failed to fetch lifetime stats.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	msg := fmt.Sprintf("Lifetime sessions for <@%s>: %d sessions, Net=%.2f", userID, count, net)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: msg, Flags: discordgo.MessageFlagsEphemeral},
	})
}

// handle message-based parsing like: /session 100 250 "Casino" "1/2"
func handlePokerMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.ID == s.State.User.ID {
		return
	}
	content := strings.TrimSpace(m.Content)
	if !strings.HasPrefix(content, "/session") && !strings.HasPrefix(content, "/lifetime") {
		return
	}
	parts := strings.Fields(content)
	if strings.HasPrefix(content, "/lifetime") {
		// /lifetime (@user optional)
		var userID string
		if m.Mentions != nil && len(m.Mentions) > 0 {
			userID = m.Mentions[0].ID
		} else if len(parts) >= 2 {
			re := regexp.MustCompile(`^<@!?(\d+)>$`)
			if sub := re.FindStringSubmatch(parts[1]); len(sub) == 2 {
				userID = sub[1]
			}
		}
		if userID == "" {
			userID = m.Author.ID
		}
		count, net, err := GetPokerLifetime(userID)
		if err != nil {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Failed to fetch lifetime stats.")
			return
		}
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Lifetime sessions for <@%s>: %d sessions, Net=%.2f", userID, count, net))
		return
	}

	// /session in out (location) (stakes)
	// Try to extract numbers and optional quoted strings
	// Basic parsing: first two numeric tokens are in and out
	if len(parts) < 3 {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Usage: /session [in] [out] (location) (stakes)")
		return
	}
	inStr := parts[1]
	outStr := parts[2]
	inAmt, err1 := strconv.ParseFloat(inStr, 64)
	outAmt, err2 := strconv.ParseFloat(outStr, 64)
	if err1 != nil || err2 != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Could not parse in/out amounts. Use numbers like 100 or 100.50")
		return
	}
	location := ""
	stakes := ""
	if len(parts) >= 4 {
		// parts[3] might be a quoted string including spaces; attempt to join remaining and pull quoted strings
		re := regexp.MustCompile(`"([^"]+)"`)
		joined := strings.Join(parts[3:], " ")
		matches := re.FindAllStringSubmatch(joined, -1)
		if len(matches) >= 1 {
			location = matches[0][1]
		}
		if len(matches) >= 2 {
			stakes = matches[1][1]
		} else if len(parts) >= 5 {
			// fallback: take the next token as stakes
			stakes = parts[4]
		}
	}
	userID := m.Author.ID
	if err := CreatePokerSession(userID, inAmt, outAmt, location, stakes); err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Failed to save poker session.")
		return
	}
	profit := outAmt - inAmt
	msg := fmt.Sprintf("Session logged: In=%.2f Out=%.2f Profit=%.2f", inAmt, outAmt, profit)
	if location != "" {
		msg += " Location=" + location
	}
	if stakes != "" {
		msg += " Stakes=" + stakes
	}
	_, _ = s.ChannelMessageSend(m.ChannelID, msg)
}
