-- +goose Up
CREATE TYPE account_status AS ENUM ('pending', 'active', 'suspended');

CREATE TABLE accounts (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email                 TEXT NOT NULL,
    password_hash         TEXT NOT NULL,
    status                account_status NOT NULL DEFAULT 'pending',
    email_verified_at     TIMESTAMPTZ,
    failed_login_attempts INTEGER NOT NULL DEFAULT 0 CHECK (failed_login_attempts >= 0),
    locked_until          TIMESTAMPTZ,
    last_login_at         TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Emails are compared case-insensitively.
CREATE UNIQUE INDEX accounts_email_lower_key ON accounts (lower(email));

-- +goose Down
DROP TABLE accounts;
DROP TYPE account_status;
