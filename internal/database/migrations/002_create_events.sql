CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    location TEXT NOT NULL DEFAULT '',
    event_date TIMESTAMP NOT NULL,
    registration_deadline TIMESTAMP,
    max_capacity INTEGER,
    attendee_list_public BOOLEAN NOT NULL DEFAULT TRUE,
    registration_open BOOLEAN NOT NULL DEFAULT TRUE,
    created_by TEXT NOT NULL REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
