#!/bin/bash
set -e

# Deployment script for Cocktail Bot on Digital Ocean

# Log function for better visibility
log() {
  echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

# Variables
APP_DIR="/srv/cocktail-bot"
CONFIG_FILE="$APP_DIR/config.yaml"
DATA_DIR="$APP_DIR/data"
SERVICE_NAME="cocktail-bot"
DOCKER_IMAGE="ceesaxp/cocktail-bot:latest"

# Create necessary directories
log "Creating application directories"
sudo mkdir -p $APP_DIR
sudo mkdir -p $DATA_DIR

# Generate API token if needed
API_TOKEN=""
if [ -f "$APP_DIR/api_tokens.yaml" ]; then
  # Try to read existing token from file
  API_TOKEN=$(sudo grep -oP '(?<=- )[A-Za-z0-9_-]+' "$APP_DIR/api_tokens.yaml" | head -1 || echo "")
fi

# Generate a new token if none exists
if [ -z "$API_TOKEN" ]; then
  log "Generating new API token"
  API_TOKEN=$(openssl rand -base64 24 | tr -dc 'a-zA-Z0-9_-')
  log "API token generated: $API_TOKEN"
fi

# Check if config file exists
if [ ! -f "$CONFIG_FILE" ]; then
  log "Config file not found, creating from template"
  cat > /tmp/config.yaml << EOF
log_level: info
telegram:
  token: "${TELEGRAM_BOT_TOKEN}"
  user: "${TELEGRAM_BOT_USERNAME}"
database:
  type: "${DATABASE_TYPE}"
  connection_string: "${DATABASE_CONNECTION_STRING}"
rate_limiting:
  requests_per_minute: 10
  requests_per_hour: 100
api:
  enabled: true
  port: 8080
  tokens_file: "$APP_DIR/api_tokens.yaml"
  rate_limit_per_min: 30
  rate_limit_per_hour: 300
EOF
  sudo mv /tmp/config.yaml $CONFIG_FILE
fi

# Create systemd service if it doesn't exist
if [ ! -f "/etc/systemd/system/$SERVICE_NAME.service" ]; then
  log "Creating systemd service"
  cat > /tmp/$SERVICE_NAME.service << EOF
[Unit]
Description=Cocktail Bot
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
Restart=always
RestartSec=10
Environment="DOCKER_IMAGE=$DOCKER_IMAGE"
Environment="API_TOKEN=$API_TOKEN"
ExecStartPre=-/usr/bin/docker pull \${DOCKER_IMAGE}
ExecStartPre=-/usr/bin/docker rm -f $SERVICE_NAME
ExecStart=/usr/bin/docker run --name $SERVICE_NAME \
  -v $CONFIG_FILE:/app/config.yaml \
  -v $DATA_DIR:/app/data \
  -p 8080:8080 \
  -e COCKTAILBOT_API_TOKENS=\${API_TOKEN} \
  --restart unless-stopped \
  \${DOCKER_IMAGE}
ExecStop=/usr/bin/docker stop $SERVICE_NAME
User=root
Group=root

[Install]
WantedBy=multi-user.target
EOF
  sudo mv /tmp/$SERVICE_NAME.service /etc/systemd/system/$SERVICE_NAME.service
  sudo systemctl daemon-reload
fi

# Update permissions
log "Setting correct permissions"
sudo chown -R root:root $APP_DIR
sudo chmod -R 755 $APP_DIR
sudo chmod -R 777 $DATA_DIR  # Ensure SQLite has write permissions

# Pull latest image
log "Pulling Docker image: $DOCKER_IMAGE"
if ! sudo docker pull $DOCKER_IMAGE; then
  log "Error pulling $DOCKER_IMAGE, checking for tagged images instead"
  # Try to find the most recent tag if latest doesn't exist
  LATEST_TAG=$(curl -s "https://hub.docker.com/v2/repositories/ceesaxp/cocktail-bot/tags/" | grep -o '"name":"[^"]*' | grep -v latest | sed 's/"name":"//g' | head -1)
  if [ -n "$LATEST_TAG" ]; then
    log "Found alternative tag: $LATEST_TAG, using it instead"
    DOCKER_IMAGE="ceesaxp/cocktail-bot:$LATEST_TAG"
    sudo docker pull $DOCKER_IMAGE
  else
    log "ERROR: Could not find any valid Docker image tag. Deployment failed."
    exit 1
  fi
fi

# Delete any existing webhook before starting the bot
log "Deleting any existing webhook"
curl -s "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/deleteWebhook" > /dev/null

# Restart service
log "Restarting service"
sudo systemctl restart $SERVICE_NAME
sudo systemctl enable $SERVICE_NAME

# Check service status
log "Service status:"
sudo systemctl status $SERVICE_NAME --no-pager

# Verify the service is running properly
sleep 5
log "Checking container logs"
sudo docker logs cocktail-bot | tail -n 15

log "Deployment completed successfully"
