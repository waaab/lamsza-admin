package account

type FavoriteItem struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Slug       *string  `json:"slug"`
	CountySlug *string  `json:"county_slug"`
	Latitude   *float64 `json:"latitude"`
	Longitude  *float64 `json:"longitude"`
	CreatedAt  string   `json:"created_at"`
}

type favoritesResponse struct {
	Settlements []FavoriteItem `json:"settlements"`
	Attractions []FavoriteItem `json:"attractions"`
	Entries     []FavoriteItem `json:"entries"`
	Events      []FavoriteItem `json:"events"`
}

type favoriteBody struct {
	Type string  `json:"type"`
	ID   jsonInt `json:"id"`
}

type favoriteRow struct {
	Type      string `json:"type"`
	ID        int    `json:"id"`
	CreatedAt string `json:"created_at"`
}
