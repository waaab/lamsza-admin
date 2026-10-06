package account

const defaultLinkColor = "#e6f0ff"

type UserLink struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	BgColor  string `json:"bg_color"`
	Position int    `json:"position"`
}

type linkBody struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	BgColor string `json:"bg_color"`
}

type linksPutBody struct {
	Links []UserLink `json:"links"`
}

type importLink struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	BgColor string `json:"bg_color"`
}
