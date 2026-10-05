package account

import "testing"

func TestDiffSuggestionKeepsOnlyChanges(t *testing.T) {
	before := map[string]any{"name": "Régi", "phone": "111", "notes": "szöveg"}
	after := map[string]any{"name": "Régi", "phone": "", "notes": "szöveg"}
	got := diffSuggestion(before, after)
	if _, ok := got["name"]; ok {
		t.Fatal("unchanged name must be omitted")
	}
	if got["phone"] != "" {
		t.Fatalf("cleared phone: %v", got["phone"])
	}
}
