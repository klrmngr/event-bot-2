package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func registerEventCreation(s *discordgo.Session, guildID string) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "event",
		Description: "Create an event.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "event_name",
				Description: "Name of the event",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "time",
				Description: "Time/date of the event (flexible formats like YYYY-MM-DD HH:MM:SS; partials accepted e.g. 2025-05)",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "location",
				Description: "Location of the event",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "price",
				Description: "Price of the event (default: Free)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "emoji",
				Description: "Custom emoji for the event (default: :loudspeaker:)",
				Required:    false,
			},
		},
	}

	_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd)
	if err != nil {
		log.Printf("Cannot create '/event' command: %v", err)
	}
}

func handleEventCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// only handle application command interactions here
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	if i.ApplicationCommandData().Name != "event" {
		return
	}
	options := i.ApplicationCommandData().Options
	var eventName, location, price, emoji string
	var timeStr string
	for _, opt := range options {
		switch opt.Name {
		case "event_name":
			eventName = opt.StringValue()
		case "time":
			timeStr = opt.StringValue()
		case "location":
			location = opt.StringValue()
		case "price":
			price = opt.StringValue()
		case "emoji":
			emoji = opt.StringValue()
		}
	}
	if price == "" {
		price = "Free"
	}
	if emoji == "" {
		emoji = ":loudspeaker:"
	}

	// parse flexible time input (several date formats) before creating channel
	when, perr := ParseFlexibleTime(timeStr)
	if perr != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Please provide a valid time (formats like YYYY-MM-DD HH:MM:SS).", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	// Find "Active Plans" category
	categories, _ := s.GuildChannels(i.GuildID)
	var categoryID string
	for _, c := range categories {
		if c.Type == discordgo.ChannelTypeGuildCategory && strings.ToLower(c.Name) == "active plans" {
			categoryID = c.ID
			break
		}
	}

	// Set up permissions
	overwrites := []*discordgo.PermissionOverwrite{
		{
			ID:    i.GuildID,
			Type:  discordgo.PermissionOverwriteTypeRole,
			Allow: 0,
			Deny:  discordgo.PermissionViewChannel,
		},
		{
			ID:    i.Member.User.ID,
			Type:  discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionAllChannel ,
			Deny:  0,
		},
	}

	channelName := strings.ReplaceAll(strings.ToLower(eventName), " ", "-")
	ch, err := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{
		Name:                 channelName,
		Type:                 discordgo.ChannelTypeGuildText,
		ParentID:             categoryID,
		Topic:                fmt.Sprintf("Event planning for %s", eventName),
		PermissionOverwrites: overwrites,
	})
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to create event channel.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}


	// Render the message before touching the DB — build it inline since no DB row exists yet.
	timeDisplay := "TBD"
	if !when.IsZero() {
		loc, lerr := time.LoadLocation("America/Chicago")
		if lerr != nil {
			loc = time.FixedZone("CST", -6*3600)
		}
		human := when.In(loc).Format("January 2, 2006, 3:04 PM")
		timeDisplay = fmt.Sprintf("%s CST, <t:%d:R>", human, when.Unix())
	}
	rendered := fmt.Sprintf("%s **%s**\nOrganized by: <@%s>\n\n:date: Date: %s\n:round_pushpin: Location: %s\n:dollar: Price: %s\n\n:white_check_mark: Going: (0)\n\n:question: Maybe: (0)\n\n:x: Can't make it: (0)\n\n:pencil: Notes:\n", emoji, eventName, i.Member.User.ID, timeDisplay, location, price)

	sent, err := s.ChannelMessageSend(ch.ID, rendered)
	if err != nil {
		log.Printf("Failed to send event message: %v", err)
	} else {
		if err := upsertChannel(ch.ID, channelName); err != nil {
			log.Printf("Failed to upsert channel: %v", err)
		}
		if _, err := CreateEvent(ch.ID, sent.ID, emoji, eventName, location, price, i.Member.User.ID, when); err != nil {
			log.Printf("Failed to persist event to DB: %v", err)
		} else {
			addRSVPReactions(s, ch.ID, sent.ID)
		}
		if err := InsertMessage(sent.ID, ch.ID, channelName, s.State.User.ID, s.State.User.Username, sent.Content); err != nil {
			log.Printf("Failed to insert bot message into DB: %v", err)
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Event channel '%s' created!", channelName),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
