# Makefile for Cocktail Bot

.PHONY: build run docker docker-run clean test lint

BIN_NAME=cocktail-bot
DOCKER_IMAGE=cocktail-bot

# Build the binary
build:
	go build -o $(BIN_NAME) ./cmd/bot

# Run the binary
run: build
	./$(BIN_NAME) --config config.yaml

# Build Docker image
docker:
	docker build -t $(DOCKER_IMAGE) .

# Run Docker container
docker-run: docker
	docker run -v $(PWD)/config.yaml:/app/config.yaml -v $(PWD)/data:/app/data $(DOCKER_IMAGE)

# Run Docker Compose
docker-compose:
	docker-compose up -d

# Clean build artifacts
clean:
	rm -f $(BIN_NAME)

# Run tests
test:
	go test -v ./...

# Run linter
lint:
	golangci-lint run

# Initialize a new project
init:
	mkdir -p data
	cp config.yaml.example config.yaml

# Generate a sample CSV file
sample-csv:
	echo 'ID,Email,Date Added,Already Consumed' > data/users.csv
	echo '1,user1@example.com,$(shell date -u +"%Y-%m-%dT%H:%M:%SZ"),""' >> data/users.csv
	echo '2,user2@example.com,$(shell date -u +"%Y-%m-%dT%H:%M:%SZ"),""' >> data/users.csv
