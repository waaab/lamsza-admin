package account

import (
	"regexp"
)

var tagRE = regexp.MustCompile(`<[^>]*>`)
