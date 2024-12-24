package main

import (
	"testing"

	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/config"
	"github.com/gotify/plugin-api"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock structs
type MockAPI struct {
	mock.Mock
}

type MockTelegram struct {
	mock.Mock
}

func TestAPICompatibility(t *testing.T) {
	assert.Implements(t, (*plugin.Plugin)(nil), new(Plugin))
	// Add other interfaces you intend to implement here
}

func TestPluginStruct_DefaultConfig(t *testing.T) {
	tests := []struct {
		name          string
		envVars       map[string]string
		expectError   bool
		validateCalls int
	}{
		{
			name:          "successful default config",
			envVars:       map[string]string{},
			expectError:   false,
			validateCalls: 1,
		},
		{
			name: "valid env vars override",
			envVars: map[string]string{
				"TG_PLUGIN__LOG_LEVEL":                  "debug",
				"TG_PLUGIN__GOTIFY_URL":                 "http://example.com",
				"TG_PLUGIN__GOTIFY_CLIENT_TOKEN":        "token123",
				"TG_PLUGIN__TELEGRAM_DEFAULT_BOT_TOKEN": "bot123",
				"TG_PLUGIN__TELEGRAM_DEFAULT_CHAT_IDS":  "chat1,chat2",
			},
			expectError:   false,
			validateCalls: 1,
		},
		{
			name: "invalid config after env vars",
			envVars: map[string]string{
				"TG_PLUGIN__GOTIFY_URL": "invalid://url",
			},
			expectError:   true,
			validateCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			for k := range tt.envVars {
				t.Setenv(k, "")
			}

			// Set test environment
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			// Create plugin instance
			logger := zerolog.New(zerolog.NewTestWriter(t))
			p := &Plugin{
				logger: &logger,
			}

			// Get default config
			cfg := p.DefaultConfig()

			// Verify the config
			assert.NotNil(t, cfg)
			assert.IsType(t, &config.Plugin{}, cfg)

			// Verify specific values based on environment
			pluginCfg := cfg.(*config.Plugin)
			if len(tt.envVars) > 0 {
				// Check if environment variables were applied
				if v, ok := tt.envVars["TG_PLUGIN__LOG_LEVEL"]; ok {
					assert.Equal(t, v, pluginCfg.Settings.LogOptions.LogLevel)
				}
			}
		})
	}
}

func TestPluginStruct_ValidateAndSetConfig(t *testing.T) {
	tests := []struct {
		name          string
		userConfig    *config.Plugin
		wantConfig    *config.Plugin
		envVars       map[string]string
		wantError     bool
		validateCalls int
	}{
		{
			name: "should create a config where the env vars take precedence",
			userConfig: &config.Plugin{
				Settings: config.Settings{
					LogOptions: config.LogOptions{
						LogLevel: "info",
					},
					Telegram: config.Telegram{
						DefaultBotToken: "user-provided-token",
						DefaultChatIDs:  []string{"chat123", "chat456"},
						Bots: map[string]config.TelegramBot{
							"bot1": {
								Token:   "bot1-token",
								ChatIDs: []string{"chat123", "chat456"},
								AppIDs:  []uint32{1, 2},
							},
							"bot2": {
								Token:   "bot2-token",
								ChatIDs: []string{"chat789"},
								AppIDs:  []uint32{3, 4},
							},
						},
					},
					GotifyServer: config.GotifyServer{
						RawUrl:      "http://mydomain.com",
						ClientToken: "token123",
					},
				},
			},
			wantConfig: &config.Plugin{
				Settings: config.Settings{
					LogOptions: config.LogOptions{
						LogLevel: "debug",
					},
					Telegram: config.Telegram{
						DefaultBotToken: "bot123",
						DefaultChatIDs:  []string{"chat1", "chat2"},
						Bots: map[string]config.TelegramBot{
							"bot1": {
								Token:   "bot1-token",
								ChatIDs: []string{"chat123", "chat456"},
								AppIDs:  []uint32{1, 2},
							},
							"bot2": {
								Token:   "bot2-token",
								ChatIDs: []string{"chat789"},
								AppIDs:  []uint32{3, 4},
							},
						},
						MessageFormatOptions: config.MessageFormatOptions{
							IncludeAppName:   false,
							IncludeTimestamp: false,
							ParseMode:        "MarkdownV2",
						},
					},
					GotifyServer: config.GotifyServer{
						RawUrl:      "http://example.com",
						ClientToken: "token123",
					},
				},
			},
			envVars: map[string]string{
				"TG_PLUGIN__LOG_LEVEL":                  "debug",
				"TG_PLUGIN__GOTIFY_URL":                 "http://example.com",
				"TG_PLUGIN__GOTIFY_CLIENT_TOKEN":        "token123",
				"TG_PLUGIN__TELEGRAM_DEFAULT_BOT_TOKEN": "bot123",
				"TG_PLUGIN__TELEGRAM_DEFAULT_CHAT_IDS":  "chat1,chat2",
			},
			wantError:     false,
			validateCalls: 1,
		},
		{
			name: "should create a config where the env vars are ignored",
			userConfig: &config.Plugin{
				Settings: config.Settings{
					IgnoreEnvVars: true,
					LogOptions: config.LogOptions{
						LogLevel: "info",
					},
					Telegram: config.Telegram{
						DefaultBotToken: "user-provided-token",
						DefaultChatIDs:  []string{"chat123", "chat456"},
						Bots: map[string]config.TelegramBot{
							"bot1": {
								Token:   "bot1-token",
								ChatIDs: []string{"chat123", "chat456"},
								AppIDs:  []uint32{1, 2},
							},
							"bot2": {
								Token:   "bot2-token",
								ChatIDs: []string{"chat789"},
								AppIDs:  []uint32{3, 4},
							},
						},
					},
					GotifyServer: config.GotifyServer{
						RawUrl:      "http://mydomain.com",
						ClientToken: "token123",
					},
				},
			},
			wantConfig: &config.Plugin{
				Settings: config.Settings{
					IgnoreEnvVars: true,
					LogOptions: config.LogOptions{
						LogLevel: "info",
					},
					Telegram: config.Telegram{
						DefaultBotToken: "user-provided-token",
						DefaultChatIDs:  []string{"chat123", "chat456"},
						Bots: map[string]config.TelegramBot{
							"bot1": {
								Token:   "bot1-token",
								ChatIDs: []string{"chat123", "chat456"},
								AppIDs:  []uint32{1, 2},
							},
							"bot2": {
								Token:   "bot2-token",
								ChatIDs: []string{"chat789"},
								AppIDs:  []uint32{3, 4},
							},
						},
					},
					GotifyServer: config.GotifyServer{
						RawUrl:      "http://mydomain.com",
						ClientToken: "token123",
					},
				},
			},
			envVars: map[string]string{
				"TG_PLUGIN__LOG_LEVEL":                  "debug",
				"TG_PLUGIN__GOTIFY_URL":                 "http://example.com",
				"TG_PLUGIN__GOTIFY_CLIENT_TOKEN":        "token123",
				"TG_PLUGIN__TELEGRAM_DEFAULT_BOT_TOKEN": "bot123",
				"TG_PLUGIN__TELEGRAM_DEFAULT_CHAT_IDS":  "chat1,chat2",
			},
			wantError:     false,
			validateCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			for k := range tt.envVars {
				t.Setenv(k, "")
			}

			// Set test environment
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			// Create plugin instance
			logger := zerolog.New(zerolog.NewTestWriter(t))
			p := &Plugin{
				logger:  &logger,
				enabled: false,
			}

			// Call ValidateAndSetConfig
			err := p.ValidateAndSetConfig(tt.userConfig)

			p.logger.Debug().Interface("config", p.config).Msg("config")
			tt.wantConfig.Settings.GotifyServer.Url = tt.wantConfig.Settings.GotifyServer.URL()

			// Verify the result
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, p.config)
				assert.Equal(t, p.config, tt.wantConfig)
			}
		})
	}
}
