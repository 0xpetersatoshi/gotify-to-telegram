package config

import (
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test struct to avoid using the full Plugin config
type TestConfig struct {
	StringField     string   `env:"TEST_STRING"`
	BoolField       bool     `env:"TEST_BOOL"`
	IntField        int      `env:"TEST_INT"`
	UintField       uint     `env:"TEST_UINT"`
	SliceField      []string `env:"TEST_SLICE"`
	UntaggedField   string
	unexportedField string
}

func TestGetEnvName(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expected string
	}{
		{
			name:     "should correctly get env name for tagged field",
			field:    "StringField",
			expected: "TEST_STRING",
		},
		{
			name:     "should not get env name for untagged field",
			field:    "UntaggedField",
			expected: "",
		},
	}

	typ := reflect.TypeOf(TestConfig{})
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			field, _ := typ.FieldByName(tc.field)
			envName := getEnvName(field)
			assert.Equal(t, tc.expected, envName)
		})
	}
}

func TestSetFieldFromEnv(t *testing.T) {
	tests := []struct {
		name      string
		envName   string
		envValue  string
		field     string
		expected  interface{}
		shouldSet bool
	}{
		{
			name:      "should correctly set string field",
			envName:   "TEST_STRING",
			envValue:  "test value",
			field:     "StringField",
			expected:  "test value",
			shouldSet: true,
		},
		{
			name:      "should correctly set bool field true",
			envName:   "TEST_BOOL",
			envValue:  "true",
			field:     "BoolField",
			expected:  true,
			shouldSet: true,
		},
		{
			name:      "should correctly set bool field false",
			envName:   "TEST_BOOL",
			envValue:  "false",
			field:     "BoolField",
			expected:  false,
			shouldSet: true,
		},
		{
			name:      "sould correctly set int field",
			envName:   "TEST_INT",
			envValue:  "42",
			field:     "IntField",
			expected:  int(42),
			shouldSet: true,
		},
		{
			name:      "should correctly set uint field",
			envName:   "TEST_UINT",
			envValue:  "42",
			field:     "UintField",
			expected:  uint(42),
			shouldSet: true,
		},
		{
			name:      "should correctly set slice field",
			envName:   "TEST_SLICE",
			envValue:  "a,b,c",
			field:     "SliceField",
			expected:  []string{"a", "b", "c"},
			shouldSet: true,
		},
		{
			name:      "should set field to false if invalid bool value",
			envName:   "TEST_BOOL",
			envValue:  "not a bool",
			field:     "BoolField",
			expected:  false,
			shouldSet: false,
		},
		{
			name:      "should set field to 0 if invalid int value",
			envName:   "TEST_INT",
			envValue:  "not an int",
			field:     "IntField",
			expected:  0,
			shouldSet: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up environment
			os.Setenv(tc.envName, tc.envValue)
			defer os.Unsetenv(tc.envName)

			testStruct := TestConfig{}
			val := reflect.ValueOf(&testStruct).Elem()
			field := val.FieldByName(tc.field)

			setFieldFromEnv(field, tc.envName)

			// Check result
			switch field.Kind() {
			case reflect.String:
				assert.Equal(t, tc.expected, field.String())
			case reflect.Bool:
				assert.Equal(t, tc.expected, field.Bool())
			case reflect.Int:
				assert.Equal(t, tc.expected, int(field.Int()))
			case reflect.Uint:
				assert.Equal(t, tc.expected, uint(field.Uint()))
			case reflect.Slice:
				assert.Equal(t, tc.expected, field.Interface())
			}
		})
	}
}

func TestProcessStruct(t *testing.T) {
	// Set up test environment
	os.Setenv("TEST_STRING", "test value")
	os.Setenv("TEST_BOOL", "true")
	os.Setenv("TEST_INT", "42")
	os.Setenv("TEST_UINT", "43")
	os.Setenv("TEST_SLICE", "x,y,z")
	defer func() {
		os.Unsetenv("TEST_STRING")
		os.Unsetenv("TEST_BOOL")
		os.Unsetenv("TEST_INT")
		os.Unsetenv("TEST_UINT")
		os.Unsetenv("TEST_SLICE")
	}()

	testStruct := TestConfig{
		UntaggedField:   "should not change",
		unexportedField: "should not change",
	}

	val := reflect.ValueOf(&testStruct).Elem()
	processStruct(val)

	assert.Equal(t, "test value", testStruct.StringField)
	assert.Equal(t, true, testStruct.BoolField)
	assert.Equal(t, 42, testStruct.IntField)
	assert.Equal(t, uint(43), testStruct.UintField)
	assert.Equal(t, []string{"x", "y", "z"}, testStruct.SliceField)
	assert.Equal(t, "should not change", testStruct.UntaggedField)
	assert.Equal(t, "should not change", testStruct.unexportedField)
}

func TestOverlayEnvVars(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		setup   func() *Plugin
		verify  func(*testing.T, *Plugin, error)
	}{
		{
			name: "should set valid URL",
			envVars: map[string]string{
				"TG_PLUGIN__GOTIFY_URL": "http://example.com",
			},
			setup: func() *Plugin {
				return &Plugin{
					Settings: Settings{
						GotifyServer: GotifyServer{
							RawUrl: "http://original.com",
						},
					},
				}
			},
			verify: func(t *testing.T, p *Plugin, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "http://example.com", p.Settings.GotifyServer.RawUrl)
				expectedURL, _ := url.Parse("http://example.com")
				assert.Equal(t, expectedURL, p.Settings.GotifyServer.Url)
			},
		},
		{
			name: "should not set invalid URL",
			envVars: map[string]string{
				"TG_PLUGIN__GOTIFY_URL": "://invalid",
			},
			setup: func() *Plugin {
				return &Plugin{
					Settings: Settings{
						GotifyServer: GotifyServer{
							RawUrl: "http://original.com",
						},
					},
				}
			},
			verify: func(t *testing.T, p *Plugin, err error) {
				assert.Error(t, err)
				assert.Equal(t, "http://original.com", p.Settings.GotifyServer.RawUrl)
			},
		},
		{
			name: "should correctly overlay telegram env vars",
			envVars: map[string]string{
				"TG_PLUGIN__TELEGRAM_DEFAULT_BOT_TOKEN": "new_bot_token",
				"TG_PLUGIN__TELEGRAM_DEFAULT_CHAT_IDS":  "111,222",
			},
			setup: func() *Plugin {
				return &Plugin{
					Settings: Settings{
						Telegram: Telegram{
							DefaultBotToken: "old_bot_token",
							DefaultChatIDs:  []string{"123", "456"},
						},
					},
				}
			},
			verify: func(t *testing.T, p *Plugin, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "new_bot_token", p.Settings.Telegram.DefaultBotToken)
				assert.Equal(t, []string{"111", "222"}, p.Settings.Telegram.DefaultChatIDs)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up environment
			for k, v := range tc.envVars {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tc.envVars {
					os.Unsetenv(k)
				}
			}()

			// Run test
			cfg := tc.setup()
			err := overlayEnvVars(cfg)
			tc.verify(t, cfg, err)
		})
	}
}
