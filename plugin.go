package main

import (
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
	config     *config.Config
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

func (p *Plugin) getTelegramBotConfigForAppID(appID uint32) config.TelegramBotConfig {
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
		return config.TelegramBotConfig{
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
	// TODO: add instructions
	if p.userCtx.Admin {
		return "You are an admin! You have super cow powers."
	} else {
		return "You are **NOT** an admin! You can do nothing:("
	}
}

// DefaultConfig implements plugin.Configurer
// The default configuration will be provided to the user for future editing. Also used for Unmarshaling.
// Invoked whenever an unmarshaling is required.
func (p *Plugin) DefaultConfig() interface{} {
	tgConfig := config.TelegramConfig{
		DefaultBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		DefaultChatID:   os.Getenv("TELEGRAM_CHAT_ID"),
	}

	serverConfig := config.GotifyServerConfig{
		Hostname:    os.Getenv("GOTIFY_HOSTNAME"),
		Protocol:    os.Getenv("GOTIFY_PROTOCOL"),
		Port:        os.Getenv("GOTIFY_PORT"),
		ClientToken: os.Getenv("GOTIFY_CLIENT_TOKEN"),
	}
	return &config.Config{
		TelegramConfig:     tgConfig,
		GotifyServerConfig: serverConfig,
	}
}

// ValidateAndSetConfig will be called every time the plugin is initialized or the configuration has been changed by the user.
// Plugins should check whether the configuration is valid and optionally return an error.
// Parameter is guaranteed to be the same type as the return type of DefaultConfig()
func (p *Plugin) ValidateAndSetConfig(newConfig interface{}) error {
	c, ok := newConfig.(*config.Config)
	if !ok {
		return fmt.Errorf("invalid config type: expected *config.Config, got %T", newConfig)
	}
	// Validate Telegram config
	if c.TelegramConfig.DefaultBotToken == "" {
		return errors.New("telegram.default_bot_token is required")
	}
	if c.TelegramConfig.DefaultChatID == "" {
		return errors.New("telegram.default_chat_id is required")
	}

	// Validate Gotify server config
	if c.GotifyServerConfig.Hostname == "" {
		return errors.New("gotify_server.hostname is required")
	}

	p.config = c
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

	apiOpts := api.ClientOpts{
		Host:        os.Getenv("GOTIFY_HOST"),
		ClientToken: os.Getenv("GOTIFY_CLIENT_TOKEN"),
		Ssl:         false,
		Logger:      &logger,
		Messages:    messages,
		ErrChan:     errChan,
	}
	apiclient := api.NewClient(apiOpts)
	tgclient := telegram.NewClient(os.Getenv("TELEGRAM_BOT_TOKEN"), &logger, "MarkdownV2")

	logger.Debug().Msg("creating plugin instance")

	return &Plugin{
		userCtx:   ctx,
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
