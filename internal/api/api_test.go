package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var mockApps = []Application{
	{
		ID:          1,
		Token:       "test-token",
		Name:        "Test App",
		Description: "Test Description",
	},
	{
		ID:          2,
		Token:       "test-token-2",
		Name:        "Test App 2",
		Description: "Test Description 2",
	},
}

func setupTestServer(t *testing.T) (*httptest.Server, *websocket.Upgrader) {
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/stream":
			// Handle WebSocket connection
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				t.Logf("Failed to upgrade connection: %v", err)
				return
			}
			defer conn.Close()

			// Keep connection alive
			for {
				select {
				case <-r.Context().Done():
					return
				}
			}

		case "/application":
			// Return mock applications
			json.NewEncoder(w).Encode(mockApps)

		default:
			http.NotFound(w, r)
		}
	}))

	return server, upgrader
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name       string
		config     Config
		wantURL    string
		wantErrors bool
	}{
		{
			name: "valid configuration",
			config: Config{
				Url:              &url.URL{Scheme: "http", Host: "example.com"},
				ClientToken:      "test-token",
				HandshakeTimeout: 10,
			},
			wantURL:    "http://example.com",
			wantErrors: false,
		},
		{
			name: "nil URL defaults to localhost",
			config: Config{
				ClientToken:      "test-token",
				HandshakeTimeout: 10,
			},
			wantURL:    "http://localhost:80",
			wantErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			messages := make(chan Message, 1)
			errChan := make(chan error, 1)

			tt.config.Messages = messages
			tt.config.ErrChan = errChan

			client := NewClient(ctx, tt.config)

			assert.NotNil(t, client)
			assert.Equal(t, tt.wantURL, client.serverURL.String())
			assert.Equal(t, tt.config.ClientToken, client.clientToken)
		})
	}
}

func TestClientStruct_connect(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	tests := []struct {
		name        string
		clientToken string
		wantError   bool
	}{
		{
			name:        "successful connection",
			clientToken: "valid-token",
			wantError:   false,
		},
		{
			name:        "empty client token",
			clientToken: "",
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			messages := make(chan Message, 1)
			errChan := make(chan error, 1)

			client := NewClient(ctx, Config{
				Url:              serverURL,
				ClientToken:      tt.clientToken,
				HandshakeTimeout: 1,
				Messages:         messages,
				ErrChan:          errChan,
			})

			err := client.connect()

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, client.isConnected)
				assert.NotNil(t, client.conn)
			}

			client.Close()
		})
	}
}

func TestClientStruct_processMessage(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	ctx := context.Background()
	messages := make(chan Message, 1)
	errChan := make(chan error, 1)

	client := NewClient(ctx, Config{
		Url:              serverURL,
		ClientToken:      "test-token",
		HandshakeTimeout: 1,
		Messages:         messages,
		ErrChan:          errChan,
	})

	msg := Message{
		Id:       1,
		AppID:    1,
		Message:  "Test Message",
		Title:    "Test Title",
		Priority: 1,
		Date:     time.Now(),
	}

	err = client.processMessage(msg)
	require.NoError(t, err)

	// Verify the message was processed and sent to the channel
	select {
	case receivedMsg := <-messages:
		assert.Equal(t, msg.Id, receivedMsg.Id)
		assert.Equal(t, "Test App", receivedMsg.AppName)
		assert.Equal(t, "Test Description", receivedMsg.AppDescription)
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestClientStruct_getApplications(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	ctx := context.Background()
	messages := make(chan Message, 1)
	errChan := make(chan error, 1)

	client := NewClient(ctx, Config{
		Url:              serverURL,
		ClientToken:      "test-token",
		HandshakeTimeout: 1,
		Messages:         messages,
		ErrChan:          errChan,
	})

	apps, err := client.getApplications()
	require.NoError(t, err)
	assert.Equal(t, mockApps, apps)
}

func TestClientStruct_getApplicationByID(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	ctx := context.Background()
	messages := make(chan Message, 1)
	errChan := make(chan error, 1)

	client := NewClient(ctx, Config{
		Url:              serverURL,
		ClientToken:      "test-token",
		HandshakeTimeout: 1,
		Messages:         messages,
		ErrChan:          errChan,
	})

	tests := []struct {
		name      string
		appID     uint32
		wantError bool
	}{
		{
			name:      "existing application",
			appID:     1,
			wantError: false,
		},
		{
			name:      "non-existent application",
			appID:     999,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := client.getApplicationByID(tt.appID)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, app)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, app)
				assert.Equal(t, tt.appID, app.ID)
			}
		})
	}
}
