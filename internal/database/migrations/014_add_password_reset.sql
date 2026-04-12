ALTER TABLE users ADD COLUMN password_reset_token TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN password_reset_expires TIMESTAMP;
