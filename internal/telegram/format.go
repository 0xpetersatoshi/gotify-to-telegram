package telegram

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/api"
	"github.com/rs/zerolog"
)

// Regular expression to find markdown links
var linkRegex = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

// processMarkdownLinks processes markdown links in the given text
func processMarkdownLinks(text string, logger *zerolog.Logger) string {
	// Process all matches at once using regex
	return linkRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Extract link text and URL using regex submatches
		submatches := linkRegex.FindStringSubmatch(match)
		if len(submatches) != 3 {
			logger.Error().Msg("Invalid link format found")
			return escapeMarkdownV2(match)
		}

		linkText := submatches[1]
		url := submatches[2]

		// Escape the link text and URL separately
		escapedLinkText := escapeMarkdownV2(linkText)
		escapedURL := escapeURLForMarkdown(url)

		// Return the properly formatted markdown link
		return fmt.Sprintf("[%s](%s)", escapedLinkText, escapedURL)
	})
}

// escapeURLForMarkdown escapes specific characters in URLs
func escapeURLForMarkdown(url string) string {
	// Escape only specific characters in URLs
	// Note: We need to be more selective about what we escape in URLs
	specialChars := []string{")", "("}
	result := url
	for _, char := range specialChars {
		result = strings.ReplaceAll(result, char, "\\"+char)
	}
	return result
}

// escapeMarkdownV2 escapes specific characters in MarkdownV2
func escapeMarkdownV2(s string) string {
	// Characters that need escaping in MarkdownV2
	specialChars := []string{
		"_", "*", "[", "]", "(", ")", "~", "`", ">",
		"#", "+", "-", "=", "|", "{", "}", ".", "!",
	}

	result := s
	for _, char := range specialChars {
		result = strings.ReplaceAll(result, char, "\\"+char)
	}
	return result
}

// convertImageMarkdownToURL converts image markdown to plain URLs
func convertImageMarkdownToURL(message string, logger *zerolog.Logger) string {
	imgRegex := regexp.MustCompile(`!\[\]\((.*?)\)`)
	return imgRegex.ReplaceAllStringFunc(message, func(match string) string {
		submatches := imgRegex.FindStringSubmatch(match)
		if len(submatches) != 2 {
			logger.Error().Msg("Invalid image markdown format")
			return match
		}
		return submatches[1] // Return just the URL
	})
}

// formatMessageForTelegram formats the message for Telegram
func formatMessageForTelegram(msg api.Message, logger *zerolog.Logger) string {
	var builder strings.Builder

	// Title in bold
	if msg.Title != "" {
		builder.WriteString(fmt.Sprintf("*%s*\n\n", escapeMarkdownV2(msg.Title)))
	}

	// Process the message content
	messageContent := msg.Message

	// First convert any image markdown to plain URLs
	if strings.Contains(messageContent, "![](") {
		messageContent = convertImageMarkdownToURL(messageContent, logger)
	}

	// Then handle any regular markdown links
	messageContent = processMarkdownLinks(messageContent, logger)

	// Escape any remaining special characters in the message
	// But avoid escaping the already processed links
	parts := strings.Split(messageContent, "\n")
	for i, part := range parts {
		if !strings.Contains(part, "](") { // Only escape lines that don't contain links
			parts[i] = escapeMarkdownV2(part)
		}
	}
	messageContent = strings.Join(parts, "\n")

	builder.WriteString(messageContent + "\n")

	// Priority indicator using emojis
	if msg.Priority > 0 {
		builder.WriteString("\n")
		builder.WriteString(escapeMarkdownV2(getPriorityIndicator(int(msg.Priority))))
	}

	// Application ID
	builder.WriteString(fmt.Sprintf("\n\n`App ID: %d`", msg.Appid))

	// Add any extras if present and not empty
	if len(msg.Extras) > 0 {
		builder.WriteString("\n\n*Additional Info:*")
		for key, value := range msg.Extras {
			escapedKey := escapeMarkdownV2(key)
			escapedValue := escapeMarkdownV2(fmt.Sprint(value))
			builder.WriteString(fmt.Sprintf("\n• %s: `%s`", escapedKey, escapedValue))
		}
	}

	return builder.String()
}

// getPriorityIndicator returns the emoji indicator for the priority
func getPriorityIndicator(priority int) string {
	switch {
	case priority >= 8:
		return "🔴 Critical Priority"
	case priority >= 6:
		return "🟠 High Priority"
	case priority >= 4:
		return "🟡 Medium Priority"
	default:
		return "🟢 Low Priority"
	}
}
