-- name: ListAPITokens :many
SELECT * FROM api_tokens ORDER BY created_at;

-- name: CreateAPIToken :one
INSERT INTO api_tokens (name, token) VALUES ($1, $2) RETURNING *;

-- name: DeleteAPIToken :exec
DELETE FROM api_tokens WHERE id = $1;

-- name: IntrospectAPIToken :one
UPDATE api_tokens SET last_used_at = NOW() WHERE token = $1 RETURNING id;
