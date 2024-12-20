package config

import "net/url"

type GotifyServer struct {
	Url         *url.URL `yaml:"url"`
	ClientToken string   `yaml:"client_token"`
}

type Telegram struct {
	DefaultBotToken string                 `yaml:"default_bot_token"`
	DefaultChatID   string                 `yaml:"default_chat_id"`
	Bots            map[string]TelegramBot `yaml:"bots"`
}

type TelegramBot struct {
	Token  string `yaml:"token"`
	ChatID string `yaml:"chat_id"`
}

type RoutingRule struct {
	AppIDs  []uint32 `yaml:"app_ids"`  // List of Gotify App IDs
	BotName string   `yaml:"bot_name"` // References a bot in the bots config
}

type Plugin struct {
	TelegramConfig     Telegram      `yaml:"telegram"`
	Rules              []RoutingRule `yaml:"rules"` // List of routing rules
	GotifyServerConfig GotifyServer  `yaml:"gotify_server"`
}

func CreateDefaultPluginConfig() *Plugin {
	return &Plugin{
		TelegramConfig: Telegram{
			DefaultBotToken: "",
			DefaultChatID:   "",
			Bots:            map[string]TelegramBot{},
		},
		Rules: []RoutingRule{},
		GotifyServerConfig: GotifyServer{
			Url: &url.URL{
				Scheme: "http",
				Host:   "localhost:80",
			},
			ClientToken: "",
		},
	}
}
