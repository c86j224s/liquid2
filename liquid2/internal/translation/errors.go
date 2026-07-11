package translation

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidJobPayload        = errors.New("invalid translation job payload")
	ErrProviderFailed           = errors.New("translation provider failed")
	ErrTranslationAlreadyQueued = errors.New("translation already queued")
	ErrTranslationUnavailable   = errors.New("translation unavailable")
)

func invalidPayload(message string, causes ...error) error {
	return classifiedError(ErrInvalidJobPayload, message, causes...)
}

func providerFailed(message string, causes ...error) error {
	return classifiedError(ErrProviderFailed, message, causes...)
}

func classifiedError(kind error, message string, causes ...error) error {
	if len(causes) > 0 && causes[0] != nil {
		return fmt.Errorf("%w: %s: %w", kind, message, causes[0])
	}
	return fmt.Errorf("%w: %s", kind, message)
}
