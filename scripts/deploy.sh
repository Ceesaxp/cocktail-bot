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
ExecStartPre=-/usr/bin/docker pull $DOCKER_IMAGE
ExecStartPre=-/usr/bin/docker rm -f $SERVICE_NAME
ExecStart=/usr/bin/docker run --name $SERVICE_NAME \
  -v $CONFIG_FILE:/app/config.yaml \
  -v $DATA_DIR:/app/data \
  -p 8080:8080 \
  --restart unless-stopped \
  $DOCKER_IMAGE
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

# Pull latest image
log "Pulling latest Docker image"
sudo docker pull $DOCKER_IMAGE

# Restart service
log "Restarting service"
sudo systemctl restart $SERVICE_NAME
sudo systemctl enable $SERVICE_NAME

# Check service status
log "Service status:"
sudo systemctl status $SERVICE_NAME --no-pager

log "Deployment completed successfully"
