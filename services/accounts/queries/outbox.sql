-- name: InsertOutboxEvent :one
INSERT INTO outbox_events (aggregate_id, event_type, payload)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListUnpublishedOutboxEvents :many
SELECT * FROM outbox_events
WHERE published_at IS NULL
ORDER BY id
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxEventPublished :exec
UPDATE outbox_events
SET published_at = now()
WHERE id = $1;
