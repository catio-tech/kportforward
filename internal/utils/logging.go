package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// Logger represents a simple logger for the application
type Logger struct {
	*log.Logger
	level   LogLevel
	output  io.Writer
	logFile *os.File // Keep reference to close file if needed
}

// LogLevel represents different logging levels
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

var logLevelNames = map[LogLevel]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
}

// NewLogger creates a new logger instance with stdout output
func NewLogger(level LogLevel) *Logger {
	return NewLoggerWithOutput(level, os.Stdout)
}

// NewLoggerWithOutput creates a new logger instance with custom output
func NewLoggerWithOutput(level LogLevel, output io.Writer) *Logger {
	return &Logger{
		Logger: log.New(output, "", 0),
		level:  level,
		output: output,
	}
}

// NewLoggerWithFile creates a new logger instance that writes to a file
func NewLoggerWithFile(level LogLevel, filePath string) (*Logger, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", filePath, err)
	}

	return &Logger{
		Logger:  log.New(file, "", 0),
		level:   level,
		output:  file,
		logFile: file,
	}, nil
}

// logf formats and logs a message at the specified level
func (l *Logger) logf(level LogLevel, format string, args ...any) {
	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelName := logLevelNames[level]
	message := fmt.Sprintf(format, args...)

	l.Printf("[%s] %s: %s", timestamp, levelName, message)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...any) {
	l.logf(LevelDebug, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...any) {
	l.logf(LevelInfo, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...any) {
	l.logf(LevelWarn, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...any) {
	l.logf(LevelError, format, args...)
}

// SetLevel changes the logging level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// Close closes the log file if one is open
func (l *Logger) Close() error {
	if l.logFile != nil {
		err := l.logFile.Close()
		l.logFile = nil // Set to nil to prevent double close
		return err
	}
	return nil
}

// FormatUptime formats a duration as a human-readable uptime string
func FormatUptime(duration time.Duration) string {
	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		return fmt.Sprintf("%dh%dm", hours, minutes)
	} else {
		days := int(duration.Hours()) / 24
		hours := int(duration.Hours()) % 24
		return fmt.Sprintf("%dd%dh", days, hours)
	}
}
