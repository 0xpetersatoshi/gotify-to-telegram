package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/utils"
	"github.com/rs/zerolog"
)

const DefaultURL = "http://localhost:80"

// Settings represents global plugin settings
type Settings struct {
	// Ignores env variables when true
	IgnoreEnvVars bool `yaml:"ignore_env_vars"`
	// Log options
	LogOptions LogOptions `yaml:"log_options"`
	// Gotify server settings
	GotifyServer GotifyServer `yaml:"gotify_server"`
	// Telegram settings
	Telegram Telegram `yaml:"telegram"`
}

// Log options
type LogOptions struct {
	// LogLevel can be "debug", "info", "warn", "error"
	LogLevel string `yaml:"log_level" env:"TG_PLUGIN__LOG_LEVEL"`
}

// Message formatting options
type MessageFormatOptions struct {
	// Whether to include app name in message
	IncludeAppName bool `yaml:"include_app_name" env:"TG_PLUGIN__MESSAGE_INCLUDE_APP_NAME"`
	// Whether to include timestamp in message
	IncludeTimestamp bool `yaml:"include_timestamp" env:"TG_PLUGIN__MESSAGE_INCLUDE_TIMESTAMP"`
	// Whether to include message extras in message
	IncludeExtras bool `yaml:"include_extras" env:"TG_PLUGIN__MESSAGE_INCLUDE_EXTRAS"`
	// Telegram parse mode (Markdown, MarkdownV2, HTML)
	ParseMode string `yaml:"parse_mode" env:"TG_PLUGIN__MESSAGE_PARSE_MODE"`
	// Whether to include the message priority in the message
	IncludePriority bool `yaml:"include_priority" env:"TG_PLUGIN__MESSAGE_INCLUDE_PRIORITY"`
	// Whether to include the message priority above a certain level
	PriorityThreshold int `yaml:"priority_threshold" env:"TG_PLUGIN__MESSAGE_PRIORITY_THRESHOLD"`
}

// Websocket settings
type Websocket struct {
	// Timeout for initial connection (in seconds)
	HandshakeTimeout int `yaml:"handshake_timeout" env:"TG_PLUGIN__WS_HANDSHAKE_TIMEOUT"`
}

// GotifyServer settings
type GotifyServer struct {
	// Gotify server in url.URL format
	Url *url.URL `yaml:"-"`
	// Gotify server URL
	RawUrl string `yaml:"url" env:"TG_PLUGIN__GOTIFY_URL" envDefault:"http://localhost:80"`
	// Gotify client token
	ClientToken string `yaml:"client_token" env:"TG_PLUGIN__GOTIFY_CLIENT_TOKEN" envDefault:""`
	// Websocket settings
	Websocket Websocket `yaml:"websocket"`
}

// Url returns the parsed Gotify server URL
func (g *GotifyServer) URL() *url.URL {
	if g.Url == nil {
		if parsedURL, err := url.Parse(g.RawUrl); err == nil {
			g.Url = parsedURL
		} else {
			// Fallback to default if parsing fails
			defaultURL, _ := url.Parse(DefaultURL)
			g.Url = defaultURL
		}
	}
	return g.Url
}

// Telegram settings
type Telegram struct {
	// Default bot token
	DefaultBotToken string `yaml:"default_bot_token" env:"TG_PLUGIN__TELEGRAM_DEFAULT_BOT_TOKEN" envDefault:""`
	// Default chat ID
	DefaultChatIDs []string `yaml:"default_chat_ids" env:"TG_PLUGIN__TELEGRAM_DEFAULT_CHAT_IDS" envDefault:""`
	// Mapping of bot names to bot tokens/chat IDs
	Bots map[string]TelegramBot `yaml:"bots"`
	// Message formatting options
	MessageFormatOptions MessageFormatOptions `yaml:"default_message_format_options"`
}

// TelegramBot settings
type TelegramBot struct {
	// Bot token
	Token string `yaml:"token"`
	// Chat IDs
	ChatIDs []string `yaml:"chat_ids"`
	// Gotify app ids
	AppIDs []uint32 `yaml:"gotify_app_ids"`
	// Bot message formatting options
	MessageFormatOptions *MessageFormatOptions `yaml:"message_format_options"`
}

// Plugin settings
type Plugin struct {
	Settings Settings `yaml:"settings"`
}

// Validate validates that required fields are set and valid
func (p *Plugin) Validate() error {
	if p.Settings.Telegram.DefaultBotToken == "" {
		return errors.New("settings.telegram.default_bot_token is required")
	}

	if len(p.Settings.Telegram.DefaultChatIDs) == 0 {
		return errors.New("settings.telegram.default_chat_ids is required")
	}

	if p.Settings.GotifyServer.RawUrl == "" {
		return errors.New("settings.gotify_server.url is required")
	}

	parsedURL, err := url.Parse(p.Settings.GotifyServer.RawUrl)
	if err != nil {
		return err
	}

	p.Settings.GotifyServer.Url = parsedURL

	if p.Settings.GotifyServer.Url == nil || p.Settings.GotifyServer.Url.Hostname() == "" {
		return errors.New("settings.gotify_server.url is invalid. Should be in format http://localhost:80 or http://example.com")
	}

	if p.Settings.GotifyServer.ClientToken == "" {
		return errors.New("settings.gotify_server.client_token is required")
	}

	return nil
}

// SafeString returns a string representation of the plugin configuration
// with sensitive data masked
func (p *Plugin) SafeString() string {
	// Create a deep copy of the config to avoid modifying the original
	configCopy := *p

	// Mask Gotify client token
	configCopy.Settings.GotifyServer.ClientToken = utils.MaskToken(configCopy.Settings.GotifyServer.ClientToken)

	// Mask default Telegram bot token
	configCopy.Settings.Telegram.DefaultBotToken = utils.MaskToken(configCopy.Settings.Telegram.DefaultBotToken)

	// Mask tokens for all configured bots
	for botName, bot := range configCopy.Settings.Telegram.Bots {
		botCopy := bot
		botCopy.Token = utils.MaskToken(bot.Token)
		configCopy.Settings.Telegram.Bots[botName] = botCopy
	}

	// Marshal the masked config to JSON
	jsonBytes, err := json.MarshalIndent(configCopy, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error marshaling config: %v", err)
	}

	return string(jsonBytes)
}

func DefaultConfig() *Plugin {
	URL, _ := url.Parse(DefaultURL)
	bot := TelegramBot{
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

	botMap := make(map[string]TelegramBot)
	botMap["example_bot"] = bot

	telegram := Telegram{
		DefaultBotToken: "",
		DefaultChatIDs:  []string{},
		Bots:            botMap,
		MessageFormatOptions: MessageFormatOptions{
			IncludeAppName:   false,
			IncludeTimestamp: false,
			ParseMode:        "MarkdownV2",
		},
	}

	gotifyServer := GotifyServer{
		Url:         URL,
		RawUrl:      DefaultURL,
		ClientToken: "",
		Websocket: Websocket{
			HandshakeTimeout: 10,
		},
	}

	settings := Settings{
		LogOptions:   LogOptions{LogLevel: "info"},
		Telegram:     telegram,
		GotifyServer: gotifyServer,
	}
	return &Plugin{
		Settings: settings,
	}
}

// GetZerologLevel converts string log level to zerolog level
func (l *LogOptions) GetZerologLevel() zerolog.Level {
	switch strings.ToLower(l.LogLevel) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// Load loads the yaml (and optionally) the environment variables into the plugin config
func Load(newCfg *Plugin) (*Plugin, error) {
	// Optionally load config from env vars
	if !newCfg.Settings.IgnoreEnvVars {
		if err := overlayEnvVars(newCfg); err != nil {
			return nil, err
		}
	}

	if err := newCfg.Validate(); err != nil {
		return nil, err
	}

	return newCfg, nil
}
