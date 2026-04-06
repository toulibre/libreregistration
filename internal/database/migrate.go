package database

import (
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Migrate(db *DB) error {
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

	return nil
}
