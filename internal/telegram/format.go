package telegram

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/config"
)

// charactersToEscape contains all special characters that need to be escaped in regular text
var charactersToEscape = []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}

// urlRegex matches URLs in the text
var urlRegex = regexp.MustCompile(`https?://[^\s]+`)

// imageMarkdownRegex matches image markdown syntax
var imageMarkdownRegex = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

// inlineURLRegex matches inline URL markdown syntax
var inlineURLRegex = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

// escapeSpecialCharacters escapes all special characters in a text string
func escapeSpecialCharacters(text string) string {
	escaped := text
	for _, char := range charactersToEscape {
		// Don't escape backslashes that are part of \n
		if strings.Contains(escaped, `\n`) {
			parts := strings.Split(escaped, `\n`)
			for i, part := range parts {
				if char != `\` { // Don't escape backslashes
					parts[i] = strings.ReplaceAll(part, char, `\`+char)
				}
			}
			escaped = strings.Join(parts, `\n`)
		} else {
			escaped = strings.ReplaceAll(escaped, char, `\`+char)
		}
	}
	return escaped
}

// formatPlainURL escapes special characters in a URL
func formatPlainURL(url string) string {
	return escapeSpecialCharacters(url)
}

// extractAndFormatImageURL extracts the URL from an image markdown and formats it
func extractAndFormatImageURL(imageMarkdown string) string {
	matches := imageMarkdownRegex.FindStringSubmatch(imageMarkdown)
	if len(matches) < 3 {
		return imageMarkdown
	}
	return matches[2]
}

// preserveInlineURL keeps the inline URL markdown format intact
func preserveInlineURL(inlineURL string) string {
	// Since the inline URL format is allowed, we return it as is
	return inlineURL
}

func formatMessageAsMarkdownV2(input string) string {
	// First, collect all inline URLs to preserve them
	inlineURLs := make(map[string]string)
	currentText := input

	// Preserve inline URLs by replacing them with placeholders
	currentText = inlineURLRegex.ReplaceAllStringFunc(currentText, func(match string) string {
		placeholder := "INLINEURL" + strconv.Itoa(len(inlineURLs))
		inlineURLs[placeholder] = match
		return placeholder
	})

	// Handle image markdown by extracting and formatting URLs
	currentText = imageMarkdownRegex.ReplaceAllStringFunc(currentText, func(match string) string {
		return extractAndFormatImageURL(match)
	})

	// Format plain URLs
	currentText = urlRegex.ReplaceAllStringFunc(currentText, func(match string) string {
		// Skip if this URL is part of our preserved inline URLs
		for _, preserved := range inlineURLs {
			if strings.Contains(preserved, match) {
				return match
			}
		}
		return formatPlainURL(match)
	})

	// Escape special characters in the remaining text
	words := strings.Split(currentText, " ")
	for i, word := range words {
		// Skip if this is one of our placeholders
		if _, isPreserved := inlineURLs[word]; !isPreserved {
			// Skip if this word is already a formatted URL
			if !urlRegex.MatchString(word) {
				words[i] = escapeSpecialCharacters(word)
			}
		}
	}
	currentText = strings.Join(words, " ")

	// Restore preserved inline URLs
	for placeholder, original := range inlineURLs {
		currentText = strings.ReplaceAll(currentText, placeholder, original)
	}

	return currentText
}

// FormatMessage formats the input text according to Telegram MarkdownV2 rules
func FormatMessage(input string, formatOpts config.MessageFormatOptions) (string, error) {
	switch formatOpts.ParseMode {
	case "MarkdownV2":
		return formatMessageAsMarkdownV2(input), nil
	default:
		return "", fmt.Errorf("parse mode %s is not supported", formatOpts.ParseMode)
	}
}
