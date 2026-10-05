package handlers

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const entryImageURLPrefix = "/api/media/entry-images/"
const maxEntryImageBytes = 8 << 20
const defaultPhotoWidth = 1600
const defaultPhotoHeight = 1200

// EntryImagesDir returns the filesystem directory for uploaded entry gallery images.
func EntryImagesDir() string {
	d := os.Getenv("ENTRY_IMAGES_DIR")
	if d == "" {
		d = "data/entry-images"
	}
	return d
}

// EnsureEntryImagesDir creates the upload directory if missing.
func EnsureEntryImagesDir() error {
	return os.MkdirAll(EntryImagesDir(), 0o755)
}

// HandleEntryImageUpload POST multipart/form-data field "file", optional "entry_id" for filename prefix.
func HandleEntryImageUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(maxEntryImageBytes); err != nil {
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

	data, err := io.ReadAll(io.LimitReader(file, maxEntryImageBytes+1))
	if err != nil {
		http.Error(w, "Upload failed", http.StatusInternalServerError)
		return
	}
	if len(data) == 0 {
		http.Error(w, "Empty file", http.StatusBadRequest)
		return
	}
	if int64(len(data)) > maxEntryImageBytes {
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return
	}

	width, height := imageSize(data)

	randPart := make([]byte, 8)
	if _, err := rand.Read(randPart); err != nil {
		http.Error(w, "Random error", http.StatusInternalServerError)
		return
	}
	hexRand := hex.EncodeToString(randPart)

	var baseName string
	if idStr := strings.TrimSpace(r.FormValue("entry_id")); idStr != "" {
		if id, err := strconv.Atoi(idStr); err == nil && id > 0 {
			baseName = fmt.Sprintf("entry-%d-%s%s", id, hexRand[:12], ext)
		}
	}
	if baseName == "" {
		baseName = hexRand + ext
	}

	dir := EntryImagesDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		http.Error(w, "Cannot write file", http.StatusInternalServerError)
		return
	}
	fullPath := filepath.Join(dir, baseName)
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		http.Error(w, "Cannot write file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"url":    entryImageURLPrefix + baseName,
		"width":  width,
		"height": height,
	})
}

func imageSize(data []byte) (int, int) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err == nil && cfg.Width > 0 && cfg.Height > 0 {
		return cfg.Width, cfg.Height
	}
	if w, h, ok := webpDimensions(data); ok {
		return w, h
	}
	return defaultPhotoWidth, defaultPhotoHeight
}

func webpDimensions(b []byte) (int, int, bool) {
	if len(b) < 30 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WEBP" {
		return 0, 0, false
	}
	kind := string(b[12:16])
	switch kind {
	case "VP8X":
		w := 1 + int(b[24]) + int(b[25])<<8 + int(b[26])<<16
		h := 1 + int(b[27]) + int(b[28])<<8 + int(b[29])<<16
		return w, h, w > 0 && h > 0
	case "VP8 ":
		for i := 20; i+10 < len(b); i++ {
			if b[i] == 0x9d && b[i+1] == 0x01 && b[i+2] == 0x2a {
				w := int(b[i+3]) | int(b[i+4])<<8
				h := int(b[i+5]) | int(b[i+6])<<8
				return w & 0x3fff, h & 0x3fff, true
			}
		}
	case "VP8L":
		if len(b) < 25 || b[20] != 0x2f {
			return 0, 0, false
		}
		bits := uint32(b[21]) | uint32(b[22])<<8 | uint32(b[23])<<16 | uint32(b[24])<<24
		w := int(bits&0x3fff) + 1
		h := int((bits>>14)&0x3fff) + 1
		return w, h, w > 0 && h > 0
	}
	return 0, 0, false
}
