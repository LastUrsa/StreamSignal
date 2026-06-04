package sqlite

import (
	"database/sql"
	"fmt"
)

type migration struct {
	version int
	name    string
	sql     string
}

var migrations = []migration{
	{
		version: 1,
		name:    "create_destinations",
		sql: `
create table if not exists destinations (
  id text primary key,
  platform text not null,
  name text not null,
  enabled integer not null,
  template text not null,
  config_json text not null,
  created_at text not null,
  updated_at text not null
);`,
	},
	{
		version: 2,
		name:    "create_settings",
		sql: `
create table if not exists settings (
  id integer primary key check (id = 1),
  test_mode_enabled integer not null,
  default_stream_url text not null,
  default_hashtags text not null,
  duplicate_protection_enabled integer not null,
  duplicate_window_minutes integer not null,
  end_stream_post_enabled integer not null,
  end_stream_template text not null
);`,
	},
	{
		version: 3,
		name:    "create_logs",
		sql: `
create table if not exists logs (
  id integer primary key autoincrement,
  timestamp text not null,
  destination text not null,
  action text not null,
  status text not null,
  message text not null
);`,
	},
	{
		version: 4,
		name:    "create_post_history",
		sql: `
create table if not exists post_history (
  id integer primary key autoincrement,
  destination_id text not null,
  rendered_content_hash text not null,
  rendered_content text not null,
  posted_at text not null
);`,
	},
	{
		version: 5,
		name:    "add_test_mode_destination_fields",
		sql: `
alter table settings add column test_discord_webhook_key text not null default '';
alter table settings add column test_bluesky_account_identifier text not null default '';
alter table settings add column test_bluesky_credential_key text not null default '';
alter table settings add column test_mastodon_credential_key text not null default '';
alter table settings add column test_mastodon_instance_url text not null default '';`,
	},
	{
		version: 6,
		name:    "create_bluesky_live_now_sessions",
		sql: `
create table if not exists bluesky_live_now_sessions (
  destination_id text primary key,
  destination_name text not null,
  platform text not null,
  account_identifier text not null,
  credential_key text not null,
  stream_url text not null,
  stream_title text not null,
  started_at text not null
);`,
	},
	{
		version: 7,
		name:    "create_destination_groups",
		sql: `
create table if not exists destination_groups (
  id text primary key,
  name text not null,
  description text not null,
  created_at text not null,
  updated_at text not null
);
create unique index if not exists destination_groups_name_unique on destination_groups(lower(name));
create table if not exists destination_group_memberships (
  group_id text not null,
  destination_id text not null,
  primary key (group_id, destination_id),
  foreign key (group_id) references destination_groups(id) on delete cascade,
  foreign key (destination_id) references destinations(id) on delete cascade
);
insert or ignore into destination_groups(id, name, description, created_at, updated_at)
values('all-destinations', 'All Destinations', '', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
insert or ignore into destination_group_memberships(group_id, destination_id)
select 'all-destinations', id from destinations;`,
	},
	{
		version: 8,
		name:    "add_selected_destination_group_to_settings",
		sql: `
alter table settings add column selected_destination_group_id text not null default 'all-destinations';`,
	},
	{
		version: 9,
		name:    "create_announcement_profiles",
		sql: `
create table if not exists announcement_profiles (
  id text primary key,
  name text not null,
  description text not null,
  default_stream_url text not null,
  default_category text not null,
  default_message text not null,
  default_hashtags text not null,
  default_announcement_image_url text not null,
  default_announcement_image_data_url text not null,
  default_bluesky_card_thumbnail_url text not null,
  default_bluesky_card_thumbnail_data_url text not null,
  default_group_id text,
  created_at text not null,
  updated_at text not null,
  foreign key (default_group_id) references destination_groups(id) on delete set null
);
create unique index if not exists announcement_profiles_name_unique on announcement_profiles(lower(name));`,
	},
}

func ApplyMigrations(db *sql.DB) error {
	if _, err := db.Exec(`
create table if not exists schema_migrations (
  version integer primary key,
  name text not null,
  applied_at text not null default current_timestamp
);`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	for _, item := range migrations {
		var applied bool
		err := db.QueryRow(`select exists(select 1 from schema_migrations where version = ?)`, item.version).Scan(&applied)
		if err != nil {
			return fmt.Errorf("check migration %d: %w", item.version, err)
		}
		if applied {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", item.version, err)
		}

		if _, err := tx.Exec(item.sql); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %d (%s): %w", item.version, item.name, err)
		}

		if _, err := tx.Exec(`insert into schema_migrations(version, name) values(?, ?)`, item.version, item.name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d (%s): %w", item.version, item.name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d (%s): %w", item.version, item.name, err)
		}
	}

	return nil
}
