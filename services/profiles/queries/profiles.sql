-- name: CreateProfile :exec
-- Idempotent: a redelivered account.created event leaves the existing profile unchanged.
INSERT INTO profiles (account_id, account_type)
VALUES ($1, $2)
ON CONFLICT (account_id) DO NOTHING;

-- name: GetProfileByAccountID :one
SELECT * FROM profiles
WHERE account_id = $1;
