package handlers

const directoryCatalogV2Key = "directory_catalog_v2"

// Tables wiped by the catalog migration or emptied via CASCADE. entry_tags and
// entry_members use composite primary keys and have no serial sequence.
var directorySerialTables = []string{
	"entries",
	"tags",
	"entry_reviews",
	"entry_suggestions",
	"entry_types",
	"entry_categories",
}
