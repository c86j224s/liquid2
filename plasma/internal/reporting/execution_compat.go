package reporting

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
)

const (
	DefaultMode  = reportexecution.DefaultMode
	ModeOneTake  = reportexecution.ModeOneTake
	ModePlanned  = reportexecution.ModePlanned
	ModeLongForm = reportexecution.ModeLongForm

	DefaultSessionPolicy      = reportexecution.DefaultSessionPolicy
	SessionPolicySameSession  = reportexecution.SessionPolicySameSession
	SessionPolicyIsolatedFork = reportexecution.SessionPolicyIsolatedFork
	SessionPolicyFreshSession = reportexecution.SessionPolicyFreshSession

	SessionPolicySelectionAutoFreshSession          = reportexecution.SessionPolicySelectionAutoFreshSession
	SessionPolicySelectionAutoIsolatedFork          = reportexecution.SessionPolicySelectionAutoIsolatedFork
	SessionPolicySelectionAutoSameSessionNoSession  = reportexecution.SessionPolicySelectionAutoSameSessionNoSession
	SessionPolicySelectionAutoSameSessionNoForker   = reportexecution.SessionPolicySelectionAutoSameSessionNoForker
	SessionPolicySelectionAutoSameSessionForkFailed = reportexecution.SessionPolicySelectionAutoSameSessionForkFailed
	SessionPolicySelectionAutoSameSessionOneTake    = reportexecution.SessionPolicySelectionAutoSameSessionOneTake
	SessionPolicySelectionExplicitIsolatedFork      = reportexecution.SessionPolicySelectionExplicitIsolatedFork
	SessionPolicySelectionExplicitSameSession       = reportexecution.SessionPolicySelectionExplicitSameSession

	DesignTargetDesigned = reportexecution.DesignTargetDesigned

	ExportKindSelfContainedHTML   = reportexecution.ExportKindSelfContainedHTML
	ExportKindDesignedHTML        = reportexecution.ExportKindDesignedHTML
	ExportKindHumanizedMarkdown   = reportexecution.ExportKindHumanizedMarkdown
	ExportTargetSelfContainedHTML = reportexecution.ExportTargetSelfContainedHTML
	ExportTargetDesignedHTML      = reportexecution.ExportTargetDesignedHTML
	ExportTargetHumanizedMarkdown = reportexecution.ExportTargetHumanizedMarkdown
	DesignedContentModelContract  = reportexecution.DesignedContentModelContract
	HumanizeProfileH5             = reportexecution.HumanizeProfileH5
	HumanizeTransportPatch        = reportexecution.HumanizeTransportPatch

	DirectionAdvisory = reportexecution.DirectionAdvisory
)

type Service = reportexecution.Service
type DraftRequest = reportexecution.DraftRequest
type SessionPolicySelectionInput = reportexecution.SessionPolicySelectionInput
type DesignRequest = reportexecution.DesignRequest
type HumanizeRequest = reportexecution.HumanizeRequest
type PatchRequest = reportexecution.PatchRequest
type PatchFinalizedEventRequest = reportexecution.PatchFinalizedEventRequest
type SelfContainedHTMLExportEventRequest = reportexecution.SelfContainedHTMLExportEventRequest
type DesignedHTMLExportEventRequest = reportexecution.DesignedHTMLExportEventRequest
type FailurePayloadProvider = reportexecution.FailurePayloadProvider
type InFlight = reportexecution.InFlight
type StageFailureError = reportexecution.StageFailureError
type StageFailureRequest = reportexecution.StageFailureRequest

// Runner preserves the reporting package's historical execution surface while delegating to reportexecution.
type Runner reportexecution.Runner

func NormalizeMode(mode string) (string, error) { return reportexecution.NormalizeMode(mode) }
func NormalizeSessionPolicy(policy string) (string, error) {
	return reportexecution.NormalizeSessionPolicy(policy)
}
func SelectSessionPolicy(input SessionPolicySelectionInput) (string, string, error) {
	return reportexecution.SelectSessionPolicy(input)
}
func ValidateSessionPolicy(policy string, reportMode string, canForkSession bool, hasPreReportResearchSession bool, forkReady bool) error {
	return reportexecution.ValidateSessionPolicy(policy, reportMode, canForkSession, hasPreReportResearchSession, forkReady)
}
func ModeLabel(mode string) string { return reportexecution.ModeLabel(mode) }
func NormalizeDirectionHint(value string) string {
	return reportexecution.NormalizeDirectionHint(value)
}
func FormatDirectionHint(value string) string { return reportexecution.FormatDirectionHint(value) }
func NewStageFailure(kind, planID string, part, section int, cause error) *StageFailureError {
	return reportexecution.NewStageFailure(kind, planID, part, section, cause)
}

func BuildSelfContainedHTMLExportAppendRequest(req SelfContainedHTMLExportEventRequest) ledger.AppendRequest {
	return reportexecution.BuildSelfContainedHTMLExportAppendRequest(req)
}
func BuildDesignedHTMLExportAppendRequest(req DesignedHTMLExportEventRequest) ledger.AppendRequest {
	return reportexecution.BuildDesignedHTMLExportAppendRequest(req)
}
func BuildPatchFinalizedAppendRequest(req PatchFinalizedEventRequest) ledger.AppendRequest {
	return reportexecution.BuildPatchFinalizedAppendRequest(req)
}
func DraftRequestFromPendingEvent(event ledger.Event) (DraftRequest, error) {
	return reportexecution.DraftRequestFromPendingEvent(event)
}
func DesignRequestFromPendingEvent(event ledger.Event) (DesignRequest, error) {
	return reportexecution.DesignRequestFromPendingEvent(event)
}
func HumanizeRequestFromPendingEvent(event ledger.Event) (HumanizeRequest, error) {
	return reportexecution.HumanizeRequestFromPendingEvent(event)
}
func PatchRequestFromPendingEvent(event ledger.Event) (PatchRequest, error) {
	return reportexecution.PatchRequestFromPendingEvent(event)
}
func CompletedPendingEventIDs(events []ledger.Event) map[string]struct{} {
	return reportexecution.CompletedPendingEventIDs(events)
}

func (runner Runner) StartDraft(ctx context.Context, missionID string, req DraftRequest, producer ledger.Producer) (ledger.Event, error) {
	return runner.exec().StartDraft(ctx, missionID, req, producer)
}
func (runner Runner) ResumeDraft(ctx context.Context, missionID string, pending ledger.Event) error {
	return runner.exec().ResumeDraft(ctx, missionID, pending)
}
func (runner Runner) RunDraft(ctx context.Context, missionID string, req DraftRequest, pendingEventID string) error {
	return runner.exec().RunDraft(ctx, missionID, req, pendingEventID)
}
func (runner Runner) StartDesign(ctx context.Context, missionID string, req DesignRequest, producer ledger.Producer) (ledger.Event, error) {
	return runner.exec().StartDesign(ctx, missionID, req, producer)
}
func (runner Runner) ResumeDesign(ctx context.Context, missionID string, pending ledger.Event) error {
	return runner.exec().ResumeDesign(ctx, missionID, pending)
}
func (runner Runner) RunDesign(ctx context.Context, missionID string, req DesignRequest, pendingEventID string) error {
	return runner.exec().RunDesign(ctx, missionID, req, pendingEventID)
}
func (runner Runner) StartHumanize(ctx context.Context, missionID string, req HumanizeRequest, producer ledger.Producer) (ledger.Event, error) {
	return runner.exec().StartHumanize(ctx, missionID, req, producer)
}
func (runner Runner) ResumeHumanize(ctx context.Context, missionID string, pending ledger.Event) error {
	return runner.exec().ResumeHumanize(ctx, missionID, pending)
}
func (runner Runner) RunHumanize(ctx context.Context, missionID string, req HumanizeRequest, pendingEventID string) error {
	return runner.exec().RunHumanize(ctx, missionID, req, pendingEventID)
}
func (runner Runner) StartPatch(ctx context.Context, missionID string, req PatchRequest, producer ledger.Producer) (ledger.Event, error) {
	return runner.exec().StartPatch(ctx, missionID, req, producer)
}
func (runner Runner) ResumePatch(ctx context.Context, missionID string, pending ledger.Event) error {
	return runner.exec().ResumePatch(ctx, missionID, pending)
}
func (runner Runner) RunPatch(ctx context.Context, missionID string, req PatchRequest, pendingEventID string) error {
	return runner.exec().RunPatch(ctx, missionID, req, pendingEventID)
}
func (runner Runner) AppendDraftFailed(ctx context.Context, missionID string, pendingEventID string, executor string, reportMode string, cause error) (ledger.Event, error) {
	return runner.exec().AppendDraftFailed(ctx, missionID, pendingEventID, executor, reportMode, cause)
}
func (runner Runner) AppendPatchFailed(ctx context.Context, missionID string, pendingEventID string, executor string, baseArtifactID string, cause error) (ledger.Event, error) {
	return runner.exec().AppendPatchFailed(ctx, missionID, pendingEventID, executor, baseArtifactID, cause)
}
func (runner Runner) AppendHumanizeFailed(ctx context.Context, missionID string, pendingEventID string, executor string, sourceArtifactID string, reportMode string, cause error) (ledger.Event, error) {
	return runner.exec().AppendHumanizeFailed(ctx, missionID, pendingEventID, executor, sourceArtifactID, reportMode, cause)
}
func (runner Runner) AppendDesignFailed(ctx context.Context, missionID string, pendingEventID string, executor string, sourceArtifactID string, rendererVersion string, cause error) (ledger.Event, error) {
	return runner.exec().AppendDesignFailed(ctx, missionID, pendingEventID, executor, sourceArtifactID, rendererVersion, cause)
}
func (runner Runner) AppendCanceled(ctx context.Context, missionID string, pending ledger.Event, canceledInFlight bool, producer ledger.Producer) (ledger.Event, error) {
	return runner.exec().AppendCanceled(ctx, missionID, pending, canceledInFlight, producer)
}
func (runner Runner) AppendDraftCanceled(ctx context.Context, missionID string, pending ledger.Event, canceledInFlight bool, producer ledger.Producer) (ledger.Event, error) {
	return runner.exec().AppendDraftCanceled(ctx, missionID, pending, canceledInFlight, producer)
}
func (runner Runner) AppendPatchCanceled(ctx context.Context, missionID string, pending ledger.Event, canceledInFlight bool, producer ledger.Producer) (ledger.Event, error) {
	return runner.exec().AppendPatchCanceled(ctx, missionID, pending, canceledInFlight, producer)
}
func (runner Runner) AppendDesignCanceled(ctx context.Context, missionID string, pending ledger.Event, canceledInFlight bool, producer ledger.Producer) (ledger.Event, error) {
	return runner.exec().AppendDesignCanceled(ctx, missionID, pending, canceledInFlight, producer)
}
func (runner Runner) AppendHumanizeCanceled(ctx context.Context, missionID string, pending ledger.Event, canceledInFlight bool, producer ledger.Producer) (ledger.Event, error) {
	return runner.exec().AppendHumanizeCanceled(ctx, missionID, pending, canceledInFlight, producer)
}
func (runner Runner) AppendStageFailure(ctx context.Context, req StageFailureRequest) (ledger.Event, error) {
	return runner.exec().AppendStageFailure(ctx, req)
}

func (runner Runner) exec() reportexecution.Runner { return reportexecution.Runner(runner) }

func (runner Runner) id(prefix string) string {
	if runner.NewID == nil {
		return prefix + "_report"
	}
	return runner.NewID(prefix)
}

func mustJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
