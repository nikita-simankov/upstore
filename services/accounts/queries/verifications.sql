-- name: CreateEmailVerification :one
INSERT INTO email_verifications (account_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ConsumeEmailVerification :one
-- Marks the token used and returns its account. No row means the token is unknown, already
-- used, or expired, and the caller must refuse it without saying which. The current time is
-- passed in, so expiry follows the service's clock.
UPDATE email_verifications
SET used_at = sqlc.arg(now_at)::timestamptz
WHERE token_hash = sqlc.arg(token_hash)
  AND used_at IS NULL
  AND expires_at > sqlc.arg(now_at)::timestamptz
RETURNING account_id;

-- name: InvalidateEmailVerifications :exec
-- Used before a new link is issued, so only the newest link works.
UPDATE email_verifications
SET used_at = now()
WHERE account_id = $1
  AND used_at IS NULL;
