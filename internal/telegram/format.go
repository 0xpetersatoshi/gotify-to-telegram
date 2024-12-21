package telegram

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/api"
	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/config"
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

// formatTitle formats the title for Telegram
func formatTitle(msg api.Message) string {
	return fmt.Sprintf("[%s] %s", msg.AppName, msg.Title)
}

// formatExtras handles the recursive formatting of nested maps
func formatExtras(builder *strings.Builder, extras map[string]interface{}, prefix string, logger *zerolog.Logger) {
	for key, value := range extras {
		escapedKey := escapeMarkdownV2(key)

		// Handle nested maps
		if nestedMap, ok := value.(map[string]interface{}); ok {
			builder.WriteString(fmt.Sprintf("\n%s• %s:", prefix, escapedKey))
			formatExtras(builder, nestedMap, prefix+"  ", logger) // Increase indentation for nested items
		} else {
			// Format simple values
			escapedValue := escapeMarkdownV2(fmt.Sprint(value))
			builder.WriteString(fmt.Sprintf("\n%s• %s: `%s`", prefix, escapedKey, escapedValue))
		}
	}
}

// formatMessageForTelegram formats the message for Telegram
func formatMessageForTelegram(msg api.Message, formatOpts config.MessageFormatOptions, logger *zerolog.Logger) string {
	var (
		builder      strings.Builder
		messageTitle string
	)

	// Title in bold
	if msg.Title != "" {
		if formatOpts.IncludeAppName {
			messageTitle = formatTitle(msg)
		} else {
			messageTitle = msg.Title
		}
		builder.WriteString(fmt.Sprintf("*%s*\n\n", escapeMarkdownV2(messageTitle)))
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
	if int(msg.Priority) > formatOpts.PriorityThreshold && formatOpts.IncludePriority {
		builder.WriteString("\n")
		builder.WriteString(escapeMarkdownV2(getPriorityIndicator(int(msg.Priority))))
	}

	// Add any extras if present and not empty
	if len(msg.Extras) > 0 {
		builder.WriteString("\n\n*Additional Info:*")
		formatExtras(&builder, msg.Extras, "", logger)
	}

	// Add timestamp
	if formatOpts.IncludeTimestamp {
		builder.WriteString(fmt.Sprintf("\n\n*%s*", time.Now().Format(time.RFC3339)))
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
