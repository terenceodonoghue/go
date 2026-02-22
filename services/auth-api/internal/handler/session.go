package handler

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"go.local/services/auth-api/internal/db"
	"go.local/services/auth-api/internal/store"
)

func (h *Handler) createAuthSession(w http.ResponseWriter, r *http.Request, user db.User) error {
	token, err := generateSessionID()
	if err != nil {
		return err
	}

	uid, err := uuid.FromBytes(user.ID.Bytes[:])
	if err != nil {
		return err
	}

	session := &store.AuthSession{
		UserID: uid.String(),
		Email:  user.Email,
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

	session, err := h.Store.GetAuthSession(r.Context(), cookie.Value)
	if err != nil {
		h.handleUnauthorized(w, r)
		return
	}

	w.Header().Set("Remote-User", session.UserID)
	w.Header().Set("Remote-Email", session.Email)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleUnauthorized(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("mode") == "web" && h.LoginURL != "" {
		host := r.Header.Get("X-Forwarded-Host")
		uri := r.Header.Get("X-Forwarded-Uri")
		redirectURI := fmt.Sprintf("https://%s%s", host, uri)

		target, err := url.Parse(h.LoginURL)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		q := target.Query()
		q.Set("redirect_uri", redirectURI)
		target.RawQuery = q.Encode()

		http.Redirect(w, r, target.String(), http.StatusFound)
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
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
