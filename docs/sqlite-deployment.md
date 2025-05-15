# SQLite Deployment Guide

This document provides additional details for deploying the Cocktail Bot with SQLite database backend.

## Overview

SQLite is a great choice for many deployments as it's:
- Self-contained
- Zero-configuration
- Reliable
- Portable
- Performant for moderate workloads

However, there are some specific considerations when using SQLite with Docker, which we'll cover here.

## SQLite and CGO

SQLite in Go requires CGO to be enabled during compilation. By default, many Docker images disable CGO for better portability, but this won't work with SQLite. Our Dockerfile is specifically configured to:

1. Enable CGO during build
2. Include necessary SQLite development libraries
3. Include SQLite runtime libraries in the final image

## Testing SQLite Support

You can verify your Docker image has proper SQLite support by running the test script:

```bash
chmod +x ./scripts/test-docker-build.sh
./scripts/test-docker-build.sh
```

This script will:
- Build the Docker image for your platform
- Create a test directory with proper permissions
- Run the container with SQLite configuration
- Check logs for any errors
- Verify the database file is created

## Troubleshooting

### Permission Issues

The most common issues with SQLite in Docker relate to permissions:

1. **File permissions**: The container user must have write access to the SQLite database file and its directory:
   ```bash
   # On the host machine
   sudo chmod 777 /path/to/data/directory
   ```

2. **Volume mounts**: Ensure the volume is correctly mounted:
   ```bash
   docker run -v /absolute/path/to/data:/app/data ...
   ```

### Platform Issues

When building on Apple Silicon (M1/M2 Macs) and deploying to AMD64 servers:

1. Specify the platform when building:
   ```bash
   docker build --platform linux/amd64 -t cocktail-bot:latest .
   ```

2. For local testing on ARM:
   ```bash
   docker build --platform linux/arm64 -t cocktail-bot:local .
   ```

### Runtime Errors

If you see errors related to SQLite:

1. **Missing libraries**: Ensure the container has SQLite installed:
   ```
   apk add --no-cache sqlite
   ```

2. **Connection string**: Verify the path in your config.yaml:
   ```yaml
   database:
     type: "sqlite"
     connection_string: "/app/data/users.sqlite"  # Path inside container
   ```

3. **Database initialization**: Check if the database is properly initialized:
   ```bash
   docker exec -it cocktail-bot ls -la /app/data
   ```

## Multi-Architecture Builds

For production deployments, you might want to build for multiple architectures:

```bash
docker buildx create --use
docker buildx build --platform linux/amd64,linux/arm64 -t username/cocktail-bot:latest --push .
```

This allows the same image to run on both x86_64 servers and ARM-based machines.

## SQLite Performance Tips

For optimal SQLite performance in production:

1. Use WAL mode if there are multiple simultaneous connections
2. Consider periodic vacuum operations for database maintenance
3. Add proper indexes for frequently queried fields
4. Monitor database file size over time

## Migrating From CSV to SQLite

If you're migrating from CSV to SQLite, you can use the importcsv tool:

```bash
go run ./cmd/importcsv/main.go -input ./data/users.csv -output ./data/users.sqlite
```

This will create a new SQLite database with data from your CSV file.