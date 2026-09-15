package source

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
)

func validateClientRelativePath(relativePath string) error {
	trimmed := strings.TrimSpace(relativePath)
	if trimmed == "" {
		return nil
	}
	if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, `\`) || strings.HasPrefix(trimmed, "~") {
		return fmt.Errorf("%w: relative_path must be root-relative", producterror.ErrInvalidInput)
	}
	firstSegment := trimmed
	if slash := strings.IndexAny(firstSegment, `/\`); slash >= 0 {
		firstSegment = firstSegment[:slash]
	}
	if strings.Contains(firstSegment, ":") {
		return fmt.Errorf("%w: relative_path must be root-relative", producterror.ErrInvalidInput)
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

func sourceRetrievalPolicy(snapshot sourcecontract.Snapshot) string {
	policy := strings.TrimSpace(snapshot.Access.RetrievalPolicy)
	if policy == "" {
		return sourcecontract.RetrievalPolicySnapshotOnly
	}
	return policy
}

func mediaLocatorFromSnapshot(snapshot sourcecontract.Snapshot) (sourcecontract.MediaLocator, error) {
	if len(snapshot.Locators) == 0 {
		return sourcecontract.MediaLocator{}, fmt.Errorf("%w: media source locator is required", producterror.ErrInvalidInput)
	}
	var locator sourcecontract.MediaLocator
	if err := json.Unmarshal(snapshot.Locators, &locator); err == nil && mcpLocatorType(locator.LocatorType, locator.Kind) != "" {
		return normalizeMCPMediaLocator(locator)
	}
	var locators []sourcecontract.MediaLocator
	if err := json.Unmarshal(snapshot.Locators, &locators); err != nil {
		return sourcecontract.MediaLocator{}, fmt.Errorf("%w: media source locator must be an object or array", producterror.ErrInvalidInput)
	}
	for _, locator := range locators {
		if mcpLocatorType(locator.LocatorType, locator.Kind) == sourcecontract.LocatorTypeMedia {
			return normalizeMCPMediaLocator(locator)
		}
	}
	return sourcecontract.MediaLocator{}, fmt.Errorf("%w: media source locator is missing", producterror.ErrInvalidInput)
}

func normalizeMCPMediaLocator(locator sourcecontract.MediaLocator) (sourcecontract.MediaLocator, error) {
	discriminator := mcpLocatorType(locator.LocatorType, locator.Kind)
	locator.MediaKind = strings.TrimSpace(locator.MediaKind)
	if discriminator != sourcecontract.LocatorTypeMedia {
		return sourcecontract.MediaLocator{}, fmt.Errorf("%w: media source locator kind is invalid", producterror.ErrInvalidInput)
	}
	locator.LocatorType = sourcecontract.LocatorTypeMedia
	locator.Kind = ""
	switch locator.MediaKind {
	case sourcecontract.MediaKindImage, sourcecontract.MediaKindAudio, sourcecontract.MediaKindVideo:
		return locator, nil
	default:
		return sourcecontract.MediaLocator{}, fmt.Errorf("%w: media source kind is unsupported", producterror.ErrInvalidInput)
	}
}

func mcpLocatorType(locatorType string, legacyKind string) string {
	if trimmed := strings.TrimSpace(locatorType); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(legacyKind)
}

func sourceState(snapshot sourcecontract.Snapshot) sourcecontract.State {
	state := snapshot.State
	if state.Removed || strings.TrimSpace(state.State) == sourcecontract.StateRemoved {
		state.State = sourcecontract.StateRemoved
		state.Removed = true
		return state
	}
	if strings.TrimSpace(state.State) == "" {
		state.State = sourcecontract.StateActive
	}
	return state
}

func observationEventID(event *ledger.Event) string {
	if event == nil {
		return ""
	}
	return strings.TrimSpace(event.EventID)
}

func observationEventIDs(event *ledger.Event) []string {
	eventID := observationEventID(event)
	if eventID == "" {
		return nil
	}
	return []string{eventID}
}

func selectedSnapshotArtifactID(snapshot sourcecontract.Snapshot, requested string) (string, error) {
	artifactID := strings.TrimSpace(requested)
	if artifactID == "" {
		if len(snapshot.ArtifactIDs) == 1 {
			return snapshot.ArtifactIDs[0], nil
		}
		return "", fmt.Errorf("%w: artifact_id is required when a source snapshot has multiple artifacts", producterror.ErrInvalidInput)
	}
	if err := validateID("art_", artifactID); err != nil {
		return "", err
	}
	for _, snapshotArtifactID := range snapshot.ArtifactIDs {
		if snapshotArtifactID == artifactID {
			return artifactID, nil
		}
	}
	return "", fmt.Errorf("%w: artifact_id is not part of the source snapshot", producterror.ErrInvalidInput)
}

func BoundedArtifactContent(content []byte, offset int, maxBytes int) (string, int, int, bool, error) {
	return BoundedArtifactContentWithLimit(content, offset, maxBytes, 50000)
}

func BoundedArtifactContentWithLimit(content []byte, offset int, maxBytes, maxLimit int) (string, int, int, bool, error) {
	if !utf8.Valid(content) {
		return "", 0, 0, false, fmt.Errorf("%w: source artifact is not UTF-8 text", producterror.ErrInvalidInput)
	}
	if offset < 0 {
		return "", 0, 0, false, fmt.Errorf("%w: source artifact offset must be non-negative", producterror.ErrInvalidInput)
	}
	if offset > len(content) {
		return "", 0, 0, false, fmt.Errorf("%w: source artifact offset is beyond content length", producterror.ErrInvalidInput)
	}
	if offset < len(content) && !utf8.RuneStart(content[offset]) {
		return "", 0, 0, false, fmt.Errorf("%w: source artifact offset must align to UTF-8 boundary", producterror.ErrInvalidInput)
	}
	if maxLimit < 1 {
		return "", 0, 0, false, fmt.Errorf("%w: source artifact byte ceiling must be positive", producterror.ErrInvalidInput)
	}
	limit := maxBytes
	if limit <= 0 {
		limit = 20000
		if limit > maxLimit {
			limit = maxLimit
		}
	} else if limit > maxLimit {
		limit = maxLimit
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
		return "", 0, 0, false, fmt.Errorf("%w: source artifact could not be sliced as UTF-8", producterror.ErrInvalidInput)
	}
	return string(content[offset:cut]), offset, cut, true, nil
}

func validateID(prefix, id string) error {
	trimmed := strings.TrimSpace(id)
	if !strings.HasPrefix(trimmed, prefix) || len(trimmed) <= len(prefix) {
		return fmt.Errorf("%w: id must start with %s", producterror.ErrInvalidInput, prefix)
	}
	return nil
}
