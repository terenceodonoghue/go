package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-webauthn/webauthn/webauthn"
	"go.local/services/auth-api/internal/db"
	"go.local/services/auth-api/internal/model"
)

// BeginLogin starts a discoverable login ceremony (no username required).
func (h *Handler) BeginLogin(w http.ResponseWriter, r *http.Request) {
	assertion, session, err := h.WebAuthn.BeginDiscoverableLogin()
	if err != nil {
		http.Error(w, "failed to begin login", http.StatusInternalServerError)
		return
	}

	sessionID, err := generateSessionID()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := h.Store.SaveWebAuthnSession(r.Context(), sessionID, session); err != nil {
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
	json.NewEncoder(w).Encode(assertion)
}

// FinishLogin completes the discoverable login ceremony.
func (h *Handler) FinishLogin(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("webauthn_session")
	if err != nil {
		http.Error(w, "missing session cookie", http.StatusBadRequest)
		return
	}

	session, err := h.Store.GetWebAuthnSession(r.Context(), cookie.Value)
	if err != nil {
		http.Error(w, "session expired or invalid", http.StatusBadRequest)
		return
	}

	h.Store.DeleteWebAuthnSession(r.Context(), cookie.Value)

	var authenticatedCred db.Credential

	discoverableUserHandler := func(rawID, userHandle []byte) (webauthn.User, error) {
		dbCred, err := h.Queries.GetCredential(r.Context(), userHandle)
		if err != nil {
			return nil, err
		}
		authenticatedCred = dbCred
		return &model.Credential{DB: dbCred}, nil
	}

	credential, err := h.WebAuthn.FinishDiscoverableLogin(discoverableUserHandler, *session, r)
	if err != nil {
		http.Error(w, "login failed", http.StatusUnauthorized)
		return
	}

	if err := h.Queries.UpdateCredential(r.Context(), db.UpdateCredentialParams{
		ID:              credential.ID,
		SignCount:       int64(credential.Authenticator.SignCount),
		FlagBackupState: credential.Flags.BackupState,
	}); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := h.createAuthSession(w, r, authenticatedCred.DisplayName.String); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "authenticated"})
}
