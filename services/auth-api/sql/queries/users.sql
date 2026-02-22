-- name: CreateUser :one
INSERT INTO users (webauthn_id, email)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserByWebAuthnID :one
SELECT * FROM users WHERE webauthn_id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;
