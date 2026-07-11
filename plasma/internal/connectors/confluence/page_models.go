package confluence

type confluencePageResponse struct {
	ID        string                `json:"id"`
	Status    string                `json:"status"`
	Title     string                `json:"title"`
	SpaceID   string                `json:"spaceId"`
	CreatedAt string                `json:"createdAt"`
	Version   confluencePageVersion `json:"version"`
	Body      confluencePageBody    `json:"body"`
	Links     confluenceLinks       `json:"_links"`
}

type confluencePageVersion struct {
	CreatedAt string `json:"createdAt"`
	Message   string `json:"message"`
	Number    int    `json:"number"`
	MinorEdit bool   `json:"minorEdit"`
	AuthorID  string `json:"authorId"`
}

type confluencePageBody struct {
	Storage confluenceBodyValue `json:"storage"`
}

type confluenceBodyValue struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

type confluencePageMetadata struct {
	CloudID string                 `json:"cloud_id"`
	SiteURL string                 `json:"site_url,omitempty"`
	Page    confluencePageMetaPage `json:"page"`
	Links   confluenceLinks        `json:"links"`
}

type confluencePageMetaPage struct {
	ID        string                `json:"id"`
	Status    string                `json:"status"`
	Title     string                `json:"title"`
	SpaceID   string                `json:"space_id,omitempty"`
	CreatedAt string                `json:"created_at,omitempty"`
	Version   confluencePageVersion `json:"version"`
}

func (response confluencePageResponse) metadata(cloudID string, siteURL string) confluencePageMetadata {
	return confluencePageMetadata{
		CloudID: cloudID,
		SiteURL: siteURL,
		Page: confluencePageMetaPage{
			ID:        response.ID,
			Status:    response.Status,
			Title:     response.Title,
			SpaceID:   response.SpaceID,
			CreatedAt: response.CreatedAt,
			Version:   response.Version,
		},
		Links: response.Links,
	}
}
