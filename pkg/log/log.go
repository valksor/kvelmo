// Package log provides structured logging for kvelmo.
package log

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Level represents a log level.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel parses a level string.
func ParseLevel(s string) Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN", "WARNING":
		return LevelWarn
	case "ERROR":
		return LevelError
	default:
		return LevelInfo
	}
}

// Logger is a structured logger.
type Logger struct {
	mu       sync.Mutex
	out      io.Writer
	level    Level
	prefix   string
	fields   map[string]any
	colorize bool
}

// New creates a new logger.
func New() *Logger {
	return &Logger{
		out:      os.Stderr,
		level:    LevelInfo,
		fields:   make(map[string]any),
		colorize: true,
	}
}

// SetOutput sets the output writer.
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = w
}

// SetLevel sets the minimum log level.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetColorize enables or disables color output.
func (l *Logger) SetColorize(c bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.colorize = c
}

// WithPrefix returns a logger with a prefix.
func (l *Logger) WithPrefix(prefix string) *Logger {
	return &Logger{
		out:      l.out,
		level:    l.level,
		prefix:   prefix,
		fields:   copyFields(l.fields),
		colorize: l.colorize,
	}
}

// WithField returns a logger with an additional field.
func (l *Logger) WithField(key string, value any) *Logger {
	fields := copyFields(l.fields)
	fields[key] = value

	return &Logger{
		out:      l.out,
		level:    l.level,
		prefix:   l.prefix,
		fields:   fields,
		colorize: l.colorize,
	}
}

// WithFields returns a logger with additional fields.
func (l *Logger) WithFields(fields map[string]any) *Logger {
	newFields := copyFields(l.fields)
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		out:      l.out,
		level:    l.level,
		prefix:   l.prefix,
		fields:   newFields,
		colorize: l.colorize,
	}
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, args ...any) {
	l.log(LevelDebug, msg, args...)
}

// Info logs an info message.
func (l *Logger) Info(msg string, args ...any) {
	l.log(LevelInfo, msg, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, args ...any) {
	l.log(LevelWarn, msg, args...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, args ...any) {
	l.log(LevelError, msg, args...)
}

func (l *Logger) log(level Level, msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level {
		return
	}

	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	timestamp := time.Now().Format("15:04:05.000")
	levelStr := level.String()

	if l.colorize {
		levelStr = colorLevel(level)
	}

	var prefix string
	if l.prefix != "" {
		prefix = fmt.Sprintf("[%s] ", l.prefix)
	}

	var fieldsStr string
	if len(l.fields) > 0 {
		parts := make([]string, 0, len(l.fields))
		for k, v := range l.fields {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		fieldsStr = " " + strings.Join(parts, " ")
	}

	_, _ = fmt.Fprintf(l.out, "%s %s %s%s%s\n", timestamp, levelStr, prefix, msg, fieldsStr)
}

func colorLevel(level Level) string {
	switch level {
	case LevelDebug:
		return "\033[36mDEBUG\033[0m" // Cyan
	case LevelInfo:
		return "\033[32mINFO\033[0m" // Green
	case LevelWarn:
		return "\033[33mWARN\033[0m" // Yellow
	case LevelError:
		return "\033[31mERROR\033[0m" // Red
	default:
		return level.String()
	}
}

func copyFields(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}

	return dst
}

// Default logger.
var defaultLogger = New()

// SetLevel sets the default logger level.
func SetLevel(level Level) {
	defaultLogger.SetLevel(level)
}

// SetOutput sets the default logger output.
func SetOutput(w io.Writer) {
	defaultLogger.SetOutput(w)
}

// Debug logs a debug message to the default logger.
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Info logs an info message to the default logger.
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Warn logs a warning message to the default logger.
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Error logs an error message to the default logger.
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// WithPrefix returns a logger with a prefix.
func WithPrefix(prefix string) *Logger {
	return defaultLogger.WithPrefix(prefix)
}

// WithField returns a logger with an additional field.
func WithField(key string, value any) *Logger {
	return defaultLogger.WithField(key, value)
}

// WithFields returns a logger with additional fields.
func WithFields(fields map[string]any) *Logger {
	return defaultLogger.WithFields(fields)
}
