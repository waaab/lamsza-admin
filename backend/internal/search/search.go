package search

import (
	"backend/internal/db"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func HandleAutosuggest(w http.ResponseWriter, r *http.Request) {
	query := strings.ToLower(r.URL.Query().Get("q"))
	if len(query) < 3 {
		w.Write([]byte(`[]`))
		return
	}

	sqlQuery := `
		SELECT word FROM (
			SELECT word, priority, 
			       similarity(LOWER(word), LOWER($1)) as sim 
			FROM (
				SELECT name as word, 1 as priority FROM entry_categories
				UNION ALL
				SELECT name as word, 2 as priority FROM tags
				UNION ALL
				SELECT name as word, 3 as priority FROM locations
				UNION ALL
				SELECT name_ro as word, 3 as priority FROM locations WHERE name_ro IS NOT NULL
				UNION ALL
				SELECT name_de as word, 3 as priority FROM locations WHERE name_de IS NOT NULL
				UNION ALL
				SELECT name as word, 4 as priority FROM attractions
				UNION ALL
				SELECT name as word, 4 as priority FROM historical_seats
			) raw
			WHERE word != '' 
			  AND (
			      unaccent(LOWER(word)) LIKE '%' || unaccent(LOWER($1)) || '%' 
			      OR similarity(unaccent(LOWER(word)), unaccent(LOWER($1))) > 0.2
			  )
			GROUP BY word, priority
		) s
		ORDER BY 
			CASE 
				WHEN unaccent(LOWER(word)) = unaccent(LOWER($1)) THEN 1
				WHEN unaccent(LOWER(word)) LIKE unaccent(LOWER($1)) || '%' THEN 2
				ELSE 3
			END,
			sim DESC,
			priority ASC,
			word ASC
		LIMIT 10
	`

	rows, err := db.DB.Query(sqlQuery, query)
	if err != nil {
		log.Println("Autosuggest query error:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	results := []string{}
	for rows.Next() {
		var word string
		if err := rows.Scan(&word); err == nil {
			results = append(results, word)
		}
	}

	if results == nil {
		results = []string{}
	}
	json.NewEncoder(w).Encode(results)
}
