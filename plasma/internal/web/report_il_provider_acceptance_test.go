package web

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentcapability"
	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	plasmamcp "github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/pdfdocument"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/reportilphase0"
	"github.com/c86j224s/liquid2/plasma/internal/reportpipeline"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

const (
	reportILProviderAcceptanceGateEnv = "PLASMA_REPORT_IL_PROVIDER_ACCEPTANCE"
	reportILProviderAcceptanceDirEnv  = "PLASMA_REPORT_IL_ACCEPTANCE_DIR"
	reportILProviderCodexCommandEnv   = "PLASMA_REPORT_IL_CODEX_COMMAND"
)

const reportILProviderAcceptanceDirection = "Write a concise architecture and internal-dogfood assessment grounded only in the accepted source. Include one compact contract table."

const reportILProviderAcceptanceSource = `Experimental IL product acceptance facts

The independent report pipeline is opt-in and does not replace classic planned or long-form reports. When the accepted source set exceeds the complete-read budget or contains unusable entries, one source-selection call ranks the candidate catalog. The editorial-memory author then reads every selected source completely and records material connected accounts before prose compression. Each account preserves related actors, roles, actions, dates, durations, cases, relationships, and uncertainty together, marks its editorial importance, and carries the source keys that directly support it.

The report author reads that finalized server-owned editorial memory rather than the raw source bodies. It writes the complete publishable manuscript in a server-owned MCP document workspace and binds each reader-facing block to the editorial accounts it realizes. The server derives the block's source keys and citations from those account bindings, so citation lineage cannot silently diverge from the connected facts used in the prose.

A publication reader first reads the complete editorial memory as a meaning boundary, opens the author artifact, and may make prose-only replacements for concrete reading defects without changing account or source bindings. A final continuity editor then reads that publication-edited artifact together with every connected account and its server-verified exact frozen-source anchors. Its MCP surface permits only account-aware local block revision, so it fact-checks sentences and restores a subtly lost or distorted relationship with the smallest possible wording correction rather than rewriting sound paragraphs. The memory author, report author, publication reader, and final continuity editor each finalize a hash-bound server-owned artifact; provider terminal output does not carry the manuscript.

The server compiles the final continuity artifact while preserving its title, section order, paragraphs, lists, quotations, callouts, code, tables, and block-level citation bindings, adding stable node identities and citation metadata. Markdown, static self-contained HTML, and Chrome PDF are rendered directly from that same semantic document.

Successful completion stores exactly six product artifacts: a compact server-derived narrative contract, semantic IL JSON, Markdown, static self-contained HTML, Chrome PDF, and a manifest. The manifest records content-free identity, hash, size, revision, and account-count lineage for the editorial memory without embedding its prose. Markdown is the final artifact; HTML and PDF are deterministic content projections of the same semantic document, subject to Chrome PDF byte variability.

The source-selection, editorial-memory, authoring, publication, and final-continuity contracts fix Codex, gpt-5.6-luna, xhigh reasoning, a fresh ephemeral session, ignored user configuration, and disabled post-report style processing. Source bodies enter only the editorial-memory stage through frozen list and read tools rather than prompts. After the complete read, exact bounded excerpts are registered and reconstructed by the server inside editorial memory; downstream authoring and editing stages have no raw-source tools. The editorial-memory author, report author, publication reader, and final continuity editor each make exactly one provider call and have no semantic repair retry. Unsupported visual-asset requests fail closed. There is no silent fallback to a classic writer and no experimental retry action.

Acceptance should explain the architecture itself, summarize the fixed contract in a compact table, identify the fail-closed boundaries, and end with a cautious internal-dogfood recommendation rather than a default-migration claim.`

type reportILProviderCallReceipt struct {
	Ordinal            int      `json:"ordinal"`
	Stage              string   `json:"stage"`
	Attempt            int      `json:"attempt"`
	Model              string   `json:"model"`
	ReasoningEffort    string   `json:"reasoning_effort"`
	CapabilityProfile  string   `json:"capability_profile"`
	ProfileRevision    string   `json:"profile_revision"`
	MCPMode            string   `json:"mcp_mode"`
	DisableTools       bool     `json:"disable_tools"`
	IgnoreUserConfig   bool     `json:"ignore_user_config"`
	EphemeralSession   bool     `json:"ephemeral_session"`
	ReplaceMCPTools    bool     `json:"replace_mcp_tools"`
	ExtraMCPTools      []string `json:"extra_mcp_tools"`
	SourceBinding      bool     `json:"source_binding"`
	SourceCatalogSHA   string   `json:"source_catalog_sha256,omitempty"`
	SourceStage        string   `json:"source_stage,omitempty"`
	SourceAttempt      int      `json:"source_attempt,omitempty"`
	OutputSchemaSHA256 string   `json:"output_schema_sha256"`
	OutputSchemaBytes  int      `json:"output_schema_bytes"`
	ObservedToolEvents int      `json:"observed_tool_events"`
	ArticleContract    bool     `json:"article_contract,omitempty"`
}

type reportILTransportReceipt struct {
	Ordinal          int
	Ephemeral        bool
	IgnoreUserConfig bool
	JSON             bool
	OutputSchema     bool
	SchemaSHA256     string
	SchemaBytes      int
	ReadOnlySandbox  bool
	SkipGitCheck     bool
	IgnoreRules      bool
	Stdin            bool
	Model            string
	Effort           string
	ProfileConfigs   int
	MCPConfigs       int
}

type reportILArtifactReceipt struct {
	Kind      string `json:"kind"`
	Artifact  string `json:"artifact_id"`
	MediaType string `json:"media_type"`
	Filename  string `json:"filename"`
	Role      string `json:"role"`
	SHA256    string `json:"sha256"`
	ByteSize  int    `json:"byte_size"`
}

type reportILProviderAcceptanceReceipt struct {
	SchemaVersion       string                        `json:"schema_version"`
	Status              string                        `json:"status"`
	MissionID           string                        `json:"mission_id"`
	PendingEventID      string                        `json:"pending_event_id"`
	SourceSHA256        string                        `json:"source_sha256"`
	Provider            string                        `json:"provider"`
	Model               string                        `json:"model"`
	ReasoningEffort     string                        `json:"reasoning_effort"`
	ProviderCallCount   int                           `json:"provider_call_count"`
	Calls               []reportILProviderCallReceipt `json:"calls"`
	TerminalEventType   string                        `json:"terminal_event_type"`
	CompletionEventType string                        `json:"completion_event_type,omitempty"`
	FailedStage         string                        `json:"failed_stage,omitempty"`
	SafeErrorClass      string                        `json:"safe_error_class,omitempty"`
	SafeFailureReason   string                        `json:"safe_failure_reason,omitempty"`
	ArtifactCount       int                           `json:"artifact_count"`
	Artifacts           []reportILArtifactReceipt     `json:"artifacts,omitempty"`
	PDFPageCount        int                           `json:"pdf_page_count,omitempty"`
	PrivacyScanPassed   bool                          `json:"privacy_scan_passed"`
	NoSilentFallback    bool                          `json:"no_silent_fallback"`
	SourceReadTraced    bool                          `json:"source_read_traced"`
	SourceMCPConfigured bool                          `json:"source_mcp_configured"`
	TransportVerified   bool                          `json:"transport_verified"`
}

type reportILAcceptanceExecutor struct {
	mu       sync.Mutex
	delegate agentexec.AgentExecutor
	calls    []reportILProviderCallReceipt
	attempts map[string]int
}

func (executor *reportILAcceptanceExecutor) Run(ctx context.Context, req agentexec.AgentRequest) (agentexec.AgentResult, error) {
	stage := strings.TrimPrefix(strings.TrimSpace(req.UserText), "report IL ")
	schemaHash := sha256.Sum256(req.OutputJSONSchema)
	executor.mu.Lock()
	if executor.attempts == nil {
		executor.attempts = map[string]int{}
	}
	executor.attempts[stage]++
	call := reportILProviderCallReceipt{
		Ordinal: len(executor.calls) + 1, Stage: stage, Attempt: executor.attempts[stage],
		Model: req.Model, ReasoningEffort: req.ReasoningEffort,
		CapabilityProfile: string(req.CapabilityProfile), ProfileRevision: req.ProfileRevision,
		MCPMode: req.MCPMode, DisableTools: req.DisableTools,
		IgnoreUserConfig: req.IgnoreUserConfig, EphemeralSession: req.EphemeralSession,
		ReplaceMCPTools: req.ReplaceMCPTools, ExtraMCPTools: append([]string(nil), req.ExtraMCPTools...),
		OutputSchemaSHA256: hex.EncodeToString(schemaHash[:]), OutputSchemaBytes: len(req.OutputJSONSchema),
		ArticleContract: strings.Contains(req.Prompt, "ARTICLE CONTRACT") && strings.Contains(req.Prompt, "one continuous reader journey"),
	}
	if req.ReportILSources != nil {
		call.SourceBinding = true
		call.SourceCatalogSHA = req.ReportILSources.Catalog.SHA256
		call.SourceStage = req.ReportILSources.Stage
		call.SourceAttempt = req.ReportILSources.Attempt
	}
	executor.calls = append(executor.calls, call)
	callIndex := len(executor.calls) - 1
	executor.mu.Unlock()

	result, err := executor.runDelegate(ctx, req, func(observation agentexec.AgentObservation) {
		if observation.Type != agentexec.AgentObservationTool {
			return
		}
		executor.mu.Lock()
		executor.calls[callIndex].ObservedToolEvents++
		executor.mu.Unlock()
	})
	return result, err
}

func (executor *reportILAcceptanceExecutor) runDelegate(ctx context.Context, req agentexec.AgentRequest, observer agentexec.AgentObserver) (agentexec.AgentResult, error) {
	if streaming, ok := executor.delegate.(agentexec.StreamingAgentExecutor); ok {
		return streaming.RunWithObserver(ctx, req, observer)
	}
	return executor.delegate.Run(ctx, req)
}

func (executor *reportILAcceptanceExecutor) snapshot() []reportILProviderCallReceipt {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return append([]reportILProviderCallReceipt(nil), executor.calls...)
}

type reportILSyntheticExecutor struct {
	mu         sync.Mutex
	service    *app.Service
	failAuthor bool
}

func (executor *reportILSyntheticExecutor) Run(ctx context.Context, req agentexec.AgentRequest) (agentexec.AgentResult, error) {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	stage := strings.TrimPrefix(strings.TrimSpace(req.UserText), "report IL ")
	server, err := executor.syntheticServer(req)
	if err != nil {
		return agentexec.AgentResult{}, err
	}
	switch stage {
	case "il_editorial_memory":
		if err := readSyntheticFrozenCatalog(ctx, server, stage); err != nil {
			return agentexec.AgentResult{}, err
		}
		if err := writeSyntheticEditorialMemory(ctx, server); err != nil {
			return agentexec.AgentResult{}, err
		}
	case "il_narrative":
		if req.ReportILSources.MaxReadBytes > 0 {
			if err := readSyntheticKeyedFrozenCatalog(ctx, server, req.ReportILSources.Catalog, stage); err != nil {
				return agentexec.AgentResult{}, err
			}
		} else if err := readSyntheticEditorialMemory(ctx, server, ""); err != nil {
			return agentexec.AgentResult{}, err
		}
		if executor.failAuthor {
			usage := agentusage.New("codex", "codex", req.Model, req.ReasoningEffort, req.Prompt).
				WithProviderUsage(agentusage.ProviderUsage{InputTokens: 10, OutputTokens: 10}, "synthetic_preflight")
			return agentexec.AgentResult{Text: "must-not-persist", SessionID: "must-not-persist", Log: "must-not-persist", Usage: usage}, nil
		}
		if err := executor.writeSyntheticAuthorDocument(ctx, server, req.ReportILSources.MaxReadBytes > 0); err != nil {
			return agentexec.AgentResult{}, err
		}
	case "il_long_form_plan":
		if req.ReportILSources.MaxReadBytes > 0 {
			if err := readSyntheticKeyedFrozenCatalog(ctx, server, req.ReportILSources.Catalog, stage); err != nil {
				return agentexec.AgentResult{}, err
			}
		} else if err := readSyntheticEditorialMemory(ctx, server, ""); err != nil {
			return agentexec.AgentResult{}, err
		}
		if executor.failAuthor {
			usage := agentusage.New("codex", "codex", req.Model, req.ReasoningEffort, req.Prompt).
				WithProviderUsage(agentusage.ProviderUsage{InputTokens: 10, OutputTokens: 10}, "synthetic_preflight")
			return agentexec.AgentResult{Text: "must-not-persist", SessionID: "must-not-persist", Log: "must-not-persist", Usage: usage}, nil
		}
		if err := writeSyntheticLongFormPlan(ctx, server, req.ReportILSources); err != nil {
			return agentexec.AgentResult{}, err
		}
	case "il_long_form_section":
		if req.ReportILSources.MaxReadBytes > 0 {
			if err := readSyntheticKeyedSources(ctx, server, req.ReportILSources.Catalog, stage, req.ReportILSources.LongFormSourceKeys); err != nil {
				return agentexec.AgentResult{}, err
			}
		} else if err := readSyntheticEditorialMemory(ctx, server, ""); err != nil {
			return agentexec.AgentResult{}, err
		}
		if err := writeSyntheticLongFormSection(ctx, server, req.ReportILSources); err != nil {
			return agentexec.AgentResult{}, err
		}
	case "il_long_form_part", "il_long_form_final":
		if req.ReportILSources.EditorialMemoryArtifactID != "" {
			if err := readSyntheticEditorialMemory(ctx, server, ""); err != nil {
				return agentexec.AgentResult{}, err
			}
		}
		if err := finalizeSyntheticLongFormEdit(ctx, server, stage); err != nil {
			return agentexec.AgentResult{}, err
		}
	case "il_continuity":
		if err := readSyntheticEditorialMemory(ctx, server, ""); err != nil {
			return agentexec.AgentResult{}, err
		}
		if err := executor.finalizeSyntheticContinuityDocument(ctx, server); err != nil {
			return agentexec.AgentResult{}, err
		}
	case "il_reader":
		if err := readSyntheticEditorialMemory(ctx, server, ""); err != nil {
			return agentexec.AgentResult{}, err
		}
		if err := executor.reviseSyntheticPublicationDocument(ctx, server); err != nil {
			return agentexec.AgentResult{}, err
		}
	default:
		return agentexec.AgentResult{}, fmt.Errorf("unexpected synthetic IL stage %q", stage)
	}
	usage := agentusage.New("codex", "codex", req.Model, req.ReasoningEffort, req.Prompt).
		WithProviderUsage(agentusage.ProviderUsage{InputTokens: 10, OutputTokens: 10}, "synthetic_preflight")
	return agentexec.AgentResult{Text: "workspace finalized", Usage: usage}, nil
}

func (executor *reportILSyntheticExecutor) syntheticServer(req agentexec.AgentRequest) (*plasmamcp.Server, error) {
	if executor.service == nil || req.ReportILSources == nil {
		return nil, errors.New("synthetic report IL fixture is not bound")
	}
	return plasmamcp.NewServer(
		executor.service,
		plasmamcp.WithBinding(plasmamcp.Binding{
			MissionID:      req.MissionID,
			AgentSessionID: req.ToolSessionID,
			AgentExecutor:  req.AgentExecutor,
		}),
		plasmamcp.WithReportILSourceBinding(*req.ReportILSources),
		plasmamcp.WithEnabledTools(req.ExtraMCPTools),
	), nil
}

func readSyntheticFrozenCatalog(ctx context.Context, server *plasmamcp.Server, stage string) error {
	listed := server.Call(ctx, plasmamcp.ToolCall{
		Name:      reportilcontract.SourceListTool,
		Arguments: json.RawMessage(`{}`),
	})
	if listed.Error != nil || listed.TraceError != "" {
		return fmt.Errorf("synthetic source list failed: %#v", listed)
	}
	for {
		read := server.Call(ctx, plasmamcp.ToolCall{
			Name:      reportilcontract.SourceReadTool,
			Arguments: json.RawMessage(`{}`),
		})
		if read.Error != nil || read.TraceError != "" || read.TraceEventID == "" {
			return fmt.Errorf("synthetic server-ordered source read failed: %#v", read)
		}
		encoded, err := json.Marshal(read.Content)
		if err != nil {
			return err
		}
		var batch struct {
			Stage            string `json:"stage"`
			Sources          []any  `json:"sources"`
			RemainingSources int    `json:"remaining_sources"`
		}
		if err := json.Unmarshal(encoded, &batch); err != nil {
			return err
		}
		if batch.Stage != stage || len(batch.Sources) == 0 {
			return fmt.Errorf("synthetic server-ordered source batch violated stage binding: %#v", batch)
		}
		if batch.RemainingSources == 0 {
			return nil
		}
	}
}

func readSyntheticKeyedFrozenCatalog(
	ctx context.Context,
	server *plasmamcp.Server,
	catalog reportilcontract.SourceCatalog,
	stage string,
) error {
	listed := server.Call(ctx, plasmamcp.ToolCall{
		Name:      reportilcontract.SourceListTool,
		Arguments: json.RawMessage(`{}`),
	})
	if listed.Error != nil || listed.TraceError != "" {
		return fmt.Errorf("synthetic source list failed: %#v", listed)
	}
	keys := make([]string, 0, len(catalog.Sources))
	for _, entry := range catalog.Sources {
		keys = append(keys, entry.SourceKey)
	}
	return readSyntheticKeyedSources(ctx, server, catalog, stage, keys)
}

func readSyntheticKeyedSources(
	ctx context.Context,
	server *plasmamcp.Server,
	catalog reportilcontract.SourceCatalog,
	stage string,
	sourceKeys []string,
) error {
	for _, sourceKey := range sourceKeys {
		entry, ok := catalog.Entry(sourceKey)
		if !ok {
			return fmt.Errorf("synthetic bound source is unavailable: %s", sourceKey)
		}
		offset := 0
		for {
			read := server.Call(ctx, plasmamcp.ToolCall{
				Name: reportilcontract.SourceReadTool,
				Arguments: mustJSON(map[string]any{
					"source_key": entry.SourceKey,
					"offset":     offset,
					"max_bytes":  reportilcontract.DefaultSourceReadMaxBytes,
				}),
			})
			if read.Error != nil || read.TraceError != "" || read.TraceEventID == "" {
				return fmt.Errorf("synthetic keyed source read failed: %#v", read)
			}
			var output struct {
				Stage      string `json:"stage"`
				SourceKey  string `json:"source_key"`
				NextOffset int    `json:"next_offset"`
				Truncated  bool   `json:"truncated"`
			}
			if err := json.Unmarshal(takeJSON(read.Content), &output); err != nil {
				return err
			}
			if output.Stage != stage || output.SourceKey != entry.SourceKey {
				return fmt.Errorf("synthetic keyed source read violated stage binding: %#v", output)
			}
			if !output.Truncated {
				break
			}
			offset = output.NextOffset
		}
	}
	return nil
}

func writeSyntheticEditorialMemory(ctx context.Context, server *plasmamcp.Server) error {
	quoted := server.Call(ctx, plasmamcp.ToolCall{
		Name: reportilcontract.SourceQuoteRegisterTool,
		Arguments: mustJSON(map[string]any{
			"source_key": "source_001",
			"quote":      "The independent report pipeline is opt-in and does not replace classic planned or long-form reports.",
		}),
	})
	if quoted.Error != nil {
		return fmt.Errorf("synthetic editorial memory quote failed: %#v", quoted)
	}
	var quote struct {
		SourceReceipt string `json:"source_receipt"`
	}
	if err := json.Unmarshal(takeJSON(quoted.Content), &quote); err != nil {
		return err
	}
	if quote.SourceReceipt == "" {
		return errors.New("synthetic editorial memory quote returned no receipt")
	}
	started := server.Call(ctx, plasmamcp.ToolCall{
		Name: reportilcontract.EditorialMemoryStartTool,
		Arguments: mustJSON(map[string]any{
			"language": "en",
		}),
	})
	if started.Error != nil {
		return fmt.Errorf("synthetic editorial memory start failed: %#v", started)
	}
	var state syntheticDocumentState
	if err := json.Unmarshal(takeJSON(started.Content), &state); err != nil {
		return err
	}
	appended := server.Call(ctx, plasmamcp.ToolCall{
		Name: reportilcontract.EditorialMemoryAppendTool,
		Arguments: mustJSON(map[string]any{
			"workspace_id":   state.WorkspaceID,
			"importance":     "essential",
			"account":        "The independent report pipeline is opt-in and does not replace classic planned or long-form reports. It preserves connected facts in server-owned editorial memory before prose, binds authored blocks to those accounts, and deterministically projects the final semantic document to Markdown, self-contained HTML, and PDF without silent fallback.",
			"source_keys":    []string{"source_001"},
			"source_anchors": []string{quote.SourceReceipt},
		}),
	})
	if appended.Error != nil {
		return fmt.Errorf("synthetic editorial memory append failed: %#v", appended)
	}
	if err := readSyntheticEditorialMemory(ctx, server, state.WorkspaceID); err != nil {
		return err
	}
	finalized := server.Call(ctx, plasmamcp.ToolCall{
		Name:      reportilcontract.EditorialMemoryFinalizeTool,
		Arguments: mustJSON(map[string]any{"workspace_id": state.WorkspaceID}),
	})
	if finalized.Error != nil || finalized.TraceError != "" {
		return fmt.Errorf("synthetic editorial memory finalize failed: %#v", finalized)
	}
	return nil
}

func readSyntheticEditorialMemory(ctx context.Context, server *plasmamcp.Server, workspaceID string) error {
	offset := 0
	for {
		arguments := map[string]any{"offset": offset, "max_bytes": 65536}
		if workspaceID != "" {
			arguments["workspace_id"] = workspaceID
		}
		read := server.Call(ctx, plasmamcp.ToolCall{
			Name:      reportilcontract.EditorialMemoryReadTool,
			Arguments: mustJSON(arguments),
		})
		if read.Error != nil || read.TraceError != "" || read.TraceEventID == "" {
			return fmt.Errorf("synthetic editorial memory read failed: %#v", read)
		}
		var output struct {
			NextOffset int  `json:"next_offset"`
			Truncated  bool `json:"truncated"`
		}
		if err := json.Unmarshal(takeJSON(read.Content), &output); err != nil {
			return err
		}
		if !output.Truncated {
			return nil
		}
		offset = output.NextOffset
	}
}

func (executor *reportILSyntheticExecutor) writeSyntheticAuthorDocument(
	ctx context.Context,
	server *plasmamcp.Server,
	directSource bool,
) error {
	started := server.Call(ctx, plasmamcp.ToolCall{Name: reportilcontract.AuthorDocumentStartTool, Arguments: mustJSON(map[string]any{
		"title": "A bounded path to multi-format reports", "language": "en",
	})})
	if started.Error != nil {
		return fmt.Errorf("synthetic author start failed: %#v", started)
	}
	var state syntheticDocumentState
	if err := json.Unmarshal(takeJSON(started.Content), &state); err != nil {
		return err
	}
	blocks := []map[string]any{
		{"section_title": "What the architecture establishes", "kind": "prose", "prose": "The independent report pipeline is opt-in and does not replace classic planned or long-form reports.", "items": []string{}, "code": "", "language": nil, "table": nil},
		{"section_title": "What the architecture establishes", "kind": "table", "prose": "", "items": []string{}, "code": "", "language": nil, "table": map[string]any{"caption": "Fixed product contract", "columns": []string{"Boundary", "Value"}, "rows": []any{map[string]any{"cells": []string{"Author", "One source-aware author"}}, map[string]any{"cells": []string{"Outputs", "Six durable artifacts"}}}}},
		{"section_title": "How one manuscript reaches every format", "kind": "prose", "prose": "One source-aware author writes the complete manuscript before deterministic projections produce the durable output bundle.", "items": []string{}, "code": "", "language": nil, "table": nil},
		{"section_title": "Why the classic paths remain separate", "kind": "prose", "prose": "Classic planned and long-form reports remain separate choices while the independent pipeline preserves its own source and output boundaries.", "items": []string{}, "code": "", "language": nil, "table": nil},
		{"section_title": "The adoption boundary", "kind": "prose", "prose": "The evidence supports explicit internal dogfood for text and tables. It does not justify changing the existing report default, enabling visual assets, or adding silent fallback.", "items": []string{}, "code": "", "language": nil, "table": nil},
	}
	appendTool := reportilcontract.AuthorDocumentAppendTool
	for _, block := range blocks {
		block["workspace_id"] = state.WorkspaceID
		if directSource {
			appendTool = reportilcontract.AuthorDocumentAppendSourceTool
			block["evidence_source_keys"] = []string{"source_001"}
		} else {
			block["editorial_account_keys"] = []string{"account_001"}
		}
		result := server.Call(ctx, plasmamcp.ToolCall{Name: appendTool, Arguments: mustJSON(block)})
		if result.Error != nil {
			return fmt.Errorf("synthetic author append failed: %#v", result)
		}
	}
	return syntheticReadAndFinalizeDocument(ctx, server, state.WorkspaceID)
}

func writeSyntheticLongFormPlan(ctx context.Context, server *plasmamcp.Server, binding *reportilcontract.SourceAccessBinding) error {
	parts := make([]map[string]any, 2)
	for partIndex := range parts {
		sections := make([]map[string]any, 3)
		for sectionIndex := range sections {
			sections[sectionIndex] = map[string]any{
				"title": "Section", "purpose": "Explain one material question.",
				"representations": []string{}, "evidence_source_keys": []string{"source_001"},
			}
			if binding.EditorialMemoryArtifactID != "" {
				sections[sectionIndex]["editorial_account_keys"] = []string{"account_001"}
			}
		}
		parts[partIndex] = map[string]any{
			"title": "Part", "purpose": "Develop the reader's answer.", "sections": sections,
		}
	}
	result := server.Call(ctx, plasmamcp.ToolCall{
		Name: reportilcontract.LongFormPlanSubmitTool,
		Arguments: mustJSON(map[string]any{
			"title":   "A genuinely long report",
			"summary": "Explain the architecture as a reader-facing answer.", "parts": parts,
		}),
	})
	if result.Error != nil || result.TraceError != "" {
		return fmt.Errorf("synthetic long-form plan failed: %#v", result)
	}
	return nil
}

func writeSyntheticLongFormSection(ctx context.Context, server *plasmamcp.Server, binding *reportilcontract.SourceAccessBinding) error {
	if err := readSyntheticLongFormPlan(ctx, server); err != nil {
		return err
	}
	started := server.Call(ctx, plasmamcp.ToolCall{
		Name:      reportilcontract.LongFormDocumentStartTool,
		Arguments: mustJSON(map[string]any{"title": "Section", "language": "en"}),
	})
	if started.Error != nil {
		return fmt.Errorf("synthetic long-form Section start failed: %#v", started)
	}
	var state syntheticDocumentState
	if err := json.Unmarshal(takeJSON(started.Content), &state); err != nil {
		return err
	}
	for blockIndex := 0; blockIndex < 2; blockIndex++ {
		prose := fmt.Sprintf("This substantial reader-facing passage develops %s with concrete detail %d.", binding.LongFormSectionKey, blockIndex+1)
		if binding.LongFormSectionKey == "part_001.section_001" && blockIndex == 0 {
			prose = "The independent report pipeline is opt-in and does not replace classic planned or long-form reports."
		}
		arguments := map[string]any{
			"workspace_id": state.WorkspaceID, "section_key": binding.LongFormSectionKey,
			"kind": "prose", "prose": prose,
			"items": []string{}, "code": "", "language": nil, "table": nil,
			"editorial_account_keys": []string{}, "evidence_source_keys": []string{"source_001"},
		}
		if binding.EditorialMemoryArtifactID != "" {
			arguments["editorial_account_keys"] = []string{"account_001"}
		}
		appended := server.Call(ctx, plasmamcp.ToolCall{Name: reportilcontract.LongFormDocumentAppendTool, Arguments: mustJSON(arguments)})
		if appended.Error != nil {
			return fmt.Errorf("synthetic long-form Section append failed: %#v", appended)
		}
	}
	return syntheticReadAndFinalizeLongFormDocument(ctx, server, state.WorkspaceID)
}

func finalizeSyntheticLongFormEdit(ctx context.Context, server *plasmamcp.Server, stage string) error {
	if err := readSyntheticLongFormPlan(ctx, server); err != nil {
		return err
	}
	started := server.Call(ctx, plasmamcp.ToolCall{
		Name:      reportilcontract.LongFormDocumentStartTool,
		Arguments: mustJSON(map[string]any{"title": "A genuinely long report", "language": "en"}),
	})
	if started.Error != nil {
		return fmt.Errorf("synthetic %s start failed: %#v", stage, started)
	}
	var state syntheticDocumentState
	if err := json.Unmarshal(takeJSON(started.Content), &state); err != nil {
		return err
	}
	return syntheticReadAndFinalizeLongFormDocument(ctx, server, state.WorkspaceID)
}

func readSyntheticLongFormPlan(ctx context.Context, server *plasmamcp.Server) error {
	offset := 0
	for {
		read := server.Call(ctx, plasmamcp.ToolCall{
			Name:      reportilcontract.LongFormPlanReadTool,
			Arguments: mustJSON(map[string]any{"offset": offset, "max_bytes": 65536}),
		})
		if read.Error != nil || read.TraceError != "" {
			return fmt.Errorf("synthetic long-form plan read failed: %#v", read)
		}
		var output struct {
			NextOffset int  `json:"next_offset"`
			Truncated  bool `json:"truncated"`
		}
		if err := json.Unmarshal(takeJSON(read.Content), &output); err != nil {
			return err
		}
		if !output.Truncated {
			return nil
		}
		offset = output.NextOffset
	}
}

func syntheticReadAndFinalizeLongFormDocument(ctx context.Context, server *plasmamcp.Server, workspaceID string) error {
	if err := syntheticReadDocumentWithTool(ctx, server, reportilcontract.LongFormDocumentReadTool, workspaceID); err != nil {
		return err
	}
	finalized := server.Call(ctx, plasmamcp.ToolCall{
		Name:      reportilcontract.LongFormDocumentFinalizeTool,
		Arguments: mustJSON(map[string]any{"workspace_id": workspaceID}),
	})
	if finalized.Error != nil || finalized.TraceError != "" {
		return fmt.Errorf("synthetic long-form document finalize failed: %#v", finalized)
	}
	return nil
}

func (executor *reportILSyntheticExecutor) reviseSyntheticPublicationDocument(ctx context.Context, server *plasmamcp.Server) error {
	opened := server.Call(ctx, plasmamcp.ToolCall{Name: reportilcontract.AuthorDocumentOpenTool, Arguments: json.RawMessage(`{}`)})
	if opened.Error != nil {
		return fmt.Errorf("synthetic continuity open failed: %#v", opened)
	}
	var state syntheticDocumentState
	if err := json.Unmarshal(takeJSON(opened.Content), &state); err != nil {
		return err
	}
	if err := syntheticReadDocument(ctx, server, state.WorkspaceID); err != nil {
		return err
	}
	oldText := "The independent report pipeline is opt-in and does not replace classic planned or long-form reports."
	newText := "The independent report pipeline is opt-in and preserves classic planned and long-form reports as separate choices."
	revised := server.Call(ctx, plasmamcp.ToolCall{Name: reportilcontract.AuthorDocumentEditTextTool, Arguments: mustJSON(map[string]any{
		"workspace_id": state.WorkspaceID,
		"target_kind":  "block_text",
		"target_key":   "part_001.section_001.block_001",
		"old_text":     oldText,
		"new_text":     newText,
	})})
	if revised.Error != nil {
		return fmt.Errorf("synthetic continuity revision failed: %#v", revised)
	}
	return syntheticReadAndFinalizeDocument(ctx, server, state.WorkspaceID)
}

func (executor *reportILSyntheticExecutor) finalizeSyntheticContinuityDocument(ctx context.Context, server *plasmamcp.Server) error {
	opened := server.Call(ctx, plasmamcp.ToolCall{Name: reportilcontract.AuthorDocumentOpenTool, Arguments: json.RawMessage(`{}`)})
	if opened.Error != nil {
		return fmt.Errorf("synthetic publication open failed: %#v", opened)
	}
	var state syntheticDocumentState
	if err := json.Unmarshal(takeJSON(opened.Content), &state); err != nil {
		return err
	}
	return syntheticReadAndFinalizeDocument(ctx, server, state.WorkspaceID)
}

type syntheticDocumentState struct {
	WorkspaceID string `json:"workspace_id"`
}

func syntheticReadDocument(ctx context.Context, server *plasmamcp.Server, workspaceID string) error {
	return syntheticReadDocumentWithTool(ctx, server, reportilcontract.AuthorDocumentReadTool, workspaceID)
}

func syntheticReadDocumentWithTool(ctx context.Context, server *plasmamcp.Server, toolName, workspaceID string) error {
	offset := 0
	for {
		read := server.Call(ctx, plasmamcp.ToolCall{Name: toolName, Arguments: mustJSON(map[string]any{
			"workspace_id": workspaceID, "offset": offset, "max_bytes": 65536,
		})})
		if read.Error != nil {
			return fmt.Errorf("synthetic document read failed: %#v", read)
		}
		var output struct {
			NextOffset int  `json:"next_offset"`
			Truncated  bool `json:"truncated"`
		}
		if err := json.Unmarshal(takeJSON(read.Content), &output); err != nil {
			return err
		}
		if !output.Truncated {
			return nil
		}
		offset = output.NextOffset
	}
}

func syntheticReadAndFinalizeDocument(ctx context.Context, server *plasmamcp.Server, workspaceID string) error {
	if err := syntheticReadDocument(ctx, server, workspaceID); err != nil {
		return err
	}
	finalized := server.Call(ctx, plasmamcp.ToolCall{Name: reportilcontract.AuthorDocumentFinalizeTool, Arguments: mustJSON(map[string]any{"workspace_id": workspaceID})})
	if finalized.Error != nil || finalized.TraceError != "" {
		return fmt.Errorf("synthetic document finalize failed: %#v", finalized)
	}
	return nil
}

func takeJSON(value any) []byte {
	encoded, _ := json.Marshal(value)
	return encoded
}

func acceptancePromptValue(prompt, prefix string) (string, error) {
	index := strings.Index(prompt, prefix)
	if index < 0 {
		return "", fmt.Errorf("acceptance prompt lacks %q", prefix)
	}
	value := prompt[index+len(prefix):]
	if lineEnd := strings.IndexByte(value, '\n'); lineEnd >= 0 {
		value = value[:lineEnd]
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("acceptance prompt has empty %q", prefix)
	}
	return value, nil
}

func TestReportILProviderAcceptanceFixturePreflight(t *testing.T) {
	chromePath := testChromePath()
	if chromePath == "" {
		t.Skip("Chrome or Chromium is required for the Experimental IL provider acceptance preflight")
	}
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	observer := &reportILAcceptanceExecutor{delegate: &reportILSyntheticExecutor{service: service}}
	server := httptest.NewServer(NewServer(service, Options{AgentExecutor: observer, ReportILChromePath: chromePath}))
	defer server.Close()

	missionID, pendingID, err := startReportILAcceptanceHTTP(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	terminal, events, err := waitReportILAcceptanceTerminal(ctx, service, missionID, pendingID, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if terminal.EventType != "report.artifact.created" {
		failure := safeReportILFailureReceipt(terminal)
		t.Fatalf("synthetic acceptance terminal = %s/%#v calls=%#v", terminal.EventType, failure, observer.snapshot())
	}
	artifacts, err := inspectReportILAcceptanceSuccess(ctx, service, missionID, pendingID, terminal)
	if err != nil {
		t.Fatal(err)
	}
	calls := observer.snapshot()
	if len(calls) != 13 || len(artifacts) != 6 {
		t.Fatalf("synthetic acceptance calls/artifacts = %d/%d", len(calls), len(artifacts))
	}
	if err := validateLongFormReportILProviderCalls(calls, reportilphase0.ValidationProfileStrict); err != nil {
		t.Fatal(err)
	}
	if err := scanReportILAcceptancePrivacy(events, service, missionID); err != nil {
		t.Fatal(err)
	}
}

func TestLongFormArticleHTTPReusesReportILProductPath(t *testing.T) {
	chromePath := testChromePath()
	if chromePath == "" {
		t.Skip("Chrome or Chromium is required for the long-form Article preflight")
	}
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	observer := &reportILAcceptanceExecutor{delegate: &reportILSyntheticExecutor{service: service}}
	server := httptest.NewServer(NewServer(service, Options{AgentExecutor: observer, ReportILChromePath: chromePath}))
	defer server.Close()

	request := reportILProviderAcceptanceRequest()
	request["output_kind"] = "article"
	request["execution_strategy"] = "section_fanout"
	request["article_intent"] = map[string]any{
		"audience":       "장문 기능을 설계하는 제품 엔지니어",
		"reader_promise": "기존 경로로 단편 책자를 만드는 순서를 이해한다",
		"emphasis":       "재사용 우선 접근",
	}
	missionID, pendingID, err := startReportILAcceptanceHTTPWithRequest(server.URL, request)
	if err != nil {
		t.Fatal(err)
	}
	terminal, _, err := waitReportILAcceptanceTerminal(ctx, service, missionID, pendingID, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if terminal.EventType != "report.artifact.created" {
		t.Fatalf("long-form Article terminal = %s/%#v calls=%#v", terminal.EventType, safeReportILFailureReceipt(terminal), observer.snapshot())
	}
	var payload map[string]any
	if err := json.Unmarshal(terminal.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	intent, _ := payload["article_intent"].(map[string]any)
	if payload["output_kind"] != "article" || payload["report_mode"] != "long_form" || payload["pipeline_family"] != reportilcontract.PipelineFamily || intent["audience"] != "장문 기능을 설계하는 제품 엔지니어" {
		t.Fatalf("long-form Article terminal identity = %#v", payload)
	}
	if _, err := inspectReportILAcceptanceSuccess(ctx, service, missionID, pendingID, terminal); err != nil {
		t.Fatal(err)
	}
	for _, call := range observer.snapshot() {
		if call.Stage == "il_long_form_plan" || call.Stage == "il_long_form_section" || call.Stage == "il_long_form_part" || call.Stage == "il_long_form_final" || call.Stage == "il_reader" || call.Stage == "il_continuity" {
			if !call.ArticleContract {
				t.Fatalf("stage %s lost Article contract", call.Stage)
			}
		}
	}
}

func TestReportILUnverifiedProfileHTTPProductPreflight(t *testing.T) {
	chromePath := testChromePath()
	if chromePath == "" {
		t.Skip("Chrome or Chromium is required for the unverified IL product preflight")
	}
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	observer := &reportILAcceptanceExecutor{delegate: &reportILSyntheticExecutor{service: service}}
	server := httptest.NewServer(NewServer(service, Options{AgentExecutor: observer, ReportILChromePath: chromePath}))
	defer server.Close()

	request := reportILProviderAcceptanceRequest()
	request["rigor_level"] = "unverified"
	missionID, pendingID, err := startReportILAcceptanceHTTPWithRequest(server.URL, request)
	if err != nil {
		t.Fatal(err)
	}
	terminal, events, err := waitReportILAcceptanceTerminal(ctx, service, missionID, pendingID, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if terminal.EventType != "report.artifact.created" {
		failure := safeReportILFailureReceipt(terminal)
		t.Fatalf("unverified IL terminal = %s/%#v calls=%#v", terminal.EventType, failure, observer.snapshot())
	}
	artifacts, err := inspectReportILAcceptanceSuccessForProfile(
		ctx, service, missionID, pendingID, terminal, reportilphase0.ValidationProfileUnverified,
	)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, artifact := range artifacts {
		kinds[artifact.Kind] = true
	}
	for _, kind := range []string{"narrative", "semantic_il", "markdown", "html", "pdf", "manifest"} {
		if !kinds[kind] {
			t.Fatalf("unverified IL bundle lacks %q: %#v", kind, artifacts)
		}
	}
	calls := observer.snapshot()
	if err := validateLongFormReportILProviderCalls(calls, reportilphase0.ValidationProfileUnverified); err != nil {
		t.Fatalf("unverified IL provider calls: %v: %#v", err, calls)
	}
	for _, event := range events {
		switch event.EventType {
		case "report.il_editorial_memory.started", "report.il_editorial_memory.completed",
			"report.il_reader.started", "report.il_reader.completed",
			"report.il_continuity.started", "report.il_continuity.completed":
			t.Fatalf("unverified IL ran omitted content stage: %s", event.EventType)
		}
	}
	var pendingPayload map[string]any
	for _, event := range events {
		if event.EventID == pendingID {
			if err := json.Unmarshal(event.Payload, &pendingPayload); err != nil {
				t.Fatal(err)
			}
		}
	}
	if pendingPayload["pipeline_family"] != reportilcontract.PipelineFamily ||
		pendingPayload["pipeline_graph"] != reportpipeline.ExperimentalILValidationProfilesGraph ||
		pendingPayload["rigor_level"] != "unverified" || pendingPayload["rigor_label"] != "무검증" {
		t.Fatalf("unverified IL pending payload = %#v", pendingPayload)
	}
	payload, err := reportilcontract.DecodeTerminalPayload(terminal.Payload)
	if err != nil {
		t.Fatal(err)
	}
	manifestArtifact, err := reportILAcceptanceArtifactByKind(ctx, service, terminal, "manifest")
	if err != nil {
		t.Fatal(err)
	}
	var manifest reportilphase0.ProductManifest
	if err := json.Unmarshal(manifestArtifact.Content, &manifest); err != nil {
		t.Fatal(err)
	}
	if payload.Bundle.PipelineFamily != reportilcontract.PipelineFamily || manifest.AuthoringMode != reportilphase0.AuthoringModeLongForm || manifest.ValidationProfile != "unverified" {
		t.Fatalf("unverified IL terminal/manifest = %q/%q", payload.Bundle.PipelineFamily, manifest.ValidationProfile)
	}
}

func TestReportILExploratoryProfileHTTPProductPreflight(t *testing.T) {
	chromePath := testChromePath()
	if chromePath == "" {
		t.Skip("Chrome or Chromium is required for the exploratory IL product preflight")
	}
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	observer := &reportILAcceptanceExecutor{delegate: &reportILSyntheticExecutor{service: service}}
	server := httptest.NewServer(NewServer(service, Options{AgentExecutor: observer, ReportILChromePath: chromePath}))
	defer server.Close()

	request := reportILProviderAcceptanceRequest()
	request["rigor_level"] = "exploratory"
	missionID, pendingID, err := startReportILAcceptanceHTTPWithRequest(server.URL, request)
	if err != nil {
		t.Fatal(err)
	}
	terminal, events, err := waitReportILAcceptanceTerminal(ctx, service, missionID, pendingID, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if terminal.EventType != "report.artifact.created" {
		failure := safeReportILFailureReceipt(terminal)
		t.Fatalf("exploratory IL terminal = %s/%#v calls=%#v", terminal.EventType, failure, observer.snapshot())
	}
	artifacts, err := inspectReportILAcceptanceSuccessForProfile(
		ctx, service, missionID, pendingID, terminal, reportilphase0.ValidationProfileExploratory,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 6 {
		t.Fatalf("exploratory IL artifacts = %d, want 6: %#v", len(artifacts), artifacts)
	}
	calls := observer.snapshot()
	if err := validateLongFormReportILProviderCalls(calls, reportilphase0.ValidationProfileExploratory); err != nil {
		t.Fatalf("exploratory IL provider stages: %v: %#v", err, calls)
	}
	for _, event := range events {
		if event.EventType == "report.il_continuity.started" || event.EventType == "report.il_continuity.completed" {
			t.Fatalf("exploratory IL ran strict-only continuity stage: %s", event.EventType)
		}
	}
	var pendingPayload map[string]any
	for _, event := range events {
		if event.EventID == pendingID {
			if err := json.Unmarshal(event.Payload, &pendingPayload); err != nil {
				t.Fatal(err)
			}
		}
	}
	if pendingPayload["pipeline_family"] != reportilcontract.PipelineFamily ||
		pendingPayload["pipeline_graph"] != reportpipeline.ExperimentalILValidationProfilesGraph ||
		pendingPayload["rigor_level"] != "exploratory" || pendingPayload["rigor_label"] != "탐색형" {
		t.Fatalf("exploratory IL pending payload = %#v", pendingPayload)
	}
}

func TestReportILProviderAcceptanceFailurePreflight(t *testing.T) {
	chromePath := testChromePath()
	if chromePath == "" {
		t.Skip("Chrome or Chromium is required for the Experimental IL provider acceptance failure preflight")
	}
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	observer := &reportILAcceptanceExecutor{delegate: &reportILSyntheticExecutor{service: service, failAuthor: true}}
	server := httptest.NewServer(NewServer(service, Options{AgentExecutor: observer, ReportILChromePath: chromePath}))
	defer server.Close()

	missionID, pendingID, err := startReportILAcceptanceHTTP(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	terminal, events, err := waitReportILAcceptanceTerminal(ctx, service, missionID, pendingID, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	failure := safeReportILFailureReceipt(terminal)
	calls := observer.snapshot()
	if terminal.EventType != "report.draft.failed" || failure.FailedStage != "il_long_form_plan" || failure.SafeFailureReason != string(reportexecution.ProviderFailureReasonSemanticValidation) || failure.SafeValidationCode != string(reportexecution.ProviderValidationCodeDocumentContract) {
		t.Fatalf("synthetic failure terminal = %s/%#v", terminal.EventType, failure)
	}
	if err := validateLongFormReportILFailedProviderCalls(calls, failure.FailedStage); err != nil {
		t.Fatal(err)
	}
	if err := validateReportILFailureUsage(terminal, events, calls); err != nil {
		t.Fatal(err)
	}
	if countReportILAcceptanceClassicTerminals(events) != 0 {
		t.Fatal("synthetic failure silently fell back")
	}
	artifacts, err := service.ListRawArtifacts(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("synthetic failure raw artifacts = %d, want the accepted source and finalized editorial memory", len(artifacts))
	}
	mediaTypes := map[string]bool{}
	for _, artifact := range artifacts {
		mediaTypes[artifact.MediaType] = true
	}
	if !mediaTypes["text/plain; charset=utf-8"] || !mediaTypes[reportilcontract.EditorialMemoryMediaType] {
		t.Fatalf("synthetic failure raw artifact media types = %#v", mediaTypes)
	}
	if err := scanReportILAcceptancePrivacy(events, service, missionID); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(terminal.Payload), "must-not-persist") {
		t.Fatalf("synthetic failure leaked result internals: %s", terminal.Payload)
	}
}

func TestReportILProviderAcceptance(t *testing.T) {
	if strings.TrimSpace(os.Getenv(reportILProviderAcceptanceGateEnv)) != "1" {
		t.Skip(reportILProviderAcceptanceGateEnv + "=1 is required for the external-cost acceptance run")
	}
	outputDir := strings.TrimSpace(os.Getenv(reportILProviderAcceptanceDirEnv))
	if outputDir == "" || !filepath.IsAbs(outputDir) {
		t.Fatal(reportILProviderAcceptanceDirEnv + " must name a new absolute durable evidence directory")
	}
	if entries, err := os.ReadDir(outputDir); err == nil && len(entries) != 0 {
		t.Fatalf("acceptance evidence directory must be new and empty: %s", outputDir)
	} else if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outputDir, 0o700); err != nil {
		t.Fatal(err)
	}

	receipt := reportILProviderAcceptanceReceipt{
		SchemaVersion: "plasma.report_il.provider_acceptance.v2", Status: "started",
		SourceSHA256: acceptanceSHA256([]byte(reportILProviderAcceptanceSource)),
		Provider:     "codex", Model: "gpt-5.6-luna", ReasoningEffort: "xhigh",
	}
	if err := writeReportILProviderAcceptanceReceipt(outputDir, receipt); err != nil {
		t.Fatalf("write initial provider acceptance receipt: %v", err)
	}
	writeReceipt := func() {
		if err := writeReportILProviderAcceptanceReceipt(outputDir, receipt); err != nil {
			t.Errorf("write provider acceptance receipt: %v", err)
		}
	}
	if err := writeReportILAcceptanceInputs(outputDir); err != nil {
		receipt.Status = "preflight_failed"
		writeReceipt()
		t.Fatal(err)
	}

	chromePath := testChromePath()
	if chromePath == "" {
		receipt.Status = "preflight_failed"
		writeReceipt()
		t.Fatal("Chrome or Chromium is required for the provider acceptance run")
	}
	realCodex := strings.TrimSpace(os.Getenv(reportILProviderCodexCommandEnv))
	if realCodex == "" {
		realCodex = "codex"
	}
	realCodex = agentexec.ResolveAgentCommand(realCodex)
	if info, err := os.Stat(realCodex); err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		receipt.Status = "preflight_failed"
		writeReceipt()
		t.Fatalf("Codex executable is unavailable: %v", err)
	}
	transportPath := filepath.Join(outputDir, "transport.tsv")
	temporaryDir := t.TempDir()
	wrapperPath, err := writeReportILAcceptanceCodexWrapper(temporaryDir)
	if err != nil {
		receipt.Status = "preflight_failed"
		writeReceipt()
		t.Fatal(err)
	}
	plasmaBinary := filepath.Join(temporaryDir, "plasma")
	build := exec.Command("go", "build", "-o", plasmaBinary, "./cmd/plasma")
	build.Dir = filepath.Clean(filepath.Join("..", ".."))
	if output, buildErr := build.CombinedOutput(); buildErr != nil {
		receipt.Status = "preflight_failed"
		writeReceipt()
		t.Fatalf("build Plasma MCP acceptance binary: %v: %s", buildErr, output)
	}
	databasePath := filepath.Join(outputDir, "plasma.db")
	env := append(agentexec.CodexEnvironment(nil),
		"PLASMA_IL_REAL_CODEX="+realCodex,
		"PLASMA_IL_TRANSPORT_RECEIPT="+transportPath,
	)
	realExecutor := agentexec.CodexExecutor{
		Command: wrapperPath, WorkDir: filepath.Clean(filepath.Join("..", "..")), Timeout: 20 * time.Minute, Env: env,
		MCPServer: agentexec.CodexMCPServer{
			Name: "plasma", Command: plasmaBinary, Args: []string{"mcp", "-db", databasePath},
			Required: true, StartupTimeoutSec: 30, ToolTimeoutSec: 240,
		},
	}
	observer := &reportILAcceptanceExecutor{delegate: realExecutor}
	ctx := context.Background()
	store, err := sqlite.Open(ctx, databasePath)
	if err != nil {
		receipt.Status = "preflight_failed"
		writeReceipt()
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	server := httptest.NewServer(NewServer(service, Options{AgentExecutor: observer, ReportILChromePath: chromePath}))
	defer server.Close()

	missionID, pendingID, err := startReportILAcceptanceHTTP(server.URL)
	if err != nil {
		receipt.Status = "start_failed"
		writeReceipt()
		t.Fatal(err)
	}
	receipt.MissionID, receipt.PendingEventID = missionID, pendingID
	writeReceipt()

	terminal, events, waitErr := waitReportILAcceptanceTerminal(ctx, service, missionID, pendingID, 65*time.Minute)
	receipt.Calls = observer.snapshot()
	receipt.ProviderCallCount = len(receipt.Calls)
	if terminal.EventID != "" {
		receipt.TerminalEventType = terminal.EventType
	}
	if waitErr != nil {
		receipt.Status = "terminal_wait_failed"
		writeReceipt()
		t.Fatal(waitErr)
	}
	if terminal.EventType == "report.draft.failed" {
		failure := safeReportILFailureReceipt(terminal)
		receipt.Status = "failed"
		receipt.FailedStage = failure.FailedStage
		receipt.SafeErrorClass = failure.SafeErrorClass
		receipt.SafeFailureReason = failure.SafeFailureReason
		receipt.NoSilentFallback = countReportILAcceptanceClassicTerminals(events) == 0
		transport, transportErr := readReportILTransportReceipts(transportPath)
		contractErr := validateLongFormReportILFailedProviderCalls(receipt.Calls, failure.FailedStage)
		transportContractErr := validateReportILTransport(receipt.Calls, transport)
		privacyErr := scanReportILAcceptancePrivacy(events, service, missionID)
		usageErr := validateReportILFailureUsage(terminal, events, receipt.Calls)
		if transportErr == nil && transportContractErr == nil {
			receipt.TransportVerified = true
			receipt.SourceMCPConfigured = reportILAcceptanceMCPConfigs(transport) > 0
		}
		receipt.SourceReadTraced = countReportILSourceReadSessions(events) == countReportILRawSourceReadingCalls(receipt.Calls)
		receipt.PrivacyScanPassed = privacyErr == nil
		writeReceipt()
		if contractErr != nil || transportErr != nil || transportContractErr != nil || privacyErr != nil || usageErr != nil || !receipt.NoSilentFallback || !receipt.SourceReadTraced || !receipt.SourceMCPConfigured {
			t.Fatalf("provider-backed Experimental IL failed with invalid evidence: calls=%v transport_read=%v transport=%v privacy=%v usage=%v fallback=%v source_reads=%v source_mcp=%v; preserved at %s", contractErr, transportErr, transportContractErr, privacyErr, usageErr, receipt.NoSilentFallback, receipt.SourceReadTraced, receipt.SourceMCPConfigured, outputDir)
		}
		t.Fatalf("provider-backed Experimental IL ended in typed %s/%s failure; durable evidence preserved at %s", receipt.FailedStage, receipt.SafeFailureReason, outputDir)
	}

	artifacts, inspectErr := inspectReportILAcceptanceSuccess(ctx, service, missionID, pendingID, terminal)
	if inspectErr != nil {
		receipt.Status = "inspection_failed"
		writeReceipt()
		t.Fatal(inspectErr)
	}
	receipt.Artifacts = artifacts
	receipt.ArtifactCount = len(artifacts)
	for _, event := range events {
		if event.EventType == "report.run.completed" && event.CorrelationID == pendingID {
			receipt.CompletionEventType = event.EventType
		}
	}
	transport, err := readReportILTransportReceipts(transportPath)
	if err != nil {
		receipt.Status = "transport_inspection_failed"
		writeReceipt()
		t.Fatal(err)
	}
	if err := validateReportILProviderCalls(receipt.Calls); err != nil {
		receipt.Status = "call_contract_failed"
		writeReceipt()
		t.Fatal(err)
	}
	if err := validateReportILTransport(receipt.Calls, transport); err != nil {
		receipt.Status = "transport_contract_failed"
		writeReceipt()
		t.Fatal(err)
	}
	pdfArtifact, err := reportILAcceptanceArtifactByKind(ctx, service, terminal, "pdf")
	if err != nil {
		receipt.Status = "pdf_inspection_failed"
		writeReceipt()
		t.Fatal(err)
	}
	pdfInfo, err := pdfdocument.Inspect(pdfArtifact.Content)
	if err != nil || pdfInfo.PageCount < 1 {
		receipt.Status = "pdf_inspection_failed"
		writeReceipt()
		t.Fatalf("inspect provider PDF: pages=%d err=%v", pdfInfo.PageCount, err)
	}
	receipt.PDFPageCount = pdfInfo.PageCount
	if err := scanReportILAcceptancePrivacy(events, service, missionID); err != nil {
		receipt.Status = "privacy_scan_failed"
		writeReceipt()
		t.Fatal(err)
	}
	receipt.PrivacyScanPassed = true
	receipt.NoSilentFallback = countReportILAcceptanceClassicTerminals(events) == 0
	receipt.SourceReadTraced = countReportILSourceReadSessions(events) == countReportILRawSourceReadingCalls(receipt.Calls)
	receipt.SourceMCPConfigured = reportILAcceptanceMCPConfigs(transport) > 0
	receipt.TransportVerified = true
	if receipt.CompletionEventType != "report.run.completed" || !receipt.NoSilentFallback || !receipt.SourceReadTraced || !receipt.SourceMCPConfigured {
		receipt.Status = "boundary_inspection_failed"
		writeReceipt()
		t.Fatalf("provider acceptance boundary failed: completion=%q fallback=%v source_reads=%v source_mcp=%v", receipt.CompletionEventType, receipt.NoSilentFallback, receipt.SourceReadTraced, receipt.SourceMCPConfigured)
	}
	receipt.Status = "completed"
	writeReceipt()
}

func startReportILAcceptanceHTTP(baseURL string) (string, string, error) {
	return startReportILAcceptanceHTTPWithRequest(baseURL, reportILProviderAcceptanceRequest())
}

func startReportILAcceptanceHTTPWithRequest(
	baseURL string,
	request map[string]any,
) (string, string, error) {
	mission, err := reportILAcceptancePostJSON(baseURL+"/api/missions", map[string]any{
		"title": "Experimental IL provider acceptance",
	})
	if err != nil {
		return "", "", err
	}
	missionID, err := acceptanceNestedString(mission, "projection", "mission_id")
	if err != nil {
		return "", "", err
	}
	if _, err := reportILAcceptancePostJSON(baseURL+"/api/missions/"+missionID+"/sources/text", map[string]any{
		"title": "Frozen Experimental IL acceptance source", "content": reportILProviderAcceptanceSource,
	}); err != nil {
		return "", "", err
	}
	started, err := reportILAcceptancePostJSON(baseURL+"/api/missions/"+missionID+"/reports", request)
	if err != nil {
		return "", "", err
	}
	pendingID, err := acceptanceNestedString(started, "pending_event", "EventID")
	if err != nil {
		return "", "", err
	}
	return missionID, pendingID, nil
}

func reportILProviderAcceptanceRequest() map[string]any {
	return map[string]any{
		"title":           "Experimental IL product acceptance",
		"direction_hint":  reportILProviderAcceptanceDirection,
		"pipeline_family": reportilcontract.PipelineFamily,
		"report_mode":     "long_form", "execution_strategy": "section_fanout",
		"agent_executor": "claude", "agent_model": "wrong-model", "agent_reasoning_effort": "low",
		"mcp_mode": "auto", "rigor_level": "strict", "report_session_policy": "same_session",
		"post_report_humanize": "enabled", "generation_guidance_profile": "wrong-profile",
	}
}

func writeReportILAcceptanceInputs(directory string) error {
	if err := os.WriteFile(filepath.Join(directory, "source.txt"), []byte(reportILProviderAcceptanceSource), 0o600); err != nil {
		return err
	}
	request, err := json.MarshalIndent(reportILProviderAcceptanceRequest(), "", "  ")
	if err != nil {
		return err
	}
	request = append(request, '\n')
	return os.WriteFile(filepath.Join(directory, "request.json"), request, 0o600)
}

func reportILAcceptancePostJSON(url string, body any) (map[string]any, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	response, err := http.Post(url, "application/json", bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	var decoded map[string]any
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("acceptance HTTP POST returned %d", response.StatusCode)
	}
	return decoded, nil
}

func acceptanceNestedString(value map[string]any, path ...string) (string, error) {
	var current any = value
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return "", fmt.Errorf("acceptance response lacks %s", strings.Join(path, "."))
		}
		current = object[key]
	}
	text, ok := current.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("acceptance response lacks %s", strings.Join(path, "."))
	}
	return text, nil
}

func waitReportILAcceptanceTerminal(ctx context.Context, service *app.Service, missionID, pendingID string, timeout time.Duration) (ledger.Event, []ledger.Event, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		events, err := service.ListEvents(ctx, missionID)
		if err != nil {
			return ledger.Event{}, nil, err
		}
		for _, event := range events {
			if event.EventType != "report.artifact.created" && event.EventType != "report.draft.failed" {
				continue
			}
			var payload struct {
				PendingEventID string `json:"pending_event_id"`
			}
			if json.Unmarshal(event.Payload, &payload) == nil && payload.PendingEventID == pendingID {
				if event.EventType == "report.artifact.created" {
					for time.Now().Before(deadline) {
						latest, listErr := service.ListEvents(ctx, missionID)
						if listErr != nil {
							return ledger.Event{}, nil, listErr
						}
						for _, candidate := range latest {
							if candidate.EventType == "report.run.completed" && candidate.CorrelationID == pendingID {
								return event, latest, nil
							}
						}
						time.Sleep(20 * time.Millisecond)
					}
					return ledger.Event{}, events, errors.New("report IL terminal lacked report.run.completed")
				}
				return event, events, nil
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	return ledger.Event{}, nil, errors.New("timed out waiting for provider-backed report IL terminal")
}

func inspectReportILAcceptanceSuccess(ctx context.Context, service *app.Service, missionID, pendingID string, terminal ledger.Event) ([]reportILArtifactReceipt, error) {
	return inspectReportILAcceptanceSuccessForProfile(
		ctx, service, missionID, pendingID, terminal, reportilphase0.ValidationProfileStrict,
	)
}

func inspectReportILAcceptanceSuccessForProfile(
	ctx context.Context,
	service *app.Service,
	missionID,
	pendingID string,
	terminal ledger.Event,
	validationProfile string,
) ([]reportILArtifactReceipt, error) {
	payload, err := reportilcontract.DecodeTerminalPayload(terminal.Payload)
	if err != nil {
		return nil, err
	}
	if payload.PendingEventID != pendingID {
		return nil, errors.New("provider terminal pending binding mismatch")
	}
	actual := make([]reportilcontract.Artifact, 0, len(payload.Bundle.Artifacts))
	byKind := map[string]artifactcontract.Raw{}
	receipts := make([]reportILArtifactReceipt, 0, len(payload.Bundle.Artifacts))
	for _, entry := range payload.Bundle.Artifacts {
		artifact, err := service.GetRawArtifact(ctx, entry.ArtifactID)
		if err != nil {
			return nil, err
		}
		if artifact.MissionID != missionID {
			return nil, errors.New("provider artifact mission binding mismatch")
		}
		actual = append(actual, reportilcontract.Artifact{
			ArtifactID: artifact.ArtifactID, MissionID: artifact.MissionID, MediaType: artifact.MediaType,
			Filename: artifact.Filename, ByteSize: artifact.ByteSize, SHA256: artifact.SHA256, Content: artifact.Content,
		})
		byKind[entry.Kind] = artifact
		receipts = append(receipts, reportILArtifactReceipt{
			Kind: entry.Kind, Artifact: entry.ArtifactID, MediaType: entry.MediaType, Filename: entry.Filename,
			Role: entry.Role, SHA256: entry.SHA256, ByteSize: entry.ByteSize,
		})
	}
	if err := reportilcontract.ValidateActual(payload.Bundle, actual); err != nil {
		return nil, err
	}
	var narrative reportilphase0.Narrative
	var document reportilphase0.Document
	var manifest reportilphase0.ProductManifest
	for value, content := range map[any][]byte{
		&narrative: byKind["narrative"].Content, &document: byKind["semantic_il"].Content,
		&manifest: byKind["manifest"].Content,
	} {
		decoder := json.NewDecoder(bytes.NewReader(content))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(value); err != nil {
			return nil, err
		}
	}
	if err := reportilphase0.ValidateBundle(reportilphase0.Bundle{
		SchemaVersion: reportilphase0.BundleSchemaVersion, Arm: "T1", Narrative: &narrative, Document: document,
	}); err != nil {
		return nil, err
	}
	markdown, _, err := reportilphase0.RenderMarkdown(document)
	if err != nil || !bytes.Equal(markdown, byKind["markdown"].Content) {
		return nil, fmt.Errorf("stored Markdown is not the final semantic IL projection: %v", err)
	}
	if manifest.ProjectionSHA256 != acceptanceSHA256(markdown) {
		return nil, errors.New("projection hash binding mismatch")
	}
	var terminalMeta struct {
		AgentModel           string `json:"agent_model"`
		AgentReasoningEffort string `json:"agent_reasoning_effort"`
	}
	if err := json.Unmarshal(terminal.Payload, &terminalMeta); err != nil {
		return nil, err
	}
	if manifest.SchemaVersion != reportilphase0.ProductManifestSchemaVersion || manifest.CompilerVersion != reportilphase0.CompilerVersion || manifest.PipelineFamily != reportilcontract.PipelineFamily || manifest.AuthoringMode != reportilphase0.AuthoringModeLongForm || manifest.ValidationProfile != validationProfile || manifest.MissionID != missionID || len(manifest.CatalogSHA256) != 64 || manifest.Provider != "codex" || manifest.Model != terminalMeta.AgentModel || manifest.Effort != terminalMeta.AgentReasoningEffort || manifest.DocumentID != document.DocumentID || manifest.RevisionID != document.RevisionID {
		return nil, errors.New("provider manifest identity binding mismatch")
	}
	if manifest.NarrativeSHA256 != byKind["narrative"].SHA256 || manifest.DocumentSHA256 != byKind["semantic_il"].SHA256 || len(manifest.Artifacts) != 5 {
		return nil, errors.New("provider manifest artifact binding mismatch")
	}
	if err := validateStoredLongFormAuthoringLineage(ctx, service, missionID, manifest.LongFormAuthoring); err != nil {
		return nil, err
	}
	if validationProfile == reportilphase0.ValidationProfileUnverified {
		if manifest.EditorialMemory != nil {
			return nil, errors.New("unverified provider manifest fabricated editorial-memory lineage")
		}
	} else {
		if manifest.EditorialMemory == nil {
			return nil, errors.New("verified provider manifest lacks editorial-memory lineage")
		}
		memoryArtifact, err := service.GetRawArtifact(ctx, manifest.EditorialMemory.ArtifactID)
		if err != nil {
			return nil, fmt.Errorf("load provider editorial memory: %w", err)
		}
		if memoryArtifact.MissionID != missionID ||
			memoryArtifact.MediaType != reportilcontract.EditorialMemoryMediaType ||
			memoryArtifact.SHA256 != manifest.EditorialMemory.SHA256 ||
			int(memoryArtifact.ByteSize) != manifest.EditorialMemory.ByteSize ||
			len(memoryArtifact.Content) != manifest.EditorialMemory.ByteSize ||
			manifest.EditorialMemory.Revision < 1 || manifest.EditorialMemory.Accounts < 1 {
			return nil, errors.New("provider manifest editorial-memory binding mismatch")
		}
		var editorialMemory reportilcontract.EditorialMemoryArtifact
		memoryDecoder := json.NewDecoder(bytes.NewReader(memoryArtifact.Content))
		memoryDecoder.DisallowUnknownFields()
		if err := memoryDecoder.Decode(&editorialMemory); err != nil {
			return nil, fmt.Errorf("decode provider editorial memory: %w", err)
		}
		if len(editorialMemory.Anchors) < 1 {
			return nil, errors.New("provider editorial memory lacks exact anchors")
		}
		if manifest.EditorialMemory.Accounts != len(editorialMemory.Accounts) {
			return nil, errors.New("provider manifest editorial-memory account count mismatch")
		}
	}
	selection := payload.Bundle.SourceSelection
	if selection == nil ||
		selection.Applied != manifest.SourceSelection.Applied ||
		selection.AcceptedSources != manifest.SourceSelection.AcceptedSources ||
		selection.UsableSources != manifest.SourceSelection.UsableSources ||
		selection.SelectedSources != manifest.SourceSelection.SelectedSources ||
		selection.SupplementalSources != manifest.SourceSelection.SupplementalSources ||
		selection.ExcludedUnusableSources != manifest.SourceSelection.ExcludedUnusableSources ||
		selection.ExcludedBudgetSources != manifest.SourceSelection.ExcludedBudgetSources {
		return nil, errors.New("provider terminal source selection does not match the manifest")
	}
	manifestKinds := map[string]bool{}
	for _, entry := range manifest.Artifacts {
		artifact, ok := byKind[entry.Kind]
		if !ok || manifestKinds[entry.Kind] || entry.ArtifactID != artifact.ArtifactID || entry.MediaType != artifact.MediaType || entry.Filename != artifact.Filename || entry.SHA256 != artifact.SHA256 || entry.ByteSize != int(artifact.ByteSize) {
			return nil, fmt.Errorf("provider manifest %s entry does not match stored bytes", entry.Kind)
		}
		manifestKinds[entry.Kind] = true
	}
	for _, kind := range []string{"narrative", "semantic_il", "markdown", "html", "pdf"} {
		if !manifestKinds[kind] {
			return nil, fmt.Errorf("provider manifest lacks %s", kind)
		}
	}
	html := strings.ToLower(string(byKind["html"].Content))
	for _, required := range []string{"<!doctype html>", "content-security-policy", "default-src 'none'", "<main>"} {
		if !strings.Contains(html, required) {
			return nil, fmt.Errorf("stored HTML lacks %q", required)
		}
	}
	for _, forbidden := range []string{"<script", "<link", "src=\"http:", "src=\"https:", "url(http", "url(https"} {
		if strings.Contains(html, forbidden) {
			return nil, fmt.Errorf("stored HTML contains external or active dependency %q", forbidden)
		}
	}
	if info, err := pdfdocument.Inspect(byKind["pdf"].Content); err != nil || info.PageCount < 1 {
		return nil, fmt.Errorf("stored PDF inspection failed: pages=%d err=%v", info.PageCount, err)
	}
	sort.Slice(receipts, func(i, j int) bool { return receipts[i].Kind < receipts[j].Kind })
	return receipts, nil
}

func validateStoredLongFormAuthoringLineage(
	ctx context.Context,
	service *app.Service,
	missionID string,
	lineage *reportilphase0.LongFormAuthoringReceipt,
) error {
	if lineage == nil || lineage.Parts < reportilcontract.MinLongFormParts ||
		lineage.Sections < reportilcontract.MinLongFormSections ||
		lineage.SectionAuthors != lineage.Sections || lineage.PartEditors != lineage.Parts ||
		len(lineage.SectionArtifacts) != lineage.Sections || len(lineage.PartArtifacts) != lineage.Parts ||
		len(lineage.Finalizations) < 1 || lineage.Finalizations[len(lineage.Finalizations)-1].Artifact != lineage.Final {
		return errors.New("provider manifest lacks complete long-form authoring lineage")
	}
	ordered := make([]reportilphase0.LongFormArtifactReceipt, 0, 1+len(lineage.SectionArtifacts)+len(lineage.PartArtifacts)+len(lineage.Finalizations))
	ordered = append(ordered, lineage.Plan)
	ordered = append(ordered, lineage.SectionArtifacts...)
	ordered = append(ordered, lineage.PartArtifacts...)
	finalizationStart := len(ordered)
	wantFinalizationStages := []string{"il_long_form_final"}
	switch len(lineage.Finalizations) {
	case 1:
	case 2:
		wantFinalizationStages = append(wantFinalizationStages, "il_reader")
	case 3:
		wantFinalizationStages = append(wantFinalizationStages, "il_reader", "il_continuity")
	default:
		return errors.New("provider manifest long-form finalization inventory is invalid")
	}
	for index, finalization := range lineage.Finalizations {
		if finalization.Stage != wantFinalizationStages[index] {
			return errors.New("provider manifest long-form finalization stage is invalid")
		}
		ordered = append(ordered, finalization.Artifact)
	}
	seen := make(map[string]reportilphase0.LongFormArtifactReceipt, len(ordered))
	for index, receipt := range ordered {
		previous, duplicate := seen[receipt.ArtifactID]
		reused := false
		if index >= finalizationStart {
			finalization := lineage.Finalizations[index-finalizationStart]
			reused = finalization.Reused
			if reused != (duplicate && index > finalizationStart && ordered[index-1].ArtifactID == receipt.ArtifactID) {
				return fmt.Errorf("provider manifest long-form finalization receipt %d has an invalid reuse marker", index-finalizationStart)
			}
		}
		if !strings.HasPrefix(receipt.ArtifactID, "art_") || len(receipt.SHA256) != 64 || receipt.ByteSize < 1 || duplicate && !reused {
			return fmt.Errorf("provider manifest long-form artifact receipt %d is invalid (previous=%#v): %#v", index, previous, receipt)
		}
		seen[receipt.ArtifactID] = receipt
		artifact, err := service.GetRawArtifact(ctx, receipt.ArtifactID)
		if err != nil {
			return fmt.Errorf("load long-form artifact %d: %w", index, err)
		}
		sum := sha256.Sum256(artifact.Content)
		mediaType := reportilcontract.AuthorDocumentMediaType
		producerID := reportilcontract.LongFormDocumentFinalizeTool
		switch receipt.Stage {
		case "il_long_form_plan":
			mediaType = reportilcontract.LongFormPlanMediaType
			producerID = reportilcontract.LongFormPlanSubmitTool
		case "il_long_form_section", "il_long_form_part", "il_long_form_final":
		case "il_reader", "il_continuity":
			producerID = reportilcontract.AuthorDocumentFinalizeTool
		default:
			return fmt.Errorf("provider manifest long-form artifact receipt %d has an invalid stage", index)
		}
		if artifact.MissionID != missionID || artifact.MediaType != mediaType ||
			artifact.Producer.Type != "mcp_tool" || artifact.Producer.ID != producerID ||
			artifact.SHA256 != receipt.SHA256 || hex.EncodeToString(sum[:]) != receipt.SHA256 ||
			int(artifact.ByteSize) != receipt.ByteSize || len(artifact.Content) != receipt.ByteSize {
			return fmt.Errorf("stored long-form artifact %d does not match the manifest receipt", index)
		}
		decoder := json.NewDecoder(bytes.NewReader(artifact.Content))
		decoder.DisallowUnknownFields()
		if receipt.Stage == "il_long_form_plan" {
			var plan reportilcontract.LongFormPlan
			if err := decoder.Decode(&plan); err != nil {
				return fmt.Errorf("decode stored long-form plan: %w", err)
			}
			if len(plan.Parts) != lineage.Parts || reportilcontract.LongFormPlanSectionCount(plan) != lineage.Sections {
				return errors.New("stored long-form plan inventory does not match the manifest")
			}
		} else {
			var document reportilcontract.AuthorDocument
			if err := decoder.Decode(&document); err != nil {
				return fmt.Errorf("decode stored long-form document %d: %w", index, err)
			}
			sectionCount := 0
			for _, part := range document.Parts {
				sectionCount += len(part.Sections)
			}
			if index == len(ordered)-1 && (len(document.Parts) != lineage.Parts || sectionCount != lineage.Sections) {
				return errors.New("stored final long-form inventory does not match the manifest")
			}
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			return fmt.Errorf("stored long-form artifact %d contains multiple JSON values", index)
		}
	}
	return nil
}

func reportILAcceptanceArtifactByKind(ctx context.Context, service *app.Service, terminal ledger.Event, kind string) (artifactcontract.Raw, error) {
	payload, err := reportilcontract.DecodeTerminalPayload(terminal.Payload)
	if err != nil {
		return artifactcontract.Raw{}, err
	}
	for _, entry := range payload.Bundle.Artifacts {
		if entry.Kind == kind {
			return service.GetRawArtifact(ctx, entry.ArtifactID)
		}
	}
	return artifactcontract.Raw{}, fmt.Errorf("provider bundle lacks %s", kind)
}

func validateReportILProviderCalls(calls []reportILProviderCallReceipt) error {
	counts, err := validateReportILProviderCallContract(calls)
	if err != nil {
		return err
	}
	if counts["il_editorial_memory"] != 1 {
		return fmt.Errorf("report editorial-memory calls = %d, want 1", counts["il_editorial_memory"])
	}
	if counts["il_narrative"] != 1 {
		return fmt.Errorf("report author calls = %d, want 1", counts["il_narrative"])
	}
	if counts["il_continuity"] != 1 {
		return fmt.Errorf("report continuity editor calls = %d, want 1", counts["il_continuity"])
	}
	if counts["il_reader"] != 1 {
		return fmt.Errorf("report reader calls = %d, want 1", counts["il_reader"])
	}
	if counts["il_source_selection"] > 2 || len(calls) != 4+counts["il_source_selection"] {
		return fmt.Errorf("provider calls = %d, want up to two source-selection attempts plus one editorial-memory author, one report author, one continuity editor, and one reader", len(calls))
	}
	if counts["il_document"] != 0 || counts["il_flow"] != 0 {
		return fmt.Errorf("server-owned compile and render made provider calls")
	}
	return nil
}

func validateReportILFailedProviderCalls(calls []reportILProviderCallReceipt, failedStage string) error {
	counts, err := validateReportILProviderCallContract(calls)
	if err != nil {
		return err
	}
	order := []string{"il_source_selection", "il_editorial_memory", "il_narrative", "il_reader", "il_continuity"}
	failedIndex := -1
	for index, stage := range order {
		if stage == failedStage {
			failedIndex = index
			break
		}
	}
	if failedIndex < 0 {
		return fmt.Errorf("unsupported failed provider stage %q", failedStage)
	}
	for index, stage := range order {
		switch {
		case index < failedIndex && (stage == "il_editorial_memory" || stage == "il_narrative" || stage == "il_continuity") && counts[stage] != 1:
			return fmt.Errorf("provider stage %s did not complete before %s failure", stage, failedStage)
		case index == failedIndex && counts[stage] < 1:
			return fmt.Errorf("failed provider stage %s made no call", stage)
		case index > failedIndex && counts[stage] != 0:
			return fmt.Errorf("provider stage %s ran after %s failure", stage, failedStage)
		}
	}
	return nil
}

func validateLongFormReportILFailedProviderCalls(calls []reportILProviderCallReceipt, failedStage string) error {
	allowedFailures := map[string]bool{
		"il_long_form_plan":     true,
		"il_long_form_sections": true,
		"il_long_form_parts":    true,
		"il_long_form_final":    true,
	}
	if !allowedFailures[failedStage] || len(calls) < 2 || calls[0].Stage != "il_editorial_memory" {
		return fmt.Errorf("long-form failure calls = %#v", calls)
	}
	stageReached := false
	stageAttempts := map[string]int{}
	for index, call := range calls {
		stageAttempts[call.Stage]++
		if call.Ordinal != index+1 || call.Attempt != stageAttempts[call.Stage] || call.Model == "" || call.ReasoningEffort == "" ||
			call.CapabilityProfile != string(agentcapability.ProfileReportILSourceV1) || call.ProfileRevision != agentcapability.RevisionV1 ||
			call.MCPMode != "source_read_only" || call.DisableTools || !call.IgnoreUserConfig || !call.EphemeralSession ||
			!call.ReplaceMCPTools || !call.SourceBinding || len(call.SourceCatalogSHA) != 64 || call.SourceStage != call.Stage ||
			call.SourceAttempt != 1 || call.OutputSchemaBytes != 0 {
			return errors.New("failed long-form request violated the fixed provider contract")
		}
		var wantTools []string
		switch call.Stage {
		case "il_editorial_memory":
			wantTools = []string{
				reportilcontract.SourceListTool, reportilcontract.SourceReadTool,
				reportilcontract.SourceQuoteRegisterTool, reportilcontract.EditorialMemoryStartTool,
				reportilcontract.EditorialMemoryAppendTool, reportilcontract.EditorialMemoryReadTool,
				reportilcontract.EditorialMemoryFinalizeTool,
			}
		case "il_long_form_plan":
			wantTools = []string{reportilcontract.EditorialMemoryReadTool, reportilcontract.LongFormPlanSubmitTool}
			stageReached = stageReached || failedStage == "il_long_form_plan"
		case "il_long_form_section":
			wantTools = []string{
				reportilcontract.EditorialMemoryReadTool, reportilcontract.LongFormPlanReadTool,
				reportilcontract.LongFormDocumentStartTool, reportilcontract.LongFormDocumentAppendTool,
				reportilcontract.LongFormDocumentReadTool, reportilcontract.LongFormDocumentReplaceTool,
				reportilcontract.LongFormDocumentFinalizeTool,
			}
			stageReached = stageReached || failedStage == "il_long_form_sections"
		case "il_long_form_part", "il_long_form_final":
			wantTools = []string{
				reportilcontract.EditorialMemoryReadTool, reportilcontract.LongFormPlanReadTool,
				reportilcontract.LongFormDocumentStartTool, reportilcontract.LongFormDocumentReadTool,
				reportilcontract.LongFormDocumentReplaceTool, reportilcontract.LongFormDocumentFinalizeTool,
			}
			stageReached = stageReached || failedStage == "il_long_form_parts" && call.Stage == "il_long_form_part" ||
				failedStage == "il_long_form_final" && call.Stage == "il_long_form_final"
		default:
			return fmt.Errorf("unexpected long-form failure provider stage %q", call.Stage)
		}
		if !reflect.DeepEqual(call.ExtraMCPTools, wantTools) {
			return errors.New("failed long-form request used the wrong tool surface")
		}
	}
	if !stageReached {
		return fmt.Errorf("failed long-form stage %q made no provider call", failedStage)
	}
	return nil
}

func validateLongFormReportILProviderCalls(calls []reportILProviderCallReceipt, profile string) error {
	wantStages := []string{
		"il_long_form_plan",
		"il_long_form_section", "il_long_form_section", "il_long_form_section",
		"il_long_form_section", "il_long_form_section", "il_long_form_section",
		"il_long_form_part", "il_long_form_part", "il_long_form_final",
	}
	withMemory := profile != reportilphase0.ValidationProfileUnverified
	if withMemory {
		wantStages = append([]string{"il_editorial_memory"}, wantStages...)
	}
	if profile == reportilphase0.ValidationProfileExploratory || profile == reportilphase0.ValidationProfileStrict {
		wantStages = append(wantStages, "il_reader")
	}
	if profile == reportilphase0.ValidationProfileStrict {
		wantStages = append(wantStages, "il_continuity")
	}
	if len(calls) != len(wantStages) {
		return fmt.Errorf("provider call count = %d, want %d", len(calls), len(wantStages))
	}
	attempts := map[string]int{}
	for index, call := range calls {
		if call.Ordinal != index+1 || call.Stage != wantStages[index] {
			return fmt.Errorf("provider stage order is invalid at call %d", index+1)
		}
		attempts[call.Stage]++
		if call.Attempt != attempts[call.Stage] || call.Model == "" || call.ReasoningEffort == "" ||
			call.CapabilityProfile != string(agentcapability.ProfileReportILSourceV1) || call.ProfileRevision != agentcapability.RevisionV1 ||
			call.MCPMode != "source_read_only" || call.DisableTools || !call.IgnoreUserConfig || !call.EphemeralSession ||
			!call.ReplaceMCPTools || !call.SourceBinding || len(call.SourceCatalogSHA) != 64 || call.SourceStage != call.Stage ||
			call.SourceAttempt != 1 || call.OutputSchemaBytes != 0 {
			return fmt.Errorf("provider request %d violated the fixed long-form contract", call.Ordinal)
		}
		memoryTools := []string{reportilcontract.EditorialMemoryReadTool}
		var wantTools []string
		switch call.Stage {
		case "il_editorial_memory":
			wantTools = []string{
				reportilcontract.SourceListTool, reportilcontract.SourceReadTool,
				reportilcontract.SourceQuoteRegisterTool, reportilcontract.EditorialMemoryStartTool,
				reportilcontract.EditorialMemoryAppendTool, reportilcontract.EditorialMemoryReadTool,
				reportilcontract.EditorialMemoryFinalizeTool,
			}
		case "il_long_form_plan":
			if withMemory {
				wantTools = append(memoryTools, reportilcontract.LongFormPlanSubmitTool)
			} else {
				wantTools = []string{reportilcontract.SourceListTool, reportilcontract.SourceReadTool, reportilcontract.LongFormPlanSubmitTool}
			}
		case "il_long_form_section":
			if withMemory {
				wantTools = append([]string(nil), memoryTools...)
			} else {
				wantTools = []string{reportilcontract.SourceReadTool}
			}
			wantTools = append(wantTools,
				reportilcontract.LongFormPlanReadTool, reportilcontract.LongFormDocumentStartTool,
				reportilcontract.LongFormDocumentAppendTool, reportilcontract.LongFormDocumentReadTool,
				reportilcontract.LongFormDocumentReplaceTool, reportilcontract.LongFormDocumentCorrectBlockTool,
				reportilcontract.LongFormDocumentFinalizeTool,
			)
		case "il_long_form_part", "il_long_form_final":
			if withMemory {
				wantTools = append([]string(nil), memoryTools...)
			}
			wantTools = append(wantTools,
				reportilcontract.LongFormPlanReadTool, reportilcontract.LongFormDocumentStartTool,
				reportilcontract.LongFormDocumentReadTool, reportilcontract.LongFormDocumentReplaceTool,
				reportilcontract.LongFormDocumentFinalizeTool,
			)
		case "il_reader":
			wantTools = []string{
				reportilcontract.EditorialMemoryReadTool, reportilcontract.AuthorDocumentOpenTool,
				reportilcontract.AuthorDocumentReadTool, reportilcontract.AuthorDocumentEditTextTool,
				reportilcontract.AuthorDocumentFinalizeTool,
			}
		case "il_continuity":
			wantTools = []string{
				reportilcontract.EditorialMemoryReadTool, reportilcontract.AuthorDocumentOpenTool,
				reportilcontract.AuthorDocumentReadTool, reportilcontract.AuthorDocumentReviseBlockTool,
				reportilcontract.AuthorDocumentFinalizeTool,
			}
		}
		if !reflect.DeepEqual(call.ExtraMCPTools, wantTools) {
			return fmt.Errorf("provider request %d used the wrong long-form tools", call.Ordinal)
		}
	}
	return nil
}

func validateReportILProviderCallContract(calls []reportILProviderCallReceipt) (map[string]int, error) {
	if len(calls) < 1 || len(calls) > 6 {
		return nil, fmt.Errorf("provider call count = %d, want 1..6", len(calls))
	}
	counts := map[string]int{}
	lastOrder := -1
	order := map[string]int{"il_source_selection": 0, "il_editorial_memory": 1, "il_narrative": 2, "il_reader": 3, "il_continuity": 4}
	schemaByStage := map[string]string{}
	for _, call := range calls {
		stageOrder, ok := order[call.Stage]
		if !ok || stageOrder < lastOrder {
			return nil, fmt.Errorf("provider stage order is invalid at %q", call.Stage)
		}
		lastOrder = stageOrder
		counts[call.Stage]++
		maxAttempts := 2
		if call.Stage == "il_editorial_memory" || call.Stage == "il_narrative" || call.Stage == "il_continuity" || call.Stage == "il_reader" {
			maxAttempts = 1
		}
		if call.Attempt != counts[call.Stage] || call.Attempt > maxAttempts {
			return nil, fmt.Errorf("provider stage %s attempt = %d", call.Stage, call.Attempt)
		}
		if call.Model == "" || call.ReasoningEffort == "" || call.ProfileRevision != agentcapability.RevisionV1 || !call.IgnoreUserConfig || !call.EphemeralSession || !call.ReplaceMCPTools || len(call.OutputSchemaSHA256) != 64 {
			return nil, fmt.Errorf("provider request %d violated the fixed isolation contract", call.Ordinal)
		}
		var wantTools []string
		switch call.Stage {
		case "il_source_selection":
			wantTools = []string{
				reportilcontract.SourceListTool,
				reportilcontract.SourceReadTool,
			}
		case "il_editorial_memory":
			wantTools = []string{
				reportilcontract.SourceListTool,
				reportilcontract.SourceReadTool,
				reportilcontract.SourceQuoteRegisterTool,
				reportilcontract.EditorialMemoryStartTool,
				reportilcontract.EditorialMemoryAppendTool,
				reportilcontract.EditorialMemoryReadTool,
				reportilcontract.EditorialMemoryFinalizeTool,
			}
		case "il_narrative":
			wantTools = []string{
				reportilcontract.EditorialMemoryReadTool,
				reportilcontract.AuthorDocumentStartTool,
				reportilcontract.AuthorDocumentAppendTool,
				reportilcontract.AuthorDocumentReadTool,
				reportilcontract.AuthorDocumentReplaceTool,
				reportilcontract.AuthorDocumentFinalizeTool,
			}
		case "il_continuity":
			wantTools = []string{
				reportilcontract.EditorialMemoryReadTool,
				reportilcontract.AuthorDocumentOpenTool,
				reportilcontract.AuthorDocumentReadTool,
				reportilcontract.AuthorDocumentReviseBlockTool,
				reportilcontract.AuthorDocumentFinalizeTool,
			}
		case "il_reader":
			wantTools = []string{
				reportilcontract.EditorialMemoryReadTool,
				reportilcontract.AuthorDocumentOpenTool,
				reportilcontract.AuthorDocumentReadTool,
				reportilcontract.AuthorDocumentEditTextTool,
				reportilcontract.AuthorDocumentFinalizeTool,
			}
		}
		if call.CapabilityProfile != string(agentcapability.ProfileReportILSourceV1) || call.MCPMode != "source_read_only" || call.DisableTools || !reflect.DeepEqual(call.ExtraMCPTools, wantTools) || !call.SourceBinding || len(call.SourceCatalogSHA) != 64 || call.SourceStage != call.Stage || call.SourceAttempt != call.Attempt {
			return nil, fmt.Errorf("provider source request %d violated the source-bound tool contract", call.Ordinal)
		}
		if (call.Stage == "il_source_selection") != (call.OutputSchemaBytes > 0) {
			return nil, fmt.Errorf("provider request %d used the wrong output contract", call.Ordinal)
		}
		if previous := schemaByStage[call.Stage]; previous != "" && previous != call.OutputSchemaSHA256 {
			return nil, fmt.Errorf("provider repair changed %s output schema", call.Stage)
		}
		schemaByStage[call.Stage] = call.OutputSchemaSHA256
	}
	return counts, nil
}

func TestValidateReportILProviderCallsAllowsOptionalSelectionAuthorAndReader(t *testing.T) {
	call := func(stage string, attempt int) reportILProviderCallReceipt {
		tools := []string{reportilcontract.SourceListTool, reportilcontract.SourceReadTool}
		schemaBytes := 1
		switch stage {
		case "il_editorial_memory":
			tools = append(tools,
				reportilcontract.SourceQuoteRegisterTool,
				reportilcontract.EditorialMemoryStartTool,
				reportilcontract.EditorialMemoryAppendTool,
				reportilcontract.EditorialMemoryReadTool,
				reportilcontract.EditorialMemoryFinalizeTool,
			)
			schemaBytes = 0
		case "il_narrative":
			tools = []string{
				reportilcontract.EditorialMemoryReadTool,
				reportilcontract.AuthorDocumentStartTool,
				reportilcontract.AuthorDocumentAppendTool,
				reportilcontract.AuthorDocumentReadTool,
				reportilcontract.AuthorDocumentReplaceTool,
				reportilcontract.AuthorDocumentFinalizeTool,
			}
			schemaBytes = 0
		case "il_continuity":
			tools = []string{
				reportilcontract.EditorialMemoryReadTool,
				reportilcontract.AuthorDocumentOpenTool,
				reportilcontract.AuthorDocumentReadTool,
				reportilcontract.AuthorDocumentReviseBlockTool,
				reportilcontract.AuthorDocumentFinalizeTool,
			}
			schemaBytes = 0
		case "il_reader":
			tools = []string{
				reportilcontract.EditorialMemoryReadTool,
				reportilcontract.AuthorDocumentOpenTool,
				reportilcontract.AuthorDocumentReadTool,
				reportilcontract.AuthorDocumentEditTextTool,
				reportilcontract.AuthorDocumentFinalizeTool,
			}
			schemaBytes = 0
		}
		return reportILProviderCallReceipt{
			Ordinal:            attempt,
			Stage:              stage,
			Attempt:            attempt,
			Model:              "gpt-5.6-luna",
			ReasoningEffort:    "xhigh",
			CapabilityProfile:  string(agentcapability.ProfileReportILSourceV1),
			ProfileRevision:    agentcapability.RevisionV1,
			MCPMode:            "source_read_only",
			IgnoreUserConfig:   true,
			EphemeralSession:   true,
			ReplaceMCPTools:    true,
			ExtraMCPTools:      tools,
			SourceBinding:      true,
			SourceCatalogSHA:   strings.Repeat("a", 64),
			SourceStage:        stage,
			SourceAttempt:      attempt,
			OutputSchemaSHA256: strings.Repeat("b", 64),
			OutputSchemaBytes:  schemaBytes,
		}
	}
	cases := []struct {
		name    string
		calls   []reportILProviderCallReceipt
		wantErr bool
	}{
		{name: "memory author reader and continuity", calls: []reportILProviderCallReceipt{call("il_editorial_memory", 1), call("il_narrative", 1), call("il_reader", 1), call("il_continuity", 1)}},
		{name: "selection memory author reader continuity", calls: []reportILProviderCallReceipt{call("il_source_selection", 1), call("il_editorial_memory", 1), call("il_narrative", 1), call("il_reader", 1), call("il_continuity", 1)}},
		{name: "selection repair memory author reader continuity", calls: []reportILProviderCallReceipt{call("il_source_selection", 1), call("il_source_selection", 2), call("il_editorial_memory", 1), call("il_narrative", 1), call("il_reader", 1), call("il_continuity", 1)}},
		{name: "missing memory rejected", calls: []reportILProviderCallReceipt{call("il_narrative", 1), call("il_reader", 1), call("il_continuity", 1)}, wantErr: true},
		{name: "missing continuity rejected", calls: []reportILProviderCallReceipt{call("il_editorial_memory", 1), call("il_narrative", 1), call("il_reader", 1)}, wantErr: true},
		{name: "missing reader rejected", calls: []reportILProviderCallReceipt{call("il_editorial_memory", 1), call("il_narrative", 1), call("il_continuity", 1)}, wantErr: true},
		{name: "memory repair rejected", calls: []reportILProviderCallReceipt{call("il_editorial_memory", 1), call("il_editorial_memory", 2), call("il_narrative", 1), call("il_reader", 1), call("il_continuity", 1)}, wantErr: true},
		{name: "author repair rejected", calls: []reportILProviderCallReceipt{call("il_editorial_memory", 1), call("il_narrative", 1), call("il_narrative", 2), call("il_reader", 1), call("il_continuity", 1)}, wantErr: true},
		{name: "continuity repair rejected", calls: []reportILProviderCallReceipt{call("il_editorial_memory", 1), call("il_narrative", 1), call("il_reader", 1), call("il_continuity", 1), call("il_continuity", 2)}, wantErr: true},
		{name: "reader repair rejected", calls: []reportILProviderCallReceipt{call("il_editorial_memory", 1), call("il_narrative", 1), call("il_reader", 1), call("il_reader", 2), call("il_continuity", 1)}, wantErr: true},
		{name: "flow rejected", calls: []reportILProviderCallReceipt{call("il_editorial_memory", 1), call("il_narrative", 1), call("il_flow", 1)}, wantErr: true},
		{name: "selection after memory rejected", calls: []reportILProviderCallReceipt{call("il_editorial_memory", 1), call("il_source_selection", 1), call("il_narrative", 1), call("il_reader", 1), call("il_continuity", 1)}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for index := range tc.calls {
				tc.calls[index].Ordinal = index + 1
			}
			err := validateReportILProviderCalls(tc.calls)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func countReportILRawSourceReadingCalls(calls []reportILProviderCallReceipt) int {
	count := 0
	for _, call := range calls {
		if call.SourceBinding && (call.Stage == "il_source_selection" || call.Stage == "il_editorial_memory") {
			count++
		}
	}
	return count
}

func validateReportILFailureUsage(terminal ledger.Event, events []ledger.Event, calls []reportILProviderCallReceipt) error {
	type failureUsagePayload struct {
		Reason  reportexecution.ProviderFailureReason       `json:"safe_failure_reason"`
		Receipt reportexecution.ProviderFailureUsageReceipt `json:"provider_usage_receipt"`
	}
	var payload failureUsagePayload
	if err := json.Unmarshal(terminal.Payload, &payload); err != nil {
		return err
	}
	if !payload.Reason.Valid() {
		return fmt.Errorf("failure reason %q is not closed", payload.Reason)
	}
	if len(payload.Receipt.Attempts) != len(calls) {
		return fmt.Errorf("failure usage attempts = %d, provider calls = %d", len(payload.Receipt.Attempts), len(calls))
	}
	for index, attempt := range payload.Receipt.Attempts {
		call := calls[index]
		if string(attempt.Stage) != call.Stage || attempt.Attempt != call.Attempt {
			return fmt.Errorf("failure usage attempt %d does not match provider call", index+1)
		}
	}
	var terminalPayload map[string]any
	if err := json.Unmarshal(terminal.Payload, &terminalPayload); err != nil {
		return err
	}
	companionID, _ := terminalPayload["stage_failure_event_id"].(string)
	if companionID == "" {
		return errors.New("failure terminal lacks stage companion ID")
	}
	var companionPayload failureUsagePayload
	found := false
	for _, event := range events {
		if event.EventID != companionID {
			continue
		}
		if err := json.Unmarshal(event.Payload, &companionPayload); err != nil {
			return err
		}
		found = true
		break
	}
	if !found {
		return errors.New("failure stage companion is missing")
	}
	if !reflect.DeepEqual(payload, companionPayload) {
		return errors.New("failure stage companion and terminal usage receipts differ")
	}
	return nil
}

func writeReportILAcceptanceCodexWrapper(directory string) (string, error) {
	path := filepath.Join(directory, "codex-report-il-transport")
	content := `#!/bin/sh
set -eu
receipt=${PLASMA_IL_TRANSPORT_RECEIPT:?}
real=${PLASMA_IL_REAL_CODEX:?}
count_file="${receipt}.count"
ordinal=0
if [ -f "$count_file" ]; then ordinal=$(tr -d ' ' < "$count_file"); fi
ordinal=$((ordinal + 1))
printf '%s\n' "$ordinal" > "$count_file"
ephemeral=0; ignore_user=0; json=0; output_schema=0; schema_path=''; readonly=0
skip_git=0; ignore_rules=0; stdin_arg=0; model=''; effort=''; profile=0; mcp=0
previous=''
for arg in "$@"; do
  if [ "$previous" = '--model' ]; then model=$arg; fi
  if [ "$previous" = '--output-schema' ]; then schema_path=$arg; output_schema=1; fi
  if [ "$previous" = '--sandbox' ] && [ "$arg" = 'read-only' ]; then readonly=1; fi
  case "$arg" in
    --ephemeral) ephemeral=1 ;;
    --ignore-user-config) ignore_user=1 ;;
    --json) json=1 ;;
    --skip-git-repo-check) skip_git=1 ;;
    --ignore-rules) ignore_rules=1 ;;
    -) stdin_arg=1 ;;
    'model_reasoning_effort="xhigh"') effort=xhigh ;;
    features.tool_suggest=false|features.recommended_plugins=false|tools.experimental_request_user_input.enabled=false|features.apps=false|features.plugins=false|features.multi_agent=false|features.shell_tool=false|features.view_image=false|tools.update_plan.enabled=false|skills.include_instructions=false|project_doc_max_bytes=0) profile=$((profile + 1)) ;;
    mcp_servers.*) mcp=$((mcp + 1)) ;;
  esac
  previous=$arg
done
schema_sha=''; schema_bytes=0
if [ "$output_schema" -eq 1 ]; then
  schema_sha=$(shasum -a 256 "$schema_path" | cut -d ' ' -f 1)
  schema_bytes=$(wc -c < "$schema_path" | tr -d ' ')
fi
printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
  "$ordinal" "$ephemeral" "$ignore_user" "$json" "$output_schema" "$schema_sha" "$schema_bytes" \
  "$readonly" "$skip_git" "$ignore_rules" "$stdin_arg" "$model" "$effort" "$profile" "$mcp" >> "$receipt"
unset PLASMA_IL_TRANSPORT_RECEIPT PLASMA_IL_REAL_CODEX
exec "$real" "$@"
`
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		return "", err
	}
	return path, nil
}

func readReportILTransportReceipts(path string) ([]reportILTransportReceipt, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	result := make([]reportILTransportReceipt, 0, len(lines))
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) != 15 {
			return nil, errors.New("malformed redacted transport receipt")
		}
		ordinal, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, err
		}
		schemaBytes, err := strconv.Atoi(fields[6])
		if err != nil {
			return nil, err
		}
		profileConfigs, err := strconv.Atoi(fields[13])
		if err != nil {
			return nil, err
		}
		mcpConfigs, err := strconv.Atoi(fields[14])
		if err != nil {
			return nil, err
		}
		result = append(result, reportILTransportReceipt{
			Ordinal: ordinal, Ephemeral: fields[1] == "1", IgnoreUserConfig: fields[2] == "1",
			JSON: fields[3] == "1", OutputSchema: fields[4] == "1", SchemaSHA256: fields[5], SchemaBytes: schemaBytes,
			ReadOnlySandbox: fields[7] == "1", SkipGitCheck: fields[8] == "1", IgnoreRules: fields[9] == "1",
			Stdin: fields[10] == "1", Model: fields[11], Effort: fields[12], ProfileConfigs: profileConfigs, MCPConfigs: mcpConfigs,
		})
	}
	return result, nil
}

func validateReportILTransport(calls []reportILProviderCallReceipt, transport []reportILTransportReceipt) error {
	if len(calls) != len(transport) {
		return fmt.Errorf("provider/transport call count mismatch: %d/%d", len(calls), len(transport))
	}
	for index, item := range transport {
		call := calls[index]
		wantMCPConfigs := 0
		if call.SourceBinding {
			wantMCPConfigs = 8
		}
		if item.Ordinal != call.Ordinal || !item.Ephemeral || !item.IgnoreUserConfig || !item.JSON || !item.OutputSchema || !item.ReadOnlySandbox || !item.SkipGitCheck || !item.IgnoreRules || !item.Stdin || item.Model != "gpt-5.6-luna" || item.Effort != "xhigh" || item.ProfileConfigs != 11 || item.MCPConfigs != wantMCPConfigs || item.SchemaSHA256 != call.OutputSchemaSHA256 || item.SchemaBytes != call.OutputSchemaBytes {
			return fmt.Errorf("redacted Codex transport %d violated the product contract", item.Ordinal)
		}
	}
	return nil
}

func safeReportILFailureReceipt(event ledger.Event) struct {
	FailedStage        string
	SafeErrorClass     string
	SafeFailureReason  string
	SafeValidationCode string
} {
	var payload struct {
		FailedStage        string `json:"failed_stage_kind"`
		SafeErrorClass     string `json:"safe_error_class"`
		SafeFailureReason  string `json:"safe_failure_reason"`
		SafeValidationCode string `json:"safe_validation_code"`
	}
	_ = json.Unmarshal(event.Payload, &payload)
	return struct {
		FailedStage        string
		SafeErrorClass     string
		SafeFailureReason  string
		SafeValidationCode string
	}{payload.FailedStage, payload.SafeErrorClass, payload.SafeFailureReason, payload.SafeValidationCode}
}

func scanReportILAcceptancePrivacy(events []ledger.Event, service *app.Service, missionID string) error {
	for _, event := range events {
		if event.EventType == "mcp.tool.called" {
			if err := validateReportILSourceTracePrivacy(event, missionID); err != nil {
				return err
			}
			continue
		}
		text := strings.ToLower(string(event.Payload))
		for _, forbidden := range []string{"agent_session_id", "previous_agent_session_id", "raw_response", "provider_log", "\"prompt\"", "authorization:", "bearer ", ".ts.net", "localhost", "/users/"} {
			if strings.Contains(text, forbidden) {
				return fmt.Errorf("ledger privacy scan found forbidden material %q in %s", forbidden, event.EventType)
			}
		}
	}
	artifacts, err := service.ListRawArtifacts(context.Background(), missionID)
	if err != nil {
		return err
	}
	for _, artifact := range artifacts {
		text := strings.ToLower(string(artifact.Content))
		for _, forbidden := range []string{"authorization:", "bearer ", ".ts.net", "localhost", "/users/", "file:///"} {
			if strings.Contains(text, forbidden) {
				return fmt.Errorf("artifact privacy scan found forbidden material %q in %s", forbidden, artifact.Filename)
			}
		}
	}
	return nil
}

func validateReportILSourceTracePrivacy(event ledger.Event, missionID string) error {
	var payload struct {
		ToolName      string         `json:"tool_name"`
		ToolSessionID string         `json:"tool_session_id"`
		MissionID     string         `json:"mission_id"`
		Success       bool           `json:"success"`
		Arguments     map[string]any `json:"arguments"`
		Result        map[string]any `json:"result"`
		IOMetrics     struct {
			WriteKind            string `json:"write_kind"`
			WorkspaceID          string `json:"workspace_id"`
			ArtifactID           string `json:"artifact_id"`
			SHA256               string `json:"sha256"`
			ByteSize             int    `json:"byte_size"`
			Revision             int    `json:"revision"`
			Accounts             int    `json:"accounts"`
			Parts                int    `json:"parts"`
			Sections             int    `json:"sections"`
			Finalized            bool   `json:"finalized"`
			SourceKey            string `json:"source_key"`
			CatalogSHA256        string `json:"catalog_sha256"`
			ReportILStage        string `json:"report_il_stage"`
			SourceReceipt        string `json:"source_receipt"`
			SourceOffset         int    `json:"source_offset"`
			SourceByteSize       int    `json:"source_byte_size"`
			SourceSHA256         string `json:"source_sha256"`
			ReturnedOffset       int    `json:"returned_offset"`
			ReturnedContentBytes int    `json:"returned_content_bytes"`
			ContentLength        int    `json:"content_length"`
			SourceReads          []struct {
				SourceKey            string `json:"source_key"`
				ReturnedOffset       int    `json:"returned_offset"`
				ReturnedContentBytes int    `json:"returned_content_bytes"`
				ContentLength        int    `json:"content_length"`
				ResponseTruncated    bool   `json:"response_truncated"`
				NextOffset           int    `json:"next_offset"`
			} `json:"source_reads"`
		} `json:"io_metrics"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}
	if event.CorrelationID != payload.ToolSessionID || !strings.HasPrefix(payload.ToolSessionID, "ses_") || payload.MissionID != missionID || !payload.Success {
		return errors.New("report IL source trace binding is invalid")
	}
	switch payload.ToolName {
	case reportilcontract.SourceListTool:
	case reportilcontract.SourceQuoteRegisterTool:
		if !validReportILSourceKey(payload.IOMetrics.SourceKey) ||
			payload.IOMetrics.ReportILStage != "il_editorial_memory" ||
			len(payload.IOMetrics.CatalogSHA256) != 64 ||
			!strings.HasPrefix(payload.IOMetrics.SourceReceipt, "quote_") ||
			payload.IOMetrics.SourceOffset < 0 || payload.IOMetrics.SourceByteSize < 1 ||
			len(payload.IOMetrics.SourceSHA256) != 64 {
			return errors.New("report IL source quote trace metrics are incomplete")
		}
	case reportilcontract.SourceReadTool:
		if len(payload.IOMetrics.CatalogSHA256) != 64 {
			return errors.New("report IL source read trace metrics are incomplete")
		}
		if len(payload.IOMetrics.SourceReads) > 0 {
			if payload.IOMetrics.ReportILStage != "il_source_selection" &&
				payload.IOMetrics.ReportILStage != "il_editorial_memory" {
				return errors.New("report IL source batch trace stage is invalid")
			}
			for _, read := range payload.IOMetrics.SourceReads {
				if !validReportILSourceKey(read.SourceKey) ||
					read.ReturnedOffset < 0 ||
					read.ReturnedContentBytes < 1 ||
					read.ContentLength < read.ReturnedOffset+read.ReturnedContentBytes ||
					(read.ResponseTruncated && read.NextOffset != read.ReturnedOffset+read.ReturnedContentBytes) ||
					(!read.ResponseTruncated && read.NextOffset != 0) {
					return errors.New("report IL source batch trace metrics are incomplete")
				}
			}
		} else if !validReportILSourceKey(payload.IOMetrics.SourceKey) ||
			payload.IOMetrics.ReturnedOffset < 0 ||
			payload.IOMetrics.ReturnedContentBytes < 1 ||
			payload.IOMetrics.ContentLength < payload.IOMetrics.ReturnedOffset+payload.IOMetrics.ReturnedContentBytes {
			return errors.New("report IL source read trace metrics are incomplete")
		}
	case reportilcontract.EditorialMemoryStartTool,
		reportilcontract.EditorialMemoryAppendTool,
		reportilcontract.EditorialMemoryReadTool,
		reportilcontract.EditorialMemoryFinalizeTool:
		if payload.IOMetrics.WriteKind != "report_il_editorial_memory_workspace" ||
			payload.IOMetrics.Revision < 1 ||
			(payload.IOMetrics.ReportILStage != "il_editorial_memory" &&
				payload.IOMetrics.ReportILStage != "il_narrative" &&
				payload.IOMetrics.ReportILStage != "il_long_form_plan" &&
				payload.IOMetrics.ReportILStage != "il_long_form_section" &&
				payload.IOMetrics.ReportILStage != "il_long_form_part" &&
				payload.IOMetrics.ReportILStage != "il_long_form_final" &&
				payload.IOMetrics.ReportILStage != "il_continuity" &&
				payload.IOMetrics.ReportILStage != "il_reader") {
			return errors.New("report IL editorial-memory trace metrics are incomplete")
		}
		if payload.ToolName != reportilcontract.EditorialMemoryReadTool &&
			payload.IOMetrics.ReportILStage != "il_editorial_memory" {
			return errors.New("report IL editorial-memory mutation trace stage is invalid")
		}
		if payload.IOMetrics.ReportILStage != "il_editorial_memory" &&
			payload.ToolName != reportilcontract.EditorialMemoryReadTool {
			return errors.New("downstream report IL stage mutated editorial memory")
		}
		if payload.ToolName != reportilcontract.EditorialMemoryReadTool && !strings.HasPrefix(payload.IOMetrics.WorkspaceID, "ilm_") {
			return errors.New("report IL editorial-memory workspace trace is incomplete")
		}
		if payload.ToolName == reportilcontract.EditorialMemoryAppendTool && payload.IOMetrics.Accounts < 1 {
			return errors.New("report IL editorial-memory append trace is incomplete")
		}
		if payload.ToolName == reportilcontract.EditorialMemoryFinalizeTool &&
			(!payload.IOMetrics.Finalized || !strings.HasPrefix(payload.IOMetrics.ArtifactID, "art_") ||
				len(payload.IOMetrics.SHA256) != 64 || payload.IOMetrics.ByteSize < 1 || payload.IOMetrics.Accounts < 1) {
			return errors.New("report IL editorial-memory finalize trace is incomplete")
		}
	case reportilcontract.AuthorDocumentStartTool,
		reportilcontract.AuthorDocumentOpenTool,
		reportilcontract.AuthorDocumentAppendTool,
		reportilcontract.AuthorDocumentReadTool,
		reportilcontract.AuthorDocumentReplaceTool,
		reportilcontract.AuthorDocumentEditTextTool,
		reportilcontract.AuthorDocumentReviseBlockTool,
		reportilcontract.AuthorDocumentFinalizeTool,
		reportilcontract.LongFormDocumentStartTool,
		reportilcontract.LongFormDocumentAppendTool,
		reportilcontract.LongFormDocumentReadTool,
		reportilcontract.LongFormDocumentReplaceTool,
		reportilcontract.LongFormDocumentCorrectBlockTool,
		reportilcontract.LongFormDocumentFinalizeTool:
		if payload.IOMetrics.WriteKind != "report_il_document_workspace" ||
			payload.IOMetrics.Revision < 1 || !strings.HasPrefix(payload.IOMetrics.WorkspaceID, "ilw_") {
			return errors.New("report IL document trace metrics are incomplete")
		}
		if (payload.ToolName == reportilcontract.AuthorDocumentFinalizeTool || payload.ToolName == reportilcontract.LongFormDocumentFinalizeTool) &&
			(!payload.IOMetrics.Finalized || !strings.HasPrefix(payload.IOMetrics.ArtifactID, "art_") ||
				len(payload.IOMetrics.SHA256) != 64 || payload.IOMetrics.ByteSize < 1) {
			return errors.New("report IL document finalize trace is incomplete")
		}
	case reportilcontract.LongFormPlanSubmitTool:
		if payload.IOMetrics.WriteKind != "report_il_long_form_plan" ||
			payload.IOMetrics.ReportILStage != "il_long_form_plan" ||
			!strings.HasPrefix(payload.IOMetrics.ArtifactID, "art_") || len(payload.IOMetrics.SHA256) != 64 ||
			payload.IOMetrics.ByteSize < 1 || payload.IOMetrics.Parts < reportilcontract.MinLongFormParts ||
			payload.IOMetrics.Sections < reportilcontract.MinLongFormSections {
			return errors.New("report IL long-form plan trace metrics are incomplete")
		}
	case reportilcontract.LongFormPlanReadTool:
		if payload.IOMetrics.WriteKind != "report_il_long_form_plan_read" ||
			(payload.IOMetrics.ReportILStage != "il_long_form_section" &&
				payload.IOMetrics.ReportILStage != "il_long_form_part" &&
				payload.IOMetrics.ReportILStage != "il_long_form_final") ||
			payload.IOMetrics.ReturnedOffset < 0 || payload.IOMetrics.ReturnedContentBytes < 1 ||
			payload.IOMetrics.ContentLength < payload.IOMetrics.ReturnedOffset+payload.IOMetrics.ReturnedContentBytes {
			return errors.New("report IL long-form plan read trace metrics are incomplete")
		}
	default:
		return fmt.Errorf("report IL acceptance traced unexpected tool %q", payload.ToolName)
	}
	encoded, err := json.Marshal(map[string]any{"arguments": payload.Arguments, "result": payload.Result})
	if err != nil {
		return err
	}
	text := strings.ToLower(string(encoded))
	for _, forbidden := range []string{"experimental il product acceptance facts", "the independent report pipeline is opt-in", "editorial memory before prose", "locator", "filename", "url", "relative_path", "absolute_path", "authorization:", "bearer ", ".ts.net", "localhost", "/users/"} {
		if strings.Contains(text, forbidden) {
			return fmt.Errorf("report IL source trace leaked forbidden material %q", forbidden)
		}
	}
	return nil
}

func TestValidateReportILSourceTracePrivacyAcceptsEditorialMemoryContinuation(t *testing.T) {
	payload, err := json.Marshal(map[string]any{
		"tool_name": reportilcontract.SourceReadTool, "tool_session_id": "ses_editorial_memory",
		"mission_id": "mis_editorial_memory", "success": true, "arguments": map[string]any{},
		"result": map[string]any{},
		"io_metrics": map[string]any{
			"catalog_sha256": strings.Repeat("a", 64), "report_il_stage": "il_editorial_memory",
			"returned_content_bytes": 9,
			"source_reads": []any{map[string]any{
				"source_key": "source_001", "returned_offset": 8,
				"returned_content_bytes": 9, "content_length": 20,
				"response_truncated": true, "next_offset": 17,
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	event := ledger.Event{CorrelationID: "ses_editorial_memory", Payload: payload}
	if err := validateReportILSourceTracePrivacy(event, "mis_editorial_memory"); err != nil {
		t.Fatal(err)
	}

	var invalid map[string]any
	if err := json.Unmarshal(payload, &invalid); err != nil {
		t.Fatal(err)
	}
	metrics := invalid["io_metrics"].(map[string]any)
	reads := metrics["source_reads"].([]any)
	reads[0].(map[string]any)["next_offset"] = float64(16)
	invalidPayload, err := json.Marshal(invalid)
	if err != nil {
		t.Fatal(err)
	}
	event.Payload = invalidPayload
	if err := validateReportILSourceTracePrivacy(event, "mis_editorial_memory"); err == nil {
		t.Fatal("report IL source trace accepted a non-contiguous editorial-memory next_offset")
	}
}

func countReportILAcceptanceClassicTerminals(events []ledger.Event) int {
	count := 0
	for _, event := range events {
		switch event.EventType {
		case "report.drafted", "report.artifact.exported":
			count++
		case "report.artifact.created":
			if _, err := reportilcontract.DecodeTerminalPayload(event.Payload); err != nil {
				count++
			}
		}
	}
	return count
}

func TestReportILAcceptanceTransportWrapperRedactsLocalEnvironment(t *testing.T) {
	directory := t.TempDir()
	receiptPath := filepath.Join(directory, "transport.tsv")
	probePath := filepath.Join(directory, "probe")
	probe := "#!/bin/sh\nset -eu\n" +
		"if [ -n \"${PLASMA_IL_TRANSPORT_RECEIPT:-}\" ] || [ -n \"${PLASMA_IL_REAL_CODEX:-}\" ]; then exit 44; fi\n" +
		"schema=''\nprevious=''\nfor arg in \"$@\"; do if [ \"$previous\" = '--output-schema' ]; then schema=$arg; fi; previous=$arg; done\n" +
		"printf '%s\\n' '{\"ok\":true}' > \"${schema:?}\"\n"
	if err := os.WriteFile(probePath, []byte(probe), 0o700); err != nil {
		t.Fatal(err)
	}
	wrapperPath, err := writeReportILAcceptanceCodexWrapper(directory)
	if err != nil {
		t.Fatal(err)
	}
	schemaPath := filepath.Join(directory, "schema.json")
	if err := os.WriteFile(schemaPath, []byte(`{"type":"object"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(wrapperPath,
		"exec", "--ephemeral", "--ignore-user-config", "--model", "gpt-5.6-luna",
		"-c", `model_reasoning_effort="xhigh"`, "--json", "--output-schema", schemaPath,
		"-c", "features.tool_suggest=false", "--sandbox", "read-only",
		"--skip-git-repo-check", "--ignore-rules", "-",
	)
	command.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"PLASMA_IL_REAL_CODEX=" + probePath,
		"PLASMA_IL_TRANSPORT_RECEIPT=" + receiptPath,
	}
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("transport wrapper probe: %v: %s", err, output)
	}
	receipts, err := readReportILTransportReceipts(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(receipts) != 1 || !receipts[0].Ephemeral || !receipts[0].IgnoreUserConfig || !receipts[0].OutputSchema || receipts[0].MCPConfigs != 0 {
		t.Fatalf("transport wrapper receipt = %#v", receipts)
	}
}

func countReportILSourceReadSessions(events []ledger.Event) int {
	sessions := map[string]struct{}{}
	for _, event := range events {
		if event.EventType != "mcp.tool.called" {
			continue
		}
		var payload struct {
			ToolName      string `json:"tool_name"`
			ToolSessionID string `json:"tool_session_id"`
			Success       bool   `json:"success"`
			IOMetrics     struct {
				ReturnedContentBytes int `json:"returned_content_bytes"`
				SourceReads          []struct {
					ReturnedContentBytes int `json:"returned_content_bytes"`
				} `json:"source_reads"`
			} `json:"io_metrics"`
		}
		if json.Unmarshal(event.Payload, &payload) == nil && payload.ToolName == reportilcontract.SourceReadTool && strings.HasPrefix(payload.ToolSessionID, "ses_") && payload.Success && (payload.IOMetrics.ReturnedContentBytes > 0 || len(payload.IOMetrics.SourceReads) > 0) {
			sessions[payload.ToolSessionID] = struct{}{}
		}
	}
	return len(sessions)
}

func validReportILSourceKey(value string) bool {
	if !strings.HasPrefix(value, "source_") || len(value) != len("source_000") {
		return false
	}
	for _, digit := range value[len("source_"):] {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return value != "source_000"
}

func reportILAcceptanceMCPConfigs(receipts []reportILTransportReceipt) int {
	total := 0
	for _, receipt := range receipts {
		total += receipt.MCPConfigs
	}
	return total
}

func writeReportILProviderAcceptanceReceipt(directory string, receipt reportILProviderAcceptanceReceipt) error {
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	temporary, err := os.CreateTemp(directory, ".receipt-*.json")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(encoded); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, filepath.Join(directory, "receipt.json"))
}

func acceptanceSHA256(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
