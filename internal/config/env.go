package config

import (
	"net/url"

	"github.com/caarlos0/env/v11"
)

type Env struct {
	GotifyServerURL   *url.URL `env:"GOTIFY_SERVER_URL" envDefault:"http://localhost:80"`
	GotifyClientToken string   `env:"GOTIFY_CLIENT_TOKEN" envDefault:"replaceme"`
	TelegramBotToken  string   `env:"TELEGRAM_BOT_TOKEN" envDefault:"replaceme"`
	TelegramChatID    string   `env:"TELEGRAM_CHAT_ID" envDefault:"replaceme"`
}

func ParseEnvVars() (*Env, error) {
	cfg := &Env{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func CreateDefaultEnvConfig() *Env {
	return &Env{
		GotifyServerURL:   &url.URL{Scheme: "http", Host: "localhost:80"},
		GotifyClientToken: "replaceme",
		TelegramBotToken:  "replaceme",
		TelegramChatID:    "replaceme",
	}
}
