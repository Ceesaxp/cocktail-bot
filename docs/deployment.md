# Deployment Guide

This document outlines the deployment process for the Cocktail Bot using the CI/CD pipeline.

## Overview

The Cocktail Bot uses GitHub Actions for continuous integration and deployment to a Digital Ocean droplet. The pipeline:

1. Runs tests and linting
2. Builds and pushes a Docker image
3. Deploys to production when manually triggered

## Prerequisites

Before setting up the CI/CD pipeline, you need:

1. A GitHub repository for the code
2. A Digital Ocean droplet running Ubuntu
3. A Docker Hub account
4. A Telegram Bot token

## Setup GitHub Secrets

Add the following secrets to your GitHub repository:

1. `DOCKER_USERNAME` - Your Docker Hub username
2. `DOCKER_PASSWORD` - Your Docker Hub password or access token
3. `SSH_PRIVATE_KEY` - Private SSH key for accessing your Digital Ocean droplet
4. `SSH_KNOWN_HOSTS` - SSH known hosts entry for your droplet
5. `DROPLET_IP` - IP address of your Digital Ocean droplet
6. `DEPLOY_USER` - Username for SSH access to your droplet

## Setting Up the Digital Ocean Droplet

1. Create a new Ubuntu droplet on Digital Ocean
2. Install Docker:
   ```bash
   sudo apt update
   sudo apt install -y apt-transport-https ca-certificates curl software-properties-common
   curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
   sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable"
   sudo apt update
   sudo apt install -y docker-ce
   sudo systemctl enable docker
   ```

3. Create the application directory:
   ```bash
   sudo mkdir -p /srv/cocktail-bot/data
   # Ensure correct permissions for SQLite database
   sudo chmod 777 /srv/cocktail-bot/data
   ```

4. Create a configuration file:
   ```bash
   sudo nano /srv/cocktail-bot/config.yaml
   ```

   With the following content (replace with your values):
   ```yaml
   log_level: info
   telegram:
     token: "YOUR_TELEGRAM_BOT_TOKEN"
     user: "your_bot_username"
   database:
     type: "sqlite"
     connection_string: "/app/data/users.sqlite"
   rate_limiting:
     requests_per_minute: 10
     requests_per_hour: 100
   api:
     enabled: true
     port: 8080
     tokens_file: "/app/api_tokens.yaml"
     rate_limit_per_min: 30
     rate_limit_per_hour: 300
   ```

> **Note**: The Docker image is built with SQLite support enabled. The connection string path `/app/data/users.sqlite` refers to the path inside the container, which is mapped to `/srv/cocktail-bot/data` on the host system.

## Manual Deployment

If you need to deploy manually without using the CI/CD pipeline:

1. Build the Docker image locally (with SQLite support):
   ```bash
   # For AMD64 (Digital Ocean droplet)
   docker build --platform linux/amd64 -t ceesaxp/cocktail-bot:latest .
   
   # If building on Apple Silicon (M1/M2 Mac) for testing locally
   docker build --platform linux/arm64 -t ceesaxp/cocktail-bot:local .
   ```

2. Push the image to Docker Hub:
   ```bash
   docker push ceesaxp/cocktail-bot:latest
   ```

3. SSH into your Digital Ocean droplet:
   ```bash
   ssh user@your-droplet-ip
   ```

4. Pull and run the Docker image:
   ```bash
   docker pull ceesaxp/cocktail-bot:latest
   docker run -d --name cocktail-bot \
     -v /srv/cocktail-bot/config.yaml:/app/config.yaml \
     -v /srv/cocktail-bot/data:/app/data \
     -p 8080:8080 \
     --restart unless-stopped \
     ceesaxp/cocktail-bot:latest
   ```
   
5. Verify the container is running properly:
   ```bash
   docker logs cocktail-bot
   ```
   
   You should see log output indicating the bot has connected to Telegram and the SQLite database has been initialized successfully.

## Triggering Deployment with GitHub Actions

To deploy using the CI/CD pipeline:

1. Go to the "Actions" tab in your GitHub repository
2. Select the "CI/CD Pipeline" workflow
3. Click "Run workflow"
4. Select the branch to deploy from
5. Set "Deploy to production?" to "true"
6. Click "Run workflow"

## Monitoring and Logs

To view logs of the running container:

```bash
docker logs -f cocktail-bot
```

To check the status of the service:

```bash
sudo systemctl status cocktail-bot
```

## Rolling Back

If you need to roll back to a previous version:

1. SSH into your Digital Ocean droplet
2. Pull the specific tagged version:
   ```bash
   docker pull ceesaxp/cocktail-bot:previous-tag
   ```
3. Update the systemd service to use this tag:
   ```bash
   sudo sed -i 's/ceesaxp\/cocktail-bot:latest/ceesaxp\/cocktail-bot:previous-tag/g' /etc/systemd/system/cocktail-bot.service
   ```
4. Restart the service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl restart cocktail-bot
   ```
