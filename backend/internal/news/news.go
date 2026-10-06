package news

import (
	"backend/internal/db"
	"backend/internal/models"
	"encoding/json"
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var imgSrcRe = regexp.MustCompile(`<img[^>]+src=["']([^"']+)["']`)

type rssRoot struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string       `xml:"title"`
	Link        string       `xml:"link"`
	Description string       `xml:"description"`
	PubDate     string       `xml:"pubDate"`
	Enclosure   rssEnclosure `xml:"enclosure"`
	Content     rssMedia     `xml:"content"`
	Thumbnail   rssMedia     `xml:"thumbnail"`
}

type rssEnclosure struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

type rssMedia struct {
	URL string `xml:"url,attr"`
}

type newsItem struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	PubDate int64  `json:"pubDate"`
	Source  string `json:"source"`
	BgColor string `json:"bgColor"`
	Image   string `json:"image,omitempty"`
}

// NewsItem is exported for unified search
type NewsItem struct {
	Title   string
	Link    string
	PubDate int64
	Source  string
	BgColor string
	Image   string
}

const newsCacheTTL = 2 * time.Minute

var (
	newsMu          sync.Mutex
	newsItemsCached []newsItem
	newsCachedAt    time.Time
	newsInflight    bool
	newsWaiters     []chan struct{}
	// Replaced in tests so search can be proven not to wait on feeds.
	loadFeedItems = fetchAllFeedItems
)

func fetchAllFeedItems() []newsItem {
	rows, err := db.DB.Query("SELECT id, title, feed_url, COALESCE(bg_color, '#ffebd6') FROM news_feeds ORDER BY LOWER(title) ASC, id ASC")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var feeds []models.NewsFeed
	for rows.Next() {
		var nf models.NewsFeed
		if err := rows.Scan(&nf.ID, &nf.Title, &nf.FeedURL, &nf.BgColor); err == nil {
			feeds = append(feeds, nf)
		}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var allItems []newsItem

	for _, feed := range feeds {
		wg.Add(1)
		go func(f models.NewsFeed) {
			defer wg.Done()
			items := fetchAndParseFeed(f, 10)
			if len(items) > 0 {
				mu.Lock()
				allItems = append(allItems, items...)
				mu.Unlock()
			}
		}(feed)
	}
	wg.Wait()

	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].PubDate > allItems[j].PubDate
	})
	return allItems
}

func parsePubDate(s string) int64 {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UnixMilli()
		}
	}
	return 0
}

func extractImage(item rssItem) string {
	if item.Enclosure.URL != "" && strings.HasPrefix(item.Enclosure.Type, "image") {
		return item.Enclosure.URL
	}
	if item.Content.URL != "" {
		return item.Content.URL
	}
	if item.Thumbnail.URL != "" {
		return item.Thumbnail.URL
	}
	if m := imgSrcRe.FindStringSubmatch(item.Description); len(m) > 1 {
		return m[1]
	}
	return ""
}

func fetchAndParseFeed(feed models.NewsFeed, maxItems int) []newsItem {
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(feed.FeedURL)
	if err != nil {
		log.Printf("RSS fetch error for %s: %v", feed.FeedURL, err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		log.Printf("RSS read error for %s: %v", feed.FeedURL, err)
		return nil
	}

	var rss rssRoot
	if err := xml.Unmarshal(body, &rss); err != nil {
		log.Printf("RSS parse error for %s: %v", feed.FeedURL, err)
		return nil
	}

	items := rss.Channel.Items
	if len(items) > maxItems {
		items = items[:maxItems]
	}

	result := make([]newsItem, 0, len(items))
	for _, item := range items {
		result = append(result, newsItem{
			Title:   item.Title,
			Link:    item.Link,
			PubDate: parsePubDate(item.PubDate),
			Source:  feed.Title,
			BgColor: feed.BgColor,
			Image:   extractImage(item),
		})
	}
	return result
}

func HandleAdminNewsFeeds(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		rows, err := db.DB.Query("SELECT id, title, feed_url, COALESCE(bg_color, '#ffebd6') FROM news_feeds ORDER BY LOWER(title) ASC, id ASC")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		var res []models.NewsFeed
		for rows.Next() {
			var nf models.NewsFeed
			if err := rows.Scan(&nf.ID, &nf.Title, &nf.FeedURL, &nf.BgColor); err == nil {
				res = append(res, nf)
			}
		}
		if res == nil {
			res = []models.NewsFeed{}
		}
		json.NewEncoder(w).Encode(res)

	case "POST":
		var nf models.NewsFeed
		if err := json.NewDecoder(r.Body).Decode(&nf); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		bgColor := nf.BgColor
		if bgColor == "" {
			bgColor = "#ffebd6"
		}
		err := db.DB.QueryRow("INSERT INTO news_feeds (title, feed_url, bg_color) VALUES ($1, $2, $3) RETURNING id", nf.Title, nf.FeedURL, bgColor).Scan(&nf.ID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		nf.BgColor = bgColor
		json.NewEncoder(w).Encode(nf)

	case "PUT":
		var nf models.NewsFeed
		if err := json.NewDecoder(r.Body).Decode(&nf); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		_, err := db.DB.Exec("UPDATE news_feeds SET title=$1, feed_url=$2, bg_color=$3 WHERE id=$4",
			nf.Title, nf.FeedURL, nf.BgColor, nf.ID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusOK)

	case "DELETE":
		id := r.URL.Query().Get("id")
		if id != "" {
			db.DB.Exec("DELETE FROM news_feeds WHERE id = $1", id)
		}
		w.WriteHeader(http.StatusOK)
	}
}
