package app

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/ports"
)

const maxAnnouncementFieldValueLength = 500

type AnnouncementFieldService struct {
	repository ports.AnnouncementFieldRepository
	clock      Clock
}

func NewAnnouncementFieldService(repository ports.AnnouncementFieldRepository) *AnnouncementFieldService {
	return &AnnouncementFieldService{
		repository: repository,
		clock:      systemClock{},
	}
}

func (s *AnnouncementFieldService) List(ctx context.Context) ([]domain.AnnouncementField, error) {
	return s.repository.List(ctx)
}

func (s *AnnouncementFieldService) UpdateValues(ctx context.Context, values map[string]string) error {
	if len(values) == 0 {
		return nil
	}
	normalized := make(map[string]string, len(values))
	for id, value := range values {
		id = strings.TrimSpace(strings.ToLower(id))
		if id == "" {
			return fmt.Errorf("announcement field id is required")
		}
		value = strings.TrimSpace(value)
		if len(value) > maxAnnouncementFieldValueLength {
			return fmt.Errorf("announcement field value must be %d characters or less", maxAnnouncementFieldValueLength)
		}
		normalized[id] = value
	}
	if err := s.repository.UpdateValues(ctx, normalized, s.clock.Now()); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("announcement field does not exist")
		}
		return err
	}
	return nil
}
