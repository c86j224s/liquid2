package web

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/confluenceaccess"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"net/url"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
)

type webConfluenceSnapshotSiteIdentity struct {
	CloudID string
	SiteURL string
}

func (server *Server) confluenceUpdateConnector(ctx context.Context, missionID string, connectionID string, snapshotID string) (confluencesource.ConfluenceSourceConnector, error) {
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
		return webConfluenceSnapshotSiteIdentity{}, fmt.Errorf("%w: confluence snapshot belongs to another mission", producterror.ErrInvalidInput)
	}
	if snapshot.Connector.ConnectorID != confluencesource.ConfluenceConnectorID ||
		snapshot.Connector.ConnectorType != confluencesource.ConfluenceConnectorType {
		return webConfluenceSnapshotSiteIdentity{}, fmt.Errorf("%w: confluence snapshot connector is required", producterror.ErrInvalidInput)
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
	return webConfluenceSnapshotSiteIdentity{}, fmt.Errorf("%w: confluence cloud id is required", producterror.ErrInvalidInput)
}

func (server *Server) confluenceSnapshotArtifactSiteIdentity(ctx context.Context, snapshot sourcecontract.Snapshot) webConfluenceSnapshotSiteIdentity {
	for _, artifactID := range snapshot.ArtifactIDs {
		artifact, err := server.service.GetRawArtifact(ctx, artifactID)
		if err != nil || artifact.MediaType != confluencesource.ConfluenceSnapshotMediaType {
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

func webConfluenceConnectionCloudIDForSnapshot(connection confluenceaccess.Connection, snapshot webConfluenceSnapshotSiteIdentity) (string, error) {
	snapshot.CloudID = strings.TrimSpace(snapshot.CloudID)
	if snapshot.CloudID == "" {
		return "", fmt.Errorf("%w: confluence cloud id is required", producterror.ErrInvalidInput)
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
			if connection.AuthType == confluenceaccess.AuthAPIToken &&
				!webConfluenceAPITokenCloudIDMatchesSiteURL(snapshot.CloudID, snapshot.SiteURL) &&
				webConfluenceAPITokenCloudIDMatchesSiteURL(siteCloudID, site.URL) {
				return siteCloudID, nil
			}
			if connection.AuthType == confluenceaccess.AuthOAuth &&
				webConfluenceAPITokenCloudIDMatchesSiteURL(snapshot.CloudID, snapshot.SiteURL) &&
				webConfluenceOAuthDiscoveredSite(site) {
				return siteCloudID, nil
			}
		}
	}
	if snapshotHost == "" {
		return "", fmt.Errorf("%w: confluence snapshot site URL is required to use a different connection site", producterror.ErrInvalidInput)
	}
	return "", fmt.Errorf("%w: confluence snapshot site URL is not available in the selected connection", producterror.ErrInvalidInput)
}

func webConfluenceAPITokenCloudIDMatchesSiteURL(cloudID string, siteURL string) bool {
	derived, err := confluenceaccess.ConfluenceAPITokenSiteCloudID(siteURL)
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

func webConfluenceOAuthDiscoveredSite(site confluenceaccess.Site) bool {
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
