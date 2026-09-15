package source

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// NormalizeConnector trims connector identity fields without changing their meaning.
func NormalizeConnector(connector ConnectorRef) ConnectorRef {
	return ConnectorRef{
		ConnectorID:      strings.TrimSpace(connector.ConnectorID),
		ConnectorType:    strings.TrimSpace(connector.ConnectorType),
		ExternalSourceID: strings.TrimSpace(connector.ExternalSourceID),
		ExternalURI:      strings.TrimSpace(connector.ExternalURI),
		ExternalVersion:  strings.TrimSpace(connector.ExternalVersion),
		ConnectorVersion: strings.TrimSpace(connector.ConnectorVersion),
	}
}

func defaultSourceAccess(access Access) Access {
	visibility := strings.TrimSpace(access.Visibility)
	if visibility == "" {
		visibility = "private"
	}
	license := strings.TrimSpace(access.License)
	if license == "" {
		license = "unknown"
	}
	retrievalPolicy := strings.TrimSpace(access.RetrievalPolicy)
	if retrievalPolicy == "" {
		retrievalPolicy = RetrievalPolicySnapshotOnly
	}
	return Access{
		Visibility:      visibility,
		License:         license,
		RetrievalPolicy: retrievalPolicy,
	}
}

func validateSourceRetrievalPolicy(policy string) error {
	switch strings.TrimSpace(policy) {
	case RetrievalPolicySnapshotOnly, RetrievalPolicyLiveReference:
		return nil
	default:
		return fmt.Errorf("%w: unsupported source retrieval policy", producterror.ErrInvalidInput)
	}
}

func validateLiveReference(req CreateRequest) error {
	connector := NormalizeConnector(req.Connector)
	switch connector.ConnectorType {
	case ConnectorTypeLocalPath:
		return validateLocalPathLiveReference(req)
	case ConnectorTypeMediaURL:
		return validateMediaURLLiveReference(req)
	default:
		return fmt.Errorf("%w: live_reference requires local_path or media_url connector", producterror.ErrInvalidInput)
	}
}

func validateLocalPathLiveReference(req CreateRequest) error {
	connector := NormalizeConnector(req.Connector)
	if connector.ConnectorType != ConnectorTypeLocalPath {
		return fmt.Errorf("%w: live_reference requires local_path connector", producterror.ErrInvalidInput)
	}
	if connector.ExternalURI != "" {
		return fmt.Errorf("%w: local_path live reference must not store absolute external uri", producterror.ErrInvalidInput)
	}
	locator, err := ParseLocalPathLocator(req.Locators)
	if err != nil {
		return err
	}
	if connector.ExternalSourceID != locator.RootID+":"+locator.RelativePath {
		return fmt.Errorf("%w: local_path external source id must be root_id:relative_path", producterror.ErrInvalidInput)
	}
	return nil
}

func validateMediaURLLiveReference(req CreateRequest) error {
	connector := NormalizeConnector(req.Connector)
	if connector.ConnectorType != ConnectorTypeMediaURL {
		return fmt.Errorf("%w: media live reference requires media_url connector", producterror.ErrInvalidInput)
	}
	if connector.ExternalURI == "" {
		return fmt.Errorf("%w: media live reference requires external uri", producterror.ErrInvalidInput)
	}
	locator, err := ParseMediaLocator(req.Locators)
	if err != nil {
		return err
	}
	if locator.MediaKind != MediaKindAudio && locator.MediaKind != MediaKindVideo {
		return fmt.Errorf("%w: only audio and video media live references are supported", producterror.ErrInvalidInput)
	}
	if strings.TrimSpace(locator.CanonicalURL) == "" && strings.TrimSpace(locator.DirectMediaURL) == "" {
		return fmt.Errorf("%w: media live reference requires a canonical or direct media URL", producterror.ErrInvalidInput)
	}
	return nil
}
