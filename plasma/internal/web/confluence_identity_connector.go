package web

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"net/url"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
)

type webConfluenceIdentityMappingConnector struct {
	delegate          confluencesource.ConfluenceSourceConnector
	snapshotCloudID   string
	snapshotSiteURL   string
	connectionCloudID string
}

// SearchConfluenceSources는 웹 및 에이전트 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (connector *webConfluenceIdentityMappingConnector) SearchConfluenceSources(ctx context.Context, req confluencesource.ConfluenceSourceSearchRequest) (confluencesource.ConfluenceSourceSearchResult, error) {
	req.CloudID = connector.mapRequestCloudID(req.CloudID)
	result, err := connector.delegate.SearchConfluenceSources(ctx, req)
	if err != nil {
		return confluencesource.ConfluenceSourceSearchResult{}, err
	}
	result.CloudID = connector.mapResponseCloudID(result.CloudID)
	for i := range result.Candidates {
		result.Candidates[i] = connector.mapCandidate(result.Candidates[i])
	}
	return result, nil
}

// ReadConfluenceSource는 웹 및 에이전트 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (connector *webConfluenceIdentityMappingConnector) ReadConfluenceSource(ctx context.Context, req confluencesource.ConfluenceSourceReadRequest) (confluencesource.ConfluenceSourcePage, error) {
	req.CloudID = connector.mapRequestCloudID(req.CloudID)
	page, err := connector.delegate.ReadConfluenceSource(ctx, req)
	if err != nil {
		return confluencesource.ConfluenceSourcePage{}, err
	}
	if err := connector.validateResponseSiteURL(page.SiteURL); err != nil {
		return confluencesource.ConfluenceSourcePage{}, err
	}
	return connector.mapPage(page), nil
}

// GetConfluenceSourceVersion는 웹 및 에이전트 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (connector *webConfluenceIdentityMappingConnector) GetConfluenceSourceVersion(ctx context.Context, req confluencesource.ConfluenceSourceReadRequest) (confluencesource.ConfluenceSourceVersion, error) {
	req.CloudID = connector.mapRequestCloudID(req.CloudID)
	if versionConnector, ok := connector.delegate.(confluencesource.ConfluenceSourceVersionConnector); ok {
		version, err := versionConnector.GetConfluenceSourceVersion(ctx, req)
		if err != nil {
			return confluencesource.ConfluenceSourceVersion{}, err
		}
		if err := connector.validateResponseSiteURL(version.SiteURL); err != nil {
			return confluencesource.ConfluenceSourceVersion{}, err
		}
		return connector.mapVersion(version), nil
	}
	page, err := connector.delegate.ReadConfluenceSource(ctx, req)
	if err != nil {
		return confluencesource.ConfluenceSourceVersion{}, err
	}
	if err := connector.validateResponseSiteURL(page.SiteURL); err != nil {
		return confluencesource.ConfluenceSourceVersion{}, err
	}
	return connector.mapVersion(confluencesource.ConfluenceSourceVersion{
		Connector: page.Connector,
		CloudID:   page.CloudID,
		SiteURL:   page.SiteURL,
		PageID:    page.PageID,
		SpaceID:   page.SpaceID,
		SpaceKey:  page.SpaceKey,
		Title:     page.Title,
		WebURL:    page.WebURL,
		Version:   page.Version,
		UpdatedAt: page.UpdatedAt,
	}), nil
}

func (connector *webConfluenceIdentityMappingConnector) validateResponseSiteURL(siteURL string) error {
	snapshotHost := webConfluenceURLHost(connector.snapshotSiteURL)
	responseHost := webConfluenceURLHost(siteURL)
	if snapshotHost == "" || responseHost == "" || snapshotHost == responseHost {
		return nil
	}
	return fmt.Errorf("%w: confluence response site does not match the snapshot site", producterror.ErrInvalidInput)
}

func (connector *webConfluenceIdentityMappingConnector) mapRequestCloudID(cloudID string) string {
	if strings.TrimSpace(cloudID) == connector.snapshotCloudID {
		return connector.connectionCloudID
	}
	return cloudID
}

func (connector *webConfluenceIdentityMappingConnector) mapResponseCloudID(cloudID string) string {
	if strings.TrimSpace(cloudID) == connector.connectionCloudID {
		return connector.snapshotCloudID
	}
	return cloudID
}

func (connector *webConfluenceIdentityMappingConnector) mapCandidate(candidate confluencesource.ConfluenceSourceCandidate) confluencesource.ConfluenceSourceCandidate {
	candidate.CloudID = connector.mapResponseCloudID(candidate.CloudID)
	candidate.Connector = connector.mapConnector(candidate.Connector, webConfluenceConnectorPageID(candidate.Connector))
	return candidate
}

func (connector *webConfluenceIdentityMappingConnector) mapPage(page confluencesource.ConfluenceSourcePage) confluencesource.ConfluenceSourcePage {
	page.CloudID = connector.mapResponseCloudID(page.CloudID)
	page.Connector = connector.mapConnector(page.Connector, page.PageID)
	page.Metadata = webConfluenceMapMetadataCloudID(page.Metadata, connector.connectionCloudID, connector.snapshotCloudID)
	return page
}

func (connector *webConfluenceIdentityMappingConnector) mapVersion(version confluencesource.ConfluenceSourceVersion) confluencesource.ConfluenceSourceVersion {
	version.CloudID = connector.mapResponseCloudID(version.CloudID)
	version.Connector = connector.mapConnector(version.Connector, version.PageID)
	return version
}

func (connector *webConfluenceIdentityMappingConnector) mapConnector(ref sourcecontract.ConnectorRef, pageID string) sourcecontract.ConnectorRef {
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return ref
	}
	ref.ExternalSourceID = confluencesource.ConfluenceExternalSourceID(connector.snapshotCloudID, pageID)
	ref.ExternalURI = confluencesource.ConfluenceExternalURI(connector.snapshotCloudID, pageID)
	return ref
}

func webConfluenceConnectorPageID(ref sourcecontract.ConnectorRef) string {
	externalID := strings.TrimSpace(ref.ExternalSourceID)
	if externalID != "" {
		parts := strings.Split(externalID, ":")
		if len(parts) >= 2 {
			return strings.Join(parts[1:], ":")
		}
	}
	parsed, err := url.Parse(strings.TrimSpace(ref.ExternalURI))
	if err != nil {
		return ""
	}
	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i := 0; i+1 < len(segments); i++ {
		if segments[i] == "pages" {
			return strings.TrimSpace(segments[i+1])
		}
	}
	return ""
}

func webConfluenceMapMetadataCloudID(raw json.RawMessage, from string, to string) json.RawMessage {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if len(raw) == 0 || from == "" || to == "" || from == to {
		return raw
	}
	var metadata map[string]any
	if json.Unmarshal(raw, &metadata) != nil {
		return raw
	}
	if value, ok := metadata["cloud_id"].(string); ok && strings.TrimSpace(value) == from {
		metadata["cloud_id"] = to
	}
	mapped, err := json.Marshal(metadata)
	if err != nil {
		return raw
	}
	return mapped
}
