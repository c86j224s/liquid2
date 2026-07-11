package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type webConfluenceIdentityMappingConnector struct {
	delegate          app.ConfluenceSourceConnector
	snapshotCloudID   string
	snapshotSiteURL   string
	connectionCloudID string
}

func (connector *webConfluenceIdentityMappingConnector) SearchConfluenceSources(ctx context.Context, req app.ConfluenceSourceSearchRequest) (app.ConfluenceSourceSearchResult, error) {
	req.CloudID = connector.mapRequestCloudID(req.CloudID)
	result, err := connector.delegate.SearchConfluenceSources(ctx, req)
	if err != nil {
		return app.ConfluenceSourceSearchResult{}, err
	}
	result.CloudID = connector.mapResponseCloudID(result.CloudID)
	for i := range result.Candidates {
		result.Candidates[i] = connector.mapCandidate(result.Candidates[i])
	}
	return result, nil
}

func (connector *webConfluenceIdentityMappingConnector) ReadConfluenceSource(ctx context.Context, req app.ConfluenceSourceReadRequest) (app.ConfluenceSourcePage, error) {
	req.CloudID = connector.mapRequestCloudID(req.CloudID)
	page, err := connector.delegate.ReadConfluenceSource(ctx, req)
	if err != nil {
		return app.ConfluenceSourcePage{}, err
	}
	if err := connector.validateResponseSiteURL(page.SiteURL); err != nil {
		return app.ConfluenceSourcePage{}, err
	}
	return connector.mapPage(page), nil
}

func (connector *webConfluenceIdentityMappingConnector) GetConfluenceSourceVersion(ctx context.Context, req app.ConfluenceSourceReadRequest) (app.ConfluenceSourceVersion, error) {
	req.CloudID = connector.mapRequestCloudID(req.CloudID)
	if versionConnector, ok := connector.delegate.(app.ConfluenceSourceVersionConnector); ok {
		version, err := versionConnector.GetConfluenceSourceVersion(ctx, req)
		if err != nil {
			return app.ConfluenceSourceVersion{}, err
		}
		if err := connector.validateResponseSiteURL(version.SiteURL); err != nil {
			return app.ConfluenceSourceVersion{}, err
		}
		return connector.mapVersion(version), nil
	}
	page, err := connector.delegate.ReadConfluenceSource(ctx, req)
	if err != nil {
		return app.ConfluenceSourceVersion{}, err
	}
	if err := connector.validateResponseSiteURL(page.SiteURL); err != nil {
		return app.ConfluenceSourceVersion{}, err
	}
	return connector.mapVersion(app.ConfluenceSourceVersion{
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
	return fmt.Errorf("%w: confluence response site does not match the snapshot site", app.ErrInvalidInput)
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

func (connector *webConfluenceIdentityMappingConnector) mapCandidate(candidate app.ConfluenceSourceCandidate) app.ConfluenceSourceCandidate {
	candidate.CloudID = connector.mapResponseCloudID(candidate.CloudID)
	candidate.Connector = connector.mapConnector(candidate.Connector, webConfluenceConnectorPageID(candidate.Connector))
	return candidate
}

func (connector *webConfluenceIdentityMappingConnector) mapPage(page app.ConfluenceSourcePage) app.ConfluenceSourcePage {
	page.CloudID = connector.mapResponseCloudID(page.CloudID)
	page.Connector = connector.mapConnector(page.Connector, page.PageID)
	page.Metadata = webConfluenceMapMetadataCloudID(page.Metadata, connector.connectionCloudID, connector.snapshotCloudID)
	return page
}

func (connector *webConfluenceIdentityMappingConnector) mapVersion(version app.ConfluenceSourceVersion) app.ConfluenceSourceVersion {
	version.CloudID = connector.mapResponseCloudID(version.CloudID)
	version.Connector = connector.mapConnector(version.Connector, version.PageID)
	return version
}

func (connector *webConfluenceIdentityMappingConnector) mapConnector(ref app.ConnectorRef, pageID string) app.ConnectorRef {
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return ref
	}
	ref.ExternalSourceID = app.ConfluenceExternalSourceID(connector.snapshotCloudID, pageID)
	ref.ExternalURI = app.ConfluenceExternalURI(connector.snapshotCloudID, pageID)
	return ref
}

func webConfluenceConnectorPageID(ref app.ConnectorRef) string {
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
