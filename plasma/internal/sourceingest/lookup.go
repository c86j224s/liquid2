package sourceingest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidateevents"
)

func ExistingSourceSnapshotForURL(ctx context.Context, store Store, missionID string, normalizedURL string) (SourceSnapshot, bool, error) {
	sources, err := store.ListSourceSnapshots(ctx, missionID)
	if err != nil {
		return SourceSnapshot{}, false, err
	}
	for _, source := range sources {
		for _, value := range []string{source.Connector.ExternalURI, source.Connector.ExternalSourceID} {
			existing, _, err := normalizeSourceIngestURL(value)
			if err != nil {
				continue
			}
			if existing == normalizedURL {
				return source, true, nil
			}
		}
		if confluenceSourceSnapshotMatchesURL(source, normalizedURL) {
			return source, true, nil
		}
	}
	return SourceSnapshot{}, false, nil
}

type confluenceSourceLocator struct {
	SiteURL string `json:"site_url"`
	PageID  string `json:"page_id"`
}

func confluenceSourceSnapshotMatchesURL(source SourceSnapshot, normalizedURL string) bool {
	candidate, ok := confluencePageURLKey(normalizedURL)
	if !ok {
		return false
	}
	var locators []confluenceSourceLocator
	if err := json.Unmarshal(source.Locators, &locators); err != nil {
		return false
	}
	for _, locator := range locators {
		key, ok := confluenceLocatorKey(locator.SiteURL, locator.PageID)
		if ok && key == candidate {
			return true
		}
	}
	return false
}

func confluencePageURLKey(rawURL string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Hostname() == "" {
		return "", false
	}
	segments := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	for i := 0; i < len(segments)-1; i++ {
		if strings.EqualFold(segments[i], "pages") {
			pageID, err := url.PathUnescape(segments[i+1])
			if err != nil {
				return "", false
			}
			return confluenceLocatorKey(parsed.Scheme+"://"+parsed.Host, pageID)
		}
	}
	return "", false
}

func confluenceLocatorKey(siteURL string, pageID string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(siteURL))
	if err != nil || parsed.Hostname() == "" {
		return "", false
	}
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return "", false
	}
	return strings.ToLower(parsed.Hostname()) + "\x00" + pageID, true
}

func ExistingSourceSnapshotForContentHash(ctx context.Context, store Store, missionID string, sha string) (SourceSnapshot, bool, error) {
	sha = strings.ToLower(strings.TrimSpace(sha))
	if sha == "" {
		return SourceSnapshot{}, false, nil
	}
	sources, err := store.ListSourceSnapshots(ctx, missionID)
	if err != nil {
		return SourceSnapshot{}, false, err
	}
	for _, source := range sources {
		if strings.EqualFold(strings.TrimSpace(source.ContentHash.Value), sha) {
			return source, true, nil
		}
	}
	return SourceSnapshot{}, false, nil
}

func LatestStagedSourceCandidateForURL(ctx context.Context, store Store, missionID string, normalizedURL string) (StagedSourceCandidate, bool, error) {
	events, err := store.ListEvents(ctx, missionID)
	if err != nil {
		return StagedSourceCandidate{}, false, err
	}
	selected, found := sourcecandidateevents.LatestStagedPayloadForURL(sourceCandidateEventsFromApp(events), normalizedURL, normalizeSourceCandidateURL)
	if !found {
		return StagedSourceCandidate{}, false, nil
	}
	artifact, err := store.GetRawArtifact(ctx, strings.TrimSpace(selected.ArtifactID))
	if err != nil {
		return StagedSourceCandidate{}, false, err
	}
	if artifact.MissionID != missionID {
		return StagedSourceCandidate{}, false, fmt.Errorf("%w: staged candidate artifact belongs to another mission", ErrInvalidInput)
	}
	updatedAt := time.Time{}
	if strings.TrimSpace(selected.ExternalUpdatedAt) != "" {
		updatedAt, _ = time.Parse(time.RFC3339Nano, strings.TrimSpace(selected.ExternalUpdatedAt))
	}
	return StagedSourceCandidate{
		URL:               normalizedURL,
		Title:             strings.TrimSpace(selected.Title),
		ProposalEventID:   strings.TrimSpace(selected.ProposalEventID),
		Artifact:          artifact,
		ExternalVersion:   strings.TrimSpace(selected.ExternalVersion),
		ExternalUpdatedAt: updatedAt,
	}, true, nil
}
