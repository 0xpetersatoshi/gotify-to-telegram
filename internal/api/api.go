package api

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

type Message struct {
	Id       uint32
	Appid    uint32
	Message  string
	Title    string
	Priority uint32
	Extras   map[string]interface{}
	Date     time.Time
}

// Client is a gotify API client
type Client struct {
	host        string
	clientToken string
	conn        *websocket.Conn
	logger      *zerolog.Logger
}

// NewClient creates a new gotify API client
func NewClient(host, clientToken string, logger *zerolog.Logger) *Client {
	return &Client{
		host:        host,
		clientToken: clientToken,
		logger:      logger,
	}
}

// connect connects to the gotify API
func (c *Client) connect() error {
	endpoint := c.host + "/stream?token=" + c.clientToken
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(endpoint, nil)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}

// close closes the gotify API connection
func (c *Client) close() error {
	return c.conn.Close()
}

// ListenForMessages listens for messages from the gotify API
func (c *Client) ListenForMessages() error {
	if err := c.connect(); err != nil {
		return err
	}
	defer c.close()

	c.logger.Info().
		Str("host", c.host).
		Msg("listening for messages from gotify server")

	for {
		var msg Message
		if err := c.conn.ReadJSON(&msg); err != nil {
			c.logger.Error().Err(err).Msg("failed to read message from gotify server")
			continue
		}

		bs, err := json.MarshalIndent(msg, "", "  ")
		if err != nil {
			c.logger.Error().Err(err).Msg("failed to marshal message from gotify server")
			continue
		}
		c.logger.Info().Msgf("got message from gotify server: %s", string(bs))
	}
}
