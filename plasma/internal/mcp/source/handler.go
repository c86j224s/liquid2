package source

import uploadsource "github.com/c86j224s/liquid2/plasma/internal/source"

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"github.com/c86j224s/liquid2/plasma/internal/source/liquid2source"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/pdfdocument"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
)

func (server *Handler) CallSourcesList(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input SourcesListInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := server.ValidateID("mis_", missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.EnforceMission(missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	snapshots, err := server.Service.ListSourceSnapshotsWithState(ctx, sourcecontract.ListRequest{
		MissionID:         missionID,
		IncludeRemoved:    input.IncludeRemoved,
		IncludeSuperseded: input.IncludeSuperseded,
	})
	if err != nil {
		return server.ErrorFromErr(call.Name, missionID, err, nil)
	}
	return wire.ToolResult{
		ToolName:  call.Name,
		MissionID: missionID,
		Content:   SourcesListOutput{Sources: SourceSnapshotsFromApp(snapshots)},
	}
}

func (server *Handler) CallSourcesRead(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input SourcesReadInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := server.ValidateID("mis_", missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.EnforceMission(missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	snapshotID := strings.TrimSpace(input.SnapshotID)
	if err := server.ValidateID("src_", snapshotID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	snapshot, err := server.Service.GetSourceSnapshot(ctx, snapshotID)
	if err != nil {
		return server.ErrorFromErr(call.Name, missionID, err, []string{snapshotID})
	}
	if snapshot.MissionID != missionID {
		return server.ErrorResult(call.Name, missionID, "validation", "source snapshot belongs to another mission", false, []string{snapshotID})
	}
	if sourceState(snapshot).Removed {
		return server.ErrorResult(call.Name, missionID, "validation", "source snapshot is removed", false, []string{snapshotID})
	}
	if strings.TrimSpace(input.Subpath) != "" && (sourceRetrievalPolicy(snapshot) != sourcecontract.RetrievalPolicyLiveReference || snapshot.Connector.ConnectorType != sourcecontract.ConnectorTypeLocalPath) {
		return server.ErrorResult(call.Name, missionID, "validation", "subpath is only valid for live local_path sources", false, []string{snapshotID})
	}
	if snapshot.Connector.ConnectorType == sourcecontract.ConnectorTypeMediaURL {
		return server.callSourcesReadMedia(ctx, call.Name, missionID, snapshot)
	}
	if sourceRetrievalPolicy(snapshot) == sourcecontract.RetrievalPolicyLiveReference {
		return server.callSourcesReadLiveLocalPath(ctx, call.Name, missionID, snapshot, input)
	}
	artifactID, err := selectedSnapshotArtifactID(snapshot, input.ArtifactID)
	if err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{snapshotID, input.ArtifactID})
	}
	artifact, err := server.Service.GetRawArtifact(ctx, artifactID)
	if err != nil {
		return server.ErrorFromErr(call.Name, missionID, err, []string{snapshotID, artifactID})
	}
	if artifact.MissionID != missionID {
		return server.ErrorResult(call.Name, missionID, "validation", "source artifact belongs to another mission", false, []string{artifactID})
	}
	if pdfdocument.IsPDFMediaType(artifact.MediaType) || pdfdocument.IsPDFBytes(artifact.Content) {
		chunk, err := pdfdocument.ExtractChunk(artifact.Content, input.Offset, input.MaxBytes)
		if err != nil {
			return server.ErrorResult(call.Name, missionID, "validation", "PDF text extraction failed: "+err.Error(), false, []string{artifactID})
		}
		return wire.ToolResult{
			ToolName:  call.Name,
			MissionID: missionID,
			Content: SourcesReadOutput{
				Snapshot:           SourceSnapshotFromApp(snapshot),
				Artifact:           server.ArtifactOutput(artifact),
				Content:            chunk.Text,
				Offset:             chunk.Offset,
				NextOffset:         chunk.NextOffset,
				ContentLength:      chunk.ContentLength,
				ContentLengthKnown: chunk.ContentLengthKnown,
				Truncated:          chunk.Truncated,
				Extraction: &wire.SourceExtractionOutput{
					Type:               "pdf_text",
					PageCount:          chunk.PageCount,
					TextLength:         chunk.ContentLength,
					TextLengthKnown:    chunk.ContentLengthKnown,
					SuggestedReadBytes: pdfdocument.DefaultChunkMaxBytes,
					MaxReadBytes:       pdfdocument.MaxChunkBytes,
				},
			},
		}
	}
	if uploadsource.UploadedArtifactReadKind(artifact) == "metadata" {
		return wire.ToolResult{
			ToolName:  call.Name,
			MissionID: missionID,
			Content: SourcesReadOutput{
				Snapshot:           SourceSnapshotFromApp(snapshot),
				Artifact:           server.ArtifactOutput(artifact),
				Content:            "",
				ContentLength:      int(artifact.ByteSize),
				ContentLengthKnown: true,
				MetadataOnly:       true,
			},
		}
	}
	content, offset, nextOffset, truncated, err := BoundedArtifactContent(artifact.Content, input.Offset, input.MaxBytes)
	if err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{artifactID})
	}
	contentLength := len(artifact.Content)
	return wire.ToolResult{
		ToolName:  call.Name,
		MissionID: missionID,
		Content: SourcesReadOutput{
			Snapshot:           SourceSnapshotFromApp(snapshot),
			Artifact:           server.ArtifactOutput(artifact),
			Content:            content,
			Offset:             offset,
			NextOffset:         nextOffset,
			ContentLength:      contentLength,
			ContentLengthKnown: true,
			Truncated:          truncated,
		},
	}
}

func (server *Handler) callSourcesReadMedia(
	ctx context.Context,
	toolName string,
	missionID string,
	snapshot sourcecontract.Snapshot,
) wire.ToolResult {
	locator, err := mediaLocatorFromSnapshot(snapshot)
	if err != nil {
		return server.ErrorFromErr(toolName, missionID, err, []string{snapshot.SnapshotID})
	}
	var artifactOutput wire.RawArtifactOutput
	if len(snapshot.ArtifactIDs) > 0 {
		artifact, err := server.Service.GetRawArtifact(ctx, snapshot.ArtifactIDs[0])
		if err != nil {
			return server.ErrorFromErr(toolName, missionID, err, []string{snapshot.SnapshotID, snapshot.ArtifactIDs[0]})
		}
		if artifact.MissionID != missionID {
			return server.ErrorResult(toolName, missionID, "validation", "media source artifact belongs to another mission", false, []string{artifact.ArtifactID})
		}
		artifactOutput = server.ArtifactOutput(artifact)
	}
	note := "Media source metadata is available. Binary bytes are not returned by MCP source reads."
	switch locator.MediaKind {
	case sourcecontract.MediaKindImage:
		note += " Image visual inspection is not available in this build until a real vision engine is configured."
	case sourcecontract.MediaKindAudio, sourcecontract.MediaKindVideo:
		note += " Audio and video inspection are unsupported in this build; add transcript, captions, keyframes, or other derived sources when needed."
	}
	return wire.ToolResult{
		ToolName:  toolName,
		MissionID: missionID,
		Content: MediaSourceReadOutput{
			Snapshot:       SourceSnapshotFromApp(snapshot),
			Artifact:       artifactOutput,
			Media:          locator,
			InspectionNote: note,
		},
	}
}

func (server *Handler) callSourcesReadLiveLocalPath(
	ctx context.Context,
	toolName string,
	missionID string,
	snapshot sourcecontract.Snapshot,
	input SourcesReadInput,
) wire.ToolResult {
	if snapshot.Connector.ConnectorType != sourcecontract.ConnectorTypeLocalPath {
		return server.ErrorResult(toolName, missionID, "validation", "live source connector is not supported by this tool", false, []string{snapshot.SnapshotID})
	}
	if strings.TrimSpace(input.ArtifactID) != "" {
		return server.ErrorResult(toolName, missionID, "validation", "live local path sources do not expose artifact_id reads", false, []string{snapshot.SnapshotID, input.ArtifactID})
	}
	producer, toolSessionID, err := server.ObservationProducer()
	if err != nil {
		return server.ErrorResult(toolName, missionID, "validation", err.Error(), false, []string{snapshot.SnapshotID})
	}
	result, err := server.Service.ReadLocalPathSource(ctx, app.ReadLocalPathSourceRequest{
		MissionID:     missionID,
		SnapshotID:    snapshot.SnapshotID,
		Subpath:       input.Subpath,
		Offset:        int64(input.Offset),
		MaxBytes:      int64(input.MaxBytes),
		Producer:      producer,
		ToolSessionID: toolSessionID,
	})
	if err != nil {
		return server.ErrorFromErr(toolName, missionID, err, []string{snapshot.SnapshotID})
	}
	metadata := result.Read.Metadata
	contentLength := int(metadata.Size)
	contentLengthKnown := true
	var extraction *wire.SourceExtractionOutput
	if strings.TrimSpace(metadata.Extraction) != "" {
		contentLength = int(metadata.TextLength)
		contentLengthKnown = metadata.TextLengthKnown
		extraction = &wire.SourceExtractionOutput{
			Type:               metadata.Extraction,
			PageCount:          metadata.PageCount,
			TextLength:         int(metadata.TextLength),
			TextLengthKnown:    metadata.TextLengthKnown,
			SuggestedReadBytes: pdfdocument.DefaultChunkMaxBytes,
			MaxReadBytes:       pdfdocument.MaxChunkBytes,
		}
	}
	return wire.ToolResult{
		ToolName:        toolName,
		MissionID:       missionID,
		CreatedEventIDs: observationEventIDs(result.ObservationEvent),
		Content: SourcesReadOutput{
			Snapshot:            SourceSnapshotFromApp(result.Snapshot),
			Content:             result.Read.Content,
			Offset:              input.Offset,
			NextOffset:          int(metadata.NextOffset),
			ContentLength:       contentLength,
			ContentLengthKnown:  contentLengthKnown,
			Truncated:           metadata.Truncated,
			Extraction:          extraction,
			ObservationMetadata: &metadata,
			ObservationEventID:  observationEventID(result.ObservationEvent),
		},
	}
}

func (server *Handler) CallSourcesTree(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input SourcesTreeInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := server.ValidateID("mis_", missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.EnforceMission(missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	snapshotID := strings.TrimSpace(input.SnapshotID)
	if err := server.ValidateID("src_", snapshotID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	snapshot, err := server.Service.GetSourceSnapshot(ctx, snapshotID)
	if err != nil {
		return server.ErrorFromErr(call.Name, missionID, err, []string{snapshotID})
	}
	if snapshot.MissionID != missionID {
		return server.ErrorResult(call.Name, missionID, "validation", "source snapshot belongs to another mission", false, []string{snapshotID})
	}
	if sourceState(snapshot).Removed {
		return server.ErrorResult(call.Name, missionID, "validation", "source snapshot is removed", false, []string{snapshotID})
	}
	if sourceRetrievalPolicy(snapshot) != sourcecontract.RetrievalPolicyLiveReference || snapshot.Connector.ConnectorType != sourcecontract.ConnectorTypeLocalPath {
		return server.ErrorResult(call.Name, missionID, "validation", "source snapshot is not a live local_path source", false, []string{snapshotID})
	}
	producer, toolSessionID, err := server.ObservationProducer()
	if err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{snapshotID})
	}
	result, err := server.Service.TreeLocalPathSource(ctx, app.TreeLocalPathSourceRequest{
		MissionID:     missionID,
		SnapshotID:    snapshotID,
		Subpath:       input.Subpath,
		Depth:         input.Depth,
		Limit:         input.Limit,
		Producer:      producer,
		ToolSessionID: toolSessionID,
	})
	if err != nil {
		return server.ErrorFromErr(call.Name, missionID, err, []string{snapshotID})
	}
	metadata := result.Tree.Metadata
	return wire.ToolResult{
		ToolName:        call.Name,
		MissionID:       missionID,
		CreatedEventIDs: observationEventIDs(result.ObservationEvent),
		Content: SourcesTreeOutput{
			Snapshot:            SourceSnapshotFromApp(result.Snapshot),
			Tree:                result.Tree,
			ObservationMetadata: &metadata,
			ObservationEventID:  observationEventID(result.ObservationEvent),
		},
	}
}

func (server *Handler) CallSourcesGrep(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input SourcesGrepInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := server.ValidateID("mis_", missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.EnforceMission(missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	snapshotID := strings.TrimSpace(input.SnapshotID)
	if err := server.ValidateID("src_", snapshotID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	snapshot, err := server.Service.GetSourceSnapshot(ctx, snapshotID)
	if err != nil {
		return server.ErrorFromErr(call.Name, missionID, err, []string{snapshotID})
	}
	if snapshot.MissionID != missionID {
		return server.ErrorResult(call.Name, missionID, "validation", "source snapshot belongs to another mission", false, []string{snapshotID})
	}
	if sourceState(snapshot).Removed {
		return server.ErrorResult(call.Name, missionID, "validation", "source snapshot is removed", false, []string{snapshotID})
	}
	if sourceRetrievalPolicy(snapshot) != sourcecontract.RetrievalPolicyLiveReference || snapshot.Connector.ConnectorType != sourcecontract.ConnectorTypeLocalPath {
		return server.ErrorResult(call.Name, missionID, "validation", "source snapshot is not a live local_path source", false, []string{snapshotID})
	}
	producer, toolSessionID, err := server.ObservationProducer()
	if err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{snapshotID})
	}
	result, err := server.Service.GrepLocalPathSource(ctx, app.GrepLocalPathSourceRequest{
		MissionID:     missionID,
		SnapshotID:    snapshotID,
		Subpath:       input.Subpath,
		Query:         input.Query,
		MaxSnippets:   input.MaxSnippets,
		Producer:      producer,
		ToolSessionID: toolSessionID,
	})
	if err != nil {
		return server.ErrorFromErr(call.Name, missionID, err, []string{snapshotID})
	}
	metadata := result.Grep.Metadata
	return wire.ToolResult{
		ToolName:        call.Name,
		MissionID:       missionID,
		CreatedEventIDs: observationEventIDs(result.ObservationEvent),
		Content: SourcesGrepOutput{
			Snapshot:            SourceSnapshotFromApp(result.Snapshot),
			Grep:                result.Grep,
			ObservationMetadata: &metadata,
			ObservationEventID:  observationEventID(result.ObservationEvent),
		},
	}
}

func (server *Handler) CallLocalPathRoots(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input LocalPathRootsInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if missionID == "" {
		missionID = strings.TrimSpace(server.MissionID())
	}
	if missionID != "" {
		if err := server.ValidateID("mis_", missionID); err != nil {
			return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
		}
		if err := server.EnforceMission(missionID); err != nil {
			return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
		}
	}
	roots, err := server.Service.ListLocalPathRoots(ctx)
	if err != nil {
		return server.ErrorFromErr(call.Name, missionID, err, nil)
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: LocalPathRootsOutput{Roots: roots}}
}

func (server *Handler) CallLocalPathTree(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input LocalPathTreeInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := server.ValidateID("mis_", missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.EnforceMission(missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateClientRelativePath(input.RelativePath); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	tree, err := server.Service.BrowseLocalPathRoot(ctx, app.BrowseLocalPathRootRequest{
		RootID:       input.RootID,
		RelativePath: input.RelativePath,
		Depth:        input.Depth,
		Limit:        input.Limit,
	})
	if err != nil {
		return server.ErrorFromErr(call.Name, missionID, err, []string{input.RootID, input.RelativePath})
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: LocalPathTreeOutput{Tree: tree}}
}

func (server *Handler) CallLocalPathAttach(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input LocalPathAttachInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, producer, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.RequireSession(common); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := validateClientRelativePath(input.RelativePath); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{input.RootID, input.RelativePath})
	}
	result, err := server.Service.AttachLocalPathSource(ctx, app.AttachLocalPathSourceRequest{
		MissionID:    common.MissionID,
		SnapshotID:   input.SnapshotID,
		RootID:       input.RootID,
		RelativePath: input.RelativePath,
		Title:        input.Title,
		Restore:      input.Restore,
		Producer:     producer,
	})
	if err != nil {
		return server.ErrorFromErr(call.Name, common.MissionID, err, []string{input.SnapshotID, input.RootID, input.RelativePath})
	}
	return wire.ToolResult{
		ToolName:        call.Name,
		MissionID:       common.MissionID,
		CreatedEventIDs: observationEventIDs(result.Event),
		Content: LocalPathAttachOutput{
			Snapshot:        SourceSnapshotFromApp(result.Snapshot),
			EventID:         observationEventID(result.Event),
			Existing:        result.Existing,
			Restored:        result.Restored,
			RestoreRequired: result.RestoreRequired,
		},
	}
}

func (server *Handler) CallSourcesRemove(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input SourceRemoveInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, producer, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.RequireSession(common); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	result, err := server.Service.RemoveSource(ctx, app.RemoveSourceRequest{
		MissionID:  common.MissionID,
		SnapshotID: input.SnapshotID,
		Reason:     input.Reason,
		Producer:   producer,
	})
	if err != nil {
		return server.ErrorFromErr(call.Name, common.MissionID, err, []string{input.SnapshotID})
	}
	return wire.ToolResult{
		ToolName:        call.Name,
		MissionID:       common.MissionID,
		CreatedEventIDs: observationEventIDs(result.Event),
		Content: SourceStateChangeOutput{
			Snapshot:   SourceSnapshotFromApp(result.Snapshot),
			EventID:    observationEventID(result.Event),
			Idempotent: result.Idempotent,
		},
	}
}

func (server *Handler) CallSourcesRestore(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input SourceRestoreInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, producer, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.RequireSession(common); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	result, err := server.Service.RestoreSource(ctx, app.RestoreSourceRequest{
		MissionID:  common.MissionID,
		SnapshotID: input.SnapshotID,
		Producer:   producer,
	})
	if err != nil {
		return server.ErrorFromErr(call.Name, common.MissionID, err, []string{input.SnapshotID})
	}
	return wire.ToolResult{
		ToolName:        call.Name,
		MissionID:       common.MissionID,
		CreatedEventIDs: observationEventIDs(result.Event),
		Content: SourceStateChangeOutput{
			Snapshot:   SourceSnapshotFromApp(result.Snapshot),
			EventID:    observationEventID(result.Event),
			Idempotent: result.Idempotent,
		},
	}
}

func (server *Handler) CallSourcesSearch(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input SourcesSearchInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := server.ValidateID("mis_", missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.EnforceMission(missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	connectorIDs := normalizeConnectors(input.Connectors)
	if len(connectorIDs) == 0 {
		connectorIDs = []string{liquid2source.Liquid2ConnectorID}
	}
	candidates := []SourceCandidateOutput{}
	nextCursors := map[string]string{}
	for _, connectorID := range connectorIDs {
		switch connectorID {
		case liquid2source.Liquid2ConnectorID:
			connector, ok := server.Liquid2Connector()
			if !ok {
				return server.ErrorResult(call.Name, missionID, "connector", "liquid2 connector is not mounted", true, nil)
			}
			result, err := server.Service.SearchLiquid2Sources(ctx, connector, liquid2source.Liquid2SourceSearchRequest{
				MissionID: missionID,
				Query:     input.Query,
				Limit:     input.Limit,
				Cursor:    input.Cursor,
			})
			if err != nil {
				return server.ErrorFromErr(call.Name, missionID, err, nil)
			}
			for _, candidate := range result.Candidates {
				candidates = append(candidates, SourceCandidateFromApp(candidate))
			}
			if strings.TrimSpace(result.NextCursor) != "" {
				nextCursors[connectorID] = strings.TrimSpace(result.NextCursor)
			}
		case confluencesource.ConfluenceConnectorID:
			access, err := server.Service.GetMissionConnectorAccess(ctx, missionID, confluencesource.ConfluenceConnectorID)
			if err != nil {
				return server.ErrorFromErr(call.Name, missionID, err, nil)
			}
			if !access.Enabled || access.Status != app.ConnectorAccessStatusEnabled {
				return server.ErrorResult(call.Name, missionID, "permission", "Confluence MCP search is disabled for this mission. Enable mission Confluence agent search in Plasma before using this connector.", false, nil)
			}
			inputConnectionID := strings.TrimSpace(input.ConnectionID)
			if inputConnectionID != "" && inputConnectionID != access.ConnectionID {
				return server.ErrorResult(call.Name, missionID, "permission", "Confluence connection_id does not match the mission grant.", false, nil)
			}
			inputCloudID := strings.TrimSpace(input.CloudID)
			if inputCloudID != "" && inputCloudID != access.CloudID {
				return server.ErrorResult(call.Name, missionID, "permission", "Confluence cloud_id does not match the mission grant.", false, nil)
			}
			spaceKey := strings.TrimSpace(input.SpaceKey)
			if access.SpaceKey != "" {
				if spaceKey != "" && spaceKey != access.SpaceKey {
					return server.ErrorResult(call.Name, missionID, "permission", "Confluence space_key does not match the mission grant.", false, nil)
				}
				spaceKey = access.SpaceKey
			}
			if server.ConfluenceFactory() == nil {
				return server.ErrorResult(call.Name, missionID, "connector", "confluence connector is not mounted", true, nil)
			}
			connector, err := server.ConfluenceFactory()(ctx, ConfluenceConnectorRequest{
				ConnectionID: access.ConnectionID,
				CloudID:      access.CloudID,
				SpaceKey:     spaceKey,
			})
			if err != nil {
				return server.ErrorFromErr(call.Name, missionID, err, nil)
			}
			result, err := server.Service.SearchConfluenceSources(ctx, connector, confluencesource.ConfluenceSourceSearchRequest{
				MissionID: missionID,
				CloudID:   access.CloudID,
				Query:     input.Query,
				Limit:     input.Limit,
				Cursor:    input.Cursor,
				SpaceKey:  spaceKey,
			})
			if err != nil {
				return server.ErrorFromErr(call.Name, missionID, err, nil)
			}
			for _, candidate := range result.Candidates {
				candidates = append(candidates, SourceCandidateFromConfluence(candidate))
			}
			if strings.TrimSpace(result.NextCursor) != "" {
				nextCursors[connectorID] = strings.TrimSpace(result.NextCursor)
			}
		default:
			return server.ErrorResult(call.Name, missionID, "connector", "unsupported connector", false, []string{connectorID})
		}
	}
	if len(nextCursors) == 0 {
		nextCursors = nil
	}
	return wire.ToolResult{
		ToolName:  call.Name,
		MissionID: missionID,
		Content:   SourcesSearchOutput{Candidates: candidates, NextCursors: nextCursors},
	}
}

func (server *Handler) CallSourcesSnapshot(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input SourcesSnapshotInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, producer, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	_ = producer
	return server.ApprovalRequired(
		call.Name,
		common.MissionID,
		"source snapshots require user approval and must be created by the approved application path",
		[]string{input.SnapshotID, input.ArtifactID, input.Connector.ExternalSourceID},
	)
}
