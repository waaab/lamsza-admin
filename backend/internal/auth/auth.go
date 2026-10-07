package auth

import (
	"backend/internal/config"
	"backend/internal/db"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// The admin session is deliberately not the public one (BOG-45).
//
// Before this, both apps used the cookie name `lamsza_session` and the
// `sessions` table, so a token the public site handed to any signed-in visitor
// was accepted by every route on this API. The two apps are separate
// deployments; the trust boundary was not.
//
// Two things keep them apart, and each covers what the other cannot:
//
//   - A different cookie name. Browsers scope cookies by host and ignore the
//     port, so on localhost a cookie set by lamsza on :3001/:5174 is sent to
//     this API on :3000 regardless. A different name means no collision.
//   - A different table. The name alone is only presentation - anyone can put
//     any token in any cookie. The store is the real boundary: an admin token
//     hash exists only in `admin_sessions`, a public one only in `sessions`,
//     so neither API can accept the other's token even when it is handed over
//     deliberately.
//
// `admin_sessions` is created by the main lamsza backend, which owns this
// shared schema - this process runs no DDL (see docs/ARCHITECTURE.md).
const (
	SessionCookieName = "lamsza_admin_session"
	sessionTTL        = 30 * 24 * time.Hour
)

type contextKey int

const userContextKey contextKey = 1

type User struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type GoogleIdentity struct {
	Sub        string
	Email      string
	Name       string
	GivenName  string
	FamilyName string
	Picture    string
	Locale     string
}

type UserProfile struct {
	Sub        string
	Email      string
	Name       string
	GivenName  string
	FamilyName string
	Picture    string
	Locale     string
}

func ApplyGoogleProfile(_ UserProfile, ident GoogleIdentity) UserProfile {
	return UserProfile{
		Sub:        ident.Sub,
		Email:      ident.Email,
		Name:       ident.Name,
		GivenName:  ident.GivenName,
		FamilyName: ident.FamilyName,
		Picture:    ident.Picture,
		Locale:     ident.Locale,
	}
}

type IDTokenVerifier func(idToken, audience string) (GoogleIdentity, error)

// VerifyIDToken is the Google ID-token checker. Tests replace this.
var VerifyIDToken IDTokenVerifier = VerifyGoogleIDToken

// Schema note: the users, sessions and admin_sessions tables are owned by the
// main lamsza backend, which creates them on its own startup path. The admin
// process shares that database and must not run DDL, so the Migrate() helper
// that used to live here was deleted rather than left as a loaded gun.
//
// That includes `admin_sessions`, the table this package reads and writes:
// `lamsza/backend/internal/auth/auth.go` creates it, and
// `lamsza/backend/migrations/admin_sessions.sql` is the explicit form. Do not
// add a CREATE TABLE here when a fresh database is missing it - run the main
// backend, or that migration.

func IsAdmin(email string) bool {
	want := strings.ToLower(strings.TrimSpace(email))
	if want == "" {
		return false
	}
	for _, a := range config.AppConfig.AdminGoogleEmails {
		if strings.ToLower(strings.TrimSpace(a)) == want {
			return true
		}
	}
	return false
}

func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := UserFromRequest(r)
		if err != nil || u == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !IsAdmin(u.Email) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey, u)))
	}
}

type adminUser struct {
	ID            int       `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	GivenName     string    `json:"given_name"`
	FamilyName    string    `json:"family_name"`
	DisplayName   string    `json:"display_name"`
	Locale        string    `json:"locale"`
	Settlement    string    `json:"settlement"`
	CreatedAt     time.Time `json:"created_at"`
	LastLoginAt   time.Time `json:"last_login_at"`
	WebsiteBanned bool      `json:"website_banned"`
	IsAdmin       bool      `json:"is_admin"`
}

// HandleAdminUsers lists registered accounts for the admin users table.
func HandleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rows, err := db.DB.Query(`
		SELECT u.id, u.email, u.name, u.given_name, u.family_name, u.display_name, u.locale,
		       COALESCE(s.name, ''), u.created_at, u.last_login_at, u.website_banned
		FROM users u
		LEFT JOIN settlements s ON s.id = u.preferred_settlement_id
		ORDER BY u.created_at DESC, u.id DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	out := []adminUser{}
	for rows.Next() {
		var u adminUser
		if err := rows.Scan(
			&u.ID, &u.Email, &u.Name, &u.GivenName, &u.FamilyName, &u.DisplayName, &u.Locale,
			&u.Settlement, &u.CreatedAt, &u.LastLoginAt, &u.WebsiteBanned,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		u.IsAdmin = IsAdmin(u.Email)
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(out)
}

func UserFromContext(ctx context.Context) *User {
	u, _ := ctx.Value(userContextKey).(*User)
	return u
}

// WithUser puts the signed-in admin on a context the way RequireAdmin does.
//
// The context key is unexported so nothing outside this package can forge an
// identity by writing to the request context. That also makes the signed-in
// case untestable from another package, which is a problem for the audit trail
// (BOG-48) - its whole job is recording *who*. This is the one sanctioned way
// in, and it takes an *auth.User, so a caller still has to have a real user.
func WithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, userContextKey, u)
}

func UserFromRequest(r *http.Request) (*User, error) {
	c, err := r.Cookie(SessionCookieName)
	if err != nil || c == nil || c.Value == "" {
		return nil, errors.New("no session")
	}
	hash := hashToken(c.Value)
	var u User
	err = db.DB.QueryRow(`
		SELECT u.id, u.email, u.name
		FROM admin_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > NOW()
	`, hash).Scan(&u.ID, &u.Email, &u.Name)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

type googleLoginBody struct {
	Credential string `json:"credential"`
}

func HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	audience := strings.TrimSpace(config.AppConfig.GoogleClientID)
	if audience == "" {
		http.Error(w, "Google login is not configured", http.StatusServiceUnavailable)
		return
	}
	var body googleLoginBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Credential) == "" {
		http.Error(w, "missing credential", http.StatusBadRequest)
		return
	}
	ident, err := VerifyIDToken(body.Credential, audience)
	if err != nil {
		log.Printf("google id token: %v", err)
		http.Error(w, "invalid google token", http.StatusUnauthorized)
		return
	}
	ident.Email = strings.ToLower(strings.TrimSpace(ident.Email))
	if ident.Email == "" || ident.Sub == "" {
		http.Error(w, "google account has no email", http.StatusUnauthorized)
		return
	}

	// Only an allowlisted account gets an admin session. Every route on this API
	// is behind RequireAdmin, so a session for anyone else bought nothing - but
	// it did put a public-grade row in `admin_sessions`, which is the one thing
	// a separate session store must never hold. A row here now means an admin is
	// signed in, and that is what makes the table worth separating.
	if !IsAdmin(ident.Email) {
		http.Error(w, "Ehhez a fiókhoz nincs admin jogosultság.", http.StatusForbidden)
		return
	}

	profile := ApplyGoogleProfile(UserProfile{}, ident)

	var userID int
	err = db.DB.QueryRow(`
		INSERT INTO users (google_sub, email, name, given_name, family_name, picture, locale, last_login_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (google_sub) DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			given_name = EXCLUDED.given_name,
			family_name = EXCLUDED.family_name,
			picture = EXCLUDED.picture,
			locale = EXCLUDED.locale,
			last_login_at = NOW()
		RETURNING id
	`, profile.Sub, profile.Email, profile.Name, profile.GivenName, profile.FamilyName, profile.Picture, profile.Locale).Scan(&userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := newSessionToken()
	if err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	expires := time.Now().Add(sessionTTL)
	_, err = db.DB.Exec(
		`INSERT INTO admin_sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		hashToken(token), userID, expires,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, sessionCookie(r, token, expires))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"email":    ident.Email,
		"name":     ident.Name,
		"is_admin": IsAdmin(ident.Email),
	})
}

func HandleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	u, err := UserFromRequest(r)
	if err != nil || u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := WriteMe(w, u.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func WriteMe(w http.ResponseWriter, userID int) error {
	var (
		email, name, givenName, familyName, picture, locale, googleSub, displayName string
		theme                                                                       sql.NullString
		quicklinkSlots                                                              sql.NullInt64
		lastLoginAt, createdAt                                                      time.Time
		preferredID                                                                 sql.NullInt64
		preferredSlug, preferredName, preferredCounty                               sql.NullString
	)
	err := db.DB.QueryRow(`
		SELECT u.email, u.name, u.given_name, u.family_name, u.picture, u.locale, u.google_sub, u.display_name,
		       u.last_login_at, u.created_at, u.theme, u.quicklink_slots,
		       u.preferred_settlement_id, s.slug, s.name, c.slug
		FROM users u
		LEFT JOIN settlements s ON s.id = u.preferred_settlement_id
		LEFT JOIN counties c ON c.id = s.county_id
		WHERE u.id = $1
	`, userID).Scan(
		&email, &name, &givenName, &familyName, &picture, &locale, &googleSub, &displayName,
		&lastLoginAt, &createdAt, &theme, &quicklinkSlots,
		&preferredID, &preferredSlug, &preferredName, &preferredCounty,
	)
	if err != nil {
		return err
	}
	resp := map[string]interface{}{
		"email":         email,
		"name":          name,
		"is_admin":      IsAdmin(email),
		"given_name":    givenName,
		"family_name":   familyName,
		"picture":       picture,
		"locale":        locale,
		"google_sub":    googleSub,
		"display_name":  displayName,
		"last_login_at": lastLoginAt,
		"created_at":    createdAt,
	}
	if theme.Valid {
		resp["theme"] = theme.String
	} else {
		resp["theme"] = nil
	}
	if quicklinkSlots.Valid {
		resp["quicklink_slots"] = quicklinkSlots.Int64
	} else {
		resp["quicklink_slots"] = nil
	}
	// No prefs_imported_at: the shared auth store runs its one-time browser
	// import only when /api/auth/me reports that field, and this app has no
	// /api/account/import to receive it (the main lamsza backend does).
	if preferredID.Valid && preferredSlug.Valid && strings.TrimSpace(preferredSlug.String) != "" {
		resp["preferred_location"] = map[string]interface{}{
			"id":          preferredID.Int64,
			"slug":        preferredSlug.String,
			"name":        preferredName.String,
			"county_slug": preferredCounty.String,
		}
	} else {
		resp["preferred_location"] = nil
	}
	if IsAdmin(email) {
		var queueCount int
		err = db.DB.QueryRow(`
			SELECT
			  (SELECT COUNT(*) FROM entries WHERE published = false) +
			  (SELECT COUNT(*) FROM entry_members WHERE status = 'pending' AND role = 'member') +
			  (SELECT COUNT(*) FROM entry_members WHERE status = 'pending' AND role = 'owner') +
			  (SELECT COUNT(*) FROM entry_suggestions WHERE status = 'open') +
			  (SELECT COUNT(*) FROM attraction_suggestions WHERE status = 'open') +
			  (SELECT COUNT(*) FROM websites WHERE status = 'pending')
		`).Scan(&queueCount)
		if err != nil {
			return err
		}
		resp["admin_queue_count"] = queueCount
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(resp)
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if c, err := r.Cookie(SessionCookieName); err == nil && c != nil && c.Value != "" {
		_, _ = db.DB.Exec(`DELETE FROM admin_sessions WHERE token_hash = $1`, hashToken(c.Value))
	}
	expired := time.Unix(0, 0)
	http.SetCookie(w, sessionCookie(r, "", expired))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func sessionCookie(r *http.Request, value string, expires time.Time) *http.Cookie {
	maxAge := int(time.Until(expires).Seconds())
	if maxAge < 0 || value == "" {
		maxAge = -1
	}
	secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

func newSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func VerifyGoogleIDToken(idToken, audience string) (GoogleIdentity, error) {
	if strings.TrimSpace(idToken) == "" || strings.TrimSpace(audience) == "" {
		return GoogleIdentity{}, errors.New("missing token or audience")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	endpoint := "https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(idToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return GoogleIdentity{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return GoogleIdentity{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return GoogleIdentity{}, fmt.Errorf("tokeninfo status %d", resp.StatusCode)
	}
	var info struct {
		Aud           string          `json:"aud"`
		Sub           string          `json:"sub"`
		Email         string          `json:"email"`
		EmailVerified json.RawMessage `json:"email_verified"`
		Name          string          `json:"name"`
		GivenName     string          `json:"given_name"`
		FamilyName    string          `json:"family_name"`
		Picture       string          `json:"picture"`
		Locale        string          `json:"locale"`
	}
	if err := json.Unmarshal(raw, &info); err != nil {
		return GoogleIdentity{}, err
	}
	if info.Aud != audience {
		return GoogleIdentity{}, errors.New("token audience mismatch")
	}
	if !emailVerified(info.EmailVerified) {
		return GoogleIdentity{}, errors.New("google email not verified")
	}
	if info.Email == "" || info.Sub == "" {
		return GoogleIdentity{}, errors.New("token missing email")
	}
	return GoogleIdentity{
		Sub:        info.Sub,
		Email:      info.Email,
		Name:       info.Name,
		GivenName:  info.GivenName,
		FamilyName: info.FamilyName,
		Picture:    info.Picture,
		Locale:     info.Locale,
	}, nil
}

func emailVerified(raw json.RawMessage) bool {
	s := strings.TrimSpace(strings.Trim(string(raw), `"`))
	return s == "true"
}

// ParseTestIDToken accepts "test:email@domain" credentials. Tests only.
func ParseTestIDToken(idToken, audience string) (GoogleIdentity, error) {
	if audience == "" {
		return GoogleIdentity{}, errors.New("missing audience")
	}
	if !strings.HasPrefix(idToken, "test:") {
		return GoogleIdentity{}, errors.New("invalid test token")
	}
	email := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(idToken, "test:")))
	if email == "" || !strings.Contains(email, "@") {
		return GoogleIdentity{}, errors.New("invalid test email")
	}
	return GoogleIdentity{
		Sub:   "test-sub-" + email,
		Email: email,
		Name:  "Test User",
	}, nil
}
