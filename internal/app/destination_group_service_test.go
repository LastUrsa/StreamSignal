package app

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"StreamSignal/internal/domain"
	sqlitestore "StreamSignal/internal/storage/sqlite"
)

func TestDestinationGroupServiceValidatesUniqueNames(t *testing.T) {
	db := openDestinationGroupServiceDB(t)
	service := NewDestinationGroupService(sqlitestore.NewDestinationGroupRepository(db))
	ctx := context.Background()

	first, err := service.Save(ctx, domain.DestinationGroup{Name: "Main Announcements"})
	if err != nil {
		t.Fatalf("save first group: %v", err)
	}
	if first.ID == "" {
		t.Fatal("expected generated group id")
	}

	_, err = service.Save(ctx, domain.DestinationGroup{Name: " main announcements "})
	if err == nil || !strings.Contains(err.Error(), "unique") {
		t.Fatalf("expected unique name error, got %v", err)
	}
}

func TestDestinationGroupServiceDuplicatesMembership(t *testing.T) {
	db := openDestinationGroupServiceDB(t)
	ctx := context.Background()
	destinationRepository := sqlitestore.NewDestinationRepository(db)
	groupRepository := sqlitestore.NewDestinationGroupRepository(db)
	service := NewDestinationGroupService(groupRepository)
	now := time.Now().UTC().Round(time.Microsecond)

	destination := domain.Destination{
		ID:         "discord-main",
		Platform:   domain.PlatformDiscord,
		Name:       "Main Discord",
		Enabled:    true,
		Template:   "{{stream_title}}",
		ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/main"}`,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := destinationRepository.Save(ctx, destination); err != nil {
		t.Fatalf("save destination: %v", err)
	}

	group, err := service.Save(ctx, domain.DestinationGroup{Name: "Main Announcements"})
	if err != nil {
		t.Fatalf("save group: %v", err)
	}
	if err := service.SetDestinations(ctx, group.ID, []string{destination.ID}); err != nil {
		t.Fatalf("set destinations: %v", err)
	}

	clone, err := service.Duplicate(ctx, group.ID)
	if err != nil {
		t.Fatalf("duplicate group: %v", err)
	}
	if clone.Name != "Main Announcements Copy" {
		t.Fatalf("unexpected clone name %q", clone.Name)
	}
	destinations, err := service.GetDestinations(ctx, clone.ID)
	if err != nil {
		t.Fatalf("get clone destinations: %v", err)
	}
	if len(destinations) != 1 || destinations[0].ID != destination.ID {
		t.Fatalf("expected cloned membership, got %+v", destinations)
	}
}

func TestDestinationGroupServiceBlocksDefaultGroupDeletion(t *testing.T) {
	db := openDestinationGroupServiceDB(t)
	service := NewDestinationGroupService(sqlitestore.NewDestinationGroupRepository(db))

	err := service.Delete(context.Background(), domain.DefaultDestinationGroupID)
	if err == nil || !strings.Contains(err.Error(), "cannot be deleted") {
		t.Fatalf("expected default group deletion error, got %v", err)
	}
}

func openDestinationGroupServiceDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sqlitestore.OpenInMemory()
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}
