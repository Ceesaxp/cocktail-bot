# Troubleshooting Guide

This document covers common issues you might encounter when running the Cocktail Bot and how to resolve them.

## Telegram Bot Issues

### Webhook Conflict Error

**Error message:**
```
Conflict: can't use getUpdates method while webhook is active; use deleteWebhook to delete the webhook first
Failed to get updates, retrying in 3 seconds...
```

**Cause:**
This error occurs when the Telegram bot has a webhook configured on the Telegram server side, but your bot is trying to use the polling method. Both methods cannot be used simultaneously.

**Solutions:**

1. **Use the delete-webhook script:**
   ```bash
   ./scripts/delete-webhook.sh
   ```
   
2. **Manually delete the webhook via the Telegram API:**
   ```bash
   curl -s "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/deleteWebhook"
   ```

3. **Check webhook status:**
   ```bash
   curl -s "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getWebhookInfo"
   ```

### Bot Not Responding to Messages

If your bot is running without errors but not responding to messages, check:

1. Ensure your bot token is correct in the config.yaml file
2. Make sure your bot's privacy mode is disabled in BotFather settings
3. Try sending the /start command to initialize the conversation

## Database Issues

### SQLite Permission Errors

**Error message:**
```
unable to open database file: ./data/users.sqlite
```

**Solutions:**

1. **Check directory permissions:**
   ```bash
   # Create data directory with correct permissions
   mkdir -p ./data
   chmod 777 ./data
   ```

2. **For Docker:**
   ```bash
   # Ensure the volume mount is correct
   docker run -v $(pwd)/data:/app/data ...
   ```

### Connection Issues with Remote Databases

If you're using PostgreSQL, MySQL, or MongoDB and experiencing connection issues:

1. Verify connection string format is correct
2. Ensure the database server allows connections from your bot's IP
3. Check that database credentials are correct

## Docker Issues

### Image Not Found

**Error message:**
```
Error response from daemon: manifest for ceesaxp/cocktail-bot:latest not found: manifest unknown
```

**Solutions:**

1. **Use the manual push script:**
   ```bash
   ./scripts/push-docker-latest.sh
   ```

2. **Use the direct deployment script:**
   ```bash
   DEPLOY_USER=your_username DROPLET_IP=your_server_ip ./scripts/direct-deploy.sh
   ```

### Logs Show CGO Not Enabled

If your logs show SQLite errors and you're using Docker, it might be because CGO was disabled during build.

**Solution:**
Rebuild the Docker image with CGO enabled:
```bash
docker build --build-arg CGO_ENABLED=1 -t cocktail-bot .
```

## API Server Issues

### API Not Accessible

If the API server is running but not accessible:

1. Check if the API is enabled in config.yaml (`api.enabled: true`)
2. Verify the port is not blocked by a firewall
3. Make sure you're using the correct port as specified in config.yaml

### Authentication Issues

If you're getting authentication errors with the API:

1. Check that your API token is correctly configured in api_tokens.yaml
2. Ensure you're sending the token in the Authorization header
3. Verify you haven't exceeded the rate limits