package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/ports"

	"github.com/google/uuid"
)

const (
	maxAnnouncementProfileNameLength        = 80
	maxAnnouncementProfileDescriptionLength = 250
)

type AnnouncementProfileService struct {
	repository ports.AnnouncementProfileRepository
	groups     ports.DestinationGroupRepository
	clock      Clock
}

func NewAnnouncementProfileService(repository ports.AnnouncementProfileRepository, groups ports.DestinationGroupRepository) *AnnouncementProfileService {
	return &AnnouncementProfileService{
		repository: repository,
		groups:     groups,
		clock:      systemClock{},
	}
}

func (s *AnnouncementProfileService) List(ctx context.Context) ([]domain.AnnouncementProfile, error) {
	return s.repository.List(ctx)
}

func (s *AnnouncementProfileService) Get(ctx context.Context, id string) (domain.AnnouncementProfile, error) {
	if strings.TrimSpace(id) == "" {
		return domain.AnnouncementProfile{}, fmt.Errorf("announcement profile id is required")
	}
	return s.repository.Get(ctx, id)
}

func (s *AnnouncementProfileService) Save(ctx context.Context, profile domain.AnnouncementProfile) (domain.AnnouncementProfile, error) {
	profile = normalizeAnnouncementProfile(profile)
	if err := validateAnnouncementProfile(profile); err != nil {
		return domain.AnnouncementProfile{}, err
	}
	if err := s.ensureUniqueName(ctx, profile); err != nil {
		return domain.AnnouncementProfile{}, err
	}
	if err := s.ensureGroupExists(ctx, profile.DefaultGroupID); err != nil {
		return domain.AnnouncementProfile{}, err
	}

	now := s.clock.Now()
	if profile.ID == "" {
		profile.ID = uuid.NewString()
		profile.CreatedAt = now
	} else {
		existing, err := s.repository.Get(ctx, profile.ID)
		if err == nil {
			profile.CreatedAt = existing.CreatedAt
		} else if !errors.Is(err, sql.ErrNoRows) {
			return domain.AnnouncementProfile{}, err
		} else if profile.CreatedAt.IsZero() {
			profile.CreatedAt = now
		}
	}
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now

	if err := s.repository.Save(ctx, profile); err != nil {
		return domain.AnnouncementProfile{}, err
	}
	return profile, nil
}

func (s *AnnouncementProfileService) Duplicate(ctx context.Context, id string) (domain.AnnouncementProfile, error) {
	source, err := s.Get(ctx, id)
	if err != nil {
		return domain.AnnouncementProfile{}, err
	}
	clone := source
	clone.ID = ""
	clone.Name = s.duplicateName(ctx, source.Name)
	clone.CreatedAt = s.clock.Now()
	clone.UpdatedAt = clone.CreatedAt
	return s.Save(ctx, clone)
}

func (s *AnnouncementProfileService) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("announcement profile id is required")
	}
	return s.repository.Delete(ctx, id)
}

func normalizeAnnouncementProfile(profile domain.AnnouncementProfile) domain.AnnouncementProfile {
	profile.ID = strings.TrimSpace(profile.ID)
	profile.Name = strings.TrimSpace(profile.Name)
	profile.Description = strings.TrimSpace(profile.Description)
	profile.DefaultStreamURL = strings.TrimSpace(profile.DefaultStreamURL)
	profile.DefaultCategory = strings.TrimSpace(profile.DefaultCategory)
	profile.DefaultMessage = strings.TrimSpace(profile.DefaultMessage)
	profile.DefaultHashtags = strings.TrimSpace(profile.DefaultHashtags)
	profile.DefaultAnnouncementImageURL = strings.TrimSpace(profile.DefaultAnnouncementImageURL)
	profile.DefaultAnnouncementImageDataURL = strings.TrimSpace(profile.DefaultAnnouncementImageDataURL)
	profile.DefaultBlueskyCardThumbnailURL = strings.TrimSpace(profile.DefaultBlueskyCardThumbnailURL)
	profile.DefaultBlueskyCardThumbnailDataURL = strings.TrimSpace(profile.DefaultBlueskyCardThumbnailDataURL)
	profile.DefaultGroupID = strings.TrimSpace(profile.DefaultGroupID)
	return profile
}

func validateAnnouncementProfile(profile domain.AnnouncementProfile) error {
	if profile.Name == "" {
		return fmt.Errorf("announcement profile name is required")
	}
	if len(profile.Name) > maxAnnouncementProfileNameLength {
		return fmt.Errorf("announcement profile name must be %d characters or less", maxAnnouncementProfileNameLength)
	}
	if len(profile.Description) > maxAnnouncementProfileDescriptionLength {
		return fmt.Errorf("announcement profile description must be %d characters or less", maxAnnouncementProfileDescriptionLength)
	}
	if err := validateOptionalHTTPURL(profile.DefaultStreamURL, "default stream URL"); err != nil {
		return err
	}
	if err := validateOptionalHTTPURL(profile.DefaultAnnouncementImageURL, "default announcement image URL"); err != nil {
		return err
	}
	if err := validateOptionalHTTPURL(profile.DefaultBlueskyCardThumbnailURL, "default Bluesky card thumbnail URL"); err != nil {
		return err
	}
	return nil
}

func validateOptionalHTTPURL(value string, label string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return fmt.Errorf("%s must be a valid URL", label)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use http or https", label)
	}
	if parsed.Host == "" {
		return fmt.Errorf("%s must include a host", label)
	}
	return nil
}

func (s *AnnouncementProfileService) ensureUniqueName(ctx context.Context, profile domain.AnnouncementProfile) error {
	existing, err := s.repository.GetByName(ctx, profile.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.ID != profile.ID {
		return fmt.Errorf("announcement profile name must be unique")
	}
	return nil
}

func (s *AnnouncementProfileService) ensureGroupExists(ctx context.Context, groupID string) error {
	if groupID == "" {
		return nil
	}
	if s.groups == nil {
		return nil
	}
	if _, err := s.groups.Get(ctx, groupID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("default destination group does not exist")
		}
		return err
	}
	return nil
}

func (s *AnnouncementProfileService) duplicateName(ctx context.Context, name string) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = "Announcement Profile"
	}
	for index := 2; index < 100; index++ {
		candidate := fmt.Sprintf("%s Copy", base)
		if index > 2 {
			candidate = fmt.Sprintf("%s Copy %d", base, index)
		}
		if len(candidate) > maxAnnouncementProfileNameLength {
			candidate = candidate[:maxAnnouncementProfileNameLength]
		}
		if _, err := s.repository.GetByName(ctx, candidate); errors.Is(err, sql.ErrNoRows) {
			return candidate
		}
	}
	return uuid.NewString()
}
