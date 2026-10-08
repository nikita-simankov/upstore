-- name: CreateAccount :one
INSERT INTO accounts (email, password_hash)
VALUES ($1, $2)
RETURNING *;

-- name: GetAccountByID :one
SELECT * FROM accounts
WHERE id = $1;

-- name: GetAccountByEmail :one
SELECT * FROM accounts
WHERE lower(email) = lower($1);

-- name: MarkEmailVerified :exec
UPDATE accounts
SET email_verified_at = now(),
    status = 'active',
    updated_at = now()
WHERE id = $1;

-- name: SetAccountStatus :exec
UPDATE accounts
SET status = $2,
    updated_at = now()
WHERE id = $1;

-- name: RecordFailedLogin :exec
UPDATE accounts
SET failed_login_attempts = failed_login_attempts + 1,
    locked_until = $2,
    updated_at = now()
WHERE id = $1;

-- name: RecordSuccessfulLogin :exec
UPDATE accounts
SET failed_login_attempts = 0,
    locked_until = NULL,
    last_login_at = now(),
    updated_at = now()
WHERE id = $1;

-- name: CreateAuthIdentity :one
INSERT INTO auth_identities (account_id, provider, provider_user_id, provider_email)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetAccountByAuthIdentity :one
SELECT a.* FROM accounts a
JOIN auth_identities i ON i.account_id = a.id
WHERE i.provider = $1
  AND i.provider_user_id = $2;

-- name: SetAccountBan :exec
UPDATE accounts
SET banned_until = $2,
    ban_reason = $3,
    updated_at = now()
WHERE id = $1;
