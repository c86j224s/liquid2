package confluence

type confluenceSpacesResponse struct {
	Results []confluenceSpaceResponse `json:"results"`
	Links   confluenceLinks           `json:"_links"`
}

type confluenceSpaceResponse struct {
	ID     string          `json:"id"`
	Key    string          `json:"key"`
	Name   string          `json:"name"`
	Type   string          `json:"type"`
	Status string          `json:"status"`
	Links  confluenceLinks `json:"_links"`
}

type confluencePageListResponse struct {
	Results []confluencePageSummaryResponse `json:"results"`
	Links   confluenceLinks                 `json:"_links"`
}

type confluencePageSummaryResponse struct {
	ID        string                `json:"id"`
	Status    string                `json:"status"`
	Title     string                `json:"title"`
	SpaceID   string                `json:"spaceId"`
	ParentID  string                `json:"parentId"`
	CreatedAt string                `json:"createdAt"`
	Version   confluencePageVersion `json:"version"`
	Links     confluenceLinks       `json:"_links"`
}
