package config

import (
	"errors"
	"net/url"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/rs/zerolog"
)

const DefaultURL = "http://localhost:80"

// Settings represents global plugin settings
type Settings struct {
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
	LogLevel string `yaml:"log_level" env:"TG_PLUGIN__LOG_LEVEL" envDefault:"info"`
}

// Message formatting options
type MessageFormatOptions struct {
	// Whether to include app name in message
	IncludeAppName bool `yaml:"include_app_name" env:"TG_PLUGIN__MESSAGE_INCLUDE_APP_NAME" envDefault:"false"`
	// Whether to include timestamp in message
	IncludeTimestamp bool `yaml:"include_timestamp" env:"TG_PLUGIN__MESSAGE_INCLUDE_TIMESTAMP" envDefault:"false"`
	// Telegram parse mode (Markdown, MarkdownV2, HTML)
	ParseMode string `yaml:"parse_mode" env:"TG_PLUGIN__MESSAGE_PARSE_MODE" envDefault:"MarkdownV2"`
	// Whether to include the message priority in the message
	IncludePriority bool `yaml:"include_priority" env:"TG_PLUGIN__MESSAGE_INCLUDE_PRIORITY" envDefault:"false"`
	// Whether to include the message priority above a certain level
	PriorityThreshold int `yaml:"priority_threshold" env:"TG_PLUGIN__MESSAGE_PRIORITY_THRESHOLD" envDefault:"0"`
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
	// Gotify server in url.URL format
	Url *url.URL `yaml:"-"`
	// Gotify server URL
	RawUrl string `yaml:"url" env:"TG_PLUGIN__GOTIFY_URL" envDefault:"http://localhost:80"`
	// Gotify client token
	ClientToken string `yaml:"client_token" env:"TG_PLUGIN__GOTIFY_CLIENT_TOKEN" envDefault:""`
	// Websocket settings
	Websocket Websocket `yaml:"websocket"`
}

// parseURL parses the Gotify server URL
func (g *GotifyServer) parseURL() error {
	_, err := url.Parse(g.RawUrl)
	if err != nil {
		return err
	}
	return nil
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
	// Gotify app ids to telegram bot routing rules
	RoutingRules []RoutingRule `yaml:"routing_rules"`
	// Message formatting options
	MessageFormatOptions MessageFormatOptions `yaml:"message_format_options"`
}

// TelegramBot settings
type TelegramBot struct {
	// Bot token
	Token string `yaml:"token" env:"TG_PLUGIN__TELEGRAM_BOT_TOKEN" envDefault:""`
	// Chat IDs
	ChatIDs []string `yaml:"chat_ids" env:"TG_PLUGIN__TELEGRAM_CHAT_IDS" envDefault:""`
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

// Validate validates that required fields are set and valid
func (p *Plugin) Validate() error {
	if p.Settings.Telegram.DefaultBotToken == "" {
		return errors.New("settings.telegram.default_bot_token is required")
	}

	if len(p.Settings.Telegram.DefaultChatIDs) == 0 {
		return errors.New("settings.telegram.default_chat_ids is required")
	}

	p.Settings.GotifyServer.Url, _ = url.Parse(p.Settings.GotifyServer.RawUrl)

	if p.Settings.GotifyServer.Url == nil || p.Settings.GotifyServer.Url.Hostname() == "" {
		return errors.New("settings.gotify_server.url is required")
	}

	if p.Settings.GotifyServer.ClientToken == "" {
		return errors.New("settings.gotify_server.client_token is required")
	}

	return nil
}

func CreateDefaultPluginConfig() *Plugin {
	URL, _ := url.Parse(DefaultURL)
	telegram := Telegram{
		DefaultBotToken: "",
		DefaultChatIDs:  []string{},
		Bots:            map[string]TelegramBot{},
		RoutingRules:    []RoutingRule{},
		MessageFormatOptions: MessageFormatOptions{
			IncludeAppName:   true,
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
			PingInterval:     30,
			PongWait:         60,
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

func ParseEnvVars() (*Plugin, error) {
	cfg := &Plugin{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	cfg.Settings.GotifyServer.Url = cfg.Settings.GotifyServer.URL()

	// Handle invalid URL by setting default
	if cfg.Settings.GotifyServer.Url.Hostname() == "" {
		defaultURL, _ := url.Parse(DefaultURL)
		cfg.Settings.GotifyServer.Url = defaultURL
	}

	return cfg, nil
}
