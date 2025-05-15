#!/bin/bash
set -e

# Direct deployment script for Cocktail Bot
# This script builds, tags, and directly deploys to the server

# Colors for better readability
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configuration
TAG="manual-$(date +%Y%m%d-%H%M%S)"
IMAGE_NAME="ceesaxp/cocktail-bot:$TAG"
SERVER_USER=${DEPLOY_USER:-"your-username"}  # Set DEPLOY_USER env var or change default
SERVER_IP=${DROPLET_IP:-"your-server-ip"}    # Set DROPLET_IP env var or change default
REMOTE_DIR="/srv/cocktail-bot"

# Check if required variables are set
if [[ "$SERVER_USER" == "your-username" || "$SERVER_IP" == "your-server-ip" ]]; then
  echo -e "${RED}ERROR: Please set DEPLOY_USER and DROPLET_IP environment variables${NC}"
  echo "Example: DEPLOY_USER=ubuntu DROPLET_IP=123.456.789.0 ./scripts/direct-deploy.sh"
  exit 1
fi

# Log function
log() {
  echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
  echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

error() {
  echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

# Step 1: Build the image for x86_64 (AMD64)
log "Building Docker image for AMD64: $IMAGE_NAME"
docker build --platform linux/amd64 -t $IMAGE_NAME .

# Step 2: Save the image to a tarball
log "Saving image to tarball"
docker save $IMAGE_NAME -o ./cocktail-bot-image.tar

# Step 3: Ensure remote directories exist
log "Preparing server directories"
ssh $SERVER_USER@$SERVER_IP "sudo mkdir -p $REMOTE_DIR/data && sudo chmod 777 $REMOTE_DIR/data"

# Step 4: Copy the image to the server
log "Copying image to server (this may take a while)"
scp ./cocktail-bot-image.tar $SERVER_USER@$SERVER_IP:/tmp/cocktail-bot-image.tar

# Step 5: Copy config if it doesn't exist
if [ ! -f "$REMOTE_DIR/config.yaml" ]; then
  warn "Remote config file not found, checking for local config"
  if [ -f "./config.yaml" ]; then
    log "Copying local config to server"
    scp ./config.yaml $SERVER_USER@$SERVER_IP:/tmp/config.yaml
    ssh $SERVER_USER@$SERVER_IP "sudo mv /tmp/config.yaml $REMOTE_DIR/config.yaml"
  else
    error "No local config.yaml found! Please create one manually on the server."
  fi
fi

# Step 6: Delete webhook before deployment
log "Deleting Telegram webhook"
TOKEN=$(grep -oP '(?<=token: ")[^"]*' config.yaml || echo "")
if [ -n "$TOKEN" ]; then
  curl -s "https://api.telegram.org/bot$TOKEN/deleteWebhook" > /dev/null
  log "Webhook deleted successfully"
else
  warn "Couldn't extract Telegram token from config.yaml. Webhook deletion skipped."
fi

# Step 7: Load and run the image on the server
log "Loading and running image on server"
ssh $SERVER_USER@$SERVER_IP << EOF
  # Load the image
  sudo docker load -i /tmp/cocktail-bot-image.tar
  
  # Clean up old container if it exists
  sudo docker rm -f cocktail-bot || true
  
  # Run the new container
  sudo docker run -d --name cocktail-bot \\
    -v $REMOTE_DIR/config.yaml:/app/config.yaml \\
    -v $REMOTE_DIR/data:/app/data \\
    -p 8080:8080 \\
    --restart unless-stopped \\
    $IMAGE_NAME
    
  # Clean up the tarball
  rm /tmp/cocktail-bot-image.tar
  
  # Show logs
  echo "Container logs:"
  sleep 5
  sudo docker logs cocktail-bot
EOF

# Step 7: Clean up local tarball
log "Cleaning up local files"
rm ./cocktail-bot-image.tar

log "Deployment completed successfully!"
log "Image: $IMAGE_NAME is now running on $SERVER_IP"