package ingest

type Extractor interface {
	Extract(pageURL string, contentType string, data []byte) (ExtractedContent, error)
}
