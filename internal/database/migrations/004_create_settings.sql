CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
);

INSERT INTO settings (key, value) VALUES ('site_name', 'LibreRegistration') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('accent_color', '#6d28d9') ON CONFLICT (key) DO NOTHING;
