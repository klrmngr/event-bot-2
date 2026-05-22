package main

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

var rsvpReactionEmojis = []struct {
	emoji    string
	response string
}{
	{"✅", "yes"},
	{"❓", "maybe"},
	{"❌", "no"},
}

func addRSVPReactions(s *discordgo.Session, channelID, messageID string) {
	for _, r := range rsvpReactionEmojis {
		if err := s.MessageReactionAdd(channelID, messageID, r.emoji); err != nil {
			log.Printf("Failed to add RSVP reaction %s: %v", r.emoji, err)
		}
	}
}

func responseFromReactionEmoji(emoji discordgo.Emoji) string {
	if emoji.ID != "" {
		return ""
	}
	switch emoji.Name {
	case "✅", "white_check_mark", "☑️", "✔️":
		return "yes"
	case "❓", "question", "?":
		return "maybe"
	case "❌", "x", "❎", "negative_squared_cross_mark":
		return "no"
	default:
		return ""
	}
}

func reactionUserID(r *discordgo.MessageReaction) string {
	if r == nil {
		return ""
	}
	if r.UserID != "" {
		return r.UserID
	}
	return ""
}

func onReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r == nil || r.MessageReaction == nil {
		return
	}
	userID := reactionUserID(r.MessageReaction)
	if userID == "" && r.Member != nil && r.Member.User != nil {
		userID = r.Member.User.ID
	}
	if userID == "" || userID == s.State.User.ID {
		return
	}

	response := responseFromReactionEmoji(r.Emoji)
	if response == "" {
		return
	}

	ev, err := GetEventByChannel(r.ChannelID)
	if err != nil || ev.MessageID == "" || ev.MessageID != r.MessageID {
		return
	}

	if err := UpsertResponse(ev.ID, userID, response); err != nil {
		log.Printf("Failed to persist RSVP from reaction: %v", err)
		return
	}

	clearOtherRSVPReactions(s, r.ChannelID, r.MessageID, userID, r.Emoji)
	refreshEventMessage(s, r.ChannelID)
}

func onReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	if r == nil || r.MessageReaction == nil {
		return
	}
	userID := reactionUserID(r.MessageReaction)
	if userID == "" || userID == s.State.User.ID {
		return
	}

	response := responseFromReactionEmoji(r.Emoji)
	if response == "" {
		return
	}

	ev, err := GetEventByChannel(r.ChannelID)
	if err != nil || ev.MessageID == "" || ev.MessageID != r.MessageID {
		return
	}

	current, err := GetUserResponse(ev.ID, userID)
	if err != nil {
		log.Printf("Failed to load RSVP for reaction remove: %v", err)
		return
	}
	if current != response {
		return
	}

	if err := DeleteResponse(ev.ID, userID); err != nil {
		log.Printf("Failed to remove RSVP from reaction: %v", err)
		return
	}
	refreshEventMessage(s, r.ChannelID)
}

func clearOtherRSVPReactions(s *discordgo.Session, channelID, messageID, userID string, added discordgo.Emoji) {
	addedResponse := responseFromReactionEmoji(added)
	if addedResponse == "" {
		return
	}
	for _, r := range rsvpReactionEmojis {
		if r.response == addedResponse {
			continue
		}
		if err := s.MessageReactionRemove(channelID, messageID, r.emoji, userID); err != nil {
			log.Printf("Failed to clear RSVP reaction %s for user %s: %v", r.emoji, userID, err)
		}
	}
}
