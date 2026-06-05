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

	"StreamSignal/internal/domain"
)

func TestReadOnlySIPV1ContractForLivePanel(t *testing.T) {
	service := newTestService()
	handler := service.Handler()

	appPayload := requestRawJSON(t, handler, http.MethodGet, "/api/v1/app", nil, http.StatusOK)
	assertJSONFields(t, appPayload, map[string]any{
		"appId":           "streamsignal",
		"name":            "StreamSignal",
		"version":         "0.3.1",
		"mode":            "standalone",
		"protocolVersion": "1.0",
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

func TestReadOnlyEndpointsRejectNonGETMethods(t *testing.T) {
	service := newTestService()
	for _, path := range []string{"/api/v1/app", "/api/v1/health", "/api/v1/capabilities", "/api/v1/status", "/api/v1/profiles"} {
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
