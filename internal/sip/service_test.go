package sip

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"StreamSignal/internal/domain"
)

func TestHandlerReturnsIdentityAndCapabilities(t *testing.T) {
	service := newTestService()
	server := httptest.NewServer(service.Handler())
	defer server.Close()

	var app AppResponse
	getJSON(t, server.URL+"/api/v1/app", &app)
	if app.AppID != "streamsignal" || app.Name != "StreamSignal" || app.Version != "0.3.1" || app.Mode != "standalone" || app.ProtocolVersion != "1.0" {
		t.Fatalf("unexpected app response: %+v", app)
	}

	var capabilities CapabilitiesResponse
	getJSON(t, server.URL+"/api/v1/capabilities", &capabilities)
	if !capabilities.SupportsProfiles || !capabilities.SupportsStatusReporting || !capabilities.SupportsAnnouncements {
		t.Fatalf("expected core capabilities, got %+v", capabilities)
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

func TestHandlerRejectsNonLocalHost(t *testing.T) {
	service := newTestService()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/app", nil)
	request.Host = "example.com"
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
}

func TestHandlerSetsSecurityHeaders(t *testing.T) {
	service := newTestService()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/app", nil)
	request.Host = "127.0.0.1"
	response := httptest.NewRecorder()

	service.Handler().ServeHTTP(response, request)

	if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected nosniff header, got %q", got)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("expected no-store cache header, got %q", got)
	}
}

func TestActionsRemainOutOfScope(t *testing.T) {
	service := newTestService()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/actions", nil)
	request.Host = "127.0.0.1"
	response := httptest.NewRecorder()

	service.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected actions endpoint to remain absent, got %d", response.Code)
	}
}

func newTestService() *Service {
	return NewService(AppInfo{
		AppID:    "streamsignal",
		Name:     "StreamSignal",
		Version:  "0.3.1",
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
	if target != nil {
		if err := json.NewDecoder(response.Body).Decode(target); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	}
}
