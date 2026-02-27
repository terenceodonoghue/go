# auth-api

Passwordless authentication server using [WebAuthn/passkeys](https://webauthn.io/). Users register and sign in using discoverable credentials — no passwords, no email verification.

## Prerequisites

- Go 1.26+
- Docker and Docker Compose

## Quick start

Start PostgreSQL and Redis:

```sh
docker compose up -d
```

Create a `.env` file:

```
DATABASE_URL=postgres://auth:auth@localhost:5432/auth?sslmode=disable
REDIS_ADDR=localhost:6379
RP_ID=localhost
RP_ORIGINS=http://localhost:3000
```

Run the server:

```sh
set -a && source .env && set +a && go run .
```

## Configuration

All environment variables are required unless noted otherwise.

| Variable | Description | Example |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@host:5432/db` |
| `REDIS_ADDR` | Redis address | `host:6379` |
| `RP_ID` | WebAuthn relying party ID (your domain) | `example.com` |
| `RP_ORIGINS` | Comma-separated origins the browser sends during WebAuthn ceremonies | `https://example.com,https://auth.example.com` |
| `ADDR` | Listen address (optional, defaults to `:8081`) | `:8080` |

## API

### Registration (2 steps)

| Method | Path | Description |
|---|---|---|
| POST | `/api/register/begin` | Send `{"name": "..."}` to receive WebAuthn creation options |
| POST | `/api/register/finish` | Complete the WebAuthn ceremony with the authenticator response |

### Login (2 steps)

| Method | Path | Description |
|---|---|---|
| POST | `/api/login/begin` | Request WebAuthn assertion options for discoverable login |
| POST | `/api/login/finish` | Complete the WebAuthn ceremony with the authenticator response |

### Session

Successful registration and login set an `auth_session` cookie with a 15-minute sliding TTL. The session is stored in Redis and refreshed on each access.

### Logout

| Method | Path | Description |
|---|---|---|
| POST | `/api/logout` | Delete the session and clear the cookie |

Returns `204 No Content`.

### API tokens

All token endpoints require a valid passkey session.

| Method | Path | Description |
|---|---|---|
| GET | `/api/tokens` | List all API tokens |
| POST | `/api/tokens` | Create a token — send `{"name": "..."}` |
| DELETE | `/api/tokens/{id}` | Delete a token by UUID |

### Introspect

| Method | Path | Description |
|---|---|---|
| POST | `/api/introspect` | Validate a session cookie or Bearer token |

Returns `200` if valid, `401` otherwise. Designed for use with Caddy's `forward_auth` directive.

## Docker

Build the image:

```sh
docker build -t auth .
```

Run with your own infrastructure:

```sh
docker run -p 8081:8081 \
  -e DATABASE_URL=postgres://user:pass@host:5432/db \
  -e REDIS_ADDR=host:6379 \
  -e RP_ID=example.com \
  -e RP_ORIGINS=https://example.com \
  auth
```

## Notes

- WebAuthn requires HTTPS in production. `localhost` is the only exception for development.
- Passkeys are scoped to `RP_ID`. Changing it after users have registered will invalidate their credentials.
- The schema is embedded in the binary and applied automatically on startup.
