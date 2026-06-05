package sip

import "errors"

const (
	HealthReady    = "ready"
	HealthDegraded = "degraded"
	HealthError    = "error"

	StatusIdle    = "idle"
	StatusWarning = "warning"
	StatusError   = "error"
)

var (
	ErrInvalidRequest  = errors.New("InvalidRequest")
	ErrProfileNotFound = errors.New("ProfileNotFound")
	ErrForbidden       = errors.New("Forbidden")
)

type AppResponse struct {
	AppID           string `json:"appId"`
	Name            string `json:"name"`
	Version         string `json:"version"`
	Mode            string `json:"mode"`
	ProtocolVersion string `json:"protocolVersion"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type CapabilitiesResponse struct {
	SupportsProfiles        bool `json:"supportsProfiles"`
	SupportsStatusReporting bool `json:"supportsStatusReporting"`
	SupportsAnnouncements   bool `json:"supportsAnnouncements"`
}

type StatusResponse struct {
	State           string `json:"state"`
	Message         string `json:"message"`
	ActiveProfile   string `json:"activeProfile,omitempty"`
	ActiveProfileID string `json:"activeProfileId,omitempty"`
}

type ProfilesResponse struct {
	Profiles []string `json:"profiles"`
}

type ActivateProfileRequest struct {
	Profile string `json:"profile"`
}

type ProfileActivationResponse struct {
	Success   bool   `json:"success"`
	Profile   string `json:"profile,omitempty"`
	ProfileID string `json:"profileId,omitempty"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}
