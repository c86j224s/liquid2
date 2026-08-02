package web

import (
	"context"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/sourcediagnostics"
	"github.com/c86j224s/liquid2/plasma/internal/sourceingest"
)

func (server *Server) renderFetchedURLIfBrowserCandidate(ctx context.Context, normalizedURL string, fetched fetchedURLSource) (fetchedURLSource, bool, error) {
	if !fetchedURLNeedsBrowserRender(fetched) {
		return fetched, false, nil
	}
	rawSHA := sha256Hex(fetched.Content)
	rendered, err := server.renderBrowserCandidateSource(ctx, normalizedURL, fetched.Title)
	if err != nil {
		return fetchedURLSource{}, true, err
	}
	rendered.RawFetchSHA256 = rawSHA
	return rendered, true, nil
}

func (server *Server) renderStagedURLCandidate(ctx context.Context, normalizedURL string, staged sourceingest.StagedSourceCandidate) (fetchedURLSource, error) {
	rawSHA := strings.TrimSpace(staged.Artifact.SHA256)
	if rawSHA == "" {
		rawSHA = sha256Hex(staged.Artifact.Content)
	}
	rendered, err := server.renderBrowserCandidateSource(ctx, normalizedURL, staged.Title)
	if err != nil {
		return fetchedURLSource{}, err
	}
	rendered.RawFetchSHA256 = rawSHA
	rendered.RawFetchArtifactID = staged.Artifact.ArtifactID
	return rendered, nil
}

func (server *Server) renderBrowserCandidateSource(ctx context.Context, normalizedURL string, fallbackTitle string) (fetchedURLSource, error) {
	if server.renderBrowserURLSource == nil {
		return fetchedURLSource{}, browserRenderSourceError(app.ErrInvalidInput)
	}
	rendered, err := server.renderBrowserURLSource(ctx, normalizedURL)
	if err != nil {
		return fetchedURLSource{}, err
	}
	if strings.TrimSpace(rendered.Title) == "" {
		rendered.Title = strings.TrimSpace(fallbackTitle)
	}
	rendered.RetrievalMethod = sourceRetrievalMethodBrowserRender
	rendered.MediaType = sourceFirstNonEmpty(rendered.MediaType, "text/html; charset=utf-8")
	rendered.ByteSize = int64(len(rendered.Content))
	rendered.TextLengthKnown = true
	return rendered, nil
}

func fetchedURLNeedsBrowserRender(fetched fetchedURLSource) bool {
	diagnosis := sourcediagnostics.DiagnoseBrowserRenderCandidate(fetched.Content, fetched.MediaType)
	return diagnosis.Candidate
}

func stagedURLNeedsBrowserRender(staged sourceingest.StagedSourceCandidate) bool {
	diagnosis := sourcediagnostics.DiagnoseBrowserRenderCandidate(staged.Artifact.Content, staged.Artifact.MediaType)
	return diagnosis.Candidate
}

func sourceFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
