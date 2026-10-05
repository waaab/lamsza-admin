package directory

import "testing"

func TestNormalizeCategoryIDs(t *testing.T) {
	got, err := NormalizeCategoryIDs(71, []int{72, 71, 0, 73})
	if err != nil {
		t.Fatal(err)
	}
	want := []int{71, 72, 73}
	if len(got) != len(want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ids = %v, want %v", got, want)
		}
	}
}

func TestNormalizeCategoryIDsRejectsASixthLeaf(t *testing.T) {
	_, err := NormalizeCategoryIDs(1, []int{2, 3, 4, 5, 6})
	if err != ErrCategorySet {
		t.Fatalf("err = %v, want ErrCategorySet", err)
	}
}

func TestNormalizeCategoryIDsRequiresAPrimary(t *testing.T) {
	_, err := NormalizeCategoryIDs(0, []int{71})
	if err != ErrCategorySet {
		t.Fatalf("err = %v, want ErrCategorySet", err)
	}
}

func TestSearchNameJoinsEveryShelf(t *testing.T) {
	got := SearchName([]string{"Webfejlesztés", "Webdizájn", "E-kereskedelem"})
	want := "Webfejlesztés Webdizájn E-kereskedelem"
	if got != want {
		t.Fatalf("SearchName = %q, want %q", got, want)
	}
}
