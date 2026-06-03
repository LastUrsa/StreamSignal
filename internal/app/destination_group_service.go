package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/ports"

	"github.com/google/uuid"
)

const (
	maxDestinationGroupNameLength        = 80
	maxDestinationGroupDescriptionLength = 250
)

type DestinationGroupService struct {
	repository ports.DestinationGroupRepository
	clock      Clock
}

func NewDestinationGroupService(repository ports.DestinationGroupRepository) *DestinationGroupService {
	return &DestinationGroupService{
		repository: repository,
		clock:      systemClock{},
	}
}

func (s *DestinationGroupService) List(ctx context.Context) ([]domain.DestinationGroup, error) {
	return s.repository.List(ctx)
}

func (s *DestinationGroupService) Get(ctx context.Context, id string) (domain.DestinationGroup, error) {
	if strings.TrimSpace(id) == "" {
		return domain.DestinationGroup{}, fmt.Errorf("destination group id is required")
	}
	return s.repository.Get(ctx, id)
}

func (s *DestinationGroupService) GetDestinations(ctx context.Context, groupID string) ([]domain.Destination, error) {
	if strings.TrimSpace(groupID) == "" {
		groupID = domain.DefaultDestinationGroupID
	}
	return s.repository.ListDestinations(ctx, groupID)
}

func (s *DestinationGroupService) GetGroupIDsForDestination(ctx context.Context, destinationID string) ([]string, error) {
	if strings.TrimSpace(destinationID) == "" {
		return nil, fmt.Errorf("destination id is required")
	}
	return s.repository.ListGroupIDsForDestination(ctx, destinationID)
}

func (s *DestinationGroupService) Save(ctx context.Context, group domain.DestinationGroup) (domain.DestinationGroup, error) {
	group.Name = strings.TrimSpace(group.Name)
	group.Description = strings.TrimSpace(group.Description)
	if err := validateDestinationGroup(group); err != nil {
		return domain.DestinationGroup{}, err
	}
	if err := s.ensureUniqueName(ctx, group); err != nil {
		return domain.DestinationGroup{}, err
	}

	now := s.clock.Now()
	if group.ID == "" {
		group.ID = uuid.NewString()
		group.CreatedAt = now
	} else {
		existing, err := s.repository.Get(ctx, group.ID)
		if err == nil {
			group.CreatedAt = existing.CreatedAt
		} else if !errors.Is(err, sql.ErrNoRows) {
			return domain.DestinationGroup{}, err
		} else if group.CreatedAt.IsZero() {
			group.CreatedAt = now
		}
	}
	if group.CreatedAt.IsZero() {
		group.CreatedAt = now
	}
	group.UpdatedAt = now

	if err := s.repository.Save(ctx, group); err != nil {
		return domain.DestinationGroup{}, err
	}
	return group, nil
}

func (s *DestinationGroupService) Duplicate(ctx context.Context, id string) (domain.DestinationGroup, error) {
	source, err := s.Get(ctx, id)
	if err != nil {
		return domain.DestinationGroup{}, err
	}
	clone := domain.DestinationGroup{
		Name:        s.duplicateName(ctx, source.Name),
		Description: source.Description,
	}
	saved, err := s.Save(ctx, clone)
	if err != nil {
		return domain.DestinationGroup{}, err
	}
	destinations, err := s.repository.ListDestinations(ctx, source.ID)
	if err != nil {
		return domain.DestinationGroup{}, err
	}
	destinationIDs := make([]string, 0, len(destinations))
	for _, destination := range destinations {
		destinationIDs = append(destinationIDs, destination.ID)
	}
	if err := s.repository.SetDestinations(ctx, saved.ID, destinationIDs); err != nil {
		return domain.DestinationGroup{}, err
	}
	return saved, nil
}

func (s *DestinationGroupService) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("destination group id is required")
	}
	if id == domain.DefaultDestinationGroupID {
		return fmt.Errorf("%s cannot be deleted", domain.DefaultDestinationGroupName)
	}
	return s.repository.Delete(ctx, id)
}

func (s *DestinationGroupService) SetDestinations(ctx context.Context, groupID string, destinationIDs []string) error {
	if strings.TrimSpace(groupID) == "" {
		return fmt.Errorf("destination group id is required")
	}
	return s.repository.SetDestinations(ctx, groupID, destinationIDs)
}

func (s *DestinationGroupService) SetGroupsForDestination(ctx context.Context, destinationID string, groupIDs []string) error {
	if strings.TrimSpace(destinationID) == "" {
		return fmt.Errorf("destination id is required")
	}
	if len(groupIDs) == 0 {
		groupIDs = []string{domain.DefaultDestinationGroupID}
	}
	return s.repository.SetGroupsForDestination(ctx, destinationID, groupIDs)
}

func (s *DestinationGroupService) AddDestination(ctx context.Context, groupID string, destinationID string) error {
	if strings.TrimSpace(groupID) == "" {
		return fmt.Errorf("destination group id is required")
	}
	if strings.TrimSpace(destinationID) == "" {
		return fmt.Errorf("destination id is required")
	}
	return s.repository.AddDestination(ctx, groupID, destinationID)
}

func (s *DestinationGroupService) RemoveDestination(ctx context.Context, groupID string, destinationID string) error {
	if strings.TrimSpace(groupID) == "" {
		return fmt.Errorf("destination group id is required")
	}
	if strings.TrimSpace(destinationID) == "" {
		return fmt.Errorf("destination id is required")
	}
	return s.repository.RemoveDestination(ctx, groupID, destinationID)
}

func validateDestinationGroup(group domain.DestinationGroup) error {
	if group.Name == "" {
		return fmt.Errorf("destination group name is required")
	}
	if len(group.Name) > maxDestinationGroupNameLength {
		return fmt.Errorf("destination group name must be %d characters or less", maxDestinationGroupNameLength)
	}
	if len(group.Description) > maxDestinationGroupDescriptionLength {
		return fmt.Errorf("destination group description must be %d characters or less", maxDestinationGroupDescriptionLength)
	}
	return nil
}

func (s *DestinationGroupService) ensureUniqueName(ctx context.Context, group domain.DestinationGroup) error {
	existing, err := s.repository.GetByName(ctx, group.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.ID != group.ID {
		return fmt.Errorf("destination group name must be unique")
	}
	return nil
}

func (s *DestinationGroupService) duplicateName(ctx context.Context, name string) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = "Destination Group"
	}
	for index := 2; index < 100; index++ {
		candidate := fmt.Sprintf("%s Copy", base)
		if index > 2 {
			candidate = fmt.Sprintf("%s Copy %d", base, index)
		}
		if len(candidate) > maxDestinationGroupNameLength {
			candidate = candidate[:maxDestinationGroupNameLength]
		}
		if _, err := s.repository.GetByName(ctx, candidate); errors.Is(err, sql.ErrNoRows) {
			return candidate
		}
	}
	return uuid.NewString()
}
