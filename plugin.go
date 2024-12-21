package main

import (
	"embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/api"
	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/config"
	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/telegram"
	"github.com/gin-gonic/gin"
	"github.com/gotify/plugin-api"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

//go:embed README.md
var content embed.FS

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
	msgHandler plugin.MessageHandler
	userCtx    plugin.UserContext
	logger     *zerolog.Logger
	apiclient  *api.Client
	tgclient   *telegram.Client
	config     *config.Plugin
	messages   chan api.Message
	done       chan struct{}
	errChan    chan error
}

// Enable enables the plugin.
func (p *Plugin) Enable() error {
	p.logger.Info().Msg("enabling plugin")
	go p.Start()
	return nil
}

// Disable disables the plugin.
func (p *Plugin) Disable() error {
	p.logger.Debug().Msg("disabling plugin")

	if p.apiclient != nil {
		if err := p.apiclient.Close(); err != nil {
			p.logger.Error().Err(err).Msg("failed to close api client")
			return err
		}
		p.logger.Debug().Msg("api client closed")
	}

	p.logger.Debug().Msg("sending done signal")
	p.done <- struct{}{}

	return nil
}

// RegisterWebhook implements plugin.Webhooker.
func (p *Plugin) RegisterWebhook(basePath string, g *gin.RouterGroup) {
}

func (p *Plugin) getTelegramBotConfigForAppID(appID uint32) config.TelegramBot {
	var botName string
	if p.config != nil {
		for _, rule := range p.config.Rules {
			for _, appid := range rule.AppIDs {
				if appid == appID {
					botName = rule.BotName
				}
			}
		}
	}

	botConfig, exists := p.config.TelegramConfig.Bots[botName]
	if exists {
		p.logger.Debug().
			Uint32("app_id", appID).
			Str("bot_name", botName).
			Msg("rule found for app_id")
		return botConfig
	} else {
		// Fallback to default if no rule matches
		p.logger.Warn().
			Uint32("app_id", appID).
			Str("bot_name", botName).
			Msgf("no rule found for app_id: %d. Using default config", appID)
		return config.TelegramBot{
			Token:  p.config.TelegramConfig.DefaultBotToken,
			ChatID: p.config.TelegramConfig.DefaultChatID,
		}
	}
}

func (p *Plugin) handleMessage(msg api.Message) {
	config := p.getTelegramBotConfigForAppID(msg.AppID)
	if err := p.tgclient.Send(msg, config); err != nil {
		p.logger.Error().Err(err).Msg("failed to send message to Telegram")
	}
}

// Start starts the plugin.
func (p *Plugin) Start() error {
	p.logger.Debug().Msg("starting plugin")

	if p.apiclient == nil {
		p.errChan <- errors.New("api client is not initialized")
	} else {
		p.logger.Debug().Msg("api client initialized")
		go p.apiclient.Start()
	}

	p.logger.Debug().Msg("starting Telegram client")
	for {
		select {
		case <-p.done:
			p.logger.Debug().Msg("stopping plugin")
			return nil

		case err := <-p.errChan:
			if err != nil {
				p.logger.Error().Err(err).Msg("error received")
			}

		case msg := <-p.messages:
			p.logger.Debug().Msgf("message received from gotify server: %s", msg.Message)
			p.handleMessage(msg)
		}
	}
}

// SetMessageHandler implements plugin.Messenger
// Invoked during initialization
func (p *Plugin) SetMessageHandler(handler plugin.MessageHandler) {
	p.msgHandler = handler
}

// GetDisplay implements plugin.Displayer
// Invoked when the user views the plugin settings. Plugins do not need to be enabled to handle GetDisplay calls.
func (p *Plugin) GetDisplay(location *url.URL) string {
	readme, err := content.ReadFile("README.md")
	if err != nil {
		p.logger.Error().Err(err).Msg("failed to read README.md")
		return "Gotify to Telegram plugin - forwards Gotify messages to Telegram bots based on configurable routing rules."
	}

	return string(readme)
}

// DefaultConfig implements plugin.Configurer
// The default configuration will be provided to the user for future editing. Also used for Unmarshaling.
// Invoked whenever an unmarshaling is required.
func (p *Plugin) DefaultConfig() interface{} {
	envCfg, err := config.ParseEnvVars()
	if err != nil {
		p.logger.Error().Err(err).Msg("failed to parse env vars. Using defaults")
		envCfg = config.CreateDefaultEnvConfig()
	}

	tgConfig := config.Telegram{
		DefaultBotToken: envCfg.TelegramBotToken,
		DefaultChatID:   envCfg.TelegramChatID,
	}

	serverConfig := config.GotifyServer{
		Url:         envCfg.GotifyServerURL,
		ClientToken: envCfg.GotifyClientToken,
	}
	return &config.Plugin{
		TelegramConfig:     tgConfig,
		GotifyServerConfig: serverConfig,
	}
}

// ValidateAndSetConfig will be called every time the plugin is initialized or the configuration has been changed by the user.
// Plugins should check whether the configuration is valid and optionally return an error.
// Parameter is guaranteed to be the same type as the return type of DefaultConfig()
func (p *Plugin) ValidateAndSetConfig(newConfig interface{}) error {
	pluginCfg, ok := newConfig.(*config.Plugin)
	if !ok {
		return fmt.Errorf("invalid config type: expected *config.Config, got %T", newConfig)
	}
	// Validate Telegram config
	if pluginCfg.TelegramConfig.DefaultBotToken == "" {
		return errors.New("telegram.default_bot_token is required")
	}
	if pluginCfg.TelegramConfig.DefaultChatID == "" {
		return errors.New("telegram.default_chat_id is required")
	}

	// Validate Gotify server config
	if pluginCfg.GotifyServerConfig.Url == nil || pluginCfg.GotifyServerConfig.Url.String() == "" {
		return errors.New("gotify_server.url is required")
	}

	if pluginCfg.GotifyServerConfig.ClientToken == "" {
		return errors.New("gotify_server.client_token is required")
	}

	p.config = pluginCfg

	if err := p.RestartAPIWithNewConfig(); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) RestartAPIWithNewConfig() error {
	p.logger.Info().Msg("stopping api client")
	if err := p.apiclient.Close(); err != nil {
		return err
	}

	apiConfig := api.Config{
		Url:         p.config.GotifyServerConfig.Url,
		ClientToken: p.config.GotifyServerConfig.ClientToken,
		Logger:      p.logger,
		Messages:    p.messages,
		ErrChan:     p.errChan,
	}

	p.logger.Info().Msg("creating api client with new config")
	apiclient := api.NewClient(apiConfig)
	p.apiclient = apiclient

	p.logger.Info().Msg("restarting api client")
	go p.apiclient.Start()

	return nil
}

// NewGotifyPluginInstance creates a plugin instance for a user context.
func NewGotifyPluginInstance(ctx plugin.UserContext) plugin.Plugin {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stdout}).With().
		Uint("user_id", ctx.ID).
		Str("user_name", ctx.Name).
		Bool("is_admin", ctx.Admin).
		Caller().
		Logger()

	done := make(chan struct{}, 1)
	messages := make(chan api.Message, 100)
	errChan := make(chan error, 100)

	envCfg, err := config.ParseEnvVars()
	if err != nil {
		logger.Error().Err(err).Msg("failed to parse env vars. Using defaults")
		envCfg = config.CreateDefaultEnvConfig()
	}

	apiConfig := api.Config{
		Url:         envCfg.GotifyServerURL,
		ClientToken: envCfg.GotifyClientToken,
		Logger:      &logger,
		Messages:    messages,
		ErrChan:     errChan,
	}
	apiclient := api.NewClient(apiConfig)
	tgclient := telegram.NewClient(&logger, "MarkdownV2")

	logger.Debug().Msg("creating plugin instance")

	pluginCfg := &config.Plugin{
		TelegramConfig: config.Telegram{
			DefaultBotToken: envCfg.TelegramBotToken,
			DefaultChatID:   envCfg.TelegramChatID,
		},
		GotifyServerConfig: config.GotifyServer{
			Url:         envCfg.GotifyServerURL,
			ClientToken: envCfg.GotifyClientToken,
		},
	}

	return &Plugin{
		userCtx:   ctx,
		config:    pluginCfg,
		logger:    &logger,
		apiclient: apiclient,
		tgclient:  tgclient,
		messages:  messages,
		done:      done,
		errChan:   errChan,
	}
}

func main() {
	ctx := plugin.UserContext{
		ID:    1,
		Name:  "0xPeterSatoshi",
		Admin: true,
	}
	p := NewGotifyPluginInstance(ctx)
	if err := p.Enable(); err != nil {
		panic(err)
	}

	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stdout}).With().
		Uint("user_id", ctx.ID).
		Str("user_name", ctx.Name).
		Bool("is_admin", ctx.Admin).
		Logger()

	// Create channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal
	<-sigChan

	// Clean shutdown
	if err := p.Disable(); err != nil {
		logger.Error().Err(err).Msg("failed to disable plugin")
	}
	logger.Info().Msg("shutdown complete")
}
