package search

import (
	"backend/internal/db"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const proxyMaxBytes = 8 << 20

var wikiThumb = regexp.MustCompile(`(?i)^(https?://upload\.wikimedia\.org/wikipedia/[^/]+/)thumb/(.+)/\d+px-[^/?#]+$`)

// ProxyHandler fetches a URL the site already stores (crest, attraction image, or news feed).
// It does not forward the caller's method, body, or headers, and it refuses private addresses.
func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	targetURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if targetURL == "" {
		http.Error(w, "Missing url parameter", http.StatusBadRequest)
		return
	}
	parsed, err := parsePublicProxyURL(targetURL)
	if err != nil {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}
	if ip := net.ParseIP(parsed.Hostname()); ip != nil && !publicIP(ip) {
		http.Error(w, "url not allowed", http.StatusForbidden)
		return
	}
	allowed, err := proxyTargetAllowed(parsed)
	if err != nil {
		http.Error(w, "proxy lookup failed", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "url not allowed", http.StatusForbidden)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, parsed.String(), nil)
	if err != nil {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}
	req.Header.Set("User-Agent", "LamszaProxy/1.2")
	req.Header.Set("Accept", "*/*")

	resp, err := proxyClient.Do(req)
	if err != nil {
		http.Error(w, "upstream failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "" {
		w.Header().Set("Cache-Control", cc)
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, io.LimitReader(resp.Body, proxyMaxBytes))
}

func parsePublicProxyURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return nil, fmt.Errorf("invalid url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("scheme")
	}
	if u.User != nil {
		return nil, fmt.Errorf("userinfo")
	}
	u.Fragment = ""
	host := strings.ToLower(u.Hostname())
	if port := u.Port(); port != "" && !((u.Scheme == "https" && port == "443") || (u.Scheme == "http" && port == "80")) {
		u.Host = net.JoinHostPort(host, port)
	} else {
		u.Host = host
	}
	return u, nil
}

func proxyTargetAllowed(target *url.URL) (bool, error) {
	allowed, err := loadProxyAllowlist()
	if err != nil {
		return false, err
	}
	_, ok := allowed[target.String()]
	return ok, nil
}

func loadProxyAllowlist() (map[string]struct{}, error) {
	rows, err := db.DB.Query(`
		SELECT crest FROM settlements WHERE btrim(COALESCE(crest, '')) <> ''
		UNION ALL
		SELECT featured_image FROM attractions WHERE btrim(COALESCE(featured_image, '')) <> ''
		UNION ALL
		SELECT url FROM attraction_images WHERE btrim(COALESCE(url, '')) <> ''
		UNION ALL
		SELECT feed_url FROM news_feeds WHERE btrim(COALESCE(feed_url, '')) <> ''
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	set := map[string]struct{}{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		addProxyKey(set, raw)
	}
	return set, rows.Err()
}

func addProxyKey(set map[string]struct{}, raw string) {
	parsed, err := parsePublicProxyURL(raw)
	if err != nil {
		return
	}
	set[parsed.String()] = struct{}{}
	upgraded := upgradePhotoURL(parsed.String())
	if upgraded == parsed.String() {
		return
	}
	again, err := parsePublicProxyURL(upgraded)
	if err != nil {
		return
	}
	set[again.String()] = struct{}{}
}

func upgradePhotoURL(raw string) string {
	if m := wikiThumb.FindStringSubmatch(raw); m != nil {
		return m[1] + m[2]
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	host := strings.ToLower(u.Hostname())
	if strings.Contains(strings.ToLower(raw), "tripadvisor.com/media/") || host == "dynamic-media-cdn.tripadvisor.com" {
		if !u.Query().Has("w") {
			q := u.Query()
			q.Set("w", "1600")
			q.Set("h", "-1")
			q.Set("s", "1")
			u.RawQuery = q.Encode()
			return u.String()
		}
	}
	return raw
}

func publicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	return true
}

var proxyClient = &http.Client{
	Timeout: 15 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		if req.URL == nil || req.URL.User != nil {
			return fmt.Errorf("redirect userinfo")
		}
		if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
			return fmt.Errorf("redirect scheme")
		}
		return nil
	},
	Transport: &http.Transport{
		DialContext: dialPublic,
	},
}

func dialPublic(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses")
	}
	for _, ip := range ips {
		if !publicIP(ip.IP) {
			return nil, fmt.Errorf("blocked address")
		}
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
}
