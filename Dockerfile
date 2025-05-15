FROM golang:1.21-alpine AS builder

# Install necessary packages
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 go build -o cocktail-bot ./cmd/bot

# Create final lightweight image
FROM alpine:3.18

# Install ca-certificates for HTTPS connections
RUN apk add --no-cache ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/cocktail-bot /app/cocktail-bot

# Create data directory
RUN mkdir -p /app/data

# Default environment variables
ENV COCKTAILBOT_LOG_LEVEL=info

# Expose API port
EXPOSE 8080

# Command to run
ENTRYPOINT ["/app/cocktail-bot"]
CMD ["--config", "/app/config.yaml"]