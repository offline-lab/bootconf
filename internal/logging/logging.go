package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
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

type Logger struct {
	logger *log.Logger
	level  LogLevel
}

func New(w io.Writer, level LogLevel) *Logger {
	return &Logger{
		logger: log.New(w, "", 0),
		level:  level,
	}
}

func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

func (l *Logger) Debug(section, format string, args ...interface{}) {
	l.logf(DEBUG, section, format, args...)
}

func (l *Logger) Info(section, format string, args ...interface{}) {
	l.logf(INFO, section, format, args...)
}

func (l *Logger) Warn(section, format string, args ...interface{}) {
	l.logf(WARN, section, format, args...)
}

func (l *Logger) Error(section, format string, args ...interface{}) {
	l.logf(ERROR, section, format, args...)
}

func (l *Logger) logf(level LogLevel, section, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("bootconf: %s %s section::%s %s", timestamp, level, section, msg)
}

var defaultLogger = New(os.Stdout, INFO)

func SetLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}

func Debug(section, format string, args ...interface{}) {
	defaultLogger.Debug(section, format, args...)
}

func Info(section, format string, args ...interface{}) {
	defaultLogger.Info(section, format, args...)
}

func Warn(section, format string, args ...interface{}) {
	defaultLogger.Warn(section, format, args...)
}

func Error(section, format string, args ...interface{}) {
	defaultLogger.Error(section, format, args...)
}
