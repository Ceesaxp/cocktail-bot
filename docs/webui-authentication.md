# WebUI Authentication

The Cocktail Bot WebUI uses token-based authentication, sharing the same authentication tokens with the REST API. The WebUI acts as a client to the API, using the API endpoints to fetch all data.

## Configuration

### 1. Enable both API and WebUI in `config.yaml`

```yaml
# API must be enabled for WebUI to work
api:
  enabled: true
  host: "0.0.0.0"
  port: 8080
  auth_tokens:
    - "your_secure_token_1"
    - "your_secure_token_2"

# WebUI configuration
webui:
  enabled: true
  host: "0.0.0.0"
  port: 8081
```

### 2. Configure Authentication Tokens

Authentication tokens are shared between the API and WebUI. Configure them in the API section:

```yaml
api:
  # Option 1: Direct token configuration
  auth_tokens:
    - "your_secure_token_1"
    - "your_secure_token_2"
  
  # Option 2: Load tokens from file
  tokens_file: "./api_tokens.yaml"
```

If using a tokens file (`api_tokens.yaml`):

```yaml
tokens:
  - "your_secure_token_1"
  - "your_secure_token_2"
```

## Usage

### Accessing the WebUI

1. Navigate to `http://your-server:8081` (or configured port)
2. You'll be redirected to the login page
3. Enter one of your configured authentication tokens
4. Click "Login" to access the dashboard

### Authentication Flow

- The WebUI uses HTTP cookies to maintain sessions
- Authentication tokens are validated on each request
- Sessions expire after 24 hours or when you logout
- Invalid tokens are rejected with an error message

### Security Considerations

1. **Use Strong Tokens**: Generate secure, random tokens (e.g., using the token generator tool)
2. **HTTPS**: Always use HTTPS in production (configure via reverse proxy like Caddy)
3. **Token Storage**: Keep your tokens secure and never commit them to version control
4. **Shared Authentication**: The same tokens work for both API and WebUI access

### Reverse Proxy Configuration

Since HTTPS is managed by your reverse proxy, here's an example Caddy configuration:

```caddy
webui.yourdomain.com {
    reverse_proxy localhost:8081
}
```

### Architecture

The WebUI operates as a client to the REST API:

1. **Authentication**: WebUI validates user tokens locally
2. **Data Fetching**: All data is fetched from API endpoints
3. **No Direct Database Access**: WebUI doesn't access the database directly
4. **API Dependency**: WebUI requires the API to be running

This architecture provides:
- Single source of truth for data access
- Consistent business logic through the API
- Simplified codebase with no duplicate logic
- Better separation of concerns

### API Endpoints Used by WebUI

- `/api/v1/report/all` - Get all users
- `/api/v1/report/redeemed` - Get redeemed users
- `/api/v1/report/added` - Get recently added users

### Differences from Previous Implementation

The WebUI no longer uses username/password authentication. Instead:

- Uses the same token-based authentication as the API
- No need to manage separate WebUI credentials
- Simpler configuration and maintenance
- Better security through token rotation
- No direct database access - all data comes from API

### Troubleshooting

1. **"Invalid authentication token" error**
   - Verify the token is correctly configured in `config.yaml`
   - Check that the API tokens are loaded (check logs)
   - Ensure no typos in the token

2. **Redirect loop**
   - Clear browser cookies for the site
   - Verify WebUI is properly started

3. **Can't access after login**
   - Check browser console for errors
   - Verify the auth_token cookie is set
   - Check server logs for authentication errors

4. **"Error loading data" on dashboard/users pages**
   - Ensure the API is running and accessible
   - Check that both API and WebUI are enabled in config
   - Verify API is responding on the configured port
   - Check logs for API connection errors

5. **WebUI fails to start**
   - If you see "WebUI requires API to be enabled", enable the API in config
   - Ensure different ports are used for API and WebUI