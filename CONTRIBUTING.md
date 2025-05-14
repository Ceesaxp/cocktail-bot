# Contributing to Cocktail Bot

Thank you for considering contributing to Cocktail Bot! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

Please be respectful and considerate of others when contributing to this project.

## How to Contribute

1. Fork the repository
2. Create a new branch for your feature or bug fix: `git checkout -b feature/your-feature-name` or `git checkout -b fix/issue-name`
3. Make your changes
4. Run tests: `go test ./...`
5. Commit your changes with a descriptive commit message
6. Push to your fork: `git push origin your-branch-name`
7. Create a pull request to the main repository

## Development Setup

1. Clone the repository
2. Install Go 1.21 or later
3. Run `go mod download` to fetch dependencies
4. Run `go build ./cmd/bot` to build
5. Create a `config.yaml` file (copy from `config.yaml.example`)
6. Run tests with `go test ./...`

## Project Structure

- `cmd/bot`: Main application entry point
- `cmd/importcsv`: CSV import utility
- `internal/config`: Configuration handling
- `internal/domain`: Domain models and interfaces
- `internal/logger`: Logging system
- `internal/repository`: Database implementations
- `internal/service`: Business logic
- `internal/telegram`: Telegram bot interface
- `internal/utils`: Utility functions

## Testing

All code should include appropriate tests. Run tests with:

```bash
go test ./...
```

## Database Support

When adding or modifying database support, ensure you implement the `domain.Repository` interface and update the factory in `repository/factory.go`.

## Telegram Bot API

The bot uses the [go-telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api) library for Telegram integration.

## Submitting Changes

1. Make sure your code adheres to the existing style
2. Include tests for all new functionality
3. Update documentation as needed
4. Keep pull requests focused on a single change
5. Write a clear PR description explaining the changes

## License

By contributing to this project, you agree that your contributions will be licensed under the project's MIT License.
