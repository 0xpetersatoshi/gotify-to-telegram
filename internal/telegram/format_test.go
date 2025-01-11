package telegram

import (
	"testing"

	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/config"
	"github.com/stretchr/testify/require"
)

func TestEscapeSpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic special characters",
			input:    "Hello! This is a (test) with [brackets]",
			expected: "Hello\\! This is a \\(test\\) with \\[brackets\\]",
		},
		{
			name:     "multiple special characters",
			input:    "test.test-test_test",
			expected: "test\\.test\\-test\\_test",
		},
		{
			name:     "no special characters",
			input:    "Hello World",
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeSpecialCharacters(tt.input)
			if result != tt.expected {
				t.Errorf("escapeSpecialCharacters(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatPlainURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple URL",
			input:    "https://example.com",
			expected: "https://example\\.com",
		},
		{
			name:     "complex URL",
			input:    "https://example.com/path-to/something.html",
			expected: "https://example\\.com/path\\-to/something\\.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPlainURL(tt.input)
			if result != tt.expected {
				t.Errorf("formatPlainURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
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
			name:     "basic image markdown",
			input:    "![](https://example.com/image.jpg)",
			expected: "https://example\\.com/image\\.jpg",
		},
		{
			name:     "image markdown with alt text",
			input:    "![alt text](https://example.com/image.jpg)",
			expected: "https://example\\.com/image\\.jpg",
		},
		{
			name:     "invalid image markdown",
			input:    "![invalid markdown",
			expected: "![invalid markdown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAndFormatImageURL(tt.input)
			if result != tt.expected {
				t.Errorf("extractAndFormatImageURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatText(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expected   string
		formatOpts config.MessageFormatOptions
	}{
		{
			name: "complex text with multiple elements",
			input: `Terminator Salvation (2009) [Remux-1080p] https://www.imdb.com/title/tt0438488/
![](https://image.tmdb.org/t/p/original/gw6JhlekZgtKUFlDTezq3j5JEPK.jpg)
[some url here](https://imdb.com)`,
			expected: `Terminator Salvation \\(2009\\) \\[Remux\\-1080p\\] https://www\\.imdb\\.com/title/tt0438488/
https://image\\.tmdb\\.org/t/p/original/gw6JhlekZgtKUFlDTezq3j5JEPK\\.jpg
[some url here](https://imdb.com)`,
		},
		{
			name:     "text with inline URL",
			input:    "Check out this [link](https://example.com/test.html) and this text.",
			expected: "Check out this [link](https://example.com/test.html) and this text\\.",
		},
		{
			name:     "text with plain URL",
			input:    "Visit https://example.com/test.html for more info!",
			expected: "Visit https://example\\.com/test\\.html for more info\\!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatMessage(tt.input, tt.formatOpts)
			require.NoError(t, err)
			if result != tt.expected {
				t.Errorf("FormatText(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
