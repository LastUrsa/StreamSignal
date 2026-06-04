package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
select id, name, description, default_stream_title, default_stream_url, default_category, default_message, default_hashtags,
  default_announcement_image_url, default_announcement_image_data_url,
  default_bluesky_card_thumbnail_url, default_bluesky_card_thumbnail_data_url,
  coalesce(default_group_id, ''), last_used_at, use_count, created_at, updated_at
from announcement_profiles
order by last_used_at is null asc, last_used_at desc, use_count desc, lower(name) asc`)
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
select id, name, description, default_stream_title, default_stream_url, default_category, default_message, default_hashtags,
  default_announcement_image_url, default_announcement_image_data_url,
  default_bluesky_card_thumbnail_url, default_bluesky_card_thumbnail_data_url,
  coalesce(default_group_id, ''), last_used_at, use_count, created_at, updated_at
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
select id, name, description, default_stream_title, default_stream_url, default_category, default_message, default_hashtags,
  default_announcement_image_url, default_announcement_image_data_url,
  default_bluesky_card_thumbnail_url, default_bluesky_card_thumbnail_data_url,
  coalesce(default_group_id, ''), last_used_at, use_count, created_at, updated_at
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
	lastUsedAt := sql.NullString{Valid: profile.LastUsedAt != nil}
	if profile.LastUsedAt != nil {
		lastUsedAt.String = formatTime(*profile.LastUsedAt)
	}
	_, err := r.db.ExecContext(ctx, `
insert into announcement_profiles(
  id, name, description, default_stream_title, default_stream_url, default_category, default_message, default_hashtags,
  default_announcement_image_url, default_announcement_image_data_url,
  default_bluesky_card_thumbnail_url, default_bluesky_card_thumbnail_data_url,
  default_group_id, last_used_at, use_count, created_at, updated_at
)
values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
on conflict(id) do update set
  name = excluded.name,
  description = excluded.description,
  default_stream_title = excluded.default_stream_title,
  default_stream_url = excluded.default_stream_url,
  default_category = excluded.default_category,
  default_message = excluded.default_message,
  default_hashtags = excluded.default_hashtags,
  default_announcement_image_url = excluded.default_announcement_image_url,
  default_announcement_image_data_url = excluded.default_announcement_image_data_url,
  default_bluesky_card_thumbnail_url = excluded.default_bluesky_card_thumbnail_url,
  default_bluesky_card_thumbnail_data_url = excluded.default_bluesky_card_thumbnail_data_url,
  default_group_id = excluded.default_group_id,
  last_used_at = excluded.last_used_at,
  use_count = excluded.use_count,
  created_at = excluded.created_at,
  updated_at = excluded.updated_at`,
		profile.ID,
		profile.Name,
		profile.Description,
		profile.DefaultStreamTitle,
		profile.DefaultStreamURL,
		profile.DefaultCategory,
		profile.DefaultMessage,
		profile.DefaultHashtags,
		profile.DefaultAnnouncementImageURL,
		profile.DefaultAnnouncementImageDataURL,
		profile.DefaultBlueskyCardThumbnailURL,
		profile.DefaultBlueskyCardThumbnailDataURL,
		defaultGroupID,
		lastUsedAt,
		profile.UseCount,
		formatTime(profile.CreatedAt),
		formatTime(profile.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("save announcement profile %q: %w", profile.ID, err)
	}
	return nil
}

func (r *AnnouncementProfileRepository) MarkUsed(ctx context.Context, id string, usedAt time.Time) error {
	result, err := r.db.ExecContext(ctx, `
update announcement_profiles
set last_used_at = ?, use_count = use_count + 1, updated_at = ?
where id = ?`, formatTime(usedAt), formatTime(usedAt), id)
	if err != nil {
		return fmt.Errorf("mark announcement profile %q used: %w", id, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark announcement profile %q used rows affected: %w", id, err)
	}
	if affected == 0 {
		return sql.ErrNoRows
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
	var lastUsedAt sql.NullString
	var createdAt string
	var updatedAt string

	err := scanner.Scan(
		&profile.ID,
		&profile.Name,
		&profile.Description,
		&profile.DefaultStreamTitle,
		&profile.DefaultStreamURL,
		&profile.DefaultCategory,
		&profile.DefaultMessage,
		&profile.DefaultHashtags,
		&profile.DefaultAnnouncementImageURL,
		&profile.DefaultAnnouncementImageDataURL,
		&profile.DefaultBlueskyCardThumbnailURL,
		&profile.DefaultBlueskyCardThumbnailDataURL,
		&profile.DefaultGroupID,
		&lastUsedAt,
		&profile.UseCount,
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
	if lastUsedAt.Valid && lastUsedAt.String != "" {
		parsedLastUsedAt, err := parseTime(lastUsedAt.String)
		if err != nil {
			return domain.AnnouncementProfile{}, fmt.Errorf("parse announcement profile last_used_at: %w", err)
		}
		profile.LastUsedAt = &parsedLastUsedAt
	}

	profile.CreatedAt = parsedCreatedAt
	profile.UpdatedAt = parsedUpdatedAt
	return profile, nil
}
