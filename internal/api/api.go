package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

type Message struct {
	Id             uint32
	AppID          uint32
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
	serverURL   *url.URL
	clientToken string
	conn        *websocket.Conn
	logger      *zerolog.Logger
	cache       *cache.Cache
	messages    chan<- Message
	errChan     chan<- error
	ctx         context.Context
	mu          sync.Mutex
	isConnected bool
}

type Config struct {
	Url         *url.URL
	ClientToken string
	Logger      *zerolog.Logger
	Messages    chan<- Message
	ErrChan     chan<- error
}

// NewClient creates a new gotify API client
func NewClient(ctx context.Context, c Config) *Client {
	cache := cache.New(60*time.Minute, 120*time.Minute)

	if c.Logger == nil {
		logger := zerolog.New(io.Discard).With().Timestamp().Logger()
		c.Logger = &logger
	}

	// TODO: Get default gotify url from config package function
	if c.Url == nil || (c.Url != nil && c.Url.String() == "") {
		// if no url is provided, default to localhost
		c.Logger.Warn().Msg("gotify url is not set. Defaulting to localhost")
		c.Url = &url.URL{
			Scheme: "http",
			Host:   "localhost:80",
		}
	}

	return &Client{
		serverURL:   c.Url,
		clientToken: c.ClientToken,
		logger:      c.Logger,
		messages:    c.Messages,
		errChan:     c.ErrChan,
		cache:       cache,
		ctx:         ctx,
	}
}

// connect connects to the gotify API
func (c *Client) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isConnected {
		c.logger.Debug().Msg("already connected to gotify server")
		return nil
	}

	if c.serverURL.Host == "" {
		return errors.New("gotify host is not set")
	}

	if c.clientToken == "" {
		return errors.New("gotify client token is not set")
	}

	protocol := "ws://"
	if c.serverURL.Scheme == "https" {
		protocol = "wss://"
	}
	endpoint := protocol + c.serverURL.Host + "/stream?token=" + c.clientToken

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(c.ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.conn = conn
	c.isConnected = true

	c.logger.Info().
		Str("protocol", protocol).
		Str("host", c.serverURL.Host).
		Msg("connected to gotify server")

	return nil
}

// Start establishes a websocket connection and starts reading incoming messages
func (c *Client) Start() {
	c.logger.Info().Msg("starting new gotify websocket connection")

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug().
				Err(c.ctx.Err()).
				Msg("stopping client...")
			return
		default:
			if err := c.connect(); err != nil {
				c.logger.Error().Err(err).Msg("failed to connect")
				select {
				case <-c.ctx.Done():
					c.logger.Debug().
						Err(c.ctx.Err()).
						Msg("closing websocket connection")
					if err := c.Close(); err != nil {
						c.logger.Error().Err(err).Msg("error closing connection")
						c.errChan <- err
						return
					}
					return
				case <-time.After(5 * time.Second):
					continue
				}
			}

			// Start message reading
			if err := c.readMessages(); err != nil {
				if !errors.Is(err, context.Canceled) {
					c.logger.Error().Err(err).Msg("error reading messages")
				}
			}

			// Reset connection state
			c.mu.Lock()
			c.isConnected = false
			c.mu.Unlock()
		}
	}
}

// Close closes the gotify websocket connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && c.isConnected {
		// Send close message
		err := c.conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		if err != nil {
			c.logger.Warn().Err(err).Msg("error sending close message")
		}

		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("error closing connection: %w", err)
		}
		c.isConnected = false
		c.logger.Debug().Msg("websocket connection closed")
	}

	return nil
}

// readMessages reads messages received from the gotify server and sends them to the messages channel
func (c *Client) readMessages() error {
	// Create channels for the reader goroutine
	msgChan := make(chan Message)
	errChan := make(chan error)

	// Start a separate goroutine for reading
	go func() {
		for {
			var msg Message
			if err := c.conn.ReadJSON(&msg); err != nil {
				c.mu.Lock()
				c.isConnected = false
				c.mu.Unlock()

				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					errChan <- fmt.Errorf("websocket error: %w", err)
					return
				}
				errChan <- err
				return
			}
			msgChan <- msg
		}
	}()

	c.logger.Info().Msg("listening for new messages")
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug().
				Err(c.ctx.Err()).
				Msg("stopping read event loop")
			// Close the websocket connection to unblock the reader goroutine
			if err := c.Close(); err != nil {
				c.logger.Error().Err(err).Msg("error closing websocket connection")
			}
			return c.ctx.Err()

		case err := <-errChan:
			return err

		case msg := <-msgChan:
			if err := c.processMessage(msg); err != nil {
				c.logger.Error().Err(err).Msg("failed to process message")
				continue
			}
		}
	}
}

func (c *Client) processMessage(msg Message) error {
	c.logger.Debug().Msg("processing new message")
	appItem, found := c.cache.Get(fmt.Sprintf("%d", msg.AppID))
	if found {
		app := appItem.(Application)
		msg.AppName = app.Name
		msg.AppDescription = app.Description
	} else {
		app, err := c.getApplicationByID(msg.AppID)
		if err != nil {
			return fmt.Errorf("failed to get application: %w", err)
		}
		c.cache.SetDefault(fmt.Sprintf("%d", msg.AppID), *app)
		msg.AppName = app.Name
		msg.AppDescription = app.Description
	}

	select {
	case <-c.ctx.Done():
		c.logger.Debug().
			Err(c.ctx.Err()).
			Msg("stopping message processing")
		return c.ctx.Err()
	case c.messages <- msg:
		c.logger.Info().Msg("message sent to message channel")
	}

	return nil
}

// makeRequest makes a request to the gotify API and returns the raw response
func (c *Client) makeRequest(method string, endpoint string, body *bytes.Buffer) (*http.Response, error) {
	// Create request body if provided
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}
	req, err := http.NewRequest(method, endpoint, reqBody)
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
	endpoint := c.serverURL.String() + "/application?token=" + c.clientToken

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
