package model

import (
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"go.local/services/auth/internal/db"
)

// User wraps a database user and their credentials to implement the webauthn.User interface.
type User struct {
	DB          db.User
	Credentials []webauthn.Credential
}

func (u *User) WebAuthnID() []byte                         { return u.DB.WebauthnID }
func (u *User) WebAuthnName() string                       { return u.DB.Email }
func (u *User) WebAuthnDisplayName() string                { return u.DB.Email }
func (u *User) WebAuthnCredentials() []webauthn.Credential { return u.Credentials }

// ToWebAuthnCredentials converts database credential rows to webauthn.Credential structs.
func ToWebAuthnCredentials(rows []db.Credential) []webauthn.Credential {
	creds := make([]webauthn.Credential, len(rows))
	for i, row := range rows {
		transport := make([]protocol.AuthenticatorTransport, len(row.Transport))
		for j, t := range row.Transport {
			transport[j] = protocol.AuthenticatorTransport(t)
		}

		creds[i] = webauthn.Credential{
			ID:        row.ID,
			PublicKey: row.PublicKey,
			Transport: transport,
			Flags: webauthn.CredentialFlags{
				BackupEligible: row.FlagBackupEligible,
				BackupState:    row.FlagBackupState,
			},
			Authenticator: webauthn.Authenticator{
				AAGUID:    row.Aaguid,
				SignCount: uint32(row.SignCount),
			},
		}
	}
	return creds
}
