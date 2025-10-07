package main

import (
	"fmt"
	"log"
	"strings"

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
				Description: "General time/date of the event",
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
	if i.ApplicationCommandData().Name != "event" {
		return
	}
	options := i.ApplicationCommandData().Options
	var eventName, time, location, price, emoji string
	for _, opt := range options {
		switch opt.Name {
		case "event_name":
			eventName = opt.StringValue()
		case "time":
			time = opt.StringValue()
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
			Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages,
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

	msg := fmt.Sprintf("%s **%s**\nTime: %s\nLocation: %s\nPrice: %s\nCreated by: <@%s>", emoji, eventName, time, location, price, i.Member.User.ID)
	_, err = s.ChannelMessageSend(ch.ID, msg)
	if err != nil {
		log.Printf("Failed to send event message: %v", err)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Event channel '%s' created!", channelName),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
