# Cocktail Bot

A Telegram bot for cocktail redemption via email validation.

## Overview

This bot allows users to check if their email address is eligible for a free cocktail. If eligible, they can choose to redeem or skip the offer. The bot tracks redemptions to prevent multiple claims.

## Features

- Email validation against a database
- Rate limiting to prevent abuse
- Redemption tracking via timestamp
- Interactive Telegram buttons for user decision
- Configurable database backend
- Comprehensive event logging

## Requirements

- Go 1.21 or later
- Telegram Bot Token (from BotFather)
- Database backend (CSV, SQLite, Google Sheets, PostgreSQL, MySQL, or MongoDB)

## Configuration

Configuration can be provided via `config.yaml` file or environment variables:

```yaml
# Log level (debug, info, warn, error)
log_level: info

# Telegram settings
telegram:
  # Bot token (get from BotFather)
  token: "YOUR_TELEGRAM_BOT_TOKEN"
  # Bot username
  user: "your_bot_username"

# Database settings
database:
  # Database type (csv, sqlite, googlesheet, postgresql, mysql, mongodb)
  type: "csv"
  # Connection string or path
  connection_string: "./data/users.csv"

# Rate limiting settings
rate_limiting:
  # Maximum requests per minute per user
  requests_per_minute: 10
  # Maximum requests per hour per user
  requests_per_hour: 100
```

Environment variables can be used with the `COCKTAILBOT_` prefix, e.g., `COCKTAILBOT_LOG_LEVEL=debug`.

## Building

```bash
go build -o cocktail-bot ./cmd/bot
```

## Running

```bash
./cocktail-bot --config config.yaml
```

## Docker

Build the Docker image:

```bash
docker build -t cocktail-bot .
```

Run the container:

```bash
docker run -v $(pwd)/config.yaml:/app/config.yaml -v $(pwd)/data:/app/data cocktail-bot
```

## Database Options

### CSV

```yaml
database:
  type: "csv"
  connection_string: "./data/users.csv"
```

### SQLite

```yaml
database:
  type: "sqlite"
  connection_string: "./data/users.db"
```

### PostgreSQL

```yaml
database:
  type: "postgresql"
  connection_string: "postgres://username:password@localhost:5432/dbname"
```

### MySQL

```yaml
database:
  type: "mysql"
  connection_string: "username:password@tcp(localhost:3306)/dbname"
```

### MongoDB

```yaml
database:
  type: "mongodb"
  connection_string: "mongodb://username:password@localhost:27017/dbname"
```

### Google Sheets

```yaml
database:
  type: "googlesheet"
  connection_string: "credentials.json:sheet_id:Sheet1!A1:D100"
```

## License

MIT
