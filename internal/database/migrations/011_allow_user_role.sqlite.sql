-- SQLite doesn't support ALTER CHECK constraints, so we need to recreate the table
CREATE TABLE users_new (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'manager' CHECK (role IN ('admin', 'manager', 'user')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    name TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL DEFAULT '',
    avatar_path TEXT NOT NULL DEFAULT ''
);

INSERT INTO users_new (id, username, password_hash, role, created_at, updated_at, name, email, avatar_path)
    SELECT id, username, password_hash, role, created_at, updated_at, name, email, avatar_path FROM users;

DROP TABLE users;
ALTER TABLE users_new RENAME TO users;
