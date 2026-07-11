package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type ArtifactStore interface {
	CreateRawArtifact(context.Context, RawArtifact) error
	GetRawArtifact(context.Context, string) (RawArtifact, error)
	CreateSourceSnapshot(context.Context, SourceSnapshot) error
	GetSourceSnapshot(context.Context, string) (SourceSnapshot, error)
}

type SourceSnapshotListStore interface {
	ListSourceSnapshots(context.Context, string) ([]SourceSnapshot, error)
}

func (s *Service) CreateRawArtifact(ctx context.Context, req CreateRawArtifactRequest) (RawArtifact, error) {
	artifact, err := buildRawArtifact(req)
	if err != nil {
		return RawArtifact{}, err
	}
	if err := s.store.CreateRawArtifact(ctx, artifact); err != nil {
		return RawArtifact{}, err
	}
	return artifact, nil
}

func buildRawArtifact(req CreateRawArtifactRequest) (RawArtifact, error) {
	artifactID := strings.TrimSpace(req.ArtifactID)
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("art_", artifactID); err != nil {
		return RawArtifact{}, err
	}
	if err := validateID("mis_", missionID); err != nil {
		return RawArtifact{}, err
	}
	if strings.TrimSpace(req.MediaType) == "" {
		return RawArtifact{}, fmt.Errorf("%w: media type is required", ErrInvalidInput)
	}
	if strings.TrimSpace(req.Producer.Type) == "" || strings.TrimSpace(req.Producer.ID) == "" {
		return RawArtifact{}, fmt.Errorf("%w: producer type and id are required", ErrInvalidInput)
	}
	if len(req.Content) == 0 {
		return RawArtifact{}, fmt.Errorf("%w: artifact content is required", ErrInvalidInput)
	}

	sum := sha256.Sum256(req.Content)
	sha := hex.EncodeToString(sum[:])
	if req.ExpectedSHA256 != "" && !strings.EqualFold(strings.TrimSpace(req.ExpectedSHA256), sha) {
		return RawArtifact{}, fmt.Errorf("%w: artifact sha256 mismatch", ErrInvalidInput)
	}

	artifact := RawArtifact{
		ArtifactID: artifactID,
		MissionID:  missionID,
		MediaType:  strings.TrimSpace(req.MediaType),
		ByteSize:   int64(len(req.Content)),
		SHA256:     sha,
		StorageURI: artifactStorageURI(missionID, sha),
		Filename:   strings.TrimSpace(req.Filename),
		Producer: Producer{
			Type: strings.TrimSpace(req.Producer.Type),
			ID:   strings.TrimSpace(req.Producer.ID),
		},
		CreatedAt: time.Now().UTC(),
		Content:   append([]byte(nil), req.Content...),
	}
	return artifact, nil
}

func (s *Service) GetRawArtifact(ctx context.Context, artifactID string) (RawArtifact, error) {
	trimmed := strings.TrimSpace(artifactID)
	if err := validateID("art_", trimmed); err != nil {
		return RawArtifact{}, err
	}
	return s.store.GetRawArtifact(ctx, trimmed)
}

func (s *Service) CreateSourceSnapshot(ctx context.Context, req CreateSourceSnapshotRequest) (SourceSnapshot, error) {
	snapshot, err := s.buildSourceSnapshot(ctx, req, nil)
	if err != nil {
		return SourceSnapshot{}, err
	}
	if err := s.store.CreateSourceSnapshot(ctx, snapshot); err != nil {
		return SourceSnapshot{}, err
	}
	return snapshot, nil
}

func (s *Service) buildSourceSnapshot(
	ctx context.Context,
	req CreateSourceSnapshotRequest,
	pendingArtifacts []RawArtifact,
) (SourceSnapshot, error) {
	snapshotID := strings.TrimSpace(req.SnapshotID)
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("src_", snapshotID); err != nil {
		return SourceSnapshot{}, err
	}
	if err := validateID("mis_", missionID); err != nil {
		return SourceSnapshot{}, err
	}
	if strings.TrimSpace(req.Connector.ConnectorID) == "" || strings.TrimSpace(req.Connector.ConnectorType) == "" {
		return SourceSnapshot{}, fmt.Errorf("%w: connector id and type are required", ErrInvalidInput)
	}
	if strings.TrimSpace(req.Connector.ExternalSourceID) == "" && strings.TrimSpace(req.Connector.ExternalURI) == "" {
		return SourceSnapshot{}, fmt.Errorf("%w: external source id or uri is required", ErrInvalidInput)
	}
	if len(req.Locators) > 0 && !json.Valid(req.Locators) {
		return SourceSnapshot{}, fmt.Errorf("%w: locators must be valid JSON", ErrInvalidInput)
	}
	access := defaultSourceAccess(req.Access)
	if err := validateSourceRetrievalPolicy(access.RetrievalPolicy); err != nil {
		return SourceSnapshot{}, err
	}
	liveReference := access.RetrievalPolicy == SourceRetrievalPolicyLiveReference
	if liveReference {
		if len(req.ArtifactIDs) != 0 {
			return SourceSnapshot{}, fmt.Errorf("%w: live reference sources must not store raw artifacts", ErrInvalidInput)
		}
		if err := validateLiveReference(req); err != nil {
			return SourceSnapshot{}, err
		}
	} else if len(req.ArtifactIDs) == 0 {
		return SourceSnapshot{}, fmt.Errorf("%w: snapshot artifact ids are required", ErrInvalidInput)
	}

	pendingByID := map[string]RawArtifact{}
	for _, artifact := range pendingArtifacts {
		pendingByID[artifact.ArtifactID] = artifact
	}

	artifacts := make([]RawArtifact, 0, len(req.ArtifactIDs))
	artifactIDs := make([]string, 0, len(req.ArtifactIDs))
	seenArtifactIDs := map[string]struct{}{}
	for _, artifactID := range req.ArtifactIDs {
		trimmedArtifactID := strings.TrimSpace(artifactID)
		if err := validateID("art_", trimmedArtifactID); err != nil {
			return SourceSnapshot{}, err
		}
		if _, ok := seenArtifactIDs[trimmedArtifactID]; ok {
			return SourceSnapshot{}, fmt.Errorf("%w: duplicate snapshot artifact id", ErrInvalidInput)
		}
		seenArtifactIDs[trimmedArtifactID] = struct{}{}
		artifact, ok := pendingByID[trimmedArtifactID]
		if !ok {
			var err error
			artifact, err = s.store.GetRawArtifact(ctx, trimmedArtifactID)
			if err != nil {
				return SourceSnapshot{}, err
			}
		}
		if artifact.MissionID != missionID {
			return SourceSnapshot{}, fmt.Errorf("%w: snapshot artifact belongs to another mission", ErrInvalidInput)
		}
		artifacts = append(artifacts, artifact)
		artifactIDs = append(artifactIDs, trimmedArtifactID)
	}

	contentHash := ContentHash{Algorithm: "none", Value: ""}
	if !liveReference {
		var err error
		contentHash, err = verifiedSnapshotHash(req.ContentHash, artifacts)
		if err != nil {
			return SourceSnapshot{}, err
		}
	}
	capturedAt := time.Now().UTC()
	locators := append(json.RawMessage(nil), req.Locators...)
	if len(locators) == 0 {
		locators = json.RawMessage(`[]`)
	}

	snapshot := SourceSnapshot{
		SnapshotID:        snapshotID,
		MissionID:         missionID,
		Connector:         normalizeConnector(req.Connector),
		Title:             strings.TrimSpace(req.Title),
		CapturedAt:        capturedAt,
		ExternalUpdatedAt: req.ExternalUpdatedAt,
		ArtifactIDs:       artifactIDs,
		ContentHash:       contentHash,
		Locators:          locators,
		Access:            access,
	}
	return snapshot, nil
}

func (s *Service) GetSourceSnapshot(ctx context.Context, snapshotID string) (SourceSnapshot, error) {
	trimmed := strings.TrimSpace(snapshotID)
	if err := validateID("src_", trimmed); err != nil {
		return SourceSnapshot{}, err
	}
	snapshot, err := s.store.GetSourceSnapshot(ctx, trimmed)
	if err != nil {
		return SourceSnapshot{}, err
	}
	state, err := s.sourceState(ctx, snapshot.MissionID, snapshot.SnapshotID)
	if err != nil {
		return SourceSnapshot{}, err
	}
	snapshot.State = state
	return snapshot, nil
}

func (s *Service) ListSourceSnapshots(ctx context.Context, missionID string) ([]SourceSnapshot, error) {
	return s.ListSourceSnapshotsWithState(ctx, ListSourceSnapshotsRequest{MissionID: missionID})
}

func artifactStorageURI(missionID, sha string) string {
	prefix := sha
	if len(prefix) > 2 {
		prefix = prefix[:2]
	}
	return fmt.Sprintf("plasma-artifact://%s/%s/%s", missionID, prefix, sha)
}

func verifiedSnapshotHash(requested ContentHash, artifacts []RawArtifact) (ContentHash, error) {
	algorithm := strings.TrimSpace(requested.Algorithm)
	if algorithm == "" {
		algorithm = "sha256"
	}
	if !strings.EqualFold(algorithm, "sha256") {
		return ContentHash{}, fmt.Errorf("%w: snapshot content hash algorithm must be sha256", ErrInvalidInput)
	}

	value := snapshotHashValue(artifacts)
	if requested.Value != "" && !strings.EqualFold(strings.TrimSpace(requested.Value), value) {
		return ContentHash{}, fmt.Errorf("%w: snapshot content hash mismatch", ErrInvalidInput)
	}
	return ContentHash{Algorithm: "sha256", Value: value}, nil
}

func snapshotHashValue(artifacts []RawArtifact) string {
	if len(artifacts) == 1 {
		return artifacts[0].SHA256
	}
	hash := sha256.New()
	for _, artifact := range artifacts {
		hash.Write([]byte(artifact.ArtifactID))
		hash.Write([]byte{0})
		hash.Write([]byte(artifact.SHA256))
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func normalizeConnector(connector ConnectorRef) ConnectorRef {
	return ConnectorRef{
		ConnectorID:      strings.TrimSpace(connector.ConnectorID),
		ConnectorType:    strings.TrimSpace(connector.ConnectorType),
		ExternalSourceID: strings.TrimSpace(connector.ExternalSourceID),
		ExternalURI:      strings.TrimSpace(connector.ExternalURI),
		ExternalVersion:  strings.TrimSpace(connector.ExternalVersion),
		ConnectorVersion: strings.TrimSpace(connector.ConnectorVersion),
	}
}

func defaultSourceAccess(access SourceAccess) SourceAccess {
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
		retrievalPolicy = SourceRetrievalPolicySnapshotOnly
	}
	return SourceAccess{
		Visibility:      visibility,
		License:         license,
		RetrievalPolicy: retrievalPolicy,
	}
}

func validateSourceRetrievalPolicy(policy string) error {
	switch strings.TrimSpace(policy) {
	case SourceRetrievalPolicySnapshotOnly, SourceRetrievalPolicyLiveReference:
		return nil
	default:
		return fmt.Errorf("%w: unsupported source retrieval policy", ErrInvalidInput)
	}
}

func validateLiveReference(req CreateSourceSnapshotRequest) error {
	connector := normalizeConnector(req.Connector)
	switch connector.ConnectorType {
	case SourceConnectorTypeLocalPath:
		return validateLocalPathLiveReference(req)
	case SourceConnectorTypeMediaURL:
		return validateMediaURLLiveReference(req)
	default:
		return fmt.Errorf("%w: live_reference requires local_path or media_url connector", ErrInvalidInput)
	}
}

func validateLocalPathLiveReference(req CreateSourceSnapshotRequest) error {
	connector := normalizeConnector(req.Connector)
	if connector.ConnectorType != SourceConnectorTypeLocalPath {
		return fmt.Errorf("%w: live_reference requires local_path connector", ErrInvalidInput)
	}
	if connector.ExternalURI != "" {
		return fmt.Errorf("%w: local_path live reference must not store absolute external uri", ErrInvalidInput)
	}
	locator, err := parseLocalPathLocator(req.Locators)
	if err != nil {
		return err
	}
	if connector.ExternalSourceID != locator.RootID+":"+locator.RelativePath {
		return fmt.Errorf("%w: local_path external source id must be root_id:relative_path", ErrInvalidInput)
	}
	return nil
}

func validateMediaURLLiveReference(req CreateSourceSnapshotRequest) error {
	connector := normalizeConnector(req.Connector)
	if connector.ConnectorType != SourceConnectorTypeMediaURL {
		return fmt.Errorf("%w: media live reference requires media_url connector", ErrInvalidInput)
	}
	if connector.ExternalURI == "" {
		return fmt.Errorf("%w: media live reference requires external uri", ErrInvalidInput)
	}
	locator, err := parseMediaLocator(req.Locators)
	if err != nil {
		return err
	}
	if locator.MediaKind != MediaKindAudio && locator.MediaKind != MediaKindVideo {
		return fmt.Errorf("%w: only audio and video media live references are supported", ErrInvalidInput)
	}
	if strings.TrimSpace(locator.CanonicalURL) == "" && strings.TrimSpace(locator.DirectMediaURL) == "" {
		return fmt.Errorf("%w: media live reference requires a canonical or direct media URL", ErrInvalidInput)
	}
	return nil
}

func parseMediaLocator(raw json.RawMessage) (MediaLocator, error) {
	if len(raw) == 0 {
		return MediaLocator{}, fmt.Errorf("%w: media locator is required", ErrInvalidInput)
	}
	var locator MediaLocator
	if err := json.Unmarshal(raw, &locator); err == nil && locatorDiscriminator(locator.LocatorType, locator.Kind) != "" {
		return normalizeMediaLocator(locator)
	}
	var locators []MediaLocator
	if err := json.Unmarshal(raw, &locators); err != nil {
		return MediaLocator{}, fmt.Errorf("%w: media locator must be an object or one-item array", ErrInvalidInput)
	}
	if len(locators) != 1 {
		return MediaLocator{}, fmt.Errorf("%w: media locator must contain exactly one locator", ErrInvalidInput)
	}
	return normalizeMediaLocator(locators[0])
}

func normalizeMediaLocator(locator MediaLocator) (MediaLocator, error) {
	discriminator := locatorDiscriminator(locator.LocatorType, locator.Kind)
	locator.MediaKind = strings.TrimSpace(locator.MediaKind)
	if discriminator != SourceLocatorTypeMedia {
		return MediaLocator{}, fmt.Errorf("%w: media locator kind is required", ErrInvalidInput)
	}
	locator.LocatorType = SourceLocatorTypeMedia
	locator.Kind = ""
	switch locator.MediaKind {
	case MediaKindImage, MediaKindAudio, MediaKindVideo:
	default:
		return MediaLocator{}, fmt.Errorf("%w: unsupported media kind", ErrInvalidInput)
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

func parseLocalPathLocator(raw json.RawMessage) (LocalPathLocator, error) {
	if len(raw) == 0 {
		return LocalPathLocator{}, fmt.Errorf("%w: local_path locator is required", ErrInvalidInput)
	}
	var locator LocalPathLocator
	if err := json.Unmarshal(raw, &locator); err == nil && locatorDiscriminator(locator.LocatorType, locator.Kind) != "" {
		return normalizeLocalPathLocator(locator)
	}
	var locators []LocalPathLocator
	if err := json.Unmarshal(raw, &locators); err != nil {
		return LocalPathLocator{}, fmt.Errorf("%w: local_path locator must be an object or one-item array", ErrInvalidInput)
	}
	if len(locators) != 1 {
		return LocalPathLocator{}, fmt.Errorf("%w: local_path locator must contain exactly one locator", ErrInvalidInput)
	}
	return normalizeLocalPathLocator(locators[0])
}

func normalizeLocalPathLocator(locator LocalPathLocator) (LocalPathLocator, error) {
	discriminator := locatorDiscriminator(locator.LocatorType, locator.Kind)
	locator.RootID = strings.TrimSpace(locator.RootID)
	locator.RelativePath = normalizePublicRelativePath(locator.RelativePath)
	locator.PathKind = strings.TrimSpace(locator.PathKind)
	if discriminator != SourceLocatorTypeLocalPath {
		return LocalPathLocator{}, fmt.Errorf("%w: local_path locator kind is required", ErrInvalidInput)
	}
	locator.LocatorType = SourceLocatorTypeLocalPath
	locator.Kind = ""
	if !validLocalPathRootID(locator.RootID) {
		return LocalPathLocator{}, fmt.Errorf("%w: invalid local path root id", ErrInvalidInput)
	}
	if err := validatePublicRelativePath(locator.RelativePath); err != nil {
		return LocalPathLocator{}, err
	}
	switch locator.PathKind {
	case "file", "directory":
	default:
		return LocalPathLocator{}, fmt.Errorf("%w: local_path path_kind must be file or directory", ErrInvalidInput)
	}
	return locator, nil
}

func locatorDiscriminator(locatorType string, legacyKind string) string {
	if trimmed := strings.TrimSpace(locatorType); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(legacyKind)
}

func normalizePublicRelativePath(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		return "."
	}
	cleaned := path.Clean(value)
	if cleaned == "/" {
		return "."
	}
	return cleaned
}

func validatePublicRelativePath(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%w: relative path is required", ErrInvalidInput)
	}
	if strings.ContainsRune(value, 0) || containsControl(value) {
		return fmt.Errorf("%w: relative path contains control characters", ErrInvalidInput)
	}
	if filepath.IsAbs(value) || strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\\`) || looksLikeWindowsAbs(value) {
		return fmt.Errorf("%w: relative path must not be absolute", ErrInvalidInput)
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." {
			return fmt.Errorf("%w: relative path must not contain traversal", ErrInvalidInput)
		}
	}
	return nil
}

func validLocalPathRootID(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func looksLikeWindowsAbs(value string) bool {
	if len(value) >= 3 && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':' && (value[2] == '/' || value[2] == '\\') {
		return true
	}
	return strings.HasPrefix(value, "//")
}

func containsControl(value string) bool {
	for _, r := range value {
		if r >= 0 && r < 0x20 {
			return true
		}
	}
	return false
}
