package utils

import "strings"

const (
	EntryCategoryEgeszsegugy    = "Egészségügy"
	EntryCategoryOktatas        = "Oktatás"
	EntryCategoryMesteremberek  = "Mesteremberek"
	EntryCategoryHivatalok      = "Hivatalok"
	EntryCategoryVendeglo       = "Vendéglő"
	EntryCategoryEtterem        = "Étterem"
	EntryCategoryKavezo         = "Kávézó"
	EntryCategoryCukraszda      = "Cukrászda"
	EntryCategoryPekseg         = "Pékség"
	EntryCategorySorozo         = "Söröző"
	EntryCategoryBolt           = "Bolt"
	EntryCategorySportegyesulet = "Sportegyesület"
	EntryCategoryEgyeb          = "Egyéb"
)

// SeedEntryCategories returns every name in the v2 directory catalog.
func SeedEntryCategories() []string {
	names := make([]string, 0, 94)
	for _, node := range DirectoryParents() {
		names = append(names, node.Name)
	}
	for _, node := range DirectoryChildren() {
		names = append(names, node.Name)
	}
	return names
}

// CategoryOffersDelivery is true for the Étkezés subcategories that take orders out.
func CategoryOffersDelivery(name string) bool {
	switch strings.TrimSpace(name) {
	case EntryCategoryEtterem, EntryCategoryKavezo, EntryCategoryCukraszda, EntryCategoryPekseg, EntryCategorySorozo:
		return true
	default:
		return false
	}
}

// CanonicalEntryCategory returns the stored category name.
func CanonicalEntryCategory(raw string) string {
	return strings.TrimSpace(raw)
}

// DefaultEntryCategory is used when an entry has no category.
func DefaultEntryCategory() string {
	return ""
}
