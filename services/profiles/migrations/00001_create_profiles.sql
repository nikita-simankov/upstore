-- +goose Up
CREATE TYPE account_type AS ENUM ('seller', 'shopper');

-- One profile per account. account_id matches accounts.accounts.id, but there is no
-- foreign key: the accounts service owns that table and may live in another database.
CREATE TABLE profiles (
    account_id   UUID PRIMARY KEY,
    account_type account_type NOT NULL,
    -- Empty until the user fills in their profile after sign-up.
    full_name    TEXT,
    phone        TEXT,
    locale       TEXT NOT NULL DEFAULT 'ru' CHECK (locale IN ('ru', 'en')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE profiles;
DROP TYPE account_type;
