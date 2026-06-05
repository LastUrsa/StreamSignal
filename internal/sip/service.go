package sip

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"StreamSignal/internal/domain"
)

const ProtocolVersion = "1.0"
const StandaloneMode = "standalone"

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

type AppInfo struct {
	AppID    string
	Name     string
	Version  string
	Mode     string
	Protocol string
}

type Service struct {
	info     AppInfo
	settings SettingsService
	profiles ProfileService
	recovery RecoveryService

	mu            sync.RWMutex
	activeProfile *domain.AnnouncementProfile
}

func NewService(info AppInfo, settings SettingsService, profiles ProfileService, recovery RecoveryService) *Service {
	if info.Protocol == "" {
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

func (s *Service) App(ctx context.Context) (AppResponse, error) {
	if s == nil {
		return AppResponse{}, fmt.Errorf("sip service is unavailable")
	}
	return AppResponse{
		AppID:           s.info.AppID,
		Name:            s.info.Name,
		Version:         s.info.Version,
		Mode:            s.info.Mode,
		ProtocolVersion: s.info.Protocol,
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
		SupportsProfiles:        true,
		SupportsStatusReporting: true,
		SupportsAnnouncements:   true,
	}, nil
}

func (s *Service) Status(ctx context.Context) (StatusResponse, error) {
	health, _ := s.Health(ctx)
	response := StatusResponse{
		State:   StatusIdle,
		Message: "Ready for announcement setup.",
	}
	switch health.Status {
	case HealthDegraded:
		response.State = StatusWarning
		response.Message = health.Message
	case HealthError:
		response.State = StatusError
		response.Message = health.Message
	}

	if active := s.ActiveProfile(); active != nil {
		response.ActiveProfile = active.Name
		response.ActiveProfileID = active.ID
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

func (s *Service) ActiveProfile() *domain.AnnouncementProfile {
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
