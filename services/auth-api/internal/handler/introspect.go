package handler

import (
	"net/http"
	"strings"
)

// Introspect validates either a session cookie or a Bearer token.
// Used by Caddy's forward_auth directive.
func (h *Handler) Introspect(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("auth_session"); err == nil {
		if _, err := h.Store.GetAuthSession(r.Context(), cookie.Value); err == nil {
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	if token := parseBearerToken(r); token != "" {
		if _, err := h.Queries.IntrospectAPIToken(r.Context(), token); err == nil {
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	w.WriteHeader(http.StatusUnauthorized)
}

func parseBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if token, ok := strings.CutPrefix(auth, "Bearer "); ok {
		return token
	}
	return ""
}
