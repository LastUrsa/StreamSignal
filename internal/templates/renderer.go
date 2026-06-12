package templates

import (
	"sort"
	"strings"
	"time"

	"StreamSignal/internal/domain"
)

func Render(template string, announcement domain.Announcement, platform domain.DestinationPlatform, now time.Time) string {
	replacements := map[string]string{
		"{{stream_title}}": announcement.StreamTitle,
		"{{stream_url}}":   announcement.StreamURL,
		"{{category}}":     announcement.Category,
		"{{message}}":      announcement.Message,
		"{{hashtags}}":     announcement.Hashtags,
		"{{date}}":         now.Format("2006-01-02"),
		"{{time}}":         now.Format("15:04"),
		"{{platform}}":     string(platform),
	}

	rendered := template
	for key, value := range replacements {
		rendered = strings.ReplaceAll(rendered, key, value)
	}
	if len(announcement.Fields) > 0 {
		fieldIDs := make([]string, 0, len(announcement.Fields))
		for id := range announcement.Fields {
			fieldIDs = append(fieldIDs, id)
		}
		sort.Strings(fieldIDs)
		for _, id := range fieldIDs {
			value := announcement.Fields[id]
			normalizedID := strings.TrimSpace(strings.ToLower(id))
			if normalizedID == "" {
				continue
			}
			rendered = strings.ReplaceAll(rendered, "{"+normalizedID+"}", value)
		}
	}

	return rendered
}
