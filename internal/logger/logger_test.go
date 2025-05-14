package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestLoggerLevels(t *testing.T) {
	tests := []struct {
		level    interface{}
		message  string
		logFunc  func(l *Logger, msg string, args ...interface{})
		expected bool
	}{
		{"info", "test debug message", func(l *Logger, msg string, args ...interface{}) { l.Debug(msg, args...) }, false},
		{"info", "test info message", func(l *Logger, msg string, args ...interface{}) { l.Info(msg, args...) }, true},
		{"info", "test warn message", func(l *Logger, msg string, args ...interface{}) { l.Warn(msg, args...) }, true},
		{"info", "test error message", func(l *Logger, msg string, args ...interface{}) { l.Error(msg, args...) }, true},
		{"debug", "test debug message", func(l *Logger, msg string, args ...interface{}) { l.Debug(msg, args...) }, true},
		{"warn", "test info message", func(l *Logger, msg string, args ...interface{}) { l.Info(msg, args...) }, false},
		{"warn", "test warn message", func(l *Logger, msg string, args ...interface{}) { l.Warn(msg, args...) }, true},
		{"error", "test warn message", func(l *Logger, msg string, args ...interface{}) { l.Warn(msg, args...) }, false},
		{"error", "test error message", func(l *Logger, msg string, args ...interface{}) { l.Error(msg, args...) }, true},
	}

	for _, test := range tests {
		var buf bytes.Buffer
		logger := NewWithWriter(test.level, &buf)

		// Disable timestamp for predictable output
		logger.DisableTimestamp()

		test.logFunc(logger, test.message)

		if test.expected {
			if !strings.Contains(buf.String(), test.message) {
				t.Errorf("Expected message '%s' to be logged with level '%s', but it wasn't", test.message, test.level)
			}
		} else {
			if strings.Contains(buf.String(), test.message) {
				t.Errorf("Expected message '%s' NOT to be logged with level '%s', but it was", test.message, test.level)
			}
		}
	}
}

func TestLoggerKeyValues(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter("info", &buf)
	logger.DisableTimestamp()

	logger.Info("test message", "key1", "value1", "key2", 123)

	output := buf.String()
	if !strings.Contains(output, "key1=value1") {
		t.Errorf("Expected key-value pair 'key1=value1' in output, but it wasn't found")
	}

	if !strings.Contains(output, "key2=123") {
		t.Errorf("Expected key-value pair 'key2=123' in output, but it wasn't found")
	}
}

func TestLoggerUnevenKeyValues(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter("info", &buf)
	logger.DisableTimestamp()

	// Test with odd number of args
	logger.Info("test message", "key1", "value1", "orphan_key")

	output := buf.String()
	if !strings.Contains(output, "key1=value1") {
		t.Errorf("Expected key-value pair 'key1=value1' in output, but it wasn't found")
	}

	if !strings.Contains(output, "orphan_key=MISSING_VALUE") {
		t.Errorf("Expected orphan key to have MISSING_VALUE placeholder")
	}
}

func TestLoggerPrefix(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter("info", &buf)
	logger.DisableTimestamp()

	// Create prefixed logger
	prefixedLogger := logger.WithPrefix("TEST")
	prefixedLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "[TEST]") {
		t.Errorf("Expected prefix '[TEST]' in output, but it wasn't found")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", DebugLevel},
		{"DEBUG", DebugLevel},
		{"info", InfoLevel},
		{"INFO", InfoLevel},
		{"warn", WarnLevel},
		{"warning", WarnLevel},
		{"WARN", WarnLevel},
		{"error", ErrorLevel},
		{"ERROR", ErrorLevel},
		{"fatal", FatalLevel},
		{"FATAL", FatalLevel},
		{"unknown", InfoLevel}, // Default to info
		{"", InfoLevel},        // Default to info
	}

	for _, test := range tests {
		result := parseLevel(test.input)
		if result != test.expected {
			t.Errorf("parseLevel(%q) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestLevelToString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DebugLevel, "debug"},
		{InfoLevel, "info"},
		{WarnLevel, "warn"},
		{ErrorLevel, "error"},
		{FatalLevel, "fatal"},
		{Level(99), "unknown"},
	}

	for _, test := range tests {
		result := LevelToString(test.level)
		if result != test.expected {
			t.Errorf("LevelToString(%v) = %q, expected %q", test.level, result, test.expected)
		}
	}
}

func TestLoggerWithIntLevel(t *testing.T) {
	var buf bytes.Buffer

	// Test with integer level
	logger := NewWithWriter(2, &buf) // WarnLevel
	logger.DisableTimestamp()

	logger.Debug("debug message") // Should be filtered
	logger.Info("info message")   // Should be filtered
	logger.Warn("warn message")   // Should be logged
	logger.Error("error message") // Should be logged

	output := buf.String()
	
	if strings.Contains(output, "debug message") {
		t.Error("Debug message should not appear when level is set to Warn")
	}
	
	if strings.Contains(output, "info message") {
		t.Error("Info message should not appear when level is set to Warn")
	}
	
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message should appear when level is set to Warn")
	}
	
	if !strings.Contains(output, "error message") {
		t.Error("Error message should appear when level is set to Warn")
	}
}

func TestSetAndGetLevel(t *testing.T) {
	logger := New("info")
	
	if logger.GetLevel() != InfoLevel {
		t.Errorf("Expected initial level to be InfoLevel, got %v", logger.GetLevel())
	}
	
	logger.SetLevel("debug")
	if logger.GetLevel() != DebugLevel {
		t.Errorf("Expected level to be DebugLevel after SetLevel, got %v", logger.GetLevel())
	}
	
	logger.SetLevel("error")
	if logger.GetLevel() != ErrorLevel {
		t.Errorf("Expected level to be ErrorLevel after SetLevel, got %v", logger.GetLevel())
	}
	
	// Test setting level with integer
	logger.SetLevel(0) // DebugLevel
	if logger.GetLevel() != DebugLevel {
		t.Errorf("Expected level to be DebugLevel after SetLevel(0), got %v", logger.GetLevel())
	}
	
	// Test setting level with Level type
	logger.SetLevel(ErrorLevel)
	if logger.GetLevel() != ErrorLevel {
		t.Errorf("Expected level to be ErrorLevel after SetLevel(ErrorLevel), got %v", logger.GetLevel())
	}
}

func TestTimestampControl(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter("info", &buf)
	
	// Enable timestamp (default)
	logger.EnableTimestamp()
	buf.Reset()
	logger.Info("with timestamp")
	withTimestamp := buf.String()
	
	// Disable timestamp
	logger.DisableTimestamp()
	buf.Reset()
	logger.Info("without timestamp")
	withoutTimestamp := buf.String()
	
	// First log should be longer due to timestamp
	if len(withoutTimestamp) >= len(withTimestamp) {
		t.Error("Expected log with timestamp to be longer than log without timestamp")
	}
}