package telegram

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/api"
	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockHTTPClient is a mock HTTP client for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestNewClient(t *testing.T) {
	errChan := make(chan error, 1)
	client := NewClient(errChan)

	assert.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
	assert.NotNil(t, client.logger)
	assert.Equal(t, errChan, client.errChan)
}

func TestClientStruct_BuildBotEndpoint(t *testing.T) {
	client := NewClient(make(chan error, 1))

	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "valid token",
			token:    "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
			expected: "https://api.telegram.org/bot123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11/sendMessage",
		},
		{
			name:     "empty token",
			token:    "",
			expected: "https://api.telegram.org/bot/sendMessage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.buildBotEndpoint(tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClientStruct_Send(t *testing.T) {
	tests := []struct {
		name           string
		message        api.Message
		token          string
		chatID         string
		formatOpts     config.MessageFormatOptions
		mockResponse   *http.Response
		mockError      error
		expectedError  bool
		expectedErrMsg string
	}{
		{
			name: "successful send",
			message: api.Message{
				AppID:    1,
				AppName:  "TestApp",
				Message:  "Test Message",
				Title:    "Test Title",
				Priority: 1,
			},
			token:  "valid-token",
			chatID: "123456",
			formatOpts: config.MessageFormatOptions{
				ParseMode: "MarkdownV2",
			},
			mockResponse: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"ok":true}`)),
			},
			expectedError: false,
		},
		{
			name:           "empty token",
			token:          "",
			chatID:         "123456",
			expectedError:  true,
			expectedErrMsg: "telegram bot token is empty",
		},
		{
			name:           "empty chat ID",
			token:          "valid-token",
			chatID:         "",
			expectedError:  true,
			expectedErrMsg: "telegram chat ID is empty",
		},
		{
			name:    "API error response",
			token:   "valid-token",
			chatID:  "123456",
			message: api.Message{Message: "Test"},
			formatOpts: config.MessageFormatOptions{
				ParseMode: "MarkdownV2",
			},
			mockResponse: &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(bytes.NewBufferString(`{"ok":false,"error":"Bad Request"}`)),
			},
			expectedError:  true,
			expectedErrMsg: "telegram API error (status 400)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errChan := make(chan error, 1)
			client := NewClient(errChan)

			// Mock HTTP client if a response is provided
			if tt.mockResponse != nil {
				client.httpClient = &MockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						if tt.mockError != nil {
							return nil, tt.mockError
						}
						return tt.mockResponse, nil
					},
				}
			}

			// Send message
			client.Send(tt.message, tt.token, tt.chatID, tt.formatOpts)

			// Check for errors
			select {
			case err := <-errChan:
				if !tt.expectedError {
					t.Errorf("unexpected error: %v", err)
				} else if tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			case <-time.After(time.Second):
				if tt.expectedError {
					t.Error("expected error but got none")
				}
			}
		})
	}
}

func TestClientStruct_MakeRequest(t *testing.T) {
	tests := []struct {
		name           string
		endpoint       string
		payload        *bytes.Buffer
		mockResponse   *http.Response
		mockError      error
		expectedError  bool
		expectedErrMsg string
	}{
		{
			name:     "successful request",
			endpoint: "https://api.telegram.org/bot123456:ABC/sendMessage",
			payload:  bytes.NewBufferString(`{"chat_id":"123","text":"test"}`),
			mockResponse: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"ok":true}`)),
			},
			expectedError: false,
		},
		{
			name:           "invalid endpoint",
			endpoint:       "://invalid-url",
			payload:        bytes.NewBufferString(`{}`),
			expectedError:  true,
			expectedErrMsg: "failed to create request",
		},
		{
			name:     "server error",
			endpoint: "https://api.telegram.org/bot123456:ABC/sendMessage",
			payload:  bytes.NewBufferString(`{}`),
			mockResponse: &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewBufferString(`{"ok":false}`)),
			},
			expectedError:  true,
			expectedErrMsg: "telegram API error (status 500)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(make(chan error, 1))

			if tt.mockResponse != nil {
				client.httpClient = &MockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						if tt.mockError != nil {
							return nil, tt.mockError
						}
						return tt.mockResponse, nil
					},
				}
			}

			err := client.makeRequest(tt.endpoint, tt.payload)

			if tt.expectedError {
				require.Error(t, err)
				if tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPayload_Marshal(t *testing.T) {
	tests := []struct {
		name     string
		payload  Payload
		expected string
	}{
		{
			name: "basic payload",
			payload: Payload{
				ChatID:    "123456",
				Text:      "test message",
				ParseMode: "MarkdownV2",
			},
			expected: `{"chat_id":"123456","text":"test message","parse_mode":"MarkdownV2"}`,
		},
		{
			name: "empty parse mode",
			payload: Payload{
				ChatID: "123456",
				Text:   "test message",
			},
			expected: `{"chat_id":"123456","text":"test message","parse_mode":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.payload)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))
		})
	}
}
