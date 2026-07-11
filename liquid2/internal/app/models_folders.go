package app

// Folder is a hierarchical container for documents.
type Folder struct {
	// ID is the stable folder identifier.
	ID string `json:"id"`
	// ParentID points to the parent folder when this folder is nested.
	ParentID *string `json:"parentId"`
	// Name is the folder display name.
	Name string `json:"name"`
	// SystemRole marks app-owned folders such as inbox, feeds, or trash.
	SystemRole *string `json:"systemRole,omitempty"`
	// SortOrder controls sibling ordering.
	SortOrder int `json:"sortOrder"`
	// CreatedAt is the creation timestamp in Unix milliseconds.
	CreatedAt int64 `json:"createdAt"`
	// UpdatedAt is the last update timestamp in Unix milliseconds.
	UpdatedAt int64 `json:"updatedAt"`
	// Children contains nested folders in tree responses.
	Children []Folder `json:"children"`
}

// FolderBreadcrumb is one display segment in a document folder path.
type FolderBreadcrumb struct {
	// ID is the stable folder identifier.
	ID string `json:"id"`
	// Name is the folder display name.
	Name string `json:"name"`
}
