package config

import (
	"net/url"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestParseEnvVars(t *testing.T) {
	// Setup environment variables
	envVars := map[string]string{
		"TG_PLUGIN__LOG_LEVEL":                  "debug",
		"TG_PLUGIN__GOTIFY_URL":                 "http://gotify.example.com:8080",
		"TG_PLUGIN__GOTIFY_CLIENT_TOKEN":        "some-client-token",
		"TG_PLUGIN__WS_HANDSHAKE_TIMEOUT":       "15",
		"TG_PLUGIN__WS_PING_INTERVAL":           "45",
		"TG_PLUGIN__WS_PONG_WAIT":               "90",
		"TG_PLUGIN__TELEGRAM_DEFAULT_BOT_TOKEN": "default-bot-token",
		"TG_PLUGIN__TELEGRAM_DEFAULT_CHAT_IDS":  "123,456",
		"TG_PLUGIN__MESSAGE_INCLUDE_APP_NAME":   "true",
		"TG_PLUGIN__MESSAGE_INCLUDE_TIMESTAMP":  "true",
		"TG_PLUGIN__MESSAGE_PARSE_MODE":         "HTML",
		"TG_PLUGIN__MESSAGE_INCLUDE_PRIORITY":   "true",
		"TG_PLUGIN__MESSAGE_PRIORITY_THRESHOLD": "5",
	}

	// Set environment variables
	for k, v := range envVars {
		err := os.Setenv(k, v)
		assert.NoError(t, err)
	}

	// Cleanup environment variables after test
	defer func() {
		for k := range envVars {
			os.Unsetenv(k)
		}
	}()

	// Parse environment variables
	cfg, err := ParseEnvVars()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Validate parsed config
	expectedURL, _ := url.Parse("http://gotify.example.com:8080")

	// Log Options
	assert.Equal(t, "debug", cfg.Settings.LogOptions.LogLevel)

	// Gotify Server
	assert.Equal(t, expectedURL, cfg.Settings.GotifyServer.Url)
	assert.Equal(t, "some-client-token", cfg.Settings.GotifyServer.ClientToken)
	assert.Equal(t, 15, cfg.Settings.GotifyServer.Websocket.HandshakeTimeout)
	assert.Equal(t, 45, cfg.Settings.GotifyServer.Websocket.PingInterval)
	assert.Equal(t, 90, cfg.Settings.GotifyServer.Websocket.PongWait)

	// Telegram
	assert.Equal(t, "default-bot-token", cfg.Settings.Telegram.DefaultBotToken)
	assert.Equal(t, []string{"123", "456"}, cfg.Settings.Telegram.DefaultChatIDs)

	// Message Format Options
	assert.True(t, cfg.Settings.Telegram.MessageFormatOptions.IncludeAppName)
	assert.True(t, cfg.Settings.Telegram.MessageFormatOptions.IncludeTimestamp)
	assert.Equal(t, "HTML", cfg.Settings.Telegram.MessageFormatOptions.ParseMode)
	assert.True(t, cfg.Settings.Telegram.MessageFormatOptions.IncludePriority)
	assert.Equal(t, 5, cfg.Settings.Telegram.MessageFormatOptions.PriorityThreshold)
}

func TestParseEnvVars_DefaultValues(t *testing.T) {
	// Clear any existing environment variables that might interfere
	envVars := []string{
		"TG_PLUGIN__LOG_LEVEL",
		"TG_PLUGIN__GOTIFY_URL",
		"TG_PLUGIN__GOTIFY_CLIENT_TOKEN",
		"TG_PLUGIN__WS_HANDSHAKE_TIMEOUT",
		"TG_PLUGIN__WS_PING_INTERVAL",
		"TG_PLUGIN__WS_PONG_WAIT",
		"TG_PLUGIN__TELEGRAM_DEFAULT_BOT_TOKEN",
		"TG_PLUGIN__TELEGRAM_DEFAULT_CHAT_IDS",
		"TG_PLUGIN__MESSAGE_INCLUDE_APP_NAME",
		"TG_PLUGIN__MESSAGE_INCLUDE_TIMESTAMP",
		"TG_PLUGIN__MESSAGE_PARSE_MODE",
		"TG_PLUGIN__MESSAGE_INCLUDE_PRIORITY",
		"TG_PLUGIN__MESSAGE_PRIORITY_THRESHOLD",
	}

	for _, env := range envVars {
		os.Unsetenv(env)
	}

	// Parse environment variables
	cfg, err := ParseEnvVars()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify default values
	assert.Equal(t, "info", cfg.Settings.LogOptions.LogLevel)
	assert.Equal(t, "http://localhost:80", cfg.Settings.GotifyServer.Url.String())
	assert.Equal(t, "", cfg.Settings.GotifyServer.ClientToken)
	assert.Equal(t, 10, cfg.Settings.GotifyServer.Websocket.HandshakeTimeout)
	assert.Equal(t, 30, cfg.Settings.GotifyServer.Websocket.PingInterval)
	assert.Equal(t, 60, cfg.Settings.GotifyServer.Websocket.PongWait)
	assert.Equal(t, "", cfg.Settings.Telegram.DefaultBotToken)
	assert.Empty(t, cfg.Settings.Telegram.DefaultChatIDs)
	assert.False(t, cfg.Settings.Telegram.MessageFormatOptions.IncludeAppName)
	assert.False(t, cfg.Settings.Telegram.MessageFormatOptions.IncludeTimestamp)
	assert.Equal(t, "MarkdownV2", cfg.Settings.Telegram.MessageFormatOptions.ParseMode)
	assert.False(t, cfg.Settings.Telegram.MessageFormatOptions.IncludePriority)
	assert.Equal(t, 0, cfg.Settings.Telegram.MessageFormatOptions.PriorityThreshold)
}

func TestCreateDefaultPluginConfig(t *testing.T) {
	cfg := CreateDefaultPluginConfig()
	assert.NotNil(t, cfg)

	// Test LogOptions defaults
	assert.Equal(t, "info", cfg.Settings.LogOptions.LogLevel)

	// Test GotifyServer defaults
	expectedURL := &url.URL{
		Scheme: "http",
		Host:   "localhost:80",
	}
	assert.Equal(t, expectedURL.String(), cfg.Settings.GotifyServer.Url.String())
	assert.Equal(t, "", cfg.Settings.GotifyServer.ClientToken)

	// Test Websocket defaults
	assert.Equal(t, 10, cfg.Settings.GotifyServer.Websocket.HandshakeTimeout)
	assert.Equal(t, 30, cfg.Settings.GotifyServer.Websocket.PingInterval)
	assert.Equal(t, 60, cfg.Settings.GotifyServer.Websocket.PongWait)

	// Test Telegram defaults
	assert.Equal(t, "", cfg.Settings.Telegram.DefaultBotToken)
	assert.Empty(t, cfg.Settings.Telegram.DefaultChatIDs)
	assert.Empty(t, cfg.Settings.Telegram.Bots)

	// Test MessageFormatOptions defaults
	assert.True(t, cfg.Settings.Telegram.MessageFormatOptions.IncludeAppName)
	assert.False(t, cfg.Settings.Telegram.MessageFormatOptions.IncludeTimestamp)
	assert.Equal(t, "MarkdownV2", cfg.Settings.Telegram.MessageFormatOptions.ParseMode)
	assert.False(t, cfg.Settings.Telegram.MessageFormatOptions.IncludePriority)
	assert.Equal(t, 0, cfg.Settings.Telegram.MessageFormatOptions.PriorityThreshold)

	// Test Rules defaults
	assert.Empty(t, cfg.Settings.Telegram.RoutingRules)
}

func TestLogOptionsStruct_GetZerologLevel(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		want     zerolog.Level
	}{
		{
			name:     "debug level",
			logLevel: "debug",
			want:     zerolog.DebugLevel,
		},
		{
			name:     "info level",
			logLevel: "info",
			want:     zerolog.InfoLevel,
		},
		{
			name:     "warn level",
			logLevel: "warn",
			want:     zerolog.WarnLevel,
		},
		{
			name:     "error level",
			logLevel: "error",
			want:     zerolog.ErrorLevel,
		},
		{
			name:     "uppercase DEBUG",
			logLevel: "DEBUG",
			want:     zerolog.DebugLevel,
		},
		{
			name:     "mixed case DeBuG",
			logLevel: "DeBuG",
			want:     zerolog.DebugLevel,
		},
		{
			name:     "invalid level",
			logLevel: "invalid",
			want:     zerolog.InfoLevel,
		},
		{
			name:     "empty level",
			logLevel: "",
			want:     zerolog.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logOpts := &LogOptions{LogLevel: tt.logLevel}
			got := logOpts.GetZerologLevel()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfig_URLHandling(t *testing.T) {
	tests := []struct {
		name     string
		envURL   string
		wantHost string
		wantPort string
	}{
		{
			name:     "default url",
			envURL:   "http://localhost:80",
			wantHost: "localhost",
			wantPort: "80",
		},
		{
			name:     "custom port",
			envURL:   "http://gotify.example.com:8080",
			wantHost: "gotify.example.com",
			wantPort: "8080",
		},
		{
			name:     "https url",
			envURL:   "https://gotify.secure.com:443",
			wantHost: "gotify.secure.com",
			wantPort: "443",
		},
		{
			name:     "invalid url falls back to default",
			envURL:   "not-a-url",
			wantHost: "localhost",
			wantPort: "80",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear existing env vars
			os.Unsetenv("TG_PLUGIN__GOTIFY_URL")

			// Set test env var
			os.Setenv("TG_PLUGIN__GOTIFY_URL", tt.envURL)
			defer os.Unsetenv("TG_PLUGIN__GOTIFY_URL")

			cfg, err := ParseEnvVars()
			assert.NoError(t, err)
			assert.NotNil(t, cfg)
			assert.NotNil(t, cfg.Settings.GotifyServer.Url)

			parsedURL := cfg.Settings.GotifyServer.Url
			assert.Equal(t, tt.wantHost, parsedURL.Hostname())
			assert.Equal(t, tt.wantPort, parsedURL.Port())
		})
	}
}