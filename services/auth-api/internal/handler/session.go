package handler

import (
	"encoding/json"
	"net/http"

	"go.local/services/auth-api/internal/store"
)

func (h *Handler) createAuthSession(w http.ResponseWriter, r *http.Request, displayName string) error {
	token, err := generateSessionID()
	if err != nil {
		return err
	}

	session := &store.AuthSession{
		DisplayName: displayName,
	}

	if err := h.Store.SaveAuthSession(r.Context(), token, session); err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.SecureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "webauthn_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.SecureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	return nil
}

// VerifySession validates the auth_session cookie for Caddy's forward_auth directive.
func (h *Handler) VerifySession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("auth_session")
	if err != nil {
		h.handleUnauthorized(w, r)
		return
	}

	if _, err := h.Store.GetAuthSession(r.Context(), cookie.Value); err != nil {
		h.handleUnauthorized(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleUnauthorized(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
}

// GetNetworkContext returns the network context injected by Caddy via X-Network-Context.
func (h *Handler) GetNetworkContext(w http.ResponseWriter, r *http.Request) {
	network := r.Header.Get("X-Network-Context")
	if network != "local" {
		network = "public"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"network": network})
}

// Logout deletes the auth session and clears the cookie.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("auth_session")
	if err == nil {
		h.Store.DeleteAuthSession(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.SecureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	w.WriteHeader(http.StatusNoContent)
}
