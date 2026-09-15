package mcp

import (
	"context"
	"encoding/json"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/reportil"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"strings"
	"testing"
)

func TestReportILEditorialMemoryWorkspaceRequiresLatestCompleteReread(t *testing.T) {
	service, binding, accountText := reportILSourceToolFixture(t, "Yamana Sōzen commissioned the castle in 1433 and had it built over thirteen years.", 256, 256)
	binding.Stage = "il_editorial_memory"
	server := newReportILSourceToolServer(service, binding)

	beforeSourceRead := server.Call(context.Background(), ToolCall{
		Name:      ToolReportILEditorialMemoryStart,
		Arguments: mustArgs(t, map[string]any{"language": "en"}),
	})
	if beforeSourceRead.Error == nil || !strings.Contains(beforeSourceRead.Error.Message, "complete frozen catalog read") {
		t.Fatalf("memory start before source read = %#v", beforeSourceRead)
	}
	if read := server.Call(context.Background(), ToolCall{Name: ToolReportILSourcesRead, Arguments: json.RawMessage(`{}`)}); read.Error != nil {
		t.Fatalf("source read = %#v", read)
	}
	started := server.Call(context.Background(), ToolCall{
		Name:      ToolReportILEditorialMemoryStart,
		Arguments: mustArgs(t, map[string]any{"language": "en"}),
	})
	if started.Error != nil {
		t.Fatalf("memory start = %#v", started)
	}
	workspaceID := started.Content.(reportILEditorialMemoryStateOutput).WorkspaceID
	appendAccount := func(account string) {
		t.Helper()
		registered := server.Call(context.Background(), ToolCall{
			Name: ToolReportILSourcesQuote,
			Arguments: mustArgs(t, map[string]any{
				"source_key": "source_001", "quote": accountText,
			}),
		})
		if registered.Error != nil {
			t.Fatalf("source anchor = %#v", registered)
		}
		result := server.Call(context.Background(), ToolCall{
			Name: ToolReportILEditorialMemoryAppend,
			Arguments: mustArgs(t, map[string]any{
				"workspace_id":   workspaceID,
				"importance":     "essential",
				"account":        account,
				"source_keys":    []string{"source_001"},
				"source_anchors": []string{registered.Content.(reportILSourceQuoteOutput).SourceReceipt},
			}),
		})
		if result.Error != nil {
			t.Fatalf("memory append = %#v", result)
		}
	}
	appendAccount(accountText)

	beforeReread := server.Call(context.Background(), ToolCall{
		Name:      ToolReportILEditorialMemoryFinalize,
		Arguments: mustArgs(t, map[string]any{"workspace_id": workspaceID}),
	})
	if beforeReread.Error == nil || !strings.Contains(beforeReread.Error.Message, "reread completely") {
		t.Fatalf("memory finalize before reread = %#v", beforeReread)
	}
	read := server.Call(context.Background(), ToolCall{
		Name: ToolReportILEditorialMemoryRead,
		Arguments: mustArgs(t, map[string]any{
			"workspace_id": workspaceID,
			"offset":       0,
			"max_bytes":    reportil.ReportILDocumentMaxReadBytes,
		}),
	})
	if read.Error != nil || read.Content.(reportILEditorialMemoryReadOutput).Truncated {
		t.Fatalf("memory reread = %#v", read)
	}
	appendAccount("The same museum account identifies Sōzen as Tajima's provincial governor.")
	stale := server.Call(context.Background(), ToolCall{
		Name:      ToolReportILEditorialMemoryFinalize,
		Arguments: mustArgs(t, map[string]any{"workspace_id": workspaceID}),
	})
	if stale.Error == nil || !strings.Contains(stale.Error.Message, "reread completely") {
		t.Fatalf("memory finalize after append without reread = %#v", stale)
	}
	wrongOffset := server.Call(context.Background(), ToolCall{
		Name: ToolReportILEditorialMemoryRead,
		Arguments: mustArgs(t, map[string]any{
			"workspace_id": workspaceID,
			"offset":       1,
			"max_bytes":    reportil.ReportILDocumentMaxReadBytes,
		}),
	})
	if wrongOffset.Error == nil || !strings.Contains(wrongOffset.Error.Message, "start at offset 0") {
		t.Fatalf("memory reread accepted wrong offset = %#v", wrongOffset)
	}
	if reread := server.Call(context.Background(), ToolCall{
		Name: ToolReportILEditorialMemoryRead,
		Arguments: mustArgs(t, map[string]any{
			"workspace_id": workspaceID,
			"offset":       0,
			"max_bytes":    reportil.ReportILDocumentMaxReadBytes,
		}),
	}); reread.Error != nil || reread.Content.(reportILEditorialMemoryReadOutput).Truncated {
		t.Fatalf("latest memory reread = %#v", reread)
	}
	finalized := server.Call(context.Background(), ToolCall{
		Name:      ToolReportILEditorialMemoryFinalize,
		Arguments: mustArgs(t, map[string]any{"workspace_id": workspaceID}),
	})
	state := finalized.Content.(reportILEditorialMemoryStateOutput)
	if finalized.Error != nil || !state.Finalized || state.Accounts != 2 || !strings.HasPrefix(state.ArtifactID, "art_") || len(state.SHA256) != 64 {
		t.Fatalf("memory finalize = %#v", finalized)
	}
	duplicate := server.Call(context.Background(), ToolCall{
		Name:      ToolReportILEditorialMemoryFinalize,
		Arguments: mustArgs(t, map[string]any{"workspace_id": workspaceID}),
	})
	if duplicate.Error == nil || duplicate.Error.ErrorKind != "conflict" {
		t.Fatalf("duplicate memory finalize = %#v", duplicate)
	}
	for _, event := range service.events {
		trace := string(event.Payload)
		if !strings.Contains(trace, "plasma.report_il.memory.") {
			continue
		}
		for _, forbidden := range []string{accountText, "same museum account", "Tajima's provincial governor", "source_001"} {
			if strings.Contains(strings.ToLower(trace), strings.ToLower(forbidden)) {
				t.Fatalf("memory trace leaked %q: %s", forbidden, trace)
			}
		}
	}
}

func TestReportILEditorialMemoryPersistsExactCausativeAnchor(t *testing.T) {
	const sourceText = "1433（嘉吉3）年に但馬国守護山名宗全（そうぜん）が13年かけて築かせた城といわれている。"
	const causative = "山名宗全（そうぜん）が13年かけて築かせた"
	service, binding, _ := reportILSourceToolFixture(t, sourceText, 256, 256)
	binding.Stage = "il_editorial_memory"
	server := newReportILSourceToolServer(service, binding)
	if read := server.Call(context.Background(), ToolCall{Name: ToolReportILSourcesRead, Arguments: json.RawMessage(`{}`)}); read.Error != nil {
		t.Fatalf("source read = %#v", read)
	}
	quote := server.Call(context.Background(), ToolCall{
		Name: ToolReportILSourcesQuote,
		Arguments: mustArgs(t, map[string]any{
			"source_key": "source_001", "quote": causative,
		}),
	})
	if quote.Error != nil {
		t.Fatalf("source quote = %#v", quote)
	}
	started := server.Call(context.Background(), ToolCall{
		Name:      ToolReportILEditorialMemoryStart,
		Arguments: mustArgs(t, map[string]any{"language": "ko"}),
	})
	workspaceID := started.Content.(reportILEditorialMemoryStateOutput).WorkspaceID
	appended := server.Call(context.Background(), ToolCall{
		Name: ToolReportILEditorialMemoryAppend,
		Arguments: mustArgs(t, map[string]any{
			"workspace_id": workspaceID, "importance": "essential",
			"account":        "1433년 다지마 수호 야마나 소젠이 성을 직접 쌓은 것이 아니라 13년에 걸쳐 쌓게 했다고 전한다.",
			"source_keys":    []string{"source_001"},
			"source_anchors": []string{quote.Content.(reportILSourceQuoteOutput).SourceReceipt},
		}),
	})
	if appended.Error != nil {
		t.Fatalf("memory append = %#v", appended)
	}
	read := server.Call(context.Background(), ToolCall{
		Name: ToolReportILEditorialMemoryRead,
		Arguments: mustArgs(t, map[string]any{
			"workspace_id": workspaceID, "offset": 0, "max_bytes": reportil.ReportILDocumentMaxReadBytes,
		}),
	})
	if read.Error != nil || !strings.Contains(read.Content.(reportILEditorialMemoryReadOutput).Content, causative) {
		t.Fatalf("memory read lost causative anchor = %#v", read)
	}
	finalized := server.Call(context.Background(), ToolCall{
		Name:      ToolReportILEditorialMemoryFinalize,
		Arguments: mustArgs(t, map[string]any{"workspace_id": workspaceID}),
	})
	if finalized.Error != nil {
		t.Fatalf("memory finalize = %#v", finalized)
	}
	stored, err := service.GetRawArtifact(context.Background(), finalized.Content.(reportILEditorialMemoryStateOutput).ArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	var artifact reportilcontract.EditorialMemoryArtifact
	if err := json.Unmarshal(stored.Content, &artifact); err != nil {
		t.Fatal(err)
	}
	if len(artifact.Anchors) != 1 || artifact.Anchors[0].Excerpt != causative ||
		artifact.Anchors[0].SHA256 != sha256Hex([]byte(causative)) {
		t.Fatalf("persisted causative anchor = %#v", artifact.Anchors)
	}
	for _, event := range service.events {
		trace := string(event.Payload)
		if strings.Contains(trace, sourceText) || strings.Contains(trace, causative) {
			t.Fatalf("source trace leaked exact anchor text: %s", trace)
		}
	}
}

func TestReportILEditorialMemoryRejectsMissingMismatchedOrConsumedAnchor(t *testing.T) {
	const firstText = "Alpha says the governor had the castle built."
	service, binding, _ := reportILSourceToolFixture(t, firstText, 256, 256)
	secondText := []byte("Beta gives a different account.")
	secondHash := sha256Hex(secondText)
	binding.Catalog.Sources = append(binding.Catalog.Sources, reportilcontract.SourceCatalogEntry{
		SourceKey: "source_002", SnapshotID: "src_second",
		SnapshotReceipt: reportilcontract.SourceSnapshotReceipt("src_second", secondHash),
		ContentHash:     secondHash, RetrievalPolicy: sourcecontract.RetrievalPolicySnapshotOnly,
		Artifacts: []reportilcontract.SourceCatalogArtifact{{
			ArtifactID: "art_second", SHA256: secondHash, ByteSize: int64(len(secondText)), MediaType: "text/plain",
		}},
		ReadableSHA256: secondHash, ReadableBytes: len(secondText), Extraction: "stored_text",
	})
	var err error
	binding.Catalog, err = reportilcontract.SealSourceCatalog(binding.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	service.sources = append(service.sources, sourcecontract.Snapshot{
		SnapshotID: "src_second", MissionID: binding.Catalog.MissionID, ArtifactIDs: []string{"art_second"},
		ContentHash: sourcecontract.ContentHash{Algorithm: "sha256", Value: secondHash},
		Access:      sourcecontract.Access{RetrievalPolicy: sourcecontract.RetrievalPolicySnapshotOnly},
		State:       sourcecontract.State{State: sourcecontract.StateActive},
	})
	service.artifacts["art_second"] = artifactcontract.Raw{
		ArtifactID: "art_second", MissionID: binding.Catalog.MissionID, MediaType: "text/plain",
		ByteSize: int64(len(secondText)), SHA256: secondHash, Content: secondText,
	}
	binding.Stage = "il_editorial_memory"
	server := newReportILSourceToolServer(service, binding)
	if read := server.Call(context.Background(), ToolCall{Name: ToolReportILSourcesRead, Arguments: json.RawMessage(`{}`)}); read.Error != nil {
		t.Fatalf("source read = %#v", read)
	}
	quote := server.Call(context.Background(), ToolCall{
		Name:      ToolReportILSourcesQuote,
		Arguments: mustArgs(t, map[string]any{"source_key": "source_001", "quote": "had the castle built"}),
	})
	if quote.Error != nil {
		t.Fatalf("source quote = %#v", quote)
	}
	receipt := quote.Content.(reportILSourceQuoteOutput).SourceReceipt
	started := server.Call(context.Background(), ToolCall{
		Name:      ToolReportILEditorialMemoryStart,
		Arguments: mustArgs(t, map[string]any{"language": "en"}),
	})
	workspaceID := started.Content.(reportILEditorialMemoryStateOutput).WorkspaceID
	appendAccount := func(sourceKeys, anchors []string) ToolResult {
		return server.Call(context.Background(), ToolCall{
			Name: ToolReportILEditorialMemoryAppend,
			Arguments: mustArgs(t, map[string]any{
				"workspace_id": workspaceID, "importance": "essential", "account": "The governor had the castle built.",
				"source_keys": sourceKeys, "source_anchors": anchors,
			}),
		})
	}
	for name, result := range map[string]ToolResult{
		"source without anchor":    appendAccount([]string{"source_001", "source_002"}, []string{receipt}),
		"receipt for wrong source": appendAccount([]string{"source_002"}, []string{receipt}),
	} {
		if result.Error == nil || result.Error.ErrorKind != "validation" {
			t.Fatalf("%s accepted = %#v", name, result)
		}
	}
	accepted := appendAccount([]string{"source_001"}, []string{receipt})
	if accepted.Error != nil {
		t.Fatalf("valid anchor rejected = %#v", accepted)
	}
	consumed := appendAccount([]string{"source_001"}, []string{receipt})
	if consumed.Error == nil || consumed.Error.ErrorKind != "validation" {
		t.Fatalf("consumed anchor reused = %#v", consumed)
	}
	if got := len(server.reportILState.Editorial.Workspaces[workspaceID].Memory.Accounts); got != 1 {
		t.Fatalf("invalid append mutated account count = %d", got)
	}
}

func TestReportILEditorialMemoryDownstreamReadRequiresBoundHashAndIsReadOnly(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "Alpha fact supports the report.", 64, 64)
	bindSingleSourceReportILEditorialMemory(t, service, &binding)
	binding.Stage = "il_narrative"
	binding.MaxCallBytes = 0
	binding.MaxReadBytes = 0
	server := newReportILSourceToolServer(service, binding)

	first := server.Call(context.Background(), ToolCall{
		Name: ToolReportILEditorialMemoryRead,
		Arguments: mustArgs(t, map[string]any{
			"offset": 0, "max_bytes": reportil.ReportILDocumentMaxReadBytes,
		}),
	})
	if first.Error != nil || first.Content.(reportILEditorialMemoryReadOutput).ReportILStage != "il_narrative" {
		t.Fatalf("bound downstream memory read = %#v", first)
	}
	for _, mutation := range []ToolCall{
		{Name: ToolReportILEditorialMemoryStart, Arguments: mustArgs(t, map[string]any{"language": "en"})},
		{Name: ToolReportILEditorialMemoryAppend, Arguments: mustArgs(t, map[string]any{
			"workspace_id": "ilm_forbidden", "importance": "essential", "account": "Forbidden", "source_keys": []string{"source_001"},
		})},
		{Name: ToolReportILEditorialMemoryFinalize, Arguments: mustArgs(t, map[string]any{"workspace_id": "ilm_forbidden"})},
	} {
		result := server.Call(context.Background(), mutation)
		if result.Error == nil || !strings.Contains(result.Error.Message, "not enabled") {
			t.Fatalf("downstream stage reached memory mutation %s: %#v", mutation.Name, result)
		}
	}

	tampered := service.artifacts[binding.EditorialMemoryArtifactID]
	tampered.Content = append([]byte(nil), tampered.Content...)
	tampered.Content[len(tampered.Content)-2] ^= 1
	service.artifacts[tampered.ArtifactID] = tampered
	fresh := newReportILSourceToolServer(service, binding)
	read := fresh.Call(context.Background(), ToolCall{
		Name: ToolReportILEditorialMemoryRead,
		Arguments: mustArgs(t, map[string]any{
			"offset": 0, "max_bytes": reportil.ReportILDocumentMaxReadBytes,
		}),
	})
	if read.Error == nil || !strings.Contains(read.Error.Message, "artifact binding is invalid") {
		t.Fatalf("tampered memory bytes were accepted = %#v", read)
	}
}

func TestReportILEditorialMemoryRejectsDuplicateOrUnavailableAccountSources(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "Alpha fact supports the report.", 64, 64)
	binding.Stage = "il_editorial_memory"
	server := newReportILSourceToolServer(service, binding)
	if read := server.Call(context.Background(), ToolCall{Name: ToolReportILSourcesRead, Arguments: json.RawMessage(`{}`)}); read.Error != nil {
		t.Fatalf("source read = %#v", read)
	}
	started := server.Call(context.Background(), ToolCall{
		Name:      ToolReportILEditorialMemoryStart,
		Arguments: mustArgs(t, map[string]any{"language": "en"}),
	})
	workspaceID := started.Content.(reportILEditorialMemoryStateOutput).WorkspaceID
	registered := server.Call(context.Background(), ToolCall{
		Name: ToolReportILSourcesQuote,
		Arguments: mustArgs(t, map[string]any{
			"source_key": "source_001", "quote": "Alpha fact supports the report.",
		}),
	})
	if registered.Error != nil {
		t.Fatalf("source anchor = %#v", registered)
	}
	for name, sourceKeys := range map[string][]string{
		"duplicate":   {"source_001", "source_001"},
		"unavailable": {"source_999"},
	} {
		t.Run(name, func(t *testing.T) {
			result := server.Call(context.Background(), ToolCall{
				Name: ToolReportILEditorialMemoryAppend,
				Arguments: mustArgs(t, map[string]any{
					"workspace_id":   workspaceID,
					"importance":     "essential",
					"account":        "Alpha fact remains connected.",
					"source_keys":    sourceKeys,
					"source_anchors": []string{registered.Content.(reportILSourceQuoteOutput).SourceReceipt},
				}),
			})
			if result.Error == nil || result.Error.ErrorKind != "validation" {
				t.Fatalf("invalid account sources accepted = %#v", result)
			}
		})
	}
	if len(server.reportILState.Editorial.Workspaces[workspaceID].Memory.Accounts) != 0 {
		t.Fatal("invalid memory append mutated workspace")
	}
}
