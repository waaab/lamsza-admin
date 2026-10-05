package directory

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

// MaxCategories is one primary leaf plus four more.
const MaxCategories = 5

// ErrCategorySet means the ids are missing, not leaves, or over the cap.
var ErrCategorySet = errors.New("invalid category set")

// Queryer is the subset of *sql.DB and *sql.Tx these helpers use.
type Queryer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// NormalizeCategoryIDs keeps the primary leaf first, drops blanks and duplicates,
// and rejects a set larger than MaxCategories.
func NormalizeCategoryIDs(primary int, extra []int) ([]int, error) {
	if primary <= 0 {
		return nil, ErrCategorySet
	}
	out := []int{primary}
	seen := map[int]bool{primary: true}
	for _, id := range extra {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
		if len(out) > MaxCategories {
			return nil, ErrCategorySet
		}
	}
	return out, nil
}

// SearchName joins every leaf name so the entry search vector matches each shelf.
func SearchName(names []string) string {
	return strings.Join(names, " ")
}

// ValidateLeaves checks that every id is a subcategory and returns the names in order.
func ValidateLeaves(q Queryer, ids []int) ([]string, error) {
	if len(ids) == 0 || len(ids) > MaxCategories {
		return nil, ErrCategorySet
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		var name string
		var parentID sql.NullInt64
		err := q.QueryRow(`SELECT name, parent_id FROM entry_categories WHERE id = $1`, id).Scan(&name, &parentID)
		if err != nil || !parentID.Valid {
			return nil, ErrCategorySet
		}
		names = append(names, name)
	}
	return names, nil
}

// ReplaceEntryCategories stores the set, with ids[0] as the primary leaf.
func ReplaceEntryCategories(q Queryer, entryID int, ids []int) error {
	names, err := ValidateLeaves(q, ids)
	if err != nil {
		return err
	}
	if _, err := q.Exec(`DELETE FROM entry_category_links WHERE entry_id = $1`, entryID); err != nil {
		return err
	}
	for i, id := range ids {
		if _, err := q.Exec(`
			INSERT INTO entry_category_links (entry_id, category_id, is_primary)
			VALUES ($1, $2, $3)
		`, entryID, id, i == 0); err != nil {
			return err
		}
	}
	_, err = q.Exec(`UPDATE entries SET category_id = $1, cat_name = $2 WHERE id = $3`, ids[0], SearchName(names), entryID)
	return err
}

// ReplaceWebsiteCategories stores the set, with ids[0] as the primary leaf.
func ReplaceWebsiteCategories(q Queryer, websiteID int, ids []int) error {
	if _, err := ValidateLeaves(q, ids); err != nil {
		return err
	}
	if _, err := q.Exec(`DELETE FROM website_category_links WHERE website_id = $1`, websiteID); err != nil {
		return err
	}
	for i, id := range ids {
		if _, err := q.Exec(`
			INSERT INTO website_category_links (website_id, category_id, is_primary)
			VALUES ($1, $2, $3)
		`, websiteID, id, i == 0); err != nil {
			return err
		}
	}
	_, err := q.Exec(`UPDATE websites SET category_id = $1 WHERE id = $2`, ids[0], websiteID)
	return err
}

// LoadEntryCategoryIDs returns the primary leaf first, then the other leaves.
func LoadEntryCategoryIDs(q Queryer, entryID int) ([]int, error) {
	return loadIDs(q, "entry_category_links", "entry_id", entryID)
}

// LoadWebsiteCategoryIDs returns the primary leaf first, then the other leaves.
func LoadWebsiteCategoryIDs(q Queryer, websiteID int) ([]int, error) {
	return loadIDs(q, "website_category_links", "website_id", websiteID)
}

func loadIDs(q Queryer, table, column string, id int) ([]int, error) {
	rows, err := q.Query(fmt.Sprintf(`
		SELECT category_id FROM %s WHERE %s = $1
		ORDER BY is_primary DESC, category_id ASC
	`, table, column), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int{}
	for rows.Next() {
		var categoryID int
		if err := rows.Scan(&categoryID); err != nil {
			return nil, err
		}
		out = append(out, categoryID)
	}
	return out, rows.Err()
}

// NamesForEntries maps an entry id to its category names, primary first.
func NamesForEntries(q Queryer, ids []int) (map[int][]string, error) {
	out := map[int][]string{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.Query(`
		SELECT l.entry_id, c.name
		FROM entry_category_links l
		JOIN entry_categories c ON c.id = l.category_id
		WHERE l.entry_id = ANY($1)
		ORDER BY l.entry_id, l.is_primary DESC, c.sort_order, c.name
	`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var entryID int
		var name string
		if err := rows.Scan(&entryID, &name); err != nil {
			return nil, err
		}
		out[entryID] = append(out[entryID], name)
	}
	return out, rows.Err()
}

// NamesForWebsites maps a website id to its category names, primary first.
func NamesForWebsites(q Queryer, ids []int) (map[int][]string, error) {
	out := map[int][]string{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.Query(`
		SELECT l.website_id, c.name
		FROM website_category_links l
		JOIN entry_categories c ON c.id = l.category_id
		WHERE l.website_id = ANY($1)
		ORDER BY l.website_id, l.is_primary DESC, c.sort_order, c.name
	`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var websiteID int
		var name string
		if err := rows.Scan(&websiteID, &name); err != nil {
			return nil, err
		}
		out[websiteID] = append(out[websiteID], name)
	}
	return out, rows.Err()
}

// MoveCategoryLinks retargets membership rows from one leaf to another.
func MoveCategoryLinks(q Queryer, fromID, toID int) error {
	for _, table := range []string{"entry_category_links", "website_category_links"} {
		if _, err := q.Exec(fmt.Sprintf(`
			DELETE FROM %s AS old
			WHERE old.category_id = $1
			AND EXISTS (
				SELECT 1 FROM %s AS keep
				WHERE keep.category_id = $2
				AND keep.%s = old.%s
			)
		`, table, table, ownerColumn(table), ownerColumn(table)), fromID, toID); err != nil {
			return err
		}
		if _, err := q.Exec(fmt.Sprintf(`
			UPDATE %s SET category_id = $1 WHERE category_id = $2
		`, table), toID, fromID); err != nil {
			return err
		}
		if _, err := q.Exec(fmt.Sprintf(`
			UPDATE %s AS dest
			SET is_primary = true
			WHERE dest.category_id = $1
			AND NOT EXISTS (
				SELECT 1 FROM %s AS prim
				WHERE prim.%s = dest.%s AND prim.is_primary
			)
		`, table, table, ownerColumn(table), ownerColumn(table)), toID); err != nil {
			return err
		}
	}
	if _, err := q.Exec(`
		UPDATE entries e
		SET category_id = l.category_id,
			cat_name = (
				SELECT string_agg(c.name, ' ' ORDER BY l2.is_primary DESC, c.sort_order, c.name)
				FROM entry_category_links l2
				JOIN entry_categories c ON c.id = l2.category_id
				WHERE l2.entry_id = e.id
			)
		FROM entry_category_links l
		WHERE l.entry_id = e.id AND l.is_primary AND l.category_id = $1
	`, toID); err != nil {
		return err
	}
	_, err := q.Exec(`
		UPDATE websites w
		SET category_id = l.category_id
		FROM website_category_links l
		WHERE l.website_id = w.id AND l.is_primary AND l.category_id = $1
	`, toID)
	return err
}

func ownerColumn(table string) string {
	if table == "website_category_links" {
		return "website_id"
	}
	return "entry_id"
}
