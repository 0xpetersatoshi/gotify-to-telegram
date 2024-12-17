package main

import (
	"os"

	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/api"
	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/telegram"
	"github.com/gin-gonic/gin"
	"github.com/gotify/plugin-api"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// GetGotifyPluginInfo returns gotify plugin info.
func GetGotifyPluginInfo() plugin.Info {
	return plugin.Info{
		ModulePath:  "github.com/0xPeterSatoshi/gotify-to-telegram",
		Version:     "1.0.0",
		Author:      "0xPeterSatoshi",
		Website:     "https://gotify.net/docs/plugin",
		Description: "Send gotify notifications to telegram",
		License:     "MIT",
		Name:        "gotify-to-telegram",
	}
}

// Plugin is the gotify plugin instance.
type Plugin struct {
	logger    *zerolog.Logger
	apiclient *api.Client
	tgclient  *telegram.Client
}

// Enable enables the plugin.
func (p *Plugin) Enable() error {
	return nil
}

// Disable disables the plugin.
func (p *Plugin) Disable() error {
	return nil
}

// RegisterWebhook implements plugin.Webhooker.
func (p *Plugin) RegisterWebhook(basePath string, g *gin.RouterGroup) {
}

// Start starts the plugin.
func (p *Plugin) Start() error {
	p.logger.Debug().Msg("starting plugin")
	p.apiclient.ListenForMessages()

	return nil
}

// NewGotifyPluginInstance creates a plugin instance for a user context.
func NewGotifyPluginInstance(ctx plugin.UserContext) plugin.Plugin {
	return &Plugin{}
}

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	apiclient := api.NewClient("ws://localhost:8888", os.Getenv("GOTIFY_CLIENT_TOKEN"), &logger)
	tgclient := telegram.NewClient(os.Getenv("TELEGRAM_BOT_TOKEN"), &logger)
	p := &Plugin{
		logger:    &logger,
		apiclient: apiclient,
		tgclient:  tgclient,
	}
	p.Start()
}
