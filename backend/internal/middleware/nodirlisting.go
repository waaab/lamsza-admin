package middleware

import (
	"net/http"
	"strings"
)

// NoDirListing wraps a file server so it serves files but never a directory
// index. http.FileServer lists a directory's contents for any path ending in
// "/", which told anyone the names of every uploaded image.
func NoDirListing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
