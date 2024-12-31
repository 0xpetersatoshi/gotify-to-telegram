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

func TestPluginStruct_ValidateAndSetConfig(t *testing.T) {
	tests := []struct {
		name       string
		userConfig interface{}
		envVars    map[string]string
		verify     func(*testing.T, *Plugin, error)
	}{
		{
			name: "should validate and set valid config with env vars",
			userConfig: &config.Plugin{
				Settings: config.Settings{
					LogOptions: config.LogOptions{
						LogLevel: "info",
					},
					Telegram: config.Telegram{
						DefaultBotToken: "user-token",
						DefaultChatIDs:  []string{"123", "456"},
					},
					GotifyServer: config.GotifyServer{
						RawUrl:      "http://example.com",
						ClientToken: "client-token",
					},
				},
			},
			envVars: map[string]string{
				"TG_PLUGIN__TELEGRAM_DEFAULT_BOT_TOKEN": "env-token",
				"TG_PLUGIN__TELEGRAM_DEFAULT_CHAT_IDS":  "111,222",
			},
			verify: func(t *testing.T, p *Plugin, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "env-token", p.config.Settings.Telegram.DefaultBotToken)
				assert.Equal(t, []string{"111", "222"}, p.config.Settings.Telegram.DefaultChatIDs)
			},
		},
		{
			name:       "should fail to validate and set invalid config type",
			userConfig: &struct{}{},
			verify: func(t *testing.T, p *Plugin, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid config type")
			},
		},
		{
			name: "should fail to validate and set invalid config - missing required fields",
			userConfig: &config.Plugin{
				Settings: config.Settings{
					LogOptions: config.LogOptions{
						LogLevel: "info",
					},
				},
			},
			verify: func(t *testing.T, p *Plugin, err error) {
				assert.Error(t, err)
			},
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

			// Verify results
			tt.verify(t, p, err)
		})
	}
}

func TestPlugin_Configure(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.Plugin
		envVars   map[string]string
		wantError bool
		setup     func(*Plugin)
		verify    func(*testing.T, *Plugin)
	}{
		{
			name: "should configure valid config without env vars",
			config: &config.Plugin{
				Settings: config.Settings{
					IgnoreEnvVars: true,
					LogOptions: config.LogOptions{
						LogLevel: "info",
					},
					Telegram: config.Telegram{
						DefaultBotToken: "test-token",
						DefaultChatIDs:  []string{"123", "456"},
					},
					GotifyServer: config.GotifyServer{
						RawUrl:      "http://example.com",
						ClientToken: "client-token",
					},
				},
			},
			wantError: false,
			verify: func(t *testing.T, p *Plugin) {
				assert.Equal(t, "test-token", p.config.Settings.Telegram.DefaultBotToken)
				assert.Equal(t, []string{"123", "456"}, p.config.Settings.Telegram.DefaultChatIDs)
				assert.Equal(t, "http://example.com", p.config.Settings.GotifyServer.RawUrl)
			},
		},
		{
			name: "should configure valid config with env vars",
			config: &config.Plugin{
				Settings: config.Settings{
					LogOptions: config.LogOptions{
						LogLevel: "info",
					},
					Telegram: config.Telegram{
						DefaultBotToken: "original-token",
						DefaultChatIDs:  []string{"original"},
					},
					GotifyServer: config.GotifyServer{
						RawUrl:      "http://original.com",
						ClientToken: "original-client-token",
					},
				},
			},
			envVars: map[string]string{
				"TG_PLUGIN__TELEGRAM_DEFAULT_BOT_TOKEN": "env-token",
				"TG_PLUGIN__TELEGRAM_DEFAULT_CHAT_IDS":  "111,222",
				"TG_PLUGIN__GOTIFY_URL":                 "http://env.com",
			},
			wantError: false,
			verify: func(t *testing.T, p *Plugin) {
				assert.Equal(t, "env-token", p.config.Settings.Telegram.DefaultBotToken)
				assert.Equal(t, []string{"111", "222"}, p.config.Settings.Telegram.DefaultChatIDs)
				assert.Equal(t, "http://env.com", p.config.Settings.GotifyServer.RawUrl)
			},
		},
		{
			name: "should error for invalid config - missing required fields",
			config: &config.Plugin{
				Settings: config.Settings{
					IgnoreEnvVars: true,
					LogOptions: config.LogOptions{
						LogLevel: "info",
					},
				},
			},
			wantError: true,
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

			if tt.setup != nil {
				tt.setup(p)
			}

			err := p.Configure(tt.config)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, p.config)
				if tt.verify != nil {
					tt.verify(t, p)
				}
			}
		})
	}
}
