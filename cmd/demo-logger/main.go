package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/logger"
)

func main() {
	fmt.Println("Logger Demo")
	fmt.Println("===========")

	// Create loggers with different levels
	debugLogger := logger.New("debug")
	infoLogger := logger.New("info")
	warnLogger := logger.New("warn") // Will use below

	// Show basic logging at different levels
	fmt.Println("\n1. Basic logging with different levels:")
	debugLogger.Debug("This is a debug message")
	debugLogger.Info("This is an info message")
	debugLogger.Warn("This is a warning message")
	debugLogger.Error("This is an error message")

	// Show level filtering
	fmt.Println("\n2. Level filtering (info level):")
	infoLogger.Debug("This debug message won't show with info level") // This won't show
	infoLogger.Info("This info message will show")
	infoLogger.Warn("This warning will show")
	infoLogger.Error("This error will show")
	
	// Show warn level filtering
	fmt.Println("\n2a. Level filtering (warn level):")
	warnLogger.Debug("This debug message won't show with warn level") // This won't show
	warnLogger.Info("This info message won't show with warn level")   // This won't show
	warnLogger.Warn("This warning will show")
	warnLogger.Error("This error will show")

	// Show structured logging with key-values
	fmt.Println("\n3. Structured logging with key-value pairs:")
	debugLogger.Info(
		"User logged in",
		"user_id", 12345,
		"ip_address", "192.168.1.1",
		"login_time", time.Now().Format(time.RFC3339),
	)

	// Using with prefix for module/component logging
	fmt.Println("\n4. Component-specific logging with prefixes:")
	dbLogger := debugLogger.WithPrefix("DATABASE")
	apiLogger := debugLogger.WithPrefix("API")
	
	dbLogger.Info("Database connection established", "host", "localhost", "port", 5432)
	apiLogger.Warn("API rate limit approaching", "endpoint", "/users", "rate", "95%")

	// Changing log level dynamically
	fmt.Println("\n5. Changing log level dynamically:")
	dynamicLogger := logger.New("info")
	dynamicLogger.Info("Current level is INFO")
	dynamicLogger.Debug("This debug message won't show") // Won't show
	
	fmt.Println("Changing log level to DEBUG...")
	dynamicLogger.SetLevel("debug")
	
	dynamicLogger.Debug("Now this debug message will show") // Will show
	
	// Using a custom writer
	fmt.Println("\n6. Writing logs to a file:")
	f, err := os.Create("demo.log")
	if err != nil {
		fmt.Printf("Error creating log file: %v\n", err)
		return
	}
	defer f.Close()
	
	fileLogger := logger.NewWithWriter("info", f)
	fileLogger.Info("This message goes to the file", "file", "demo.log")
	
	fmt.Println("Log written to demo.log")
	fmt.Println("\nDemo completed. Check terminal output and demo.log file.")
}