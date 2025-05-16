#!/bin/bash
set -e

# Script to set up passwordless sudo for specific commands needed by the deployment script
# This should be run once during the initial server setup

# Verify script is being run as root
if [ "$EUID" -ne 0 ]; then
  echo "This script must be run as root"
  exit 1
fi

# Get deployment username as argument or prompt for it
DEPLOY_USER=${1:-$(read -p "Enter deployment username: " username; echo $username)}

if [ -z "$DEPLOY_USER" ]; then
  echo "Deployment username cannot be empty"
  exit 1
fi

# Verify user exists
if ! id "$DEPLOY_USER" &>/dev/null; then
  echo "User $DEPLOY_USER does not exist"
  exit 1
fi

# Create a sudoers file for the deployment user
SUDOERS_FILE="/etc/sudoers.d/cocktail-bot-deploy"

cat > "$SUDOERS_FILE" << EOF
# Passwordless sudo for Cocktail Bot deployment
# Created $(date)

# Allow deployment user to manage the Docker service
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl start docker
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop docker
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart docker
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl status docker

# Allow deployment user to manage the Cocktail Bot service
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl start cocktail-bot
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop cocktail-bot
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart cocktail-bot
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl status cocktail-bot
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl enable cocktail-bot
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/systemctl daemon-reload

# Allow deployment user to manage Docker containers
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/docker pull *
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/docker rm *
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/docker stop *
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/docker run *
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/docker logs *

# Allow deployment user to manage application directories
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/mkdir -p /srv/cocktail-bot
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/mkdir -p /srv/cocktail-bot/data
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/chown -R * /srv/cocktail-bot
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/chmod -R * /srv/cocktail-bot
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/chmod -R * /srv/cocktail-bot/data

# Allow deployment user to move files to the application directory
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/mv /tmp/config.yaml /srv/cocktail-bot/config.yaml
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/mv /tmp/cocktail-bot.service /etc/systemd/system/cocktail-bot.service

# Allow deployment user to read from the application directory
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/cat /srv/cocktail-bot/api_tokens.yaml
$DEPLOY_USER ALL=(ALL) NOPASSWD: /usr/bin/grep * /srv/cocktail-bot/api_tokens.yaml
EOF

# Set the correct permissions
chmod 440 "$SUDOERS_FILE"

# Verify the syntax
if visudo -c -f "$SUDOERS_FILE"; then
  echo "Sudoers file created successfully"
else
  echo "Sudoers file syntax check failed. Removing file."
  rm "$SUDOERS_FILE"
  exit 1
fi

echo "Passwordless sudo configured for $DEPLOY_USER for Docker and Cocktail Bot operations"
echo "To test, run: su - $DEPLOY_USER -c 'sudo docker ps'"