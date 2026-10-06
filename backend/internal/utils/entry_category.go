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
