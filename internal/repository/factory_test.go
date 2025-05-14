package repository

import (
	"context"
	"os"
	"testing"

	"github.com/ceesaxp/cocktail-bot/internal/config"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
)

func TestNew(t *testing.T) {
	// Create a logger
	testLogger := logger.New("debug")

	tests := []struct {
		name           string
		dbType         string
		connectionStr  string
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name:          "CSV Repository",
			dbType:        "csv",
			connectionStr: "test_users.csv",
			expectedErr:   false,
		},
		{
			name:           "Unsupported Database Type",
			dbType:         "unknown",
			connectionStr:  "connection",
			expectedErr:    true,
			expectedErrMsg: "unsupported database type: unknown",
		},
		{
			name:           "Empty Connection String",
			dbType:         "csv",
			connectionStr:  "",
			expectedErr:    true,
			expectedErrMsg: "file path cannot be empty",
		},
		{
			name:           "Nil Logger",
			dbType:         "csv",
			connectionStr:  "test_users.csv",
			expectedErr:    true,
			expectedErrMsg: "logger cannot be nil",
		},
	}

	// Clean up test file after tests
	defer os.Remove("test_users.csv")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create database config
			dbConfig := config.DatabaseConfig{
				Type:             tt.dbType,
				ConnectionString: tt.connectionStr,
			}

			// Create context - can be any value since we're using the any type
			ctx := context.Background()

			// Test with nil logger for the specific test case
			var logger *logger.Logger
			if tt.name != "Nil Logger" {
				logger = testLogger
			}

			// Create repository
			repo, err := New(ctx, dbConfig, logger)

			// Check for expected errors
			if tt.expectedErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}
				if err.Error() != tt.expectedErrMsg {
					t.Errorf("Expected error message %q but got %q", tt.expectedErrMsg, err.Error())
				}
				return
			}

			// Check repository was created successfully
			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}
			if repo == nil {
				t.Error("Expected repository to be created but got nil")
				return
			}

			// Close repository
			if err := repo.Close(); err != nil {
				t.Errorf("Failed to close repository: %v", err)
			}
		})
	}
}