package app

// Tag labels documents across folders.
type Tag struct {
	// ID is the stable tag identifier.
	ID string `json:"id"`
	// Name is the user-visible tag name.
	Name string `json:"name"`
	// Slug is the normalized unique tag key.
	Slug string `json:"slug"`
	// CreatedAt is the creation timestamp in Unix milliseconds.
	CreatedAt int64 `json:"createdAt"`
}
