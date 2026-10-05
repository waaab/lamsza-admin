package utils

import (
	"fmt"
	"strings"
	"unicode"
)

func Slugify(s string) string {
	s = strings.ToLower(s)
	// Latin diacritics → ASCII (HU / RO / DE); then keep only letters/digits, collapse separators to '-'.
	replacer := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ö", "o", "ő", "o", "ú", "u", "ü", "u", "ű", "u",
		"ă", "a", "â", "a", "î", "i", "ș", "s", "ț", "t",
		"ş", "s", "ţ", "t",
		"ä", "a", "ß", "ss",
	)
	s = replacer.Replace(s)

	var res strings.Builder
	lastDash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			res.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			res.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(res.String(), "-")
}

func JavaToPg(i int) string {
	return fmt.Sprintf("%d", i)
}
