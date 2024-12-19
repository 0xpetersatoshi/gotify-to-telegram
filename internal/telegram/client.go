package telegram

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/api"
	"github.com/rs/zerolog"
)

type Payload struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

type Client struct {
	token            string
	logger           *zerolog.Logger
	botAPIEndpoint   string
	defaultParseMode string
}

// NewClient creates a new Telegram client
func NewClient(token string, logger *zerolog.Logger, parseMode string) *Client {
	return &Client{
		token:            token,
		logger:           logger,
		botAPIEndpoint:   "https://api.telegram.org/bot" + token + "/sendMessage",
		defaultParseMode: parseMode,
	}
}

// Send sends a message to Telegram
func (c *Client) Send(message api.Message, chatID string) error {
	c.logger.Debug().Msg("sending message to Telegram")
	formattedMessage := formatMessageForTelegram(message, c.logger)
	c.logger.Debug().Msgf("formatted message: %s", formattedMessage)

	payload := Payload{
		ChatID:    chatID,
		Text:      formattedMessage,
		ParseMode: c.defaultParseMode,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	c.logger.Debug().Msg("making request to Telegram API")
	if err := c.makeRequest(bytes.NewBuffer(body)); err != nil {
		return err
	}
	return nil
}

// makeRequest makes a request to the Telegram API
func (c *Client) makeRequest(body *bytes.Buffer) error {
	req, err := http.NewRequest("POST", c.botAPIEndpoint, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		c.logger.
			Error().
			Err(err).
			Str("status", res.Status).
			Msg("failed to send message to Telegram")
		bs, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		c.logger.
			Error().
			Msgf("error from API: %s", string(bs))
		return err
	}

	return nil
}