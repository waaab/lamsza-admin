package handlers

type reviewPostBody struct {
	Slug  string `json:"slug"`
	Score int    `json:"score"`
	Text  string `json:"text"`
}
