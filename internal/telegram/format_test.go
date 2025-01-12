package telegram

import (
	"strings"
	"testing"

	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/api"
	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestEscapeMarkdownV2(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "it shoudl escape special characters",
			input:    "Hello_World*[Test]",
			expected: "Hello\\_World\\*\\[Test\\]",
		},
		{
			name:     "it should preserve newline characters",
			input:    "Hello\nWorld",
			expected: "Hello\nWorld",
		},
		{
			name:     "it should handle mixed special chars and newlines",
			input:    "Hello_World\nTest*Now",
			expected: "Hello\\_World\nTest\\*Now",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeMarkdownV2(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatPlainURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "it should escape special characters in a simple url",
			url:      "https://example.com",
			expected: "https://example\\.com",
		},
		{
			name:     "it should escape special characters in a url with special characters",
			url:      "https://example.com/path_to/file.txt",
			expected: "https://example\\.com/path\\_to/file\\.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPlainURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractAndFormatImageURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "it should return the URL from valid image markdown",
			input:    "![alt text](https://example.com/image.jpg)",
			expected: "https://example.com/image.jpg",
		},
		{
			name:     "it should return the entire string from invalid image markdown",
			input:    "invalid markdown",
			expected: "invalid markdown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAndFormatImageURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatMessageAsMarkdownV2(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "it should preserve text with inline URL",
			input:    "Check [this link](https://example.com)",
			expected: "Check [this link](https://example.com)",
		},
		{
			name:     "it should extract and escape URL from image markdown",
			input:    "See this: ![](https://example.com/img.jpg)",
			expected: "See this: https://example\\.com/img\\.jpg",
		},
		{
			name:     "it should escape text with special characters",
			input:    "Hello_World*Test",
			expected: "Hello\\_World\\*Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMessageAsMarkdownV2(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPriorityIndicator(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		expected string
	}{
		{"critical priority", 8, "🔴 Critical Priority"},
		{"high priority", 6, "🟠 High Priority"},
		{"medium priority", 4, "🟡 Medium Priority"},
		{"low priority", 2, "🟢 Low Priority"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPriorityIndicator(tt.priority)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatExtras(t *testing.T) {
	tests := []struct {
		name     string
		extras   map[string]interface{}
		expected string
	}{
		{
			name: "it should format simple extras",
			extras: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			expected: "\n• key1: `value1`\n• key2: `value2`\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var builder strings.Builder
			formatExtras(&builder, tt.extras, "")
			assert.Equal(t, tt.expected, builder.String())
		})
	}
}

func TestFormatMessage_Integration(t *testing.T) {
	msg := api.Message{
		Title:    "Test Title",
		Message:  "Hello_World with [link](https://example.com)",
		AppName:  "TestApp",
		Priority: 8,
		Extras: map[string]interface{}{
			"key": "value",
		},
	}

	opts := config.MessageFormatOptions{
		ParseMode:         "MarkdownV2",
		IncludeAppName:    true,
		IncludePriority:   true,
		IncludeExtras:     true,
		IncludeTimestamp:  true,
		PriorityThreshold: 5,
	}

	result, err := FormatMessage(msg, opts)
	assert.NoError(t, err)
	assert.Contains(t, result, `\[TestApp\]`)
	assert.Contains(t, result, "Hello\\_World")
	assert.Contains(t, result, "[link](https://example.com)")
	assert.Contains(t, result, "🔴 Critical Priority")
	assert.Contains(t, result, "key: `value`")
	assert.Contains(t, result, "timestamp:")
}

func TestFormatMessage_InvalidParseMode(t *testing.T) {
	msg := api.Message{
		Title:   "Test",
		Message: "Test",
	}

	opts := config.MessageFormatOptions{
		ParseMode: "InvalidMode",
	}

	_, err := FormatMessage(msg, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse mode InvalidMode is not supported")
}
