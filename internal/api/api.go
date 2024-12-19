package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

type Message struct {
	Id             uint32
	Appid          uint32
	AppName        string
	AppDescription string
	Message        string
	Title          string
	Priority       uint32
	Extras         map[string]interface{}
	Date           time.Time
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
	cache       *cache.Cache
	messages    chan<- Message
	errChan     chan<- error
}

type ClientOpts struct {
	Host        string
	ClientToken string
	Ssl         bool
	Logger      *zerolog.Logger
	Messages    chan<- Message
	ErrChan     chan<- error
}

// NewClient creates a new gotify API client
func NewClient(opts ClientOpts) *Client {
	cache := cache.New(60*time.Minute, 120*time.Minute)

	if opts.Host == "" {
		opts.ErrChan <- errors.New("gotify host is not set. Please set the GOTIFY_HOST environment variable.")
	}

	if opts.ClientToken == "" {
		opts.ErrChan <- errors.New("gotify client token is not set. Please set the GOTIFY_CLIENT_TOKEN environment variable.")
	}

	if opts.Logger == nil {
		logger := zerolog.New(io.Discard).With().Timestamp().Logger()
		opts.Logger = &logger
	}

	return &Client{
		host:        opts.Host,
		clientToken: opts.ClientToken,
		logger:      opts.Logger,
		ssl:         opts.Ssl,
		cache:       cache,
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
		c.logger.Error().Err(err).Msg("failed to connect to gotify server")
		return err
	}

	c.logger.Info().
		Str("host", c.host).
		Msg("connected to gotify server")

	c.conn = conn
	return nil
}

// Start starts the gotify API client
func (c *Client) Start() {
	if err := c.connect(); err != nil {
		c.errChan <- err
	}
	defer c.Close()

	c.ReadMessages()
}

// Close closes the gotify API connection
func (c *Client) Close() error {
	if c.conn == nil {
		c.logger.Debug().Msg("connection is not open")
		return nil
	}
	return c.conn.Close()
}

// ReadMessages reads messages received from the gotify server and sends them to the messages channel
func (c *Client) ReadMessages() {
	c.logger.Info().
		Str("host", c.host).
		Msg("listening for messages from gotify server")

	for {
		var msg Message
		if err := c.conn.ReadJSON(&msg); err != nil {
			c.logger.Error().Err(err).Msg("failed to read message from gotify server")
			c.errChan <- err
			continue
		}

		// add app name and description to message
		appItem, found := c.cache.Get(fmt.Sprintf("%d", msg.Appid))
		if found {
			app := appItem.(Application)
			msg.AppName = app.Name
			msg.AppDescription = app.Description
		} else {
			app, err := c.getApplicationByID(msg.Appid)
			if err != nil {
				c.logger.Error().Err(err).Msg("failed to get application from gotify server")
				c.errChan <- err
				continue
			}
			c.cache.SetDefault(fmt.Sprintf("%d", msg.Appid), *app)
			msg.AppName = app.Name
			msg.AppDescription = app.Description
		}

		c.messages <- msg

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

// getApplications returns a list of applications
func (c *Client) getApplications() ([]Application, error) {
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

// getApplicationByID returns an application by id
func (c *Client) getApplicationByID(id uint32) (*Application, error) {
	applications, err := c.getApplications()
	if err != nil {
		return nil, err
	}

	for _, application := range applications {
		if application.ID == id {
			return &application, nil
		}
	}

	return nil, fmt.Errorf("application with id %d not found", id)
}
