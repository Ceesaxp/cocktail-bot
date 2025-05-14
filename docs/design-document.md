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

## Technical Architecture

### Components
1. **Telegram Interface**
   - Handles message reception and button interactions
   - Formats responses
   - Processes /start command

2. **Email Validator**
   - Checks email format before database lookup
   - Converts email to lowercase for comparison

3. **Rate Limiter**
   - Tracks request frequency by user ID
   - Enforces configurable limits

4. **Database Adapter**
   - Abstract interface for database operations
   - Concrete implementations for each supported database type
   - Error handling for database unavailability

5. **Business Logic**
   - Email validation
   - Redemption status checking
   - Update operations

6. **Configuration Manager**
   - Loading database configuration from file and/or environment variables
   - Telegram token management
   - Rate limit settings

7. **Logger**
   - Records email lookups and redemptions
   - Timestamps and user identifiers

### Database Schema
```
User {
  ID: unique identifier
  Email: string (stored as lowercase)
  DateAdded: datetime
  AlreadyConsumed: datetime (nullable)
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

### User Interaction
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

## Implementation Plan
1. Set up project structure following domain-driven design
2. Implement core domain models and interfaces
3. Create database adapters for each supported type
4. Implement Telegram bot interface and command handling
5. Add email validation and rate limiting
6. Implement logging system
7. Create configuration system
8. Build Docker deployment
9. Write tests for all components
10. Document API and deployment instructions