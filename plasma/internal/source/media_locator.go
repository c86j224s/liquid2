package source

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// ParseMediaLocator accepts a media locator object or one-item array and normalizes it.
func ParseMediaLocator(raw json.RawMessage) (MediaLocator, error) {
	if len(raw) == 0 {
		return MediaLocator{}, fmt.Errorf("%w: media locator is required", producterror.ErrInvalidInput)
	}
	var locator MediaLocator
	if err := json.Unmarshal(raw, &locator); err == nil && locatorDiscriminator(locator.LocatorType, locator.Kind) != "" {
		return normalizeMediaLocator(locator)
	}
	var locators []MediaLocator
	if err := json.Unmarshal(raw, &locators); err != nil {
		return MediaLocator{}, fmt.Errorf("%w: media locator must be an object or one-item array", producterror.ErrInvalidInput)
	}
	if len(locators) != 1 {
		return MediaLocator{}, fmt.Errorf("%w: media locator must contain exactly one locator", producterror.ErrInvalidInput)
	}
	return normalizeMediaLocator(locators[0])
}

func normalizeMediaLocator(locator MediaLocator) (MediaLocator, error) {
	discriminator := locatorDiscriminator(locator.LocatorType, locator.Kind)
	locator.MediaKind = strings.TrimSpace(locator.MediaKind)
	if discriminator != LocatorTypeMedia {
		return MediaLocator{}, fmt.Errorf("%w: media locator kind is required", producterror.ErrInvalidInput)
	}
	locator.LocatorType = LocatorTypeMedia
	locator.Kind = ""
	switch locator.MediaKind {
	case MediaKindImage, MediaKindAudio, MediaKindVideo:
	default:
		return MediaLocator{}, fmt.Errorf("%w: unsupported media kind", producterror.ErrInvalidInput)
	}
	locator.Provider = strings.TrimSpace(locator.Provider)
	locator.CanonicalURL = strings.TrimSpace(locator.CanonicalURL)
	locator.SourcePageURL = strings.TrimSpace(locator.SourcePageURL)
	locator.DirectMediaURL = strings.TrimSpace(locator.DirectMediaURL)
	locator.MIMEType = strings.TrimSpace(locator.MIMEType)
	locator.Title = strings.TrimSpace(locator.Title)
	locator.Attribution = strings.TrimSpace(locator.Attribution)
	locator.License = strings.TrimSpace(locator.License)
	locator.SHA256 = strings.TrimSpace(locator.SHA256)
	locator.InspectionSupport = strings.TrimSpace(locator.InspectionSupport)
	return locator, nil
}
