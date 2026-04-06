ALTER TABLE registrations ADD COLUMN user_id TEXT;
INSERT INTO settings (key, value) VALUES ('allow_self_registration', 'false') ON CONFLICT (key) DO NOTHING;
