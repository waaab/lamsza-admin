package utils

// DirectoryNode is one row in the two-level directory catalog seed.
type DirectoryNode struct {
	ID        int
	ParentID  *int
	Name      string
	SortOrder int
}
