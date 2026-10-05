package webdomain

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

var tagRE = regexp.MustCompile(`<[^>]*>`)

var multiSuffix = map[string]bool{
	"co.uk":  true,
	"org.uk": true,
	"com.au": true,
}

func CanonicalDomain(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", errors.New("empty domain")
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return "", errors.New("invalid domain")
	}
	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")
	labels := strings.Split(host, ".")
	if len(labels) < 2 || labels[0] == "" || strings.Contains(host, " ") {
		return "", errors.New("invalid domain")
	}
	suffix := labels[len(labels)-2] + "." + labels[len(labels)-1]
	if multiSuffix[suffix] {
		if len(labels) < 3 {
			return "", errors.New("invalid domain")
		}
		return strings.Join(labels[len(labels)-3:], "."), nil
	}
	return strings.Join(labels[len(labels)-2:], "."), nil
}

func Plain(raw string, maxRunes int) (string, error) {
	s := strings.TrimSpace(tagRE.ReplaceAllString(raw, ""))
	if s == "" {
		return "", errors.New("empty")
	}
	if utf8.RuneCountInString(s) > maxRunes {
		return "", errors.New("too long")
	}
	return s, nil
}
