package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"text/template"
	"time"
)

// RenderEventMessage builds the event message text from the template and DB row.
func RenderEventMessage(channelID string) (string, error) {
	ev, err := GetEventByChannel(channelID)
	if err != nil {
		return "", err
	}

	// minimal data for template: RSVP lists are empty until we persist them separately
	// Fetch RSVP responses
	goingIDs, maybeIDs, cantIDs, gerr := GetResponsesForEvent(ev.ID)
	if gerr != nil {
		// ignore errors and use empty lists
		goingIDs, maybeIDs, cantIDs = []string{}, []string{}, []string{}
	}
	mentions := func(ids []string) []string {
		out := make([]string, 0, len(ids))
		for _, id := range ids {
			out = append(out, "<@"+id+">")
		}
		return out
	}
	data := map[string]interface{}{
		"Emoji":     ev.Emoji,
		"Title":     ev.Title,
		"Organizer": "<@" + ev.AuthorID + ">",
		"Dates": func() string {
			if ev.Date != nil {
				// Use America/Chicago so dates/times are shown in Chicago local time (DST-aware)
				loc, err := time.LoadLocation("America/Chicago")
				if err != nil {
					// Fallback to fixed -6h if the location can't be loaded
					loc = time.FixedZone("CST", -6*3600)
				}
				tChicago := ev.Date.In(loc)
				// Format: Month day, year, HH:MM AM/PM (e.g. "January 2, 2006, 3:04 PM")
				// Append Discord timestamp template so callers can include the relative timestamp in Discord
				human := tChicago.Format("January 2, 2006, 3:04 PM")
				discord := fmt.Sprintf("<t:%d:R>", ev.Date.Unix())
				return fmt.Sprintf("%s, %s", human, discord)
			}
			return "TBD"
		}(),
		"Location":   ev.Location,
		"Price":      ev.Price,
		"Going":      mentions(goingIDs),
		"Maybe":      mentions(maybeIDs),
		"CantMakeIt": mentions(cantIDs),
		"Notes": func() []string {
			if ev.Description != "" {
				return []string{ev.Description}
			}
			return []string{}
		}(),
	}

	tmplPath := filepath.Join(".", "event.tmpl")
	b, err := ioutil.ReadFile(tmplPath)
	if err != nil {
		return "", err
	}
	t, err := template.New("event").Parse(string(b))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
