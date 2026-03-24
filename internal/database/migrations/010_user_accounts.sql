ALTER TABLE registrations ADD COLUMN user_id TEXT;
INSERT OR IGNORE INTO settings (key, value) VALUES ('allow_self_registration', 'false');
