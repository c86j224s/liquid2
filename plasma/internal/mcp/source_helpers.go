package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func validateClientRelativePath(relativePath string) error {
	trimmed := strings.TrimSpace(relativePath)
	if trimmed == "" {
		return nil
	}
	if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, `\`) || strings.HasPrefix(trimmed, "~") {
		return fmt.Errorf("%w: relative_path must be root-relative", app.ErrInvalidInput)
	}
	firstSegment := trimmed
	if slash := strings.IndexAny(firstSegment, `/\`); slash >= 0 {
		firstSegment = firstSegment[:slash]
	}
	if strings.Contains(firstSegment, ":") {
		return fmt.Errorf("%w: relative_path must be root-relative", app.ErrInvalidInput)
	}
	return nil
}

func normalizeConnectors(connectors []string) []string {
	normalized := make([]string, 0, len(connectors))
	seen := map[string]struct{}{}
	for _, connector := range connectors {
		trimmed := strings.TrimSpace(connector)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func includeRequested(includes []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, include := range includes {
		normalized := strings.TrimSpace(include)
		if normalized == target || normalized == "*" || normalized == "all" {
			return true
		}
	}
	return false
}

func sourceRetrievalPolicy(snapshot app.SourceSnapshot) string {
	policy := strings.TrimSpace(snapshot.Access.RetrievalPolicy)
	if policy == "" {
		return app.SourceRetrievalPolicySnapshotOnly
	}
	return policy
}

func mediaLocatorFromSnapshot(snapshot app.SourceSnapshot) (app.MediaLocator, error) {
	if len(snapshot.Locators) == 0 {
		return app.MediaLocator{}, fmt.Errorf("%w: media source locator is required", app.ErrInvalidInput)
	}
	var locator app.MediaLocator
	if err := json.Unmarshal(snapshot.Locators, &locator); err == nil && mcpLocatorType(locator.LocatorType, locator.Kind) != "" {
		return normalizeMCPMediaLocator(locator)
	}
	var locators []app.MediaLocator
	if err := json.Unmarshal(snapshot.Locators, &locators); err != nil {
		return app.MediaLocator{}, fmt.Errorf("%w: media source locator must be an object or array", app.ErrInvalidInput)
	}
	for _, locator := range locators {
		if mcpLocatorType(locator.LocatorType, locator.Kind) == app.SourceLocatorTypeMedia {
			return normalizeMCPMediaLocator(locator)
		}
	}
	return app.MediaLocator{}, fmt.Errorf("%w: media source locator is missing", app.ErrInvalidInput)
}

func normalizeMCPMediaLocator(locator app.MediaLocator) (app.MediaLocator, error) {
	discriminator := mcpLocatorType(locator.LocatorType, locator.Kind)
	locator.MediaKind = strings.TrimSpace(locator.MediaKind)
	if discriminator != app.SourceLocatorTypeMedia {
		return app.MediaLocator{}, fmt.Errorf("%w: media source locator kind is invalid", app.ErrInvalidInput)
	}
	locator.LocatorType = app.SourceLocatorTypeMedia
	locator.Kind = ""
	switch locator.MediaKind {
	case app.MediaKindImage, app.MediaKindAudio, app.MediaKindVideo:
		return locator, nil
	default:
		return app.MediaLocator{}, fmt.Errorf("%w: media source kind is unsupported", app.ErrInvalidInput)
	}
}

func mcpLocatorType(locatorType string, legacyKind string) string {
	if trimmed := strings.TrimSpace(locatorType); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(legacyKind)
}

func sourceState(snapshot app.SourceSnapshot) app.SourceState {
	state := snapshot.State
	if state.Removed || strings.TrimSpace(state.State) == app.SourceStateRemoved {
		state.State = app.SourceStateRemoved
		state.Removed = true
		return state
	}
	if strings.TrimSpace(state.State) == "" {
		state.State = app.SourceStateActive
	}
	return state
}

func observationEventID(event *app.LedgerEvent) string {
	if event == nil {
		return ""
	}
	return strings.TrimSpace(event.EventID)
}

func observationEventIDs(event *app.LedgerEvent) []string {
	eventID := observationEventID(event)
	if eventID == "" {
		return nil
	}
	return []string{eventID}
}

func selectedSnapshotArtifactID(snapshot app.SourceSnapshot, requested string) (string, error) {
	artifactID := strings.TrimSpace(requested)
	if artifactID == "" {
		if len(snapshot.ArtifactIDs) == 1 {
			return snapshot.ArtifactIDs[0], nil
		}
		return "", fmt.Errorf("%w: artifact_id is required when a source snapshot has multiple artifacts", app.ErrInvalidInput)
	}
	if err := validateID("art_", artifactID); err != nil {
		return "", err
	}
	for _, snapshotArtifactID := range snapshot.ArtifactIDs {
		if snapshotArtifactID == artifactID {
			return artifactID, nil
		}
	}
	return "", fmt.Errorf("%w: artifact_id is not part of the source snapshot", app.ErrInvalidInput)
}

func boundedArtifactContent(content []byte, offset int, maxBytes int) (string, int, int, bool, error) {
	if !utf8.Valid(content) {
		return "", 0, 0, false, fmt.Errorf("%w: source artifact is not UTF-8 text", app.ErrInvalidInput)
	}
	if offset < 0 {
		return "", 0, 0, false, fmt.Errorf("%w: source artifact offset must be non-negative", app.ErrInvalidInput)
	}
	if offset > len(content) {
		return "", 0, 0, false, fmt.Errorf("%w: source artifact offset is beyond content length", app.ErrInvalidInput)
	}
	if offset < len(content) && !utf8.RuneStart(content[offset]) {
		return "", 0, 0, false, fmt.Errorf("%w: source artifact offset must align to UTF-8 boundary", app.ErrInvalidInput)
	}
	limit := maxBytes
	if limit <= 0 {
		limit = 20000
	} else if limit > 50000 {
		limit = 50000
	}
	remaining := content[offset:]
	if len(remaining) <= limit {
		return string(remaining), offset, 0, false, nil
	}
	cut := offset + limit
	for cut > offset && !utf8.Valid(content[offset:cut]) {
		cut--
	}
	if cut == offset {
		return "", 0, 0, false, fmt.Errorf("%w: source artifact could not be sliced as UTF-8", app.ErrInvalidInput)
	}
	return string(content[offset:cut]), offset, cut, true, nil
}
