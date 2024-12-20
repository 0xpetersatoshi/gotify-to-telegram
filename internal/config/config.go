package config

type GotifyServerConfig struct {
	Hostname    string `yaml:"hostname"`
	Protocol    string `yaml:"protocol"`
	Port        string `yaml:"port"`
	ClientToken string `yaml:"client_token"`
}

type TelegramConfig struct {
	DefaultBotToken string                       `yaml:"default_bot_token"`
	DefaultChatID   string                       `yaml:"default_chat_id"`
	Bots            map[string]TelegramBotConfig `yaml:"bots"`
}

type TelegramBotConfig struct {
	Token  string `yaml:"token"`
	ChatID string `yaml:"chat_id"`
}

type RoutingRule struct {
	AppIDs  []uint32 `yaml:"app_ids"`  // List of Gotify App IDs
	BotName string   `yaml:"bot_name"` // References a bot in the bots config
}

type Config struct {
	TelegramConfig     TelegramConfig     `yaml:"telegram"`
	Rules              []RoutingRule      `yaml:"rules"` // List of routing rules
	GotifyServerConfig GotifyServerConfig `yaml:"gotify_server"`
}
