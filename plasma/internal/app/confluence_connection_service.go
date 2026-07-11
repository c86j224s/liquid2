package app

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

type ConfluenceConnectionStore interface {
	UpsertConfluenceConnection(context.Context, ConfluenceConnection) error
	GetConfluenceConnection(context.Context, string) (ConfluenceConnection, error)
	ListConfluenceConnections(context.Context) ([]ConfluenceConnection, error)
	DeleteConfluenceConnection(context.Context, string) error
}

func (s *Service) UpsertConfluenceConnection(ctx context.Context, req UpsertConfluenceConnectionRequest) (ConfluenceConnection, error) {
	store, err := s.confluenceConnectionStore()
	if err != nil {
		return ConfluenceConnection{}, err
	}
	connection, err := normalizeConfluenceConnection(req)
	if err != nil {
		return ConfluenceConnection{}, err
	}
	now := time.Now().UTC()
	if connection.CreatedAt.IsZero() {
		connection.CreatedAt = now
	}
	connection.UpdatedAt = now
	if err := store.UpsertConfluenceConnection(ctx, connection); err != nil {
		return ConfluenceConnection{}, err
	}
	return connection, nil
}

func (s *Service) GetConfluenceConnection(ctx context.Context, connectionID string) (ConfluenceConnection, error) {
	store, err := s.confluenceConnectionStore()
	if err != nil {
		return ConfluenceConnection{}, err
	}
	trimmed := strings.TrimSpace(connectionID)
	if err := validateID("cnf_", trimmed); err != nil {
		return ConfluenceConnection{}, err
	}
	return store.GetConfluenceConnection(ctx, trimmed)
}

func (s *Service) ListConfluenceConnections(ctx context.Context) ([]ConfluenceConnection, error) {
	store, err := s.confluenceConnectionStore()
	if err != nil {
		return nil, err
	}
	return store.ListConfluenceConnections(ctx)
}

func (s *Service) DeleteConfluenceConnection(ctx context.Context, connectionID string) error {
	store, err := s.confluenceConnectionStore()
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(connectionID)
	if err := validateID("cnf_", trimmed); err != nil {
		return err
	}
	return store.DeleteConfluenceConnection(ctx, trimmed)
}

func (s *Service) RenameConfluenceConnection(ctx context.Context, connectionID string, displayName string) (ConfluenceConnection, error) {
	connection, err := s.GetConfluenceConnection(ctx, connectionID)
	if err != nil {
		return ConfluenceConnection{}, err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return ConfluenceConnection{}, fmt.Errorf("%w: confluence display name is required", ErrInvalidInput)
	}
	connection.DisplayName = displayName
	connection.UpdatedAt = time.Now().UTC()
	store, err := s.confluenceConnectionStore()
	if err != nil {
		return ConfluenceConnection{}, err
	}
	if err := store.UpsertConfluenceConnection(ctx, connection); err != nil {
		return ConfluenceConnection{}, err
	}
	return connection, nil
}

func (s *Service) RevokeConfluenceConnection(ctx context.Context, connectionID string) (ConfluenceConnection, error) {
	connection, err := s.GetConfluenceConnection(ctx, connectionID)
	if err != nil {
		return ConfluenceConnection{}, err
	}
	connection.AccessToken = ""
	connection.RefreshToken = ""
	connection.TokenExpiresAt = time.Time{}
	connection.Revoked = true
	connection.UpdatedAt = time.Now().UTC()
	store, err := s.confluenceConnectionStore()
	if err != nil {
		return ConfluenceConnection{}, err
	}
	if err := store.UpsertConfluenceConnection(ctx, connection); err != nil {
		return ConfluenceConnection{}, err
	}
	return connection, nil
}

func (s *Service) RefreshConfluenceConnectionSites(
	ctx context.Context,
	connectionID string,
	lister ConfluenceSiteLister,
) (ConfluenceConnection, error) {
	if lister == nil {
		return ConfluenceConnection{}, fmt.Errorf("%w: confluence site lister is required", ErrInvalidInput)
	}
	connection, err := s.GetConfluenceConnection(ctx, connectionID)
	if err != nil {
		return ConfluenceConnection{}, err
	}
	result, err := lister.ListConfluenceSites(ctx)
	if err != nil {
		return ConfluenceConnection{}, err
	}
	sites, err := normalizeConfluenceSites(result.Sites, connection.AuthType)
	if err != nil {
		return ConfluenceConnection{}, err
	}
	connection.Sites = sites
	connection.UpdatedAt = time.Now().UTC()
	store, err := s.confluenceConnectionStore()
	if err != nil {
		return ConfluenceConnection{}, err
	}
	if err := store.UpsertConfluenceConnection(ctx, connection); err != nil {
		return ConfluenceConnection{}, err
	}
	return connection, nil
}

func (s *Service) confluenceConnectionStore() (ConfluenceConnectionStore, error) {
	store, ok := s.store.(ConfluenceConnectionStore)
	if !ok {
		return nil, fmt.Errorf("%w: confluence connection store is required", ErrInvalidInput)
	}
	return store, nil
}

func normalizeConfluenceConnection(req UpsertConfluenceConnectionRequest) (ConfluenceConnection, error) {
	connectionID := strings.TrimSpace(req.ConnectionID)
	if err := validateID("cnf_", connectionID); err != nil {
		return ConfluenceConnection{}, err
	}
	authType := strings.TrimSpace(req.AuthType)
	switch authType {
	case ConfluenceAuthTypeOAuth, ConfluenceAuthTypeAPIToken:
	default:
		return ConfluenceConnection{}, fmt.Errorf("%w: unsupported confluence auth type", ErrInvalidInput)
	}
	accessToken := strings.TrimSpace(req.AccessToken)
	if accessToken == "" && !req.Revoked {
		return ConfluenceConnection{}, fmt.Errorf("%w: confluence access token is required", ErrInvalidInput)
	}
	if authType == ConfluenceAuthTypeAPIToken && strings.TrimSpace(req.AccountName) == "" && !req.Revoked {
		return ConfluenceConnection{}, fmt.Errorf("%w: confluence api token connections require account email", ErrInvalidInput)
	}
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = "Confluence"
		if req.AccountName != "" {
			displayName = strings.TrimSpace(req.AccountName)
		}
	}
	sites, err := normalizeConfluenceSites(req.Sites, authType)
	if err != nil {
		return ConfluenceConnection{}, err
	}
	return ConfluenceConnection{
		ConnectionID:   connectionID,
		DisplayName:    displayName,
		AuthType:       authType,
		AccountID:      strings.TrimSpace(req.AccountID),
		AccountName:    strings.TrimSpace(req.AccountName),
		AccessToken:    accessToken,
		RefreshToken:   strings.TrimSpace(req.RefreshToken),
		TokenExpiresAt: req.TokenExpiresAt.UTC(),
		Scopes:         normalizeStringSet(req.Scopes),
		Sites:          sites,
		Revoked:        req.Revoked,
	}, nil
}

func normalizeConfluenceSites(sites []ConfluenceSite, authType string) ([]ConfluenceSite, error) {
	normalized := make([]ConfluenceSite, 0, len(sites))
	seen := map[string]struct{}{}
	for _, site := range sites {
		cloudID := strings.TrimSpace(site.CloudID)
		siteURL := strings.TrimRight(strings.TrimSpace(site.URL), "/")
		if authType == ConfluenceAuthTypeAPIToken {
			var err error
			siteURL, err = NormalizeConfluenceAPITokenSiteURL(siteURL)
			if err != nil {
				return nil, err
			}
			derivedCloudID, err := ConfluenceAPITokenSiteCloudID(siteURL)
			if err != nil {
				return nil, err
			}
			if cloudID != "" && cloudID != derivedCloudID {
				return nil, fmt.Errorf("%w: confluence api token cloud id must match the site URL", ErrInvalidInput)
			}
			cloudID = derivedCloudID
		}
		if cloudID == "" {
			continue
		}
		if _, ok := seen[cloudID]; ok {
			continue
		}
		seen[cloudID] = struct{}{}
		name := strings.TrimSpace(site.Name)
		if name == "" && authType == ConfluenceAuthTypeAPIToken {
			if host, err := confluenceTrustedURLHost(siteURL); err == nil {
				name = host
			}
		}
		normalized = append(normalized, ConfluenceSite{
			CloudID: cloudID,
			Name:    name,
			URL:     siteURL,
			Scopes:  normalizeStringSet(site.Scopes),
		})
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].CloudID < normalized[j].CloudID
	})
	return normalized, nil
}

func NormalizeConfluenceAPITokenSiteURL(value string) (string, error) {
	normalized, err := normalizeConfluenceAPITokenTrustedURL(value, "confluence api token site URL")
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(normalized)
	if err != nil {
		return "", fmt.Errorf("%w: confluence api token site URL must be a valid HTTPS URL", ErrInvalidInput)
	}
	if parsed.Path != "" && parsed.Path != "/wiki" {
		return "", fmt.Errorf("%w: confluence api token site URL must be the Atlassian site root or /wiki URL", ErrInvalidInput)
	}
	return normalized, nil
}

func ConfluenceAPITokenSiteCloudID(siteURL string) (string, error) {
	normalized, err := NormalizeConfluenceAPITokenSiteURL(siteURL)
	if err != nil {
		return "", err
	}
	host, err := confluenceTrustedURLHost(normalized)
	if err != nil {
		return "", err
	}
	return "site_" + host, nil
}

func NormalizeConfluenceAPITokenAPIBaseURL(value string) (string, error) {
	return normalizeConfluenceAPITokenTrustedURL(value, "confluence api token API base URL")
}

func NormalizeConfluenceAPITokenAPIBaseURLForSite(value string, siteURL string) (string, error) {
	normalizedBaseURL, err := NormalizeConfluenceAPITokenAPIBaseURL(value)
	if err != nil {
		return "", err
	}
	normalizedSiteURL, err := NormalizeConfluenceAPITokenSiteURL(siteURL)
	if err != nil {
		return "", err
	}
	baseHost, err := confluenceTrustedURLHost(normalizedBaseURL)
	if err != nil {
		return "", err
	}
	siteHost, err := confluenceTrustedURLHost(normalizedSiteURL)
	if err != nil {
		return "", err
	}
	if baseHost != siteHost {
		return "", fmt.Errorf("%w: confluence api token API base URL host must match the selected site URL host", ErrInvalidInput)
	}
	return normalizedBaseURL, nil
}

func normalizeConfluenceAPITokenTrustedURL(value string, label string) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(value), "/")
	if trimmed == "" {
		return "", fmt.Errorf("%w: %s is required", ErrInvalidInput, label)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("%w: %s must be a valid HTTPS URL", ErrInvalidInput, label)
	}
	if parsed.Scheme != "https" {
		return "", fmt.Errorf("%w: %s must use HTTPS", ErrInvalidInput, label)
	}
	if parsed.User != nil {
		return "", fmt.Errorf("%w: %s must not include credentials", ErrInvalidInput, label)
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "atlassian.net" && !strings.HasSuffix(host, ".atlassian.net") {
		return "", fmt.Errorf("%w: %s must be an Atlassian Cloud atlassian.net host", ErrInvalidInput, label)
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String(), nil
}

func confluenceTrustedURLHost(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("%w: confluence URL host is required", ErrInvalidInput)
	}
	return strings.ToLower(parsed.Hostname()), nil
}

func normalizeStringSet(values []string) []string {
	seen := map[string]struct{}{}
	var normalized []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized
}
