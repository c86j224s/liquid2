package mcp

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func validateSourcesSnapshotInput(input sourcesSnapshotInput) error {
	if err := validateID("art_", input.ArtifactID); err != nil {
		return err
	}
	if err := validateID("src_", input.SnapshotID); err != nil {
		return err
	}
	if err := validateID("evt_", input.EventID); err != nil {
		return err
	}
	if strings.TrimSpace(input.Connector.ExternalSourceID) == "" {
		return fmt.Errorf("%w: connector external_source_id is required", app.ErrInvalidInput)
	}
	for _, contentRange := range input.Ranges {
		if strings.TrimSpace(contentRange.ContentID) == "" {
			return fmt.Errorf("%w: range content_id is required", app.ErrInvalidInput)
		}
		if contentRange.Start < 0 || contentRange.End < 0 || contentRange.End < contentRange.Start {
			return fmt.Errorf("%w: invalid source range", app.ErrInvalidInput)
		}
	}
	return nil
}
