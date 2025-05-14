package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// Logger represents a structured logger for the application.
type Logger struct {
	level     Level
	prefix    string
	mu        sync.Mutex
	out       io.Writer
	timestamp bool
	logger    *log.Logger
}

// Level represents the logging level.
type Level int

const (
	// Debug level for detailed troubleshooting.
	DebugLevel Level = iota
	// Info level for general operational information.
	InfoLevel
	// Warn level for warning events that might lead to issues.
	WarnLevel
	// Error level for error events that might still allow the application to continue.
	ErrorLevel
	// Fatal level for critical errors that prevent the application from continuing.
	FatalLevel
)

var levelNames = map[Level]string{
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
	FatalLevel: "FATAL",
}

var levelColors = map[Level]string{
	DebugLevel: "\033[36m", // Cyan
	InfoLevel:  "\033[32m", // Green
	WarnLevel:  "\033[33m", // Yellow
	ErrorLevel: "\033[31m", // Red
	FatalLevel: "\033[35m", // Magenta
}

const (
	colorReset = "\033[0m"
)

// New creates a new logger with the specified logging level.
func New(level any) *Logger {
	var logLevel Level

	switch v := level.(type) {
	case string:
		logLevel = parseLevel(v)
	case int:
		logLevel = Level(v)
	case Level:
		logLevel = v
	default:
		logLevel = InfoLevel
	}

	l := &Logger{
		level:     logLevel,
		prefix:    "",
		out:       os.Stdout,
		timestamp: true,
		logger:    log.New(os.Stdout, "", 0),
	}
	return l
}

// FromConfig creates a new logger from the application config.
func FromConfig(logLevel string) *Logger {
	return New(logLevel)
}

// NewWithWriter creates a new logger with the specified logging level and writer.
func NewWithWriter(level any, writer io.Writer) *Logger {
	var logLevel Level

	switch v := level.(type) {
	case string:
		logLevel = parseLevel(v)
	case int:
		logLevel = Level(v)
	case Level:
		logLevel = v
	default:
		logLevel = InfoLevel
	}

	l := &Logger{
		level:     logLevel,
		prefix:    "",
		out:       writer,
		timestamp: true,
		logger:    log.New(writer, "", 0),
	}
	return l
}

// parseLevel converts a string level to a Level.
func parseLevel(level string) Level {
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	case "fatal":
		return FatalLevel
	default:
		return InfoLevel
	}
}

// LevelToString converts a Level to its string representation.
func LevelToString(level Level) string {
	switch level {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	default:
		return "unknown"
	}
}

// WithPrefix creates a new logger with the specified prefix.
func (l *Logger) WithPrefix(prefix string) *Logger {
	newLogger := &Logger{
		level:     l.level,
		prefix:    prefix,
		out:       l.out,
		timestamp: l.timestamp,
		logger:    log.New(l.out, "", 0),
	}
	return newLogger
}

// Debug logs a debug message with key-value pairs.
func (l *Logger) Debug(msg string, args ...any) {
	l.log(DebugLevel, msg, args...)
}

// Info logs an information message with key-value pairs.
func (l *Logger) Info(msg string, args ...any) {
	l.log(InfoLevel, msg, args...)
}

// Warn logs a warning message with key-value pairs.
func (l *Logger) Warn(msg string, args ...any) {
	l.log(WarnLevel, msg, args...)
}

// Error logs an error message with key-value pairs.
func (l *Logger) Error(msg string, args ...any) {
	l.log(ErrorLevel, msg, args...)
}

// Fatal logs a fatal message with key-value pairs and then exits the program.
func (l *Logger) Fatal(msg string, args ...any) {
	l.log(FatalLevel, msg, args...)
	os.Exit(1)
}

// log formats and writes the log message.
func (l *Logger) log(level Level, msg string, args ...any) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Format timestamp
	var timeStr string
	if l.timestamp {
		timeStr = time.Now().Format("2006-01-02 15:04:05.000")
	}

	// Format level with color
	levelName := levelNames[level]
	levelColor := levelColors[level]

	// Format key-value pairs
	var kvStr string
	if len(args) > 0 {
		if len(args)%2 != 0 {
			args = append(args, "MISSING_VALUE")
		}
		pairs := make([]string, 0, len(args)/2)
		for i := 0; i < len(args); i += 2 {
			key, ok := args[i].(string)
			if !ok {
				key = fmt.Sprintf("%v", args[i])
			}
			value := args[i+1]
			pairs = append(pairs, fmt.Sprintf("%s=%v", key, value))
		}
		kvStr = " " + strings.Join(pairs, " ")
	}

	// Format prefix
	var prefixStr string
	if l.prefix != "" {
		prefixStr = "[" + l.prefix + "] "
	}

	// Build the final log line
	var builder strings.Builder

	// Add timestamp if enabled
	if l.timestamp {
		builder.WriteString(fmt.Sprintf("%s ", timeStr))
	}

	// Add colored level
	builder.WriteString(fmt.Sprintf("%s%-5s%s ", levelColor, levelName, colorReset))

	// Add prefix if present
	if prefixStr != "" {
		builder.WriteString(prefixStr)
	}

	// Add message
	builder.WriteString(msg)

	// Add key-value pairs
	builder.WriteString(kvStr)

	// Output the log line
	l.logger.Println(builder.String())
}

// SetLevel sets the logging level.
func (l *Logger) SetLevel(level any) {
	switch v := level.(type) {
	case string:
		l.level = parseLevel(v)
	case int:
		l.level = Level(v)
	case Level:
		l.level = v
	}
}

// GetLevel returns the current logging level.
func (l *Logger) GetLevel() Level {
	return l.level
}

// DisableTimestamp disables the timestamp in log messages.
func (l *Logger) DisableTimestamp() {
	l.timestamp = false
}

// EnableTimestamp enables the timestamp in log messages.
func (l *Logger) EnableTimestamp() {
	l.timestamp = true
}
