package account

const historyStoreMax = 12

type HistoryItem struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Location string `json:"location"`
	Photo    string `json:"photo"`
}

type historyBody struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Location string `json:"location"`
	Photo    string `json:"photo"`
}

type importHistoryItem struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Location string `json:"location"`
	Photo    string `json:"photo"`
}
