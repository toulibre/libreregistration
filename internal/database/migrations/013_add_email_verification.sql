ALTER TABLE users ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN email_verify_token TEXT NOT NULL DEFAULT '';

-- Existing users (admin-created) are considered verified
UPDATE users SET email_verified = true;
