CREATE TABLE IF NOT EXISTS credentials (
    id                   BYTEA PRIMARY KEY,                          -- credential ID assigned by the authenticator
    user_handle          BYTEA UNIQUE NOT NULL,                      -- opaque user handle sent to the authenticator; used in discoverable login
    display_name         TEXT,                                       -- user-supplied passkey label (e.g. "Terence's MacBook")
    public_key           BYTEA NOT NULL,                             -- verifies signatures during login
    transport            TEXT[] NOT NULL DEFAULT '{}',               -- how the authenticator communicates (internal, usb, ble, nfc)
    sign_count           BIGINT NOT NULL DEFAULT 0,                  -- increments each login; detects cloned credentials
    flag_backup_eligible BOOLEAN NOT NULL DEFAULT false,             -- whether this credential can be synced (set at registration)
    flag_backup_state    BOOLEAN NOT NULL DEFAULT false,             -- whether this credential is currently backed up (updated each login)
    aaguid               BYTEA NOT NULL,                             -- identifies the authenticator model (e.g. YubiKey 5, iCloud Keychain)
    last_used_at         TIMESTAMPTZ,                                -- updated on every successful login
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);
