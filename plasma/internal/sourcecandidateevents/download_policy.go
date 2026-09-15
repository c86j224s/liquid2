package sourcecandidateevents

import (
	"encoding/json"
	"fmt"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"net/url"
	"strings"
)

func snapshotHasDownloadArtifact(artifactIDs []string, requested string) bool {
	for _, artifactID := range artifactIDs {
		if strings.TrimSpace(artifactID) == requested {
			return true
		}
	}
	return false
}

func DownloadCandidateExcluded(normalizedURL string) bool {
	parsed, err := url.Parse(normalizedURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "atlassian.net" && !strings.HasSuffix(host, ".atlassian.net") {
		return false
	}
	_, confluence := confluenceCandidateKey(normalizedURL)
	return confluence
}

func LatestDownloadDecisionRejected(events []Event, normalizedURL string) bool {
	var latest Event
	found := false
	for _, event := range events {
		if event.EventType != "source.candidate.rejected" && event.EventType != "source.candidate.restored" {
			continue
		}
		var payload struct {
			URL string `json:"url"`
		}
		if json.Unmarshal(event.Payload, &payload) != nil {
			continue
		}
		candidateURL, err := NormalizeDownloadURL(payload.URL)
		if err != nil || candidateURL != normalizedURL {
			continue
		}
		if !found || event.Sequence >= latest.Sequence {
			latest, found = event, true
		}
	}
	return found && latest.EventType == "source.candidate.rejected"
}

func DownloadCandidateConsumed(events []Event, snapshots []sourcecontract.Snapshot, artifactID, normalizedURL string) bool {
	for _, snapshot := range snapshots {
		if snapshotHasDownloadArtifact(snapshot.ArtifactIDs, artifactID) || snapshotIdentityConsumesCandidate(snapshot, normalizedURL) {
			return true
		}
	}
	for _, event := range events {
		if event.EventType != "source.snapshotted" {
			continue
		}
		if sourceSnapshotConsumesCandidate(event.Payload, normalizedURL) {
			return true
		}
	}
	return false
}

func snapshotIdentityConsumesCandidate(snapshot sourcecontract.Snapshot, normalizedURL string) bool {
	for _, value := range []string{snapshot.Connector.ExternalURI, snapshot.Connector.ExternalSourceID} {
		candidateURL, err := NormalizeDownloadURL(value)
		if err == nil && candidateURL == normalizedURL {
			return true
		}
	}
	candidate, ok := confluenceCandidateKey(normalizedURL)
	if !ok {
		return false
	}
	var locators []struct {
		SiteURL string `json:"site_url"`
		PageID  string `json:"page_id"`
	}
	if json.Unmarshal(snapshot.Locators, &locators) != nil {
		return false
	}
	for _, locator := range locators {
		if site, ok := confluenceLocatorKey(locator.SiteURL, locator.PageID); ok && site == candidate {
			return true
		}
	}
	return false
}

func confluenceCandidateKey(raw string) (string, bool) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return "", false
	}
	if pageID := parsed.Query().Get("pageId"); isConfluencePageID(pageID) {
		return confluenceLocatorKey(parsed.Scheme+"://"+parsed.Host, pageID)
	}
	segments := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	for i := 0; i+1 < len(segments); i++ {
		marker, err := url.PathUnescape(segments[i])
		if err != nil {
			continue
		}
		if strings.EqualFold(marker, "pages") || strings.EqualFold(marker, "edit-v2") {
			pageID, err := url.PathUnescape(segments[i+1])
			if err != nil || !isConfluencePageID(pageID) {
				return "", false
			}
			return confluenceLocatorKey(parsed.Scheme+"://"+parsed.Host, pageID)
		}
	}
	return "", false
}

func isConfluencePageID(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	for _, r := range strings.TrimSpace(value) {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func confluenceLocatorKey(siteURL, pageID string) (string, bool) {
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

// sourceSnapshotConsumesCandidate uses candidate-specific URL identity; proposal provenance alone is not a candidate identity.
func sourceSnapshotConsumesCandidate(raw json.RawMessage, normalizedURL string) bool {
	var payload struct {
		ProposalEventID string `json:"source_candidate_proposal_event_id"`
		URL             string `json:"url"`
		Connector       struct {
			ConnectorID      string `json:"connector_id"`
			ConnectorType    string `json:"connector_type"`
			ExternalURI      string `json:"external_uri"`
			ExternalSourceID string `json:"external_source_id"`
		} `json:"connector"`
	}
	if json.Unmarshal(raw, &payload) != nil {
		return false
	}
	for _, value := range []string{payload.URL, payload.Connector.ExternalURI, payload.Connector.ExternalSourceID} {
		candidateURL, err := NormalizeDownloadURL(value)
		if err == nil && candidateURL == normalizedURL {
			return true
		}
	}
	candidateKey, candidateOK := confluenceCandidateKey(normalizedURL)
	legacyKey, legacyOK := legacyConfluenceExternalSourceKey(
		payload.Connector.ConnectorID,
		payload.Connector.ConnectorType,
		payload.Connector.ExternalSourceID,
	)
	return candidateOK && legacyOK && candidateKey == legacyKey
}

func legacyConfluenceExternalSourceKey(connectorID, connectorType, externalSourceID string) (string, bool) {
	connectorID = strings.ToLower(strings.TrimSpace(connectorID))
	connectorType = strings.ToLower(strings.TrimSpace(connectorType))
	if connectorID != confluencesource.ConfluenceConnectorID && connectorType != confluencesource.ConfluenceConnectorType {
		return "", false
	}
	externalSourceID = strings.TrimSpace(externalSourceID)
	if !strings.HasPrefix(strings.ToLower(externalSourceID), "site_") {
		return "", false
	}
	separator := strings.Index(externalSourceID[5:], ":")
	if separator < 1 {
		return "", false
	}
	separator += 5
	host := strings.ToLower(strings.TrimSpace(externalSourceID[5:separator]))
	pageID := strings.TrimSpace(externalSourceID[separator+1:])
	if host == "" || strings.ContainsAny(host, "/\\") || !isConfluencePageID(pageID) {
		return "", false
	}
	return confluenceLocatorKey("https://"+host, pageID)
}

func NormalizeDownloadURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return "", fmt.Errorf("invalid candidate URL")
	}
	if !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https") {
		return "", fmt.Errorf("invalid candidate URL")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Fragment = ""
	return parsed.String(), nil
}
