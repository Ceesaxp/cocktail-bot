# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cocktail Bot is a Telegram bot that validates user emails against a database to determine if they're eligible for a free cocktail. Users can check their eligibility by submitting their email, and if eligible, they can choose to redeem or skip their free cocktail.

## Architecture

The project follows a domain-driven design approach with clear separation of concerns:

- **Domain Layer**: Core models, repository interfaces, and domain errors
- **Repository Layer**: Multiple database implementations with a factory pattern
- **Service Layer**: Business logic for email validation and cocktail redemption
- **Telegram Layer**: Bot interface and message handlers
- **Config/Logger**: Cross-cutting concerns

## Key Components

1. **Repository Pattern**: Supports multiple database backends (CSV, Google Sheets, PostgreSQL, MySQL, MongoDB)
2. **Factory Pattern**: Creates appropriate repository implementation based on configuration
3. **Rate Limiting**: Prevents abuse by limiting requests per user
4. **Error Handling**: Domain-specific errors with friendly user messages

## Common Commands

### Building and Running

```bash
# Build the binary
make build

# Run with configuration
make run

# Clean build artifacts
make clean
```

### Testing

```bash
# Run all tests
make test

# Run tests for a specific package
go test -v ./internal/repository

# Run a specific test
go test -v ./internal/repository -run TestFindByEmail
```

### Linting

```bash
# Run linter
make lint
```

### Docker

```bash
# Deployment with Docker Compose (MySQL + Caddy)
# Copy environment variables template and update values
cp .env.example .env

# Build images and start services
docker-compose up -d
```

### Project Setup

```bash
# Initialize project (creates data directory and config.yaml)
make init

# Generate sample CSV data
make sample-csv
```

## Configuration

Configuration is handled via `config.yaml` file or environment variables with `COCKTAILBOT_` prefix (overridden by Docker Compose `.env` settings):

```yaml
log_level: info
telegram:
  token: "YOUR_TELEGRAM_BOT_TOKEN"
  user: "your_bot_username"
database:
  type: "mysql"  # mysql, postgresql, sqlite, csv, googlesheet, mongodb
  connection_string: "cocktailbot:your_mysql_password@tcp(mysql:3306)/cocktailbot?parseTime=true"
rate_limiting:
  requests_per_minute: 10
  requests_per_hour: 100
```

## Database Schema

The core entity is `User`:
- ID: string
- Email: string
- DateAdded: time.Time
- Redeemed: *time.Time (nullable, indicates redemption)