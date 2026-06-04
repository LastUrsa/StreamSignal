package app

import (
	"context"
	"strings"
	"testing"
	"time"

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
		DefaultStreamTitle:             "Late Night Music",
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
	if saved.DefaultStreamTitle != "Late Night Music" {
		t.Fatalf("expected stream title to round trip, got %+v", saved)
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

func TestAnnouncementProfileServiceMarksUsageAndPreservesItOnSave(t *testing.T) {
	db := openDestinationGroupServiceDB(t)
	ctx := context.Background()
	repository := sqlitestore.NewAnnouncementProfileRepository(db)
	service := NewAnnouncementProfileService(repository, sqlitestore.NewDestinationGroupRepository(db))
	usedAt := time.Date(2026, 6, 4, 12, 30, 0, 0, time.UTC)
	service.clock = fixedClock{now: usedAt}

	saved, err := service.Save(ctx, domain.AnnouncementProfile{
		Name:           "Music Stream",
		DefaultMessage: "Going live with music.",
	})
	if err != nil {
		t.Fatalf("save profile: %v", err)
	}

	if err := service.MarkUsed(ctx, saved.ID); err != nil {
		t.Fatalf("mark profile used: %v", err)
	}

	used, err := service.Get(ctx, saved.ID)
	if err != nil {
		t.Fatalf("get used profile: %v", err)
	}
	if used.LastUsedAt == nil || !used.LastUsedAt.Equal(usedAt) || used.UseCount != 1 {
		t.Fatalf("expected usage fields to be updated, got %+v", used)
	}

	used.Description = "Updated description"
	service.clock = fixedClock{now: usedAt.Add(time.Hour)}
	updated, err := service.Save(ctx, used)
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if updated.LastUsedAt == nil || !updated.LastUsedAt.Equal(usedAt) || updated.UseCount != 1 {
		t.Fatalf("expected usage fields to be preserved, got %+v", updated)
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
