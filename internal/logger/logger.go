package logger

import (
	"os"
	"sync"

	"github.com/gotify/plugin-api"
	"github.com/rs/zerolog"
)

var (
	globalLogger *zerolog.Logger
	once         sync.Once
	mu           sync.RWMutex
)

// Init initializes the global logger with initial configuration
func Init(pluginName string, userCtx plugin.UserContext) *zerolog.Logger {
	once.Do(func() {
		logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
			With().
			Str("plugin", pluginName).
			Uint("user_id", userCtx.ID).
			Str("user_name", userCtx.Name).
			Bool("is_admin", userCtx.Admin).
			Caller().
			Timestamp().
			Logger()

		globalLogger = &logger
	})

	return globalLogger
}

// Get returns the global logger instance
func Get() *zerolog.Logger {
	if globalLogger == nil {
		// If not initialized, create a default logger
		logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
			With().
			Timestamp().
			Logger()
		globalLogger = &logger
	}
	return globalLogger
}

// UpdateLogLevel updates the log level of the global logger
func UpdateLogLevel(level zerolog.Level) {
	mu.Lock()
	defer mu.Unlock()

	if globalLogger != nil {
		newLogger := globalLogger.Level(level)
		globalLogger = &newLogger
	}
}

// WithComponent adds a component field to the logger
// Useful for package-specific logging
func WithComponent(component string) *zerolog.Logger {
	logger := Get().With().Str("component", component).Logger()
	return &logger
}
