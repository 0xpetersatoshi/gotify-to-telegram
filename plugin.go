package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/api"
	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/config"
	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/logger"
	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/telegram"
	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/utils"
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
	enabled    bool
	msgHandler plugin.MessageHandler
	userCtx    plugin.UserContext
	ctx        context.Context
	cancel     context.CancelFunc
	logger     *zerolog.Logger
	apiclient  *api.Client
	tgclient   *telegram.Client
	config     *config.Plugin
	messages   chan api.Message
	errChan    chan error
}

// Enable enables the plugin.
func (p *Plugin) Enable() error {
	p.enabled = true
	p.logger.Info().Msg("enabling plugin and starting services")
	go p.Start()
	return nil
}

// Disable disables the plugin.
func (p *Plugin) Disable() error {
	p.enabled = false
	p.logger.Debug().Msg("disabling plugin")
	p.cancel()

	return nil
}

func (p *Plugin) getTelegramBotConfigForAppID(appID uint32) config.TelegramBot {
	if p.config != nil {
		for _, bot := range p.config.Settings.Telegram.Bots {
			for _, appid := range bot.AppIDs {
				if appid == appID {
					return bot
				}
			}
		}
	}

	// Fallback to default if app id not found for bot config
	p.logger.Warn().
		Uint32("app_id", appID).
		Msgf("no rule found for app_id: %d. Using default config", appID)
	return config.TelegramBot{
		Token:   p.config.Settings.Telegram.DefaultBotToken,
		ChatIDs: p.config.Settings.Telegram.DefaultChatIDs,
	}
}

func (p *Plugin) handleMessage(msg api.Message) {
	p.logger.Debug().
		Str("app_name", msg.AppName).
		Uint32("app_id", msg.AppID).
		Msg("handling message")

	config := p.getTelegramBotConfigForAppID(msg.AppID)
	if config.MessageFormatOptions == nil {
		config.MessageFormatOptions = &p.config.Settings.Telegram.MessageFormatOptions
	}

	p.logger.Debug().
		Str("bot_token", utils.MaskToken(config.Token)).
		Strs("chat_id", config.ChatIDs).
		Msg("using telegram config")

	for _, chatID := range config.ChatIDs {
		go p.tgclient.Send(msg, config.Token, chatID, *config.MessageFormatOptions)
	}
}

// Start starts the plugin.
func (p *Plugin) Start() error {
	p.logger.Info().Msg("starting plugin services")

	if p.apiclient == nil {
		p.errChan <- errors.New("api client is not initialized")
	} else {
		p.logger.Debug().Msg("starting api client")
		go p.apiclient.Start()
	}

	for {
		select {
		case <-p.ctx.Done():
			p.logger.Info().Msg("stopping services")
			return nil

		case err := <-p.errChan:
			if err != nil {
				p.logger.Error().Err(err).Msg("error received")
			}

		case msg := <-p.messages:
			p.logger.Debug().
				Interface("message", msg).
				Msg("message received from gotify server")
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
	cfg := config.CreateDefaultPluginConfig()

	if !cfg.Settings.IgnoreEnvVars {
		if err := config.MergeWithEnvVars(cfg); err != nil {
			p.logger.Error().Err(err).Msg("failed to merge with env vars")
		}
	}

	if err := cfg.Validate(); err != nil {
		p.logger.Error().Err(err).Msg("failed to validate default config")
	}

	return cfg
}

// ValidateAndSetConfig will be called every time the plugin is initialized or the configuration has been changed by the user.
// Plugins should check whether the configuration is valid and optionally return an error.
// Parameter is guaranteed to be the same type as the return type of DefaultConfig()
func (p *Plugin) ValidateAndSetConfig(newConfig interface{}) error {
	pluginCfg, ok := newConfig.(*config.Plugin)
	if !ok {
		return fmt.Errorf("invalid config type: expected *config.Config, got %T", newConfig)
	}

	if err := pluginCfg.Validate(); err != nil {
		return err
	}

	if !pluginCfg.Settings.IgnoreEnvVars {
		p.logger.Debug().Msg("merging env vars with config")
		// Env vars take precedence over yaml config
		if err := config.MergeWithEnvVars(pluginCfg); err != nil {
			return err
		}

		p.logger.Debug().Msg("re-validating config")
		// re-validate after merging with env vars
		if err := pluginCfg.Validate(); err != nil {
			return err
		}
	}

	p.logger.Info().Msg("validated and setting new config")
	p.config = pluginCfg

	if p.enabled {
		p.logger.Info().Msg("plugin is enabled. Cancelling existing goroutines")
		// Stop existing goroutines
		p.cancel()
	}

	updatedLogger := p.logger.Level(pluginCfg.Settings.LogOptions.GetZerologLevel())
	p.logger = &updatedLogger

	p.logger.Debug().Msg("creating new context")
	ctx, cancel := context.WithCancel(context.Background())
	p.ctx = ctx
	p.cancel = cancel

	if err := p.updateAPIConfig(ctx); err != nil {
		return err
	}

	if err := p.updateTelegramConfig(); err != nil {
		return err
	}

	if p.enabled {
		p.logger.Info().Msg("plugin is enabled. Starting new goroutines")
		go p.Start()
	}

	return nil
}

func (p *Plugin) updateAPIConfig(ctx context.Context) error {
	apiConfig := api.Config{
		Url:              p.config.Settings.GotifyServer.Url,
		ClientToken:      p.config.Settings.GotifyServer.ClientToken,
		HandshakeTimeout: p.config.Settings.GotifyServer.Websocket.HandshakeTimeout,
		Messages:         p.messages,
		ErrChan:          p.errChan,
	}

	p.logger.Debug().Msg("creating api client with new config")
	apiclient := api.NewClient(ctx, apiConfig)
	p.apiclient = apiclient

	return nil
}

func (p *Plugin) updateTelegramConfig() error {
	p.logger.Debug().Msg("updating telegram client")
	p.tgclient = telegram.NewClient(p.errChan)
	return nil
}

// NewGotifyPluginInstance creates a plugin instance for a user context.
func NewGotifyPluginInstance(userCtx plugin.UserContext) plugin.Plugin {
	ctx, cancel := context.WithCancel(context.Background())
	log := logger.Init("gotify-to-telegram", userCtx)

	messages := make(chan api.Message, 100)
	errChan := make(chan error, 100)

	cfg, err := config.ParseEnvVars()
	if err != nil {
		log.Error().Err(err).Msg("failed to parse env vars. Using defaults")
		cfg = config.CreateDefaultPluginConfig()
	}

	logLevel := cfg.Settings.LogOptions.GetZerologLevel()
	logger.UpdateLogLevel(logLevel)

	apiConfig := api.Config{
		Url:              cfg.Settings.GotifyServer.Url,
		ClientToken:      cfg.Settings.GotifyServer.ClientToken,
		HandshakeTimeout: cfg.Settings.GotifyServer.Websocket.HandshakeTimeout,
		Messages:         messages,
		ErrChan:          errChan,
	}
	apiclient := api.NewClient(ctx, apiConfig)
	tgclient := telegram.NewClient(errChan)

	log.Info().Msg("creating new plugin instance")

	return &Plugin{
		userCtx:   userCtx,
		ctx:       ctx,
		cancel:    cancel,
		config:    cfg,
		logger:    log,
		apiclient: apiclient,
		tgclient:  tgclient,
		messages:  messages,
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
		Str("plugin", "gotify-to-telegram").
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
