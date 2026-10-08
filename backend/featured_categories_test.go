package main

import (
	"backend/internal/db"
	"net/http"
	"testing"
)

func TestAdminFeaturedCategories(t *testing.T) {
	requireDB(t)
	const path = "/api/admin/entry_categories/featured"
	reset := func() {
		if _, err := db.DB.Exec(`UPDATE entry_categories SET featured_order = NULL WHERE featured_order IS NOT NULL`); err != nil {
			t.Fatal(err)
		}
	}
	reset()
	defer reset()

	top, leaf := topLevelCategoryID(t), leafCategoryID(t)

	// The list replaces the chips in order; a subcategory may be one.
	rr := asAdmin(t, http.MethodPut, path, map[string]interface{}{"ids": []int{leaf, top}})
	mustOK(t, rr, "set featured")
	got := decodeArray(t, asAdmin(t, http.MethodGet, path, nil), "featured list")
	if len(got) != 2 || idOf(t, got[0], "first") != leaf || idOf(t, got[1], "second") != top {
		t.Fatalf("featured order: got %v, want [%d %d]", got, leaf, top)
	}

	// A second PUT replaces the list, it does not add to it.
	mustOK(t, asAdmin(t, http.MethodPut, path, map[string]interface{}{"ids": []int{top}}), "replace featured")
	got = decodeArray(t, asAdmin(t, http.MethodGet, path, nil), "featured list after replace")
	if len(got) != 1 || idOf(t, got[0], "only") != top {
		t.Fatalf("after replace: got %v, want [%d]", got, top)
	}

	// Refused, and the list stays as it was.
	for name, ids := range map[string][]int{
		"seven":     {1, 2, 3, 4, 5, 6, 7},
		"twice":     {top, top},
		"not found": {top, 999999},
	} {
		mustStatus(t, asAdmin(t, http.MethodPut, path, map[string]interface{}{"ids": ids}), http.StatusBadRequest, name)
		got = decodeArray(t, asAdmin(t, http.MethodGet, path, nil), "featured list after "+name)
		if len(got) != 1 || idOf(t, got[0], "only") != top {
			t.Fatalf("%s changed the list: %v", name, got)
		}
	}

	// An empty list clears the chips.
	mustOK(t, asAdmin(t, http.MethodPut, path, map[string]interface{}{"ids": []int{}}), "clear featured")
	if got = decodeArray(t, asAdmin(t, http.MethodGet, path, nil), "featured list after clear"); len(got) != 0 {
		t.Fatalf("after clear: %v", got)
	}

	// Admins only.
	mustStatus(t, anon(t, http.MethodGet, path, nil), http.StatusUnauthorized, "anonymous read")
}
