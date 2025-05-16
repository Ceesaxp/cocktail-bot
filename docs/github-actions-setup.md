# GitHub Actions Deployment Setup

This document provides details on setting up the GitHub Actions workflow for deploying the Cocktail Bot application to a DigitalOcean Droplet using SSH.

## Required GitHub Secrets

To use the improved CI/CD workflow, you need to add the following secrets to your GitHub repository:

1. **SSH_PRIVATE_KEY**
   - The private SSH key used to connect to your DigitalOcean Droplet
   - Generate a dedicated deployment key (don't use your personal key)
   - Command to generate a new key: `ssh-keygen -t ed25519 -C "github-actions-deploy"`
   - Add the public key to your server's `~/.ssh/authorized_keys` file

2. **SSH_KNOWN_HOSTS**
   - The SSH known hosts entry for your server to verify its identity
   - Run this command to get the value: `ssh-keyscan -H your-server-ip`

3. **DROPLET_IP**
   - The IP address of your DigitalOcean Droplet

4. **DEPLOY_USER**
   - The username used for SSH access to the Droplet

5. **DOCKER_USERNAME** and **DOCKER_PASSWORD**
   - Credentials for DockerHub to push the container image

6. **TELEGRAM_BOT_TOKEN**
   - Your Telegram Bot token obtained from BotFather

7. **TELEGRAM_BOT_USERNAME**
   - The username of your Telegram Bot

8. **DATABASE_TYPE**
   - The type of database to use (csv, sqlite, googlesheet, etc.)

9. **DATABASE_CONNECTION_STRING**
   - The connection string for your database

## Setting Up GitHub Secrets

1. Navigate to your GitHub repository page
2. Go to Settings → Secrets and variables → Actions
3. Click "New repository secret"
4. Add each of the required secrets listed above
5. Ensure the SSH_PRIVATE_KEY contains the entire private key including begin and end lines

## Server Setup

1. Ensure Docker is installed on your DigitalOcean Droplet
2. Create the required directories:
   ```bash
   sudo mkdir -p /srv/cocktail-bot/data
   sudo chmod -R 777 /srv/cocktail-bot/data
   ```

3. Ensure the deployment user has sudo privileges
4. Add the deployment SSH public key to authorized_keys:
   ```bash
   echo "ssh-ed25519 AAAA..." >> ~/.ssh/authorized_keys
   ```

## Triggering Deployment

The workflow can be triggered in two ways:

1. **Automatically**: When code is pushed to the main/master branch, the workflow will run tests and build the Docker image, but will not deploy automatically.

2. **Manually**: Using the GitHub Actions workflow_dispatch trigger:
   - Go to Actions → "Improved CI/CD Pipeline"
   - Click "Run workflow"
   - Select the branch (usually main or master)
   - Set "Deploy to production?" to "true"
   - Click "Run workflow"

## Troubleshooting

If the deployment fails, check the following:

1. Verify all secrets are correctly set in GitHub
2. Check that the SSH key has been added to the server's authorized_keys
3. Ensure the deployment user has sudo privileges on the server
4. Check the GitHub Actions logs for specific error messages
5. Manually SSH into the server to verify connectivity
6. Check Docker is running on the server: `systemctl status docker`

## Security Considerations

1. Use a dedicated deployment key rather than a personal SSH key
2. Consider restricting the deployment user's sudo privileges to only the commands needed
3. Regularly rotate the SSH keys and DockerHub credentials
4. For production, consider using GitHub's OpenID Connect (OIDC) for keyless authentication