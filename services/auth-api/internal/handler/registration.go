package handler

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5/pgtype"
	"go.local/services/auth-api/internal/db"
	"go.local/services/auth-api/internal/model"
	"go.local/services/auth-api/internal/store"
)

type Handler struct {
	WebAuthn     *webauthn.WebAuthn
	Queries      *db.Queries
	Store        *store.RedisStore
	SecureCookie bool
}

// BeginPasskeyRegistration starts the WebAuthn registration ceremony directly.
// Only accessible when X-Network-Context: local is present.
func (h *Handler) BeginPasskeyRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Network-Context") != "local" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	userHandle := make([]byte, 32)
	if _, err := rand.Read(userHandle); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	cred := &model.Credential{DB: db.Credential{
		UserHandle:  userHandle,
		DisplayName: pgtype.Text{String: req.Name, Valid: true},
	}}

	creation, session, err := h.WebAuthn.BeginRegistration(cred,
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
	)
	if err != nil {
		http.Error(w, "failed to begin registration", http.StatusInternalServerError)
		return
	}

	sessionID, err := generateSessionID()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := h.Store.SaveRegistrationSession(r.Context(), sessionID, &store.RegistrationSession{
		DisplayName: req.Name,
		WebAuthn:    session,
	}); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "webauthn_session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.SecureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(creation)
}

// FinishRegistration completes the WebAuthn registration ceremony and persists the credential.
func (h *Handler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("webauthn_session")
	if err != nil {
		http.Error(w, "missing session cookie", http.StatusBadRequest)
		return
	}

	regSession, err := h.Store.GetRegistrationSession(r.Context(), cookie.Value)
	if err != nil {
		http.Error(w, "session expired or invalid", http.StatusBadRequest)
		return
	}

	h.Store.DeleteRegistrationSession(r.Context(), cookie.Value)

	// WebAuthn.UserID is the user_handle echoed back from BeginRegistration.
	cred := &model.Credential{DB: db.Credential{
		UserHandle:  regSession.WebAuthn.UserID,
		DisplayName: pgtype.Text{String: regSession.DisplayName, Valid: true},
	}}

	credential, err := h.WebAuthn.FinishRegistration(cred, *regSession.WebAuthn, r)
	if err != nil {
		http.Error(w, "registration failed", http.StatusBadRequest)
		return
	}

	transport := make([]string, len(credential.Transport))
	for i, t := range credential.Transport {
		transport[i] = string(t)
	}

	_, err = h.Queries.CreateCredential(r.Context(), db.CreateCredentialParams{
		ID:                 credential.ID,
		UserHandle:         regSession.WebAuthn.UserID,
		DisplayName:        pgtype.Text{String: regSession.DisplayName, Valid: true},
		PublicKey:          credential.PublicKey,
		Transport:          transport,
		SignCount:          int64(credential.Authenticator.SignCount),
		FlagBackupEligible: credential.Flags.BackupEligible,
		FlagBackupState:    credential.Flags.BackupState,
		Aaguid:             credential.Authenticator.AAGUID,
	})
	if err != nil {
		http.Error(w, "failed to save credential", http.StatusInternalServerError)
		return
	}

	if err := h.createAuthSession(w, r, regSession.DisplayName); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "registered"})
}

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
