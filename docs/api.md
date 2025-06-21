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

### Bulk Upload Emails

```
POST /api/v1/email/bulk
```

Adds multiple email addresses to the database in a single request. This endpoint supports three content types:

1. **JSON payload** with Content-Type: `application/json`
2. **CSV data** with Content-Type: `text/csv` or `application/csv`
3. **File upload** with Content-Type: `multipart/form-data` (supports both CSV and JSON files)

#### JSON Request Example:

```json
{
  "emails": [
    "user1@example.com",
    "user2@example.com",
    "user3@example.com"
  ]
}
```

#### CSV Request Example:

Simple CSV format with one email per line:
```
email
user1@example.com
user2@example.com
user3@example.com
```
The first line is treated as a header if it contains the word "email" and will be skipped.

#### File Upload Example:

Upload a CSV or JSON file using a multipart form with the field name "file".

**Successful Response (200 OK):**

```json
{
  "total": 3,
  "success": 2,
  "failed": 0,
  "duplicate": 1,
  "failures": []
}
```

If any emails fail validation or insertion, details are provided in the `failures` array:

```json
{
  "total": 3,
  "success": 1,
  "failed": 1,
  "duplicate": 1,
  "failures": [
    "invalid@email.: invalid format"
  ]
}
```

**Error Responses:**

1. No valid emails found (400 Bad Request):
```json
{
  "error": "Invalid request",
  "code": 400,
  "details": "No valid emails found in payload"
}
```

2. Request too large (413 Request Entity Too Large):
```json
{
  "error": "Request too large",
  "code": 413,
  "details": "Maximum 1000 emails allowed per request"
}
```

3. Invalid content type (415 Unsupported Media Type):
```json
{
  "error": "Invalid Content-Type",
  "code": 415,
  "details": "Content-Type must be application/json, text/csv, or multipart/form-data"
}
```

The endpoint also returns the same authentication and rate limit error responses as the single email endpoint.

### Generate Reports

The following endpoints allow you to generate reports about users in various formats.

#### Common Parameters for All Report Endpoints

- **from** (optional): Start date for the report in YYYY-MM-DD format. Defaults to 7 days ago.
- **to** (optional): End date for the report in YYYY-MM-DD format. Defaults to current date.
- **format** (optional): Response format, either "json" (default) or "csv".

#### Redeemed Users Report

```
GET /api/v1/report/redeemed
```

Returns users who have redeemed their cocktails within the specified date range.

#### Added Users Report

```
GET /api/v1/report/added
```

Returns users who were added to the system within the specified date range.

#### All Users Report

```
GET /api/v1/report/all
```

Returns all users within the specified date range.

#### JSON Response Example

**Successful Response (200 OK):**

```json
{
  "type": "redeemed",
  "from": "2023-01-01T00:00:00Z",
  "to": "2023-12-31T23:59:59Z",
  "count": 2,
  "users": [
    {
      "ID": "user_123",
      "Email": "user1@example.com",
      "DateAdded": "2023-01-15T10:30:00Z",
      "Redeemed": "2023-01-16T14:20:00Z"
    },
    {
      "ID": "user_456",
      "Email": "user2@example.com",
      "DateAdded": "2023-02-20T08:45:00Z",
      "Redeemed": "2023-02-21T17:10:00Z"
    }
  ],
  "generated": "2023-05-10T15:30:00Z"
}
```

#### CSV Response Example

When using `format=csv`, the response will be a downloadable CSV file with the following format:

```
ID,Email,DateAdded,Redeemed
user_123,user1@example.com,2023-01-15T10:30:00Z,2023-01-16T14:20:00Z
user_456,user2@example.com,2023-02-20T08:45:00Z,2023-02-21T17:10:00Z
```

The Content-Disposition header will be set to `attachment; filename="redeemed-report-2023-05-10.csv"`.

#### Error Responses

1. Authentication error (401 Unauthorized):
```json
{
  "error": "Unauthorized",
  "code": 401,
  "details": "Invalid or missing authentication token"
}
```

2. Invalid date format (400 Bad Request):
```json
{
  "error": "Invalid date format",
  "code": 400,
  "details": "invalid 'from' date format. Use YYYY-MM-DD"
}
```

3. Invalid date range (400 Bad Request):
```json
{
  "error": "Invalid date format",
  "code": 400,
  "details": "'from' date cannot be after 'to' date"
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
  "details": "Error generating report"
}
```

## Configuration

The API is configured in the `config.yaml` file under the `api` section:

```yaml
api:
  # Enable or disable the REST API
  enabled: true
  # Host to bind to (default is 0.0.0.0)
  host: "0.0.0.0"
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
