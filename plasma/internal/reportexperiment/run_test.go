package reportexperiment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestRunProviderFreeV3FinalizationWritesArchiveOutputs(t *testing.T) {
	ctx := context.Background()
	archive := t.TempDir()
	repo := t.TempDir()
	fixturePath := writeTestFixture(t, archive, "run-v3")
	rewriteFixture(t, fixturePath, func(fixture *Fixture) {
		fixture.SourceProvenance.Notes = "do-not-leak-free-form-notes"
	})
	executor := &fakeV3Executor{}
	result, err := Run(ctx, Config{
		ArchiveRoot: archive, FixturePath: fixturePath, RunID: "run-v3", RepositoryRoot: repo,
		AgentModel: " gpt-test ", AgentReasoningEffort: " HIGH ",
		ServiceFactory: sqliteServiceFactory,
		BinaryPair: BinaryPair{
			Experiment: BinaryMetadata{Path: "/tmp/plasma-report-experiment", SHA256: strings.Repeat("a", 64), VCSRevision: "abc123", VCSRevisionKnown: true, VCSModifiedKnown: true},
			PlasmaMCP:  BinaryMetadata{Path: "/tmp/plasma", SHA256: strings.Repeat("b", 64), VCSRevision: "abc123", VCSRevisionKnown: true, VCSModifiedKnown: true},
			Codex:      BinaryMetadata{Path: "/tmp/codex", SHA256: strings.Repeat("c", 64)},
		},
		ExecutorFactory: func(_ context.Context, input ExecutorContext) (agentexec.AgentExecutor, error) {
			executor.store = input.Service
			return executor, nil
		},
		StartedAt: time.Date(2026, 8, 9, 1, 2, 3, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for _, path := range []string{result.DBPath, result.ReportPath, result.LedgerPath, result.ManifestPath} {
		if info, err := os.Stat(path); err != nil || info.Size() == 0 {
			t.Fatalf("expected non-empty output at %s: info=%v err=%v", path, info, err)
		}
	}
	if got := result.Manifest.RunnerScope; got != RunnerScope {
		t.Fatalf("runner scope = %q", got)
	}
	if got := result.Manifest.FinalArtifact.ReportSHA256; got == "" || got != fileSHA256ForTest(t, result.ReportPath) {
		t.Fatalf("report sha mismatch: %q", got)
	}
	if got := result.Manifest.FinalArtifact.LedgerSHA256; got == "" || got != fileSHA256ForTest(t, result.LedgerPath) {
		t.Fatalf("ledger sha mismatch: %q", got)
	}
	if result.Manifest.Bootstrap.SessionID != "provider_reportexperiment_fake_bootstrap" {
		t.Fatalf("bootstrap session ID = %q", result.Manifest.Bootstrap.SessionID)
	}
	if result.Manifest.Bootstrap.PromptSHA256 != bytesSHA256([]byte(bootstrapPrompt)) {
		t.Fatalf("bootstrap prompt SHA mismatch: %q", result.Manifest.Bootstrap.PromptSHA256)
	}
	if result.Manifest.Binaries.Codex.SHA256 != strings.Repeat("c", 64) {
		t.Fatalf("Codex binary receipt missing: %#v", result.Manifest.Binaries.Codex)
	}
	if result.Manifest.Agent.Model != "gpt-test" || result.Manifest.Agent.ReasoningEffort != "high" {
		t.Fatalf("manifest agent was not normalized consistently: %#v", result.Manifest.Agent)
	}
	if !reflect.DeepEqual(result.Manifest.IntentionalDifferences, []string{
		"tools_disabled_bootstrap_session_is_a_finalization_only_harness_receipt_not_a_replacement_for_product_planning_stage",
	}) {
		t.Fatalf("unexpected intentional differences: %#v", result.Manifest.IntentionalDifferences)
	}
	gotRequests := make([]string, 0, len(result.Manifest.AgentRequests))
	for _, request := range result.Manifest.AgentRequests {
		if request.PromptSHA256 == "" {
			t.Fatalf("request %d has empty prompt SHA", request.Index)
		}
		if request.Model != "gpt-test" || request.ReasoningEffort != "high" {
			t.Fatalf("request %d model settings = %q/%q, want normalized gpt-test/high", request.Index, request.Model, request.ReasoningEffort)
		}
		if request.DisableTools {
			gotRequests = append(gotRequests, "bootstrap")
			if !request.IgnoreUserConfig {
				t.Fatalf("bootstrap request did not ignore user config: %#v", request)
			}
			if request.PromptSHA256 != result.Manifest.Bootstrap.PromptSHA256 {
				t.Fatalf("bootstrap request prompt SHA = %q, want %q", request.PromptSHA256, result.Manifest.Bootstrap.PromptSHA256)
			}
			continue
		}
		gotRequests = append(gotRequests, request.Stage)
		if request.IgnoreUserConfig {
			t.Fatalf("finalization request unexpectedly ignored user config: %#v", request)
		}
		if !request.ReplaceMCPTools {
			t.Fatalf("request %d did not replace MCP tools", request.Index)
		}
	}
	wantRequests := []string{
		"bootstrap",
		reporting.FinalEditStageWriter,
		reporting.FinalEditStageReader,
		reporting.FinalEditStageStyle,
		reporting.FinalEditStageStyleSemanticValidation,
		reporting.FinalEditStageEvidenceGate,
	}
	if !reflect.DeepEqual(gotRequests, wantRequests) {
		t.Fatalf("requests = %#v, want %#v", gotRequests, wantRequests)
	}
	if !reflect.DeepEqual(executor.calls, wantRequests) {
		t.Fatalf("executor calls = %#v, want %#v", executor.calls, wantRequests)
	}
	if !nodeWasObserved(result.Manifest.Observations, "reportassembly") ||
		!nodeWasObserved(result.Manifest.Observations, "finalwrite") ||
		!nodeWasObserved(result.Manifest.Observations, "readeredit") ||
		!nodeWasObserved(result.Manifest.Observations, "styleedit") ||
		!nodeWasObserved(result.Manifest.Observations, "semanticcheck") ||
		!nodeWasObserved(result.Manifest.Observations, "evidencecheck") ||
		!nodeWasObserved(result.Manifest.Observations, "finalstore") {
		t.Fatalf("manifest observations did not include the V3 final tail: %#v", result.Manifest.Observations)
	}
	evidenceIndex := terminalNodeIndex(result.Manifest.Observations, "evidencecheck")
	finalStoreIndex := terminalNodeIndex(result.Manifest.Observations, "finalstore")
	if evidenceIndex == 0 || finalStoreIndex == 0 || finalStoreIndex <= evidenceIndex {
		t.Fatalf("finalstore was not observed after evidencecheck: evidence=%d finalstore=%d observations=%#v", evidenceIndex, finalStoreIndex, result.Manifest.Observations)
	}
	manifestJSON, err := manifestJSON(result.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(manifestJSON), fixturePath) || strings.Contains(string(manifestJSON), filepath.Join(filepath.Dir(fixturePath), "part-01.md")) {
		t.Fatalf("manifest leaked absolute fixture or Part path: %s", string(manifestJSON))
	}
	if strings.Contains(string(manifestJSON), "do-not-leak-free-form-notes") {
		t.Fatalf("manifest leaked free-form provenance notes: %s", string(manifestJSON))
	}
	if result.Manifest.Fixture.FixtureID == "" || result.Manifest.Fixture.SHA256 == "" ||
		len(result.Manifest.Fixture.Parts) != 1 || result.Manifest.Fixture.Parts[0].SHA256 == "" {
		t.Fatalf("manifest fixture receipt is incomplete: %#v", result.Manifest.Fixture)
	}
	ledgerBytes, err := os.ReadFile(result.LedgerPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ledgerBytes), "provider_reportexperiment_fake_bootstrap") {
		t.Fatalf("ledger did not include bootstrap session ID")
	}
	if !strings.Contains(string(ledgerBytes), "section_fanout_report") || strings.Contains(string(ledgerBytes), "fixed_reviewed_part_terminal_experiment") {
		t.Fatalf("ledger session chain kind did not match product section_fanout lineage")
	}
	if result.Manifest.Seed.SessionChainKind != "section_fanout_report" ||
		result.Manifest.Seed.ReportPlanSessionID != "provider_reportexperiment_fake_bootstrap" ||
		result.Manifest.Seed.PreReportResearchSessionID != "" ||
		result.Manifest.Seed.ForkSourceAgentSessionID != "" {
		t.Fatalf("manifest seed session lineage = %#v", result.Manifest.Seed)
	}
	if _, err := Run(ctx, Config{
		ArchiveRoot: archive, FixturePath: fixturePath, RunID: "run-v3", RepositoryRoot: repo,
		AgentModel: "gpt-test", AgentReasoningEffort: "medium", ServiceFactory: sqliteServiceFactory,
		ExecutorFactory: func(_ context.Context, input ExecutorContext) (agentexec.AgentExecutor, error) {
			executor.store = input.Service
			return executor, nil
		},
	}); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("second run error = %v, want ErrConflict", err)
	}
}
