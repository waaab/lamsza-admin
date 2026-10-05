package handlers

import (
	"backend/internal/db"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const eventImageURLPrefix = "/api/media/event-images/"
const maxEventImageBytes = 8 << 20

// EventImagesDir returns the filesystem directory for uploaded event images (default: data/event-images under cwd).
func EventImagesDir() string {
	d := os.Getenv("EVENT_IMAGES_DIR")
	if d == "" {
		d = "data/event-images"
	}
	return d
}

// EnsureEventImagesDir creates the upload directory if missing.
func EnsureEventImagesDir() error {
	return os.MkdirAll(EventImagesDir(), 0o755)
}

// HandleEventImageUpload POST multipart/form-data field "file", optional "event_id" to replace a previous upload for that event.
func HandleEventImageUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(maxEventImageBytes); err != nil {
		http.Error(w, "Invalid multipart body", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == ".jpeg" {
		ext = ".jpg"
	}
	switch ext {
	case ".jpg", ".png", ".webp", ".gif":
	default:
		http.Error(w, "Allowed: JPEG, PNG, WebP, GIF", http.StatusBadRequest)
		return
	}

	var oldStored string
	if evStr := strings.TrimSpace(r.FormValue("event_id")); evStr != "" {
		if id, err := strconv.Atoi(evStr); err == nil && id > 0 {
			_ = db.DB.QueryRow(`SELECT COALESCE(featured_image,'') FROM events WHERE id = $1`, id).Scan(&oldStored)
		}
	}

	randPart := make([]byte, 8)
	if _, err := rand.Read(randPart); err != nil {
		http.Error(w, "Random error", http.StatusInternalServerError)
		return
	}
	hexRand := hex.EncodeToString(randPart)

	var baseName string
	if evStr := strings.TrimSpace(r.FormValue("event_id")); evStr != "" {
		if id, err := strconv.Atoi(evStr); err == nil && id > 0 {
			baseName = fmt.Sprintf("event-%d-%s%s", id, hexRand[:12], ext)
		}
	}
	if baseName == "" {
		baseName = hexRand + ext
	}

	dir := EventImagesDir()
	fullPath := filepath.Join(dir, baseName)
	out, err := os.Create(fullPath)
	if err != nil {
		http.Error(w, "Cannot write file", http.StatusInternalServerError)
		return
	}
	defer out.Close()
	n, err := io.Copy(out, file)
	if err != nil || n == 0 {
		_ = os.Remove(fullPath)
		http.Error(w, "Upload failed", http.StatusInternalServerError)
		return
	}

	newURL := eventImageURLPrefix + baseName
	if oldStored != "" && strings.HasPrefix(oldStored, eventImageURLPrefix) && oldStored != newURL {
		oldBase := filepath.Base(strings.TrimPrefix(oldStored, eventImageURLPrefix))
		if oldBase != "" && oldBase != "." && oldBase != ".." {
			oldPath := filepath.Join(dir, oldBase)
			if oldPath != fullPath {
				_ = os.Remove(oldPath)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"url": newURL})
}
