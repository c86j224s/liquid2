package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type webConfluenceSnapshotSiteIdentity struct {
	CloudID string
	SiteURL string
}

func (server *Server) confluenceUpdateConnector(ctx context.Context, missionID string, connectionID string, snapshotID string) (app.ConfluenceSourceConnector, error) {
	snapshotSite, err := server.confluenceSnapshotSiteIdentity(ctx, missionID, snapshotID)
	if err != nil {
		return nil, err
	}
	connection, err := server.confluenceConnectionForUse(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	connectionCloudID, err := webConfluenceConnectionCloudIDForSnapshot(connection, snapshotSite)
	if err != nil {
		return nil, err
	}
	connector, err := server.confluenceClientForConnection(connection, connectionCloudID)
	if err != nil {
		return nil, err
	}
	if connectionCloudID == snapshotSite.CloudID {
		return connector, nil
	}
	return &webConfluenceIdentityMappingConnector{
		delegate:          connector,
		snapshotCloudID:   snapshotSite.CloudID,
		snapshotSiteURL:   snapshotSite.SiteURL,
		connectionCloudID: connectionCloudID,
	}, nil
}

func (server *Server) confluenceSnapshotSiteIdentity(ctx context.Context, missionID string, snapshotID string) (webConfluenceSnapshotSiteIdentity, error) {
	snapshot, err := server.service.GetSourceSnapshot(ctx, snapshotID)
	if err != nil {
		return webConfluenceSnapshotSiteIdentity{}, err
	}
	if strings.TrimSpace(snapshot.MissionID) != strings.TrimSpace(missionID) {
		return webConfluenceSnapshotSiteIdentity{}, fmt.Errorf("%w: confluence snapshot belongs to another mission", app.ErrInvalidInput)
	}
	if snapshot.Connector.ConnectorID != app.ConfluenceConnectorID ||
		snapshot.Connector.ConnectorType != app.ConfluenceConnectorType {
		return webConfluenceSnapshotSiteIdentity{}, fmt.Errorf("%w: confluence snapshot connector is required", app.ErrInvalidInput)
	}
	identity := webConfluenceSnapshotSiteIdentity{}
	var locators []struct {
		CloudID string `json:"cloud_id"`
		SiteURL string `json:"site_url"`
	}
	if len(snapshot.Locators) > 0 && json.Unmarshal(snapshot.Locators, &locators) == nil {
		for _, locator := range locators {
			cloudID := strings.TrimSpace(locator.CloudID)
			if cloudID == "" {
				continue
			}
			identity = webConfluenceSnapshotSiteIdentity{
				CloudID: cloudID,
				SiteURL: strings.TrimSpace(locator.SiteURL),
			}
			break
		}
	}
	if identity.CloudID == "" {
		parts := strings.Split(strings.TrimSpace(snapshot.Connector.ExternalSourceID), ":")
		if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" {
			identity.CloudID = strings.TrimSpace(parts[0])
		}
	}
	if identity.SiteURL == "" || identity.CloudID == "" {
		artifactIdentity := server.confluenceSnapshotArtifactSiteIdentity(ctx, snapshot)
		if identity.CloudID == "" {
			identity.CloudID = artifactIdentity.CloudID
		}
		if identity.SiteURL == "" {
			identity.SiteURL = artifactIdentity.SiteURL
		}
	}
	if identity.SiteURL == "" {
		identity.SiteURL = webConfluenceSyntheticSiteURL(identity.CloudID)
	}
	if identity.CloudID != "" {
		return identity, nil
	}
	return webConfluenceSnapshotSiteIdentity{}, fmt.Errorf("%w: confluence cloud id is required", app.ErrInvalidInput)
}

func (server *Server) confluenceSnapshotArtifactSiteIdentity(ctx context.Context, snapshot app.SourceSnapshot) webConfluenceSnapshotSiteIdentity {
	for _, artifactID := range snapshot.ArtifactIDs {
		artifact, err := server.service.GetRawArtifact(ctx, artifactID)
		if err != nil || artifact.MediaType != app.ConfluenceSnapshotMediaType {
			continue
		}
		var payload struct {
			Page struct {
				CloudID string `json:"cloud_id"`
				SiteURL string `json:"site_url"`
			} `json:"page"`
		}
		if json.Unmarshal(artifact.Content, &payload) != nil {
			continue
		}
		identity := webConfluenceSnapshotSiteIdentity{
			CloudID: strings.TrimSpace(payload.Page.CloudID),
			SiteURL: strings.TrimSpace(payload.Page.SiteURL),
		}
		if identity.CloudID != "" || identity.SiteURL != "" {
			return identity
		}
	}
	return webConfluenceSnapshotSiteIdentity{}
}

func webConfluenceConnectionCloudIDForSnapshot(connection app.ConfluenceConnection, snapshot webConfluenceSnapshotSiteIdentity) (string, error) {
	snapshot.CloudID = strings.TrimSpace(snapshot.CloudID)
	if snapshot.CloudID == "" {
		return "", fmt.Errorf("%w: confluence cloud id is required", app.ErrInvalidInput)
	}
	if webConfluenceCachedSiteURL(connection, snapshot.CloudID) != "" || len(connection.Sites) == 0 {
		return snapshot.CloudID, nil
	}
	snapshotHost := webConfluenceURLHost(snapshot.SiteURL)
	if snapshotHost != "" {
		for _, site := range connection.Sites {
			siteCloudID := strings.TrimSpace(site.CloudID)
			siteHost := webConfluenceURLHost(site.URL)
			if siteCloudID == "" || siteHost == "" || siteHost != snapshotHost {
				continue
			}
			if connection.AuthType == app.ConfluenceAuthTypeAPIToken &&
				!webConfluenceAPITokenCloudIDMatchesSiteURL(snapshot.CloudID, snapshot.SiteURL) &&
				webConfluenceAPITokenCloudIDMatchesSiteURL(siteCloudID, site.URL) {
				return siteCloudID, nil
			}
			if connection.AuthType == app.ConfluenceAuthTypeOAuth &&
				webConfluenceAPITokenCloudIDMatchesSiteURL(snapshot.CloudID, snapshot.SiteURL) &&
				webConfluenceOAuthDiscoveredSite(site) {
				return siteCloudID, nil
			}
		}
	}
	if snapshotHost == "" {
		return "", fmt.Errorf("%w: confluence snapshot site URL is required to use a different connection site", app.ErrInvalidInput)
	}
	return "", fmt.Errorf("%w: confluence snapshot site URL is not available in the selected connection", app.ErrInvalidInput)
}

func webConfluenceAPITokenCloudIDMatchesSiteURL(cloudID string, siteURL string) bool {
	derived, err := app.ConfluenceAPITokenSiteCloudID(siteURL)
	return err == nil && strings.TrimSpace(cloudID) == derived
}

func webConfluenceSyntheticSiteURL(cloudID string) string {
	host, ok := strings.CutPrefix(strings.TrimSpace(cloudID), "site_")
	if !ok || strings.TrimSpace(host) == "" {
		return ""
	}
	siteURL := "https://" + host + "/wiki"
	if webConfluenceAPITokenCloudIDMatchesSiteURL(cloudID, siteURL) {
		return siteURL
	}
	return ""
}

func webConfluenceOAuthDiscoveredSite(site app.ConfluenceSite) bool {
	return strings.TrimSpace(site.CloudID) != "" &&
		!webConfluenceAPITokenCloudIDMatchesSiteURL(site.CloudID, site.URL) &&
		len(site.Scopes) > 0
}

func webConfluenceURLHost(value string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}
