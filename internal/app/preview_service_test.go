package app

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"testing"
	"time"

	"StreamSignal/internal/domain"
)

type destinationListStub struct {
	items []domain.Destination
}

func (s *destinationListStub) List(context.Context) ([]domain.Destination, error) {
	return append([]domain.Destination(nil), s.items...), nil
}

func (s *destinationListStub) Get(_ context.Context, id string) (domain.Destination, error) {
	for _, item := range s.items {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.Destination{}, sql.ErrNoRows
}

func (s *destinationListStub) Save(context.Context, domain.Destination) error {
	panic("unexpected Save call")
}

func (s *destinationListStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

type settingsLoadStub struct {
	item domain.AppSettings
}

func (s *settingsLoadStub) Load(context.Context) (domain.AppSettings, error) {
	return s.item, nil
}

func (s *settingsLoadStub) Save(context.Context, domain.AppSettings) error {
	panic("unexpected Save call")
}

type previewSessionRepositoryStub struct {
	items []domain.ActiveLiveNowSession
}

func (s *previewSessionRepositoryStub) Upsert(context.Context, domain.ActiveLiveNowSession) error {
	panic("unexpected Upsert call")
}

func (s *previewSessionRepositoryStub) List(context.Context) ([]domain.ActiveLiveNowSession, error) {
	return append([]domain.ActiveLiveNowSession(nil), s.items...), nil
}

func (s *previewSessionRepositoryStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

type destinationGroupRepositoryStub struct {
	groupsByID          map[string]domain.DestinationGroup
	groupsByName        map[string]domain.DestinationGroup
	destinationsByGroup map[string][]domain.Destination
	groupIDsByDest      map[string][]string
}

func (s *destinationGroupRepositoryStub) List(context.Context) ([]domain.DestinationGroup, error) {
	items := make([]domain.DestinationGroup, 0, len(s.groupsByID))
	for _, item := range s.groupsByID {
		items = append(items, item)
	}
	return items, nil
}

func (s *destinationGroupRepositoryStub) Get(_ context.Context, id string) (domain.DestinationGroup, error) {
	if item, ok := s.groupsByID[id]; ok {
		return item, nil
	}
	return domain.DestinationGroup{}, sql.ErrNoRows
}

func (s *destinationGroupRepositoryStub) GetByName(_ context.Context, name string) (domain.DestinationGroup, error) {
	if item, ok := s.groupsByName[strings.ToLower(strings.TrimSpace(name))]; ok {
		return item, nil
	}
	return domain.DestinationGroup{}, sql.ErrNoRows
}

func (s *destinationGroupRepositoryStub) Save(_ context.Context, group domain.DestinationGroup) error {
	if s.groupsByID == nil {
		s.groupsByID = make(map[string]domain.DestinationGroup)
	}
	if s.groupsByName == nil {
		s.groupsByName = make(map[string]domain.DestinationGroup)
	}
	s.groupsByID[group.ID] = group
	s.groupsByName[strings.ToLower(strings.TrimSpace(group.Name))] = group
	return nil
}

func (s *destinationGroupRepositoryStub) Delete(_ context.Context, id string) error {
	delete(s.groupsByID, id)
	return nil
}

func (s *destinationGroupRepositoryStub) ListDestinations(_ context.Context, groupID string) ([]domain.Destination, error) {
	return append([]domain.Destination(nil), s.destinationsByGroup[groupID]...), nil
}

func (s *destinationGroupRepositoryStub) ListGroupIDsForDestination(_ context.Context, destinationID string) ([]string, error) {
	return append([]string(nil), s.groupIDsByDest[destinationID]...), nil
}

func (s *destinationGroupRepositoryStub) SetDestinations(_ context.Context, groupID string, destinationIDs []string) error {
	return nil
}

func (s *destinationGroupRepositoryStub) SetGroupsForDestination(_ context.Context, destinationID string, groupIDs []string) error {
	return nil
}

func (s *destinationGroupRepositoryStub) AddDestination(context.Context, string, string) error {
	return nil
}

func (s *destinationGroupRepositoryStub) RemoveDestination(context.Context, string, string) error {
	return nil
}

func newPreviewServiceForTest(destinations *destinationListStub, settings *settingsLoadStub, sessions *previewSessionRepositoryStub) *PreviewService {
	service := NewPreviewService(destinations, settings, sessions)
	service.linkPreview = nil
	return service
}

func TestPreviewServiceGeneratesItemsForSelectedDestinations(t *testing.T) {
	destinations := &destinationListStub{
		items: []domain.Destination{
			{
				ID:         "discord-main",
				Platform:   domain.PlatformDiscord,
				Name:       "Main Discord",
				Template:   "{{stream_title}} {{stream_url}} {{hashtags}}",
				ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main"}`,
			},
			{
				ID:         "bsky-disabled",
				Platform:   domain.PlatformBluesky,
				Name:       "Secondary Bluesky",
				Template:   "{{stream_title}}",
				ConfigJSON: `{"accountIdentifier":"streamer.test","credentialKey":"bluesky/main"}`,
			},
		},
	}
	settings := &settingsLoadStub{
		item: domain.AppSettings{
			DefaultStreamURL: "https://example.com/live",
			DefaultHashtags:  "#default",
		},
	}
	service := newPreviewServiceForTest(destinations, settings, &previewSessionRepositoryStub{})
	now := time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)
	service.clock = fixedClock{now: now}

	items, err := service.Generate(context.Background(), domain.Announcement{
		StreamTitle:    "Going Live",
		StreamURL:      "https://example.com/live",
		Hashtags:       "#explicit",
		DestinationIDs: []string{"discord-main"},
	})
	if err != nil {
		t.Fatalf("generate preview: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 preview item, got %d", len(items))
	}

	item := items[0]
	if item.Content != "Going Live https://example.com/live #explicit" {
		t.Fatalf("unexpected preview content %q", item.Content)
	}
	if item.ValidationState != domain.PreviewValidationValid {
		t.Fatalf("expected valid preview state, got %q with notes %v", item.ValidationState, item.ValidationNotes)
	}
}

func TestPreviewServiceGeneratesItemsForSelectedDestinationGroup(t *testing.T) {
	destinations := &destinationListStub{
		items: []domain.Destination{
			{
				ID:         "discord-main",
				Platform:   domain.PlatformDiscord,
				Name:       "Main Discord",
				Enabled:    true,
				Template:   "{{stream_title}}",
				ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main"}`,
			},
		},
	}
	settings := &settingsLoadStub{item: domain.DefaultAppSettings()}
	groups := &destinationGroupRepositoryStub{
		destinationsByGroup: map[string][]domain.Destination{
			"main-announcements": {
				{
					ID:         "discord-main",
					Platform:   domain.PlatformDiscord,
					Name:       "Main Discord",
					Enabled:    true,
					Template:   "{{stream_title}}",
					ConfigJSON: `{"webhookKey":"secret://streamsignal/destinations/discord-main/discord-webhook"}`,
				},
				{
					ID:         "discord-disabled",
					Platform:   domain.PlatformDiscord,
					Name:       "Disabled Discord",
					Enabled:    false,
					Template:   "{{stream_title}}",
					ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/disabled"}`,
				},
			},
		},
	}
	service := newPreviewServiceForTest(destinations, settings, &previewSessionRepositoryStub{})
	service.UseDestinationGroups(groups)
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	items, err := service.Generate(context.Background(), domain.Announcement{
		StreamTitle:        "Going Live",
		StreamURL:          "https://example.com/live",
		DestinationGroupID: "main-announcements",
	})
	if err != nil {
		t.Fatalf("generate preview: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one enabled group destination, got %+v", items)
	}
	if items[0].DestinationID != "discord-main" {
		t.Fatalf("expected enabled destination preview, got %+v", items[0])
	}
}

func TestPreviewServiceCarriesValidationNotesIntoPreviewItems(t *testing.T) {
	destinations := &destinationListStub{
		items: []domain.Destination{
			{
				ID:         "bluesky-main",
				Platform:   domain.PlatformBluesky,
				Name:       "Main Bluesky",
				Template:   "{{message}}",
				ConfigJSON: `{"accountIdentifier":"streamer.test","credentialKey":"bluesky/main"}`,
			},
		},
	}
	settings := &settingsLoadStub{}
	service := newPreviewServiceForTest(destinations, settings, &previewSessionRepositoryStub{})
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	items, err := service.Generate(context.Background(), domain.Announcement{})
	if err != nil {
		t.Fatalf("generate preview: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 preview item, got %d", len(items))
	}

	item := items[0]
	if item.ValidationState != domain.PreviewValidationInvalid {
		t.Fatalf("expected invalid preview state, got %q", item.ValidationState)
	}
	if !slices.Contains(item.ValidationNotes, "Message is required.") {
		t.Fatalf("expected message validation note, got %v", item.ValidationNotes)
	}
	if slices.Contains(item.ValidationNotes, "Stream title is required.") || slices.Contains(item.ValidationNotes, "Stream URL is required.") {
		t.Fatalf("expected unused title and url fields to be ignored, got %v", item.ValidationNotes)
	}
}

func TestPreviewServiceAttachesLinkPreviewMetadataForUnfurlPlatforms(t *testing.T) {
	destinations := &destinationListStub{
		items: []domain.Destination{
			{
				ID:         "discord-main",
				Platform:   domain.PlatformDiscord,
				Name:       "Main Discord",
				Template:   "{{stream_title}} {{stream_url}}",
				ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main"}`,
			},
			{
				ID:         "mastodon-main",
				Platform:   domain.PlatformMastodon,
				Name:       "Main Mastodon",
				Template:   "{{stream_title}} {{stream_url}}",
				ConfigJSON: `{"accountIdentifier":"@streamer@example.social","instanceURL":"https://example.social","credentialKey":"mastodon/main"}`,
			},
			{
				ID:         "bluesky-main",
				Platform:   domain.PlatformBluesky,
				Name:       "Main Bluesky",
				Template:   "{{stream_title}} {{stream_url}}",
				ConfigJSON: `{"accountIdentifier":"streamer.test","credentialKey":"bluesky/main"}`,
			},
		},
	}
	settings := &settingsLoadStub{}
	service := newPreviewServiceForTest(destinations, settings, &previewSessionRepositoryStub{})
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}
	fetchCount := 0
	service.linkPreview = func(_ context.Context, streamURL string) (*domain.LinkPreview, error) {
		fetchCount++
		if streamURL != "https://example.com/live" {
			t.Fatalf("unexpected stream URL %q", streamURL)
		}
		return &domain.LinkPreview{
			URL:         streamURL,
			Title:       "Going Live",
			Description: "Preview description",
			ImageURL:    "https://example.com/card.png",
			SiteName:    "Example",
		}, nil
	}

	items, err := service.Generate(context.Background(), domain.Announcement{
		StreamTitle: "Going Live",
		StreamURL:   "https://example.com/live",
	})
	if err != nil {
		t.Fatalf("generate preview: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 preview items, got %d", len(items))
	}
	if fetchCount != 1 {
		t.Fatalf("expected link preview to be cached, fetched %d times", fetchCount)
	}
	if items[0].LinkPreview == nil || items[0].LinkPreview.Title != "Going Live" {
		t.Fatalf("expected Discord link preview metadata, got %+v", items[0].LinkPreview)
	}
	if items[1].LinkPreview == nil || items[1].LinkPreview.ImageURL != "https://example.com/card.png" {
		t.Fatalf("expected Mastodon link preview metadata, got %+v", items[1].LinkPreview)
	}
	if items[2].LinkPreview != nil {
		t.Fatalf("expected Bluesky to keep its existing card approximation path, got %+v", items[2].LinkPreview)
	}
}

func TestPreviewServiceSkipsBlueskyLiveStatusValidationWhenDisabled(t *testing.T) {
	destinations := &destinationListStub{
		items: []domain.Destination{
			{
				ID:         "bluesky-main",
				Platform:   domain.PlatformBluesky,
				Name:       "Main Bluesky",
				Template:   "{{message}}",
				ConfigJSON: `{"accountIdentifier":"streamer.test","credentialKey":"bluesky/main","liveStatusEnabled":false}`,
			},
		},
	}
	settings := &settingsLoadStub{}
	service := newPreviewServiceForTest(destinations, settings, &previewSessionRepositoryStub{})
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	items, err := service.Generate(context.Background(), domain.Announcement{
		Message: "Going live soon",
	})
	if err != nil {
		t.Fatalf("generate preview: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 preview item, got %d", len(items))
	}
	item := items[0]
	if item.ValidationState != domain.PreviewValidationValid {
		t.Fatalf("expected valid preview, got %q with notes %v", item.ValidationState, item.ValidationNotes)
	}
	if slices.Contains(item.ValidationNotes, "Bluesky Live Now URL must be a valid absolute URL") || slices.Contains(item.ValidationNotes, "Bluesky Live Now title is required") {
		t.Fatalf("expected Live Status validation to be skipped, got %v", item.ValidationNotes)
	}
}

func TestPreviewServiceAppliesBlueskyLiveStatusValidationWhenEnabled(t *testing.T) {
	destinations := &destinationListStub{
		items: []domain.Destination{
			{
				ID:         "bluesky-main",
				Platform:   domain.PlatformBluesky,
				Name:       "Main Bluesky",
				Template:   "{{message}}",
				ConfigJSON: `{"accountIdentifier":"streamer.test","credentialKey":"bluesky/main","liveStatusEnabled":true}`,
			},
		},
	}
	settings := &settingsLoadStub{}
	service := newPreviewServiceForTest(destinations, settings, &previewSessionRepositoryStub{})
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	items, err := service.Generate(context.Background(), domain.Announcement{
		Message: "Going live soon",
	})
	if err != nil {
		t.Fatalf("generate preview: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 preview item, got %d", len(items))
	}
	item := items[0]
	if item.ValidationState != domain.PreviewValidationInvalid {
		t.Fatalf("expected invalid preview, got %q with notes %v", item.ValidationState, item.ValidationNotes)
	}
	if !slices.Contains(item.ValidationNotes, "Bluesky Live Now URL must be a valid absolute URL") {
		t.Fatalf("expected Live Status URL validation, got %v", item.ValidationNotes)
	}
}

func TestPreviewServiceIncludesEnabledEndStreamMessages(t *testing.T) {
	destinations := &destinationListStub{
		items: []domain.Destination{
			{
				ID:         "bluesky-main",
				Platform:   domain.PlatformBluesky,
				Name:       "Main Bluesky",
				Template:   "{{stream_title}} {{stream_url}}",
				ConfigJSON: `{"accountIdentifier":"streamer.test","credentialKey":"bluesky/main","endStreamEnabled":true,"endStreamTemplate":"Thanks for watching {{stream_title}}"}`,
			},
		},
	}
	settings := &settingsLoadStub{}
	sessions := &previewSessionRepositoryStub{
		items: []domain.ActiveLiveNowSession{{
			DestinationID: "bluesky-main",
			StreamTitle:   "Going Live",
			StreamURL:     "https://example.com/live",
		}},
	}
	service := newPreviewServiceForTest(destinations, settings, sessions)
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	items, err := service.Generate(context.Background(), domain.Announcement{
		StreamTitle:    "Going Live",
		StreamURL:      "https://example.com/live",
		DestinationIDs: []string{"bluesky-main"},
	})
	if err != nil {
		t.Fatalf("generate preview: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected go live and end stream previews, got %d: %+v", len(items), items)
	}
	if items[0].PreviewLabel != "Go Live" || items[0].Content != "Going Live https://example.com/live" {
		t.Fatalf("unexpected go live preview: %+v", items[0])
	}
	if items[1].PreviewLabel != "End Stream" || items[1].Content != "Thanks for watching Going Live" {
		t.Fatalf("unexpected end stream preview: %+v", items[1])
	}
}

func TestPreviewServiceEndStreamUsesCurrentAnnouncementFields(t *testing.T) {
	destinations := &destinationListStub{
		items: []domain.Destination{
			{
				ID:         "discord-main",
				Platform:   domain.PlatformDiscord,
				Name:       "Main Discord",
				Template:   "{{stream_title}} {{stream_url}}",
				ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main","endStreamEnabled":true,"endStreamTemplate":"Thanks for hanging out at {{stream_title}} {{stream_url}}"}`,
			},
		},
	}
	settings := &settingsLoadStub{}
	service := newPreviewServiceForTest(destinations, settings, &previewSessionRepositoryStub{})
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	items, err := service.Generate(context.Background(), domain.Announcement{
		StreamTitle:    "Testy Test",
		StreamURL:      "https://twitch.tv/lastursa",
		DestinationIDs: []string{"discord-main"},
	})
	if err != nil {
		t.Fatalf("generate preview: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected go live and end stream previews, got %d: %+v", len(items), items)
	}

	item := items[1]
	if item.PreviewLabel != "End Stream" {
		t.Fatalf("expected end stream preview, got %+v", item)
	}
	if item.Content != "Thanks for hanging out at Testy Test https://twitch.tv/lastursa" {
		t.Fatalf("unexpected end stream content %q", item.Content)
	}
	if item.ValidationState != domain.PreviewValidationValid {
		t.Fatalf("expected valid end stream preview, got %q with notes %v", item.ValidationState, item.ValidationNotes)
	}
	if slices.Contains(item.ValidationNotes, "Stream title is required.") {
		t.Fatalf("expected current stream title to satisfy validation, got %v", item.ValidationNotes)
	}
}

func TestPreviewServiceEndStreamValidationOnlyRequiresTemplateFields(t *testing.T) {
	destinations := &destinationListStub{
		items: []domain.Destination{
			{
				ID:         "discord-main",
				Platform:   domain.PlatformDiscord,
				Name:       "Main Discord",
				Template:   "{{stream_title}} {{stream_url}}",
				ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main","endStreamEnabled":true,"endStreamTemplate":"Stream ended. Thanks for watching."}`,
			},
		},
	}
	settings := &settingsLoadStub{}
	service := newPreviewServiceForTest(destinations, settings, &previewSessionRepositoryStub{})
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	items, err := service.GenerateEndStream(context.Background(), domain.Announcement{
		DestinationIDs: []string{"discord-main"},
	})
	if err != nil {
		t.Fatalf("generate end stream preview: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 preview item, got %d", len(items))
	}

	item := items[0]
	if item.ValidationState != domain.PreviewValidationValid {
		t.Fatalf("expected valid end stream preview, got %q with notes %v", item.ValidationState, item.ValidationNotes)
	}
	if slices.Contains(item.ValidationNotes, "Stream title is required.") || slices.Contains(item.ValidationNotes, "Stream URL is required.") {
		t.Fatalf("expected unused title and url fields to be ignored, got %v", item.ValidationNotes)
	}
}

func TestPreviewServiceIncludesDestinationSetupValidation(t *testing.T) {
	destinations := &destinationListStub{
		items: []domain.Destination{
			{
				ID:         "discord-main",
				Platform:   domain.PlatformDiscord,
				Name:       "Main Discord",
				Template:   "{{stream_title}}",
				ConfigJSON: `{"webhookKey":"not a url"}`,
			},
		},
	}
	settings := &settingsLoadStub{}
	service := newPreviewServiceForTest(destinations, settings, &previewSessionRepositoryStub{})
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	items, err := service.Generate(context.Background(), domain.Announcement{
		StreamTitle: "Going Live",
		StreamURL:   "https://example.com/live",
	})
	if err != nil {
		t.Fatalf("generate preview: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 preview item, got %d", len(items))
	}

	item := items[0]
	if item.ValidationState != domain.PreviewValidationInvalid {
		t.Fatalf("expected invalid preview state, got %q", item.ValidationState)
	}
	if len(item.ValidationNotes) != 1 || item.ValidationNotes[0] != "Discord webhook key must be a valid absolute URL" {
		t.Fatalf("expected destination setup validation note, got %v", item.ValidationNotes)
	}
}

func TestPreviewServiceGeneratesEndStreamItemsFromDestinationConfig(t *testing.T) {
	destinations := &destinationListStub{
		items: []domain.Destination{
			{
				ID:         "bluesky-main",
				Platform:   domain.PlatformBluesky,
				Name:       "Main Bluesky",
				Template:   "{{stream_title}}",
				ConfigJSON: `{"accountIdentifier":"streamer.test","credentialKey":"bluesky/main","endStreamEnabled":true,"endStreamTemplate":"Thanks for watching {{stream_title}} {{stream_url}}"}`,
			},
			{
				ID:         "discord-main",
				Platform:   domain.PlatformDiscord,
				Name:       "Main Discord",
				Template:   "{{stream_title}}",
				ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main","endStreamEnabled":false,"endStreamTemplate":"Bye"}`,
			},
		},
	}
	settings := &settingsLoadStub{}
	sessions := &previewSessionRepositoryStub{
		items: []domain.ActiveLiveNowSession{{
			DestinationID: "bluesky-main",
			StreamTitle:   "Going Live",
			StreamURL:     "https://example.com/live",
		}},
	}
	service := newPreviewServiceForTest(destinations, settings, sessions)
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	items, err := service.GenerateEndStream(context.Background(), domain.Announcement{
		DestinationIDs: []string{"bluesky-main", "discord-main"},
	})
	if err != nil {
		t.Fatalf("generate end stream preview: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 preview items, got %d", len(items))
	}
	if items[0].Content != "Thanks for watching Going Live https://example.com/live" {
		t.Fatalf("unexpected end stream content %q", items[0].Content)
	}
	if !slices.Contains(items[1].ValidationNotes, "End Stream message disabled for this destination.") {
		t.Fatalf("expected disabled note, got %v", items[1].ValidationNotes)
	}
}
