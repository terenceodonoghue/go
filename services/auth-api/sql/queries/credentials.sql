-- name: CreateCredential :one
INSERT INTO credentials (
    id, user_handle, display_name, public_key, transport,
    sign_count, flag_backup_eligible, flag_backup_state, aaguid,
    last_used_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
RETURNING *;

-- name: GetCredential :one
SELECT * FROM credentials WHERE user_handle = $1;

-- name: UpdateCredential :exec
UPDATE credentials
SET sign_count = $2, flag_backup_state = $3, last_used_at = NOW()
WHERE id = $1;
