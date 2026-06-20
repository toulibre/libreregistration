package database

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

// Simulates the production storage convention: event times were the Paris
// wall-clock the admin typed, but stored labelled UTC (time.Local was UTC on
// the deployment). reinterpretEventDatesAsLocation must turn them into correct
// UTC instants for Europe/Paris, exactly once.
func TestReinterpretEventDatesAsLocation(t *testing.T) {
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Fatalf("load Europe/Paris: %v", err)
	}

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// Build schema. UTC location => reinterpret is a no-op and does NOT record
	// its version, so we can run it explicitly below.
	if err := Migrate(db, time.UTC); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		t.Fatalf("pragma: %v", err)
	}

	// A timed (form-entered) event at 20:00 and a date-only (imported) event at
	// midnight, both stored as naive-UTC wall-clock. The timed one also has a
	// registration deadline.
	deadline := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	insert := func(id, slug string, date time.Time, dl *time.Time) {
		var d interface{}
		if dl != nil {
			d = *dl
		}
		_, err := db.Exec(
			`INSERT INTO events (id, title, slug, event_date, registration_deadline, created_by) VALUES (?, ?, ?, ?, ?, 'u1')`,
			id, slug, slug, date, d,
		)
		if err != nil {
			t.Fatalf("insert %s: %v", id, err)
		}
	}
	insert("e1", "qjelt-juin", time.Date(2026, 6, 4, 20, 0, 0, 0, time.UTC), &deadline)
	insert("e2", "import-avril", time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC), nil)

	// Imported events were stored as naive datetime *strings*
	// ("2006-01-02 15:04:05"), not time.Time values. Cover that path too: it
	// must read back with the literal wall-clock so it is reinterpreted the
	// same way.
	if _, err := db.Exec(
		`INSERT INTO events (id, title, slug, event_date, created_by) VALUES ('e3', 'import-str', 'import-str', '2025-09-11 20:30:00', 'u1')`,
	); err != nil {
		t.Fatalf("insert e3: %v", err)
	}

	read := func(id string) (time.Time, sql.NullTime) {
		var d time.Time
		var dl sql.NullTime
		if err := db.QueryRow("SELECT event_date, registration_deadline FROM events WHERE id = ?", id).Scan(&d, &dl); err != nil {
			t.Fatalf("read %s: %v", id, err)
		}
		return d, dl
	}

	// Run the migration.
	if err := reinterpretEventDatesAsLocation(db, paris); err != nil {
		t.Fatalf("reinterpret: %v", err)
	}

	// 20:00 Paris (CEST, +02:00) == 18:00 UTC.
	d1, dl1 := read("e1")
	if got := d1.UTC(); !got.Equal(time.Date(2026, 6, 4, 18, 0, 0, 0, time.UTC)) {
		t.Errorf("e1 event_date = %s, want 2026-06-04 18:00:00Z", got.Format(time.RFC3339))
	}
	// 12:00 Paris (CEST) == 10:00 UTC.
	if !dl1.Valid || !dl1.Time.UTC().Equal(time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("e1 deadline = %v, want 2026-06-01 10:00:00Z", dl1)
	}
	// Midnight Paris (CEST) == 22:00 UTC the previous day. Crucially, displayed
	// back in Paris it is still 00:00 on 2026-04-02.
	d2, _ := read("e2")
	if got := d2.UTC(); !got.Equal(time.Date(2026, 4, 1, 22, 0, 0, 0, time.UTC)) {
		t.Errorf("e2 event_date = %s, want 2026-04-01 22:00:00Z", got.Format(time.RFC3339))
	}
	if got := d2.In(paris); got.Hour() != 0 || got.Day() != 2 {
		t.Errorf("e2 displayed in Paris = %s, want 2026-04-02 00h", got.Format(time.RFC3339))
	}

	// Naive-string row: 20:30 Paris (CEST) == 18:30 UTC.
	d3, _ := read("e3")
	if got := d3.UTC(); !got.Equal(time.Date(2025, 9, 11, 18, 30, 0, 0, time.UTC)) {
		t.Errorf("e3 event_date = %s, want 2025-09-11 18:30:00Z", got.Format(time.RFC3339))
	}

	// Idempotency: running again must not shift anything (version guard).
	if err := reinterpretEventDatesAsLocation(db, paris); err != nil {
		t.Fatalf("reinterpret (2nd): %v", err)
	}
	d1b, _ := read("e1")
	if !d1b.UTC().Equal(time.Date(2026, 6, 4, 18, 0, 0, 0, time.UTC)) {
		t.Errorf("e1 shifted on second run: %s", d1b.UTC().Format(time.RFC3339))
	}
}
