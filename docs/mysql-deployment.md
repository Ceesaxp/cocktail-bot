# MySQL Deployment Guide

This document provides additional details for deploying the Cocktail Bot with a MySQL (MariaDB) database backend using Docker Compose.

## Overview

MySQL (or MariaDB) is a popular relational database for production deployments. This guide shows how to initialize the database schema and run the application with Docker Compose.

## Prerequisites

- Docker and Docker Compose installed on your host
- Sufficient disk space for the database files

## Directory Layout

Ensure your project directory contains:
```text
docker-compose.yml
.env
db/schema/init/mysql/01-schema.sql
Caddyfile
config.yaml  # optional, environment variables override
```

## Environment Variables

The `.env` file drives the configuration. Copy and edit:

```bash
cp .env.example .env
```

Set at a minimum:
- `TELEGRAM_BOT_TOKEN`
- `TELEGRAM_BOT_USERNAME`
- `MYSQL_ROOT_PASSWORD`
- `MYSQL_PASSWORD`
- `API_TOKENS`

You can also override `REGISTRY` and `TAG` if you're using a private registry or different image tags.

## Database Initialization

On first start, the MySQL container will automatically run SQL scripts in `db/schema/init/mysql/`. The provided `01-schema.sql` defines the required tables and indexes.

## Starting Services

Run the full stack (MySQL, Cocktail Bot, Caddy):

```bash
docker-compose up -d
```

## Verifying the Deployment

Check the MySQL container health:

```bash
docker-compose ps mysql
```

Inspect the logs of the Cocktail Bot:

```bash
docker logs cocktail-bot --tail 20
```

Confirm the API health endpoint:

```bash
curl http://localhost/api/health
```

## Data Persistence

The MySQL data is stored in a Docker named volume `mysql-data`. You can back it up or change to a host bind mount by editing the `docker-compose.yml`.

## Troubleshooting

See [Troubleshooting](docs/troubleshooting.md) for common errors and tips.

## Cleaning Up

To remove containers, networks, and volumes:

```bash
docker-compose down --volumes
```