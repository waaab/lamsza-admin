package middleware

import (
	"net/http"
	"strings"
)

// Request body limits.
//
// The handlers decode r.Body without a size check, and the multipart upload
// handlers only cap how much of a form they keep in memory, not how much they
// read. http.MaxBytesReader caps the read itself and makes a decoder or
// ParseMultipartForm fail with a normal error, so the handlers need no change.
// Same mechanism as the main lamsza backend.
//
// Apply LimitBody once, around the whole mux.
const (
	// DefaultMaxBodyBytes covers every JSON route. The largest admin payloads
	// are page and event bodies, far below this.
	DefaultMaxBodyBytes = 2 << 20 // 2 MiB

	// UploadMaxBodyBytes covers the multipart image uploads. The handlers
	// accept an image of up to 8 MiB; this leaves room for the envelope.
	UploadMaxBodyBytes = 10 << 20 // 10 MiB
)

// bodyLimitByPrefix holds the routes that need more than the default. Longest
// matching prefix wins.
var bodyLimitByPrefix = map[string]int64{
	"/api/admin/entry-images": UploadMaxBodyBytes,
	"/api/admin/event-images": UploadMaxBodyBytes,
}

// MaxBodyBytesFor returns the body limit that applies to a request path.
func MaxBodyBytesFor(path string) int64 {
	var (
		best    int64 = DefaultMaxBodyBytes
		bestLen       = -1
	)
	for prefix, limit := range bodyLimitByPrefix {
		if len(prefix) > bestLen && strings.HasPrefix(path, prefix) {
			best = limit
			bestLen = len(prefix)
		}
	}
	return best
}

// LimitBody caps how much of the request body a handler can read.
func LimitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && r.Method != http.MethodGet && r.Method != http.MethodHead {
			r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytesFor(r.URL.Path))
		}
		next.ServeHTTP(w, r)
	})
}
