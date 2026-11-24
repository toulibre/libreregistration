package database

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// DB wraps *sql.DB and automatically converts ? placeholders to $1, $2, ...
// when the driver is PostgreSQL.
type DB struct {
	*sql.DB
	Driver string // "sqlite" or "pgx"
}

// Tx wraps *sql.Tx with the same placeholder rebinding.
type Tx struct {
	*sql.Tx
	driver string
}

// rebind converts ? placeholders to $1, $2, ... for PostgreSQL.
func rebind(driver, query string) string {
	if driver == "sqlite" {
		return query
	}
	var b strings.Builder
	n := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			b.WriteString("$")
			b.WriteString(strconv.Itoa(n))
			n++
		} else {
			b.WriteByte(query[i])
		}
	}
	return b.String()
}

// normalizeArgs converts time arguments to UTC for PostgreSQL.
//
// Timestamp columns are TIMESTAMP WITHOUT TIME ZONE. pgx encodes a time.Time
// into such a column by keeping its wall clock and discarding its location,
// and decodes it back as UTC. A time in the application timezone (e.g. 10:00
// Europe/Paris) would therefore be stored as 10:00 and read back as 10:00Z,
// i.e. shifted by the zone offset. Converting to UTC first makes the stored
// wall clock the UTC instant, which round-trips correctly.
func normalizeArgs(driver string, args []any) []any {
	if driver == "sqlite" {
		return args
	}
	out := make([]any, len(args))
	for i, a := range args {
		switch v := a.(type) {
		case time.Time:
			out[i] = v.UTC()
		case *time.Time:
			if v != nil {
				out[i] = v.UTC()
			} else {
				out[i] = v
			}
		case sql.NullTime:
			if v.Valid {
				v.Time = v.Time.UTC()
			}
			out[i] = v
		default:
			out[i] = a
		}
	}
	return out
}

func (db *DB) Exec(query string, args ...any) (sql.Result, error) {
	return db.DB.Exec(rebind(db.Driver, query), normalizeArgs(db.Driver, args)...)
}

func (db *DB) Query(query string, args ...any) (*sql.Rows, error) {
	return db.DB.Query(rebind(db.Driver, query), normalizeArgs(db.Driver, args)...)
}

func (db *DB) QueryRow(query string, args ...any) *sql.Row {
	return db.DB.QueryRow(rebind(db.Driver, query), normalizeArgs(db.Driver, args)...)
}

func (db *DB) Begin() (*Tx, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx, driver: db.Driver}, nil
}

func (tx *Tx) Exec(query string, args ...any) (sql.Result, error) {
	return tx.Tx.Exec(rebind(tx.driver, query), normalizeArgs(tx.driver, args)...)
}

func Open(driver, dsn string) (*DB, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if driver == "sqlite" {
		// Enable WAL mode for better concurrent read performance
		if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
			db.Close()
			return nil, fmt.Errorf("enable WAL mode: %w", err)
		}

		// Enable foreign keys
		if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
			db.Close()
			return nil, fmt.Errorf("enable foreign keys: %w", err)
		}
	}

	return &DB{DB: db, Driver: driver}, nil
}
