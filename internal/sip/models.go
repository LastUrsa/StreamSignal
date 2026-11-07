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
	AppID           string   `json:"appId"`
	AppName         string   `json:"appName"`
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	Mode            string   `json:"mode"`
	ProtocolVersion int      `json:"protocolVersion"`
	Capabilities    []string `json:"capabilities"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type CapabilitiesResponse struct {
	Capabilities            []string `json:"capabilities"`
	SupportsProfiles        bool     `json:"supportsProfiles"`
	SupportsStatusReporting bool     `json:"supportsStatusReporting"`
	SupportsAnnouncements   bool     `json:"supportsAnnouncements"`
}

type AnnouncementFieldResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type AnnouncementFieldsResponse struct {
	Fields []AnnouncementFieldResponse `json:"fields"`
}

type UpdateAnnouncementFieldRequest struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type UpdateAnnouncementFieldsRequest struct {
	Fields []UpdateAnnouncementFieldRequest `json:"fields"`
}

type SuccessResponse struct {
	Success bool `json:"success"`
}

type StatusResponse struct {
	State                   string   `json:"state"`
	Message                 string   `json:"message"`
	ActiveProfile           string   `json:"activeProfile,omitempty"`
	ActiveProfileID         string   `json:"activeProfileId,omitempty"`
	Healthy                 bool     `json:"healthy"`
	DestinationCount        int      `json:"destinationCount,omitempty"`
	DestinationGroup        string   `json:"destinationGroup,omitempty"`
	DestinationGroupID      string   `json:"destinationGroupId,omitempty"`
	EnabledDestinationCount int      `json:"enabledDestinationCount,omitempty"`
	DestinationPlatforms    []string `json:"destinationPlatforms,omitempty"`
	Image                   string   `json:"image,omitempty"`
	Template                string   `json:"template,omitempty"`
	TemplateCount           int      `json:"templateCount,omitempty"`
	StreamTitle             string   `json:"streamTitle,omitempty"`
	StreamURL               string   `json:"streamUrl,omitempty"`
	Category                string   `json:"category,omitempty"`
	Hashtags                string   `json:"hashtags,omitempty"`
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

type CurrentProfileResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AnnouncementFieldOverride struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type AnnounceRequest struct {
	Fields []AnnouncementFieldOverride `json:"fields,omitempty"`
}

type AnnounceResponse struct {
	Results              []DiagnosticResult `json:"results,omitempty"`
	OperationOutcome     string             `json:"operationOutcome,omitempty"`
	ErrorCode            string             `json:"errorCode,omitempty"`
	Success              bool               `json:"success"`
	RequiresConfirmation bool               `json:"requiresConfirmation,omitempty"`
	ConfirmationID       string             `json:"confirmationId,omitempty"`
	Error                string             `json:"error,omitempty"`
}

type AnnounceStatusResponse struct {
	Results              []DiagnosticResult `json:"results,omitempty"`
	OperationOutcome     string             `json:"operationOutcome,omitempty"`
	ErrorCode            string             `json:"errorCode,omitempty"`
	LastRun              string             `json:"lastRun"`
	Success              bool               `json:"success"`
	RequiresConfirmation bool               `json:"requiresConfirmation,omitempty"`
	ConfirmationID       string             `json:"confirmationId,omitempty"`
	Error                string             `json:"error,omitempty"`
}

type AnnounceConfirmRequest struct {
	ConfirmationID string `json:"confirmationId"`
}

type EndStreamResponse struct {
	Results          []DiagnosticResult `json:"results,omitempty"`
	OperationOutcome string             `json:"operationOutcome,omitempty"`
	ErrorCode        string             `json:"errorCode,omitempty"`
	Success          bool               `json:"success"`
	Error            string             `json:"error,omitempty"`
}

type EndStreamStatusResponse struct {
	Results          []DiagnosticResult `json:"results,omitempty"`
	OperationOutcome string             `json:"operationOutcome,omitempty"`
	ErrorCode        string             `json:"errorCode,omitempty"`
	LastRun          string             `json:"lastRun"`
	Success          bool               `json:"success"`
	Error            string             `json:"error,omitempty"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}
