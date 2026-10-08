-- name: CreateSession :one
INSERT INTO sessions (account_id, refresh_token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetSessionByRefreshHash :one
SELECT * FROM sessions
WHERE refresh_token_hash = $1;

-- name: RevokeSession :execrows
-- Returns the number of rows changed. Zero means the session was already revoked, which
-- the caller must treat as a reused token.
UPDATE sessions
SET revoked_at = now()
WHERE id = $1
  AND revoked_at IS NULL;

-- name: RevokeAccountSessions :exec
UPDATE sessions
SET revoked_at = now()
WHERE account_id = $1
  AND revoked_at IS NULL;
