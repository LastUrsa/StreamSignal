//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	appsvc "StreamSignal/internal/app"
	"StreamSignal/internal/domain"
	"StreamSignal/internal/platforms/bluesky"
	"StreamSignal/internal/platforms/mastodon"
	"StreamSignal/internal/platforms/unavailable"
	"StreamSignal/internal/ports"
)

const e2eGroupID = "e2e-test-group"

func TestWorkflowPublishesToRealTestDestinationsAndCleansUp(t *testing.T) {
	if os.Getenv("STREAMSIGNAL_E2E") != "1" {
		t.Skip("set STREAMSIGNAL_E2E=1 to run provider-backed workflow E2E tests")
	}

	destinations := e2eDestinations(t)
	if len(destinations) == 0 {
		t.Skip("set at least one E2E provider credential to run workflow E2E tests")
	}
	t.Logf("E2E workflow will publish to %d destination(s): %s", len(destinations), destinationSummary(destinations))

	destinationRepo := &memoryDestinationRepository{items: destinations}
	groupRepo := &memoryGroupRepository{destinations: destinations}
	settingsRepo := &staticSettingsRepository{settings: domain.AppSettings{
		DuplicateProtectionEnabled: false,
		DuplicateWindowMinutes:     10,
		SelectedDestinationGroupID: e2eGroupID,
	}}
	historyRepo := &memoryHistoryRepository{}
	sessionRepo := &memoryLiveNowSessionRepository{}

	execution := appsvc.NewExecutionService(
		destinationRepo,
		settingsRepo,
		historyRepo,
		sessionRepo,
		unavailable.DiscordPublisher{},
		bluesky.NewPublisher("", nil),
		mastodon.NewPublisher(nil),
	)
	execution.UseDestinationGroups(groupRepo)

	runID := time.Now().UTC().Format("20060102T150405Z")
	announcement := domain.Announcement{
		StreamTitle:        "StreamSignal E2E " + runID,
		StreamURL:          "https://example.com/streamsignal-e2e",
		Category:           "E2E",
		Message:            "Automated workflow test. This post should be deleted automatically.",
		Hashtags:           "#StreamSignalE2E",
		DestinationGroupID: e2eGroupID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	summary, err := execution.ForceGoLive(ctx, announcement)
	if err != nil {
		t.Fatalf("run workflow E2E: %v", err)
	}
	t.Logf("ForceGoLive summary: status=%s total=%d success=%d failed=%d skipped=%d validation_errors=%d", summary.Status, summary.TotalCount, summary.SuccessCount, summary.FailedCount, summary.SkippedCount, summary.ValidationErrorCount)
	t.Cleanup(func() {
		cleanupReceipts(t, destinations, summary.Results)
	})

	if summary.TotalCount != len(destinations) {
		t.Fatalf("expected %d execution results, got %d: %+v", len(destinations), summary.TotalCount, summary.Results)
	}
	if summary.SuccessCount != len(destinations) || summary.Status != domain.ExecutionSummaryStatusSuccess {
		t.Fatalf("expected all E2E destinations to publish successfully, got summary %+v", summary)
	}
	for _, result := range summary.Results {
		t.Logf(
			"Publish result: destination=%q platform=%s state=%s receipt=%s content_length=%d",
			result.DestinationName,
			result.Platform,
			result.State,
			receiptSummary(result.Receipt),
			len(result.Content),
		)
		if result.Receipt == nil {
			t.Fatalf("expected cleanup receipt for %s result: %+v", result.DestinationName, result)
		}
		if !strings.Contains(result.Content, "[StreamSignal E2E]") {
			t.Fatalf("expected rendered E2E marker in content for %s, got %q", result.DestinationName, result.Content)
		}
	}
}

func e2eDestinations(t *testing.T) []domain.Destination {
	t.Helper()
	var destinations []domain.Destination

	identifier, password := strings.TrimSpace(os.Getenv("E2E_BLUESKY_IDENTIFIER")), strings.TrimSpace(os.Getenv("E2E_BLUESKY_APP_PASSWORD"))
	if os.Getenv("E2E_EXPECT_BLUESKY") == "1" && (identifier == "" || password == "") {
		t.Fatal("E2E_EXPECT_BLUESKY=1 requires E2E_BLUESKY_IDENTIFIER and E2E_BLUESKY_APP_PASSWORD")
	}
	if identifier != "" && password != "" {
		config, err := json.Marshal(map[string]any{
			"accountIdentifier": identifier,
			"credentialKey":     password,
			"liveStatusEnabled": false,
		})
		if err != nil {
			t.Fatalf("encode Bluesky E2E config: %v", err)
		}
		destinations = append(destinations, domain.Destination{
			ID:         "e2e-bluesky",
			Platform:   domain.PlatformBluesky,
			Name:       "E2E Bluesky",
			Enabled:    true,
			Template:   "[StreamSignal E2E]\n{{stream_title}}\n{{stream_url}}\n{{message}}\n{{hashtags}}",
			ConfigJSON: string(config),
		})
	}

	instanceURL, token := strings.TrimSpace(os.Getenv("E2E_MASTODON_INSTANCE_URL")), strings.TrimSpace(os.Getenv("E2E_MASTODON_ACCESS_TOKEN"))
	if os.Getenv("E2E_EXPECT_MASTODON") == "1" && (instanceURL == "" || token == "") {
		t.Fatal("E2E_EXPECT_MASTODON=1 requires E2E_MASTODON_INSTANCE_URL and E2E_MASTODON_ACCESS_TOKEN")
	}
	if instanceURL != "" && token != "" {
		config, err := json.Marshal(map[string]any{
			"instanceURL":   instanceURL,
			"credentialKey": token,
		})
		if err != nil {
			t.Fatalf("encode Mastodon E2E config: %v", err)
		}
		destinations = append(destinations, domain.Destination{
			ID:         "e2e-mastodon",
			Platform:   domain.PlatformMastodon,
			Name:       "E2E Mastodon",
			Enabled:    true,
			Template:   "[StreamSignal E2E]\n{{stream_title}}\n{{stream_url}}\n{{message}}\n{{hashtags}}",
			ConfigJSON: string(config),
		})
	}

	return destinations
}

func cleanupReceipts(t *testing.T, destinations []domain.Destination, results []domain.ExecutionResult) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	destinationsByID := make(map[string]domain.Destination, len(destinations))
	for _, destination := range destinations {
		destinationsByID[destination.ID] = destination
	}

	for _, result := range results {
		if result.Receipt == nil {
			continue
		}
		destination := destinationsByID[result.DestinationID]
		switch result.Platform {
		case domain.PlatformBluesky:
			var config struct {
				AccountIdentifier string `json:"accountIdentifier"`
				CredentialKey     string `json:"credentialKey"`
			}
			if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
				t.Errorf("cleanup Bluesky config for %s: %v", result.DestinationName, err)
				continue
			}
			if err := bluesky.NewPublisher("", nil).DeletePost(ctx, config.AccountIdentifier, config.CredentialKey, result.Receipt.URI); err != nil {
				t.Errorf("cleanup Bluesky post %q: %v", result.Receipt.URI, err)
			} else {
				t.Logf("Cleanup deleted Bluesky post for destination=%q receipt=%s", result.DestinationName, receiptSummary(result.Receipt))
			}
		case domain.PlatformMastodon:
			var config struct {
				CredentialKey string `json:"credentialKey"`
				InstanceURL   string `json:"instanceURL"`
			}
			if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
				t.Errorf("cleanup Mastodon config for %s: %v", result.DestinationName, err)
				continue
			}
			if err := mastodon.NewPublisher(nil).DeleteStatus(ctx, config.CredentialKey, config.InstanceURL, result.Receipt.ID); err != nil {
				t.Errorf("cleanup Mastodon status %q: %v", result.Receipt.ID, err)
			} else {
				t.Logf("Cleanup deleted Mastodon status for destination=%q receipt=%s", result.DestinationName, receiptSummary(result.Receipt))
			}
		default:
			t.Errorf("no cleanup available for platform %s receipt %+v", result.Platform, result.Receipt)
		}
	}
}

func destinationSummary(destinations []domain.Destination) string {
	parts := make([]string, 0, len(destinations))
	for _, destination := range destinations {
		parts = append(parts, fmt.Sprintf("%s/%s", destination.Name, destination.Platform))
	}
	return strings.Join(parts, ", ")
}

func receiptSummary(receipt *domain.PublishReceipt) string {
	if receipt == nil {
		return "none"
	}
	parts := []string{fmt.Sprintf("provider=%s", receipt.Provider)}
	if receipt.ID != "" {
		parts = append(parts, fmt.Sprintf("id=%s", receipt.ID))
	}
	if receipt.URI != "" {
		parts = append(parts, fmt.Sprintf("uri=%s", receipt.URI))
	}
	if receipt.CID != "" {
		parts = append(parts, fmt.Sprintf("cid=%s", receipt.CID))
	}
	return strings.Join(parts, " ")
}

type memoryDestinationRepository struct {
	items []domain.Destination
}

func (r *memoryDestinationRepository) List(context.Context) ([]domain.Destination, error) {
	return append([]domain.Destination(nil), r.items...), nil
}

func (r *memoryDestinationRepository) Get(_ context.Context, id string) (domain.Destination, error) {
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.Destination{}, fmt.Errorf("destination %q not found", id)
}

func (r *memoryDestinationRepository) Save(context.Context, domain.Destination) error {
	return fmt.Errorf("unexpected destination save in E2E test")
}

func (r *memoryDestinationRepository) Delete(context.Context, string) error {
	return fmt.Errorf("unexpected destination delete in E2E test")
}

type memoryGroupRepository struct {
	destinations []domain.Destination
}

func (r *memoryGroupRepository) List(context.Context) ([]domain.DestinationGroup, error) {
	return []domain.DestinationGroup{{ID: e2eGroupID, Name: "E2E Test"}}, nil
}

func (r *memoryGroupRepository) Get(_ context.Context, id string) (domain.DestinationGroup, error) {
	if id == e2eGroupID {
		return domain.DestinationGroup{ID: e2eGroupID, Name: "E2E Test"}, nil
	}
	return domain.DestinationGroup{}, fmt.Errorf("group %q not found", id)
}

func (r *memoryGroupRepository) GetByName(_ context.Context, name string) (domain.DestinationGroup, error) {
	if name == "E2E Test" {
		return domain.DestinationGroup{ID: e2eGroupID, Name: "E2E Test"}, nil
	}
	return domain.DestinationGroup{}, fmt.Errorf("group %q not found", name)
}

func (r *memoryGroupRepository) Save(context.Context, domain.DestinationGroup) error {
	return fmt.Errorf("unexpected group save in E2E test")
}

func (r *memoryGroupRepository) Delete(context.Context, string) error {
	return fmt.Errorf("unexpected group delete in E2E test")
}

func (r *memoryGroupRepository) ListDestinations(_ context.Context, groupID string) ([]domain.Destination, error) {
	if groupID != e2eGroupID {
		return nil, fmt.Errorf("group %q not found", groupID)
	}
	return append([]domain.Destination(nil), r.destinations...), nil
}

func (r *memoryGroupRepository) ListGroupIDsForDestination(context.Context, string) ([]string, error) {
	return []string{e2eGroupID}, nil
}

func (r *memoryGroupRepository) SetDestinations(context.Context, string, []string) error {
	return fmt.Errorf("unexpected group destination mutation in E2E test")
}

func (r *memoryGroupRepository) SetGroupsForDestination(context.Context, string, []string) error {
	return fmt.Errorf("unexpected destination group mutation in E2E test")
}

func (r *memoryGroupRepository) AddDestination(context.Context, string, string) error {
	return fmt.Errorf("unexpected group destination mutation in E2E test")
}

func (r *memoryGroupRepository) RemoveDestination(context.Context, string, string) error {
	return fmt.Errorf("unexpected group destination mutation in E2E test")
}

type staticSettingsRepository struct {
	settings domain.AppSettings
}

func (r *staticSettingsRepository) Load(context.Context) (domain.AppSettings, error) {
	return r.settings, nil
}

func (r *staticSettingsRepository) Save(_ context.Context, settings domain.AppSettings) error {
	r.settings = settings
	return nil
}

type memoryHistoryRepository struct {
	records []ports.PostHistoryRecord
}

func (r *memoryHistoryRepository) Record(_ context.Context, destinationID string, _ string, hash string, postedAt time.Time) error {
	r.records = append(r.records, ports.PostHistoryRecord{DestinationID: destinationID, ContentHash: hash, PostedAt: postedAt})
	return nil
}

func (r *memoryHistoryRepository) FindRecentByDestination(_ context.Context, destinationID string, _ time.Time) ([]ports.PostHistoryRecord, error) {
	var matches []ports.PostHistoryRecord
	for _, record := range r.records {
		if record.DestinationID == destinationID {
			matches = append(matches, record)
		}
	}
	return matches, nil
}

type memoryLiveNowSessionRepository struct{}

func (r *memoryLiveNowSessionRepository) Upsert(context.Context, domain.ActiveLiveNowSession) error {
	return nil
}

func (r *memoryLiveNowSessionRepository) List(context.Context) ([]domain.ActiveLiveNowSession, error) {
	return nil, nil
}

func (r *memoryLiveNowSessionRepository) Delete(context.Context, string) error {
	return nil
}
