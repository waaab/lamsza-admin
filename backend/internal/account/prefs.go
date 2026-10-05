package account

import (
	"backend/internal/auth"
	"backend/internal/db"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"
)

func Migrate() {
	migrateUserLinks()
	migrateUserHistory()
	migrateUserFavorites()
	migrateEntryMembers()
	log.Println("Account preferences ready")
}

func NormalizeTheme(v string) (string, error) {
	switch v {
	case "light", "dark", "system":
		return v, nil
	default:
		return "", fmt.Errorf("invalid theme")
	}
}

func ClampSlots(n int) int {
	if n < 7 {
		return 7
	}
	if n > 14 {
		return 14
	}
	return n
}

const maxDisplayNameLen = 24

func NormalizeDisplayName(raw string) (string, error) {
	name := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	if utf8.RuneCountInString(name) > maxDisplayNameLen {
		return "", fmt.Errorf("invalid display_name")
	}
	return name, nil
}

type preferencesBody struct {
	Theme                 *string `json:"theme"`
	QuicklinkSlots        *int    `json:"quicklink_slots"`
	PreferredSettlementID *int    `json:"preferred_settlement_id"`
	DisplayName           *string `json:"display_name"`
}

type importBody struct {
	Theme          string          `json:"theme"`
	QuicklinkSlots *int            `json:"quicklink_slots"`
	Links          json.RawMessage `json:"links"`
	History        json.RawMessage `json:"history"`
}

func HandlePreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	u, err := auth.UserFromRequest(r)
	if err != nil || u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body preferencesBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	setClauses := make([]string, 0, 2)
	args := make([]interface{}, 0, 3)
	argN := 1

	if body.Theme != nil {
		theme, err := NormalizeTheme(*body.Theme)
		if err != nil {
			http.Error(w, "invalid theme", http.StatusBadRequest)
			return
		}
		setClauses = append(setClauses, fmt.Sprintf("theme = $%d", argN))
		args = append(args, theme)
		argN++
	}
	if body.QuicklinkSlots != nil {
		slots := *body.QuicklinkSlots
		if slots < 7 || slots > 14 {
			http.Error(w, "invalid quicklink_slots", http.StatusBadRequest)
			return
		}
		setClauses = append(setClauses, fmt.Sprintf("quicklink_slots = $%d", argN))
		args = append(args, slots)
		argN++
	}
	if body.DisplayName != nil {
		name, err := NormalizeDisplayName(*body.DisplayName)
		if err != nil {
			http.Error(w, "invalid display_name", http.StatusBadRequest)
			return
		}
		setClauses = append(setClauses, fmt.Sprintf("display_name = $%d", argN))
		args = append(args, name)
		argN++
	}
	if body.PreferredSettlementID != nil {
		id := *body.PreferredSettlementID
		if id <= 0 {
			setClauses = append(setClauses, "preferred_settlement_id = NULL")
		} else {
			var found int
			err := db.DB.QueryRow(`
				SELECT id FROM settlements
				WHERE id = $1 AND LOWER(TRIM(COALESCE(type, ''))) <> 'megye'
			`, id).Scan(&found)
			if err != nil {
				http.Error(w, "invalid preferred_settlement_id", http.StatusBadRequest)
				return
			}
			setClauses = append(setClauses, fmt.Sprintf("preferred_settlement_id = $%d", argN))
			args = append(args, id)
			argN++
		}
	}

	if len(setClauses) == 0 {
		if err := auth.WriteMe(w, u.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	args = append(args, u.ID)
	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d", joinSetClauses(setClauses), argN)
	if _, err := db.DB.Exec(query, args...); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := auth.WriteMe(w, u.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func HandleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	u, err := auth.UserFromRequest(r)
	if err != nil || u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body importBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var (
		theme           sql.NullString
		quicklinkSlots  sql.NullInt64
		prefsImportedAt sql.NullTime
	)
	err = tx.QueryRow(`
		SELECT theme, quicklink_slots, prefs_imported_at
		FROM users WHERE id = $1 FOR UPDATE
	`, u.ID).Scan(&theme, &quicklinkSlots, &prefsImportedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if prefsImportedAt.Valid {
		if err := tx.Commit(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := auth.WriteMe(w, u.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	setClauses := make([]string, 0, 3)
	args := make([]interface{}, 0, 4)
	argN := 1

	if !theme.Valid {
		if normalized, err := NormalizeTheme(body.Theme); err == nil {
			setClauses = append(setClauses, fmt.Sprintf("theme = $%d", argN))
			args = append(args, normalized)
			argN++
		}
	}
	if !quicklinkSlots.Valid && body.QuicklinkSlots != nil {
		setClauses = append(setClauses, fmt.Sprintf("quicklink_slots = $%d", argN))
		args = append(args, ClampSlots(*body.QuicklinkSlots))
		argN++
	}

	var linkCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM user_links WHERE user_id = $1`, u.ID).Scan(&linkCount); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if linkCount == 0 {
		if err := insertImportLinks(tx, u.ID, body.Links); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	var historyCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM user_entry_history WHERE user_id = $1`, u.ID).Scan(&historyCount); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if historyCount == 0 {
		if err := insertImportHistory(tx, u.ID, body.History); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	setClauses = append(setClauses, "prefs_imported_at = NOW()")
	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d", joinSetClauses(setClauses), argN)
	args = append(args, u.ID)

	if _, err := tx.Exec(query, args...); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := auth.WriteMe(w, u.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func joinSetClauses(clauses []string) string {
	out := clauses[0]
	for i := 1; i < len(clauses); i++ {
		out += ", " + clauses[i]
	}
	return out
}
