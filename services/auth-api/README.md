# auth-api

Passwordless authentication server using [WebAuthn/passkeys](https://webauthn.io/). Supports email verification, passkey registration, and discoverable login.

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
RP_ORIGIN=http://localhost:3000
LOG_VERIFICATION_CODES=true
LOGIN_URL=http://localhost:3000
```

Run the server:

```sh
set -a && source .env && set +a && go run .
```

A test frontend is available at `web/index.html`. Serve it on the port matching `RP_ORIGIN`:

```sh
python3 -m http.server 3000 -d web
```

Set `LOG_VERIFICATION_CODES=true` in your `.env` to log verification codes to the server console for testing. Replace with a real email provider in production.

## Configuration

All environment variables are required unless noted otherwise.

| Variable | Description | Example |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@host:5432/db` |
| `REDIS_ADDR` | Redis address | `host:6379` |
| `RP_ID` | WebAuthn relying party ID (your domain) | `example.com` |
| `RP_ORIGIN` | Origin the browser sends during WebAuthn ceremonies | `https://example.com` |
| `ADDR` | Listen address (optional, defaults to `:8081`) | `:8080` |
| `LOG_VERIFICATION_CODES` | Log codes to console (optional, defaults to `false`) | `true` |
| `LOGIN_URL` | Login page URL for web redirects (optional) | `https://login.example.com` |

## API

### Registration (3 steps)

| Method | Path | Description |
|---|---|---|
| POST | `/api/register/begin` | Send `{"email": "..."}` to receive a verification code |
| POST | `/api/register/verify` | Send `{"email": "...", "code": "..."}` to verify and receive WebAuthn creation options |
| POST | `/api/register/finish` | Complete the WebAuthn ceremony with the authenticator response |

### Login (2 steps)

| Method | Path | Description |
|---|---|---|
| POST | `/api/login/begin` | Request WebAuthn assertion options for discoverable login |
| POST | `/api/login/finish` | Complete the WebAuthn ceremony with the authenticator response |

### Session

Successful registration and login set an `auth_session` cookie with a 24-hour sliding TTL. The session is stored in Redis and refreshed on each verification.

### Verify

| Method | Path | Description |
|---|---|---|
| GET | `/api/verify` | Validate the session cookie |

Returns `200` with `Remote-User` and `Remote-Email` headers if the session is valid. If the session is invalid and `?mode=web` is present with `LOGIN_URL` configured, returns `302` to `${LOGIN_URL}?redirect_uri=https://${X-Forwarded-Host}${X-Forwarded-Uri}`. Otherwise returns `401`.

### Logout

| Method | Path | Description |
|---|---|---|
| POST | `/api/logout` | Delete the session and clear the cookie |

Returns `204 No Content`.

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
  -e RP_ORIGIN=https://example.com \
  -e LOGIN_URL=https://login.example.com \
  auth
```

## Notes

- WebAuthn requires HTTPS in production. `localhost` is the only exception for development.
- Passkeys are scoped to `RP_ID`. Changing it after users have registered will invalidate their credentials.
- The schema is applied automatically on first `docker compose up` via the PostgreSQL init script.
