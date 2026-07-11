package translation

import "context"

// Provider translates document content without knowing storage or job details.
type Provider interface {
	Translate(context.Context, Request) (Result, error)
}

// Request contains the runtime translation input loaded by the worker.
type Request struct {
	DocumentID      string
	SourceContentID string
	SourceLanguage  string
	TargetLanguage  string
	Format          string
	Text            string
}

// Result contains translated content returned by a provider.
type Result struct {
	Content string
	Format  string
}
