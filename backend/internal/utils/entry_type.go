package utils

import "strings"

const (
	EntryTypeSzemely     = "Személy"
	EntryTypeVallalkozas = "Vállalkozás"
	EntryTypeIntezmeny   = "Intézmény"
)

// CanonicalEntryType maps stored type strings onto catalog labels.
func CanonicalEntryType(raw string) string {
	trimmed := strings.TrimSpace(raw)
	switch trimmed {
	case EntryTypeSzemely, EntryTypeVallalkozas, EntryTypeIntezmeny:
		return trimmed
	default:
		return ""
	}
}

// DefaultEntryType is used when admin creates/updates an entry with an empty type.
func DefaultEntryType() string {
	return ""
}
