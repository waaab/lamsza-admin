package middleware

import (
	"net/http"

	"backend/internal/config"
)

// ApplyCORS answers cross-origin calls for the origins on the allowlist only.
//
// It must never echo an unknown Origin. All four Lámsza sites sit under one
// registrable domain, so the session cookie is same-site for every sibling and
// SameSite=Lax does not hold it back. Echoing the Origin together with
// Access-Control-Allow-Credentials therefore let any site, sibling or not, read
// the signed-in reply of an admin who was browsing it. Every route on this API
// is an admin route, so there is no reply here that a stranger may read.
//
// The allowlist also stands in for a CSRF token: a page cannot forge the Origin
// header, so a writing call that carries an Origin we do not know is not coming
// from one of our own pages and is refused. Clients outside a browser (curl,
// the deployment scripts) send no Origin and are unaffected.
func ApplyCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		// The reply differs per Origin, so caches must key on it even when we
		// send no Allow-Origin at all.
		w.Header().Set("Vary", "Origin")

		allowed := config.OriginAllowed(origin)
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "600")
		}

		if r.Method == http.MethodOptions {
			if origin != "" && !allowed {
				http.Error(w, "origin not allowed", http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		if origin != "" && !allowed && isWriteMethod(r.Method) {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

func isWriteMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}
