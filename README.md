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
- RESTful API for programmatic email submission
- Secure API token authentication

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

Build the Docker image with SQLite support:

```bash
# For AMD64 (Linux servers)
docker build -t cocktail-bot .

# For Apple Silicon (M1/M2 Mac)
docker build --platform linux/arm64 -t cocktail-bot:local .
```

Run the container:

```bash
# Make sure the data directory exists and has correct permissions
mkdir -p ./data
chmod 777 ./data

# Run the container
docker run \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/data:/app/data \
  -p 8080:8080 \
  -e COCKTAILBOT_API_TOKENS="token1,token2,token3" \
  cocktail-bot
```

> **Note**: The Docker image includes SQLite support enabled through CGO.

## Database Options

The Cocktail Bot supports multiple database backends, allowing you to choose the most appropriate storage solution for your deployment:

### CSV

A simple file-based storage option, ideal for testing and small deployments.

```yaml
database:
  type: "csv"
  connection_string: "./data/users.csv"
```

### SQLite

A lightweight, file-based SQL database requiring no separate server.

```yaml
database:
  type: "sqlite"
  connection_string: "./data/users.db"
```

### PostgreSQL

A powerful, open-source relational database for larger deployments.

```yaml
database:
  type: "postgresql"
  connection_string: "postgres://username:password@localhost:5432/dbname"
```

### MySQL

A popular open-source relational database.

```yaml
database:
  type: "mysql"
  connection_string: "username:password@tcp(localhost:3306)/dbname"
```

### MongoDB

A NoSQL document database for flexible schema requirements.

```yaml
database:
  type: "mongodb"
  connection_string: "mongodb://username:password@localhost:27017/dbname"
```

### Google Sheets

Store data in a Google Sheet, ideal for simple collaboration and visibility.

```yaml
database:
  type: "googlesheet"
  connection_string: "credentials.json|sheet_id|Sheet1"
```

For detailed instructions on setting up Google Sheets integration, see [Google Sheets Guide](docs/googlesheets.md)

## Documentation

- [API Documentation](docs/api.md) - RESTful API for programmatic email submission
- [Google Sheets Integration](docs/googlesheets.md) - Guide for using Google Sheets as a database
- [Design Document](docs/design-document.md) - Overall system design and architecture
- [Deployment Guide](docs/deployment.md) - Instructions for deploying to production

## CI/CD Pipeline

The project uses GitHub Actions for continuous integration and deployment:

1. **CI**: On every push and pull request, the pipeline runs tests and linting.
2. **Build**: After successful tests, a Docker image is built and pushed to DockerHub.
3. **CD**: Manual deployment to a Digital Ocean droplet can be triggered from the GitHub Actions interface.

### Deployment Options

There are several ways to deploy the application:

1. **GitHub Actions Workflow**: Use the CI/CD pipeline for automated deployment
   ```bash
   # Triggered from the GitHub Actions interface
   ```

2. **Manual Docker Hub Push**: Build and push the image to Docker Hub
   ```bash
   ./scripts/push-docker-latest.sh
   ```

3. **Direct Deployment**: Build locally and deploy directly to server
   ```bash
   DEPLOY_USER=your_username DROPLET_IP=your_server_ip ./scripts/direct-deploy.sh
   ```

### API Management

The Cocktail Bot includes a RESTful API for programmatic email submission. You can manage API tokens with these scripts:

1. **Manage API Tokens**: Add, remove, or list API tokens
   ```bash
   ./scripts/manage-api-tokens.sh add     # Add a new random token
   ./scripts/manage-api-tokens.sh list    # List all tokens
   ./scripts/manage-api-tokens.sh remove <token>  # Remove a token
   ```

2. **Test API**: Verify the API is working correctly
   ```bash
   ./scripts/test-api.sh [token] [email]  # Test with optional token and email
   ```

See the [Deployment Guide](docs/deployment.md) for detailed instructions.

## License

[MIT](LICENSE)
