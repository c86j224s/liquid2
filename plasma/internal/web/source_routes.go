package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/sourceingest"
	"github.com/c86j224s/liquid2/plasma/internal/sources/pdftext"
)

func (server *Server) handleMissionSources(w http.ResponseWriter, r *http.Request, missionID string, rest []string) {
	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			sources, err := server.service.ListSourceSnapshotsWithState(r.Context(), app.ListSourceSnapshotsRequest{
				MissionID:         missionID,
				IncludeRemoved:    queryBool(r, "include_removed"),
				IncludeSuperseded: queryBool(r, "include_superseded"),
			})
			if err != nil {
				writeAppError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"sources": sources})
		case http.MethodPost:
			server.handleTextSource(w, r, missionID)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	if len(rest) == 1 && rest[0] == "local_path" {
		server.handleLocalPathAttach(w, r, missionID)
		return
	}
	if len(rest) == 2 && rest[0] == "local_path" && rest[1] == "roots" {
		server.handleLocalPathRoots(w, r, missionID)
		return
	}
	if len(rest) == 2 && rest[0] == "local_path" && rest[1] == "tree" {
		server.handleLocalPathTree(w, r, missionID)
		return
	}
	if len(rest) == 1 && rest[0] == "text" {
		server.handleTextSource(w, r, missionID)
		return
	}
	if len(rest) == 1 && rest[0] == "upload" {
		server.handleUploadSource(w, r, missionID)
		return
	}
	if len(rest) == 1 && rest[0] == "url" {
		server.handleURLSource(w, r, missionID)
		return
	}
	if len(rest) == 1 && rest[0] == "media_url" {
		server.handleMediaURLSource(w, r, missionID)
		return
	}
	if len(rest) == 1 && rest[0] == "pdf_url" {
		server.handlePDFURLSource(w, r, missionID)
		return
	}
	if len(rest) == 2 && rest[0] == "liquid2" && rest[1] == "search" {
		server.handleLiquid2Search(w, r, missionID)
		return
	}
	if len(rest) == 2 && rest[0] == "liquid2" && rest[1] == "snapshot" {
		server.handleLiquid2Snapshot(w, r, missionID)
		return
	}
	if len(rest) == 3 && rest[0] == "confluence" && rest[1] == "connections" {
		writeConfluenceMissionLifecycleDeprecated(w)
		return
	}
	if len(rest) == 4 && rest[0] == "confluence" && rest[1] == "connections" && rest[3] == "revoke" {
		writeConfluenceMissionLifecycleDeprecated(w)
		return
	}
	if len(rest) == 2 && rest[0] == "confluence" {
		switch rest[1] {
		case "connections":
			server.handleConfluenceConnections(w, r, missionID)
			return
		case "sites":
			server.handleConfluenceSites(w, r, missionID)
			return
		case "spaces":
			server.handleConfluenceSpaces(w, r, missionID)
			return
		case "space-pages":
			server.handleConfluenceSpacePages(w, r, missionID)
			return
		case "children":
			server.handleConfluencePageChildren(w, r, missionID)
			return
		case "search":
			server.handleConfluenceSearch(w, r, missionID)
			return
		case "url":
			server.handleConfluenceURLSnapshot(w, r, missionID)
			return
		case "preview":
			server.handleConfluencePreview(w, r, missionID)
			return
		case "snapshot":
			server.handleConfluenceSnapshot(w, r, missionID)
			return
		case "check-update":
			server.handleConfluenceCheckUpdate(w, r, missionID)
			return
		case "update-preview":
			server.handleConfluenceUpdatePreview(w, r, missionID)
			return
		case "update":
			server.handleConfluenceUpdate(w, r, missionID)
			return
		}
	}
	if len(rest) == 3 && rest[0] == "confluence" && rest[1] == "oauth" {
		switch rest[2] {
		case "start":
			writeConfluenceMissionLifecycleDeprecated(w)
			return
		case "callback":
			writeConfluenceMissionLifecycleDeprecated(w)
			return
		}
	}
	if len(rest) == 2 {
		switch rest[1] {
		case "read":
			server.handleSourceRead(w, r, missionID, rest[0])
			return
		case "grep":
			server.handleSourceGrep(w, r, missionID, rest[0])
			return
		case "remove":
			server.handleSourceRemove(w, r, missionID, rest[0])
			return
		case "restore":
			server.handleSourceRestore(w, r, missionID, rest[0])
			return
		}
	}
	http.NotFound(w, r)
}

func (server *Server) handleLocalPathRoots(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	roots, err := server.service.ListLocalPathRoots(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"roots": roots, "mission_id": missionID})
}

func (server *Server) handleLocalPathTree(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	req := localPathTreeRequest{}
	if r.Method == http.MethodPost {
		if !decodeJSON(w, r, &req) {
			return
		}
	} else {
		req.RootID = r.URL.Query().Get("root_id")
		req.RelativePath = r.URL.Query().Get("relative_path")
		req.Depth = queryInt(r, "depth")
		req.Limit = queryInt(r, "limit")
	}
	if err := validateWebRelativePath(req.RelativePath); err != nil {
		writeAppError(w, err)
		return
	}
	tree, err := server.service.BrowseLocalPathRoot(r.Context(), app.BrowseLocalPathRootRequest{
		RootID:       req.RootID,
		RelativePath: req.RelativePath,
		Depth:        req.Depth,
		Limit:        req.Limit,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tree": tree, "mission_id": missionID})
}

func (server *Server) handleLocalPathAttach(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req localPathAttachRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := validateWebRelativePath(req.RelativePath); err != nil {
		writeAppError(w, err)
		return
	}
	result, err := server.service.AttachLocalPathSource(r.Context(), app.AttachLocalPathSourceRequest{
		MissionID:    missionID,
		RootID:       req.RootID,
		RelativePath: req.RelativePath,
		Title:        req.Title,
		Restore:      req.Restore,
		Producer:     app.Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	status := http.StatusCreated
	if result.Existing {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{
		"snapshot":         result.Snapshot,
		"event":            result.Event,
		"existing":         result.Existing,
		"restored":         result.Restored,
		"restore_required": result.RestoreRequired,
	})
}

func (server *Server) handleSourceRead(w http.ResponseWriter, r *http.Request, missionID string, snapshotID string) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	req := sourceReadRequest{}
	if r.Method == http.MethodPost {
		if !decodeJSON(w, r, &req) {
			return
		}
	} else {
		req.Offset = int64(queryInt(r, "offset"))
		req.MaxBytes = int64(queryInt(r, "max_bytes"))
		req.Depth = queryInt(r, "depth")
		req.Limit = queryInt(r, "limit")
	}
	snapshot, err := server.service.GetSourceSnapshot(r.Context(), snapshotID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if snapshot.MissionID != missionID {
		writeError(w, http.StatusNotFound, "source not found")
		return
	}
	if snapshot.State.Removed || snapshot.State.State == app.SourceStateRemoved {
		writeAppError(w, fmt.Errorf("%w: source is removed", app.ErrInvalidInput))
		return
	}
	if snapshot.Connector.ConnectorType == app.SourceConnectorTypeMediaURL {
		server.writeMediaSourceRead(w, r, missionID, snapshot)
		return
	}
	if snapshot.Access.RetrievalPolicy == app.SourceRetrievalPolicyLiveReference && snapshot.Connector.ConnectorType == app.SourceConnectorTypeLocalPath {
		if localPathLocatorKind(snapshot) == "directory" {
			result, err := server.service.TreeLocalPathSource(r.Context(), app.TreeLocalPathSourceRequest{
				MissionID:     missionID,
				SnapshotID:    snapshotID,
				Depth:         req.Depth,
				Limit:         req.Limit,
				Producer:      app.Producer{Type: "user", ID: "plasma-ui"},
				ToolSessionID: "plasma-ui",
			})
			if err != nil {
				writeAppError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"snapshot":             result.Snapshot,
				"tree":                 result.Tree,
				"observation_metadata": result.Tree.Metadata,
				"observation_event":    result.ObservationEvent,
				"observation_event_id": sourceEventID(result.ObservationEvent),
			})
			return
		}
		result, err := server.service.ReadLocalPathSource(r.Context(), app.ReadLocalPathSourceRequest{
			MissionID:     missionID,
			SnapshotID:    snapshotID,
			Offset:        req.Offset,
			MaxBytes:      req.MaxBytes,
			Producer:      app.Producer{Type: "user", ID: "plasma-ui"},
			ToolSessionID: "plasma-ui",
		})
		if err != nil {
			writeAppError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"snapshot":             result.Snapshot,
			"content":              result.Read.Content,
			"observation_metadata": result.Read.Metadata,
			"observation_event":    result.ObservationEvent,
			"observation_event_id": sourceEventID(result.ObservationEvent),
		})
		return
	}
	artifactID := strings.TrimSpace(req.ArtifactID)
	if artifactID == "" && len(snapshot.ArtifactIDs) == 1 {
		artifactID = snapshot.ArtifactIDs[0]
	}
	if artifactID == "" {
		writeError(w, http.StatusBadRequest, "artifact_id is required for snapshot sources")
		return
	}
	if !snapshotHasArtifactID(snapshot, artifactID) {
		writeError(w, http.StatusBadRequest, "artifact_id is not attached to this source snapshot")
		return
	}
	artifact, err := server.service.GetRawArtifact(r.Context(), artifactID)
	if err != nil || artifact.MissionID != missionID {
		writeError(w, http.StatusNotFound, "artifact not found")
		return
	}
	if pdftext.IsPDFMediaType(artifact.MediaType) || pdftext.IsPDFBytes(artifact.Content) {
		writePDFArtifactRead(w, artifact, int(req.Offset), int(req.MaxBytes))
		return
	}
	writeRawArtifactRead(w, artifact, int(req.Offset), int(req.MaxBytes))
}

func snapshotHasArtifactID(snapshot app.SourceSnapshot, artifactID string) bool {
	artifactID = strings.TrimSpace(artifactID)
	if artifactID == "" {
		return false
	}
	for _, attached := range snapshot.ArtifactIDs {
		if strings.TrimSpace(attached) == artifactID {
			return true
		}
	}
	return false
}

func (server *Server) writeMediaSourceRead(w http.ResponseWriter, r *http.Request, missionID string, snapshot app.SourceSnapshot) {
	locator, err := mediaLocatorFromJSON(snapshot.Locators)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response := map[string]any{
		"snapshot": snapshot,
		"media":    locator,
		"note":     mediaSourceReadNote(locator.MediaKind),
	}
	if len(snapshot.ArtifactIDs) > 0 {
		artifact, err := server.service.GetRawArtifact(r.Context(), snapshot.ArtifactIDs[0])
		if err != nil || artifact.MissionID != missionID {
			writeError(w, http.StatusNotFound, "artifact not found")
			return
		}
		response["artifact"] = map[string]any{
			"artifact_id": artifact.ArtifactID,
			"media_type":  artifact.MediaType,
			"byte_size":   artifact.ByteSize,
			"sha256":      artifact.SHA256,
			"filename":    artifact.Filename,
			"storage_uri": artifact.StorageURI,
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func (server *Server) handleSourceGrep(w http.ResponseWriter, r *http.Request, missionID string, snapshotID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req sourceGrepRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := server.service.GrepLocalPathSource(r.Context(), app.GrepLocalPathSourceRequest{
		MissionID:     missionID,
		SnapshotID:    snapshotID,
		Query:         req.Query,
		MaxSnippets:   req.MaxSnippets,
		Producer:      app.Producer{Type: "user", ID: "plasma-ui"},
		ToolSessionID: "plasma-ui",
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"snapshot":             result.Snapshot,
		"grep":                 result.Grep,
		"observation_event":    result.ObservationEvent,
		"observation_event_id": sourceEventID(result.ObservationEvent),
	})
}

func (server *Server) handleSourceRemove(w http.ResponseWriter, r *http.Request, missionID string, snapshotID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req sourceRemoveRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := server.service.RemoveSource(r.Context(), app.RemoveSourceRequest{
		MissionID:  missionID,
		SnapshotID: snapshotID,
		Reason:     req.Reason,
		Producer:   app.Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"snapshot": result.Snapshot, "event": result.Event, "idempotent": result.Idempotent})
}

func (server *Server) handleSourceRestore(w http.ResponseWriter, r *http.Request, missionID string, snapshotID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !decodeJSON(w, r, &struct{}{}) {
		return
	}
	result, err := server.service.RestoreSource(r.Context(), app.RestoreSourceRequest{
		MissionID:  missionID,
		SnapshotID: snapshotID,
		Producer:   app.Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"snapshot": result.Snapshot, "event": result.Event, "idempotent": result.Idempotent})
}

func (server *Server) handleTextSource(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req textSourceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	req.ExternalURI = strings.TrimSpace(req.ExternalURI)
	if req.Title == "" {
		req.Title = "Pasted text source"
	}
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "source content is required")
		return
	}
	artifactID := newID("art")
	snapshotID := newID("src")
	eventID := newID("evt")
	result, err := sourceingest.CreateTextSourceWithEvent(r.Context(), server.service, sourceingest.CreateTextSourceWithEventRequest{
		MissionID:  missionID,
		ArtifactID: artifactID,
		SnapshotID: snapshotID,
		EventID:    eventID,
		Producer:   app.Producer{Type: "user", ID: "plasma-ui"},
		Source: sourceingest.TextSourceContent{
			Title:       req.Title,
			Content:     req.Content,
			ExternalURI: req.ExternalURI,
		},
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"artifact": rawArtifactResponse(result.Artifact),
		"snapshot": result.Snapshot,
		"event":    result.Event,
	})
}

func (server *Server) handleUploadSource(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, app.UploadedFileMaxBytes+1<<20)
	upload, err := readUploadSourceMultipart(r)
	if err != nil {
		writeAppError(w, err)
		return
	}
	contentSHA := sha256Hex(upload.Content)
	unlockContent := server.sources.lock(missionID + "\x00upload-sha\x00" + contentSHA)
	defer unlockContent()
	result, err := server.service.CreateUploadedFileSourceWithEvent(r.Context(), app.CreateUploadedFileSourceRequest{
		MissionID:        missionID,
		ArtifactID:       newID("art"),
		SnapshotID:       newID("src"),
		EventID:          newID("evt"),
		Title:            upload.Title,
		OriginalFilename: upload.Filename,
		Content:          upload.Content,
		Producer:         app.Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	status := http.StatusCreated
	if result.Existing {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{
		"artifact": rawArtifactResponse(result.Artifact),
		"snapshot": result.Snapshot,
		"event":    result.Event,
		"existing": result.Existing,
	})
}

type uploadSourceMultipart struct {
	Filename string
	Title    string
	Content  []byte
}

func readUploadSourceMultipart(r *http.Request) (uploadSourceMultipart, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return uploadSourceMultipart{}, fmt.Errorf("%w: invalid multipart upload", app.ErrInvalidInput)
	}
	var upload uploadSourceMultipart
	var sawFile bool
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return uploadSourceMultipart{}, fmt.Errorf("%w: invalid multipart upload", app.ErrInvalidInput)
		}
		switch part.FormName() {
		case "title":
			title, err := readSmallMultipartField(part, 1024)
			if err != nil {
				return uploadSourceMultipart{}, err
			}
			upload.Title = strings.TrimSpace(title)
		case "file":
			if sawFile {
				return uploadSourceMultipart{}, fmt.Errorf("%w: upload accepts one file", app.ErrInvalidInput)
			}
			sawFile = true
			upload.Filename = part.FileName()
			content, err := io.ReadAll(io.LimitReader(part, app.UploadedFileMaxBytes+1))
			if err != nil {
				return uploadSourceMultipart{}, err
			}
			if int64(len(content)) > app.UploadedFileMaxBytes {
				return uploadSourceMultipart{}, fmt.Errorf("%w: uploaded source exceeds 100 MiB limit", app.ErrInvalidInput)
			}
			upload.Content = content
		default:
			_, _ = io.Copy(io.Discard, io.LimitReader(part, 4096))
		}
	}
	if !sawFile {
		return uploadSourceMultipart{}, fmt.Errorf("%w: file field is required", app.ErrInvalidInput)
	}
	return upload, nil
}

func readSmallMultipartField(r io.Reader, maxBytes int64) (string, error) {
	content, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return "", err
	}
	if int64(len(content)) > maxBytes {
		return "", fmt.Errorf("%w: multipart field is too large", app.ErrInvalidInput)
	}
	if !utf8.Valid(content) {
		return "", fmt.Errorf("%w: multipart field is not UTF-8", app.ErrInvalidInput)
	}
	return string(content), nil
}

func (server *Server) handleURLSource(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req urlSourceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	normalizedURL, err := normalizedHTTPURL(req.URL)
	if err != nil {
		writeAppError(w, err)
		return
	}
	unlock := server.sources.lock(missionID + "\x00" + normalizedURL)
	defer unlock()
	if existing, ok, err := sourceingest.ExistingSourceSnapshotForURL(r.Context(), server.service, missionID, normalizedURL); err != nil {
		writeAppError(w, err)
		return
	} else if ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"existing": true,
			"snapshot": existing,
		})
		return
	}
	if target, ok, err := parseConfluencePageURL(normalizedURL); ok {
		if err != nil {
			writeAppError(w, err)
			return
		}
		result, err := server.snapshotConfluenceURLSource(r.Context(), missionID, target, req.Title)
		if err != nil {
			server.recordSourceSnapshotFailure(r.Context(), missionID, "confluence_url", normalizedURL, err)
			writeAppError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, result)
		return
	}
	if staged, ok, err := sourceingest.LatestStagedSourceCandidateForURL(r.Context(), server.service, missionID, normalizedURL); err != nil {
		writeAppError(w, err)
		return
	} else if ok {
		if result, handled := server.createURLSourceFromStagedCandidate(w, r, missionID, normalizedURL, req.Title, staged); handled {
			if result != nil {
				writeJSON(w, http.StatusCreated, result)
			}
			return
		}
	}
	fetched, err := server.fetchURLSource(r.Context(), normalizedURL)
	if err != nil {
		server.recordSourceSnapshotFailure(r.Context(), missionID, "url", normalizedURL, err)
		writeAppError(w, err)
		return
	}
	contentSHA := sha256Hex(fetched.Content)
	unlockContent := server.sources.lock(missionID + "\x00sha\x00" + contentSHA)
	defer unlockContent()
	if existing, ok, err := sourceingest.ExistingSourceSnapshotForContentHash(r.Context(), server.service, missionID, contentSHA); err != nil {
		writeAppError(w, err)
		return
	} else if ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"existing": true,
			"snapshot": existing,
		})
		return
	}
	result, err := sourceingest.CreateFetchedURLSourceWithEvent(r.Context(), server.service, sourceingest.CreateFetchedURLSourceRequest{
		MissionID:  missionID,
		URL:        normalizedURL,
		Title:      req.Title,
		ArtifactID: newID("art"),
		SnapshotID: newID("src"),
		EventID:    newID("evt"),
		Producer:   app.Producer{Type: "user", ID: "plasma-ui"},
		Fetched:    appFetchedURLSource(fetched),
		FetchedAt:  time.Now().UTC(),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"artifact": rawArtifactResponse(result.Artifact),
		"snapshot": result.Snapshot,
		"event":    result.Event,
	})
}

func (server *Server) createURLSourceFromStagedCandidate(w http.ResponseWriter, r *http.Request, missionID string, normalizedURL string, requestedTitle string, staged sourceingest.StagedSourceCandidate) (map[string]any, bool) {
	contentSHA := staged.Artifact.SHA256
	if contentSHA == "" {
		contentSHA = sha256Hex(staged.Artifact.Content)
	}
	unlockContent := server.sources.lock(missionID + "\x00sha\x00" + contentSHA)
	defer unlockContent()
	if existing, ok, err := sourceingest.ExistingSourceSnapshotForContentHash(r.Context(), server.service, missionID, contentSHA); err != nil {
		writeAppError(w, err)
		return nil, true
	} else if ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"existing": true,
			"snapshot": existing,
		})
		return nil, true
	}
	if pdftext.IsPDFMediaType(staged.Artifact.MediaType) || pdftext.IsPDFBytes(staged.Artifact.Content) {
		return server.createPDFURLSourceFromStagedCandidate(w, r, missionID, normalizedURL, requestedTitle, staged), true
	}
	result, err := sourceingest.CreateStagedURLSourceWithEvent(r.Context(), server.service, sourceingest.CreateStagedURLSourceRequest{
		MissionID:  missionID,
		URL:        normalizedURL,
		Title:      requestedTitle,
		SnapshotID: newID("src"),
		EventID:    newID("evt"),
		Producer:   app.Producer{Type: "user", ID: "plasma-ui"},
		Staged:     staged,
	})
	if err != nil {
		writeAppError(w, err)
		return nil, true
	}
	return map[string]any{
		"artifact":                rawArtifactResponse(staged.Artifact),
		"snapshot":                result.Snapshot,
		"event":                   result.Event,
		"reused_source_candidate": true,
	}, true
}

func (server *Server) createPDFURLSourceFromStagedCandidate(w http.ResponseWriter, r *http.Request, missionID string, normalizedURL string, requestedTitle string, staged sourceingest.StagedSourceCandidate) map[string]any {
	result, err := sourceingest.CreateStagedPDFURLSourceWithEvent(r.Context(), server.service, sourceingest.CreateStagedPDFURLSourceRequest{
		MissionID:  missionID,
		URL:        normalizedURL,
		Title:      requestedTitle,
		SnapshotID: newID("src"),
		EventID:    newID("evt"),
		Producer:   app.Producer{Type: "user", ID: "plasma-ui"},
		Staged:     staged,
	})
	if err != nil {
		writeAppError(w, err)
		return nil
	}
	return map[string]any{
		"artifact":                rawArtifactResponse(staged.Artifact),
		"snapshot":                result.Snapshot,
		"event":                   result.Event,
		"reused_source_candidate": true,
	}
}

func (server *Server) handleMediaURLSource(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req mediaURLSourceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	normalizedURL, err := normalizedHTTPURL(req.URL)
	if err != nil {
		writeAppError(w, err)
		return
	}
	unlock := server.sources.lock(missionID + "\x00media\x00" + normalizedURL)
	defer unlock()
	if existing, ok, err := sourceingest.ExistingSourceSnapshotForURL(r.Context(), server.service, missionID, normalizedURL); err != nil {
		writeAppError(w, err)
		return
	} else if ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"existing": true,
			"snapshot": existing,
		})
		return
	}
	fetched, err := server.fetchMedia(r.Context(), normalizedURL)
	if err != nil {
		server.recordSourceSnapshotFailure(r.Context(), missionID, "media_url", normalizedURL, err)
		writeAppError(w, err)
		return
	}
	contentSHA := ""
	if fetched.MediaKind == app.MediaKindImage {
		contentSHA = sha256Hex(fetched.Content)
		unlockContent := server.sources.lock(missionID + "\x00media-sha\x00" + contentSHA)
		defer unlockContent()
		if existing, ok, err := sourceingest.ExistingSourceSnapshotForContentHash(r.Context(), server.service, missionID, contentSHA); err != nil {
			writeAppError(w, err)
			return
		} else if ok {
			writeJSON(w, http.StatusOK, map[string]any{
				"existing": true,
				"snapshot": existing,
			})
			return
		}
	}
	artifactID := ""
	if fetched.MediaKind == app.MediaKindImage {
		artifactID = newID("art")
	}
	result, err := sourceingest.CreateFetchedMediaURLSourceWithEvent(r.Context(), server.service, sourceingest.CreateFetchedMediaURLSourceRequest{
		MissionID:   missionID,
		URL:         normalizedURL,
		Title:       req.Title,
		License:     req.License,
		Attribution: req.Attribution,
		ArtifactID:  artifactID,
		SnapshotID:  newID("src"),
		EventID:     newID("evt"),
		Producer:    app.Producer{Type: "user", ID: "plasma-ui"},
		Fetched:     appFetchedMediaSource(fetched),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response := map[string]any{
		"snapshot": result.Snapshot,
		"event":    result.Event,
	}
	if result.HasArtifact {
		response["artifact"] = rawArtifactResponse(result.Artifact)
	}
	writeJSON(w, http.StatusCreated, response)
}

func (server *Server) handlePDFURLSource(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req pdfURLSourceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	normalizedURL, err := normalizedHTTPURL(req.URL)
	if err != nil {
		writeAppError(w, err)
		return
	}
	unlock := server.sources.lock(missionID + "\x00pdf\x00" + normalizedURL)
	defer unlock()
	if existing, ok, err := sourceingest.ExistingSourceSnapshotForURL(r.Context(), server.service, missionID, normalizedURL); err != nil {
		writeAppError(w, err)
		return
	} else if ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"existing": true,
			"snapshot": existing,
		})
		return
	}
	if staged, ok, err := sourceingest.LatestStagedSourceCandidateForURL(r.Context(), server.service, missionID, normalizedURL); err != nil {
		writeAppError(w, err)
		return
	} else if ok && (pdftext.IsPDFMediaType(staged.Artifact.MediaType) || pdftext.IsPDFBytes(staged.Artifact.Content)) {
		if result, handled := server.createURLSourceFromStagedCandidate(w, r, missionID, normalizedURL, req.Title, staged); handled {
			if result != nil {
				writeJSON(w, http.StatusCreated, result)
			}
			return
		}
	}
	fetched, err := server.fetchPDF(r.Context(), normalizedURL)
	if err != nil {
		server.recordSourceSnapshotFailure(r.Context(), missionID, app.SourceConnectorTypePDFURL, normalizedURL, err)
		writeAppError(w, err)
		return
	}
	contentSHA := sha256Hex(fetched.Content)
	unlockContent := server.sources.lock(missionID + "\x00pdf-sha\x00" + contentSHA)
	defer unlockContent()
	if existing, ok, err := sourceingest.ExistingSourceSnapshotForContentHash(r.Context(), server.service, missionID, contentSHA); err != nil {
		writeAppError(w, err)
		return
	} else if ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"existing": true,
			"snapshot": existing,
		})
		return
	}
	result, err := sourceingest.CreateFetchedPDFURLSourceWithEvent(r.Context(), server.service, sourceingest.CreateFetchedPDFURLSourceRequest{
		MissionID:  missionID,
		URL:        normalizedURL,
		Title:      req.Title,
		ArtifactID: newID("art"),
		SnapshotID: newID("src"),
		EventID:    newID("evt"),
		Producer:   app.Producer{Type: "user", ID: "plasma-ui"},
		Fetched:    appFetchedPDFSource(fetched),
		FetchedAt:  time.Now().UTC(),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"artifact": rawArtifactResponse(result.Artifact),
		"snapshot": result.Snapshot,
		"event":    result.Event,
	})
}

func (server *Server) existingSourceSnapshotForContentHash(ctx context.Context, missionID string, sha string) (app.SourceSnapshot, bool, error) {
	sha = strings.ToLower(strings.TrimSpace(sha))
	if sha == "" {
		return app.SourceSnapshot{}, false, nil
	}
	sources, err := server.service.ListSourceSnapshots(ctx, missionID)
	if err != nil {
		return app.SourceSnapshot{}, false, err
	}
	for _, source := range sources {
		if strings.EqualFold(strings.TrimSpace(source.ContentHash.Value), sha) {
			return source, true, nil
		}
	}
	return app.SourceSnapshot{}, false, nil
}

const confluenceOAuthRefreshSkew = 2 * time.Minute
