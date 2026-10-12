BEGIN;

CREATE TABLE laundry.users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    username TEXT NOT NULL CHECK (username ~ '^[a-zA-Z0-9_.-]{3,64}$'),
    email TEXT NOT NULL CHECK (position('@' IN email) > 1),
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'cashier')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX users_username_unique ON laundry.users (lower(username));
CREATE UNIQUE INDEX users_email_unique ON laundry.users (lower(email));

CREATE TABLE laundry.auth_sessions (
    token_hash BYTEA PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    user_id BIGINT NOT NULL REFERENCES laundry.users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX auth_sessions_expiry ON laundry.auth_sessions (expires_at);
CREATE INDEX auth_sessions_user ON laundry.auth_sessions (user_id);

COMMIT;
