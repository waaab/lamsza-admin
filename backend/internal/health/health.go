package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"backend/internal/db"
)

type response struct {
	OK bool   `json:"ok"`
	DB string `json:"db"`
}

// HandleHealth answers the liveness probe. It returns 200 only when the
// database answers, so a running process with a dead database reads as down.
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	status := http.StatusOK
	body := response{OK: true, DB: "up"}

	if !pingDB(r.Context()) {
		status = http.StatusServiceUnavailable
		body = response{OK: false, DB: "down"}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func pingDB(parent context.Context) bool {
	if db.DB == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	return db.DB.PingContext(ctx) == nil
}
