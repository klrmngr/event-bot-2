package main

import (
    "bytes"
    "fmt"
    "text/template"
    "io/ioutil"
    "path/filepath"
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
    "Dates":     func() string { if ev.Date != nil { return fmt.Sprintf("<t:%d:R>", ev.Date.Unix()) }; return "TBD" }(),
        "Location":  ev.Location,
        "Price":     ev.Price,
        "Going":     mentions(goingIDs),
        "Maybe":     mentions(maybeIDs),
        "CantMakeIt": mentions(cantIDs),
        "Notes":     func() []string { if ev.Description != "" { return []string{ev.Description} } ; return []string{} }(),
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
