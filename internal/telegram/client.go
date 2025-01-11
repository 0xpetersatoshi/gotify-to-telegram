package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/api"
	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/config"
	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/logger"
	"github.com/rs/zerolog"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Payload struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

type Client struct {
	logger     *zerolog.Logger
	httpClient HTTPClient
	errChan    chan error
}

// NewClient creates a new Telegram client
func NewClient(errChan chan error) *Client {
	return &Client{
		logger:     logger.WithComponent("telegram"),
		httpClient: &http.Client{},
		errChan:    errChan,
	}
}

func (c *Client) buildBotEndpoint(token string) string {
	return "https://api.telegram.org/bot" + token + "/sendMessage"
}

// Send sends a message to Telegram
func (c *Client) Send(message api.Message, token, chatID string, formatOpts config.MessageFormatOptions) {
	if token == "" {
		c.errChan <- fmt.Errorf("telegram bot token is empty")
		return
	}
	if chatID == "" {
		c.errChan <- fmt.Errorf("telegram chat ID is empty")
		return
	}

	c.logger.Debug().
		Uint32("app_id", message.AppID).
		Str("app_name", message.AppName).
		Str("chat_id", chatID).
		Msg("preparing to send message to Telegram")

	formattedMessage, err := FormatMessage(message.Message, formatOpts)
	if err != nil {
		c.errChan <- fmt.Errorf("failed to format message: %w", err)
		return
	}

	payload := Payload{
		ChatID:    chatID,
		Text:      formattedMessage,
		ParseMode: formatOpts.ParseMode,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		c.errChan <- fmt.Errorf("failed to marshal payload: %w", err)
		return
	}

	endpoint := c.buildBotEndpoint(token)
	c.logger.Debug().
		Str("endpoint", strings.Replace(endpoint, token, "***", 1)).
		Str("formattedMessage", formattedMessage).
		Str("payload", string(body)).
		Msg("sending request to Telegram API")

	if err := c.makeRequest(endpoint, bytes.NewBuffer(body)); err != nil {
		c.errChan <- fmt.Errorf("failed to make request: %w", err)
		return
	}

	c.logger.Info().Msg("message successfully sent to Telegram")
}

// makeRequest makes a request to the Telegram API
func (c *Client) makeRequest(endpoint string, body *bytes.Buffer) error {
	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API error (status %d): %s", res.StatusCode, string(resBody))
	}

	c.logger.Debug().
		Str("response", string(resBody)).
		Msg("received response from Telegram API")

	return nil
}
