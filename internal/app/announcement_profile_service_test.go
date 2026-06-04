package app

import (
	"context"
	"strings"
	"testing"

	"StreamSignal/internal/domain"
	sqlitestore "StreamSignal/internal/storage/sqlite"
)

func TestAnnouncementProfileServiceSavesAndValidatesProfiles(t *testing.T) {
	db := openDestinationGroupServiceDB(t)
	ctx := context.Background()
	service := NewAnnouncementProfileService(sqlitestore.NewAnnouncementProfileRepository(db), sqlitestore.NewDestinationGroupRepository(db))

	saved, err := service.Save(ctx, domain.AnnouncementProfile{
		Name:                           "Music Stream",
		Description:                    "Music announcements",
		DefaultStreamURL:               "https://example.com/live",
		DefaultCategory:                "Music",
		DefaultMessage:                 "Going live with music.",
		DefaultHashtags:                "#VTuber #Music",
		DefaultAnnouncementImageURL:    "https://example.com/promo.png",
		DefaultBlueskyCardThumbnailURL: "https://example.com/card.png",
		DefaultGroupID:                 domain.DefaultDestinationGroupID,
	})
	if err != nil {
		t.Fatalf("save profile: %v", err)
	}
	if saved.ID == "" {
		t.Fatal("expected generated profile id")
	}

	_, err = service.Save(ctx, domain.AnnouncementProfile{Name: " music stream "})
	if err == nil || !strings.Contains(err.Error(), "unique") {
		t.Fatalf("expected unique name error, got %v", err)
	}

	_, err = service.Save(ctx, domain.AnnouncementProfile{Name: "Bad URL", DefaultAnnouncementImageURL: "ftp://example.com/image.png"})
	if err == nil || !strings.Contains(err.Error(), "http or https") {
		t.Fatalf("expected image URL validation error, got %v", err)
	}

	_, err = service.Save(ctx, domain.AnnouncementProfile{Name: "Bad Group", DefaultGroupID: "missing-group"})
	if err == nil || !strings.Contains(err.Error(), "group") {
		t.Fatalf("expected missing group validation error, got %v", err)
	}
}

func TestAnnouncementProfileServiceDuplicatesProfiles(t *testing.T) {
	db := openDestinationGroupServiceDB(t)
	ctx := context.Background()
	service := NewAnnouncementProfileService(sqlitestore.NewAnnouncementProfileRepository(db), sqlitestore.NewDestinationGroupRepository(db))

	source, err := service.Save(ctx, domain.AnnouncementProfile{
		Name:                            "Music Stream",
		DefaultMessage:                  "Going live with music.",
		DefaultAnnouncementImageDataURL: "data:image/png;base64,ZmFrZQ==",
	})
	if err != nil {
		t.Fatalf("save source: %v", err)
	}

	clone, err := service.Duplicate(ctx, source.ID)
	if err != nil {
		t.Fatalf("duplicate profile: %v", err)
	}
	if clone.ID == source.ID {
		t.Fatal("expected clone id to differ from source")
	}
	if clone.Name != "Music Stream Copy" {
		t.Fatalf("unexpected clone name %q", clone.Name)
	}
	if clone.DefaultMessage != source.DefaultMessage || clone.DefaultAnnouncementImageDataURL != source.DefaultAnnouncementImageDataURL {
		t.Fatalf("expected duplicated defaults, got %+v", clone)
	}
}
