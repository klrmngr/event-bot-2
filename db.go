package main

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

// InitDB initializes the global db variable.
func InitDB() error {
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		return fmt.Errorf("DB_PASSWORD is not set")
	}
	// Build a URL-style connection string so passwords with spaces/special chars work
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword("discord_bot", password),
		Host:   "localhost",
		Path:   "discord_events",
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()
	connStr := u.String()

	d, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	// set some sensible defaults
	d.SetConnMaxIdleTime(5 * time.Minute)
	d.SetMaxOpenConns(10)
	if err := d.Ping(); err != nil {
		return err
	}
	db = d
	// Ensure poker_sessions table exists with separate small/blind and big/blind columns
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS poker_sessions (
        id BIGSERIAL PRIMARY KEY,
        user_id TEXT NOT NULL,
        in_amount NUMERIC(14,2) NOT NULL CHECK (in_amount >= 0),
        out_amount NUMERIC(14,2) NOT NULL CHECK (out_amount >= 0),
        location TEXT,
        stakes_sb NUMERIC(10,2),
        stakes_bb NUMERIC(10,2),
        stakes_text TEXT,
        created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
    )`)
	if err != nil {
		return err
	}

	return nil
}

// CreatePokerSession inserts a poker session row.
func CreatePokerSession(userID string, inAmt, outAmt float64, location, stakes string) error {
	if db == nil {
		return fmt.Errorf("db not initialized")
	}
	// ensure user exists in users table
	_ = upsertUser(userID, "")

	// parse stakes string like "1/2" or "0.5/1" into sb and bb
	var sbNull, bbNull sql.NullFloat64
	stakes = strings.TrimSpace(stakes)
	if stakes != "" {
		parts := strings.Split(stakes, "/")
		if len(parts) >= 1 {
			if v, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err == nil {
				sbNull.Float64 = v
				sbNull.Valid = true
			}
		}
		if len(parts) >= 2 {
			if v, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
				bbNull.Float64 = v
				bbNull.Valid = true
			}
		}
	}

	_, err := db.Exec(`INSERT INTO poker_sessions (user_id, in_amount, out_amount, location, stakes_sb, stakes_bb, stakes_text) VALUES ($1,$2,$3,$4,$5,$6,$7)`, userID, inAmt, outAmt, location, nullableFloatForExec(sbNull), nullableFloatForExec(bbNull), func() interface{} {
		if stakes == "" {
			return nil
		}
		return stakes
	}())
	return err
}

// helper: convert sql.NullFloat64 to either float64 or nil for Exec parameters
func nullableFloatForExec(n sql.NullFloat64) interface{} {
	if n.Valid {
		return n.Float64
	}
	return nil
}

// GetPokerLifetime returns the number of sessions and net profit (out - in) for a user.
func GetPokerLifetime(userID string) (count int, net float64, err error) {
	if db == nil {
		return 0, 0, fmt.Errorf("db not initialized")
	}
	row := db.QueryRow(`SELECT COALESCE(COUNT(*),0)::int, COALESCE(SUM(out_amount - in_amount),0)::float8 FROM poker_sessions WHERE user_id = $1`, userID)
	if err := row.Scan(&count, &net); err != nil {
		return 0, 0, err
	}
	return count, net, nil
}

// CreateEvent inserts a new event row. It returns the created id.
func CreateEvent(channelID, messageID, emoji, title, location, price, authorID string, date time.Time) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("db not initialized")
	}
	var id int64
	q := `INSERT INTO events (discord_channel_id, discord_message_id, emoji, date, title, location, price, author_id)
          VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`
	err := db.QueryRow(q, channelID, messageID, emoji, date, title, location, price, authorID).Scan(&id)
	return id, err
}

// Event represents an event row with fields useful for rendering the template.
type Event struct {
	ID          int64
	ChannelID   string
	MessageID   string
	Emoji       string
	Date        *time.Time
	Title       string
	Location    string
	Price       string
	Description string
	AuthorID    string
}

// GetEventByChannel fetches an event by channel_id.
func GetEventByChannel(channelID string) (*Event, error) {
	if db == nil {
		return nil, fmt.Errorf("db not initialized")
	}
	q := `SELECT id, discord_channel_id, discord_message_id, emoji, date, title, location, price, description, author_id FROM events WHERE discord_channel_id = $1 LIMIT 1`
	var e Event
	var nt sql.NullTime
	err := db.QueryRow(q, channelID).Scan(&e.ID, &e.ChannelID, &e.MessageID, &e.Emoji, &nt, &e.Title, &e.Location, &e.Price, &e.Description, &e.AuthorID)
	if err != nil {
		return nil, err
	}
	if nt.Valid {
		e.Date = &nt.Time
	}
	return &e, nil
}

// UpsertResponse inserts or updates a user's response for an event.
func UpsertResponse(eventID int64, userID, responseType string) error {
	if db == nil {
		return fmt.Errorf("db not initialized")
	}
	// normalize and validate responseType
	resp := strings.ToLower(strings.TrimSpace(responseType))
	allowed := map[string]bool{"yes": true, "maybe": true, "no": true}
	if !allowed[resp] {
		return fmt.Errorf("invalid response type: %s", responseType)
	}
	var existingID int64
	err := db.QueryRow("SELECT id FROM event_responses WHERE event_id = $1 AND user_id = $2", eventID, userID).Scan(&existingID)
	if err == sql.ErrNoRows {
		_, err := db.Exec("INSERT INTO event_responses (event_id, user_id, response_type) VALUES ($1,$2,$3)", eventID, userID, resp)
		return err
	}
	if err != nil {
		return err
	}
	_, err = db.Exec("UPDATE event_responses SET response_type = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2", resp, existingID)
	return err
}

// GetResponsesForEvent returns lists of user IDs for each response type.
func GetResponsesForEvent(eventID int64) (going, maybe, cant []string, err error) {
	if db == nil {
		return nil, nil, nil, fmt.Errorf("db not initialized")
	}
	rows, err := db.Query("SELECT user_id, response_type FROM event_responses WHERE event_id = $1", eventID)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID, resp string
		if err := rows.Scan(&userID, &resp); err != nil {
			return nil, nil, nil, err
		}
		switch resp {
		case "yes":
			going = append(going, userID)
		case "maybe":
			maybe = append(maybe, userID)
		case "no":
			cant = append(cant, userID)
		default:
			// ignore unknown
		}
	}
	return going, maybe, cant, nil
}

func UpdateEventFieldByChannel(channelID, field, value string) error {
	if db == nil {
		return fmt.Errorf("db not initialized")
	}
	// Only allow certain logical fields and map them to actual column names to avoid SQL injection.
	fieldMap := map[string]string{
		"title":       "title",
		"date":        "date",
		"location":    "location",
		"price":       "price",
		"emoji":       "emoji",
		"message_id":  "discord_message_id",
		"description": "description",
	}
	col, ok := fieldMap[field]
	if !ok {
		return fmt.Errorf("field %s not allowed", field)
	}
	q := fmt.Sprintf("UPDATE events SET %s = $1, updated_at = CURRENT_TIMESTAMP WHERE discord_channel_id = $2", col)
	_, err := db.Exec(q, value, channelID)
	return err
}

// InsertCommand logs a slash command or modal submission for auditing.
func InsertCommand(discordUserID, username, commandText string) error {
	if db == nil {
		return fmt.Errorf("db not initialized")
	}
	// ensure user record exists/updated
	if err := upsertUser(discordUserID, username); err != nil {
		return err
	}
	_, err := db.Exec("INSERT INTO commands (discord_user_id, command_text) VALUES ($1,$2)", discordUserID, commandText)
	return err
}

// InsertMessage logs a message sent in the server.
func upsertUser(discordUserID, username string) error {
	if db == nil {
		return fmt.Errorf("db not initialized")
	}
	_, err := db.Exec(`INSERT INTO users (discord_user_id, username) VALUES ($1,$2)
        ON CONFLICT (discord_user_id) DO UPDATE SET username = EXCLUDED.username, updated_at = CURRENT_TIMESTAMP`, discordUserID, username)
	return err
}

func upsertChannel(discordChannelID, channelName string) error {
	if db == nil {
		return fmt.Errorf("db not initialized")
	}
	// Only overwrite channel_name when a non-empty name is provided.
	_, err := db.Exec(`INSERT INTO channels (discord_channel_id, channel_name) VALUES ($1,$2)
        ON CONFLICT (discord_channel_id) DO UPDATE SET channel_name = COALESCE(NULLIF(EXCLUDED.channel_name, ''), channels.channel_name), updated_at = CURRENT_TIMESTAMP`, discordChannelID, channelName)
	return err
}

// InsertMessage logs a message sent in the server. channelName can be empty if unknown.
func InsertMessage(discordMessageID, discordChannelID, channelName, discordUserID, username, message string) error {
	if db == nil {
		return fmt.Errorf("db not initialized")
	}
	// ensure user exists
	if err := upsertUser(discordUserID, username); err != nil {
		return err
	}
	// best-effort ensure channel exists (channel name may be empty)
	if discordChannelID != "" {
		_ = upsertChannel(discordChannelID, channelName)
	}
	_, err := db.Exec("INSERT INTO messages (discord_message_id, discord_channel_id, discord_user_id, message) VALUES ($1,$2,$3,$4) ON CONFLICT (discord_message_id) DO NOTHING", discordMessageID, discordChannelID, discordUserID, message)
	return err
}
