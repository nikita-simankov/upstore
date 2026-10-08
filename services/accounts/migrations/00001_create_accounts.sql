-- +goose Up
CREATE TYPE account_status AS ENUM ('pending', 'active');

CREATE TYPE auth_provider AS ENUM ('google', 'apple', 'telegram');

CREATE TABLE accounts (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- NULL for Telegram accounts and for Apple accounts that hide the email.
    email                 TEXT,
    -- NULL for accounts that only sign in through an auth provider.
    password_hash         TEXT,
    status                account_status NOT NULL DEFAULT 'pending',
    email_verified_at     TIMESTAMPTZ,
    failed_login_attempts INTEGER NOT NULL DEFAULT 0 CHECK (failed_login_attempts >= 0),
    -- Temporary lock after repeated failed logins.
    locked_until          TIMESTAMPTZ,
    -- Administrative ban. NULL means not banned; 'infinity' means permanent.
    banned_until          TIMESTAMPTZ,
    ban_reason            TEXT,
    last_login_at         TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Emails are compared case-insensitively. Accounts without an email are not indexed.
CREATE UNIQUE INDEX accounts_email_lower_key ON accounts (lower(email)) WHERE email IS NOT NULL;

CREATE TABLE auth_identities (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id       UUID NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    provider         auth_provider NOT NULL,
    -- The provider's stable user ID: Google/Apple "sub" or Telegram user ID.
    provider_user_id TEXT NOT NULL,
    -- Email reported by the provider. It may differ from accounts.email.
    provider_email   TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- One external identity belongs to exactly one account.
    CONSTRAINT auth_identities_provider_user_key UNIQUE (provider, provider_user_id),
    -- An account has at most one identity per provider.
    CONSTRAINT auth_identities_account_provider_key UNIQUE (account_id, provider)
);

-- Events are written in the same transaction as the change they describe, then
-- published to RabbitMQ by the relay. published_at stays NULL until the broker confirms.
CREATE TABLE outbox_events (
    id           BIGSERIAL PRIMARY KEY,
    aggregate_id UUID NOT NULL,
    event_type   TEXT NOT NULL,
    payload      JSONB NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX outbox_events_unpublished_idx ON outbox_events (id) WHERE published_at IS NULL;

-- A session is one sign-in. Only the SHA-256 hash of the refresh token is stored, so a
-- database leak does not expose usable tokens. Each refresh replaces the row's token and
-- revokes the old one.
CREATE TABLE sessions (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id         UUID NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    refresh_token_hash BYTEA NOT NULL UNIQUE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at         TIMESTAMPTZ NOT NULL,
    revoked_at         TIMESTAMPTZ
);

CREATE INDEX sessions_account_id_idx ON sessions (account_id);

-- An email verification link. Only the SHA-256 hash of the token is stored. A token works once
-- (used_at) and only until expires_at.
CREATE TABLE email_verifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id  UUID NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    token_hash  BYTEA NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ
);

CREATE INDEX email_verifications_account_id_idx ON email_verifications (account_id);

-- +goose Down
DROP TABLE email_verifications;
DROP TABLE sessions;
DROP TABLE outbox_events;
DROP TABLE auth_identities;
DROP TABLE accounts;
DROP TYPE auth_provider;
DROP TYPE account_status;
