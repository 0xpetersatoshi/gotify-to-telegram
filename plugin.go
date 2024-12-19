package main

import (
	"errors"
	"os"
	"os/signal"
	"syscall"

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
	msgHandler plugin.MessageHandler
	userCtx    plugin.UserContext
	logger     *zerolog.Logger
	apiclient  *api.Client
	tgclient   *telegram.Client
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
			if err := p.tgclient.Send(msg, os.Getenv("TELEGRAM_CHAT_ID")); err != nil {
				p.logger.Error().Err(err).Msg("failed to send message to Telegram")
			}
		}
	}
}

func (p *Plugin) SetMessageHandler(handler plugin.MessageHandler) {
	p.msgHandler = handler
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
