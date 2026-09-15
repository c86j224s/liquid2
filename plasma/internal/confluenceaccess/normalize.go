package confluenceaccess

import (
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"net/url"
	"sort"
	"strings"
)

func NormalizeConfluenceConnection(req UpsertRequest) (Connection, error) {
	connectionID := strings.TrimSpace(req.ConnectionID)
	if err := validateID("cnf_", connectionID); err != nil {
		return Connection{}, err
	}
	authType := strings.TrimSpace(req.AuthType)
	switch authType {
	case AuthOAuth, AuthAPIToken:
	default:
		return Connection{}, fmt.Errorf("%w: unsupported confluence auth type", producterror.ErrInvalidInput)
	}
	accessToken := strings.TrimSpace(req.AccessToken)
	if accessToken == "" && !req.Revoked {
		return Connection{}, fmt.Errorf("%w: confluence access token is required", producterror.ErrInvalidInput)
	}
	if authType == AuthAPIToken && strings.TrimSpace(req.AccountName) == "" && !req.Revoked {
		return Connection{}, fmt.Errorf("%w: confluence api token connections require account email", producterror.ErrInvalidInput)
	}
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = "Confluence"
		if req.AccountName != "" {
			displayName = strings.TrimSpace(req.AccountName)
		}
	}
	sites, err := NormalizeConfluenceSites(req.Sites, authType)
	if err != nil {
		return Connection{}, err
	}
	return Connection{
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

func NormalizeConfluenceSites(sites []Site, authType string) ([]Site, error) {
	normalized := make([]Site, 0, len(sites))
	seen := map[string]struct{}{}
	for _, site := range sites {
		cloudID := strings.TrimSpace(site.CloudID)
		siteURL := strings.TrimRight(strings.TrimSpace(site.URL), "/")
		if authType == AuthAPIToken {
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
				return nil, fmt.Errorf("%w: confluence api token cloud id must match the site URL", producterror.ErrInvalidInput)
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
		if name == "" && authType == AuthAPIToken {
			if host, err := confluenceTrustedURLHost(siteURL); err == nil {
				name = host
			}
		}
		normalized = append(normalized, Site{
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

// NormalizeConfluenceAPITokenSiteURL는 애플리케이션 서비스 계층 입력을 표준 형태로 정규화하고 허용되지 않는 값은 안정 오류로 거부한다.
func NormalizeConfluenceAPITokenSiteURL(value string) (string, error) {
	normalized, err := normalizeConfluenceAPITokenTrustedURL(value, "confluence api token site URL")
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(normalized)
	if err != nil {
		return "", fmt.Errorf("%w: confluence api token site URL must be a valid HTTPS URL", producterror.ErrInvalidInput)
	}
	if parsed.Path != "" && parsed.Path != "/wiki" {
		return "", fmt.Errorf("%w: confluence api token site URL must be the Atlassian site root or /wiki URL", producterror.ErrInvalidInput)
	}
	return normalized, nil
}

// ConfluenceAPITokenSiteCloudID는 API token 연결에 저장된 site cloud ID를 반환한다.
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

// NormalizeConfluenceAPITokenAPIBaseURL는 애플리케이션 서비스 계층 입력을 표준 형태로 정규화하고 허용되지 않는 값은 안정 오류로 거부한다.
func NormalizeConfluenceAPITokenAPIBaseURL(value string) (string, error) {
	return normalizeConfluenceAPITokenTrustedURL(value, "confluence api token API base URL")
}

// NormalizeConfluenceAPITokenAPIBaseURLForSite는 애플리케이션 서비스 계층 입력을 표준 형태로 정규화하고 허용되지 않는 값은 안정 오류로 거부한다.
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
		return "", fmt.Errorf("%w: confluence api token API base URL host must match the selected site URL host", producterror.ErrInvalidInput)
	}
	return normalizedBaseURL, nil
}

func normalizeConfluenceAPITokenTrustedURL(value string, label string) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(value), "/")
	if trimmed == "" {
		return "", fmt.Errorf("%w: %s is required", producterror.ErrInvalidInput, label)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("%w: %s must be a valid HTTPS URL", producterror.ErrInvalidInput, label)
	}
	if parsed.Scheme != "https" {
		return "", fmt.Errorf("%w: %s must use HTTPS", producterror.ErrInvalidInput, label)
	}
	if parsed.User != nil {
		return "", fmt.Errorf("%w: %s must not include credentials", producterror.ErrInvalidInput, label)
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "atlassian.net" && !strings.HasSuffix(host, ".atlassian.net") {
		return "", fmt.Errorf("%w: %s must be an Atlassian Cloud atlassian.net host", producterror.ErrInvalidInput, label)
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String(), nil
}

func confluenceTrustedURLHost(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("%w: confluence URL host is required", producterror.ErrInvalidInput)
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
func validateID(prefix, id string) error {
	trimmed := strings.TrimSpace(id)
	if !strings.HasPrefix(trimmed, prefix) || len(trimmed) <= len(prefix) {
		return fmt.Errorf("%w: id must start with %s", producterror.ErrInvalidInput, prefix)
	}
	return nil
}
