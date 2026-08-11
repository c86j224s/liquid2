package reportexperiment

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func sqliteServiceFactory(ctx context.Context, dbPath string) (ServiceHandle, error) {
	store, err := sqlite.Open(ctx, dbPath)
	if err != nil {
		return ServiceHandle{}, err
	}
	return ServiceHandle{Service: app.NewService(store), Close: store.Close}, nil
}

type fakeV3Executor struct {
	store reporting.FinalEditStageStore
	next  int
	calls []string
}

func (executor *fakeV3Executor) Run(ctx context.Context, req agentexec.AgentRequest) (agentexec.AgentResult, error) {
	if req.DisableTools && req.FinalEditStage == nil {
		executor.calls = append(executor.calls, "bootstrap")
		if strings.TrimSpace(req.Prompt) != bootstrapPrompt {
			return agentexec.AgentResult{}, errors.New("unexpected bootstrap prompt")
		}
		if len(req.ExtraMCPTools) != 0 || !req.ReplaceMCPTools {
			return agentexec.AgentResult{}, errors.New("bootstrap must be tools-disabled with replaced MCP tools")
		}
		if !req.IgnoreUserConfig {
			return agentexec.AgentResult{}, errors.New("bootstrap must ignore user Codex config")
		}
		return agentexec.AgentResult{Text: bootstrapResponseToken, SessionID: "provider_reportexperiment_fake_bootstrap"}, nil
	}
	if req.FinalEditStage == nil {
		return agentexec.AgentResult{}, errors.New("final edit stage is required")
	}
	binding := *req.FinalEditStage
	executor.calls = append(executor.calls, binding.Stage)
	if _, _, err := reporting.StartFinalEditStage(ctx, executor.store, executor.eventID("start"), binding); err != nil {
		return agentexec.AgentResult{}, err
	}
	switch binding.Stage {
	case reporting.FinalEditStageWriter, reporting.FinalEditStageReader:
		markdown, err := executor.sourceMarkdown(ctx, binding)
		if err != nil {
			return agentexec.AgentResult{}, err
		}
		if _, err := reporting.SubmitFinalEditStage(ctx, executor.store, binding, executor.eventID("submit"), markdown, 0); err != nil {
			return agentexec.AgentResult{}, err
		}
		return agentexec.AgentResult{Text: "FINAL_EDIT_STAGE_SUBMITTED", SessionID: binding.ProviderSessionID}, nil
	case reporting.FinalEditStageStyle:
		markdown, err := executor.sourceMarkdown(ctx, binding)
		if err != nil {
			return agentexec.AgentResult{}, err
		}
		if _, err := reporting.SubmitFinalEditStyleStage(ctx, executor.store, binding, executor.eventID("submit"), markdown, 0, nil); err != nil {
			return agentexec.AgentResult{}, err
		}
		return agentexec.AgentResult{Text: "FINAL_EDIT_STAGE_SUBMITTED", SessionID: binding.ProviderSessionID}, nil
	case reporting.FinalEditStageStyleSemanticValidation:
		if _, err := reporting.SubmitFinalEditStyleSemanticValidation(ctx, executor.store, binding, executor.eventID("submit"), nil); err != nil {
			return agentexec.AgentResult{}, err
		}
		return agentexec.AgentResult{Text: "FINAL_EDIT_STAGE_SUBMITTED", SessionID: binding.ProviderSessionID}, nil
	case reporting.FinalEditStageEvidenceGate:
		if req.LongFormFinalize == nil {
			return agentexec.AgentResult{}, errors.New("final binding is required")
		}
		if _, err := reporting.SubmitFinalEditEvidenceGate(ctx, executor.store, reporting.FinalEditEvidenceGateSubmitRequest{
			StageBinding: binding, FinalBinding: *req.LongFormFinalize, StageEventID: executor.eventID("submit"), CanonicalEventID: executor.eventID("canonical"),
		}); err != nil {
			return agentexec.AgentResult{}, err
		}
		return agentexec.AgentResult{Text: "REPORT_FINALIZED", SessionID: binding.ProviderSessionID}, nil
	default:
		return agentexec.AgentResult{}, errors.New("unsupported stage")
	}
}

func (executor *fakeV3Executor) ForkSession(_ context.Context, sourceSessionID string) (agentexec.AgentSessionForkResult, error) {
	executor.next++
	return agentexec.AgentSessionForkResult{
		SessionID:       fmt.Sprintf("provider_reportexperiment_fake_%03d", executor.next),
		SourceSessionID: sourceSessionID,
	}, nil
}

func (executor *fakeV3Executor) sourceMarkdown(ctx context.Context, binding reporting.FinalEditStageBinding) (string, error) {
	artifact, err := executor.store.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		return "", err
	}
	return string(artifact.Content), nil
}

func (executor *fakeV3Executor) eventID(kind string) string {
	executor.next++
	return fmt.Sprintf("evt_reportexperiment_fake_%s_%03d", kind, executor.next)
}
