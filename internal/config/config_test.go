package config

import (
	"net/url"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestCreateDefaultPluginConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.NotNil(t, cfg)

	// Test LogOptions defaults
	assert.Equal(t, "info", cfg.Settings.LogOptions.LogLevel)

	// Test GotifyServer defaults
	expectedURL := &url.URL{
		Scheme: "http",
		Host:   "localhost:80",
	}

	exampleBot := &TelegramBot{
		Token: "example_token",
		ChatIDs: []string{
			"123456789",
			"987654321",
		},
		AppIDs: []uint32{
			123456789,
			987654321,
		},
		MessageFormatOptions: &MessageFormatOptions{
			IncludeAppName:   false,
			IncludeTimestamp: true,
			ParseMode:        "MarkdownV2",
		},
	}
	expectedBotMap := make(map[string]TelegramBot)
	expectedBotMap["example_bot"] = *exampleBot

	assert.Equal(t, expectedURL.String(), cfg.Settings.GotifyServer.Url.String())
	assert.Equal(t, "", cfg.Settings.GotifyServer.ClientToken)

	// Test Websocket defaults
	assert.Equal(t, 10, cfg.Settings.GotifyServer.Websocket.HandshakeTimeout)

	// Test Telegram defaults
	assert.Equal(t, "", cfg.Settings.Telegram.DefaultBotToken)
	assert.Empty(t, cfg.Settings.Telegram.DefaultChatIDs)
	assert.Equal(t, expectedBotMap, cfg.Settings.Telegram.Bots)

	// Test MessageFormatOptions defaults
	assert.False(t, cfg.Settings.Telegram.MessageFormatOptions.IncludeAppName)
	assert.False(t, cfg.Settings.Telegram.MessageFormatOptions.IncludeTimestamp)
	assert.Equal(t, "MarkdownV2", cfg.Settings.Telegram.MessageFormatOptions.ParseMode)
	assert.False(t, cfg.Settings.Telegram.MessageFormatOptions.IncludePriority)
	assert.Equal(t, 0, cfg.Settings.Telegram.MessageFormatOptions.PriorityThreshold)
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

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    *Plugin
		wantError string
	}{
		{
			name:      "empty config",
			config:    &Plugin{},
			wantError: "settings.telegram.default_bot_token is required",
		},
		{
			name: "missing default chat IDs",
			config: &Plugin{
				Settings: Settings{
					Telegram: Telegram{
						DefaultBotToken: "token",
					},
				},
			},
			wantError: "settings.telegram.default_chat_ids is required",
		},
		{
			name: "missing client token",
			config: &Plugin{
				Settings: Settings{
					Telegram: Telegram{
						DefaultBotToken: "token",
						DefaultChatIDs:  []string{"123"},
					},
					GotifyServer: GotifyServer{
						RawUrl: "http://valid.com",
					},
				},
			},
			wantError: "settings.gotify_server.client_token is required",
		},
		{
			name: "valid config",
			config: &Plugin{
				Settings: Settings{
					Telegram: Telegram{
						DefaultBotToken: "token",
						DefaultChatIDs:  []string{"123"},
					},
					GotifyServer: GotifyServer{
						RawUrl:      "http://valid.com",
						ClientToken: "client-token",
					},
				},
			},
			wantError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError != "" {
				assert.EqualError(t, err, tt.wantError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	// Set up test environment variables
	envVars := map[string]string{
		"TG_PLUGIN__GOTIFY_URL":                 "http://test-server.com",
		"TG_PLUGIN__GOTIFY_CLIENT_TOKEN":        "client_token",
		"TG_PLUGIN__TELEGRAM_DEFAULT_BOT_TOKEN": "test_bot_token",
		"TG_PLUGIN__TELEGRAM_DEFAULT_CHAT_IDS":  "123,456",
		"TG_PLUGIN__MESSAGE_INCLUDE_APP_NAME":   "true",
		"TG_PLUGIN__LOG_LEVEL":                  "debug",
	}

	// Set environment variables
	for k, v := range envVars {
		os.Setenv(k, v)
	}

	// Clean up environment after test
	defer func() {
		for k := range envVars {
			os.Unsetenv(k)
		}
	}()

	cfg := DefaultConfig()
	assert.False(t, cfg.Settings.IgnoreEnvVars)

	// Load config with environment variables
	loadedCfg, err := Load(cfg)

	// Verify results
	assert.NoError(t, err)
	assert.NotNil(t, loadedCfg)

	// Verify that env vars were properly overlaid
	assert.Equal(t, "http://test-server.com", loadedCfg.Settings.GotifyServer.RawUrl)
	assert.Equal(t, "test_bot_token", loadedCfg.Settings.Telegram.DefaultBotToken)
	assert.Equal(t, []string{"123", "456"}, loadedCfg.Settings.Telegram.DefaultChatIDs)
	assert.True(t, loadedCfg.Settings.Telegram.MessageFormatOptions.IncludeAppName)
	assert.Equal(t, "debug", loadedCfg.Settings.LogOptions.LogLevel)

	// Verify URL was properly parsed
	expectedURL, _ := url.Parse("http://test-server.com")
	assert.Equal(t, expectedURL, loadedCfg.Settings.GotifyServer.Url)
}
