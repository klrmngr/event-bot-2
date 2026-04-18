-- Run as a superuser to create the role and database, then run the rest as discord_bot.

-- 1. Create role and database (run once as postgres superuser)
CREATE ROLE discord_bot WITH LOGIN PASSWORD 'your-password-here';
CREATE DATABASE discord_events OWNER discord_bot;

\connect discord_events

-- 2. Tables

CREATE TABLE IF NOT EXISTS users (
    discord_user_id TEXT PRIMARY KEY,
    username        TEXT,
    updated_at      TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS channels (
    discord_channel_id TEXT PRIMARY KEY,
    channel_name       TEXT,
    updated_at         TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS events (
    id                 BIGSERIAL PRIMARY KEY,
    discord_channel_id TEXT REFERENCES channels(discord_channel_id),
    discord_message_id TEXT,
    emoji              TEXT,
    date               TIMESTAMPTZ,
    title              TEXT,
    location           TEXT,
    price              TEXT,
    description        TEXT,
    author_id          TEXT,
    created_at         TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS event_responses (
    id            BIGSERIAL PRIMARY KEY,
    event_id      BIGINT REFERENCES events(id),
    user_id       TEXT,
    response_type TEXT CHECK (response_type IN ('yes', 'no', 'maybe')),
    updated_at    TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (event_id, user_id)
);

CREATE TABLE IF NOT EXISTS commands (
    id               BIGSERIAL PRIMARY KEY,
    discord_user_id  TEXT REFERENCES users(discord_user_id),
    command_text     TEXT,
    created_at       TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages (
    discord_message_id  TEXT PRIMARY KEY,
    discord_channel_id  TEXT,
    discord_user_id     TEXT,
    message             TEXT,
    created_at          TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS poker_sessions (
    id         BIGSERIAL PRIMARY KEY,
    user_id    TEXT NOT NULL,
    in_amount  NUMERIC(14,2) NOT NULL CHECK (in_amount >= 0),
    out_amount NUMERIC(14,2) NOT NULL CHECK (out_amount >= 0),
    location   TEXT,
    stakes_sb  NUMERIC(10,2),
    stakes_bb  NUMERIC(10,2),
    stakes_text TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- 3. Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO discord_bot;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO discord_bot;
