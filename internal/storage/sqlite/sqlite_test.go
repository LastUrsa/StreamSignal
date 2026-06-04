package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"StreamSignal/internal/domain"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func TestApplyMigrationsIsIdempotent(t *testing.T) {
	db := openTestDB(t)

	if err := ApplyMigrations(db); err != nil {
		t.Fatalf("reapply migrations: %v", err)
	}

	var count int
	if err := db.QueryRow(`select count(*) from schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("count migrations: %v", err)
	}

	if count != len(migrations) {
		t.Fatalf("expected %d migrations, got %d", len(migrations), count)
	}
}

func TestSettingsRepositoryReturnsDefaultsWhenUnset(t *testing.T) {
	db := openTestDB(t)
	repository := NewSettingsRepository(db)

	settings, err := repository.Load(context.Background())
	if err != nil {
		t.Fatalf("load default settings: %v", err)
	}

	expected := domain.DefaultAppSettings()
	if settings != expected {
		t.Fatalf("expected default settings %+v, got %+v", expected, settings)
	}
}

func TestSettingsRepositoryPersistsRoundTrip(t *testing.T) {
	db := openTestDB(t)
	repository := NewSettingsRepository(db)

	expected := domain.AppSettings{
		DefaultStreamURL:           "https://example.com/live",
		DefaultHashtags:            "#vtuber #music",
		DuplicateProtectionEnabled: true,
		DuplicateWindowMinutes:     15,
		SelectedDestinationGroupID: "main-announcements",
	}

	if err := repository.Save(context.Background(), expected); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	actual, err := repository.Load(context.Background())
	if err != nil {
		t.Fatalf("load saved settings: %v", err)
	}

	if actual != expected {
		t.Fatalf("expected settings %+v, got %+v", expected, actual)
	}
}

func TestDestinationGroupMigrationCreatesDefaultGroup(t *testing.T) {
	db := openTestDB(t)
	repository := NewDestinationGroupRepository(db)

	groups, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("list destination groups: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected default destination group, got %+v", groups)
	}
	if groups[0].ID != domain.DefaultDestinationGroupID || groups[0].Name != domain.DefaultDestinationGroupName {
		t.Fatalf("unexpected default group: %+v", groups[0])
	}
}

func TestDestinationGroupRepositoryMembershipRoundTrip(t *testing.T) {
	db := openTestDB(t)
	destinationRepository := NewDestinationRepository(db)
	groupRepository := NewDestinationGroupRepository(db)
	now := time.Now().UTC().Round(time.Microsecond)

	first := domain.Destination{
		ID:         "discord-main",
		Platform:   domain.PlatformDiscord,
		Name:       "Main Discord",
		Enabled:    true,
		Template:   "{{stream_title}}",
		ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/main"}`,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	second := domain.Destination{
		ID:         "bluesky-main",
		Platform:   domain.PlatformBluesky,
		Name:       "Main Bluesky",
		Enabled:    true,
		Template:   "{{stream_title}}",
		ConfigJSON: `{"accountIdentifier":"streamer.test","credentialKey":"bluesky/main"}`,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := destinationRepository.Save(context.Background(), first); err != nil {
		t.Fatalf("save first destination: %v", err)
	}
	if err := destinationRepository.Save(context.Background(), second); err != nil {
		t.Fatalf("save second destination: %v", err)
	}

	group := domain.DestinationGroup{
		ID:          "main-announcements",
		Name:        "Main Announcements",
		Description: "Primary live announcement targets",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := groupRepository.Save(context.Background(), group); err != nil {
		t.Fatalf("save group: %v", err)
	}
	if err := groupRepository.SetDestinations(context.Background(), group.ID, []string{first.ID, second.ID, first.ID}); err != nil {
		t.Fatalf("set group destinations: %v", err)
	}

	destinations, err := groupRepository.ListDestinations(context.Background(), group.ID)
	if err != nil {
		t.Fatalf("list group destinations: %v", err)
	}
	if len(destinations) != 2 {
		t.Fatalf("expected two group destinations, got %+v", destinations)
	}

	groupIDs, err := groupRepository.ListGroupIDsForDestination(context.Background(), first.ID)
	if err != nil {
		t.Fatalf("list groups for destination: %v", err)
	}
	if len(groupIDs) != 1 || groupIDs[0] != group.ID {
		t.Fatalf("expected destination in group %q, got %+v", group.ID, groupIDs)
	}

	if err := groupRepository.SetGroupsForDestination(context.Background(), first.ID, []string{domain.DefaultDestinationGroupID}); err != nil {
		t.Fatalf("set groups for destination: %v", err)
	}
	groupIDs, err = groupRepository.ListGroupIDsForDestination(context.Background(), first.ID)
	if err != nil {
		t.Fatalf("list groups after destination update: %v", err)
	}
	if len(groupIDs) != 1 || groupIDs[0] != domain.DefaultDestinationGroupID {
		t.Fatalf("expected destination in default group, got %+v", groupIDs)
	}
}

func TestAnnouncementProfileRepositoryListsRecentlyUsedProfilesFirst(t *testing.T) {
	db := openTestDB(t)
	repository := NewAnnouncementProfileRepository(db)
	ctx := context.Background()
	now := time.Date(2026, 6, 4, 14, 0, 0, 0, time.UTC)

	for _, profile := range []domain.AnnouncementProfile{
		{ID: "alpha", Name: "Alpha", DefaultStreamTitle: "Alpha Title", CreatedAt: now, UpdatedAt: now},
		{ID: "beta", Name: "Beta", DefaultStreamTitle: "Beta Title", CreatedAt: now, UpdatedAt: now},
		{ID: "gamma", Name: "Gamma", DefaultStreamTitle: "Gamma Title", CreatedAt: now, UpdatedAt: now},
	} {
		if err := repository.Save(ctx, profile); err != nil {
			t.Fatalf("save profile %q: %v", profile.ID, err)
		}
	}

	if err := repository.MarkUsed(ctx, "beta", now.Add(time.Hour)); err != nil {
		t.Fatalf("mark beta used: %v", err)
	}
	if err := repository.MarkUsed(ctx, "gamma", now.Add(2*time.Hour)); err != nil {
		t.Fatalf("mark gamma used: %v", err)
	}

	profiles, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list profiles: %v", err)
	}

	got := []string{profiles[0].ID, profiles[1].ID, profiles[2].ID}
	expected := []string{"gamma", "beta", "alpha"}
	for index := range expected {
		if got[index] != expected[index] {
			t.Fatalf("expected order %+v, got %+v", expected, got)
		}
	}
	if profiles[0].LastUsedAt == nil || profiles[0].UseCount != 1 {
		t.Fatalf("expected usage fields on first profile, got %+v", profiles[0])
	}
	if profiles[0].DefaultStreamTitle != "Gamma Title" {
		t.Fatalf("expected stream title to round trip, got %+v", profiles[0])
	}
}

func TestDestinationRepositoryCRUD(t *testing.T) {
	db := openTestDB(t)
	repository := NewDestinationRepository(db)
	now := time.Now().UTC().Round(time.Microsecond)

	expected := domain.Destination{
		ID:         "discord-main",
		Platform:   domain.PlatformDiscord,
		Name:       "Main Discord",
		Enabled:    true,
		Template:   "{{stream_title}}",
		ConfigJSON: `{"webhookKey":"discord/main"}`,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := repository.Save(context.Background(), expected); err != nil {
		t.Fatalf("save destination: %v", err)
	}

	actual, err := repository.Get(context.Background(), expected.ID)
	if err != nil {
		t.Fatalf("get destination: %v", err)
	}

	if actual != expected {
		t.Fatalf("expected destination %+v, got %+v", expected, actual)
	}

	list, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("list destinations: %v", err)
	}

	if len(list) != 1 {
		t.Fatalf("expected 1 destination, got %d", len(list))
	}

	if err := repository.Delete(context.Background(), expected.ID); err != nil {
		t.Fatalf("delete destination: %v", err)
	}

	_, err = repository.Get(context.Background(), expected.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestLiveNowSessionRepositoryRoundTrip(t *testing.T) {
	db := openTestDB(t)
	repository := NewLiveNowSessionRepository(db)
	startedAt := time.Date(2026, 5, 31, 21, 0, 0, 0, time.UTC)

	expected := domain.ActiveLiveNowSession{
		DestinationID:     "bluesky-main",
		DestinationName:   "Main Bluesky",
		Platform:          string(domain.PlatformBluesky),
		AccountIdentifier: "streamer.main",
		CredentialKey:     "bluesky/main",
		StreamURL:         "https://example.com/live",
		StreamTitle:       "Going Live",
		StartedAt:         startedAt,
	}

	if err := repository.Upsert(context.Background(), expected); err != nil {
		t.Fatalf("upsert session: %v", err)
	}

	sessions, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0] != expected {
		t.Fatalf("expected one session %+v, got %+v", expected, sessions)
	}

	if err := repository.Delete(context.Background(), expected.DestinationID); err != nil {
		t.Fatalf("delete session: %v", err)
	}

	sessions, err = repository.List(context.Background())
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected no sessions after delete, got %+v", sessions)
	}
}

func TestLogRepositoryAppendAndListRecent(t *testing.T) {
	db := openTestDB(t)
	repository := NewLogRepository(db)
	now := time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)

	first := domain.LogEntry{
		Timestamp:   now,
		Destination: "preview",
		Action:      "generate_preview",
		Status:      "SUCCESS",
		Message:     "Generated preview.",
	}
	second := domain.LogEntry{
		Timestamp:   now.Add(time.Minute),
		Destination: "go_live",
		Action:      "go_live",
		Status:      "WARNING",
		Message:     "Live Now clear pending.",
	}

	if err := repository.Append(context.Background(), first); err != nil {
		t.Fatalf("append first log: %v", err)
	}
	if err := repository.Append(context.Background(), second); err != nil {
		t.Fatalf("append second log: %v", err)
	}

	entries, err := repository.ListRecent(context.Background(), 1)
	if err != nil {
		t.Fatalf("list recent logs: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one recent log entry, got %d", len(entries))
	}
	if entries[0] != second {
		t.Fatalf("expected most recent log %+v, got %+v", second, entries[0])
	}
}

func TestPostHistoryRepositoryRecordAndFindRecentByDestination(t *testing.T) {
	db := openTestDB(t)
	repository := NewPostHistoryRepository(db)
	now := time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)

	if err := repository.Record(context.Background(), "discord-main", "Going Live", "hash-1", now); err != nil {
		t.Fatalf("record first history item: %v", err)
	}
	if err := repository.Record(context.Background(), "discord-main", "Going Live Again", "hash-2", now.Add(2*time.Minute)); err != nil {
		t.Fatalf("record second history item: %v", err)
	}
	if err := repository.Record(context.Background(), "bluesky-main", "Different Destination", "hash-3", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("record third history item: %v", err)
	}

	records, err := repository.FindRecentByDestination(context.Background(), "discord-main", now.Add(30*time.Second))
	if err != nil {
		t.Fatalf("find recent history: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected one recent record, got %d", len(records))
	}
	if records[0].DestinationID != "discord-main" || records[0].ContentHash != "hash-2" {
		t.Fatalf("unexpected record: %+v", records[0])
	}
	if !records[0].PostedAt.Equal(now.Add(2 * time.Minute)) {
		t.Fatalf("expected posted at %v, got %v", now.Add(2*time.Minute), records[0].PostedAt)
	}
}
