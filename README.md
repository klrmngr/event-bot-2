# event-bot-2

discord bot for odds & ends

## Quick start

1. Create a `.env` file in the project root:

```bash
DISCORD_TOKEN="token"
CLIENT_SECRET="super-secret-secret"
GUILD_ID=12345
DB_PASSWORD="super-strong-password"
```

2. Start the bot and database:

```bash
docker compose up --build
```

## Secrets

- **`DISCORD_TOKEN`** — Discord Developer Portal → your application → Bot → reset token
- **`CLIENT_SECRET`** — Discord Developer Portal → OAuth2 → reset secret
- **`GUILD_ID`** — right-click your server in Discord (requires Developer Mode) → Copy Server ID
- **`DB_PASSWORD`** — any strong password; only used internally between the bot and Postgres
