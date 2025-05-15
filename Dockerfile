FROM golang:1.23-alpine AS builder

# Install necessary packages for SQLite and building
RUN apk add --no-cache git make gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO enabled for SQLite support
RUN CGO_ENABLED=1 go build -o cocktail-bot ./cmd/bot
RUN go build -o token-generator ./cmd/token-generator

# Create final lightweight image
FROM alpine:3.18

# Install runtime dependencies for SQLite and HTTPS
RUN apk add --no-cache ca-certificates tzdata sqlite

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/cocktail-bot /app/cocktail-bot
COPY --from=builder /app/token-generator /app/token-generator

# Create data directory
RUN mkdir -p /app/data

# Default environment variables
ENV COCKTAILBOT_LOG_LEVEL=info

# Expose API port
EXPOSE 8080

# Command to run
ENTRYPOINT ["/app/cocktail-bot"]
CMD ["--config", "/app/config.yaml"]
