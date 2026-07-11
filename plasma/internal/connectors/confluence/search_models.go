package confluence

import "encoding/json"

type confluenceSearchResponse struct {
	Results []confluenceSearchResult `json:"results"`
	Links   confluenceLinks          `json:"_links"`
}

type confluenceSearchResult struct {
	Content      confluenceSearchContent `json:"content"`
	Space        confluenceSearchSpace   `json:"space"`
	Title        string                  `json:"title"`
	Excerpt      string                  `json:"excerpt"`
	URL          string                  `json:"url"`
	LastModified string                  `json:"lastModified"`
}

type confluenceSearchContent struct {
	ID      string                  `json:"id"`
	Type    string                  `json:"type"`
	Status  string                  `json:"status"`
	Title   string                  `json:"title"`
	Space   confluenceSearchSpace   `json:"space"`
	Version confluenceSearchVersion `json:"version"`
	Links   confluenceLinks         `json:"_links"`
}

type confluenceSearchSpace struct {
	ID   json.RawMessage `json:"id"`
	Key  string          `json:"key"`
	Name string          `json:"name"`
}

type confluenceSearchVersion struct {
	When   string `json:"when"`
	Number int    `json:"number"`
}

type confluenceLinks struct {
	Base  string `json:"base"`
	WebUI string `json:"webui"`
	Next  string `json:"next"`
	Self  string `json:"self"`
}
