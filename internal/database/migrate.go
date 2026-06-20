package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// goEventTZVersion guards the one-shot Go data migration that reinterprets
// stored event timestamps as the application timezone. Recorded in
// schema_migrations so it runs exactly once.
const goEventTZVersion = "go:event-dates-to-app-timezone-v1"

func Migrate(db *DB, loc *time.Location) error {
	// Create migrations tracking table
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	// Disable FK checks during migrations (some migrations recreate tables)
	if db.Driver == "sqlite" {
		db.Exec("PRAGMA foreign_keys = OFF")
		defer db.Exec("PRAGMA foreign_keys = ON")
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	// Sort by filename to ensure order
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	// Build set of migration files, preferring driver-specific variants.
	// For example, if both 011_foo.sql and 011_foo.postgres.sql exist
	// and the driver is postgres/pgx, use 011_foo.postgres.sql but record
	// it as "011_foo.sql" in schema_migrations.
	type migration struct {
		version  string // canonical name (e.g. "011_foo.sql")
		filename string // actual file to read
	}

	// Normalize driver name for file matching
	driverFamily := db.Driver
	if driverFamily == "pgx" {
		driverFamily = "postgres"
	}

	migrations := make(map[string]*migration)
	var order []string

	for _, entry := range entries {
		name := entry.Name()

		// Detect driver-specific files: NNN_name.postgres.sql or NNN_name.sqlite.sql
		var canonical, fileDriver string
		if strings.Contains(name, ".postgres.sql") {
			canonical = strings.Replace(name, ".postgres.sql", ".sql", 1)
			fileDriver = "postgres"
		} else if strings.Contains(name, ".sqlite.sql") {
			canonical = strings.Replace(name, ".sqlite.sql", ".sql", 1)
			fileDriver = "sqlite"
		} else {
			canonical = name
			fileDriver = "" // generic
		}

		if _, exists := migrations[canonical]; !exists {
			migrations[canonical] = &migration{version: canonical, filename: name}
			order = append(order, canonical)
		}

		// Driver-specific file takes precedence for matching driver
		if fileDriver == driverFamily {
			migrations[canonical].filename = name
		}
	}

	for _, canonical := range order {
		m := migrations[canonical]

		// Check if already applied
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", m.version).Scan(&count); err != nil {
			return fmt.Errorf("check migration %s: %w", m.version, err)
		}
		if count > 0 {
			continue
		}

		// Read and execute migration
		content, err := migrationsFS.ReadFile("migrations/" + m.filename)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", m.filename, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin transaction for %s: %w", m.version, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", m.version, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", m.version); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", m.version, err)
		}

		log.Printf("Applied migration: %s", m.version)
	}

	if err := reinterpretEventDatesAsLocation(db, loc); err != nil {
		return fmt.Errorf("reinterpret event dates: %w", err)
	}

	return nil
}

// reinterpretEventDatesAsLocation is a one-shot data migration. Historically,
// event timestamps were stored as the Europe/Paris wall-clock the admin typed
// but labelled UTC (the form was parsed in time.Local, which was UTC on the
// deployment). Now that the app parses and displays dates in a configured
// timezone, those stored instants are off by the location's offset.
//
// This re-reads every event_date / registration_deadline, reinterprets its
// wall-clock components in loc, and stores the resulting true UTC instant.
// Displayed times are therefore unchanged, but the underlying instants become
// correct (and the iCal feed emits proper UTC). It runs exactly once, guarded
// by goEventTZVersion in schema_migrations.
func reinterpretEventDatesAsLocation(db *DB, loc *time.Location) error {
	if loc == nil || loc == time.UTC {
		return nil // nothing to reinterpret against UTC
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", goEventTZVersion).Scan(&count); err != nil {
		return fmt.Errorf("check version: %w", err)
	}
	if count > 0 {
		return nil
	}

	// reinterpret keeps the wall-clock digits but re-anchors them to loc, then
	// converts to UTC. e.g. 20:00 (stored) -> 20:00 Europe/Paris -> 18:00Z.
	reinterpret := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc).UTC()
	}

	rows, err := db.Query("SELECT id, event_date, registration_deadline FROM events")
	if err != nil {
		return fmt.Errorf("select events: %w", err)
	}
	type update struct {
		id       string
		date     time.Time
		deadline sql.NullTime
	}
	var updates []update
	for rows.Next() {
		var id string
		var date time.Time
		var deadline sql.NullTime
		if err := rows.Scan(&id, &date, &deadline); err != nil {
			rows.Close()
			return fmt.Errorf("scan event: %w", err)
		}
		u := update{id: id, date: reinterpret(date)}
		if deadline.Valid {
			u.deadline = sql.NullTime{Time: reinterpret(deadline.Time), Valid: true}
		}
		updates = append(updates, u)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate events: %w", err)
	}
	rows.Close()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	for _, u := range updates {
		var deadline interface{}
		if u.deadline.Valid {
			deadline = u.deadline.Time
		}
		if _, err := tx.Exec("UPDATE events SET event_date = ?, registration_deadline = ? WHERE id = ?", u.date, deadline, u.id); err != nil {
			tx.Rollback()
			return fmt.Errorf("update event %s: %w", u.id, err)
		}
	}
	if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", goEventTZVersion); err != nil {
		tx.Rollback()
		return fmt.Errorf("record version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	log.Printf("Reinterpreted %d event date(s) as %s", len(updates), loc)
	return nil
}
