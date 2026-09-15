package mcp

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/reportil"
	sourcehandler "github.com/c86j224s/liquid2/plasma/internal/mcp/source"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/reportilsource"
	"strings"
)

func (server *Server) ilSourceHandler() reportil.SourceHandler {
	return reportil.SourceHandler{Documents: &server.reportILState.Documents, Editorial: &server.reportILState.Editorial, Artifacts: server.service, NewID: newMCPID, ErrorFromErr: errorFromErr, Mu: &server.mu, State: &server.reportILState.Source, MissionID: func() string { return server.binding.MissionID }, SessionID: func() string { return server.binding.AgentSessionID }, AccessBinding: server.reportILSourceAccessBinding, ReadableSource: server.reportILReadableSource, Decode: decodeArgs, ErrorResult: errorResult, SHA256: sha256Hex, BoundedRead: sourcehandler.BoundedArtifactContentWithLimit}
}
func (server *Server) callReportILSourcesList(call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILSourcesList(call)
}
func (server *Server) callReportILSourceQuote(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILSourceQuote(ctx, call)
}
func (server *Server) callReportILSourcesRead(ctx context.Context, call ToolCall) ToolResult {
	h := server.ilSourceHandler()
	return h.CallReportILSourcesRead(ctx, call)
}
func (server *Server) reportILSourceAccessBinding() (reportilcontract.SourceAccessBinding, error) {
	if !server.reportILSourceBindingSet {
		return reportilcontract.SourceAccessBinding{}, fmt.Errorf("report IL source binding is unavailable")
	}
	binding := server.reportILSourceBinding
	if err := reportilcontract.ValidateSourceAccessBinding(binding); err != nil {
		return reportilcontract.SourceAccessBinding{}, err
	}
	if strings.TrimSpace(server.binding.MissionID) != binding.Catalog.MissionID {
		return reportilcontract.SourceAccessBinding{}, fmt.Errorf("report IL source binding conflicts with mission binding")
	}
	if !strings.HasPrefix(strings.TrimSpace(server.binding.AgentSessionID), "ses_") || server.binding.AgentExecutor != "codex" {
		return reportilcontract.SourceAccessBinding{}, fmt.Errorf("report IL source binding requires a Codex attempt session")
	}
	return binding, nil
}

func (server *Server) reportILReadableSource(ctx context.Context, binding reportilcontract.SourceAccessBinding, entry reportilcontract.SourceCatalogEntry) (reportilsource.Readable, error) {
	return reportilsource.ResolveFrozenSource(ctx, reportilsource.FrozenSourceReaders{
		GetSourceSnapshot: server.service.GetSourceSnapshot,
		GetRawArtifact:    server.service.GetRawArtifact,
		ReadLiveSource: func(ctx context.Context, missionID, snapshotID string, maxBytes int) (reportilsource.LiveSourceRead, error) {
			result, err := server.service.ReadLocalPathSource(ctx, newReportILLiveSourceReadRequest(missionID, snapshotID, server.binding.AgentSessionID, maxBytes))
			return reportilsource.LiveSourceRead{Content: result.Read.Content, Extraction: result.Read.Metadata.Extraction, Binary: result.Read.Metadata.Binary, Truncated: result.Read.Metadata.Truncated}, err
		},
	}, binding, entry)
}
