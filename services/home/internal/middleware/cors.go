package middleware

import (
	"net/http"
	"slices"
	"strings"
)

var originAllowlist = []string{
	"http://localhost:5173",
}

var methodAllowlist = []string{http.MethodGet}

func CORS(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		method := r.Header.Get("Access-Control-Request-Method")

		if slices.Contains(originAllowlist, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		if slices.Contains(methodAllowlist, method) && isPreflight(r) {
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(methodAllowlist, ", "))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions &&
		r.Header.Get("Origin") != "" &&
		r.Header.Get("Access-Control-Request-Method") != ""
}
