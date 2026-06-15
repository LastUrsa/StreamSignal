package sip

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"StreamSignal/internal/domain"
)

func TestReadOnlySIPV1ContractForLivePanel(t *testing.T) {
	service := newTestService()
	handler := service.Handler()

	appPayload := requestRawJSON(t, handler, http.MethodGet, "/api/v1/app", nil, http.StatusOK)
	assertJSONFields(t, appPayload, map[string]any{
		"appId":           "streamsignal",
		"appName":         "StreamSignal",
		"name":            "StreamSignal",
		"version":         "0.4.0",
		"mode":            "standalone",
		"protocolVersion": 1,
	})

	healthPayload := requestRawJSON(t, handler, http.MethodGet, "/api/v1/health", nil, http.StatusOK)
	assertJSONFields(t, healthPayload, map[string]any{
		"status":  HealthReady,
		"message": "StreamSignal is ready for local SIP participation.",
	})

	capabilitiesPayload := requestRawJSON(t, handler, http.MethodGet, "/api/v1/capabilities", nil, http.StatusOK)
	assertJSONFields(t, capabilitiesPayload, map[string]any{
		"supportsProfiles":        true,
		"supportsStatusReporting": true,
		"supportsAnnouncements":   true,
	})

	statusPayload := requestRawJSON(t, handler, http.MethodGet, "/api/v1/status", nil, http.StatusOK)
	assertJSONFields(t, statusPayload, map[string]any{
		"state":   StatusIdle,
		"message": "Ready for announcement setup.",
	})
	assertAbsentJSONFields(t, statusPayload, "activeProfile", "activeProfileId")
}

func TestHandlerReturnsIdentityAndCapabilities(t *testing.T) {
	service := newTestService()
	server := httptest.NewServer(service.Handler())
	defer server.Close()

	var app AppResponse
	getJSON(t, server.URL+"/api/v1/app", &app)
	if app.AppID != "streamsignal" || app.AppName != "StreamSignal" || app.Name != "StreamSignal" || app.Version != "0.4.0" || app.Mode != "standalone" || app.ProtocolVersion != 1 {
		t.Fatalf("unexpected app response: %+v", app)
	}
	if strings.Join(app.Capabilities, ",") != "profiles,announcement-fields" {
		t.Fatalf("unexpected app capabilities: %+v", app.Capabilities)
	}

	var capabilities CapabilitiesResponse
	getJSON(t, server.URL+"/api/v1/capabilities", &capabilities)
	if !capabilities.SupportsProfiles || !capabilities.SupportsStatusReporting || !capabilities.SupportsAnnouncements {
		t.Fatalf("expected core capabilities, got %+v", capabilities)
	}
	if strings.Join(capabilities.Capabilities, ",") != "profiles,announcement-fields" {
		t.Fatalf("unexpected capabilities list: %+v", capabilities.Capabilities)
	}
}

func TestAppReportsConfiguredRuntimeMode(t *testing.T) {
	service := newTestServiceWithMode(ServiceMode)

	var app AppResponse
	requestJSON(t, service.Handler(), http.MethodGet, "/api/v1/app", nil, http.StatusOK, &app)

	if app.Mode != ServiceMode {
		t.Fatalf("expected service mode, got %+v", app)
	}
}

func TestHealthReportsReadyAndDegraded(t *testing.T) {
	readyService := newTestService()
	var ready HealthResponse
	requestJSON(t, readyService.Handler(), http.MethodGet, "/api/v1/health", nil, http.StatusOK, &ready)
	if ready.Status != HealthReady {
		t.Fatalf("expected ready health, got %+v", ready)
	}

	degradedService := newTestService()
	degradedService.recovery = fakeRecovery{
		sessions: []domain.ActiveLiveNowSession{{DestinationID: "bluesky-main"}},
	}
	var degraded HealthResponse
	requestJSON(t, degradedService.Handler(), http.MethodGet, "/api/v1/health", nil, http.StatusOK, &degraded)
	if degraded.Status != HealthDegraded {
		t.Fatalf("expected degraded health, got %+v", degraded)
	}
}

func TestHealthReportsErrorForCoreDependencyFailures(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*Service)
		message   string
	}{
		{
			name: "settings unavailable",
			configure: func(service *Service) {
				service.settings = fakeSettings{err: errors.New("settings unavailable")}
			},
			message: "StreamSignal settings are unavailable.",
		},
		{
			name: "profiles unavailable",
			configure: func(service *Service) {
				service.profiles = &fakeProfiles{listErr: errors.New("profiles unavailable")}
			},
			message: "Announcement profiles are unavailable.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newTestService()
			test.configure(service)

			var health HealthResponse
			requestJSON(t, service.Handler(), http.MethodGet, "/api/v1/health", nil, http.StatusOK, &health)
			if health.Status != HealthError || health.Message != test.message {
				t.Fatalf("unexpected health error response: %+v", health)
			}

			var status StatusResponse
			requestJSON(t, service.Handler(), http.MethodGet, "/api/v1/status", nil, http.StatusOK, &status)
			if status.State != StatusError || status.Message != test.message {
				t.Fatalf("expected status to mirror health error, got %+v", status)
			}
		})
	}
}

func TestProfilesAndActivationUseExistingAnnouncementProfiles(t *testing.T) {
	profiles := &fakeProfiles{profiles: []domain.AnnouncementProfile{
		{ID: "music", Name: "Music Stream", DefaultStreamTitle: "Late Night Music"},
		{ID: "gaming", Name: "Gaming Stream", DefaultStreamTitle: "Adventure Time"},
	}}
	service := newTestService()
	service.profiles = profiles

	var listed ProfilesResponse
	requestJSON(t, service.Handler(), http.MethodGet, "/api/v1/profiles", nil, http.StatusOK, &listed)
	if got := strings.Join(listed.Profiles, ","); got != "Music Stream,Gaming Stream" {
		t.Fatalf("unexpected profiles %q", got)
	}

	var activated ProfileActivationResponse
	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/profile", map[string]string{"profile": "music stream"}, http.StatusOK, &activated)
	if !activated.Success || activated.Profile != "Music Stream" || activated.ProfileID != "music" {
		t.Fatalf("unexpected activation response: %+v", activated)
	}
	if len(profiles.markedUsed) != 1 || profiles.markedUsed[0] != "music" {
		t.Fatalf("expected profile usage to be marked, got %+v", profiles.markedUsed)
	}

	var status StatusResponse
	requestJSON(t, service.Handler(), http.MethodGet, "/api/v1/status", nil, http.StatusOK, &status)
	if status.ActiveProfile != "Music Stream" || status.ActiveProfileID != "music" {
		t.Fatalf("expected active profile in status, got %+v", status)
	}
}

func TestAnnouncementFieldsReadAndUpdateValues(t *testing.T) {
	fields := &fakeAnnouncementFields{fields: []domain.AnnouncementField{
		{ID: "stream_title", Name: "Stream Title", Value: "Late Night Stream"},
		{ID: "stream_url", Name: "Stream URL", Value: "https://example.com/live"},
		{ID: "category", Name: "Category/Game", Value: "Final Fantasy XIV"},
		{ID: "hashtags", Name: "Hashtags", Value: "#vtuber #gaming"},
		{ID: "message", Name: "Optional Message", Value: "Raid Night"},
	}}
	service := newTestService()
	service.UseAnnouncementFields(fields)

	var listed AnnouncementFieldsResponse
	requestJSON(t, service.Handler(), http.MethodGet, "/api/v1/announcement-fields", nil, http.StatusOK, &listed)
	if len(listed.Fields) != 5 || listed.Fields[0].ID != "stream_title" || listed.Fields[0].Value != "Late Night Stream" {
		t.Fatalf("unexpected announcement fields response: %+v", listed)
	}

	var updated SuccessResponse
	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/announcement-fields", UpdateAnnouncementFieldsRequest{
		Fields: []UpdateAnnouncementFieldRequest{
			{ID: " category ", Value: "Clair Obscur"},
			{ID: "message", Value: "Blind Playthrough"},
		},
	}, http.StatusOK, &updated)
	if !updated.Success {
		t.Fatalf("expected successful update, got %+v", updated)
	}
	if fields.values["category"] != "Clair Obscur" || fields.values["message"] != "Blind Playthrough" {
		t.Fatalf("expected field values to update, got %+v", fields.values)
	}
}

func TestAnnouncementFieldsRejectInvalidUpdates(t *testing.T) {
	tests := []struct {
		name string
		body any
	}{
		{
			name: "blank id",
			body: UpdateAnnouncementFieldsRequest{Fields: []UpdateAnnouncementFieldRequest{{ID: "", Value: "x"}}},
		},
		{
			name: "unknown id",
			body: UpdateAnnouncementFieldsRequest{Fields: []UpdateAnnouncementFieldRequest{{ID: "unknown", Value: "x"}}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newTestService()
			service.UseAnnouncementFields(&fakeAnnouncementFields{
				fields: []domain.AnnouncementField{{ID: "category", Name: "Category/Game"}},
				err:    errors.New("missing field"),
			})

			var errorResponse ErrorResponse
			requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/announcement-fields", test.body, http.StatusBadRequest, &errorResponse)
			if errorResponse.Error != "InvalidRequest" {
				t.Fatalf("expected InvalidRequest, got %+v", errorResponse)
			}
		})
	}
}

func TestCurrentProfileReturnsActiveAndEmptyState(t *testing.T) {
	service := newTestService()

	var empty CurrentProfileResponse
	requestJSON(t, service.Handler(), http.MethodGet, "/api/v1/profile/current", nil, http.StatusOK, &empty)
	if empty.ID != "" || empty.Name != "" {
		t.Fatalf("expected empty current profile, got %+v", empty)
	}

	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/profile", map[string]string{"profile": "Music Stream"}, http.StatusOK, nil)

	var current CurrentProfileResponse
	requestJSON(t, service.Handler(), http.MethodGet, "/api/v1/profile/current", nil, http.StatusOK, &current)
	if current.ID != "music" || current.Name != "Music Stream" {
		t.Fatalf("expected active current profile, got %+v", current)
	}
}

func TestActivateProfileErrors(t *testing.T) {
	service := newTestService()

	var missing ErrorResponse
	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/profile", map[string]string{"profile": "Missing"}, http.StatusNotFound, &missing)
	if missing.Error != "ProfileNotFound" {
		t.Fatalf("expected ProfileNotFound, got %+v", missing)
	}

	var invalid ErrorResponse
	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/profile", map[string]string{"profile": ""}, http.StatusBadRequest, &invalid)
	if invalid.Error != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest, got %+v", invalid)
	}
}

func TestAnnounceWorkflowUsesActiveProfileAndExecutionService(t *testing.T) {
	execution := &fakeExecution{
		summary: domain.ExecutionSummary{
			Status:       domain.ExecutionSummaryStatusSuccess,
			TotalCount:   1,
			SuccessCount: 1,
			Results: []domain.ExecutionResult{{
				DestinationID:   "discord-main",
				DestinationName: "Discord",
				Platform:        domain.PlatformDiscord,
				State:           domain.ExecutionStateSuccess,
				Message:         "Published successfully.",
			}},
		},
	}
	profiles := &fakeProfiles{profiles: []domain.AnnouncementProfile{{
		ID:                          "gaming",
		Name:                        "Gaming Stream",
		DefaultStreamTitle:          "Boss Night",
		DefaultStreamURL:            "https://example.com/live",
		DefaultCategory:             "Games",
		DefaultMessage:              "Going live now",
		DefaultHashtags:             "#gaming",
		DefaultAnnouncementImageURL: "https://example.com/card.png",
		DefaultGroupID:              "main-group",
	}}}
	service := newTestService()
	service.profiles = profiles
	service.UseExecution(execution)
	service.UseAnnouncementFields(&fakeAnnouncementFields{fields: []domain.AnnouncementField{
		{ID: "stream_title", Name: "Stream Title", Value: ""},
		{ID: "stream_url", Name: "Stream URL", Value: ""},
		{ID: "category", Name: "Category/Game", Value: "Clair Obscur"},
		{ID: "hashtags", Name: "Hashtags", Value: "#co33"},
		{ID: "message", Name: "Optional Message", Value: "Blind Playthrough"},
	}})

	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/profile", map[string]string{"profile": "Gaming Stream"}, http.StatusOK, nil)

	var announced AnnounceResponse
	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/announce", nil, http.StatusOK, &announced)
	if !announced.Success || announced.Error != "" {
		t.Fatalf("expected announcement success, got %+v", announced)
	}
	if len(execution.announcements) != 1 {
		t.Fatalf("expected one announcement execution, got %d", len(execution.announcements))
	}
	got := execution.announcements[0]
	if got.StreamTitle != "Boss Night" || got.StreamURL != "https://example.com/live" || got.Category != "Clair Obscur" || got.Message != "Blind Playthrough" || got.Hashtags != "#co33" || got.DestinationGroupID != "main-group" || got.AnnouncementImageURL != "https://example.com/card.png" {
		t.Fatalf("expected profile defaults to become announcement, got %+v", got)
	}

	var status AnnounceStatusResponse
	requestJSON(t, service.Handler(), http.MethodGet, "/api/v1/announce/status", nil, http.StatusOK, &status)
	if !status.Success || status.LastRun == "" || status.Error != "" {
		t.Fatalf("expected successful announce status, got %+v", status)
	}
}

func TestAnnounceAppliesOneShotFieldOverridesWithoutPersisting(t *testing.T) {
	execution := &fakeExecution{
		summary: domain.ExecutionSummary{
			Status:       domain.ExecutionSummaryStatusSuccess,
			TotalCount:   1,
			SuccessCount: 1,
		},
	}
	fields := &fakeAnnouncementFields{
		fields: []domain.AnnouncementField{
			{ID: "stream_title", Name: "Stream Title", Value: "Stored Stream"},
			{ID: "stream_url", Name: "Stream URL", Value: "https://example.com/stored"},
			{ID: "category", Name: "Category/Game", Value: "Clair Obscur"},
			{ID: "hashtags", Name: "Hashtags", Value: "#stored"},
			{ID: "message", Name: "Optional Message", Value: "Blind Playthrough"},
		},
		values: map[string]string{
			"stream_title": "Stored Stream",
			"stream_url":   "https://example.com/stored",
			"category":     "Clair Obscur",
			"hashtags":     "#stored",
			"message":      "Blind Playthrough",
		},
	}
	service := newTestService()
	service.UseExecution(execution)
	service.UseAnnouncementFields(fields)

	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/profile", map[string]string{"profile": "Music Stream"}, http.StatusOK, nil)

	var announced AnnounceResponse
	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/announce", AnnounceRequest{
		Fields: []AnnouncementFieldOverride{
			{ID: " STREAM_TITLE ", Value: " Late Night Raid "},
			{ID: "stream_url", Value: "https://example.com/live"},
			{ID: "category", Value: "Final Fantasy XIV"},
			{ID: "hashtags", Value: "#vtuber #raid"},
			{ID: "message", Value: "Raid Night"},
		},
	}, http.StatusOK, &announced)
	if !announced.Success || announced.Error != "" {
		t.Fatalf("expected announcement success, got %+v", announced)
	}
	if len(execution.announcements) != 1 {
		t.Fatalf("expected one announcement execution, got %d", len(execution.announcements))
	}
	got := execution.announcements[0]
	if got.StreamTitle != "Late Night Raid" || got.StreamURL != "https://example.com/live" || got.Category != "Final Fantasy XIV" || got.Hashtags != "#vtuber #raid" || got.Message != "Raid Night" {
		t.Fatalf("expected one-shot field overrides on execution snapshot, got %+v", got)
	}
	if fields.values["stream_title"] != "Stored Stream" || fields.values["stream_url"] != "https://example.com/stored" || fields.values["category"] != "Clair Obscur" || fields.values["hashtags"] != "#stored" || fields.values["message"] != "Blind Playthrough" {
		t.Fatalf("expected persistent field values to remain unchanged, got %+v", fields.values)
	}
	if fields.fields[0].Value != "Stored Stream" || fields.fields[1].Value != "https://example.com/stored" || fields.fields[2].Value != "Clair Obscur" || fields.fields[3].Value != "#stored" || fields.fields[4].Value != "Blind Playthrough" {
		t.Fatalf("expected field definitions to remain unchanged, got %+v", fields.fields)
	}
}

func TestAnnounceRejectsInvalidOneShotFieldOverrides(t *testing.T) {
	tests := []struct {
		name string
		body AnnounceRequest
	}{
		{
			name: "blank id",
			body: AnnounceRequest{Fields: []AnnouncementFieldOverride{{ID: "", Value: "x"}}},
		},
		{
			name: "unknown id",
			body: AnnounceRequest{Fields: []AnnouncementFieldOverride{{ID: "unknown", Value: "x"}}},
		},
		{
			name: "too long value",
			body: AnnounceRequest{Fields: []AnnouncementFieldOverride{{ID: "category", Value: strings.Repeat("x", maxAnnouncementFieldOverrideValueLength+1)}}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			execution := &fakeExecution{}
			service := newTestService()
			service.UseExecution(execution)
			service.UseAnnouncementFields(&fakeAnnouncementFields{
				fields: []domain.AnnouncementField{{ID: "category", Name: "Category/Game", Value: "Clair Obscur"}},
			})
			requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/profile", map[string]string{"profile": "Music Stream"}, http.StatusOK, nil)

			var response ErrorResponse
			requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/announce", test.body, http.StatusBadRequest, &response)
			if response.Error != "InvalidRequest" {
				t.Fatalf("expected InvalidRequest, got %+v", response)
			}
			if len(execution.announcements) != 0 {
				t.Fatalf("expected invalid overrides not to execute, got %+v", execution.announcements)
			}
		})
	}
}

func TestAnnounceReportsMissingProfileAndValidationFailures(t *testing.T) {
	t.Run("missing profile", func(t *testing.T) {
		service := newTestService()
		service.UseExecution(&fakeExecution{})

		var announced AnnounceResponse
		requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/announce", nil, http.StatusOK, &announced)
		if announced.Success || announced.Error != "No active profile" {
			t.Fatalf("expected missing profile failure, got %+v", announced)
		}

		var status AnnounceStatusResponse
		requestJSON(t, service.Handler(), http.MethodGet, "/api/v1/announce/status", nil, http.StatusOK, &status)
		if status.Success || status.Error != "No active profile" || status.LastRun == "" {
			t.Fatalf("expected missing profile announce status, got %+v", status)
		}
	})

	t.Run("validation failure", func(t *testing.T) {
		service := newTestService()
		service.UseExecution(&fakeExecution{
			summary: domain.ExecutionSummary{
				Status:               domain.ExecutionSummaryStatusPartial,
				ValidationErrorCount: 1,
				TotalCount:           1,
				Results: []domain.ExecutionResult{{
					DestinationID:   "discord-main",
					DestinationName: "Discord",
					Platform:        domain.PlatformDiscord,
					State:           domain.ExecutionStateValidationError,
					Message:         "Stream title is required.",
				}},
			},
		})
		requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/profile", map[string]string{"profile": "Music Stream"}, http.StatusOK, nil)

		var announced AnnounceResponse
		requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/announce", map[string]string{}, http.StatusOK, &announced)
		if announced.Success || announced.Error != "Stream title is required." {
			t.Fatalf("expected validation failure, got %+v", announced)
		}
	})
}

func TestAnnounceDuplicateGuardCanBeConfirmedOnce(t *testing.T) {
	execution := &fakeExecution{
		summary: domain.ExecutionSummary{
			Status:                        domain.ExecutionSummaryStatusWarning,
			RequiresDuplicateConfirmation: true,
			DuplicateWarningMessage:       "A similar announcement was recently posted within the last 10 minutes. Continue?",
		},
		forceSummary: domain.ExecutionSummary{
			Status:       domain.ExecutionSummaryStatusSuccess,
			TotalCount:   1,
			SuccessCount: 1,
			Results: []domain.ExecutionResult{{
				DestinationID:   "discord-main",
				DestinationName: "Discord",
				Platform:        domain.PlatformDiscord,
				State:           domain.ExecutionStateSuccess,
				Message:         "Published successfully.",
			}},
		},
	}
	profiles := &fakeProfiles{profiles: []domain.AnnouncementProfile{{
		ID:                 "gaming",
		Name:               "Gaming Stream",
		DefaultStreamTitle: "Boss Night",
		DefaultStreamURL:   "https://example.com/live",
		DefaultGroupID:     "main-group",
	}}}
	service := newTestService()
	service.profiles = profiles
	service.UseExecution(execution)
	service.UseAnnouncementFields(&fakeAnnouncementFields{fields: []domain.AnnouncementField{
		{ID: "stream_title", Name: "Stream Title", Value: ""},
		{ID: "stream_url", Name: "Stream URL", Value: ""},
		{ID: "category", Name: "Category/Game", Value: "Clair Obscur"},
		{ID: "hashtags", Name: "Hashtags", Value: ""},
		{ID: "message", Name: "Optional Message", Value: "Blind Playthrough"},
	}})

	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/profile", map[string]string{"profile": "Gaming Stream"}, http.StatusOK, nil)

	var guarded AnnounceResponse
	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/announce", AnnounceRequest{
		Fields: []AnnouncementFieldOverride{
			{ID: "stream_title", Value: "Late Night Raid"},
			{ID: "stream_url", Value: "https://example.com/raid"},
			{ID: "category", Value: "Final Fantasy XIV"},
			{ID: "hashtags", Value: "#raid"},
			{ID: "message", Value: "Raid Night"},
		},
	}, http.StatusOK, &guarded)
	if guarded.Success || !guarded.RequiresConfirmation || guarded.ConfirmationID == "" || !strings.Contains(guarded.Error, "Continue?") {
		t.Fatalf("expected duplicate guard response, got %+v", guarded)
	}

	var guardedStatus AnnounceStatusResponse
	requestJSON(t, service.Handler(), http.MethodGet, "/api/v1/announce/status", nil, http.StatusOK, &guardedStatus)
	if guardedStatus.Success || !guardedStatus.RequiresConfirmation || guardedStatus.ConfirmationID != guarded.ConfirmationID || guardedStatus.Error != guarded.Error {
		t.Fatalf("expected status to expose duplicate guard, got %+v", guardedStatus)
	}

	var confirmed AnnounceResponse
	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/announce/confirm", map[string]string{"confirmationId": guarded.ConfirmationID}, http.StatusOK, &confirmed)
	if !confirmed.Success || confirmed.RequiresConfirmation || confirmed.ConfirmationID != "" || confirmed.Error != "" {
		t.Fatalf("expected confirmed announcement success, got %+v", confirmed)
	}
	if len(execution.forceAnnouncements) != 1 {
		t.Fatalf("expected one force execution, got %d", len(execution.forceAnnouncements))
	}
	forced := execution.forceAnnouncements[0]
	if forced.StreamTitle != "Late Night Raid" || forced.StreamURL != "https://example.com/raid" || forced.Category != "Final Fantasy XIV" || forced.Hashtags != "#raid" || forced.Message != "Raid Night" || forced.DestinationGroupID != "main-group" {
		t.Fatalf("expected force execution to use guarded snapshot, got %+v", forced)
	}

	var confirmedStatus AnnounceStatusResponse
	requestJSON(t, service.Handler(), http.MethodGet, "/api/v1/announce/status", nil, http.StatusOK, &confirmedStatus)
	if !confirmedStatus.Success || confirmedStatus.RequiresConfirmation || confirmedStatus.ConfirmationID != "" || confirmedStatus.Error != "" {
		t.Fatalf("expected status to clear duplicate guard after confirm, got %+v", confirmedStatus)
	}

	var reused ErrorResponse
	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/announce/confirm", map[string]string{"confirmationId": guarded.ConfirmationID}, http.StatusBadRequest, &reused)
	if reused.Error != "InvalidRequest" {
		t.Fatalf("expected reused confirmation to be invalid, got %+v", reused)
	}
}

func TestAnnounceDuplicateGuardConfirmationExpires(t *testing.T) {
	service := newTestService()
	service.UseExecution(&fakeExecution{})
	service.setPendingAnnouncementConfirmation("expired-confirmation", domain.Announcement{StreamTitle: "Old"}, time.Now().UTC().Add(-time.Minute))

	var response ErrorResponse
	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/announce/confirm", map[string]string{"confirmationId": "expired-confirmation"}, http.StatusBadRequest, &response)
	if response.Error != "InvalidRequest" {
		t.Fatalf("expected expired confirmation to be invalid, got %+v", response)
	}
}

func TestAnnounceConfirmRequestValidation(t *testing.T) {
	service := newTestService()
	service.UseExecution(&fakeExecution{})

	tests := []struct {
		name        string
		body        string
		contentType string
		wantStatus  int
	}{
		{
			name:        "valid shape but missing pending confirmation is invalid",
			body:        `{"confirmationId":"missing"}`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:       "missing content type is rejected",
			body:       `{"confirmationId":"missing"}`,
			wantStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:        "non json content type is rejected",
			body:        `{"confirmationId":"missing"}`,
			contentType: "text/plain",
			wantStatus:  http.StatusUnsupportedMediaType,
		},
		{
			name:        "malformed json is rejected",
			body:        `{`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "unknown fields are rejected",
			body:        `{"confirmationId":"missing","command":"force"}`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "trailing json is rejected",
			body:        `{"confirmationId":"missing"} {}`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/announce/confirm", strings.NewReader(test.body))
			request.Host = "127.0.0.1"
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			response := httptest.NewRecorder()

			service.Handler().ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("expected status %d, got %d body %s", test.wantStatus, response.Code, response.Body.String())
			}
			if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
				t.Fatalf("expected JSON content type, got %q", got)
			}
		})
	}
}

func TestEndStreamWorkflowUsesActiveProfileAndExecutionService(t *testing.T) {
	execution := &fakeExecution{
		endStreamSummary: domain.ExecutionSummary{
			Status:       domain.ExecutionSummaryStatusSuccess,
			TotalCount:   1,
			SuccessCount: 1,
			Results: []domain.ExecutionResult{{
				DestinationID:   "bluesky-main",
				DestinationName: "Bluesky",
				Platform:        domain.PlatformBluesky,
				State:           domain.ExecutionStateSuccess,
				Message:         "End stream post published successfully.",
			}},
		},
	}
	profiles := &fakeProfiles{profiles: []domain.AnnouncementProfile{{
		ID:                 "gaming",
		Name:               "Gaming Stream",
		DefaultStreamTitle: "Boss Night",
		DefaultStreamURL:   "https://example.com/live",
		DefaultGroupID:     "main-group",
	}}}
	service := newTestService()
	service.profiles = profiles
	service.UseExecution(execution)

	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/profile", map[string]string{"profile": "Gaming Stream"}, http.StatusOK, nil)

	var ended EndStreamResponse
	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/end-stream", nil, http.StatusOK, &ended)
	if !ended.Success || ended.Error != "" {
		t.Fatalf("expected end stream success, got %+v", ended)
	}
	if len(execution.endStreamAnnouncements) != 1 {
		t.Fatalf("expected one end stream execution, got %d", len(execution.endStreamAnnouncements))
	}
	got := execution.endStreamAnnouncements[0]
	if got.StreamTitle != "Boss Night" || got.StreamURL != "https://example.com/live" || got.DestinationGroupID != "main-group" {
		t.Fatalf("expected profile defaults to become end stream announcement, got %+v", got)
	}

	var status EndStreamStatusResponse
	requestJSON(t, service.Handler(), http.MethodGet, "/api/v1/end-stream/status", nil, http.StatusOK, &status)
	if !status.Success || status.LastRun == "" || status.Error != "" {
		t.Fatalf("expected successful end stream status, got %+v", status)
	}
}

func TestEndStreamReportsMissingProfileAndValidationFailures(t *testing.T) {
	t.Run("missing profile", func(t *testing.T) {
		service := newTestService()
		service.UseExecution(&fakeExecution{})

		var ended EndStreamResponse
		requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/end-stream", nil, http.StatusOK, &ended)
		if ended.Success || ended.Error != "No active profile" {
			t.Fatalf("expected missing profile failure, got %+v", ended)
		}

		var status EndStreamStatusResponse
		requestJSON(t, service.Handler(), http.MethodGet, "/api/v1/end-stream/status", nil, http.StatusOK, &status)
		if status.Success || status.Error != "No active profile" || status.LastRun == "" {
			t.Fatalf("expected missing profile end stream status, got %+v", status)
		}
	})

	t.Run("validation failure", func(t *testing.T) {
		service := newTestService()
		service.UseExecution(&fakeExecution{
			endStreamSummary: domain.ExecutionSummary{
				Status:               domain.ExecutionSummaryStatusPartial,
				ValidationErrorCount: 1,
				TotalCount:           1,
				Results: []domain.ExecutionResult{{
					DestinationID:   "discord-main",
					DestinationName: "Discord",
					Platform:        domain.PlatformDiscord,
					State:           domain.ExecutionStateValidationError,
					Message:         "Stream URL is required.",
				}},
			},
		})
		requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/profile", map[string]string{"profile": "Music Stream"}, http.StatusOK, nil)

		var ended EndStreamResponse
		requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/end-stream", map[string]string{}, http.StatusOK, &ended)
		if ended.Success || ended.Error != "Stream URL is required." {
			t.Fatalf("expected validation failure, got %+v", ended)
		}
	})
}

func TestAnnounceRequestValidation(t *testing.T) {
	service := newTestService()
	service.UseExecution(&fakeExecution{})

	assertEmptyOperationRequestValidation(t, service.Handler(), "/api/v1/announce")

	var announced AnnounceResponse
	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/profile", map[string]string{"profile": "Music Stream"}, http.StatusOK, nil)
	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/announce", AnnounceRequest{Fields: []AnnouncementFieldOverride{}}, http.StatusOK, &announced)
}

func TestEndStreamRequestValidation(t *testing.T) {
	service := newTestService()
	service.UseExecution(&fakeExecution{})

	assertEmptyOperationRequestValidation(t, service.Handler(), "/api/v1/end-stream")
}

func assertEmptyOperationRequestValidation(t *testing.T, handler http.Handler, path string) {
	t.Helper()
	tests := []struct {
		name        string
		body        string
		contentType string
		wantStatus  int
	}{
		{
			name:       "empty body is accepted",
			wantStatus: http.StatusOK,
		},
		{
			name:        "empty json object is accepted",
			body:        `{}`,
			contentType: "application/json",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "non json content type is rejected",
			body:        `{}`,
			contentType: "text/plain",
			wantStatus:  http.StatusUnsupportedMediaType,
		},
		{
			name:       "body without content type is rejected",
			body:       `{}`,
			wantStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:        "malformed json is rejected",
			body:        `{`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "unknown fields are rejected",
			body:        `{"command":"go_live"}`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "trailing json is rejected",
			body:        `{} {}`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(test.body))
			request.Host = "127.0.0.1"
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("expected status %d, got %d body %s", test.wantStatus, response.Code, response.Body.String())
			}
			if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
				t.Fatalf("expected JSON content type, got %q", got)
			}
		})
	}
}

func TestProfileActivationRequiresJSONContentType(t *testing.T) {
	service := newTestService()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/profile", strings.NewReader(`{"profile":"Music Stream"}`))
	request.Host = "127.0.0.1"
	request.Header.Set("Content-Type", "text/plain")
	response := httptest.NewRecorder()

	service.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected unsupported media type, got %d", response.Code)
	}
	var errorResponse ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if errorResponse.Error != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest, got %+v", errorResponse)
	}
}

func TestReadOnlyEndpointsRejectNonGETMethods(t *testing.T) {
	service := newTestService()
	for _, path := range []string{"/api/v1/app", "/api/v1/health", "/api/v1/capabilities", "/api/v1/status", "/api/v1/profiles", "/api/v1/profile/current", "/api/v1/announce/status", "/api/v1/end-stream/status"} {
		t.Run(path, func(t *testing.T) {
			var errorResponse ErrorResponse
			response := requestRawJSON(t, service.Handler(), http.MethodPost, path, map[string]string{"ignored": "true"}, http.StatusMethodNotAllowed)
			if err := json.Unmarshal(response, &errorResponse); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if errorResponse.Error != "InvalidRequest" {
				t.Fatalf("expected InvalidRequest, got %+v", errorResponse)
			}
		})
	}
}

func TestAnnounceRejectsNonPOSTMethods(t *testing.T) {
	service := newTestService()

	assertOperationRejectsNonPOSTMethods(t, service.Handler(), "/api/v1/announce")
}

func TestAnnounceConfirmRejectsNonPOSTMethods(t *testing.T) {
	service := newTestService()

	assertOperationRejectsNonPOSTMethods(t, service.Handler(), "/api/v1/announce/confirm")
}

func TestEndStreamRejectsNonPOSTMethods(t *testing.T) {
	service := newTestService()

	assertOperationRejectsNonPOSTMethods(t, service.Handler(), "/api/v1/end-stream")
}

func assertOperationRejectsNonPOSTMethods(t *testing.T, handler http.Handler, path string) {
	t.Helper()
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			var errorResponse ErrorResponse
			response := requestRawJSON(t, handler, method, path, nil, http.StatusMethodNotAllowed)
			if err := json.Unmarshal(response, &errorResponse); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if errorResponse.Error != "InvalidRequest" {
				t.Fatalf("expected InvalidRequest, got %+v", errorResponse)
			}
		})
	}
}

func TestHandlerRejectsNonLocalHost(t *testing.T) {
	service := newTestService()
	for _, endpoint := range []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/api/v1/app"},
		{method: http.MethodGet, path: "/api/v1/profile/current"},
		{method: http.MethodGet, path: "/api/v1/announcement-fields"},
		{method: http.MethodPost, path: "/api/v1/announcement-fields", body: `{"fields":[]}`},
		{method: http.MethodPost, path: "/api/v1/profile", body: `{"profile":"Music Stream"}`},
		{method: http.MethodPost, path: "/api/v1/announce", body: `{}`},
		{method: http.MethodPost, path: "/api/v1/announce/confirm", body: `{"confirmationId":"missing"}`},
		{method: http.MethodGet, path: "/api/v1/announce/status"},
		{method: http.MethodPost, path: "/api/v1/end-stream", body: `{}`},
		{method: http.MethodGet, path: "/api/v1/end-stream/status"},
	} {
		t.Run(endpoint.method+" "+endpoint.path, func(t *testing.T) {
			request := httptest.NewRequest(endpoint.method, endpoint.path, strings.NewReader(endpoint.body))
			request.Host = "example.com"
			if endpoint.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			response := httptest.NewRecorder()

			service.Handler().ServeHTTP(response, request)

			if response.Code != http.StatusForbidden {
				t.Fatalf("expected forbidden for non-local host, got %d", response.Code)
			}
			var errorResponse ErrorResponse
			if err := json.NewDecoder(response.Body).Decode(&errorResponse); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if errorResponse.Error != "Forbidden" {
				t.Fatalf("expected Forbidden, got %+v", errorResponse)
			}
		})
	}
}

func TestHandlerSetsSecurityHeaders(t *testing.T) {
	service := newTestService()
	for _, endpoint := range []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/api/v1/app"},
		{method: http.MethodGet, path: "/api/v1/profile/current"},
		{method: http.MethodGet, path: "/api/v1/announcement-fields"},
		{method: http.MethodPost, path: "/api/v1/announcement-fields", body: `{"fields":[]}`},
		{method: http.MethodPost, path: "/api/v1/announce", body: `{}`},
		{method: http.MethodPost, path: "/api/v1/announce/confirm", body: `{"confirmationId":"missing"}`},
		{method: http.MethodGet, path: "/api/v1/announce/status"},
		{method: http.MethodPost, path: "/api/v1/end-stream", body: `{}`},
		{method: http.MethodGet, path: "/api/v1/end-stream/status"},
	} {
		t.Run(endpoint.method+" "+endpoint.path, func(t *testing.T) {
			request := httptest.NewRequest(endpoint.method, endpoint.path, strings.NewReader(endpoint.body))
			request.Host = "127.0.0.1"
			if endpoint.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			response := httptest.NewRecorder()

			service.Handler().ServeHTTP(response, request)

			if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Fatalf("expected nosniff header, got %q", got)
			}
			if got := response.Header().Get("Cache-Control"); got != "no-store" {
				t.Fatalf("expected no-store cache header, got %q", got)
			}
		})
	}
}

func TestActionsRemainOutOfScope(t *testing.T) {
	service := newTestService()
	for _, endpoint := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/actions"},
		{method: http.MethodPost, path: "/api/v1/actions"},
		{method: http.MethodPost, path: "/api/v1/action"},
		{method: http.MethodPost, path: "/api/v1/execute"},
		{method: http.MethodPost, path: "/api/v1/command"},
	} {
		t.Run(endpoint.method+" "+endpoint.path, func(t *testing.T) {
			request := httptest.NewRequest(endpoint.method, endpoint.path, strings.NewReader(`{}`))
			request.Host = "127.0.0.1"
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()

			service.Handler().ServeHTTP(response, request)

			if response.Code != http.StatusNotFound {
				t.Fatalf("expected generic action endpoint to remain absent, got %d", response.Code)
			}
		})
	}
}

func TestStatusIncludesDashboardEnhancements(t *testing.T) {
	service := newTestService()
	service.profiles = &fakeProfiles{profiles: []domain.AnnouncementProfile{{
		ID:                                 "music",
		Name:                               "Music Stream",
		DefaultStreamTitle:                 "Late Night Music",
		DefaultStreamURL:                   "https://example.com/live",
		DefaultCategory:                    "Music",
		DefaultHashtags:                    "#music #live",
		DefaultAnnouncementImageDataURL:    "data:image/png;base64,announcement-secret",
		DefaultBlueskyCardThumbnailDataURL: "data:image/png;base64,thumbnail-secret",
		DefaultGroupID:                     "main-announcements",
	}}}
	service.UseDestinationGroups(fakeDestinationGroups{
		groups: map[string]domain.DestinationGroup{
			"main-announcements": {ID: "main-announcements", Name: "Main Announcements"},
		},
		destinations: map[string][]domain.Destination{
			"main-announcements": {
				{ID: "discord-main", Platform: domain.PlatformDiscord, Name: "Discord", Enabled: true, Template: "Going live: {{title}}", ConfigJSON: `{"webhook":"secret"}`},
				{ID: "bluesky-main", Platform: domain.PlatformBluesky, Name: "Bluesky", Enabled: true, Template: "Live now {{url}}", ConfigJSON: `{"token":"secret"}`},
				{ID: "discord-backup", Platform: domain.PlatformDiscord, Name: "Discord Backup", Enabled: false, Template: "disabled template", ConfigJSON: `{"webhook":"disabled-secret"}`},
			},
		},
	})
	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/profile", map[string]string{"profile": "Music Stream"}, http.StatusOK, nil)

	payload := requestRawJSON(t, service.Handler(), http.MethodGet, "/api/v1/status", nil, http.StatusOK)
	var status StatusResponse
	if err := json.Unmarshal(payload, &status); err != nil {
		t.Fatalf("decode status: %v", err)
	}

	if !status.Healthy || status.ActiveProfile != "Music Stream" || status.ActiveProfileID != "music" {
		t.Fatalf("expected dashboard status fields, got %+v", status)
	}
	if status.StreamTitle != "Late Night Music" || status.StreamURL != "https://example.com/live" || status.Category != "Music" || status.Hashtags != "#music #live" {
		t.Fatalf("expected profile display defaults, got %+v", status)
	}
	if status.DestinationGroup != "Main Announcements" || status.DestinationGroupID != "main-announcements" {
		t.Fatalf("expected selected destination group, got %+v", status)
	}
	if status.DestinationCount != 3 || status.EnabledDestinationCount != 2 {
		t.Fatalf("expected group destination counts, got %+v", status)
	}
	if strings.Join(status.DestinationPlatforms, ",") != "bluesky,discord" {
		t.Fatalf("expected unique sorted enabled platforms, got %+v", status.DestinationPlatforms)
	}
	if status.Template != "2 destination templates" || status.TemplateCount != 2 {
		t.Fatalf("expected destination template summary, got %+v", status)
	}
	if status.Image != "Announcement image + Bluesky card thumbnail" {
		t.Fatalf("expected safe image summary, got %+v", status)
	}
	body := string(payload)
	for _, secret := range []string{"webhook", "token", "announcement-secret", "thumbnail-secret", "data:image"} {
		if strings.Contains(body, secret) {
			t.Fatalf("status leaked private value %q in %s", secret, body)
		}
	}
}

func TestStatusImageSummaryVariants(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*domain.AnnouncementProfile)
		wantImage string
	}{
		{
			name: "announcement image only",
			configure: func(profile *domain.AnnouncementProfile) {
				profile.DefaultAnnouncementImageURL = "https://example.com/card.png"
			},
			wantImage: "Announcement image",
		},
		{
			name: "bluesky thumbnail only",
			configure: func(profile *domain.AnnouncementProfile) {
				profile.DefaultBlueskyCardThumbnailURL = "https://example.com/thumb.png"
			},
			wantImage: "Bluesky card thumbnail",
		},
		{
			name: "both images",
			configure: func(profile *domain.AnnouncementProfile) {
				profile.DefaultAnnouncementImageURL = "https://example.com/card.png"
				profile.DefaultBlueskyCardThumbnailURL = "https://example.com/thumb.png"
			},
			wantImage: "Announcement image + Bluesky card thumbnail",
		},
		{
			name:      "no images",
			configure: func(*domain.AnnouncementProfile) {},
			wantImage: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile := domain.AnnouncementProfile{ID: "music", Name: "Music Stream"}
			test.configure(&profile)
			service := newTestService()
			service.profiles = &fakeProfiles{profiles: []domain.AnnouncementProfile{profile}}

			requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/profile", map[string]string{"profile": "Music Stream"}, http.StatusOK, nil)

			var status StatusResponse
			requestJSON(t, service.Handler(), http.MethodGet, "/api/v1/status", nil, http.StatusOK, &status)
			if status.Image != test.wantImage {
				t.Fatalf("expected image summary %q, got %+v", test.wantImage, status)
			}
		})
	}
}

func TestStatusTemplateSummaryVariants(t *testing.T) {
	tests := []struct {
		name          string
		destinations  []domain.Destination
		wantTemplate  string
		wantTemplates int
	}{
		{
			name:          "none",
			destinations:  []domain.Destination{{Enabled: true}, {Enabled: false, Template: "disabled"}},
			wantTemplate:  "No enabled templates",
			wantTemplates: 0,
		},
		{
			name:          "one",
			destinations:  []domain.Destination{{Enabled: true, Template: "one"}},
			wantTemplate:  "1 destination template",
			wantTemplates: 1,
		},
		{
			name:          "multiple",
			destinations:  []domain.Destination{{Enabled: true, Template: "one"}, {Enabled: true, Template: "two"}},
			wantTemplate:  "2 destination templates",
			wantTemplates: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newTestService()
			service.UseDestinationGroups(fakeDestinationGroups{
				groups: map[string]domain.DestinationGroup{
					domain.DefaultDestinationGroupID: {ID: domain.DefaultDestinationGroupID, Name: domain.DefaultDestinationGroupName},
				},
				destinations: map[string][]domain.Destination{
					domain.DefaultDestinationGroupID: test.destinations,
				},
			})

			requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/profile", map[string]string{"profile": "Music Stream"}, http.StatusOK, nil)

			var status StatusResponse
			requestJSON(t, service.Handler(), http.MethodGet, "/api/v1/status", nil, http.StatusOK, &status)
			if status.Template != test.wantTemplate || status.TemplateCount != test.wantTemplates {
				t.Fatalf("expected template summary %q/%d, got %+v", test.wantTemplate, test.wantTemplates, status)
			}
		})
	}
}

func TestStatusFallsBackToAllDestinationCountWithoutGroupService(t *testing.T) {
	service := newTestService()
	service.UseDestinations(fakeDestinations{destinations: []domain.Destination{
		{ID: "discord-main", Name: "Discord"},
		{ID: "bluesky-main", Name: "Bluesky"},
	}})
	requestJSON(t, service.Handler(), http.MethodPost, "/api/v1/profile", map[string]string{"profile": "Music Stream"}, http.StatusOK, nil)

	var status StatusResponse
	requestJSON(t, service.Handler(), http.MethodGet, "/api/v1/status", nil, http.StatusOK, &status)

	if !status.Healthy || status.ActiveProfile != "Music Stream" || status.DestinationCount != 2 {
		t.Fatalf("expected fallback dashboard status fields, got %+v", status)
	}
	if status.EnabledDestinationCount != 0 || status.Template != "" || len(status.DestinationPlatforms) != 0 {
		t.Fatalf("expected group-only summaries to be omitted without group service, got %+v", status)
	}
}

func newTestService() *Service {
	return newTestServiceWithMode("")
}

func newTestServiceWithMode(mode string) *Service {
	return NewService(AppInfo{
		AppID:    "streamsignal",
		Name:     "StreamSignal",
		Version:  "0.4.0",
		Mode:     mode,
		Protocol: ProtocolVersion,
	}, fakeSettings{}, &fakeProfiles{profiles: []domain.AnnouncementProfile{{ID: "music", Name: "Music Stream"}}}, fakeRecovery{})
}

type fakeSettings struct {
	err error
}

func (f fakeSettings) Load(context.Context) (domain.AppSettings, error) {
	if f.err != nil {
		return domain.AppSettings{}, f.err
	}
	return domain.DefaultAppSettings(), nil
}

type fakeProfiles struct {
	profiles   []domain.AnnouncementProfile
	listErr    error
	markErr    error
	markedUsed []string
}

func (f *fakeProfiles) List(context.Context) ([]domain.AnnouncementProfile, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.profiles, nil
}

func (f *fakeProfiles) MarkUsed(_ context.Context, id string) error {
	if f.markErr != nil {
		return f.markErr
	}
	for _, profile := range f.profiles {
		if profile.ID == id {
			f.markedUsed = append(f.markedUsed, id)
			return nil
		}
	}
	return errors.New("missing profile")
}

type fakeRecovery struct {
	sessions []domain.ActiveLiveNowSession
	err      error
}

func (f fakeRecovery) ListPending(context.Context) ([]domain.ActiveLiveNowSession, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.sessions, nil
}

type fakeDestinations struct {
	destinations []domain.Destination
	err          error
}

func (f fakeDestinations) List(context.Context) ([]domain.Destination, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.destinations, nil
}

type fakeAnnouncementFields struct {
	fields []domain.AnnouncementField
	values map[string]string
	err    error
}

func (f *fakeAnnouncementFields) List(context.Context) ([]domain.AnnouncementField, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.fields, nil
}

func (f *fakeAnnouncementFields) UpdateValues(_ context.Context, values map[string]string) error {
	if f.err != nil {
		return f.err
	}
	known := make(map[string]struct{}, len(f.fields))
	for _, field := range f.fields {
		known[field.ID] = struct{}{}
	}
	if f.values == nil {
		f.values = make(map[string]string, len(values))
	}
	for id, value := range values {
		if _, ok := known[id]; !ok {
			return errors.New("missing field")
		}
		f.values[id] = value
		for index := range f.fields {
			if f.fields[index].ID == id {
				f.fields[index].Value = value
			}
		}
	}
	return nil
}

type fakeDestinationGroups struct {
	groups       map[string]domain.DestinationGroup
	destinations map[string][]domain.Destination
	getErr       error
	listErr      error
}

func (f fakeDestinationGroups) Get(_ context.Context, id string) (domain.DestinationGroup, error) {
	if f.getErr != nil {
		return domain.DestinationGroup{}, f.getErr
	}
	if group, ok := f.groups[id]; ok {
		return group, nil
	}
	return domain.DestinationGroup{}, errors.New("missing group")
}

func (f fakeDestinationGroups) ListDestinations(_ context.Context, groupID string) ([]domain.Destination, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.destinations[groupID], nil
}

type fakeExecution struct {
	summary                domain.ExecutionSummary
	forceSummary           domain.ExecutionSummary
	endStreamSummary       domain.ExecutionSummary
	err                    error
	forceErr               error
	endStreamErr           error
	announcements          []domain.Announcement
	forceAnnouncements     []domain.Announcement
	endStreamAnnouncements []domain.Announcement
}

func (f *fakeExecution) GoLive(_ context.Context, announcement domain.Announcement) (domain.ExecutionSummary, error) {
	f.announcements = append(f.announcements, announcement)
	if f.err != nil {
		return domain.ExecutionSummary{}, f.err
	}
	return f.summary, nil
}

func (f *fakeExecution) ForceGoLive(_ context.Context, announcement domain.Announcement) (domain.ExecutionSummary, error) {
	f.forceAnnouncements = append(f.forceAnnouncements, announcement)
	if f.forceErr != nil {
		return domain.ExecutionSummary{}, f.forceErr
	}
	return f.forceSummary, nil
}

func (f *fakeExecution) EndStream(_ context.Context, announcement domain.Announcement) (domain.ExecutionSummary, error) {
	f.endStreamAnnouncements = append(f.endStreamAnnouncements, announcement)
	if f.endStreamErr != nil {
		return domain.ExecutionSummary{}, f.endStreamErr
	}
	return f.endStreamSummary, nil
}

func getJSON(t *testing.T, url string, target any) {
	t.Helper()
	response, err := http.Get(url)
	if err != nil {
		t.Fatalf("get %s: %v", url, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("get %s returned %d", url, response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
}

func requestJSON(t *testing.T, handler http.Handler, method string, path string, body any, wantStatus int, target any) {
	t.Helper()
	responseBody := requestRawJSON(t, handler, method, path, body, wantStatus)
	if target != nil {
		if err := json.Unmarshal(responseBody, target); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	}
}

func requestRawJSON(t *testing.T, handler http.Handler, method string, path string, body any, wantStatus int) []byte {
	t.Helper()
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatalf("encode request: %v", err)
		}
	}
	request := httptest.NewRequest(method, path, &payload)
	request.Host = "127.0.0.1"
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != wantStatus {
		t.Fatalf("%s %s returned %d, want %d, body %s", method, path, response.Code, wantStatus, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("expected JSON content type, got %q", got)
	}
	return response.Body.Bytes()
}

func assertJSONFields(t *testing.T, payload []byte, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	for key, wantValue := range want {
		gotValue, ok := got[key]
		if !ok {
			t.Fatalf("expected JSON field %q in %s", key, payload)
		}
		if fmt.Sprint(gotValue) != fmt.Sprint(wantValue) {
			t.Fatalf("field %q = %v, want %v in %s", key, gotValue, wantValue, payload)
		}
	}
}

func assertAbsentJSONFields(t *testing.T, payload []byte, fields ...string) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	for _, field := range fields {
		if _, ok := got[field]; ok {
			t.Fatalf("field %q should be absent in %s", field, payload)
		}
	}
}
