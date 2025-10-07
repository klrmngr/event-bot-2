# event-bot-copy-go

A small Discord event bot (Go) based on `discordgo`.

Quick start

1. Set required environment variables:

```bash
export DISCORD_TOKEN="your-bot-token-here"
export GUILD_ID="your-guild-id-here"
```

2. Run locally:

```bash
go run .
```

If environment variables are missing the program will exit and list which ones are missing.

Notes

- The project uses `github.com/bwmarrin/discordgo`.
- This repo was updated to fix a compile-time type mismatch for permission overwrites.
