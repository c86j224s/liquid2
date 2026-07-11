package mcp

import (
	"context"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/sources/pdftext"
)

func (server *Server) callSourcesList(ctx context.Context, call ToolCall) ToolResult {
	var input sourcesListInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceBoundMission(missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	snapshots, err := server.service.ListSourceSnapshotsWithState(ctx, app.ListSourceSnapshotsRequest{
		MissionID:         missionID,
		IncludeRemoved:    input.IncludeRemoved,
		IncludeSuperseded: input.IncludeSuperseded,
	})
	if err != nil {
		return errorFromErr(call.Name, missionID, err, nil)
	}
	return ToolResult{
		ToolName:  call.Name,
		MissionID: missionID,
		Content:   sourcesListOutput{Sources: sourceSnapshotsFromApp(snapshots)},
	}
}

func (server *Server) callSourcesRead(ctx context.Context, call ToolCall) ToolResult {
	var input sourcesReadInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceBoundMission(missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	snapshotID := strings.TrimSpace(input.SnapshotID)
	if err := validateID("src_", snapshotID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	snapshot, err := server.service.GetSourceSnapshot(ctx, snapshotID)
	if err != nil {
		return errorFromErr(call.Name, missionID, err, []string{snapshotID})
	}
	if snapshot.MissionID != missionID {
		return errorResult(call.Name, missionID, "validation", "source snapshot belongs to another mission", false, []string{snapshotID})
	}
	if sourceState(snapshot).Removed {
		return errorResult(call.Name, missionID, "validation", "source snapshot is removed", false, []string{snapshotID})
	}
	if strings.TrimSpace(input.Subpath) != "" && (sourceRetrievalPolicy(snapshot) != app.SourceRetrievalPolicyLiveReference || snapshot.Connector.ConnectorType != app.SourceConnectorTypeLocalPath) {
		return errorResult(call.Name, missionID, "validation", "subpath is only valid for live local_path sources", false, []string{snapshotID})
	}
	if snapshot.Connector.ConnectorType == app.SourceConnectorTypeMediaURL {
		return server.callSourcesReadMedia(ctx, call.Name, missionID, snapshot)
	}
	if sourceRetrievalPolicy(snapshot) == app.SourceRetrievalPolicyLiveReference {
		return server.callSourcesReadLiveLocalPath(ctx, call.Name, missionID, snapshot, input)
	}
	artifactID, err := selectedSnapshotArtifactID(snapshot, input.ArtifactID)
	if err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{snapshotID, input.ArtifactID})
	}
	artifact, err := server.service.GetRawArtifact(ctx, artifactID)
	if err != nil {
		return errorFromErr(call.Name, missionID, err, []string{snapshotID, artifactID})
	}
	if artifact.MissionID != missionID {
		return errorResult(call.Name, missionID, "validation", "source artifact belongs to another mission", false, []string{artifactID})
	}
	if pdftext.IsPDFMediaType(artifact.MediaType) || pdftext.IsPDFBytes(artifact.Content) {
		chunk, err := pdftext.ExtractChunk(artifact.Content, input.Offset, input.MaxBytes)
		if err != nil {
			return errorResult(call.Name, missionID, "validation", "PDF text extraction failed: "+err.Error(), false, []string{artifactID})
		}
		return ToolResult{
			ToolName:  call.Name,
			MissionID: missionID,
			Content: sourcesReadOutput{
				Snapshot:           sourceSnapshotFromApp(snapshot),
				Artifact:           rawArtifactFromApp(artifact),
				Content:            chunk.Text,
				Offset:             chunk.Offset,
				NextOffset:         chunk.NextOffset,
				ContentLength:      chunk.ContentLength,
				ContentLengthKnown: chunk.ContentLengthKnown,
				Truncated:          chunk.Truncated,
				Extraction: &sourceExtractionOutput{
					Type:               "pdf_text",
					PageCount:          chunk.PageCount,
					TextLength:         chunk.ContentLength,
					TextLengthKnown:    chunk.ContentLengthKnown,
					SuggestedReadBytes: pdftext.DefaultChunkMaxBytes,
					MaxReadBytes:       pdftext.MaxChunkBytes,
				},
			},
		}
	}
	if app.UploadedArtifactReadKind(artifact) == "metadata" {
		return ToolResult{
			ToolName:  call.Name,
			MissionID: missionID,
			Content: sourcesReadOutput{
				Snapshot:           sourceSnapshotFromApp(snapshot),
				Artifact:           rawArtifactFromApp(artifact),
				Content:            "",
				ContentLength:      int(artifact.ByteSize),
				ContentLengthKnown: true,
				MetadataOnly:       true,
			},
		}
	}
	content, offset, nextOffset, truncated, err := boundedArtifactContent(artifact.Content, input.Offset, input.MaxBytes)
	if err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{artifactID})
	}
	contentLength := len(artifact.Content)
	return ToolResult{
		ToolName:  call.Name,
		MissionID: missionID,
		Content: sourcesReadOutput{
			Snapshot:           sourceSnapshotFromApp(snapshot),
			Artifact:           rawArtifactFromApp(artifact),
			Content:            content,
			Offset:             offset,
			NextOffset:         nextOffset,
			ContentLength:      contentLength,
			ContentLengthKnown: true,
			Truncated:          truncated,
		},
	}
}

func (server *Server) callSourcesReadMedia(
	ctx context.Context,
	toolName string,
	missionID string,
	snapshot app.SourceSnapshot,
) ToolResult {
	locator, err := mediaLocatorFromSnapshot(snapshot)
	if err != nil {
		return errorFromErr(toolName, missionID, err, []string{snapshot.SnapshotID})
	}
	var artifactOutput rawArtifactOutput
	if len(snapshot.ArtifactIDs) > 0 {
		artifact, err := server.service.GetRawArtifact(ctx, snapshot.ArtifactIDs[0])
		if err != nil {
			return errorFromErr(toolName, missionID, err, []string{snapshot.SnapshotID, snapshot.ArtifactIDs[0]})
		}
		if artifact.MissionID != missionID {
			return errorResult(toolName, missionID, "validation", "media source artifact belongs to another mission", false, []string{artifact.ArtifactID})
		}
		artifactOutput = rawArtifactFromApp(artifact)
	}
	note := "Media source metadata is available. Binary bytes are not returned by MCP source reads."
	switch locator.MediaKind {
	case app.MediaKindImage:
		note += " Image visual inspection is not available in this build until a real vision engine is configured."
	case app.MediaKindAudio, app.MediaKindVideo:
		note += " Audio and video inspection are unsupported in this build; add transcript, captions, keyframes, or other derived sources when needed."
	}
	return ToolResult{
		ToolName:  toolName,
		MissionID: missionID,
		Content: mediaSourceReadOutput{
			Snapshot:       sourceSnapshotFromApp(snapshot),
			Artifact:       artifactOutput,
			Media:          locator,
			InspectionNote: note,
		},
	}
}

func (server *Server) callSourcesReadLiveLocalPath(
	ctx context.Context,
	toolName string,
	missionID string,
	snapshot app.SourceSnapshot,
	input sourcesReadInput,
) ToolResult {
	if snapshot.Connector.ConnectorType != app.SourceConnectorTypeLocalPath {
		return errorResult(toolName, missionID, "validation", "live source connector is not supported by this tool", false, []string{snapshot.SnapshotID})
	}
	if strings.TrimSpace(input.ArtifactID) != "" {
		return errorResult(toolName, missionID, "validation", "live local path sources do not expose artifact_id reads", false, []string{snapshot.SnapshotID, input.ArtifactID})
	}
	producer, toolSessionID, err := server.boundObservationProducer()
	if err != nil {
		return errorResult(toolName, missionID, "validation", err.Error(), false, []string{snapshot.SnapshotID})
	}
	result, err := server.service.ReadLocalPathSource(ctx, app.ReadLocalPathSourceRequest{
		MissionID:     missionID,
		SnapshotID:    snapshot.SnapshotID,
		Subpath:       input.Subpath,
		Offset:        int64(input.Offset),
		MaxBytes:      int64(input.MaxBytes),
		Producer:      producer,
		ToolSessionID: toolSessionID,
	})
	if err != nil {
		return errorFromErr(toolName, missionID, err, []string{snapshot.SnapshotID})
	}
	metadata := result.Read.Metadata
	contentLength := int(metadata.Size)
	contentLengthKnown := true
	var extraction *sourceExtractionOutput
	if strings.TrimSpace(metadata.Extraction) != "" {
		contentLength = int(metadata.TextLength)
		contentLengthKnown = metadata.TextLengthKnown
		extraction = &sourceExtractionOutput{
			Type:               metadata.Extraction,
			PageCount:          metadata.PageCount,
			TextLength:         int(metadata.TextLength),
			TextLengthKnown:    metadata.TextLengthKnown,
			SuggestedReadBytes: pdftext.DefaultChunkMaxBytes,
			MaxReadBytes:       pdftext.MaxChunkBytes,
		}
	}
	return ToolResult{
		ToolName:        toolName,
		MissionID:       missionID,
		CreatedEventIDs: observationEventIDs(result.ObservationEvent),
		Content: sourcesReadOutput{
			Snapshot:            sourceSnapshotFromApp(result.Snapshot),
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

func (server *Server) callSourcesTree(ctx context.Context, call ToolCall) ToolResult {
	var input sourcesTreeInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceBoundMission(missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	snapshotID := strings.TrimSpace(input.SnapshotID)
	if err := validateID("src_", snapshotID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	snapshot, err := server.service.GetSourceSnapshot(ctx, snapshotID)
	if err != nil {
		return errorFromErr(call.Name, missionID, err, []string{snapshotID})
	}
	if snapshot.MissionID != missionID {
		return errorResult(call.Name, missionID, "validation", "source snapshot belongs to another mission", false, []string{snapshotID})
	}
	if sourceState(snapshot).Removed {
		return errorResult(call.Name, missionID, "validation", "source snapshot is removed", false, []string{snapshotID})
	}
	if sourceRetrievalPolicy(snapshot) != app.SourceRetrievalPolicyLiveReference || snapshot.Connector.ConnectorType != app.SourceConnectorTypeLocalPath {
		return errorResult(call.Name, missionID, "validation", "source snapshot is not a live local_path source", false, []string{snapshotID})
	}
	producer, toolSessionID, err := server.boundObservationProducer()
	if err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{snapshotID})
	}
	result, err := server.service.TreeLocalPathSource(ctx, app.TreeLocalPathSourceRequest{
		MissionID:     missionID,
		SnapshotID:    snapshotID,
		Subpath:       input.Subpath,
		Depth:         input.Depth,
		Limit:         input.Limit,
		Producer:      producer,
		ToolSessionID: toolSessionID,
	})
	if err != nil {
		return errorFromErr(call.Name, missionID, err, []string{snapshotID})
	}
	metadata := result.Tree.Metadata
	return ToolResult{
		ToolName:        call.Name,
		MissionID:       missionID,
		CreatedEventIDs: observationEventIDs(result.ObservationEvent),
		Content: sourcesTreeOutput{
			Snapshot:            sourceSnapshotFromApp(result.Snapshot),
			Tree:                result.Tree,
			ObservationMetadata: &metadata,
			ObservationEventID:  observationEventID(result.ObservationEvent),
		},
	}
}

func (server *Server) callSourcesGrep(ctx context.Context, call ToolCall) ToolResult {
	var input sourcesGrepInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceBoundMission(missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	snapshotID := strings.TrimSpace(input.SnapshotID)
	if err := validateID("src_", snapshotID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	snapshot, err := server.service.GetSourceSnapshot(ctx, snapshotID)
	if err != nil {
		return errorFromErr(call.Name, missionID, err, []string{snapshotID})
	}
	if snapshot.MissionID != missionID {
		return errorResult(call.Name, missionID, "validation", "source snapshot belongs to another mission", false, []string{snapshotID})
	}
	if sourceState(snapshot).Removed {
		return errorResult(call.Name, missionID, "validation", "source snapshot is removed", false, []string{snapshotID})
	}
	if sourceRetrievalPolicy(snapshot) != app.SourceRetrievalPolicyLiveReference || snapshot.Connector.ConnectorType != app.SourceConnectorTypeLocalPath {
		return errorResult(call.Name, missionID, "validation", "source snapshot is not a live local_path source", false, []string{snapshotID})
	}
	producer, toolSessionID, err := server.boundObservationProducer()
	if err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{snapshotID})
	}
	result, err := server.service.GrepLocalPathSource(ctx, app.GrepLocalPathSourceRequest{
		MissionID:     missionID,
		SnapshotID:    snapshotID,
		Subpath:       input.Subpath,
		Query:         input.Query,
		MaxSnippets:   input.MaxSnippets,
		Producer:      producer,
		ToolSessionID: toolSessionID,
	})
	if err != nil {
		return errorFromErr(call.Name, missionID, err, []string{snapshotID})
	}
	metadata := result.Grep.Metadata
	return ToolResult{
		ToolName:        call.Name,
		MissionID:       missionID,
		CreatedEventIDs: observationEventIDs(result.ObservationEvent),
		Content: sourcesGrepOutput{
			Snapshot:            sourceSnapshotFromApp(result.Snapshot),
			Grep:                result.Grep,
			ObservationMetadata: &metadata,
			ObservationEventID:  observationEventID(result.ObservationEvent),
		},
	}
}

func (server *Server) callLocalPathRoots(ctx context.Context, call ToolCall) ToolResult {
	var input localPathRootsInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if missionID == "" {
		missionID = strings.TrimSpace(server.binding.MissionID)
	}
	if missionID != "" {
		if err := validateID("mis_", missionID); err != nil {
			return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
		}
		if err := server.enforceBoundMission(missionID); err != nil {
			return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
		}
	}
	roots, err := server.service.ListLocalPathRoots(ctx)
	if err != nil {
		return errorFromErr(call.Name, missionID, err, nil)
	}
	return ToolResult{ToolName: call.Name, MissionID: missionID, Content: localPathRootsOutput{Roots: roots}}
}

func (server *Server) callLocalPathTree(ctx context.Context, call ToolCall) ToolResult {
	var input localPathTreeInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceBoundMission(missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateClientRelativePath(input.RelativePath); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	tree, err := server.service.BrowseLocalPathRoot(ctx, app.BrowseLocalPathRootRequest{
		RootID:       input.RootID,
		RelativePath: input.RelativePath,
		Depth:        input.Depth,
		Limit:        input.Limit,
	})
	if err != nil {
		return errorFromErr(call.Name, missionID, err, []string{input.RootID, input.RelativePath})
	}
	return ToolResult{ToolName: call.Name, MissionID: missionID, Content: localPathTreeOutput{Tree: tree}}
}

func (server *Server) callLocalPathAttach(ctx context.Context, call ToolCall) ToolResult {
	var input localPathAttachInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, producer, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.requireBoundWriteSession(common); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := validateClientRelativePath(input.RelativePath); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{input.RootID, input.RelativePath})
	}
	result, err := server.service.AttachLocalPathSource(ctx, app.AttachLocalPathSourceRequest{
		MissionID:    common.MissionID,
		SnapshotID:   input.SnapshotID,
		RootID:       input.RootID,
		RelativePath: input.RelativePath,
		Title:        input.Title,
		Restore:      input.Restore,
		Producer:     producer,
	})
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{input.SnapshotID, input.RootID, input.RelativePath})
	}
	return ToolResult{
		ToolName:        call.Name,
		MissionID:       common.MissionID,
		CreatedEventIDs: observationEventIDs(result.Event),
		Content: localPathAttachOutput{
			Snapshot:        sourceSnapshotFromApp(result.Snapshot),
			EventID:         observationEventID(result.Event),
			Existing:        result.Existing,
			Restored:        result.Restored,
			RestoreRequired: result.RestoreRequired,
		},
	}
}

func (server *Server) callSourcesRemove(ctx context.Context, call ToolCall) ToolResult {
	var input sourceRemoveInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, producer, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.requireBoundWriteSession(common); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	result, err := server.service.RemoveSource(ctx, app.RemoveSourceRequest{
		MissionID:  common.MissionID,
		SnapshotID: input.SnapshotID,
		Reason:     input.Reason,
		Producer:   producer,
	})
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{input.SnapshotID})
	}
	return ToolResult{
		ToolName:        call.Name,
		MissionID:       common.MissionID,
		CreatedEventIDs: observationEventIDs(result.Event),
		Content: sourceStateChangeOutput{
			Snapshot:   sourceSnapshotFromApp(result.Snapshot),
			EventID:    observationEventID(result.Event),
			Idempotent: result.Idempotent,
		},
	}
}

func (server *Server) callSourcesRestore(ctx context.Context, call ToolCall) ToolResult {
	var input sourceRestoreInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, producer, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := server.requireBoundWriteSession(common); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	result, err := server.service.RestoreSource(ctx, app.RestoreSourceRequest{
		MissionID:  common.MissionID,
		SnapshotID: input.SnapshotID,
		Producer:   producer,
	})
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{input.SnapshotID})
	}
	return ToolResult{
		ToolName:        call.Name,
		MissionID:       common.MissionID,
		CreatedEventIDs: observationEventIDs(result.Event),
		Content: sourceStateChangeOutput{
			Snapshot:   sourceSnapshotFromApp(result.Snapshot),
			EventID:    observationEventID(result.Event),
			Idempotent: result.Idempotent,
		},
	}
}

func (server *Server) callSourcesSearch(ctx context.Context, call ToolCall) ToolResult {
	var input sourcesSearchInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceBoundMission(missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	connectorIDs := normalizeConnectors(input.Connectors)
	if len(connectorIDs) == 0 {
		connectorIDs = []string{app.Liquid2ConnectorID}
	}
	candidates := []sourceCandidateOutput{}
	nextCursors := map[string]string{}
	for _, connectorID := range connectorIDs {
		switch connectorID {
		case app.Liquid2ConnectorID:
			connector, ok := server.connectors[app.Liquid2ConnectorID]
			if !ok {
				return errorResult(call.Name, missionID, "connector", "liquid2 connector is not mounted", true, nil)
			}
			result, err := server.service.SearchLiquid2Sources(ctx, connector, app.Liquid2SourceSearchRequest{
				MissionID: missionID,
				Query:     input.Query,
				Limit:     input.Limit,
				Cursor:    input.Cursor,
			})
			if err != nil {
				return errorFromErr(call.Name, missionID, err, nil)
			}
			for _, candidate := range result.Candidates {
				candidates = append(candidates, sourceCandidateFromApp(candidate))
			}
			if strings.TrimSpace(result.NextCursor) != "" {
				nextCursors[connectorID] = strings.TrimSpace(result.NextCursor)
			}
		case app.ConfluenceConnectorID:
			access, err := server.service.GetMissionConnectorAccess(ctx, missionID, app.ConfluenceConnectorID)
			if err != nil {
				return errorFromErr(call.Name, missionID, err, nil)
			}
			if !access.Enabled || access.Status != app.ConnectorAccessStatusEnabled {
				return errorResult(call.Name, missionID, "permission", "Confluence MCP search is disabled for this mission. Enable mission Confluence agent search in Plasma before using this connector.", false, nil)
			}
			inputConnectionID := strings.TrimSpace(input.ConnectionID)
			if inputConnectionID != "" && inputConnectionID != access.ConnectionID {
				return errorResult(call.Name, missionID, "permission", "Confluence connection_id does not match the mission grant.", false, nil)
			}
			inputCloudID := strings.TrimSpace(input.CloudID)
			if inputCloudID != "" && inputCloudID != access.CloudID {
				return errorResult(call.Name, missionID, "permission", "Confluence cloud_id does not match the mission grant.", false, nil)
			}
			spaceKey := strings.TrimSpace(input.SpaceKey)
			if access.SpaceKey != "" {
				if spaceKey != "" && spaceKey != access.SpaceKey {
					return errorResult(call.Name, missionID, "permission", "Confluence space_key does not match the mission grant.", false, nil)
				}
				spaceKey = access.SpaceKey
			}
			if server.confluenceConnectorFactory == nil {
				return errorResult(call.Name, missionID, "connector", "confluence connector is not mounted", true, nil)
			}
			connector, err := server.confluenceConnectorFactory(ctx, ConfluenceConnectorRequest{
				ConnectionID: access.ConnectionID,
				CloudID:      access.CloudID,
				SpaceKey:     spaceKey,
			})
			if err != nil {
				return errorFromErr(call.Name, missionID, err, nil)
			}
			result, err := server.service.SearchConfluenceSources(ctx, connector, app.ConfluenceSourceSearchRequest{
				MissionID: missionID,
				CloudID:   access.CloudID,
				Query:     input.Query,
				Limit:     input.Limit,
				Cursor:    input.Cursor,
				SpaceKey:  spaceKey,
			})
			if err != nil {
				return errorFromErr(call.Name, missionID, err, nil)
			}
			for _, candidate := range result.Candidates {
				candidates = append(candidates, sourceCandidateFromConfluence(candidate))
			}
			if strings.TrimSpace(result.NextCursor) != "" {
				nextCursors[connectorID] = strings.TrimSpace(result.NextCursor)
			}
		default:
			return errorResult(call.Name, missionID, "connector", "unsupported connector", false, []string{connectorID})
		}
	}
	if len(nextCursors) == 0 {
		nextCursors = nil
	}
	return ToolResult{
		ToolName:  call.Name,
		MissionID: missionID,
		Content:   sourcesSearchOutput{Candidates: candidates, NextCursors: nextCursors},
	}
}

func (server *Server) callSourcesSnapshot(ctx context.Context, call ToolCall) ToolResult {
	var input sourcesSnapshotInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, producer, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	_ = producer
	return approvalRequiredResult(
		call.Name,
		common.MissionID,
		"source snapshots require user approval and must be created by the approved application path",
		[]string{input.SnapshotID, input.ArtifactID, input.Connector.ExternalSourceID},
	)
}
