package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"StreamSignal/internal/domain"
)

type AnnouncementProfileRepository struct {
	db *sql.DB
}

func NewAnnouncementProfileRepository(db *sql.DB) *AnnouncementProfileRepository {
	return &AnnouncementProfileRepository{db: db}
}

func (r *AnnouncementProfileRepository) List(ctx context.Context) ([]domain.AnnouncementProfile, error) {
	rows, err := r.db.QueryContext(ctx, `
select id, name, description, default_stream_url, default_category, default_message, default_hashtags,
  default_announcement_image_url, default_announcement_image_data_url,
  default_bluesky_card_thumbnail_url, default_bluesky_card_thumbnail_data_url,
  coalesce(default_group_id, ''), created_at, updated_at
from announcement_profiles
order by lower(name) asc`)
	if err != nil {
		return nil, fmt.Errorf("list announcement profiles: %w", err)
	}
	defer rows.Close()

	profiles := make([]domain.AnnouncementProfile, 0)
	for rows.Next() {
		profile, err := scanAnnouncementProfile(rows)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate announcement profiles: %w", err)
	}
	return profiles, nil
}

func (r *AnnouncementProfileRepository) Get(ctx context.Context, id string) (domain.AnnouncementProfile, error) {
	row := r.db.QueryRowContext(ctx, `
select id, name, description, default_stream_url, default_category, default_message, default_hashtags,
  default_announcement_image_url, default_announcement_image_data_url,
  default_bluesky_card_thumbnail_url, default_bluesky_card_thumbnail_data_url,
  coalesce(default_group_id, ''), created_at, updated_at
from announcement_profiles
where id = ?`, id)
	profile, err := scanAnnouncementProfile(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.AnnouncementProfile{}, err
		}
		return domain.AnnouncementProfile{}, fmt.Errorf("get announcement profile %q: %w", id, err)
	}
	return profile, nil
}

func (r *AnnouncementProfileRepository) GetByName(ctx context.Context, name string) (domain.AnnouncementProfile, error) {
	row := r.db.QueryRowContext(ctx, `
select id, name, description, default_stream_url, default_category, default_message, default_hashtags,
  default_announcement_image_url, default_announcement_image_data_url,
  default_bluesky_card_thumbnail_url, default_bluesky_card_thumbnail_data_url,
  coalesce(default_group_id, ''), created_at, updated_at
from announcement_profiles
where lower(name) = lower(?)`, name)
	profile, err := scanAnnouncementProfile(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.AnnouncementProfile{}, err
		}
		return domain.AnnouncementProfile{}, fmt.Errorf("get announcement profile by name %q: %w", name, err)
	}
	return profile, nil
}

func (r *AnnouncementProfileRepository) Save(ctx context.Context, profile domain.AnnouncementProfile) error {
	defaultGroupID := sql.NullString{String: profile.DefaultGroupID, Valid: profile.DefaultGroupID != ""}
	_, err := r.db.ExecContext(ctx, `
insert into announcement_profiles(
  id, name, description, default_stream_url, default_category, default_message, default_hashtags,
  default_announcement_image_url, default_announcement_image_data_url,
  default_bluesky_card_thumbnail_url, default_bluesky_card_thumbnail_data_url,
  default_group_id, created_at, updated_at
)
values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
on conflict(id) do update set
  name = excluded.name,
  description = excluded.description,
  default_stream_url = excluded.default_stream_url,
  default_category = excluded.default_category,
  default_message = excluded.default_message,
  default_hashtags = excluded.default_hashtags,
  default_announcement_image_url = excluded.default_announcement_image_url,
  default_announcement_image_data_url = excluded.default_announcement_image_data_url,
  default_bluesky_card_thumbnail_url = excluded.default_bluesky_card_thumbnail_url,
  default_bluesky_card_thumbnail_data_url = excluded.default_bluesky_card_thumbnail_data_url,
  default_group_id = excluded.default_group_id,
  created_at = excluded.created_at,
  updated_at = excluded.updated_at`,
		profile.ID,
		profile.Name,
		profile.Description,
		profile.DefaultStreamURL,
		profile.DefaultCategory,
		profile.DefaultMessage,
		profile.DefaultHashtags,
		profile.DefaultAnnouncementImageURL,
		profile.DefaultAnnouncementImageDataURL,
		profile.DefaultBlueskyCardThumbnailURL,
		profile.DefaultBlueskyCardThumbnailDataURL,
		defaultGroupID,
		formatTime(profile.CreatedAt),
		formatTime(profile.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("save announcement profile %q: %w", profile.ID, err)
	}
	return nil
}

func (r *AnnouncementProfileRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `delete from announcement_profiles where id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete announcement profile %q: %w", id, err)
	}
	return nil
}

func scanAnnouncementProfile(scanner rowScanner) (domain.AnnouncementProfile, error) {
	var profile domain.AnnouncementProfile
	var createdAt string
	var updatedAt string

	err := scanner.Scan(
		&profile.ID,
		&profile.Name,
		&profile.Description,
		&profile.DefaultStreamURL,
		&profile.DefaultCategory,
		&profile.DefaultMessage,
		&profile.DefaultHashtags,
		&profile.DefaultAnnouncementImageURL,
		&profile.DefaultAnnouncementImageDataURL,
		&profile.DefaultBlueskyCardThumbnailURL,
		&profile.DefaultBlueskyCardThumbnailDataURL,
		&profile.DefaultGroupID,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return domain.AnnouncementProfile{}, err
	}

	parsedCreatedAt, err := parseTime(createdAt)
	if err != nil {
		return domain.AnnouncementProfile{}, fmt.Errorf("parse announcement profile created_at: %w", err)
	}
	parsedUpdatedAt, err := parseTime(updatedAt)
	if err != nil {
		return domain.AnnouncementProfile{}, fmt.Errorf("parse announcement profile updated_at: %w", err)
	}

	profile.CreatedAt = parsedCreatedAt
	profile.UpdatedAt = parsedUpdatedAt
	return profile, nil
}
