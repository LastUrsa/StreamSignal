package domain

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	PreviewValidationValid   = "VALID"
	PreviewValidationWarning = "WARNING"
	PreviewValidationInvalid = "INVALID"
)

type ValidationSeverity string

const (
	ValidationSeverityWarning ValidationSeverity = "WARNING"
	ValidationSeverityError   ValidationSeverity = "ERROR"
)

type ValidationNote struct {
	Message  string
	Severity ValidationSeverity
}

var platformCharacterLimits = map[DestinationPlatform]int{
	PlatformDiscord:  2000,
	PlatformBluesky:  300,
	PlatformMastodon: 500,
}

func NormalizeAnnouncement(input Announcement, settings AppSettings) Announcement {
	return input
}

func ValidateAnnouncement(input Announcement) []string {
	return ValidationMessages(ValidateAnnouncementForTemplate(input, "{{stream_title}}{{stream_url}}"))
}

func ValidateAnnouncementForTemplate(input Announcement, template string) []ValidationNote {
	var notes []ValidationNote

	if templateUses(template, "{{stream_title}}") && strings.TrimSpace(input.StreamTitle) == "" {
		notes = append(notes, warningNote("Stream title is empty but used by this template."))
	}

	if templateUses(template, "{{stream_url}}") && strings.TrimSpace(input.StreamURL) == "" {
		notes = append(notes, warningNote("Stream URL is empty but used by this template."))
	} else if templateUses(template, "{{stream_url}}") && !isAbsoluteHTTPURL(strings.TrimSpace(input.StreamURL)) {
		notes = append(notes, errorNote("Stream URL must be a valid absolute URL."))
	}

	if templateUses(template, "{{category}}") && strings.TrimSpace(input.Category) == "" {
		notes = append(notes, warningNote("Category is empty but used by this template."))
	}

	if templateUses(template, "{{message}}") && strings.TrimSpace(input.Message) == "" {
		notes = append(notes, warningNote("Message is empty but used by this template."))
	}

	if templateUses(template, "{{hashtags}}") && strings.TrimSpace(input.Hashtags) == "" {
		notes = append(notes, warningNote("Hashtags are empty but used by this template."))
	}

	return notes
}

func templateUses(template string, token string) bool {
	return strings.Contains(template, token)
}

func isAbsoluteHTTPURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func ValidatePreviewContent(platform DestinationPlatform, content string) []ValidationNote {
	var notes []ValidationNote
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		notes = append(notes, errorNote("Rendered content is empty."))
	}

	limit, ok := platformCharacterLimits[platform]
	if ok && len(content) > limit {
		notes = append(notes, errorNote(fmt.Sprintf("Rendered content exceeds the %d character limit for %s.", limit, platform)))
	}

	return notes
}

func PreviewValidationState(notes []ValidationNote) string {
	if len(notes) == 0 {
		return PreviewValidationValid
	}
	if HasValidationErrors(notes) {
		return PreviewValidationInvalid
	}
	return PreviewValidationWarning
}

func ValidationMessages(notes []ValidationNote) []string {
	if len(notes) == 0 {
		return nil
	}
	messages := make([]string, 0, len(notes))
	for _, note := range notes {
		messages = append(messages, note.Message)
	}
	return messages
}

func HasValidationErrors(notes []ValidationNote) bool {
	for _, note := range notes {
		if note.Severity == ValidationSeverityError {
			return true
		}
	}
	return false
}

func FirstValidationError(notes []ValidationNote) (string, bool) {
	for _, note := range notes {
		if note.Severity == ValidationSeverityError {
			return note.Message, true
		}
	}
	return "", false
}

func ValidationError(message string) ValidationNote {
	return errorNote(message)
}

func warningNote(message string) ValidationNote {
	return ValidationNote{Message: message, Severity: ValidationSeverityWarning}
}

func errorNote(message string) ValidationNote {
	return ValidationNote{Message: message, Severity: ValidationSeverityError}
}
