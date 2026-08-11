package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"unicode"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow"
	"github.com/c86j224s/liquid2/plasma/internal/sourceevents"
)

func ensureFinalWriterV2FrozenReviewedManifest(ctx context.Context, cfg finalWriterV2AdapterConfig, binaryPath string, pair finalWriterV2ExperimentPair) (finalWriterV2FrozenManifest, string, error) {
	manifest, digest, err := loadFinalWriterV2FrozenManifest(cfg.ArchiveRoot, pair)
	if err == nil && finalWriterV2PrepProvenanceValid(ctx, cfg.ArchiveRoot, manifest) == nil {
		return manifest, digest, nil
	}
	return prepareFinalWriterV2FrozenReviewedManifest(ctx, cfg, binaryPath, pair)
}

func prepareFinalWriterV2FrozenReviewedManifest(ctx context.Context, cfg finalWriterV2AdapterConfig, binaryPath string, pair finalWriterV2ExperimentPair) (finalWriterV2FrozenManifest, string, error) {
	runDir := filepath.Join(cfg.ArchiveRoot, "prep-reviewed-parts", finalWriterV2ExperimentRunNamespace, pair.PairID)
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return finalWriterV2FrozenManifest{}, "", err
	}
	dbPath := filepath.Join(runDir, "plasma.db")
	for _, path := range []string{dbPath, dbPath + "-wal", dbPath + "-shm"} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return finalWriterV2FrozenManifest{}, "", err
		}
	}
	svc, closeStore, err := openFinalWriterV2ExperimentServicePath(ctx, dbPath)
	if err != nil {
		return finalWriterV2FrozenManifest{}, "", err
	}
	defer closeStore()

	executor := &finalWriterV2RecordingExecutor{archive: runDir}
	executor.delegate = CodexExecutor{
		Command: cfg.CodexCommand, WorkDir: runDir, Timeout: cfg.Timeout, Env: os.Environ(),
		MCPServer: CodexMCPServer{Name: "plasma", Command: binaryPath, Args: []string{"mcp", "-db", dbPath}, Required: true, StartupTimeoutSec: 30, ToolTimeoutSec: 240},
	}
	server := NewServer(svc, Options{}).(*Server)
	fragment := finalWriterV2IDFragment("prep_" + pair.PairID)
	missionID := "mis_exp55_" + fragment
	pendingID := "evt_exp55_" + fragment + "_pending"
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: pair.TopicTitle}); err != nil {
		return finalWriterV2FrozenManifest{}, "", err
	}
	if _, err := svc.AppendEvent(ctx, app.BuildMissionCreatedAppendRequest(app.MissionCreatedEventRequest{
		EventID: "evt_exp55_" + fragment + "_mission", MissionID: missionID, Title: pair.TopicTitle,
		Objective: "Prepare product-reviewed Korean Parts for the final-writer v2 fixed-input experiment.",
		Producer:  app.Producer{Type: "user", ID: "experiment"},
	})); err != nil {
		return finalWriterV2FrozenManifest{}, "", err
	}
	sourceSnapshotIDs, sourceArtifactIDs, sourceEventIDs, err := createFinalWriterV2PrepSources(ctx, svc, cfg.ArchiveRoot, pair, missionID, fragment)
	if err != nil {
		return finalWriterV2FrozenManifest{}, "", err
	}
	rigor, ok := reportRigorProfiles[pair.Rigor]
	if !ok {
		return finalWriterV2FrozenManifest{}, "", fmt.Errorf("unknown rigor %q", pair.Rigor)
	}
	guidanceProfile, guidanceSHA, err := reportprompt.SelectReportGenerationGuidanceForMode(reportModeLongForm, reportprompt.ProfilePartConnectiveEconomyVoice)
	if err != nil {
		return finalWriterV2FrozenManifest{}, "", err
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: pendingID, MissionID: missionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "experiment"},
		Payload: finalWriterV2MustJSON(map[string]any{
			"kind": "markdown_report_artifact_pending", "origin_pending_event_id": pendingID, "retry_strategy": "initial",
			"title": pair.TopicTitle, "direction_hint": finalWriterV2PrepDirectionHint(pair), "report_mode": reportModeLongForm,
			"execution_strategy": reportExecutionStrategySectionFanout, "rigor_level": pair.Rigor, "agent_executor": cfg.ExecutorName,
			"agent_model": cfg.AgentModel, "agent_reasoning_effort": cfg.ReasoningEffort, "generation_guidance_profile": guidanceProfile,
			"generation_guidance_sha256": guidanceSHA, "post_report_humanize": reporting.FinalEditHumanizeDisabled,
		}),
	}); err != nil {
		return finalWriterV2FrozenManifest{}, "", err
	}
	prefix, err := reportworkflow.NewRunner(reportworkflow.RunnerConfig{
		Service: server.service, Lifecycle: reporting.Runner(server.reportRunner()),
		Executor: executor, NewID: newID, LatestSessionID: server.latestAgentSessionID,
	}).RunLongFormPrefix(ctx, reportworkflow.DraftInput{
		MissionID: missionID, PendingEventID: pendingID, Title: pair.TopicTitle,
		DirectionHint: finalWriterV2PrepDirectionHint(pair), ExecutionStrategy: reportExecutionStrategySectionFanout,
		AgentExecutor: cfg.ExecutorName, AgentModel: cfg.AgentModel, AgentReasoningEffort: cfg.ReasoningEffort,
		AgentSelectionSource: "experiment", MCPMode: "auto", Rigor: reportWorkflowRigor(rigor),
		ReportMode: reportModeLongForm, ReportSessionPolicy: reportSessionPolicySameSession,
		ReportSessionPolicySelection: "experiment-product-reviewed-part-prep",
		PostReportHumanize:           reporting.FinalEditHumanizeDisabled,
		GenerationGuidanceProfile:    guidanceProfile, GenerationGuidanceSHA256: guidanceSHA,
	})
	if err != nil {
		return finalWriterV2FrozenManifest{}, "", err
	}
	parts := make([]sectionalReportPartDraft, len(prefix.Parts))
	for index, part := range prefix.Parts {
		parts[index] = sectionalReportPartDraft{
			Title: part.Title, Markdown: part.Markdown, ArtifactID: part.ArtifactID, WordCount: part.WordCount,
		}
	}
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		return finalWriterV2FrozenManifest{}, "", err
	}
	ledgerPath := filepath.Join(runDir, "ledger", "events.json")
	if err := writeFinalWriterV2JSONFilePath(ledgerPath, events); err != nil {
		return finalWriterV2FrozenManifest{}, "", err
	}
	return writeFinalWriterV2FrozenManifestFromParts(cfg.ArchiveRoot, pair, finalWriterV2PrepProvenance{
		ProductPath: "section_fanout_plan_requirement_sections_part_assembly_part_author",
		MissionID:   missionID, PendingEventID: pendingID, PlanEventID: prefix.PlanEvent.EventID, DBPath: dbPath,
		LedgerEventsPath: ledgerPath, LedgerEventsSHA256: finalWriterV2SHA256FileNoErr(ledgerPath),
		SourceSnapshotIDs: sourceSnapshotIDs, SourceArtifactIDs: sourceArtifactIDs, SourceEventIDs: sourceEventIDs,
		DiscardedFinalReport: true,
	}, parts, prefix.SectionArtifactIDs)
}

func createFinalWriterV2PrepSources(ctx context.Context, svc *app.Service, archive string, pair finalWriterV2ExperimentPair, missionID string, fragment string) ([]string, []string, []string, error) {
	files, err := filepath.Glob(filepath.Join(archive, "source-corpora", pair.TopicID, "*.md"))
	if err != nil {
		return nil, nil, nil, err
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, nil, nil, fmt.Errorf("archive-local source corpus is missing for %s", pair.TopicID)
	}
	snapshots, artifacts, events := []string{}, []string{}, []string{}
	for index, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, nil, err
		}
		if strings.TrimSpace(string(content)) == "" {
			return nil, nil, nil, fmt.Errorf("source corpus file is empty: %s", path)
		}
		artifactID := fmt.Sprintf("art_exp55_%s_source_%02d", fragment, index+1)
		snapshotID := fmt.Sprintf("src_exp55_%s_source_%02d", fragment, index+1)
		eventID := fmt.Sprintf("evt_exp55_%s_source_%02d", fragment, index+1)
		connector := app.ConnectorRef{
			ConnectorID: "experiment-archive", ConnectorType: app.SourceConnectorTypeFileUpload,
			ExternalSourceID: pair.TopicID + "/" + filepath.Base(path), ExternalURI: "archive://" + pair.TopicID + "/" + filepath.Base(path),
			ConnectorVersion: finalWriterV2ExperimentRunNamespace,
		}
		result, err := svc.CreateSourceSnapshotWithEvent(ctx, app.CreateSourceSnapshotWithEventRequest{
			Artifact: app.CreateRawArtifactRequest{
				ArtifactID: artifactID, MissionID: missionID, MediaType: "text/markdown; charset=utf-8",
				Filename: filepath.Base(path), Producer: app.Producer{Type: "user", ID: "experiment"}, Content: content,
			},
			Snapshot: app.CreateSourceSnapshotRequest{
				SnapshotID: snapshotID, MissionID: missionID, Connector: connector, Title: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
				Locators: json.RawMessage(`[{"locator_type":"full_text"}]`),
				Access:   app.SourceAccess{Visibility: "private", License: "experiment-corpus", RetrievalPolicy: app.SourceRetrievalPolicySnapshotOnly},
			},
			Event: app.AppendEventRequest{
				EventID: eventID, MissionID: missionID, EventType: sourceevents.SourceSnapshottedEventType,
				Producer: app.Producer{Type: "user", ID: "experiment"},
			},
		})
		if err != nil {
			return nil, nil, nil, err
		}
		snapshots = append(snapshots, result.Snapshot.SnapshotID)
		artifacts = append(artifacts, result.Artifact.ArtifactID)
		events = append(events, result.Event.EventID)
	}
	return snapshots, artifacts, events, nil
}

func writeFinalWriterV2FrozenManifestFromParts(archive string, pair finalWriterV2ExperimentPair, provenance finalWriterV2PrepProvenance, parts []sectionalReportPartDraft, sectionArtifactIDs []string) (finalWriterV2FrozenManifest, string, error) {
	if len(parts) == 0 {
		return finalWriterV2FrozenManifest{}, "", fmt.Errorf("product prep produced no reviewed Parts")
	}
	manifest := finalWriterV2FrozenManifest{
		ExperimentID: finalWriterV2ExperimentID, PairID: pair.PairID, TopicID: pair.TopicID, TopicTitle: pair.TopicTitle, Rigor: pair.Rigor,
		Source: "product_reviewed_parts_from_upstream_section_fanout", Prep: provenance,
		Receipts: map[string]string{"section_artifact_count": fmt.Sprint(len(sectionArtifactIDs))},
	}
	for index, part := range parts {
		markdown := strings.TrimSpace(part.Markdown) + "\n"
		if !finalWriterV2ContainsHangul(markdown) {
			return finalWriterV2FrozenManifest{}, "", fmt.Errorf("product reviewed Part %d is not Korean", index+1)
		}
		manifest.Parts = append(manifest.Parts, finalWriterV2FrozenPart{
			PartIndex: index + 1, Title: strings.TrimSpace(part.Title), Markdown: markdown,
			SHA256: sha256Hex([]byte(markdown)), ArtifactID: part.ArtifactID, WordCount: len(strings.Fields(markdown)),
		})
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return finalWriterV2FrozenManifest{}, "", err
	}
	encoded = append(encoded, '\n')
	digest := sha256Hex(encoded)
	path := finalWriterV2FrozenManifestPath(archive, pair)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return finalWriterV2FrozenManifest{}, "", err
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		return finalWriterV2FrozenManifest{}, "", err
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "parts.manifest.sha256"), []byte(digest+"  parts.manifest.json\n"), 0o600); err != nil {
		return finalWriterV2FrozenManifest{}, "", err
	}
	return manifest, digest, nil
}

func writeFinalWriterV2TestFrozenManifest(t *testing.T, archive string, pair finalWriterV2ExperimentPair) (finalWriterV2FrozenManifest, string) {
	t.Helper()
	parts := []sectionalReportPartDraft{
		{Title: "검증 Part 1", Markdown: "# 검증 Part 1\n\n제품 경로에서 이미 검토된 Part 바이트를 대신하는 테스트 고정 입력입니다. [T-1]\n"},
		{Title: "검증 Part 2", Markdown: "# 검증 Part 2\n\n두 번째 Part도 같은 manifest를 통해 A와 B에 주입되어야 합니다. [T-2]\n"},
	}
	manifest, digest, err := writeFinalWriterV2FrozenManifestFromParts(archive, pair, finalWriterV2PrepProvenance{
		ProductPath: "test-only-explicit-frozen-parts", MissionID: "mis_exp55_test", PendingEventID: "evt_exp55_test_pending",
		PlanEventID: "evt_exp55_test_plan", DiscardedFinalReport: true,
	}, parts, []string{"art_section_1", "art_section_2"})
	if err != nil {
		t.Fatal(err)
	}
	return manifest, digest
}

func loadFinalWriterV2FrozenManifest(archive string, pair finalWriterV2ExperimentPair) (finalWriterV2FrozenManifest, string, error) {
	path := finalWriterV2FrozenManifestPath(archive, pair)
	content, err := os.ReadFile(path)
	if err != nil {
		return finalWriterV2FrozenManifest{}, "", err
	}
	digest := sha256Hex(content)
	receipt, err := os.ReadFile(filepath.Join(filepath.Dir(path), "parts.manifest.sha256"))
	if err != nil {
		return finalWriterV2FrozenManifest{}, "", err
	}
	if got := strings.Fields(string(receipt)); len(got) == 0 || got[0] != digest {
		return finalWriterV2FrozenManifest{}, "", fmt.Errorf("frozen reviewed Part manifest receipt mismatch for %s", pair.PairID)
	}
	var manifest finalWriterV2FrozenManifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return finalWriterV2FrozenManifest{}, "", err
	}
	if manifest.ExperimentID != finalWriterV2ExperimentID || manifest.PairID != pair.PairID || manifest.TopicID != pair.TopicID || manifest.Rigor != pair.Rigor || len(manifest.Parts) == 0 {
		return finalWriterV2FrozenManifest{}, "", fmt.Errorf("frozen reviewed Part manifest identity mismatch for %s", pair.PairID)
	}
	for index, part := range manifest.Parts {
		if part.PartIndex != index+1 || strings.TrimSpace(part.Title) == "" || strings.TrimSpace(part.Markdown) == "" || part.SHA256 != sha256Hex([]byte(part.Markdown)) {
			return finalWriterV2FrozenManifest{}, "", fmt.Errorf("frozen reviewed Part %d is invalid for %s", index+1, pair.PairID)
		}
	}
	return manifest, digest, nil
}

func seedFinalWriterV2ExperimentTerminalPipeline(ctx context.Context, svc *app.Service, pair finalWriterV2ExperimentPair, arm string, manifest finalWriterV2FrozenManifest, cfg finalWriterV2AdapterConfig, planSessionID string) (finalizationPrefixFixture, error) {
	if manifest.PairID != pair.PairID {
		return finalizationPrefixFixture{}, fmt.Errorf("manifest pair mismatch")
	}
	rigor, ok := reportRigorProfiles[pair.Rigor]
	if !ok {
		return finalizationPrefixFixture{}, fmt.Errorf("unknown rigor %q", pair.Rigor)
	}
	pipeline := finalWriterV2PipelineForArm(arm)
	if pipeline == "" {
		return finalizationPrefixFixture{}, fmt.Errorf("unknown arm %q", arm)
	}
	fragment := finalWriterV2IDFragment(pair.PairID + "_" + arm + "_" + finalWriterV2ExperimentRunNamespace)
	missionID := "mis_exp55_" + fragment
	pendingID := "evt_exp55_" + fragment + "_pending"
	planID := "evt_exp55_" + fragment + "_plan"
	finalArtifactID := "art_exp55_" + fragment + "_final"
	producer := app.Producer{Type: "agent_session", ID: planSessionID}
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: pair.TopicTitle}); err != nil {
		return finalizationPrefixFixture{}, err
	}
	if _, err := svc.AppendEvent(ctx, app.BuildMissionCreatedAppendRequest(app.MissionCreatedEventRequest{
		EventID: "evt_exp55_" + fragment + "_mission", MissionID: missionID, Title: pair.TopicTitle,
		Objective: "Run fixed reviewed Part final-edit experiment", Producer: app.Producer{Type: "user", ID: "experiment"},
	})); err != nil {
		return finalizationPrefixFixture{}, err
	}
	guidanceProfile, guidanceSHA, err := reportprompt.SelectReportGenerationGuidanceForMode(reportModeLongForm, reportprompt.ProfilePartConnectiveEconomyVoice)
	if err != nil {
		return finalizationPrefixFixture{}, err
	}
	plan := finalWriterV2PlanForPair(pair, manifest)
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: pendingID, MissionID: missionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "experiment"},
		Payload: finalWriterV2MustJSON(map[string]any{
			"kind": "markdown_report_artifact_pending", "origin_pending_event_id": pendingID, "retry_strategy": "initial",
			"title": pair.TopicTitle, "report_mode": reportModeLongForm, "rigor_level": pair.Rigor, "agent_executor": cfg.ExecutorName,
			"agent_model": cfg.AgentModel, "agent_reasoning_effort": cfg.ReasoningEffort, "generation_guidance_profile": guidanceProfile,
			"generation_guidance_sha256": guidanceSHA, "post_report_humanize": cfg.PostHumanize,
		}),
	}); err != nil {
		return finalizationPrefixFixture{}, err
	}
	planEvent, err := svc.AppendEvent(ctx, reporting.BuildMarkdownReportPlanCreatedAppendRequest(reporting.MarkdownReportPlanCreatedEventRequest{
		MarkdownReportEventBase: reporting.MarkdownReportEventBase{
			EventID: planID, MissionID: missionID, PendingEventID: pendingID, Title: pair.TopicTitle,
			AgentExecutor: cfg.ExecutorName, AgentModel: cfg.AgentModel, AgentReasoningEffort: cfg.ReasoningEffort, AgentSelectionSource: "experiment",
			AgentSessionID: planSessionID, ReturnedAgentSessionID: planSessionID, ToolSessionID: "ses_exp55_" + fragment + "_plan",
			MCPMode: "auto", RigorLevel: rigor.level, RigorLabel: rigor.label, ReportMode: reportModeLongForm, ReportModeLabel: reportModeLabel(reportModeLongForm),
			ReportSessionPolicy: reportSessionPolicySameSession, ReportSessionPolicySelection: "experiment-fixed-reviewed-part",
			PostReportHumanize: cfg.PostHumanize, HumanizeEnabled: cfg.PostHumanize == reporting.FinalEditHumanizeEnabled,
			GenerationGuidanceProfile: guidanceProfile, GenerationGuidanceSHA256: guidanceSHA,
			SessionChainKind: "fixed_reviewed_part_terminal_experiment", ReportPlanSessionID: planSessionID, ForkSourceAgentSessionID: planSessionID,
			Text: "고정 reviewed Part terminal pipeline 실험 계획을 기록했습니다.", Producer: producer,
		},
		ArtifactID:          finalArtifactID,
		Plan:                plan,
		FinalEditPipeline:   pipeline,
		PartEditEnabled:     false,
		PartPlanningEnabled: false,
	}))
	if err != nil {
		return finalizationPrefixFixture{}, err
	}
	partArtifactIDs, sectionArtifactIDs := []string{}, []string{}
	parts := make([]sectionalReportPartDraft, 0, len(manifest.Parts))
	sectionWordTotal := 0
	for _, part := range manifest.Parts {
		content := []byte(part.Markdown)
		if part.SHA256 != sha256Hex(content) {
			return finalizationPrefixFixture{}, fmt.Errorf("frozen reviewed Part digest mismatch for %s part %d", pair.PairID, part.PartIndex)
		}
		partID := fmt.Sprintf("art_exp55_%s_part_%02d", fragment, part.PartIndex)
		sectionID := partID
		partEventID := fmt.Sprintf("evt_exp55_%s_part_%02d", fragment, part.PartIndex)
		artifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
			ArtifactID: partID, MissionID: missionID, MediaType: "text/markdown; charset=utf-8",
			Filename: fmt.Sprintf("part-%02d.md", part.PartIndex), Producer: producer, Content: content,
		})
		if err != nil {
			return finalizationPrefixFixture{}, err
		}
		stageBase := reporting.MarkdownReportStageEventBase{
			MissionID: missionID, PendingEventID: pendingID, PlanEventID: planID, Title: pair.TopicTitle, Artifact: artifact,
			AgentExecutor: cfg.ExecutorName, AgentModel: cfg.AgentModel, AgentReasoningEffort: cfg.ReasoningEffort, AgentSelectionSource: "experiment",
			AgentSessionID: planSessionID, ReturnedAgentSessionID: planSessionID, ToolSessionID: "ses_exp55_" + fragment + "_part_" + fmt.Sprintf("%02d", part.PartIndex),
			ReportMode: reportModeLongForm, ReportModeLabel: reportModeLabel(reportModeLongForm),
			ReportSessionPolicy: reportSessionPolicySameSession, ReportSessionPolicySelection: "experiment-fixed-reviewed-part",
			PostReportHumanize: cfg.PostHumanize, HumanizeEnabled: cfg.PostHumanize == reporting.FinalEditHumanizeEnabled,
			GenerationGuidanceProfile: guidanceProfile, GenerationGuidanceSHA256: guidanceSHA,
			SessionChainKind: "fixed_reviewed_part_terminal_experiment", ReportPlanSessionID: planSessionID, ForkSourceAgentSessionID: planSessionID,
			Text: "고정 reviewed Part artifact를 실험 입력으로 기록했습니다.", Producer: producer,
		}
		stageBase.EventID = partEventID
		if _, err := svc.AppendEvent(ctx, reporting.BuildMarkdownReportPartCreatedAppendRequest(reporting.MarkdownReportPartCreatedEventRequest{
			MarkdownReportStageEventBase: stageBase, PartIndex: part.PartIndex, SectionCount: 1, WordCount: len(strings.Fields(part.Markdown)),
		})); err != nil {
			return finalizationPrefixFixture{}, err
		}
		stageBase.EventID = fmt.Sprintf("evt_exp55_%s_section_%02d", fragment, part.PartIndex)
		if _, err := svc.AppendEvent(ctx, reporting.BuildMarkdownReportSectionCreatedAppendRequest(reporting.MarkdownReportSectionCreatedEventRequest{
			MarkdownReportStageEventBase: stageBase, PartIndex: part.PartIndex, SectionIndex: 1, WordCount: len(strings.Fields(part.Markdown)),
		})); err != nil {
			return finalizationPrefixFixture{}, err
		}
		partArtifactIDs = append(partArtifactIDs, partID)
		sectionArtifactIDs = append(sectionArtifactIDs, sectionID)
		sectionWordTotal += len(strings.Fields(part.Markdown))
		parts = append(parts, sectionalReportPartDraft{Title: part.Title, Markdown: part.Markdown, ArtifactID: partID, WordCount: len(strings.Fields(part.Markdown))})
	}
	return finalizationPrefixFixture{
		missionID: missionID, title: pair.TopicTitle, executorName: cfg.ExecutorName, agentModel: cfg.AgentModel, agentReasoningEffort: cfg.ReasoningEffort,
		agentSelectionSource: "experiment", mcpMode: "auto", rigor: rigor, reportSessionPolicy: reportSessionPolicySameSession,
		reportSessionPolicySelection: "experiment-fixed-reviewed-part", postReportHumanize: cfg.PostHumanize,
		generationGuidanceProfile: guidanceProfile, generationGuidanceSHA256: guidanceSHA, pendingEventID: pendingID, artifactID: finalArtifactID,
		planEvent: planEvent, plan: plan, parts: parts, partArtifactIDs: partArtifactIDs, sectionArtifactIDs: sectionArtifactIDs, sectionWordTotal: sectionWordTotal,
		sessionChainKind: "fixed_reviewed_part_terminal_experiment", preReportResearchSessionID: planSessionID, reportPlanSessionID: planSessionID,
		forkSourceAgentSessionID: planSessionID, requirementMap: finalWriterV2RequirementMapForPair(pair, plan, pendingID), finalTail: finalWriterV2TailForArm(arm), started: cfg.Started,
	}, nil
}

func finalWriterV2TailForArm(arm string) reportworkflow.FinalTail {
	if arm == "A" {
		return reportworkflow.FinalTailV1
	}
	return reportworkflow.FinalTailV2
}

func finalWriterV2PlanForPair(pair finalWriterV2ExperimentPair, manifest finalWriterV2FrozenManifest) agentSectionalReportPlan {
	parts := make([]reporting.ReportPlanPart, 0, len(manifest.Parts))
	for _, part := range manifest.Parts {
		parts = append(parts, reporting.ReportPlanPart{
			Title:   part.Title,
			Purpose: "Turn the fixed reviewed Part into a readable Korean section of the final report.",
			Sections: []reporting.ReportPlanSection{{
				Title:   part.Title,
				Purpose: "Preserve the reviewed facts, citation tags, caveats, and explicit requirement for this Part.",
			}},
		})
	}
	plan, err := reporting.NormalizeSectionalReportPlan(reporting.SectionalReportPlan{
		Summary: "Fixed reviewed-Part terminal-pipeline experiment for " + pair.TopicTitle + ".",
		Parts:   parts,
		WritingContract: &reporting.ReportWritingContract{
			CentralQuestion: "How should a reader understand " + pair.TopicTitle + " from the fixed reviewed Parts?",
			ReaderTakeaway:  "Use only the reviewed Parts to produce a coherent Korean report that preserves every protected fact, citation tag, caveat, and requirement.",
			ReadingPath:     []string{"State the subject and evidence boundary.", "Connect the reviewed Parts in order.", "Preserve caveats near the claims they qualify.", "End with a practical synthesis."},
			MustKeep:        []string{"all bracketed citation tags", "each Part requirement", "all explicit caveats", "the original Part order"},
			VisualRole:      "No visual output is required in this terminal-pipeline experiment.",
			ToneAndShape:    "Clear Korean long-form report prose; no process narration or source inventory tone.",
		},
	})
	if err != nil {
		panic(err)
	}
	return plan
}

func finalWriterV2RequirementMapForPair(pair finalWriterV2ExperimentPair, plan agentSectionalReportPlan, pendingID string) reporting.ReportRequirementMap {
	requirements := []reporting.ReportRequirement{}
	for index := range plan.Parts {
		requirements = append(requirements, reporting.ReportRequirement{
			RequirementID:  fmt.Sprintf("req_exp55_%s_%02d", finalWriterV2IDFragment(pair.TopicID), index+1),
			Instruction:    "Preserve the reviewed Part's explicit facts, caveats, citation tags, uncertainty boundaries, and concrete reader requirement.",
			SourceEventIDs: []string{pendingID},
			Owner:          &reporting.ReportRequirementOwner{PartIndex: index + 1, SectionIndex: 1},
		})
	}
	normalized, err := reporting.NormalizeReportRequirementMap(reporting.ReportRequirementMap{ReviewedEventIDs: []string{pendingID}, Requirements: requirements}, plan)
	if err != nil {
		panic(err)
	}
	return normalized
}

func finalWriterV2PrepDirectionHint(pair finalWriterV2ExperimentPair) string {
	switch pair.TopicID {
	case "wang-anshi-northern-song":
		return "한국어 장문 보고서로 작성한다. source corpus의 bracket citation tags를 보존하고, 재정ㆍ변방 방위ㆍ관료 충원, 신종의 후원, 사마광 반대, 청묘/노역 부담, 당파적 기억, 개혁 야망과 집행 위험의 균형을 모두 다룬다."
	case "go-raft-implementation-roadmap":
		return "한국어 장문 보고서로 작성한다. source corpus의 bracket citation tags를 보존하고, term/vote/log 지속성, randomized election timeout, AppendEntries log matching, snapshot, membership, simulator, storage, transport, observability, testable safety milestone를 모두 다룬다."
	default:
		return "한국어 장문 보고서로 작성한다. source corpus의 bracket citation tags와 명시 요구사항을 보존한다."
	}
}

func finalWriterV2FrozenManifestPath(archive string, pair finalWriterV2ExperimentPair) string {
	return filepath.Join(archive, "fixed-inputs", finalWriterV2ExperimentRunNamespace, pair.TopicID, pair.Rigor, "parts.manifest.json")
}

func finalWriterV2PipelineForArm(arm string) string {
	switch arm {
	case "A":
		return reporting.FinalEditPipelineReaderStyleGateV1
	case "B":
		return reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2
	default:
		return ""
	}
}

func finalWriterV2IDFragment(value string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(value) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}
	return strings.Trim(builder.String(), "_")
}

func finalWriterV2ContainsHangul(value string) bool {
	return strings.IndexFunc(value, func(r rune) bool {
		return unicode.In(r, unicode.Hangul)
	}) >= 0
}

func finalWriterV2PrepProvenanceValid(ctx context.Context, archive string, manifest finalWriterV2FrozenManifest) error {
	if manifest.Source != "product_reviewed_parts_from_upstream_section_fanout" {
		return fmt.Errorf("frozen manifest was not produced by the W6-B product-reviewed-Part prep")
	}
	if manifest.Prep.ProductPath != "section_fanout_plan_requirement_sections_part_assembly_part_author" ||
		manifest.Prep.MissionID == "" || manifest.Prep.PendingEventID == "" || manifest.Prep.PlanEventID == "" ||
		manifest.Prep.DBPath == "" || manifest.Prep.LedgerEventsPath == "" || manifest.Prep.LedgerEventsSHA256 == "" ||
		len(manifest.Prep.SourceSnapshotIDs) == 0 || len(manifest.Prep.SourceArtifactIDs) == 0 || len(manifest.Prep.SourceEventIDs) == 0 ||
		!manifest.Prep.DiscardedFinalReport {
		return fmt.Errorf("frozen manifest prep provenance is incomplete")
	}
	if len(manifest.Prep.SourceSnapshotIDs) != len(manifest.Prep.SourceArtifactIDs) || len(manifest.Prep.SourceSnapshotIDs) != len(manifest.Prep.SourceEventIDs) {
		return fmt.Errorf("frozen manifest source provenance cardinality mismatch")
	}
	for _, path := range []string{manifest.Prep.DBPath, manifest.Prep.LedgerEventsPath} {
		if err := finalWriterV2PathInsideArchive(archive, path); err != nil {
			return err
		}
	}
	if finalWriterV2SHA256FileNoErr(manifest.Prep.LedgerEventsPath) != manifest.Prep.LedgerEventsSHA256 {
		return fmt.Errorf("prep ledger event receipt mismatch")
	}
	svc, closeStore, err := openFinalWriterV2ExperimentServicePath(ctx, manifest.Prep.DBPath)
	if err != nil {
		return err
	}
	defer closeStore()
	dbEvents, err := svc.ListEvents(ctx, manifest.Prep.MissionID)
	if err != nil {
		return err
	}
	exported, err := finalWriterV2ReadExportedLedgerEvents(manifest.Prep.LedgerEventsPath)
	if err != nil {
		return err
	}
	if err := finalWriterV2ValidatePrepLedgerReplay(dbEvents, exported, manifest); err != nil {
		return err
	}
	if err := finalWriterV2ValidatePrepSources(ctx, svc, archive, manifest); err != nil {
		return err
	}
	if err := finalWriterV2ValidatePrepParts(ctx, svc, dbEvents, manifest); err != nil {
		return err
	}
	return nil
}

func finalWriterV2ReadExportedLedgerEvents(path string) ([]app.LedgerEvent, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var events []app.LedgerEvent
	if err := json.Unmarshal(content, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func finalWriterV2ValidatePrepLedgerReplay(dbEvents []app.LedgerEvent, exported []app.LedgerEvent, manifest finalWriterV2FrozenManifest) error {
	if len(dbEvents) == 0 || len(dbEvents) != len(exported) {
		return fmt.Errorf("prep DB and exported ledger event counts differ")
	}
	counts := map[string]int{}
	for index := range dbEvents {
		dbEvent, exportedEvent := dbEvents[index], exported[index]
		if dbEvent.EventID != exportedEvent.EventID || dbEvent.MissionID != exportedEvent.MissionID ||
			dbEvent.Sequence != exportedEvent.Sequence || dbEvent.EventType != exportedEvent.EventType ||
			finalWriterV2CompactJSON(dbEvent.Payload) != finalWriterV2CompactJSON(exportedEvent.Payload) {
			return fmt.Errorf("prep exported ledger diverges at sequence %d", index+1)
		}
		counts[dbEvent.EventType]++
	}
	required := map[string]int{
		sourceevents.SourceSnapshottedEventType: len(manifest.Prep.SourceEventIDs),
		"report.draft.pending":                  1,
		"report.plan.created":                   1,
		"report.requirements.started":           1,
		"report.requirements.mapped":            1,
		"report.section.created":                1,
		"report.part_plan.created":              len(manifest.Parts),
		"report.part_assembly.submitted":        len(manifest.Parts),
		"report.part.created":                   len(manifest.Parts),
		"report.part_edit.started":              len(manifest.Parts),
		"report.part.edited":                    len(manifest.Parts),
	}
	for eventType, minimum := range required {
		if counts[eventType] < minimum {
			return fmt.Errorf("prep ledger missing %s events", eventType)
		}
	}
	if counts["report.artifact.created"] != 0 {
		return fmt.Errorf("prep final report was not discarded")
	}
	if !finalWriterV2HasEvent(dbEvents, manifest.Prep.PendingEventID, "report.draft.pending") ||
		!finalWriterV2HasEvent(dbEvents, manifest.Prep.PlanEventID, "report.plan.created") {
		return fmt.Errorf("prep pending or plan event missing")
	}
	for _, eventID := range manifest.Prep.SourceEventIDs {
		if !finalWriterV2HasEvent(dbEvents, eventID, sourceevents.SourceSnapshottedEventType) {
			return fmt.Errorf("prep source event missing")
		}
	}
	return nil
}

func finalWriterV2ValidatePrepSources(ctx context.Context, svc *app.Service, archive string, manifest finalWriterV2FrozenManifest) error {
	for index, snapshotID := range manifest.Prep.SourceSnapshotIDs {
		artifactID := manifest.Prep.SourceArtifactIDs[index]
		eventID := manifest.Prep.SourceEventIDs[index]
		snapshot, err := svc.GetSourceSnapshot(ctx, snapshotID)
		if err != nil {
			return err
		}
		if snapshot.MissionID != manifest.Prep.MissionID || !slices.Contains(snapshot.ArtifactIDs, artifactID) ||
			snapshot.Connector.ConnectorID != "experiment-archive" || snapshot.Connector.ConnectorVersion != finalWriterV2ExperimentRunNamespace {
			return fmt.Errorf("prep source snapshot mismatch for %s", snapshotID)
		}
		artifact, err := svc.GetRawArtifact(ctx, artifactID)
		if err != nil {
			return err
		}
		sourcePath := filepath.Join(archive, "source-corpora", filepath.FromSlash(snapshot.Connector.ExternalSourceID))
		if err := finalWriterV2PathInsideArchive(archive, sourcePath); err != nil {
			return err
		}
		sourceBytes, err := os.ReadFile(sourcePath)
		if err != nil {
			return err
		}
		if artifact.MissionID != manifest.Prep.MissionID || artifact.SHA256 != sha256Hex(sourceBytes) ||
			snapshot.ContentHash.Value != artifact.SHA256 || !bytes.Equal(artifact.Content, sourceBytes) {
			return fmt.Errorf("prep source artifact byte mismatch for %s", artifactID)
		}
		if eventID == "" {
			return fmt.Errorf("prep source event missing for %s", snapshotID)
		}
	}
	return nil
}

func finalWriterV2ValidatePrepParts(ctx context.Context, svc *app.Service, events []app.LedgerEvent, manifest finalWriterV2FrozenManifest) error {
	for _, part := range manifest.Parts {
		artifact, err := svc.GetRawArtifact(ctx, part.ArtifactID)
		if err != nil {
			return err
		}
		if artifact.MissionID != manifest.Prep.MissionID || artifact.SHA256 != part.SHA256 || !bytes.Equal(artifact.Content, []byte(part.Markdown)) {
			return fmt.Errorf("prep reviewed Part artifact byte mismatch for part %d", part.PartIndex)
		}
		foundEdited := false
		for _, event := range events {
			if event.EventType == "report.part.edited" && finalWriterV2EventString(event, "artifact_id") == part.ArtifactID &&
				finalWriterV2EventString(event, "pending_event_id") == manifest.Prep.PendingEventID &&
				finalWriterV2EventString(event, "plan_event_id") == manifest.Prep.PlanEventID {
				foundEdited = true
			}
		}
		if !foundEdited {
			return fmt.Errorf("prep reviewed Part edited event missing for part %d", part.PartIndex)
		}
	}
	return nil
}

func finalWriterV2HasEvent(events []app.LedgerEvent, eventID string, eventType string) bool {
	for _, event := range events {
		if event.EventID == eventID && event.EventType == eventType {
			return true
		}
	}
	return false
}

func finalWriterV2CompactJSON(raw json.RawMessage) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return strings.TrimSpace(string(raw))
	}
	return buf.String()
}
