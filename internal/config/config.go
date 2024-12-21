package config

import (
	"net/url"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/rs/zerolog"
)

// Settings represents global plugin settings
type Settings struct {
	// LogLevel can be "debug", "info", "warn", "error"
	LogLevel string `yaml:"log_level" env:"TG_PLUGIN__LOG_LEVEL" envDefault:"info"`
	// Gotify server settings
	GotifyServer GotifyServer `yaml:"gotify_server"`
	// Telegram settings
	Telegram Telegram `yaml:"telegram"`
	// Routing rules for app IDs to Telegram bots
	Rules []RoutingRule `yaml:"rules"`
}

// Message formatting options
type MessageFormat struct {
	// Whether to include app name in message
	IncludeAppName bool `yaml:"include_app_name" env:"TG_PLUGIN__MESSAGE_INCLUDE_APP_NAME" envDefault:"true"`
	// Whether to include timestamp in message
	IncludeTimestamp bool `yaml:"include_timestamp" env:"TG_PLUGIN__MESSAGE_INCLUDE_TIMESTAMP" envDefault:"false"`
	// Telegram parse mode (Markdown, MarkdownV2, HTML)
	ParseMode string `yaml:"parse_mode" env:"TG_PLUGIN__MESSAGE_PARSE_MODE" envDefault:"MarkdownV2"`
}

// Websocket settings
type Websocket struct {
	// Timeout for initial connection (in seconds)
	HandshakeTimeout int `yaml:"handshake_timeout" env:"TG_PLUGIN__WS_HANDSHAKE_TIMEOUT" envDefault:"10"`
	// Time between ping/pong messages (in seconds)
	PingInterval int `yaml:"ping_interval" env:"TG_PLUGIN__WS_PING_INTERVAL" envDefault:"30"`
	// Time to wait for pong response (in seconds)
	PongWait int `yaml:"pong_wait" env:"TG_PLUGIN__WS_PONG_WAIT" envDefault:"60"`
}

// GotifyServer settings
type GotifyServer struct {
	// Gotify server URL
	Url *url.URL `yaml:"url" env:"TG_PLUGIN__GOTIFY_URL" envDefault:"http://localhost:80"`
	// Gotify client token
	ClientToken string `yaml:"client_token" env:"TG_PLUGIN__GOTIFY_CLIENT_TOKEN" envDefault:""`
	// Websocket settings
	Websocket Websocket `yaml:"websocket"`
}

// Telegram settings
type Telegram struct {
	// Default bot token
	DefaultBotToken string `yaml:"default_bot_token" env:"TG_PLUGIN__TELEGRAM_DEFAULT_BOT_TOKEN" envDefault:""`
	// Default chat ID
	DefaultChatID string `yaml:"default_chat_id" env:"TG_PLUGIN__TELEGRAM_DEFAULT_CHAT_ID" envDefault:""`
	// Mapping of bot names to bot tokens/chat IDs
	Bots map[string]TelegramBot `yaml:"bots"`
	// Message formatting options
	MessageFormat MessageFormat `yaml:"message_format"`
}

// TelegramBot settings
type TelegramBot struct {
	// Bot token
	Token string `yaml:"token" env:"TG_PLUGIN__TELEGRAM_BOT_TOKEN" envDefault:""`
	// Chat ID
	ChatID string `yaml:"chat_id" env:"TG_PLUGIN__TELEGRAM_CHAT_ID" envDefault:""`
}

// Telegram routing rule
type RoutingRule struct {
	// List of Gotify App IDs
	AppIDs []uint32 `yaml:"app_ids"`
	// Telegram bot name
	BotName string `yaml:"bot_name"` // References a bot in the bots config
}

// Plugin settings
type Plugin struct {
	Settings Settings `yaml:"settings"`
}

func CreateDefaultPluginConfig() *Plugin {
	telegram := Telegram{
		DefaultBotToken: "",
		DefaultChatID:   "",
		Bots:            map[string]TelegramBot{},
		MessageFormat: MessageFormat{
			IncludeAppName:   true,
			IncludeTimestamp: false,
			ParseMode:        "MarkdownV2",
		},
	}

	gotifyServer := GotifyServer{
		Url: &url.URL{
			Scheme: "http",
			Host:   "localhost:80",
		},
		ClientToken: "",
		Websocket: Websocket{
			HandshakeTimeout: 10,
			PingInterval:     30,
			PongWait:         60,
		},
	}

	settings := Settings{
		LogLevel:     "info",
		Telegram:     telegram,
		Rules:        []RoutingRule{},
		GotifyServer: gotifyServer,
	}
	return &Plugin{
		Settings: settings,
	}
}

// GetZerologLevel converts string log level to zerolog level
func (s *Settings) GetZerologLevel() zerolog.Level {
	switch strings.ToLower(s.LogLevel) {
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

func ParseEnvVars() (*Plugin, error) {
	cfg := &Plugin{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
