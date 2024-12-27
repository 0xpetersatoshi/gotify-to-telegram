package telegram

import (
	"strings"
	"testing"

	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/api"
	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestProcessMarkdownLinks(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple link",
			input:    "[text](https://example.com)",
			expected: "[text](https://example.com)",
		},
		{
			name:     "link with special characters",
			input:    "[test!](https://example.com/test!)",
			expected: "[test\\!](https://example.com/test!)",
		},
		{
			name:     "multiple links",
			input:    "[link1](url1) and [link2](url2)",
			expected: "[link1](url1) and [link2](url2)",
		},
		{
			name:     "invalid link format",
			input:    "[broken link(http://example.com)",
			expected: "\\[broken link\\(http://example\\.com\\)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processMarkdownLinks(tt.input, &logger)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeURLForMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "url with parentheses",
			input:    "http://example.com/path(1)",
			expected: "http://example.com/path\\(1\\)",
		},
		{
			name:     "simple url",
			input:    "http://example.com",
			expected: "http://example.com",
		},
		{
			name:     "url with nested parentheses",
			input:    "http://example.com/(test(1))",
			expected: "http://example.com/\\(test\\(1\\)\\)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeURLForMarkdown(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeMarkdownV2(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "special characters",
			input:    "Hello *world* with _emphasis_!",
			expected: "Hello \\*world\\* with \\_emphasis\\_\\!",
		},
		{
			name:     "code and links",
			input:    "Check `this` and [that]",
			expected: "Check \\`this\\` and \\[that\\]",
		},
		{
			name:     "dots and dashes",
			input:    "Example.com - test",
			expected: "Example\\.com \\- test",
		},
		{
			name:     "no special characters",
			input:    "Hello world",
			expected: "Hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeMarkdownV2(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertImageMarkdownToURL(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single image",
			input:    "![](https://example.com/image.jpg)",
			expected: "https://example.com/image.jpg",
		},
		{
			name:     "multiple images",
			input:    "![](image1.jpg)\n![](image2.jpg)",
			expected: "image1.jpg\nimage2.jpg",
		},
		{
			name:     "invalid image markdown",
			input:    "![broken(image.jpg)",
			expected: "![broken(image.jpg)",
		},
		{
			name:     "mixed content",
			input:    "Text ![](image.jpg) more text",
			expected: "Text image.jpg more text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertImageMarkdownToURL(tt.input, &logger)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatExtras(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	var builder strings.Builder

	tests := []struct {
		name     string
		extras   map[string]interface{}
		prefix   string
		expected string
	}{
		{
			name: "simple extras",
			extras: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
			},
			prefix:   "",
			expected: "\n• key1: `value1`\n• key2: `123`",
		},
		{
			name: "nested extras",
			extras: map[string]interface{}{
				"outer": map[string]interface{}{
					"inner": "value",
				},
			},
			prefix:   "",
			expected: "\n• outer:\n  • inner: `value`",
		},
		{
			name: "mixed types",
			extras: map[string]interface{}{
				"string": "text",
				"number": 42,
				"bool":   true,
			},
			prefix:   "",
			expected: "\n• bool: `true`\n• number: `42`\n• string: `text`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder.Reset()
			formatExtras(&builder, tt.extras, tt.prefix, &logger)
			assert.Equal(t, tt.expected, builder.String())
		})
	}
}

func TestFormatMessageForTelegram(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	tests := []struct {
		name       string
		message    api.Message
		formatOpts config.MessageFormatOptions
		expected   string
	}{
		{
			name: "basic message",
			message: api.Message{
				Title:   "Test Title",
				Message: "Test Message",
				AppName: "TestApp",
			},
			formatOpts: config.MessageFormatOptions{
				IncludeAppName:   true,
				IncludeTimestamp: false,
			},
			expected: "*\\[TestApp\\] Test Title*\n\nTest Message\n",
		},
		{
			name: "message with priority",
			message: api.Message{
				Title:    "Priority Test",
				Message:  "Important Message",
				Priority: 8,
			},
			formatOpts: config.MessageFormatOptions{
				IncludeAppName:    false,
				IncludePriority:   true,
				PriorityThreshold: 5,
			},
			expected: "*Priority Test*\n\nImportant Message\n\n🔴 Critical Priority",
		},
		{
			name: "message with extras",
			message: api.Message{
				Title:   "Extras Test",
				Message: "Test with extras",
				Extras: map[string]interface{}{
					"key": "value",
				},
			},
			formatOpts: config.MessageFormatOptions{
				IncludeExtras: true,
			},
			expected: "*Extras Test*\n\nTest with extras\n\n*Additional Info:*\n• key: `value`",
		},
		{
			name: "message with markdown",
			message: api.Message{
				Title:   "Markdown Test",
				Message: "[link](https://example.com) and ![](image.jpg)",
			},
			formatOpts: config.MessageFormatOptions{},
			expected:   "*Markdown Test*\n\n[link](https://example.com) and image.jpg\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMessageForTelegram(tt.message, tt.formatOpts, &logger)
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
		{
			name:     "critical priority",
			priority: 8,
			expected: "🔴 Critical Priority",
		},
		{
			name:     "high priority",
			priority: 6,
			expected: "🟠 High Priority",
		},
		{
			name:     "medium priority",
			priority: 4,
			expected: "🟡 Medium Priority",
		},
		{
			name:     "low priority",
			priority: 2,
			expected: "🟢 Low Priority",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPriorityIndicator(tt.priority)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatTitle(t *testing.T) {
	tests := []struct {
		name     string
		message  api.Message
		expected string
	}{
		{
			name: "basic title",
			message: api.Message{
				AppName: "TestApp",
				Title:   "Test Title",
			},
			expected: "[TestApp] Test Title",
		},
		{
			name: "empty app name",
			message: api.Message{
				AppName: "",
				Title:   "Test Title",
			},
			expected: "[] Test Title",
		},
		{
			name: "empty title",
			message: api.Message{
				AppName: "TestApp",
				Title:   "",
			},
			expected: "[TestApp] ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTitle(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}
