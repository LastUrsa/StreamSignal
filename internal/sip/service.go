package sip

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"StreamSignal/internal/domain"
	"github.com/google/uuid"
)

const ProtocolVersion = 1
const (
	StandaloneMode = "standalone"
	ServiceMode    = "service"

	CapabilityProfiles           = "profiles"
	CapabilityAnnouncementFields = "announcement-fields"
)

const announcementConfirmationTTL = 5 * time.Minute
const maxAnnouncementFieldOverrideValueLength = 500

type ProfileService interface {
	List(ctx context.Context) ([]domain.AnnouncementProfile, error)
	MarkUsed(ctx context.Context, id string) error
}

type SettingsService interface {
	Load(ctx context.Context) (domain.AppSettings, error)
}

type RecoveryService interface {
	ListPending(ctx context.Context) ([]domain.ActiveLiveNowSession, error)
}

type DestinationService interface {
	List(ctx context.Context) ([]domain.Destination, error)
}

type DestinationGroupService interface {
	Get(ctx context.Context, id string) (domain.DestinationGroup, error)
	ListDestinations(ctx context.Context, groupID string) ([]domain.Destination, error)
}

type ExecutionService interface {
	GoLive(ctx context.Context, announcement domain.Announcement) (domain.ExecutionSummary, error)
	ForceGoLive(ctx context.Context, announcement domain.Announcement) (domain.ExecutionSummary, error)
	EndStream(ctx context.Context, announcement domain.Announcement) (domain.ExecutionSummary, error)
}

type AnnouncementFieldService interface {
	List(ctx context.Context) ([]domain.AnnouncementField, error)
	UpdateValues(ctx context.Context, values map[string]string) error
}

type pendingAnnouncementConfirmation struct {
	id           string
	announcement domain.Announcement
	expiresAt    time.Time
}

type AppInfo struct {
	AppID    string
	Name     string
	Version  string
	Mode     string
	Protocol int
}

type Service struct {
	info         AppInfo
	settings     SettingsService
	profiles     ProfileService
	recovery     RecoveryService
	destinations DestinationService
	groups       DestinationGroupService
	execution    ExecutionService
	fields       AnnouncementFieldService

	mu                              sync.RWMutex
	activeProfile                   *domain.AnnouncementProfile
	lastAnnouncement                AnnounceStatusResponse
	lastEndStream                   EndStreamStatusResponse
	pendingAnnouncementConfirmation *pendingAnnouncementConfirmation
}

func NewService(info AppInfo, settings SettingsService, profiles ProfileService, recovery RecoveryService) *Service {
	if info.Protocol == 0 {
		info.Protocol = ProtocolVersion
	}
	if info.Mode == "" {
		info.Mode = StandaloneMode
	}
	return &Service{
		info:     info,
		settings: settings,
		profiles: profiles,
		recovery: recovery,
	}
}

func (s *Service) UseDestinations(destinations DestinationService) {
	if s == nil {
		return
	}
	s.destinations = destinations
}

func (s *Service) UseDestinationGroups(groups DestinationGroupService) {
	if s == nil {
		return
	}
	s.groups = groups
}

func (s *Service) UseExecution(execution ExecutionService) {
	if s == nil {
		return
	}
	s.execution = execution
}

func (s *Service) UseAnnouncementFields(fields AnnouncementFieldService) {
	if s == nil {
		return
	}
	s.fields = fields
}

func (s *Service) App(ctx context.Context) (AppResponse, error) {
	if s == nil {
		return AppResponse{}, fmt.Errorf("sip service is unavailable")
	}
	return AppResponse{
		AppID:           s.info.AppID,
		AppName:         s.info.Name,
		Name:            s.info.Name,
		Version:         s.info.Version,
		Mode:            s.info.Mode,
		ProtocolVersion: s.info.Protocol,
		Capabilities:    sipCapabilities(),
	}, nil
}

func (s *Service) Health(ctx context.Context) (HealthResponse, error) {
	if s == nil {
		return HealthResponse{Status: HealthError, Message: "SIP service is unavailable."}, nil
	}
	if s.settings == nil || s.profiles == nil {
		return HealthResponse{Status: HealthError, Message: "Core SIP dependencies are unavailable."}, nil
	}
	if _, err := s.settings.Load(ctx); err != nil {
		return HealthResponse{Status: HealthError, Message: "StreamSignal settings are unavailable."}, nil
	}
	if _, err := s.profiles.List(ctx); err != nil {
		return HealthResponse{Status: HealthError, Message: "Announcement profiles are unavailable."}, nil
	}
	if s.recovery != nil {
		sessions, err := s.recovery.ListPending(ctx)
		if err != nil {
			return HealthResponse{Status: HealthDegraded, Message: "Live Now recovery state could not be checked."}, nil
		}
		if len(sessions) > 0 {
			return HealthResponse{Status: HealthDegraded, Message: "Pending Live Now recovery sessions need attention."}, nil
		}
	}
	return HealthResponse{Status: HealthReady, Message: "StreamSignal is ready for local SIP participation."}, nil
}

func (s *Service) Capabilities(ctx context.Context) (CapabilitiesResponse, error) {
	return CapabilitiesResponse{
		Capabilities:            sipCapabilities(),
		SupportsProfiles:        true,
		SupportsStatusReporting: true,
		SupportsAnnouncements:   true,
	}, nil
}

func (s *Service) AnnouncementFields(ctx context.Context) (AnnouncementFieldsResponse, error) {
	if s == nil || s.fields == nil {
		return AnnouncementFieldsResponse{}, fmt.Errorf("announcement fields are unavailable")
	}
	fields, err := s.fields.List(ctx)
	if err != nil {
		return AnnouncementFieldsResponse{}, err
	}
	response := AnnouncementFieldsResponse{Fields: make([]AnnouncementFieldResponse, 0, len(fields))}
	for _, field := range fields {
		response.Fields = append(response.Fields, AnnouncementFieldResponse{
			ID:    field.ID,
			Name:  field.Name,
			Value: field.Value,
		})
	}
	return response, nil
}

func (s *Service) UpdateAnnouncementFields(ctx context.Context, request UpdateAnnouncementFieldsRequest) (SuccessResponse, error) {
	if s == nil || s.fields == nil {
		return SuccessResponse{}, fmt.Errorf("announcement fields are unavailable")
	}
	values := make(map[string]string, len(request.Fields))
	for _, field := range request.Fields {
		id := strings.TrimSpace(strings.ToLower(field.ID))
		if id == "" {
			return SuccessResponse{}, ErrInvalidRequest
		}
		values[id] = field.Value
	}
	if err := s.fields.UpdateValues(ctx, values); err != nil {
		return SuccessResponse{}, ErrInvalidRequest
	}
	return SuccessResponse{Success: true}, nil
}

func (s *Service) Status(ctx context.Context) (StatusResponse, error) {
	health, _ := s.Health(ctx)
	response := StatusResponse{
		State:   StatusIdle,
		Message: "Ready for announcement setup.",
		Healthy: health.Status == HealthReady,
	}
	switch health.Status {
	case HealthDegraded:
		response.State = StatusWarning
		response.Message = health.Message
		response.Healthy = true
	case HealthError:
		response.State = StatusError
		response.Message = health.Message
		response.Healthy = false
	}

	if active := s.ActiveProfile(); active != nil {
		response.ActiveProfile = active.Name
		response.ActiveProfileID = active.ID
		announcement := announcementFromProfile(*active)
		response.StreamTitle = announcement.StreamTitle
		response.StreamURL = announcement.StreamURL
		response.Category = announcement.Category
		response.Hashtags = announcement.Hashtags
		response.Image = statusImageSummary(announcement)

		groupID := strings.TrimSpace(announcement.DestinationGroupID)
		if groupID == "" {
			groupID = domain.DefaultDestinationGroupID
		}
		response.DestinationGroupID = groupID

		if s != nil && s.groups != nil {
			if group, err := s.groups.Get(ctx, groupID); err == nil {
				response.DestinationGroup = group.Name
			}
			if destinations, err := s.groups.ListDestinations(ctx, groupID); err == nil {
				applyDestinationStatusSummary(&response, destinations)
			}
			return response, nil
		}
	}
	if s != nil && s.destinations != nil {
		destinations, err := s.destinations.List(ctx)
		if err == nil {
			response.DestinationCount = len(destinations)
		}
	}
	return response, nil
}

func (s *Service) Profiles(ctx context.Context) (ProfilesResponse, error) {
	if s == nil || s.profiles == nil {
		return ProfilesResponse{}, fmt.Errorf("announcement profiles are unavailable")
	}
	profiles, err := s.profiles.List(ctx)
	if err != nil {
		return ProfilesResponse{}, err
	}
	names := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		if strings.TrimSpace(profile.Name) != "" {
			names = append(names, profile.Name)
		}
	}
	return ProfilesResponse{Profiles: names}, nil
}

func (s *Service) ActivateProfile(ctx context.Context, name string) (ProfileActivationResponse, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ProfileActivationResponse{}, ErrInvalidRequest
	}
	if s == nil || s.profiles == nil {
		return ProfileActivationResponse{}, fmt.Errorf("announcement profiles are unavailable")
	}

	profiles, err := s.profiles.List(ctx)
	if err != nil {
		return ProfileActivationResponse{}, err
	}
	for _, profile := range profiles {
		if strings.EqualFold(profile.Name, name) {
			if err := s.profiles.MarkUsed(ctx, profile.ID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ProfileActivationResponse{}, ErrProfileNotFound
				}
				return ProfileActivationResponse{}, err
			}
			s.setActiveProfile(profile)
			return ProfileActivationResponse{
				Success:   true,
				Profile:   profile.Name,
				ProfileID: profile.ID,
			}, nil
		}
	}
	return ProfileActivationResponse{}, ErrProfileNotFound
}

func (s *Service) CurrentProfile(ctx context.Context) (CurrentProfileResponse, error) {
	if s == nil {
		return CurrentProfileResponse{}, nil
	}
	active := s.ActiveProfile()
	if active == nil {
		return CurrentProfileResponse{}, nil
	}
	return CurrentProfileResponse{ID: active.ID, Name: active.Name}, nil
}

func (s *Service) Announce(ctx context.Context, request AnnounceRequest) (AnnounceResponse, error) {
	if s == nil {
		return AnnounceResponse{Success: false, Error: "SIP service is unavailable"}, nil
	}
	active := s.ActiveProfile()
	if active == nil {
		response := AnnounceResponse{Success: false, Error: "No active profile"}
		s.setLastAnnouncement(response, time.Now().UTC())
		s.clearPendingAnnouncementConfirmation()
		return response, nil
	}
	if s == nil || s.execution == nil {
		response := AnnounceResponse{Success: false, Error: "Announcement execution is unavailable"}
		s.setLastAnnouncement(response, time.Now().UTC())
		s.clearPendingAnnouncementConfirmation()
		return response, nil
	}

	announcement := s.announcementFromProfile(ctx, *active)
	if err := s.applyAnnouncementFieldOverrides(ctx, &announcement, request.Fields); err != nil {
		return AnnounceResponse{}, err
	}
	summary, err := s.execution.GoLive(ctx, announcement)
	if err != nil {
		response := AnnounceResponse{Success: false, Error: err.Error()}
		s.setLastAnnouncement(response, time.Now().UTC())
		s.clearPendingAnnouncementConfirmation()
		return response, nil
	}
	response := s.announceResponseFromSummary(summary, announcement)
	s.setLastAnnouncement(response, time.Now().UTC())
	if !response.RequiresConfirmation {
		s.clearPendingAnnouncementConfirmation()
	}
	return response, nil
}

func (s *Service) AnnounceStatus(ctx context.Context) (AnnounceStatusResponse, error) {
	if s == nil {
		return AnnounceStatusResponse{}, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastAnnouncement, nil
}

func (s *Service) ConfirmAnnouncement(ctx context.Context, confirmationID string) (AnnounceResponse, error) {
	if s == nil {
		return AnnounceResponse{Success: false, Error: "SIP service is unavailable"}, nil
	}
	if s.execution == nil {
		response := AnnounceResponse{Success: false, Error: "Announcement execution is unavailable"}
		s.setLastAnnouncement(response, time.Now().UTC())
		s.clearPendingAnnouncementConfirmation()
		return response, nil
	}

	pending, ok := s.consumePendingAnnouncementConfirmation(strings.TrimSpace(confirmationID), time.Now().UTC())
	if !ok {
		return AnnounceResponse{}, ErrInvalidRequest
	}

	summary, err := s.execution.ForceGoLive(ctx, pending.announcement)
	if err != nil {
		response := AnnounceResponse{Success: false, Error: err.Error()}
		s.setLastAnnouncement(response, time.Now().UTC())
		return response, nil
	}
	response := announceResponseFromSummary(summary)
	s.setLastAnnouncement(response, time.Now().UTC())
	return response, nil
}

func (s *Service) EndStream(ctx context.Context) (EndStreamResponse, error) {
	if s == nil {
		return EndStreamResponse{Success: false, Error: "SIP service is unavailable"}, nil
	}
	active := s.ActiveProfile()
	if active == nil {
		response := EndStreamResponse{Success: false, Error: "No active profile"}
		s.setLastEndStream(response, time.Now().UTC())
		return response, nil
	}
	if s.execution == nil {
		response := EndStreamResponse{Success: false, Error: "End stream execution is unavailable"}
		s.setLastEndStream(response, time.Now().UTC())
		return response, nil
	}

	summary, err := s.execution.EndStream(ctx, s.announcementFromProfile(ctx, *active))
	if err != nil {
		response := EndStreamResponse{Success: false, Error: err.Error()}
		s.setLastEndStream(response, time.Now().UTC())
		return response, nil
	}
	announcementResponse := announceResponseFromSummary(summary)
	response := EndStreamResponse{Success: announcementResponse.Success, Error: announcementResponse.Error}
	s.setLastEndStream(response, time.Now().UTC())
	return response, nil
}

func (s *Service) EndStreamStatus(ctx context.Context) (EndStreamStatusResponse, error) {
	if s == nil {
		return EndStreamStatusResponse{}, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastEndStream, nil
}

func (s *Service) ActiveProfile() *domain.AnnouncementProfile {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.activeProfile == nil {
		return nil
	}
	profile := *s.activeProfile
	return &profile
}

func (s *Service) setActiveProfile(profile domain.AnnouncementProfile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := profile
	s.activeProfile = &copy
}

func (s *Service) setLastAnnouncement(response AnnounceResponse, ranAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastAnnouncement = AnnounceStatusResponse{
		LastRun:              ranAt.Format(time.RFC3339),
		Success:              response.Success,
		RequiresConfirmation: response.RequiresConfirmation,
		ConfirmationID:       response.ConfirmationID,
		Error:                response.Error,
	}
}

func (s *Service) setLastEndStream(response EndStreamResponse, ranAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastEndStream = EndStreamStatusResponse{
		LastRun: ranAt.Format(time.RFC3339),
		Success: response.Success,
		Error:   response.Error,
	}
}

func announcementFromProfile(profile domain.AnnouncementProfile) domain.Announcement {
	return domain.Announcement{
		StreamTitle:                 profile.DefaultStreamTitle,
		StreamURL:                   profile.DefaultStreamURL,
		Category:                    profile.DefaultCategory,
		Message:                     profile.DefaultMessage,
		Hashtags:                    profile.DefaultHashtags,
		AnnouncementImageURL:        profile.DefaultAnnouncementImageURL,
		AnnouncementImageDataURL:    profile.DefaultAnnouncementImageDataURL,
		BlueskyCardThumbnailURL:     profile.DefaultBlueskyCardThumbnailURL,
		BlueskyCardThumbnailDataURL: profile.DefaultBlueskyCardThumbnailDataURL,
		DestinationGroupID:          profile.DefaultGroupID,
	}
}

func (s *Service) announcementFromProfile(ctx context.Context, profile domain.AnnouncementProfile) domain.Announcement {
	announcement := announcementFromProfile(profile)
	if s == nil || s.fields == nil {
		return announcement
	}
	fields, err := s.fields.List(ctx)
	if err != nil || len(fields) == 0 {
		return announcement
	}
	announcement.Fields = make(map[string]string, len(fields))
	for _, field := range fields {
		id := strings.TrimSpace(strings.ToLower(field.ID))
		if id != "" {
			announcement.Fields[id] = field.Value
		}
	}
	return announcement
}

func (s *Service) applyAnnouncementFieldOverrides(ctx context.Context, announcement *domain.Announcement, overrides []AnnouncementFieldOverride) error {
	if len(overrides) == 0 {
		return nil
	}
	if s == nil || s.fields == nil || announcement == nil {
		return ErrInvalidRequest
	}
	fields, err := s.fields.List(ctx)
	if err != nil {
		return ErrInvalidRequest
	}
	known := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		id := strings.TrimSpace(strings.ToLower(field.ID))
		if id != "" {
			known[id] = struct{}{}
		}
	}
	if announcement.Fields == nil {
		announcement.Fields = make(map[string]string, len(overrides))
	}
	for _, override := range overrides {
		id := strings.TrimSpace(strings.ToLower(override.ID))
		if id == "" {
			return ErrInvalidRequest
		}
		if _, ok := known[id]; !ok {
			return ErrInvalidRequest
		}
		value := strings.TrimSpace(override.Value)
		if len(value) > maxAnnouncementFieldOverrideValueLength {
			return ErrInvalidRequest
		}
		announcement.Fields[id] = value
	}
	return nil
}

func sipCapabilities() []string {
	return []string{CapabilityProfiles, CapabilityAnnouncementFields}
}

func applyDestinationStatusSummary(response *StatusResponse, destinations []domain.Destination) {
	if response == nil {
		return
	}
	response.DestinationCount = len(destinations)

	platforms := make(map[string]struct{})
	templateCount := 0
	enabledCount := 0
	for _, destination := range destinations {
		if !destination.Enabled {
			continue
		}
		enabledCount++
		if strings.TrimSpace(destination.Template) != "" {
			templateCount++
		}
		platform := strings.TrimSpace(string(destination.Platform))
		if platform != "" {
			platforms[platform] = struct{}{}
		}
	}
	response.EnabledDestinationCount = enabledCount
	response.TemplateCount = templateCount
	response.Template = statusTemplateSummary(templateCount)
	response.DestinationPlatforms = sortedStatusPlatforms(platforms)
}

func sortedStatusPlatforms(platforms map[string]struct{}) []string {
	if len(platforms) == 0 {
		return nil
	}
	labels := make([]string, 0, len(platforms))
	for platform := range platforms {
		labels = append(labels, platform)
	}
	sort.Strings(labels)
	return labels
}

func statusTemplateSummary(count int) string {
	switch count {
	case 0:
		return "No enabled templates"
	case 1:
		return "1 destination template"
	default:
		return fmt.Sprintf("%d destination templates", count)
	}
}

func statusImageSummary(announcement domain.Announcement) string {
	hasAnnouncementImage := strings.TrimSpace(announcement.AnnouncementImageURL) != "" || strings.TrimSpace(announcement.AnnouncementImageDataURL) != ""
	hasBlueskyThumbnail := strings.TrimSpace(announcement.BlueskyCardThumbnailURL) != "" || strings.TrimSpace(announcement.BlueskyCardThumbnailDataURL) != ""
	switch {
	case hasAnnouncementImage && hasBlueskyThumbnail:
		return "Announcement image + Bluesky card thumbnail"
	case hasAnnouncementImage:
		return "Announcement image"
	case hasBlueskyThumbnail:
		return "Bluesky card thumbnail"
	default:
		return ""
	}
}

func (s *Service) announceResponseFromSummary(summary domain.ExecutionSummary, announcement domain.Announcement) AnnounceResponse {
	if summary.RequiresDuplicateConfirmation {
		confirmationID := uuid.NewString()
		s.setPendingAnnouncementConfirmation(confirmationID, announcement, time.Now().UTC().Add(announcementConfirmationTTL))
		message := summary.DuplicateWarningMessage
		if strings.TrimSpace(message) == "" {
			message = "Announcement requires confirmation."
		}
		return AnnounceResponse{
			Success:              false,
			RequiresConfirmation: true,
			ConfirmationID:       confirmationID,
			Error:                message,
		}
	}
	return announceResponseFromSummary(summary)
}

func announceResponseFromSummary(summary domain.ExecutionSummary) AnnounceResponse {
	if summary.Status == domain.ExecutionSummaryStatusSuccess && !summary.RequiresDuplicateConfirmation {
		return AnnounceResponse{Success: true}
	}
	if summary.DuplicateWarningMessage != "" {
		return AnnounceResponse{Success: false, Error: summary.DuplicateWarningMessage}
	}
	for _, result := range summary.Results {
		if result.State != domain.ExecutionStateSuccess && strings.TrimSpace(result.Message) != "" {
			return AnnounceResponse{Success: false, Error: result.Message}
		}
	}
	return AnnounceResponse{Success: false, Error: "Announcement did not complete successfully"}
}

func (s *Service) setPendingAnnouncementConfirmation(id string, announcement domain.Announcement, expiresAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingAnnouncementConfirmation = &pendingAnnouncementConfirmation{
		id:           id,
		announcement: announcement,
		expiresAt:    expiresAt,
	}
}

func (s *Service) clearPendingAnnouncementConfirmation() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingAnnouncementConfirmation = nil
}

func (s *Service) consumePendingAnnouncementConfirmation(id string, now time.Time) (pendingAnnouncementConfirmation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pendingAnnouncementConfirmation == nil {
		return pendingAnnouncementConfirmation{}, false
	}
	pending := *s.pendingAnnouncementConfirmation
	if id == "" || pending.id != id || !now.Before(pending.expiresAt) {
		s.pendingAnnouncementConfirmation = nil
		return pendingAnnouncementConfirmation{}, false
	}
	s.pendingAnnouncementConfirmation = nil
	return pending, true
}
