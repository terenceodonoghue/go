CREATE TABLE IF NOT EXISTS users (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webauthn_id  BYTEA UNIQUE NOT NULL,                             -- 64 random bytes; opaque user handle sent to the authenticator
    email        TEXT UNIQUE NOT NULL,                              -- used as WebAuthnName and WebAuthnDisplayName
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS credentials (
    id                   BYTEA PRIMARY KEY,                          -- credential ID assigned by the authenticator
    user_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_key           BYTEA NOT NULL,                             -- verifies signatures during login
    transport            TEXT[] NOT NULL DEFAULT '{}',               -- how the authenticator communicates (internal, usb, ble, nfc)
    sign_count           BIGINT NOT NULL DEFAULT 0,                  -- increments each login; detects cloned credentials
    flag_backup_eligible BOOLEAN NOT NULL DEFAULT false,             -- whether this credential can be synced (set at registration)
    flag_backup_state    BOOLEAN NOT NULL DEFAULT false,             -- whether this credential is currently backed up (updated each login)
    aaguid               BYTEA NOT NULL,                             -- identifies the authenticator model (e.g. YubiKey 5, iCloud Keychain)
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_credentials_user_id ON credentials(user_id);
