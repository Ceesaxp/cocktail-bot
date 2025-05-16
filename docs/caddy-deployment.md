# Caddy Reverse Proxy Deployment Guide

This guide explains how to deploy the Cocktail Bot application behind a Caddy HTTP server acting as a reverse proxy.

## Why Use Caddy?

[Caddy](https://caddyserver.com/) is a powerful, enterprise-ready, open source web server with automatic HTTPS. Using Caddy as a reverse proxy provides several benefits:

1. **Automatic HTTPS** - Caddy automatically obtains and renews SSL certificates
2. **Security** - Adds a layer of security by not exposing your application directly
3. **Load balancing** - Can distribute traffic if you scale to multiple instances
4. **Path-based routing** - Can route different paths to different services
5. **TLS termination** - Handles HTTPS so your application doesn't have to

## Prerequisites

1. A server with:
   - Linux (Ubuntu/Debian recommended)
   - Go installed (for running the Cocktail Bot)
   - Caddy installed (see [Caddy installation guide](https://caddyserver.com/docs/install))
   - Port 80 and 443 available for Caddy

2. A domain name pointing to your server (for proper TLS)

## Configuration Files

Two main configuration files are needed:

1. **Caddyfile** - Configures the Caddy server
2. **config.yaml** - Configures the Cocktail Bot to listen only on localhost

## Setup Steps

### 1. Configure Cocktail Bot for Localhost Only

Create or modify your `config.yaml` to use the provided `config.yaml.localhost` file:

```bash
# Copy the localhost config
cp config.yaml.localhost config.yaml

# Edit the config and update tokens, database settings, etc.
nano config.yaml
```

Important settings in the config:

```yaml
api:
  enabled: true
  host: "127.0.0.1"  # Listen only on localhost
  port: 8080
  # ...other API settings...
```

### 2. Configure Caddy

Create or modify your `Caddyfile` with the provided configuration:

```bash
# Copy the Caddyfile to the Caddy config directory
sudo cp Caddyfile /etc/caddy/Caddyfile

# Edit domain name and other settings if needed
sudo nano /etc/caddy/Caddyfile
```

Make sure to update `cocktailbot.example.com` with your actual domain name.

### 3. Create Log Directory for Caddy

```bash
sudo mkdir -p /var/log/caddy
sudo chown caddy:caddy /var/log/caddy
```

### 4. Restart Caddy

```bash
sudo systemctl restart caddy
```

### 5. Start the Cocktail Bot

Run the Cocktail Bot with the new configuration:

```bash
make run
# or
./cocktail-bot -config ./config.yaml
```

## Testing the Setup

1. Verify Caddy is running:
   ```bash
   sudo systemctl status caddy
   ```

2. Check Caddy logs:
   ```bash
   sudo tail -f /var/log/caddy/cocktailbot.log
   ```

3. Test the API endpoint through the proxy:
   ```bash
   curl -k https://cocktailbot.example.com/api/health
   ```

4. If using a real domain with valid certificates:
   ```bash
   curl https://cocktailbot.example.com/api/health
   ```

## Troubleshooting

1. **Certificate issues**:
   - Ensure your domain is correctly pointing to your server
   - Check Caddy logs for certificate errors

2. **Connection refused**:
   - Verify the Cocktail Bot is running and listening on localhost:8080
   - Check the API is enabled in the config

3. **Permission issues**:
   - Ensure Caddy has permission to read the Caddyfile and write to log directories

4. **Caddy not starting**:
   - Validate your Caddyfile syntax: `caddy validate --config /etc/caddy/Caddyfile`

## Security Considerations

1. The Cocktail Bot only listens on localhost, making it inaccessible directly from the internet
2. Caddy handles TLS termination and provides additional security headers
3. API tokens are still required for authentication to use the API endpoints
4. Rate limiting is applied at both Caddy and application levels

## Performance Tuning

If needed, you can adjust the following:

1. In Caddy, set resource limits for higher traffic:
   ```
   {
       servers {
           protocol {
               http {
                   max_header_size 16kb
                   idle_timeout 60s
               }
           }
       }
   }
   ```

2. For the Cocktail Bot, consider using a more production-ready database like PostgreSQL or MySQL instead of SQLite

## Conclusion

Your Cocktail Bot should now be running securely behind Caddy with automatic HTTPS. The application only responds to requests from Caddy (on localhost), while Caddy handles all public-facing traffic, providing an additional layer of security and features.