package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"StreamSignal/internal/domain"
)

type AnnouncementFieldRepository struct {
	db *sql.DB
}

func NewAnnouncementFieldRepository(db *sql.DB) *AnnouncementFieldRepository {
	return &AnnouncementFieldRepository{db: db}
}

func (r *AnnouncementFieldRepository) List(ctx context.Context) ([]domain.AnnouncementField, error) {
	rows, err := r.db.QueryContext(ctx, `
select id, name, value, created_at, updated_at
from announcement_fields
order by lower(name), id`)
	if err != nil {
		return nil, fmt.Errorf("list announcement fields: %w", err)
	}
	defer rows.Close()

	fields := make([]domain.AnnouncementField, 0)
	for rows.Next() {
		field, err := scanAnnouncementField(rows)
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate announcement fields: %w", err)
	}
	return fields, nil
}

func (r *AnnouncementFieldRepository) UpdateValues(ctx context.Context, values map[string]string, updatedAt time.Time) error {
	if len(values) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin announcement field update: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	ids := make([]string, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		result, err := tx.ExecContext(ctx, `
update announcement_fields
set value = ?, updated_at = ?
where id = ?`, values[id], formatTime(updatedAt), id)
		if err != nil {
			return fmt.Errorf("update announcement field %q: %w", id, err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("update announcement field %q rows affected: %w", id, err)
		}
		if affected == 0 {
			return sql.ErrNoRows
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit announcement field update: %w", err)
	}
	return nil
}

func scanAnnouncementField(scanner rowScanner) (domain.AnnouncementField, error) {
	var field domain.AnnouncementField
	var createdAt string
	var updatedAt string
	if err := scanner.Scan(&field.ID, &field.Name, &field.Value, &createdAt, &updatedAt); err != nil {
		return domain.AnnouncementField{}, err
	}
	parsedCreatedAt, err := parseTime(createdAt)
	if err != nil {
		return domain.AnnouncementField{}, fmt.Errorf("parse announcement field created_at: %w", err)
	}
	parsedUpdatedAt, err := parseTime(updatedAt)
	if err != nil {
		return domain.AnnouncementField{}, fmt.Errorf("parse announcement field updated_at: %w", err)
	}
	field.CreatedAt = parsedCreatedAt
	field.UpdatedAt = parsedUpdatedAt
	return field, nil
}
