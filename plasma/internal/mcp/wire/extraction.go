package wire

type SourceExtractionOutput struct {
	Type               string `json:"type"`
	PageCount          int    `json:"page_count,omitempty"`
	TextLength         int    `json:"text_length,omitempty"`
	TextLengthKnown    bool   `json:"text_length_known"`
	SuggestedReadBytes int    `json:"suggested_read_bytes,omitempty"`
	MaxReadBytes       int    `json:"max_read_bytes,omitempty"`
}
