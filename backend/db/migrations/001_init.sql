CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE merchants (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT        UNIQUE NOT NULL,
    password_hash TEXT        NOT NULL,
    name          TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE stores (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id   UUID        NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    name          TEXT        NOT NULL,
    slug          TEXT        UNIQUE NOT NULL,
    logo_url      TEXT,
    plan          TEXT        NOT NULL DEFAULT 'start'
                              CHECK (plan IN ('start', 'growth', 'pro')),
    trial_ends_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 days',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE products (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id    UUID        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    description TEXT,
    price       INTEGER     NOT NULL CHECK (price >= 0),
    stock       INTEGER     NOT NULL DEFAULT 0 CHECK (stock >= 0),
    image_url   TEXT,
    published   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE orders (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id         UUID        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    customer_email   TEXT        NOT NULL,
    customer_name    TEXT        NOT NULL,
    customer_phone   TEXT,
    shipping_address TEXT        NOT NULL,
    total            INTEGER     NOT NULL CHECK (total >= 0),
    status           TEXT        NOT NULL DEFAULT 'pending'
                                 CHECK (status IN ('pending','paid','shipped','delivered','cancelled')),
    payment_provider TEXT,
    payment_id       TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE order_items (
    id         UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id   UUID    NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID    NOT NULL REFERENCES products(id),
    name       TEXT    NOT NULL,
    price      INTEGER NOT NULL,
    quantity   INTEGER NOT NULL CHECK (quantity > 0)
);
