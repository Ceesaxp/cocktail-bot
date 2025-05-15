# Cocktail Bot RESTful API Documentation

## Overview

The Cocktail Bot REST API allows you to programmatically add email addresses to the database for cocktail eligibility. This API follows RESTful principles and uses JSON for request and response formats.

## Authentication

All API requests require authentication using a bearer token.

```
Authorization: Bearer YOUR_API_TOKEN
```

API tokens can be managed using the token management script:

```bash
# List all tokens
./scripts/manage-api-tokens.sh list

# Add a new token (generates a random token if none provided)
./scripts/manage-api-tokens.sh add

# Add a specific token
./scripts/manage-api-tokens.sh add your-token-here

# Remove a token
./scripts/manage-api-tokens.sh remove your-token-here

# Reset tokens file with a single new random token
./scripts/manage-api-tokens.sh reset

# Initialize tokens file if it doesn't exist
./scripts/manage-api-tokens.sh init
```

Alternatively, you can use the token generator utility:

```bash
# Generate a single token
go run cmd/token-generator/main.go

# Generate multiple tokens
go run cmd/token-generator/main.go -count 3

# Display tokens without saving to file
go run cmd/token-generator/main.go -display-only

# For more options
go run cmd/token-generator/main.go -help
```

## Rate Limiting

The API implements rate limiting to prevent abuse. By default, clients are limited to:
- 30 requests per minute
- 300 requests per hour

When rate limits are exceeded, the API returns a `429 Too Many Requests` status code.

Rate limit information is included in the response headers:
- `X-RateLimit-Limit-Minute`: Maximum requests per minute
- `X-RateLimit-Remaining-Minute`: Remaining requests for the current minute

## Base URL

The base URL for all API endpoints is:

```
http://your-server:8080/api/
```

The port can be configured in `config.yaml` under the `api.port` setting.

## Endpoints

### Check API Health

```
GET /api/health
```

Returns a simple health check to verify the API is operational.

**Response:**

```json
{
  "status": "ok",
  "version": "1.0.0"
}
```

### Submit Email

```
POST /api/v1/email
```

Adds a new email address to the database.

**Request Body:**

```json
{
  "email": "user@example.com"
}
```

**Successful Response (201 Created):**

```json
{
  "id": "api_1653245875123456",
  "status": "created",
  "message": "Email added successfully"
}
```

**Error Responses:**

1. Email already exists (409 Conflict):
```json
{
  "id": "existing_id_123",
  "status": "exists",
  "message": "Email already exists in database"
}
```

2. Invalid email format (400 Bad Request):
```json
{
  "error": "Invalid email",
  "code": 400,
  "details": "The provided email address is not valid"
}
```

3. Authentication error (401 Unauthorized):
```json
{
  "error": "Unauthorized",
  "code": 401,
  "details": "Invalid or missing authentication token"
}
```

4. Rate limit exceeded (429 Too Many Requests):
```json
{
  "error": "Too Many Requests",
  "code": 429,
  "details": "Rate limit exceeded"
}
```

5. Server error (500 Internal Server Error):
```json
{
  "error": "Internal server error",
  "code": 500,
  "details": "Error processing request"
}
```

## Configuration

The API is configured in the `config.yaml` file under the `api` section:

```yaml
api:
  # Enable or disable the REST API
  enabled: true
  # Port to listen on
  port: 8080
  # File containing authentication tokens
  tokens_file: "./api_tokens.yaml"
  # API-specific rate limiting
  rate_limit_per_min: 30
  rate_limit_per_hour: 300
```

## Authentication Methods

There are two ways to configure API tokens:

### 1. Environment Variable

You can set API tokens directly through an environment variable:

```bash
# Set a single token
export COCKTAILBOT_API_TOKENS="your-token-here"

# Set multiple tokens (comma-separated)
export COCKTAILBOT_API_TOKENS="token1,token2,token3"
```

In Docker, you can pass tokens via the `-e` flag:

```bash
docker run -e COCKTAILBOT_API_TOKENS="token1,token2,token3" ...
```

### 2. API Tokens File

Alternatively, you can use a YAML file (`api_tokens.yaml`):

```yaml
auth_tokens:
  - "token1_abc123xyz"
  - "token2_def456uvw"
  - "token3_ghi789rst"
```

**Note:** Environment variables take precedence over the tokens file.

## Security Recommendations

1. Use HTTPS in production environments with a valid SSL certificate
2. Regularly rotate API tokens
3. Set restrictive file permissions on the tokens file
4. Configure a reverse proxy (like Nginx) for additional security in production
5. Consider using IP allowlisting for sensitive deployments