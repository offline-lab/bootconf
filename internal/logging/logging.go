// Package logging provides a leveled logger for bootconf. It uses a
// package-level default logger that writes to stderr with structured
// key=value output suitable for journalctl. Structured logs go to stderr
// so that table output on stdout stays clean.
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// LogLevel controls the verbosity of log output.
type LogLevel int

const (
	// DEBUG is the most verbose level, intended for development.
	DEBUG LogLevel = iota
	// INFO is the default level for normal operational messages.
	INFO
	// WARN indicates potential issues that are not fatal.
	WARN
	// ERROR indicates failures that should be investigated.
	ERROR
	// FATAL indicates unrecoverable errors.
	FATAL
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger writes structured, leveled log lines to an io.Writer.
type Logger struct {
	logger *log.Logger
	level  LogLevel
}

// New creates a Logger that writes to w at the given level.
func New(w io.Writer, level LogLevel) *Logger {
	return &Logger{
		logger: log.New(w, "", 0),
		level:  level,
	}
}

// SetLevel changes the minimum log level at runtime.
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// Debug logs at DEBUG level.
func (l *Logger) Debug(section, format string, args ...interface{}) {
	l.logf(DEBUG, section, format, args...)
}

// Info logs at INFO level.
func (l *Logger) Info(section, format string, args ...interface{}) {
	l.logf(INFO, section, format, args...)
}

// Warn logs at WARN level.
func (l *Logger) Warn(section, format string, args ...interface{}) {
	l.logf(WARN, section, format, args...)
}

// Error logs at ERROR level.
func (l *Logger) Error(section, format string, args ...interface{}) {
	l.logf(ERROR, section, format, args...)
}

func (l *Logger) logf(level LogLevel, section, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("bootconf: %s %s section=%s %s", timestamp, level, section, msg)
}

var defaultLogger = New(os.Stderr, INFO)

// SetLevel changes the default logger's minimum level.
func SetLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}

// Debug logs at DEBUG level on the default logger.
func Debug(section, format string, args ...interface{}) {
	defaultLogger.Debug(section, format, args...)
}

// Info logs at INFO level on the default logger.
func Info(section, format string, args ...interface{}) {
	defaultLogger.Info(section, format, args...)
}

// Warn logs at WARN level on the default logger.
func Warn(section, format string, args ...interface{}) {
	defaultLogger.Warn(section, format, args...)
}

// Error logs at ERROR level on the default logger.
func Error(section, format string, args ...interface{}) {
	defaultLogger.Error(section, format, args...)
}
