package model

import (
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"go.local/services/auth-api/internal/db"
)

// Credential wraps a database credential and implements webauthn.User.
// Each passkey is its own independent account with no relationship to other credentials.
type Credential struct {
	DB db.Credential
}

func (c *Credential) WebAuthnID() []byte                         { return c.DB.UserHandle }
func (c *Credential) WebAuthnName() string                       { return c.DB.DisplayName.String }
func (c *Credential) WebAuthnDisplayName() string                { return c.DB.DisplayName.String }
func (c *Credential) WebAuthnCredentials() []webauthn.Credential { return []webauthn.Credential{ToWebAuthnCredential(c.DB)} }

// ToWebAuthnCredential converts a database credential row to a webauthn.Credential struct.
func ToWebAuthnCredential(row db.Credential) webauthn.Credential {
	transport := make([]protocol.AuthenticatorTransport, len(row.Transport))
	for i, t := range row.Transport {
		transport[i] = protocol.AuthenticatorTransport(t)
	}

	return webauthn.Credential{
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
