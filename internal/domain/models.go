package domain

import "time"

type Announcement struct {
	StreamTitle        string    `json:"streamTitle"`
	StreamURL          string    `json:"streamURL"`
	Category           string    `json:"category"`
	Message            string    `json:"message"`
	Hashtags           string    `json:"hashtags"`
	DestinationGroupID string    `json:"destinationGroupID"`
	DestinationIDs     []string  `json:"destinationIDs"`
	CreatedAt          time.Time `json:"createdAt"`
}

type DestinationPlatform string

const (
	PlatformDiscord  DestinationPlatform = "discord"
	PlatformBluesky  DestinationPlatform = "bluesky"
	PlatformMastodon DestinationPlatform = "mastodon"
)

type Destination struct {
	ID         string              `json:"id"`
	Platform   DestinationPlatform `json:"platform"`
	Name       string              `json:"name"`
	Enabled    bool                `json:"enabled"`
	Template   string              `json:"template"`
	ConfigJSON string              `json:"configJSON"`
	CreatedAt  time.Time           `json:"createdAt"`
	UpdatedAt  time.Time           `json:"updatedAt"`
}

const (
	DefaultDestinationGroupID   = "all-destinations"
	DefaultDestinationGroupName = "All Destinations"
)

type DestinationGroup struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type DestinationGroupMembership struct {
	GroupID       string `json:"groupID"`
	DestinationID string `json:"destinationID"`
}

type AppSettings struct {
	DefaultStreamURL           string `json:"defaultStreamURL"`
	DefaultHashtags            string `json:"defaultHashtags"`
	DuplicateProtectionEnabled bool   `json:"duplicateProtectionEnabled"`
	DuplicateWindowMinutes     int    `json:"duplicateWindowMinutes"`
	SelectedDestinationGroupID string `json:"selectedDestinationGroupID"`
}

func DefaultAppSettings() AppSettings {
	return AppSettings{
		DefaultStreamURL:           "",
		DefaultHashtags:            "",
		DuplicateProtectionEnabled: true,
		DuplicateWindowMinutes:     10,
		SelectedDestinationGroupID: DefaultDestinationGroupID,
	}
}

type LogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Destination string    `json:"destination"`
	Action      string    `json:"action"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
}

type CredentialCheckResult struct {
	Platform string `json:"platform"`
	State    string `json:"state"`
	Message  string `json:"message"`
}

type PreviewItem struct {
	DestinationID   string              `json:"destinationID"`
	DestinationName string              `json:"destinationName"`
	Platform        DestinationPlatform `json:"platform"`
	PreviewLabel    string              `json:"previewLabel"`
	Content         string              `json:"content"`
	CharacterCount  int                 `json:"characterCount"`
	ValidationState string              `json:"validationState"`
	ValidationNotes []string            `json:"validationNotes"`
	LinkPreview     *LinkPreview        `json:"linkPreview,omitempty"`
}

type LinkPreview struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"imageURL"`
	SiteName    string `json:"siteName"`
}

type PostHistoryRecord struct {
	ID            int64     `json:"id"`
	DestinationID string    `json:"destinationID"`
	ContentHash   string    `json:"contentHash"`
	RenderedText  string    `json:"renderedText"`
	PostedAt      time.Time `json:"postedAt"`
}

type BlueskyLiveNowStatus struct {
	URL             string `json:"url"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	DurationMinutes int    `json:"durationMinutes"`
}

type BlueskyPostMetadata struct {
	StreamURL              string
	StreamTitle            string
	Description            string
	ThumbnailURL           string
	ThumbnailDataURL       string
	AdditionalImageURL     string
	AdditionalImageDataURL string
}

type DiscordPostMetadata struct {
	ThumbnailURL     string
	ThumbnailDataURL string
}

type MastodonPostMetadata struct {
	AdditionalImageURL     string
	AdditionalImageDataURL string
}

type ActiveLiveNowSession struct {
	DestinationID     string    `json:"destinationID"`
	DestinationName   string    `json:"destinationName"`
	Platform          string    `json:"platform"`
	AccountIdentifier string    `json:"accountIdentifier"`
	CredentialKey     string    `json:"credentialKey"`
	StreamURL         string    `json:"streamURL"`
	StreamTitle       string    `json:"streamTitle"`
	StartedAt         time.Time `json:"startedAt"`
}
