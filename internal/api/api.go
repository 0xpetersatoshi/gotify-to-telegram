package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

type Application struct {
	ID              uint32 `json:"id"`
	Token           string `json:"token"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Internal        bool   `json:"internal"`
	Image           string `json:"image"`
	DefaultPriority uint32 `json:"defaultPriority"`
	LastUsed        string `json:"lastUsed"`
}

// Client is a gotify API client
type Client struct {
	ssl         bool
	host        string
	clientToken string
	conn        *websocket.Conn
	logger      *zerolog.Logger
}

// NewClient creates a new gotify API client
func NewClient(host, clientToken string, ssl bool, logger *zerolog.Logger) *Client {
	return &Client{
		host:        host,
		clientToken: clientToken,
		logger:      logger,
		ssl:         ssl,
	}
}

// connect connects to the gotify API
func (c *Client) connect() error {
	var protocol string
	if c.ssl {
		protocol = "wss://"
	} else {
		protocol = "ws://"
	}
	endpoint := protocol + c.host + "/stream?token=" + c.clientToken
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

// ReadMessages reads messages received from the gotify server and sends them to the messages channel
func (c *Client) ReadMessages(messages chan<- Message) error {
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

		messages <- msg

		c.logger.Info().Msg("message received from gotify server and sent to channel")
	}
}

// makeRequest makes a request to the gotify API and returns the raw response
func (c *Client) makeRequest(method string, endpoint string, body *bytes.Buffer) (*http.Response, error) {
	var protocol string
	if c.ssl {
		protocol = "https://"
	} else {
		protocol = "http://"
	}
	// Create request body if provided
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}
	req, err := http.NewRequest(method, protocol+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	c.logger.Debug().Msgf("making request to %s", endpoint)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, err
	}

	return res, nil
}

// GetApplications returns a list of applications
func (c *Client) GetApplications() ([]Application, error) {
	endpoint := c.host + "/application?token=" + c.clientToken

	res, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	var applications []Application
	if err := json.NewDecoder(res.Body).Decode(&applications); err != nil {
		return nil, err
	}

	return applications, nil
}
