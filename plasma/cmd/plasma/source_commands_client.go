package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/confluenceaccess"
	confluenceconnector "github.com/c86j224s/liquid2/plasma/internal/connectors/confluence"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"strings"
)

func cliConfluenceDiscoveryClient(connection confluenceaccess.Connection, discoveryURL string, allowEndpointOverrides bool) (*confluenceconnector.DiscoveryClient, error) {
	if err := rejectConfluenceOAuthEndpointOverride(discoveryURL, "Confluence OAuth discovery URL", allowEndpointOverrides); err != nil {
		return nil, err
	}
	options := []confluenceconnector.DiscoveryOption{}
	if strings.TrimSpace(discoveryURL) != "" {
		options = append(options, confluenceconnector.WithDiscoveryBaseURL(discoveryURL))
	}
	switch connection.AuthType {
	case confluenceaccess.AuthOAuth:
		options = append(options, confluenceconnector.WithDiscoveryBearerToken(connection.AccessToken))
	default:
		return nil, fmt.Errorf("%w: confluence site discovery requires an oauth connection", app.ErrInvalidInput)
	}
	return confluenceconnector.NewDiscoveryClient(options...)
}

func cliConfluenceClient(ctx context.Context, svc *app.Service, connectionID string, cloudID string, apiBaseURL string, siteURL string, allowEndpointOverrides bool) (confluencesource.ConfluenceSourceConnector, error) {
	connection, err := svc.GetConfluenceConnection(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	if connection.Revoked {
		return nil, confluencesource.NewConfluenceValidationError(
			confluencesource.ConfluenceErrorCodeRevoked,
			"Confluence 연결이 로컬에서 해제되었습니다. 다시 연결하거나 다른 연결을 선택하세요.",
		)
	}
	cloudID = strings.TrimSpace(cloudID)
	if cloudID == "" {
		return nil, fmt.Errorf("%w: confluence cloud id is required", app.ErrInvalidInput)
	}
	baseURL := strings.TrimSpace(apiBaseURL)
	effectiveSiteURL := strings.TrimSpace(siteURL)
	if effectiveSiteURL == "" {
		effectiveSiteURL = cliConfluenceCachedSiteURL(connection, cloudID)
	}
	options := []confluenceconnector.Option{}
	switch connection.AuthType {
	case confluenceaccess.AuthOAuth:
		return nil, fmt.Errorf("%w: Confluence OAuth is disabled in Plasma 0.0. Use an API token connection instead.", app.ErrInvalidInput)
	case confluenceaccess.AuthAPIToken:
		if effectiveSiteURL == "" {
			return nil, fmt.Errorf("%w: confluence site URL is required for api_token connections", app.ErrInvalidInput)
		}
		normalizedSiteURL, err := confluenceaccess.NormalizeConfluenceAPITokenSiteURL(effectiveSiteURL)
		if err != nil {
			return nil, err
		}
		effectiveSiteURL = normalizedSiteURL
		if baseURL == "" {
			baseURL = cliConfluenceWikiBaseURL(effectiveSiteURL)
		} else {
			if err := rejectConfluenceOAuthEndpointOverride(baseURL, "Confluence API token API base URL", allowEndpointOverrides); err != nil {
				return nil, err
			}
			if allowEndpointOverrides {
				baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
			} else {
				normalizedBaseURL, err := confluenceaccess.NormalizeConfluenceAPITokenAPIBaseURLForSite(baseURL, effectiveSiteURL)
				if err != nil {
					return nil, err
				}
				baseURL = normalizedBaseURL
			}
		}
		options = append(options, confluenceconnector.WithSiteURL(effectiveSiteURL))
		options = append(options, confluenceconnector.WithBasicAuth(connection.AccountName, connection.AccessToken))
	default:
		return nil, fmt.Errorf("%w: unsupported confluence auth type", app.ErrInvalidInput)
	}
	return confluenceconnector.NewClient(baseURL, cloudID, options...)
}

func cliConfluenceBrowserConnector(ctx context.Context, svc *app.Service, connectionID string, cloudID string, apiBaseURL string, siteURL string, allowEndpointOverrides bool) (confluencesource.ConfluenceBrowserConnector, error) {
	connector, err := cliConfluenceClient(ctx, svc, connectionID, cloudID, apiBaseURL, siteURL, allowEndpointOverrides)
	if err != nil {
		return nil, err
	}
	browser, ok := connector.(confluencesource.ConfluenceBrowserConnector)
	if !ok {
		return nil, fmt.Errorf("%w: confluence browser connector is required", app.ErrInvalidInput)
	}
	return browser, nil
}

func cliConfluenceCachedSiteURL(connection confluenceaccess.Connection, cloudID string) string {
	cloudID = strings.TrimSpace(cloudID)
	for _, site := range connection.Sites {
		if strings.TrimSpace(site.CloudID) == cloudID {
			return strings.TrimSpace(site.URL)
		}
	}
	return ""
}

func cliConfluenceWikiBaseURL(siteURL string) string {
	siteURL = strings.TrimRight(strings.TrimSpace(siteURL), "/")
	if strings.HasSuffix(siteURL, "/wiki") {
		return siteURL
	}
	return siteURL + "/wiki"
}

func cliConfluenceCloudID(ctx context.Context, svc *app.Service, snapshotID string) (string, error) {
	snapshot, err := svc.GetSourceSnapshot(ctx, snapshotID)
	if err != nil {
		return "", err
	}
	var locators []struct {
		CloudID string `json:"cloud_id"`
	}
	if len(snapshot.Locators) > 0 && json.Unmarshal(snapshot.Locators, &locators) == nil {
		for _, locator := range locators {
			if cloudID := strings.TrimSpace(locator.CloudID); cloudID != "" {
				return cloudID, nil
			}
		}
	}
	parts := strings.Split(strings.TrimSpace(snapshot.Connector.ExternalSourceID), ":")
	if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" {
		return strings.TrimSpace(parts[0]), nil
	}
	return "", fmt.Errorf("%w: confluence cloud id is required", app.ErrInvalidInput)
}
