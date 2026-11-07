package sip

import (
	"StreamSignal/internal/domain"
	"context"
)

type publicationSnapshotKey struct{}
type publicationSnapshot struct {
	profile      *domain.AnnouncementProfile
	announcement domain.Announcement
}

// Freeze the selected profile and default fields before accepting background work.
// A later profile switch must not change an already accepted announcement.
func (s *Service) preparePublication(ctx context.Context, route string) context.Context {
	if route != "/api/v1/announce" && route != "/api/v1/end-stream" {
		return ctx
	}
	profile, announcement := s.currentAnnouncement(ctx)
	return context.WithValue(ctx, publicationSnapshotKey{}, publicationSnapshot{profile, announcement})
}
func (s *Service) currentAnnouncement(ctx context.Context) (*domain.AnnouncementProfile, domain.Announcement) {
	if snapshot, ok := ctx.Value(publicationSnapshotKey{}).(publicationSnapshot); ok {
		return snapshot.profile, snapshot.announcement
	}
	profile := s.ActiveProfile()
	if profile == nil {
		return nil, domain.Announcement{}
	}
	return profile, s.announcementFromProfile(ctx, *profile)
}
