# Telegram Cocktail Bot Design Document

## Project Overview
The Telegram Cocktail Bot validates user emails against a database and provides a free cocktail redemption system. Users can check their eligibility by submitting their email, and if eligible, they can choose to redeem or skip their free cocktail.

## Key Features
1. Email validation against a database with format checking
2. Rate limiting to prevent abuse
3. Redemption tracking via timestamp
4. Interactive Telegram buttons for user decision
5. Configurable database backend
6. Containerized deployment with Alpine Linux
7. Comprehensive event logging
8. REST API with authentication for programmatic access
9. Report generation with CSV export capabilities

## Domain Model

### Core Entities
- **User**: Represented by email address with redemption status
- **Bot**: Telegram interface for user interaction
- **Database**: Configurable storage for user records
- **Logger**: Records lookup and redemption events

### Use Cases
1. **Initial Interaction Flow**
   - User sends /start command
   - Bot responds with welcome message and usage instructions

2. **Email Verification Flow**
   - User sends email to bot
   - Bot validates email format
   - If valid, bot checks email (case-insensitive) against database
   - Bot responds with appropriate message and buttons if applicable
   - If invalid format, bot responds with polite correction message

3. **Redemption Flow**
   - User clicks "Get Cocktail" button
   - Bot updates database with current timestamp
   - Bot confirms redemption

4. **Skip Flow**
   - User clicks "Skip" button
   - No database update
   - Bot confirms choice was dismissed

5. **Rate Limiting Flow**
   - Bot tracks request frequency by user
   - Enforces configurable rate limits
   - Returns friendly message when limit exceeded

6. **API Email Submission Flow**
   - External system sends email via API with authentication
   - System validates email and adds to database
   - System returns success/error response

7. **Report Generation Flow**
   - External system requests report via API with authentication
   - System filters data based on report type and date range
   - System returns data in JSON or CSV format according to request

## Technical Architecture

### Components
1. **Telegram Interface**
   - Handles message reception and button interactions
   - Formats responses
   - Processes /start command

2. **REST API**
   - Token-based authentication
   - Email submission endpoint
   - Report generation endpoints with format options
   - Rate limiting per client

3. **Email Validator**
   - Checks email format before database lookup
   - Converts email to lowercase for comparison

4. **Rate Limiter**
   - Tracks request frequency by user ID or client IP
   - Enforces configurable limits for bot and API

5. **Database Adapter**
   - Abstract interface for database operations
   - Concrete implementations for each supported database type
   - Error handling for database unavailability
   - Report generation capabilities

6. **Business Logic**
   - Email validation
   - Redemption status checking
   - Update operations
   - Report filtering and generation

7. **Configuration Manager**
   - Loading database configuration from file and/or environment variables
   - Telegram token management
   - API token management
   - Rate limit settings

8. **Logger**
   - Records email lookups and redemptions
   - Timestamps and user identifiers
   - API request logging

### Database Schema
```
User {
  ID: unique identifier
  Email: string (stored as lowercase)
  DateAdded: datetime
  Redeemed: datetime (nullable)
}
```

### Configuration Options
```
Bot {
  TelegramToken: string
  TelegramUser: string
}

Database {
  Type: enum (CSV, SQLite, GoogleSheet, PostgreSQL, MySQL, MongoDB)
  ConnectionString: string
  // Type-specific configuration fields
}

RateLimiting {
  RequestsPerMinute: int
  RequestsPerHour: int
}

API {
  Enabled: bool
  Port: int
  TokensFile: string
  RateLimitPerMin: int
  RateLimitPerHour: int
}
```

## Implementation Approach

### Programming Language
- Go (Golang)

### Key Libraries
- Telegram Bot API library (github.com/go-telegram-bot-api/telegram-bot-api)
- Email validation (net/mail or github.com/badoux/checkmail)
- Rate limiting implementation
- Database drivers for supported backends
- Configuration management (viper or similar)
- Structured logging (zap or zerolog)
- HTTP server for REST API (net/http or Gorilla Mux)
- Authentication middleware for API security

### Docker Deployment
- Alpine-based image for minimal size
- Configuration via environment variables and/or config file
- Volume mounting for database files (if applicable)
- Proper signal handling for graceful shutdown

### Error Handling
- Friendly message when database is unavailable
- Retry mechanism for transient database errors
- Logging of all errors for troubleshooting

## Usage Examples

### User Interaction via Telegram Bot
1. User sends: `/start`
   - Bot: "Welcome to the Cocktail Bot! Send your email to check if you're eligible for a free cocktail."

2. User sends: `invalid-email`
   - Bot: "That doesn't look like a valid email address. Please send a properly formatted email (e.g., example@domain.com)."

3. User sends: `user@example.com` (not in database)
   - Bot: "Email is not in database."

4. User sends: `user@example.com` (in database, not redeemed)
   - Bot: "Email found! You're eligible for a free cocktail."
   - [Get Cocktail] [Skip]

5. User clicks: `[Get Cocktail]`
   - Bot: "Enjoy your free cocktail! Redeemed on May 14, 2025."

6. User sends: `user@example.com` (already redeemed)
   - Bot: "Email found, but free cocktail already consumed on May 14, 2025."

7. User sending too many requests:
   - Bot: "You've made too many requests. Please try again in a few minutes."

8. Database unavailable:
   - Bot: "Sorry, our system is temporarily unavailable. Please try again later."
   
### REST API Interaction

1. Adding a new email:
```bash
curl -X POST http://your-server:8080/api/v1/email \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-token" \
  -d '{"email": "user@example.com"}'
```

2. Getting a report of redeemed cocktails:
```bash
curl "http://your-server:8080/api/v1/report/redeemed?from=2023-01-01&to=2023-12-31" \
  -H "Authorization: Bearer your-api-token"
```

3. Getting a report in CSV format:
```bash
curl "http://your-server:8080/api/v1/report/all?format=csv" \
  -H "Authorization: Bearer your-api-token" \
  -o users_report.csv
```

4. Managing API tokens:
```bash
# Add a new token
./scripts/manage-api-tokens.sh add new-token-value

# List all available tokens
./scripts/manage-api-tokens.sh list
```

## Implementation Plan
1. Set up project structure following domain-driven design
2. Implement core domain models and interfaces
3. Create database adapters for each supported type
4. Implement Telegram bot interface and command handling
5. Add email validation and rate limiting
6. Implement logging system
7. Create configuration system
8. Implement REST API with authentication
9. Add report generation functionality with multiple formats
10. Build Docker deployment
11. Write tests for all components
12. Document API and deployment instructions