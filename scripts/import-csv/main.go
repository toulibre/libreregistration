// Script de génération SQL à partir d'un CSV d'inscriptions historiques.
// Produit un fichier SQL idempotent (INSERT ... ON CONFLICT DO NOTHING)
// exécutable directement en prod.
//
// Usage:
//
//	go run ./scripts/import-csv/ -csv inscription.csv > import.sql
//	go run ./scripts/import-csv/ -csv inscription.csv -driver postgres > import.sql
//	go run ./scripts/import-csv/ -csv inscription.csv -driver sqlite > import.sql
//
// Exécution en prod :
//
//	# PostgreSQL (docker compose)
//	docker compose exec -T postgres psql -U libreregistration libreregistration < import.sql
//
//	# SQLite (docker compose)
//	docker compose cp import.sql app:/tmp/import.sql
//	docker compose exec app sqlite3 /data/libreregistration.db < /tmp/import.sql
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
)

var validEventID = regexp.MustCompile(`^\d{4}-\d{1,2}-\d{1,2}`)

func main() {
	csvPath := flag.String("csv", "inscription.csv", "Chemin vers le fichier CSV")
	adminID := flag.String("admin-id", "", "UUID de l'admin créateur (sinon: premier admin trouvé via SQL)")
	driver := flag.String("driver", "postgres", "Driver: postgres ou sqlite")
	flag.Parse()

	if err := run(*csvPath, *adminID, *driver); err != nil {
		log.Fatal(err)
	}
}

type record struct {
	nom      string
	prenom   string
	email    string
	dateStr  string
	eventID  string
	interets string
}

func run(csvPath, adminID, driver string) error {
	records, err := parseCSV(csvPath)
	if err != nil {
		return fmt.Errorf("parse CSV: %w", err)
	}
	log.Printf("Parsed %d valid records, generating SQL...", len(records))

	// Group by event
	type eventGroup struct {
		eventID string
		records []record
	}
	seen := map[string]int{}
	var groups []eventGroup
	for _, r := range records {
		if idx, ok := seen[r.eventID]; ok {
			groups[idx].records = append(groups[idx].records, r)
		} else {
			seen[r.eventID] = len(groups)
			groups = append(groups, eventGroup{eventID: r.eventID, records: []record{r}})
		}
	}

	// Generate SQL to stdout
	w := os.Stdout

	fmt.Fprintln(w, "-- Import historique des inscriptions")
	fmt.Fprintf(w, "-- Généré le %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "-- %d événements, %d inscriptions\n", len(groups), len(records))
	fmt.Fprintln(w, "-- Rejouable : les doublons sont ignorés (ON CONFLICT DO NOTHING)")
	fmt.Fprintln(w)

	if adminID == "" {
		fmt.Fprintln(w, "-- Récupère le premier admin comme créateur des événements")
		if driver == "postgres" {
			fmt.Fprintln(w, `DO $$ DECLARE v_admin_id TEXT; BEGIN`)
			fmt.Fprintln(w, `  SELECT id INTO v_admin_id FROM users WHERE role = 'admin' LIMIT 1;`)
			fmt.Fprintln(w, `  IF v_admin_id IS NULL THEN RAISE EXCEPTION 'No admin user found'; END IF;`)
		} else {
			// SQLite doesn't have DO blocks; we use a temp table
			fmt.Fprintln(w, `CREATE TEMP TABLE IF NOT EXISTS _import_admin (id TEXT);`)
			fmt.Fprintln(w, `DELETE FROM _import_admin;`)
			fmt.Fprintln(w, `INSERT INTO _import_admin SELECT id FROM users WHERE role = 'admin' LIMIT 1;`)
		}
	}

	fmt.Fprintln(w)

	uuidCounter := 0
	nextUUID := func() string {
		uuidCounter++
		// Deterministic UUIDs based on counter for reproducibility
		return fmt.Sprintf("00000000-0000-4000-8000-%012d", uuidCounter)
	}

	for _, g := range groups {
		slug := g.eventID
		title := slugToTitle(slug)
		eventDate := slugToDate(slug)
		if eventDate.IsZero() {
			continue
		}

		eventUUID := nextUUID()
		createdBy := sqlQuote(adminID)
		if adminID == "" {
			if driver == "postgres" {
				createdBy = "v_admin_id"
			} else {
				createdBy = "(SELECT id FROM _import_admin)"
			}
		}

		fmt.Fprintf(w, "-- %s (%d inscriptions)\n", slug, len(g.records))

		if driver == "postgres" {
			fmt.Fprintf(w, "INSERT INTO events (id, title, slug, description, location, event_date, attendee_list_public, registration_open, image_path, banner_path, created_by, created_at, updated_at) VALUES (%s, %s, %s, '', '', %s, false, false, '', '', %s, %s, %s) ON CONFLICT (slug) DO NOTHING;\n",
				sqlQuote(eventUUID),
				sqlQuote(title),
				sqlQuote(slug),
				sqlQuote(eventDate.Format("2006-01-02 15:04:05")),
				createdBy,
				sqlQuote(eventDate.Format("2006-01-02 15:04:05")),
				sqlQuote(eventDate.Format("2006-01-02 15:04:05")),
			)
		} else {
			fmt.Fprintf(w, "INSERT OR IGNORE INTO events (id, title, slug, description, location, event_date, attendee_list_public, registration_open, image_path, banner_path, created_by, created_at, updated_at) VALUES (%s, %s, %s, '', '', %s, 0, 0, '', '', %s, %s, %s);\n",
				sqlQuote(eventUUID),
				sqlQuote(title),
				sqlQuote(slug),
				sqlQuote(eventDate.Format("2006-01-02 15:04:05")),
				createdBy,
				sqlQuote(eventDate.Format("2006-01-02 15:04:05")),
				sqlQuote(eventDate.Format("2006-01-02 15:04:05")),
			)
		}

		// Use a subselect for the event_id to handle the case where the event already existed with a different UUID
		eventRef := fmt.Sprintf("(SELECT id FROM events WHERE slug = %s)", sqlQuote(slug))

		for _, r := range g.records {
			regUUID := nextUUID()
			cancelToken := nextUUID()
			name := strings.TrimSpace(r.prenom + " " + r.nom)
			regDate := r.dateStr
			if regDate == "" {
				regDate = eventDate.Format("2006-01-02 15:04:05")
			}

			if driver == "postgres" {
				// Deduplicate on (event_id, name, email) using a WHERE NOT EXISTS
				fmt.Fprintf(w, "INSERT INTO registrations (id, event_id, name, email, comment, cancel_token, registered_at) SELECT %s, %s, %s, %s, %s, %s, %s WHERE NOT EXISTS (SELECT 1 FROM registrations WHERE event_id = %s AND name = %s AND email = %s);\n",
					sqlQuote(regUUID),
					eventRef,
					sqlQuote(name),
					sqlQuote(r.email),
					sqlQuote(r.interets),
					sqlQuote(cancelToken),
					sqlQuote(regDate),
					eventRef,
					sqlQuote(name),
					sqlQuote(r.email),
				)
			} else {
				fmt.Fprintf(w, "INSERT OR IGNORE INTO registrations (id, event_id, name, email, comment, cancel_token, registered_at) SELECT %s, %s, %s, %s, %s, %s, %s WHERE NOT EXISTS (SELECT 1 FROM registrations WHERE event_id = %s AND name = %s AND email = %s);\n",
					sqlQuote(regUUID),
					eventRef,
					sqlQuote(name),
					sqlQuote(r.email),
					sqlQuote(r.interets),
					sqlQuote(cancelToken),
					sqlQuote(regDate),
					eventRef,
					sqlQuote(name),
					sqlQuote(r.email),
				)
			}
		}
		fmt.Fprintln(w)
	}

	if adminID == "" && driver == "postgres" {
		fmt.Fprintln(w, "END $$;")
	}

	if adminID == "" && driver == "sqlite" {
		fmt.Fprintln(w, "DROP TABLE IF EXISTS _import_admin;")
	}

	log.Printf("Generated SQL for %d events and %d registrations", len(groups), len(records))
	return nil
}

func parseCSV(path string) ([]record, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(charmap.ISO8859_1.NewDecoder().Reader(f))
	reader.Comma = ';'
	reader.LazyQuotes = true

	var records []record
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if len(row) < 6 {
			continue
		}

		eventID := strings.TrimSpace(row[5])
		if !validEventID.MatchString(eventID) {
			continue
		}

		records = append(records, record{
			nom:      strings.TrimSpace(row[1]),
			prenom:   strings.TrimSpace(row[2]),
			email:    strings.TrimSpace(row[3]),
			dateStr:  strings.TrimSpace(row[4]),
			eventID:  eventID,
			interets: strings.TrimSpace(safeIndex(row, 6)),
		})
	}

	return records, nil
}

func safeIndex(row []string, i int) string {
	if i < len(row) {
		return row[i]
	}
	return ""
}

func slugToTitle(slug string) string {
	parts := strings.SplitN(slug, "-", 4)
	if len(parts) < 4 {
		return slug
	}

	namePart := parts[3]
	name := strings.ReplaceAll(namePart, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	words := strings.Fields(name)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	name = strings.Join(words, " ")

	d := slugToDate(slug)
	datePart := parts[2] + "/" + parts[1] + "/" + parts[0]
	if !d.IsZero() {
		datePart = d.Format("02/01/2006")
	}

	return name + " \u2014 " + datePart
}

func slugToDate(slug string) time.Time {
	parts := strings.SplitN(slug, "-", 4)
	if len(parts) < 3 {
		return time.Time{}
	}

	dateStr := fmt.Sprintf("%s-%02s-%02s", parts[0], zeroPad(parts[1]), zeroPad(parts[2]))
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}
	}
	return t
}

func zeroPad(s string) string {
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

func sqlQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
