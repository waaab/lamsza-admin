package utils

// DirectoryNode is one row in the two-level directory catalog seed.
type DirectoryNode struct {
	ID        int
	ParentID  *int
	Name      string
	SortOrder int
}

// DirectoryParents returns the main categories. Ids 1-10 are the original shelves.
func DirectoryParents() []DirectoryNode {
	return []DirectoryNode{
		{ID: 1, Name: "Étkezés", SortOrder: 1},
		{ID: 2, Name: "Szállás", SortOrder: 2},
		{ID: 3, Name: "Egészség és szépség", SortOrder: 3},
		{ID: 4, Name: "Vásárlás", SortOrder: 4},
		{ID: 5, Name: "Autó", SortOrder: 5},
		{ID: 6, Name: "Mesteremberek", SortOrder: 6},
		{ID: 7, Name: "Oktatás", SortOrder: 7},
		{ID: 8, Name: "Hivatalok", SortOrder: 8},
		{ID: 9, Name: "Sport és szabadidő", SortOrder: 9},
		{ID: 10, Name: "Pénzügy", SortOrder: 10},
		{ID: 69, Name: "Informatika és távközlés", SortOrder: 11},
		{ID: 70, Name: "Szakmai szolgáltatások", SortOrder: 12},
	}
}

// DirectoryChildren returns subcategories. Sportpálya is a venue type, not a category.
func DirectoryChildren() []DirectoryNode {
	p := func(id int) *int { return &id }
	return []DirectoryNode{
		{ID: 11, ParentID: p(1), Name: "Étterem", SortOrder: 1},
		{ID: 12, ParentID: p(1), Name: "Kávézó", SortOrder: 2},
		{ID: 13, ParentID: p(1), Name: "Cukrászda", SortOrder: 3},
		{ID: 14, ParentID: p(1), Name: "Pékség", SortOrder: 4},
		{ID: 15, ParentID: p(1), Name: "Söröző", SortOrder: 5},
		{ID: 16, ParentID: p(2), Name: "Szálloda", SortOrder: 1},
		{ID: 17, ParentID: p(2), Name: "Motel", SortOrder: 2},
		{ID: 18, ParentID: p(2), Name: "Panzió", SortOrder: 3},
		{ID: 19, ParentID: p(2), Name: "Apartman", SortOrder: 4},
		{ID: 20, ParentID: p(2), Name: "Kemping", SortOrder: 5},
		{ID: 21, ParentID: p(3), Name: "Orvos", SortOrder: 1},
		{ID: 22, ParentID: p(3), Name: "Fogászat", SortOrder: 2},
		{ID: 23, ParentID: p(3), Name: "Bőrgyógyászat", SortOrder: 3},
		{ID: 24, ParentID: p(3), Name: "Optika", SortOrder: 4},
		{ID: 25, ParentID: p(3), Name: "Csontkovács", SortOrder: 5},
		{ID: 26, ParentID: p(3), Name: "Lábgyógyászat", SortOrder: 6},
		{ID: 27, ParentID: p(3), Name: "Gyógytorna", SortOrder: 7},
		{ID: 28, ParentID: p(3), Name: "Masszázs", SortOrder: 8},
		{ID: 29, ParentID: p(3), Name: "Gyógyszertár", SortOrder: 9},
		{ID: 30, ParentID: p(3), Name: "Kórház", SortOrder: 10},
		{ID: 31, ParentID: p(3), Name: "Állatorvos", SortOrder: 11},
		{ID: 32, ParentID: p(3), Name: "Fodrász", SortOrder: 12},
		{ID: 33, ParentID: p(3), Name: "Borbély", SortOrder: 13},
		{ID: 34, ParentID: p(3), Name: "Körömszalon", SortOrder: 14},
		{ID: 35, ParentID: p(3), Name: "Spa", SortOrder: 15},
		{ID: 36, ParentID: p(4), Name: "Élelmiszer", SortOrder: 1},
		{ID: 37, ParentID: p(4), Name: "Ruházat", SortOrder: 2},
		{ID: 38, ParentID: p(4), Name: "Műszaki bolt", SortOrder: 3},
		{ID: 39, ParentID: p(4), Name: "Bútor", SortOrder: 4},
		{ID: 40, ParentID: p(4), Name: "Piac", SortOrder: 5},
		{ID: 41, ParentID: p(5), Name: "Autószerviz", SortOrder: 1},
		{ID: 42, ParentID: p(5), Name: "Karosszéria", SortOrder: 2},
		{ID: 43, ParentID: p(5), Name: "Olajcsere", SortOrder: 3},
		{ID: 44, ParentID: p(5), Name: "Gumiszerviz", SortOrder: 4},
		{ID: 45, ParentID: p(5), Name: "Turbószerviz", SortOrder: 5},
		{ID: 46, ParentID: p(5), Name: "Autómentés", SortOrder: 6},
		{ID: 47, ParentID: p(5), Name: "Autómosó", SortOrder: 7},
		{ID: 48, ParentID: p(5), Name: "Autókozmetika", SortOrder: 8},
		{ID: 49, ParentID: p(5), Name: "Parkoló", SortOrder: 9},
		{ID: 50, ParentID: p(5), Name: "Autókereskedés", SortOrder: 10},
		{ID: 51, ParentID: p(5), Name: "Autóbontó", SortOrder: 11},
		{ID: 52, ParentID: p(5), Name: "Autóalkatrész", SortOrder: 12},
		{ID: 53, ParentID: p(5), Name: "Benzinkút", SortOrder: 13},
		{ID: 54, ParentID: p(6), Name: "Villanyszerelő", SortOrder: 1},
		{ID: 55, ParentID: p(6), Name: "Vízvezeték-szerelő", SortOrder: 2},
		{ID: 56, ParentID: p(6), Name: "Asztalos", SortOrder: 3},
		{ID: 57, ParentID: p(6), Name: "Takarítás", SortOrder: 4},
		{ID: 58, ParentID: p(6), Name: "Építkezés", SortOrder: 5},
		{ID: 82, ParentID: p(6), Name: "Szabó", SortOrder: 6},
		{ID: 83, ParentID: p(6), Name: "Építész", SortOrder: 7},
		{ID: 84, ParentID: p(6), Name: "Lakberendezés", SortOrder: 8},
		{ID: 59, ParentID: p(7), Name: "Óvoda", SortOrder: 1},
		{ID: 60, ParentID: p(7), Name: "Iskola", SortOrder: 2},
		{ID: 61, ParentID: p(7), Name: "Egyetem", SortOrder: 3},
		{ID: 62, ParentID: p(8), Name: "Polgármesteri hivatal", SortOrder: 1},
		{ID: 63, ParentID: p(8), Name: "Megyei intézmény", SortOrder: 2},
		{ID: 64, ParentID: p(8), Name: "Posta", SortOrder: 3},
		{ID: 65, ParentID: p(9), Name: "Sportegyesület", SortOrder: 1},
		{ID: 67, ParentID: p(10), Name: "Bank", SortOrder: 1},
		{ID: 68, ParentID: p(10), Name: "Biztosító", SortOrder: 2},
		{ID: 80, ParentID: p(10), Name: "Könyvelő", SortOrder: 3},
		{ID: 81, ParentID: p(10), Name: "Pénzügyi tanácsadó", SortOrder: 4},
		{ID: 71, ParentID: p(69), Name: "Webfejlesztés", SortOrder: 1},
		{ID: 72, ParentID: p(69), Name: "Webdizájn", SortOrder: 2},
		{ID: 73, ParentID: p(69), Name: "Keresőoptimalizálás", SortOrder: 3},
		{ID: 74, ParentID: p(69), Name: "Szoftver", SortOrder: 4},
		{ID: 75, ParentID: p(69), Name: "Hálózat", SortOrder: 5},
		{ID: 76, ParentID: p(69), Name: "Számítógép szerviz", SortOrder: 6},
		{ID: 77, ParentID: p(69), Name: "Tárhely", SortOrder: 7},
		{ID: 78, ParentID: p(69), Name: "Internet", SortOrder: 8},
		{ID: 79, ParentID: p(69), Name: "Távközlés", SortOrder: 9},
		{ID: 95, ParentID: p(69), Name: "E-kereskedelem", SortOrder: 10},
		{ID: 85, ParentID: p(70), Name: "Ügyvéd", SortOrder: 1},
		{ID: 86, ParentID: p(70), Name: "Közjegyző", SortOrder: 2},
		{ID: 87, ParentID: p(70), Name: "Fordítóiroda", SortOrder: 3},
		{ID: 88, ParentID: p(70), Name: "Tanácsadás", SortOrder: 4},
		{ID: 89, ParentID: p(70), Name: "Marketing", SortOrder: 5},
		{ID: 90, ParentID: p(70), Name: "Grafika", SortOrder: 6},
		{ID: 91, ParentID: p(70), Name: "Toborzás", SortOrder: 7},
		{ID: 92, ParentID: p(70), Name: "Nyomda", SortOrder: 8},
		{ID: 93, ParentID: p(70), Name: "Ingatlanközvetítő", SortOrder: 9},
		{ID: 94, ParentID: p(70), Name: "Fotós", SortOrder: 10},
	}
}

// DirectoryTypes returns the three closed entry types (ids 1-3).
func DirectoryTypes() []DirectoryNode {
	return []DirectoryNode{
		{ID: 1, Name: "Személy"},
		{ID: 2, Name: "Vállalkozás"},
		{ID: 3, Name: "Intézmény"},
	}
}
