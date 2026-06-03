package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"StreamSignal/internal/domain"
)

type DestinationGroupRepository struct {
	db *sql.DB
}

func NewDestinationGroupRepository(db *sql.DB) *DestinationGroupRepository {
	return &DestinationGroupRepository{db: db}
}

func (r *DestinationGroupRepository) List(ctx context.Context) ([]domain.DestinationGroup, error) {
	rows, err := r.db.QueryContext(ctx, `
select id, name, description, created_at, updated_at
from destination_groups
order by case when id = ? then 0 else 1 end, lower(name) asc`, domain.DefaultDestinationGroupID)
	if err != nil {
		return nil, fmt.Errorf("list destination groups: %w", err)
	}
	defer rows.Close()

	groups := make([]domain.DestinationGroup, 0)
	for rows.Next() {
		group, err := scanDestinationGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate destination groups: %w", err)
	}

	return groups, nil
}

func (r *DestinationGroupRepository) Get(ctx context.Context, id string) (domain.DestinationGroup, error) {
	row := r.db.QueryRowContext(ctx, `
select id, name, description, created_at, updated_at
from destination_groups
where id = ?`, id)

	group, err := scanDestinationGroup(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.DestinationGroup{}, err
		}
		return domain.DestinationGroup{}, fmt.Errorf("get destination group %q: %w", id, err)
	}

	return group, nil
}

func (r *DestinationGroupRepository) GetByName(ctx context.Context, name string) (domain.DestinationGroup, error) {
	row := r.db.QueryRowContext(ctx, `
select id, name, description, created_at, updated_at
from destination_groups
where lower(name) = lower(?)`, name)

	group, err := scanDestinationGroup(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.DestinationGroup{}, err
		}
		return domain.DestinationGroup{}, fmt.Errorf("get destination group by name %q: %w", name, err)
	}

	return group, nil
}

func (r *DestinationGroupRepository) Save(ctx context.Context, group domain.DestinationGroup) error {
	_, err := r.db.ExecContext(ctx, `
insert into destination_groups(id, name, description, created_at, updated_at)
values(?, ?, ?, ?, ?)
on conflict(id) do update set
  name = excluded.name,
  description = excluded.description,
  created_at = excluded.created_at,
  updated_at = excluded.updated_at`,
		group.ID,
		group.Name,
		group.Description,
		formatTime(group.CreatedAt),
		formatTime(group.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("save destination group %q: %w", group.ID, err)
	}
	return nil
}

func (r *DestinationGroupRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `delete from destination_groups where id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete destination group %q: %w", id, err)
	}
	return nil
}

func (r *DestinationGroupRepository) ListDestinations(ctx context.Context, groupID string) ([]domain.Destination, error) {
	rows, err := r.db.QueryContext(ctx, `
select d.id, d.platform, d.name, d.enabled, d.template, d.config_json, d.created_at, d.updated_at
from destinations d
inner join destination_group_memberships m on m.destination_id = d.id
where m.group_id = ?
order by d.name asc`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list destinations for group %q: %w", groupID, err)
	}
	defer rows.Close()

	destinations := make([]domain.Destination, 0)
	for rows.Next() {
		destination, err := scanDestination(rows)
		if err != nil {
			return nil, err
		}
		destinations = append(destinations, destination)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate destinations for group %q: %w", groupID, err)
	}

	return destinations, nil
}

func (r *DestinationGroupRepository) ListGroupIDsForDestination(ctx context.Context, destinationID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
select group_id
from destination_group_memberships
where destination_id = ?
order by group_id asc`, destinationID)
	if err != nil {
		return nil, fmt.Errorf("list groups for destination %q: %w", destinationID, err)
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate groups for destination %q: %w", destinationID, err)
	}

	return ids, nil
}

func (r *DestinationGroupRepository) SetDestinations(ctx context.Context, groupID string, destinationIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin set destinations for group %q: %w", groupID, err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `delete from destination_group_memberships where group_id = ?`, groupID); err != nil {
		return fmt.Errorf("clear destinations for group %q: %w", groupID, err)
	}
	for _, destinationID := range uniqueStrings(destinationIDs) {
		if _, err := tx.ExecContext(ctx, `
insert or ignore into destination_group_memberships(group_id, destination_id)
values(?, ?)`, groupID, destinationID); err != nil {
			return fmt.Errorf("add destination %q to group %q: %w", destinationID, groupID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit destinations for group %q: %w", groupID, err)
	}
	return nil
}

func (r *DestinationGroupRepository) SetGroupsForDestination(ctx context.Context, destinationID string, groupIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin set groups for destination %q: %w", destinationID, err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `delete from destination_group_memberships where destination_id = ?`, destinationID); err != nil {
		return fmt.Errorf("clear groups for destination %q: %w", destinationID, err)
	}
	for _, groupID := range uniqueStrings(groupIDs) {
		if _, err := tx.ExecContext(ctx, `
insert or ignore into destination_group_memberships(group_id, destination_id)
values(?, ?)`, groupID, destinationID); err != nil {
			return fmt.Errorf("add destination %q to group %q: %w", destinationID, groupID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit groups for destination %q: %w", destinationID, err)
	}
	return nil
}

func (r *DestinationGroupRepository) AddDestination(ctx context.Context, groupID string, destinationID string) error {
	_, err := r.db.ExecContext(ctx, `
insert or ignore into destination_group_memberships(group_id, destination_id)
values(?, ?)`, groupID, destinationID)
	if err != nil {
		return fmt.Errorf("add destination %q to group %q: %w", destinationID, groupID, err)
	}
	return nil
}

func (r *DestinationGroupRepository) RemoveDestination(ctx context.Context, groupID string, destinationID string) error {
	_, err := r.db.ExecContext(ctx, `
delete from destination_group_memberships
where group_id = ? and destination_id = ?`, groupID, destinationID)
	if err != nil {
		return fmt.Errorf("remove destination %q from group %q: %w", destinationID, groupID, err)
	}
	return nil
}

func scanDestinationGroup(scanner rowScanner) (domain.DestinationGroup, error) {
	var group domain.DestinationGroup
	var createdAt string
	var updatedAt string

	err := scanner.Scan(
		&group.ID,
		&group.Name,
		&group.Description,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return domain.DestinationGroup{}, err
	}

	parsedCreatedAt, err := parseTime(createdAt)
	if err != nil {
		return domain.DestinationGroup{}, fmt.Errorf("parse destination group created_at: %w", err)
	}
	parsedUpdatedAt, err := parseTime(updatedAt)
	if err != nil {
		return domain.DestinationGroup{}, fmt.Errorf("parse destination group updated_at: %w", err)
	}

	group.CreatedAt = parsedCreatedAt
	group.UpdatedAt = parsedUpdatedAt
	return group, nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}
