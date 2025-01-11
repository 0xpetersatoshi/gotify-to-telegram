package telegram

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/0xPeterSatoshi/gotify-to-telegram/internal/api"
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

// escapeMarkdownV2 escapes all special characters in a text string
func escapeMarkdownV2(text string) string {
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
	return escapeMarkdownV2(url)
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
				words[i] = escapeMarkdownV2(word)
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

// formatTitle formats the title for Telegram
func formatTitle(msg api.Message) string {
	return fmt.Sprintf("[%s] %s", msg.AppName, msg.Title)
}

// formatExtras handles the recursive formatting of nested maps
func formatExtras(builder *strings.Builder, extras map[string]interface{}, prefix string) {
	// Get keys and sort them
	keys := make([]string, 0, len(extras))
	for key := range extras {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := extras[key]
		escapedKey := escapeMarkdownV2(key)

		// Handle nested maps
		if nestedMap, ok := value.(map[string]interface{}); ok {
			builder.WriteString(fmt.Sprintf("\n%s• %s:", prefix, escapedKey))
			formatExtras(builder, nestedMap, prefix+"  ") // Increase indentation for nested items
		} else {
			// Format simple values
			escapedValue := escapeMarkdownV2(fmt.Sprint(value))
			builder.WriteString(fmt.Sprintf("\n%s• %s: `%s`", prefix, escapedKey, escapedValue))
		}
	}

	builder.WriteString("\n\n")
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

// FormatMessage formats the input text according to Telegram MarkdownV2 rules
func FormatMessage(msg api.Message, formatOpts config.MessageFormatOptions) (string, error) {
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

	switch formatOpts.ParseMode {
	case "MarkdownV2":
		message := formatMessageAsMarkdownV2(msg.Message)
		builder.WriteString(message + "\n\n")
	default:
		return "", fmt.Errorf("parse mode %s is not supported", formatOpts.ParseMode)
	}

	// Priority indicator using emojis
	if int(msg.Priority) > formatOpts.PriorityThreshold && formatOpts.IncludePriority {
		builder.WriteString(escapeMarkdownV2(getPriorityIndicator(int(msg.Priority))) + "\n\n")
	}

	// Add any extras if present and not empty
	if len(msg.Extras) > 0 && formatOpts.IncludeExtras {
		builder.WriteString("*Additional Info:*")
		formatExtras(&builder, msg.Extras, "")
	}

	// Add timestamp
	if formatOpts.IncludeTimestamp {
		formattedTimestamp := time.Now().Format(time.RFC3339)
		builder.WriteString(fmt.Sprintf("timestamp: %s", escapeMarkdownV2(formattedTimestamp)) + "\n")
	}

	return builder.String(), nil
}
