package handler

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"go.local/services/auth-api/internal/db"
	"go.local/services/auth-api/internal/model"
	"go.local/services/auth-api/internal/store"
)

type Handler struct {
	WebAuthn             *webauthn.WebAuthn
	Queries              *db.Queries
	Store                *store.RedisStore
	SecureCookie         bool
	LogVerificationCodes bool
}

/*
BeginRegistration accepts an email, checks it isn't taken, generates a verification code,
and stores it in Redis. If LOG_VERIFICATION_CODES is enabled, the code is logged to the console.
*/
func (h *Handler) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}

	existingUser, err := h.Queries.GetUserByEmail(r.Context(), req.Email)
	if err == nil {
		creds, err := h.Queries.GetCredentialsByUserID(r.Context(), existingUser.ID)
		if err == nil && len(creds) > 0 {
			http.Error(w, "email already registered", http.StatusConflict)
			return
		}
	}

	code, err := generateCode()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := h.Store.SaveVerification(r.Context(), req.Email, code); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if h.LogVerificationCodes {
		log.Printf("Verification code for %s: %s", req.Email, code)
	} else {
		// TODO: send verification code via email provider
		log.Printf("Email sending not implemented, verification code for %s was not delivered", req.Email)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "code_sent"})
}

/*
VerifyAndBeginPasskey validates the verification code, creates the user,
and starts the WebAuthn registration ceremony.
*/
func (h *Handler) VerifyAndBeginPasskey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	storedCode, err := h.Store.GetVerification(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "no pending verification or code expired", http.StatusBadRequest)
		return
	}
	if storedCode != req.Code {
		http.Error(w, "invalid verification code", http.StatusUnauthorized)
		return
	}

	h.Store.DeleteVerification(r.Context(), req.Email)

	dbUser, err := h.Queries.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		webauthnID := make([]byte, 64)
		if _, err := rand.Read(webauthnID); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		dbUser, err = h.Queries.CreateUser(r.Context(), db.CreateUserParams{
			WebauthnID: webauthnID,
			Email:      req.Email,
		})
		if err != nil {
			http.Error(w, "failed to create user", http.StatusInternalServerError)
			return
		}
	}

	user := &model.User{DB: dbUser}

	creation, session, err := h.WebAuthn.BeginRegistration(user,
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
	json.NewEncoder(w).Encode(creation)
}

// FinishRegistration completes the WebAuthn registration ceremony and persists the credential.
func (h *Handler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
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

	user, err := h.Queries.GetUserByWebAuthnID(r.Context(), session.UserID)
	if err != nil {
		http.Error(w, "user not found", http.StatusInternalServerError)
		return
	}

	creds, err := h.Queries.GetCredentialsByUserID(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	modelUser := &model.User{
		DB:          user,
		Credentials: model.ToWebAuthnCredentials(creds),
	}

	credential, err := h.WebAuthn.FinishRegistration(modelUser, *session, r)
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
		UserID:             user.ID,
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

	if err := h.createAuthSession(w, r, user); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "registered"})
}

func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
