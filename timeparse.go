package main

import (
    "fmt"
    "strings"
    "time"
)

// ParseFlexibleTime accepts flexible
// date inputs like:
// - 2025
// - 2025-05
// - 2025-05-02
// - 2025-05-02 15:04
// - 2025-05-02 15:04:05
// Missing components default to the first valid value (start of period).
func ParseFlexibleTime(input string) (time.Time, error) {
    s := strings.TrimSpace(input)
    if s == "" {
        return time.Time{}, fmt.Errorf("empty input")
    }

    // Normalize separators and split date and time
    fields := strings.Fields(s)
    datePart := fields[0]
    timePart := ""
    if len(fields) > 1 {
        timePart = fields[1]
    }

    dateSegs := strings.Split(datePart, "-")
    year := "0000"
    month := "01"
    day := "01"
    switch len(dateSegs) {
    case 1:
        year = dateSegs[0]
    case 2:
        year = dateSegs[0]
        month = pad(dateSegs[1], 2)
    default:
        year = dateSegs[0]
        month = pad(dateSegs[1], 2)
        day = pad(dateSegs[2], 2)
    }

    hour := "00"
    min := "00"
    sec := "00"
    if timePart != "" {
        tseg := strings.Split(timePart, ":")
        if len(tseg) > 0 {
            hour = pad(tseg[0], 2)
        }
        if len(tseg) > 1 {
            min = pad(tseg[1], 2)
        }
        if len(tseg) > 2 {
            sec = pad(tseg[2], 2)
        }
    }

    combined := fmt.Sprintf("%s-%s-%sT%s:%s:%sZ", year, month, day, hour, min, sec)
    t, err := time.Parse(time.RFC3339, combined)
    if err != nil {
        return time.Time{}, err
    }
    return t.UTC(), nil
}

func pad(s string, length int) string {
    if len(s) >= length {
        return s
    }
    return strings.Repeat("0", length-len(s)) + s
}
