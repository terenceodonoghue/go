-- name: CreateCredential :one
INSERT INTO credentials (
    id, user_id, public_key, transport,
    sign_count, flag_backup_eligible, flag_backup_state, aaguid
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetCredentialsByUserID :many
SELECT * FROM credentials WHERE user_id = $1;

-- name: UpdateCredentialAfterLogin :exec
UPDATE credentials
SET sign_count = $2, flag_backup_state = $3
WHERE id = $1;
