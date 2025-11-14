package logger

import (
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger wraps zerolog logger
type Logger struct {
	logger zerolog.Logger
}

// New creates a new logger instance
func New(level, format, output string) *Logger {
	// Parse log level
	logLevel := parseLevel(level)
	zerolog.SetGlobalLevel(logLevel)

	// Set output writer
	var writer io.Writer = os.Stdout
	if output != "" && output != "stdout" {
		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to open log file")
		}
		writer = file
	}

	// Set format
	if format == "console" {
		writer = zerolog.ConsoleWriter{Out: writer}
	}

	logger := zerolog.New(writer).With().Timestamp().Caller().Logger()

	return &Logger{logger: logger}
}

// parseLevel converts string level to zerolog.Level
func parseLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

// Debug logs a debug message
func (l *Logger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

// Info logs an info message
func (l *Logger) Info() *zerolog.Event {
	return l.logger.Info()
}

// Warn logs a warning message
func (l *Logger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

// Error logs an error message
func (l *Logger) Error() *zerolog.Event {
	return l.logger.Error()
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal() *zerolog.Event {
	return l.logger.Fatal()
}

// With creates a child logger with additional context
func (l *Logger) With() zerolog.Context {
	return l.logger.With()
}

// GetLogger returns the underlying zerolog.Logger
func (l *Logger) GetLogger() zerolog.Logger {
	return l.logger
}

// Global logger instance
var global *Logger

// Init initializes the global logger
func Init(level, format, output string) {
	global = New(level, format, output)
}

// Get returns the global logger instance
func Get() *Logger {
	if global == nil {
		global = New("info", "json", "stdout")
	}
	return global
}
