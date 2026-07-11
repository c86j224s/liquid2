package reporting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	DefaultMode  = ModePlanned
	ModeOneTake  = "one_take"
	ModePlanned  = "planned"
	ModeLongForm = "long_form"

	DefaultSessionPolicy      = SessionPolicySameSession
	SessionPolicySameSession  = "same_session"
	SessionPolicyIsolatedFork = "isolated_fork"

	SessionPolicySelectionAutoIsolatedFork          = "auto_isolated_fork"
	SessionPolicySelectionAutoSameSessionNoSession  = "auto_same_session_no_pre_report_session"
	SessionPolicySelectionAutoSameSessionNoForker   = "auto_same_session_no_forkable_executor"
	SessionPolicySelectionAutoSameSessionForkFailed = "auto_same_session_fork_unavailable"
	SessionPolicySelectionAutoSameSessionOneTake    = "auto_same_session_one_take"
	SessionPolicySelectionExplicitIsolatedFork      = "explicit_isolated_fork"
	SessionPolicySelectionExplicitSameSession       = "explicit_same_session"

	labelOneTake  = "원테이크 보고서"
	labelPlanned  = "보고서"
	labelLongForm = "장문 보고서"

	DesignTargetDesigned = "designed_html"

	ExportKindSelfContainedHTML   = "self_contained_html_report_artifact"
	ExportKindDesignedHTML        = "designed_html_report_artifact"
	ExportKindHumanizedMarkdown   = "humanized_markdown_report_artifact"
	ExportTargetSelfContainedHTML = "self_contained_html"
	ExportTargetDesignedHTML      = "designed_html"
	ExportTargetHumanizedMarkdown = "humanized_markdown"
	DesignedContentModelContract  = "dh26_inline_images"
	HumanizeProfileH5             = "h5-full-report-tone-pass"
	HumanizeTransportPatch        = "mcp_patch"
)

type Service interface {
	AppendEvent(context.Context, app.AppendEventRequest) (app.LedgerEvent, error)
	AppendEventsIfNoActiveAgentWork(context.Context, string, []app.AppendEventRequest) ([]app.LedgerEvent, error)
	ListEvents(context.Context, string) ([]app.LedgerEvent, error)
}

type DraftRequest struct {
	Title                        string
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	MCPMode                      string
	RigorLevel                   string
	RigorLabel                   string
	ReportMode                   string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	PostReportHumanize           string
	GenerationGuidanceProfile    string
	GenerationGuidanceSHA256     string
}

type SessionPolicySelectionInput struct {
	RequestedPolicy             string
	ReportMode                  string
	CanForkSession              bool
	HasPreReportResearchSession bool
	ForkReady                   bool
}

type DesignRequest struct {
	SourceArtifactID     string
	SourceMediaType      string
	Title                string
	AgentExecutor        string
	AgentModel           string
	AgentReasoningEffort string
	RendererVersion      string
}

type HumanizeRequest struct {
	SourceArtifactID       string
	SourceArtifactSHA256   string
	SourceMediaType        string
	Title                  string
	AgentExecutor          string
	AgentModel             string
	AgentReasoningEffort   string
	MCPMode                string
	PreviousAgentSessionID string
	ToolSessionID          string
	ReportMode             string
	ReportPendingEventID   string
}

type PatchRequest struct {
	BaseArtifactID               string
	Instruction                  string
	Title                        string
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	MCPMode                      string
	ReportSessionID              string
	PreviousAgentSessionID       string
	ForkSourceAgentSessionID     string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	SessionChainKind             string
}

type PatchFinalizedEventRequest struct {
	EventID                      string
	MissionID                    string
	CorrelationID                string
	PendingEventID               string
	Title                        string
	Artifact                     app.RawArtifact
	BaseArtifactID               string
	PatchID                      string
	PatchInstruction             string
	PatchSummary                 string
	OperationCount               int
	Operations                   any
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	AgentSessionID               string
	PreviousAgentSessionID       string
	ReturnedAgentSessionID       string
	ReportSessionID              string
	ForkSourceAgentSessionID     string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	ToolSessionID                string
	MCPMode                      string
	ProducerToolName             string
	SessionChainKind             string
	Producer                     app.Producer
}

type SelfContainedHTMLExportEventRequest struct {
	EventID          string
	MissionID        string
	SourceArtifactID string
	Artifact         app.RawArtifact
	Producer         app.Producer
}

type DesignedHTMLExportEventRequest struct {
	EventID                string
	MissionID              string
	PendingEventID         string
	SourceArtifactID       string
	ContentModelArtifactID string
	Artifact               app.RawArtifact
	RendererVersion        string
	ImageSetFingerprint    string
	AgentExecutor          string
	AgentModel             string
	AgentReasoningEffort   string
	AgentSessionID         string
	ToolSessionID          string
	DurationMS             int64
	AgentDurationMS        int64
	AgentUsage             agentusage.AgentUsage
	AgentResumed           bool
	Producer               app.Producer
}

type Runner struct {
	Service          Service
	InFlight         *InFlight
	NewID            func(string) string
	GenerateDraft    func(context.Context, string, DraftRequest, string) error
	GenerateDesign   func(context.Context, string, DesignRequest, string) error
	GenerateHumanize func(context.Context, string, HumanizeRequest, string) error
	GeneratePatch    func(context.Context, string, PatchRequest, string) error
}

func BuildSelfContainedHTMLExportAppendRequest(req SelfContainedHTMLExportEventRequest) app.AppendEventRequest {
	artifact := req.Artifact
	return app.AppendEventRequest{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: "report.artifact.exported",
		Producer:  req.Producer,
		Payload: mustJSON(map[string]any{
			"kind":               ExportKindSelfContainedHTML,
			"source_artifact_id": req.SourceArtifactID,
			"artifact_id":        artifact.ArtifactID,
			"media_type":         artifact.MediaType,
			"target":             ExportTargetSelfContainedHTML,
			"text":               "Self-contained HTML 리포트 artifact를 생성했습니다.",
		}),
	}
}

func BuildDesignedHTMLExportAppendRequest(req DesignedHTMLExportEventRequest) app.AppendEventRequest {
	artifact := req.Artifact
	payload := map[string]any{
		"kind":                      ExportKindDesignedHTML,
		"pending_event_id":          req.PendingEventID,
		"source_artifact_id":        req.SourceArtifactID,
		"content_model_artifact_id": req.ContentModelArtifactID,
		"artifact_id":               artifact.ArtifactID,
		"media_type":                artifact.MediaType,
		"target":                    ExportTargetDesignedHTML,
		"renderer_version":          req.RendererVersion,
		"content_model_contract":    DesignedContentModelContract,
		"image_set_fingerprint":     req.ImageSetFingerprint,
		"agent_executor":            req.AgentExecutor,
		"agent_model":               req.AgentModel,
		"agent_reasoning_effort":    req.AgentReasoningEffort,
		"agent_session_id":          req.AgentSessionID,
		"tool_session_id":           req.ToolSessionID,
		"duration_ms":               req.DurationMS,
		"text":                      "Designed HTML 리포트 artifact를 생성했습니다.",
	}
	if eventUsage, ok := req.AgentUsage.ForEvent("report_design", req.AgentDurationMS, "", req.AgentSessionID, req.AgentResumed, false); ok {
		payload["agent_usage"] = eventUsage
	}
	return app.AppendEventRequest{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: "report.artifact.exported",
		Producer:  req.Producer,
		Payload:   mustJSON(payload),
	}
}

func BuildPatchFinalizedAppendRequest(req PatchFinalizedEventRequest) app.AppendEventRequest {
	artifact := req.Artifact
	return app.AppendEventRequest{
		EventID:       strings.TrimSpace(req.EventID),
		MissionID:     strings.TrimSpace(req.MissionID),
		EventType:     "report.patch.finalized",
		Producer:      req.Producer,
		CorrelationID: strings.TrimSpace(req.CorrelationID),
		Payload: mustJSON(map[string]any{
			"kind":                            "markdown_report_patch_finalized",
			"pending_event_id":                req.PendingEventID,
			"title":                           req.Title,
			"artifact_id":                     artifact.ArtifactID,
			"media_type":                      artifact.MediaType,
			"byte_size":                       artifact.ByteSize,
			"sha256":                          artifact.SHA256,
			"filename":                        artifact.Filename,
			"base_artifact_id":                req.BaseArtifactID,
			"base_report_artifact_id":         req.BaseArtifactID,
			"patch_id":                        req.PatchID,
			"patch_instruction":               req.PatchInstruction,
			"patch_summary":                   req.PatchSummary,
			"operation_count":                 req.OperationCount,
			"operations":                      req.Operations,
			"agent_executor":                  req.AgentExecutor,
			"agent_model":                     req.AgentModel,
			"agent_reasoning_effort":          req.AgentReasoningEffort,
			"agent_session_id":                req.AgentSessionID,
			"previous_agent_session_id":       req.PreviousAgentSessionID,
			"returned_agent_session_id":       req.ReturnedAgentSessionID,
			"report_session_id":               req.ReportSessionID,
			"fork_source_agent_session_id":    req.ForkSourceAgentSessionID,
			"report_session_policy":           req.ReportSessionPolicy,
			"report_session_policy_selection": req.ReportSessionPolicySelection,
			"tool_session_id":                 req.ToolSessionID,
			"mcp_mode":                        req.MCPMode,
			"producer_tool_name":              req.ProducerToolName,
			"composition_strategy":            "mcp_patch_markdown",
			"session_chain_kind":              req.SessionChainKind,
			"post_report_research_session_id": "",
			"text":                            "MCP 패치 방식으로 Markdown 리포트 artifact 새 버전을 생성했습니다.",
		}),
	}
}

type FailurePayloadProvider interface {
	FailurePayload() map[string]any
}

type InFlight struct {
	mu    sync.Mutex
	runs  map[string]inFlightRun
	newID func(string) string
}

type inFlightRun struct {
	id             string
	pendingEventID string
	cancel         context.CancelFunc
}

func (runs *InFlight) SetNewID(newID func(string) string) {
	runs.mu.Lock()
	defer runs.mu.Unlock()
	runs.newID = newID
}

func (runs *InFlight) Start(missionID string, pendingEventID string, cancel context.CancelFunc) (string, bool) {
	runs.mu.Lock()
	defer runs.mu.Unlock()
	if runs.runs == nil {
		runs.runs = map[string]inFlightRun{}
	}
	if _, ok := runs.runs[missionID]; ok {
		return "", false
	}
	newID := runs.newID
	if newID == nil {
		newID = func(prefix string) string { return prefix + "_report" }
	}
	id := newID("run")
	runs.runs[missionID] = inFlightRun{id: id, pendingEventID: pendingEventID, cancel: cancel}
	return id, true
}

func (runs *InFlight) Finish(missionID string, id string) {
	runs.mu.Lock()
	defer runs.mu.Unlock()
	if runs.runs == nil {
		return
	}
	if current, ok := runs.runs[missionID]; ok && current.id == id {
		delete(runs.runs, missionID)
	}
}

func (runs *InFlight) Owns(missionID string, pendingEventID string) bool {
	runs.mu.Lock()
	defer runs.mu.Unlock()
	if runs.runs == nil {
		return false
	}
	current, ok := runs.runs[missionID]
	return ok && current.pendingEventID == pendingEventID
}

func (runs *InFlight) PendingEventID(missionID string) (string, bool) {
	runs.mu.Lock()
	defer runs.mu.Unlock()
	if runs.runs == nil {
		return "", false
	}
	current, ok := runs.runs[missionID]
	if !ok {
		return "", false
	}
	return current.pendingEventID, true
}

func (runs *InFlight) Cancel(missionID string, pendingEventID string) bool {
	var cancel context.CancelFunc
	runs.mu.Lock()
	if runs.runs != nil {
		if current, ok := runs.runs[missionID]; ok && (strings.TrimSpace(pendingEventID) == "" || current.pendingEventID == pendingEventID) {
			cancel = current.cancel
			delete(runs.runs, missionID)
		}
	}
	runs.mu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func NormalizeMode(mode string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(mode))
	if normalized == "" {
		return DefaultMode, nil
	}
	switch normalized {
	case "planned", "standard", "default":
		return ModePlanned, nil
	case "quick", "fast", "one-take", "one_take":
		return ModeOneTake, nil
	case "long", "long-form", "long_form":
		return ModeLongForm, nil
	default:
		return "", fmt.Errorf("%w: unsupported report mode", app.ErrInvalidInput)
	}
}

func NormalizeSessionPolicy(policy string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(policy))
	if normalized == "" {
		return DefaultSessionPolicy, nil
	}
	switch normalized {
	case "same", "same-session", "same_session", "default":
		return SessionPolicySameSession, nil
	case "isolated-fork", "isolated_fork", "fork":
		return SessionPolicyIsolatedFork, nil
	default:
		return "", fmt.Errorf("%w: unsupported report session policy", app.ErrInvalidInput)
	}
}

func SelectSessionPolicy(input SessionPolicySelectionInput) (string, string, error) {
	if strings.TrimSpace(input.RequestedPolicy) != "" {
		policy, err := NormalizeSessionPolicy(input.RequestedPolicy)
		if err != nil {
			return "", "", err
		}
		if err := ValidateSessionPolicy(policy, input.ReportMode, input.CanForkSession, input.HasPreReportResearchSession, input.ForkReady); err != nil {
			return "", "", err
		}
		if policy == SessionPolicyIsolatedFork {
			return policy, SessionPolicySelectionExplicitIsolatedFork, nil
		}
		return policy, SessionPolicySelectionExplicitSameSession, nil
	}
	mode, err := NormalizeMode(input.ReportMode)
	if err != nil {
		return "", "", err
	}
	if mode == ModeOneTake {
		return SessionPolicySameSession, SessionPolicySelectionAutoSameSessionOneTake, nil
	}
	if !input.CanForkSession {
		return SessionPolicySameSession, SessionPolicySelectionAutoSameSessionNoForker, nil
	}
	if !input.HasPreReportResearchSession {
		return SessionPolicySameSession, SessionPolicySelectionAutoSameSessionNoSession, nil
	}
	if !input.ForkReady {
		return SessionPolicySameSession, SessionPolicySelectionAutoSameSessionForkFailed, nil
	}
	return SessionPolicyIsolatedFork, SessionPolicySelectionAutoIsolatedFork, nil
}

func ValidateSessionPolicy(policy string, reportMode string, canForkSession bool, hasPreReportResearchSession bool, forkReady bool) error {
	policy, err := NormalizeSessionPolicy(policy)
	if err != nil {
		return err
	}
	if policy == SessionPolicySameSession {
		return nil
	}
	mode, err := NormalizeMode(reportMode)
	if err != nil {
		return err
	}
	if mode == ModeOneTake {
		return fmt.Errorf("%w: report session policy %q is not supported for one-take reports", app.ErrInvalidInput, policy)
	}
	if !canForkSession {
		return fmt.Errorf("%w: report session policy %q is unavailable because this executor cannot fork provider sessions", app.ErrInvalidInput, policy)
	}
	if !hasPreReportResearchSession {
		return fmt.Errorf("%w: report session policy %q requires a pre-report research session", app.ErrInvalidInput, policy)
	}
	if !forkReady {
		return fmt.Errorf("%w: report session policy %q is unavailable because the provider session cannot be prepared for fork", app.ErrInvalidInput, policy)
	}
	return nil
}

func ModeLabel(mode string) string {
	switch mode {
	case ModeLongForm:
		return labelLongForm
	case ModeOneTake:
		return labelOneTake
	default:
		return labelPlanned
	}
}

func (runner Runner) StartDraft(ctx context.Context, missionID string, req DraftRequest, producer app.Producer) (app.LedgerEvent, error) {
	req = normalizeDraftRequest(req)
	appended, err := runner.Service.AppendEventsIfNoActiveAgentWork(ctx, missionID, []app.AppendEventRequest{{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.draft.pending",
		Producer:  producer,
		Payload: mustJSON(map[string]any{
			"kind":                            "markdown_report_artifact_pending",
			"title":                           req.Title,
			"agent_executor":                  req.AgentExecutor,
			"agent_model":                     req.AgentModel,
			"agent_reasoning_effort":          req.AgentReasoningEffort,
			"mcp_mode":                        req.MCPMode,
			"rigor_level":                     req.RigorLevel,
			"rigor_label":                     req.RigorLabel,
			"report_mode":                     req.ReportMode,
			"report_mode_label":               ModeLabel(req.ReportMode),
			"report_session_policy":           req.ReportSessionPolicy,
			"report_session_policy_selection": req.ReportSessionPolicySelection,
			"post_report_humanize":            req.PostReportHumanize,
			"humanize_enabled":                req.PostReportHumanize != "disabled",
			"generation_guidance_profile":     req.GenerationGuidanceProfile,
			"generation_guidance_sha256":      req.GenerationGuidanceSHA256,
			"text":                            "리포트 초안 생성 중입니다.",
			"started_at":                      time.Now().UTC().Format(time.RFC3339Nano),
		}),
	}})
	if err != nil {
		return app.LedgerEvent{}, err
	}
	pending := appended[0]
	return pending, runner.RunDraft(context.Background(), missionID, req, pending.EventID)
}

func (runner Runner) ResumeDraft(ctx context.Context, missionID string, pending app.LedgerEvent) error {
	req, err := DraftRequestFromPendingEvent(pending)
	if err != nil {
		_, failErr := runner.AppendDraftFailed(ctx, missionID, pending.EventID, "", "", err)
		return failErr
	}
	return runner.RunDraft(context.Background(), missionID, req, pending.EventID)
}

func (runner Runner) RunDraft(ctx context.Context, missionID string, req DraftRequest, pendingEventID string) error {
	if runner.InFlight == nil {
		return fmt.Errorf("%w: report runner requires in-flight registry", app.ErrInvalidInput)
	}
	if runner.GenerateDraft == nil {
		return fmt.Errorf("%w: report runner requires draft generator", app.ErrInvalidInput)
	}
	workerCtx, cancel := context.WithCancel(ctx)
	runID, ok := runner.InFlight.Start(missionID, pendingEventID, cancel)
	if !ok {
		cancel()
		if runner.isSamePendingAlreadyRunning(missionID, pendingEventID) || runner.hasTerminalEvent(context.Background(), missionID, pendingEventID) {
			return nil
		}
		_, _ = runner.AppendDraftFailed(context.Background(), missionID, pendingEventID, req.AgentExecutor, req.ReportMode, errors.New("report draft is already running for this mission"))
		return fmt.Errorf("%w: report draft is already running for this mission", app.ErrInvalidInput)
	}
	go func() {
		defer cancel()
		defer runner.InFlight.Finish(missionID, runID)
		if runner.hasTerminalEvent(context.Background(), missionID, pendingEventID) {
			return
		}
		if err := runner.GenerateDraft(workerCtx, missionID, req, pendingEventID); err != nil {
			failCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, _ = runner.AppendDraftFailed(failCtx, missionID, pendingEventID, req.AgentExecutor, req.ReportMode, err)
		}
	}()
	return nil
}

func (runner Runner) StartDesign(ctx context.Context, missionID string, req DesignRequest, producer app.Producer) (app.LedgerEvent, error) {
	appended, err := runner.Service.AppendEventsIfNoActiveAgentWork(ctx, missionID, []app.AppendEventRequest{{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.design.pending",
		Producer:  producer,
		Payload: mustJSON(map[string]any{
			"kind":                   "designed_html_report_pending",
			"source_artifact_id":     req.SourceArtifactID,
			"source_media_type":      req.SourceMediaType,
			"title":                  req.Title,
			"agent_executor":         req.AgentExecutor,
			"agent_model":            strings.TrimSpace(req.AgentModel),
			"agent_reasoning_effort": strings.TrimSpace(req.AgentReasoningEffort),
			"target":                 DesignTargetDesigned,
			"renderer_version":       req.RendererVersion,
			"text":                   "Designed HTML 리포트 artifact를 생성 중입니다.",
			"started_at":             time.Now().UTC().Format(time.RFC3339Nano),
		}),
	}})
	if err != nil {
		return app.LedgerEvent{}, err
	}
	pending := appended[0]
	return pending, runner.RunDesign(context.Background(), missionID, req, pending.EventID)
}

func (runner Runner) ResumeDesign(ctx context.Context, missionID string, pending app.LedgerEvent) error {
	req, err := DesignRequestFromPendingEvent(pending)
	if err != nil {
		_, failErr := runner.AppendDesignFailed(ctx, missionID, pending.EventID, "", "", "", err)
		return failErr
	}
	return runner.RunDesign(context.Background(), missionID, req, pending.EventID)
}

func (runner Runner) RunDesign(ctx context.Context, missionID string, req DesignRequest, pendingEventID string) error {
	if runner.InFlight == nil {
		return fmt.Errorf("%w: report runner requires in-flight registry", app.ErrInvalidInput)
	}
	if runner.GenerateDesign == nil {
		return fmt.Errorf("%w: report runner requires design generator", app.ErrInvalidInput)
	}
	workerCtx, cancel := context.WithCancel(ctx)
	runID, ok := runner.InFlight.Start(missionID, pendingEventID, cancel)
	if !ok {
		cancel()
		if runner.isSamePendingAlreadyRunning(missionID, pendingEventID) || runner.hasTerminalEvent(context.Background(), missionID, pendingEventID) {
			return nil
		}
		_, _ = runner.AppendDesignFailed(context.Background(), missionID, pendingEventID, req.AgentExecutor, req.SourceArtifactID, req.RendererVersion, errors.New("report draft is already running for this mission"))
		return fmt.Errorf("%w: report draft is already running for this mission", app.ErrInvalidInput)
	}
	go func() {
		defer cancel()
		defer runner.InFlight.Finish(missionID, runID)
		if runner.hasTerminalEvent(context.Background(), missionID, pendingEventID) {
			return
		}
		if err := runner.GenerateDesign(workerCtx, missionID, req, pendingEventID); err != nil {
			failCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, _ = runner.AppendDesignFailed(failCtx, missionID, pendingEventID, req.AgentExecutor, req.SourceArtifactID, req.RendererVersion, err)
		}
	}()
	return nil
}

func (runner Runner) StartHumanize(ctx context.Context, missionID string, req HumanizeRequest, producer app.Producer) (app.LedgerEvent, error) {
	req = normalizeHumanizeRequest(req)
	eventID := runner.id("evt")
	toolSessionID := firstNonEmpty(req.ToolSessionID, runner.id("ses"))
	req.ToolSessionID = toolSessionID
	appended, err := runner.Service.AppendEventsIfNoActiveAgentWork(ctx, missionID, []app.AppendEventRequest{{
		EventID:   eventID,
		MissionID: missionID,
		EventType: "report.humanize.pending",
		Producer:  producer,
		Payload: mustJSON(map[string]any{
			"kind":                      "humanized_markdown_report_pending",
			"target":                    "humanized_markdown",
			"profile":                   "h5-full-report-tone-pass",
			"pending_event_id":          eventID,
			"report_pending_event_id":   req.ReportPendingEventID,
			"title":                     req.Title,
			"source_artifact_id":        req.SourceArtifactID,
			"source_artifact_sha256":    req.SourceArtifactSHA256,
			"source_media_type":         req.SourceMediaType,
			"agent_executor":            req.AgentExecutor,
			"agent_model":               req.AgentModel,
			"agent_reasoning_effort":    req.AgentReasoningEffort,
			"previous_agent_session_id": req.PreviousAgentSessionID,
			"tool_session_id":           toolSessionID,
			"mcp_mode":                  req.MCPMode,
			"report_mode":               req.ReportMode,
			"report_mode_label":         ModeLabel(req.ReportMode),
			"humanize_transport":        "mcp_patch",
			"relationship":              "pending_post_report_tone_pass_of_source_artifact",
			"text":                      "H5 말투 보정 Markdown artifact를 생성하는 중입니다.",
			"started_at":                time.Now().UTC().Format(time.RFC3339Nano),
		}),
	}})
	if err != nil {
		return app.LedgerEvent{}, err
	}
	pending := appended[0]
	return pending, runner.RunHumanize(context.Background(), missionID, req, pending.EventID)
}

func (runner Runner) ResumeHumanize(ctx context.Context, missionID string, pending app.LedgerEvent) error {
	req, err := HumanizeRequestFromPendingEvent(pending)
	if err != nil {
		_, failErr := runner.AppendHumanizeFailed(ctx, missionID, pending.EventID, "", "", "", err)
		return failErr
	}
	return runner.RunHumanize(context.Background(), missionID, req, pending.EventID)
}

func (runner Runner) RunHumanize(ctx context.Context, missionID string, req HumanizeRequest, pendingEventID string) error {
	req = normalizeHumanizeRequest(req)
	if runner.InFlight == nil {
		return fmt.Errorf("%w: report runner requires in-flight registry", app.ErrInvalidInput)
	}
	if runner.GenerateHumanize == nil {
		return fmt.Errorf("%w: report runner requires humanize generator", app.ErrInvalidInput)
	}
	workerCtx, cancel := context.WithCancel(ctx)
	runID, ok := runner.InFlight.Start(missionID, pendingEventID, cancel)
	if !ok {
		cancel()
		if runner.isSamePendingAlreadyRunning(missionID, pendingEventID) || runner.hasTerminalEvent(context.Background(), missionID, pendingEventID) {
			return nil
		}
		_, _ = runner.AppendHumanizeFailed(context.Background(), missionID, pendingEventID, req.AgentExecutor, req.SourceArtifactID, req.ReportMode, errors.New("report draft is already running for this mission"))
		return fmt.Errorf("%w: report draft is already running for this mission", app.ErrInvalidInput)
	}
	go func() {
		defer cancel()
		defer runner.InFlight.Finish(missionID, runID)
		if runner.hasTerminalEvent(context.Background(), missionID, pendingEventID) {
			return
		}
		if err := runner.GenerateHumanize(workerCtx, missionID, req, pendingEventID); err != nil {
			failCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, _ = runner.AppendHumanizeFailed(failCtx, missionID, pendingEventID, req.AgentExecutor, req.SourceArtifactID, req.ReportMode, err)
		}
	}()
	return nil
}

func (runner Runner) StartPatch(ctx context.Context, missionID string, req PatchRequest, producer app.Producer) (app.LedgerEvent, error) {
	req = normalizePatchRequest(req)
	appended, err := runner.Service.AppendEventsIfNoActiveAgentWork(ctx, missionID, []app.AppendEventRequest{{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.patch.pending",
		Producer:  producer,
		Payload: mustJSON(map[string]any{
			"kind":                            "markdown_report_patch_pending",
			"base_artifact_id":                req.BaseArtifactID,
			"title":                           req.Title,
			"instruction":                     req.Instruction,
			"agent_executor":                  req.AgentExecutor,
			"agent_model":                     req.AgentModel,
			"agent_reasoning_effort":          req.AgentReasoningEffort,
			"mcp_mode":                        req.MCPMode,
			"report_session_id":               req.ReportSessionID,
			"previous_agent_session_id":       req.PreviousAgentSessionID,
			"fork_source_agent_session_id":    req.ForkSourceAgentSessionID,
			"report_session_policy":           req.ReportSessionPolicy,
			"report_session_policy_selection": req.ReportSessionPolicySelection,
			"session_chain_kind":              req.SessionChainKind,
			"text":                            "MCP 패치 방식으로 리포트 수정 중입니다.",
			"started_at":                      time.Now().UTC().Format(time.RFC3339Nano),
		}),
	}})
	if err != nil {
		return app.LedgerEvent{}, err
	}
	pending := appended[0]
	return pending, runner.RunPatch(context.Background(), missionID, req, pending.EventID)
}

func (runner Runner) ResumePatch(ctx context.Context, missionID string, pending app.LedgerEvent) error {
	req, err := PatchRequestFromPendingEvent(pending)
	if err != nil {
		_, failErr := runner.AppendPatchFailed(ctx, missionID, pending.EventID, "", "", err)
		return failErr
	}
	return runner.RunPatch(context.Background(), missionID, req, pending.EventID)
}

func (runner Runner) RunPatch(ctx context.Context, missionID string, req PatchRequest, pendingEventID string) error {
	if runner.InFlight == nil {
		return fmt.Errorf("%w: report runner requires in-flight registry", app.ErrInvalidInput)
	}
	if runner.GeneratePatch == nil {
		return fmt.Errorf("%w: report runner requires patch generator", app.ErrInvalidInput)
	}
	workerCtx, cancel := context.WithCancel(ctx)
	runID, ok := runner.InFlight.Start(missionID, pendingEventID, cancel)
	if !ok {
		cancel()
		if runner.isSamePendingAlreadyRunning(missionID, pendingEventID) || runner.hasTerminalEvent(context.Background(), missionID, pendingEventID) {
			return nil
		}
		_, _ = runner.AppendPatchFailed(context.Background(), missionID, pendingEventID, req.AgentExecutor, req.BaseArtifactID, errors.New("report draft is already running for this mission"))
		return fmt.Errorf("%w: report draft is already running for this mission", app.ErrInvalidInput)
	}
	go func() {
		defer cancel()
		defer runner.InFlight.Finish(missionID, runID)
		if runner.hasTerminalEvent(context.Background(), missionID, pendingEventID) {
			return
		}
		if err := runner.GeneratePatch(workerCtx, missionID, req, pendingEventID); err != nil {
			failCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, _ = runner.AppendPatchFailed(failCtx, missionID, pendingEventID, req.AgentExecutor, req.BaseArtifactID, err)
		}
	}()
	return nil
}

func (runner Runner) AppendDraftFailed(ctx context.Context, missionID string, pendingEventID string, executor string, reportMode string, cause error) (app.LedgerEvent, error) {
	if runner.hasTerminalEvent(ctx, missionID, pendingEventID) {
		return app.LedgerEvent{}, nil
	}
	executor = firstNonEmpty(strings.TrimSpace(executor), "plasma")
	mode, err := NormalizeMode(reportMode)
	if err != nil {
		mode = DefaultMode
	}
	payload := map[string]any{
		"kind":              "report_draft_failed",
		"pending_event_id":  pendingEventID,
		"agent_executor":    executor,
		"report_mode":       mode,
		"report_mode_label": ModeLabel(mode),
		"text":              "리포트 초안 생성에 실패했습니다.",
		"error":             cause.Error(),
		"failed_at":         time.Now().UTC().Format(time.RFC3339Nano),
	}
	mergeFailurePayload(payload, cause)
	return runner.Service.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.draft.failed",
		Producer:  app.Producer{Type: "agent", ID: executor},
		Payload:   mustJSON(payload),
	})
}

func (runner Runner) AppendPatchFailed(ctx context.Context, missionID string, pendingEventID string, executor string, baseArtifactID string, cause error) (app.LedgerEvent, error) {
	if runner.hasTerminalEvent(ctx, missionID, pendingEventID) {
		return app.LedgerEvent{}, nil
	}
	executor = validAgentExecutorOrEmpty(executor)
	producerID := firstNonEmpty(executor, "plasma")
	payload := map[string]any{
		"kind":             "report_patch_failed",
		"pending_event_id": pendingEventID,
		"base_artifact_id": baseArtifactID,
		"agent_executor":   executor,
		"text":             "MCP 리포트 패치에 실패했습니다.",
		"error":            cause.Error(),
		"failed_at":        time.Now().UTC().Format(time.RFC3339Nano),
	}
	mergeFailurePayload(payload, cause)
	return runner.Service.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.patch.failed",
		Producer:  app.Producer{Type: "agent", ID: producerID},
		Payload:   mustJSON(payload),
	})
}

func (runner Runner) AppendHumanizeFailed(ctx context.Context, missionID string, pendingEventID string, executor string, sourceArtifactID string, reportMode string, cause error) (app.LedgerEvent, error) {
	if runner.hasTerminalEvent(ctx, missionID, pendingEventID) {
		return app.LedgerEvent{}, nil
	}
	executor = validAgentExecutorOrEmpty(executor)
	producerID := firstNonEmpty(executor, "plasma")
	mode, err := NormalizeMode(reportMode)
	if err != nil {
		mode = DefaultMode
	}
	payload := map[string]any{
		"kind":                        "humanized_markdown_report_failed",
		"pending_event_id":            pendingEventID,
		"source_artifact_id":          sourceArtifactID,
		"agent_executor":              executor,
		"target":                      "humanized_markdown",
		"profile":                     "h5-full-report-tone-pass",
		"humanize_transport":          "mcp_patch",
		"report_mode":                 mode,
		"report_mode_label":           ModeLabel(mode),
		"relationship":                "failed_post_report_tone_pass_of_source_artifact",
		"preserved_original_markdown": true,
		"text":                        "H5 말투 보정이 실패해 원본 Markdown artifact를 유지했습니다.",
		"error":                       cause.Error(),
		"failed_at":                   time.Now().UTC().Format(time.RFC3339Nano),
	}
	mergeFailurePayload(payload, cause)
	return runner.Service.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.humanize.failed",
		Producer:  app.Producer{Type: "agent", ID: producerID},
		Payload:   mustJSON(payload),
	})
}

func (runner Runner) AppendDesignFailed(ctx context.Context, missionID string, pendingEventID string, executor string, sourceArtifactID string, rendererVersion string, cause error) (app.LedgerEvent, error) {
	if runner.hasTerminalEvent(ctx, missionID, pendingEventID) {
		return app.LedgerEvent{}, nil
	}
	executor = firstNonEmpty(strings.TrimSpace(executor), "plasma")
	payload := map[string]any{
		"kind":               "designed_html_report_failed",
		"pending_event_id":   pendingEventID,
		"source_artifact_id": sourceArtifactID,
		"agent_executor":     executor,
		"target":             DesignTargetDesigned,
		"renderer_version":   rendererVersion,
		"text":               "Designed HTML 리포트 artifact 생성에 실패했습니다.",
		"error":              cause.Error(),
		"failed_at":          time.Now().UTC().Format(time.RFC3339Nano),
	}
	mergeFailurePayload(payload, cause)
	return runner.Service.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.design.failed",
		Producer:  app.Producer{Type: "agent", ID: executor},
		Payload:   mustJSON(payload),
	})
}

func (runner Runner) AppendCanceled(ctx context.Context, missionID string, pending app.LedgerEvent, canceledInFlight bool, producer app.Producer) (app.LedgerEvent, error) {
	switch pending.EventType {
	case "report.design.pending":
		return runner.AppendDesignCanceled(ctx, missionID, pending, canceledInFlight, producer)
	case "report.humanize.pending":
		return runner.AppendHumanizeCanceled(ctx, missionID, pending, canceledInFlight, producer)
	case "report.patch.pending":
		return runner.AppendPatchCanceled(ctx, missionID, pending, canceledInFlight, producer)
	default:
		return runner.AppendDraftCanceled(ctx, missionID, pending, canceledInFlight, producer)
	}
}

func (runner Runner) AppendDraftCanceled(ctx context.Context, missionID string, pending app.LedgerEvent, canceledInFlight bool, producer app.Producer) (app.LedgerEvent, error) {
	if runner.hasTerminalEvent(ctx, missionID, pending.EventID) {
		return app.LedgerEvent{}, nil
	}
	var payload struct {
		AgentExecutor string `json:"agent_executor"`
		ReportMode    string `json:"report_mode"`
	}
	_ = json.Unmarshal(pending.Payload, &payload)
	executor := firstNonEmpty(payload.AgentExecutor, "plasma")
	mode, err := NormalizeMode(payload.ReportMode)
	if err != nil {
		mode = DefaultMode
	}
	return runner.Service.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.draft.failed",
		Producer:  producer,
		Payload: mustJSON(map[string]any{
			"kind":              "report_draft_canceled",
			"pending_event_id":  pending.EventID,
			"agent_executor":    executor,
			"report_mode":       mode,
			"report_mode_label": ModeLabel(mode),
			"text":              "리포트 초안 생성이 취소되었습니다.",
			"error":             "report draft canceled by user",
			"canceled":          true,
			"in_flight":         canceledInFlight,
			"canceled_at":       time.Now().UTC().Format(time.RFC3339Nano),
		}),
	})
}

func (runner Runner) AppendPatchCanceled(ctx context.Context, missionID string, pending app.LedgerEvent, canceledInFlight bool, producer app.Producer) (app.LedgerEvent, error) {
	if runner.hasTerminalEvent(ctx, missionID, pending.EventID) {
		return app.LedgerEvent{}, nil
	}
	req, err := PatchRequestFromPendingEvent(pending)
	if err != nil {
		req.AgentExecutor = "plasma"
	}
	executor := validAgentExecutorOrEmpty(req.AgentExecutor)
	return runner.Service.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.patch.failed",
		Producer:  producer,
		Payload: mustJSON(map[string]any{
			"kind":             "report_patch_canceled",
			"pending_event_id": pending.EventID,
			"base_artifact_id": req.BaseArtifactID,
			"agent_executor":   executor,
			"text":             "MCP 리포트 패치가 취소되었습니다.",
			"error":            "report patch canceled by user",
			"canceled":         true,
			"in_flight":        canceledInFlight,
			"canceled_at":      time.Now().UTC().Format(time.RFC3339Nano),
		}),
	})
}

func (runner Runner) AppendDesignCanceled(ctx context.Context, missionID string, pending app.LedgerEvent, canceledInFlight bool, producer app.Producer) (app.LedgerEvent, error) {
	if runner.hasTerminalEvent(ctx, missionID, pending.EventID) {
		return app.LedgerEvent{}, nil
	}
	var payload struct {
		SourceArtifactID string `json:"source_artifact_id"`
		AgentExecutor    string `json:"agent_executor"`
		RendererVersion  string `json:"renderer_version"`
	}
	_ = json.Unmarshal(pending.Payload, &payload)
	executor := firstNonEmpty(payload.AgentExecutor, "plasma")
	return runner.Service.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.design.failed",
		Producer:  producer,
		Payload: mustJSON(map[string]any{
			"kind":               "designed_html_report_canceled",
			"pending_event_id":   pending.EventID,
			"source_artifact_id": payload.SourceArtifactID,
			"agent_executor":     executor,
			"target":             DesignTargetDesigned,
			"renderer_version":   payload.RendererVersion,
			"text":               "Designed HTML 리포트 artifact 생성이 취소되었습니다.",
			"error":              "designed HTML report generation canceled by user",
			"canceled":           true,
			"in_flight":          canceledInFlight,
			"canceled_at":        time.Now().UTC().Format(time.RFC3339Nano),
		}),
	})
}

func (runner Runner) AppendHumanizeCanceled(ctx context.Context, missionID string, pending app.LedgerEvent, canceledInFlight bool, producer app.Producer) (app.LedgerEvent, error) {
	if runner.hasTerminalEvent(ctx, missionID, pending.EventID) {
		return app.LedgerEvent{}, nil
	}
	payload := humanizePendingPayloadFromEvent(pending)
	executor := firstNonEmpty(payload.AgentExecutor, "plasma")
	return runner.Service.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.humanize.failed",
		Producer:  producer,
		Payload: mustJSON(map[string]any{
			"kind":                        "humanized_markdown_report_canceled",
			"target":                      firstNonEmpty(payload.Target, ExportTargetHumanizedMarkdown),
			"profile":                     firstNonEmpty(payload.Profile, HumanizeProfileH5),
			"pending_event_id":            pending.EventID,
			"report_pending_event_id":     strings.TrimSpace(payload.ReportPendingEventID),
			"title":                       strings.TrimSpace(payload.Title),
			"source_artifact_id":          strings.TrimSpace(payload.SourceArtifactID),
			"source_artifact_sha256":      strings.TrimSpace(payload.SourceArtifactSHA256),
			"agent_executor":              executor,
			"agent_model":                 strings.TrimSpace(payload.AgentModel),
			"agent_reasoning_effort":      strings.TrimSpace(payload.AgentReasoningEffort),
			"previous_agent_session_id":   strings.TrimSpace(payload.PreviousSessionID),
			"tool_session_id":             strings.TrimSpace(payload.ToolSessionID),
			"mcp_mode":                    strings.TrimSpace(payload.MCPMode),
			"report_mode":                 strings.TrimSpace(payload.ReportMode),
			"report_mode_label":           strings.TrimSpace(payload.ReportModeLabel),
			"humanize_transport":          firstNonEmpty(payload.HumanizeTransport, HumanizeTransportPatch),
			"text":                        "H5 말투 보정이 취소되어 원본 Markdown artifact를 유지했습니다.",
			"error":                       "humanized Markdown report generation canceled by user",
			"canceled":                    true,
			"in_flight":                   canceledInFlight,
			"canceled_at":                 time.Now().UTC().Format(time.RFC3339Nano),
			"relationship":                "canceled_post_report_tone_pass_of_source_artifact",
			"preserved_original_markdown": true,
		}),
	})
}

func mergeFailurePayload(payload map[string]any, cause error) {
	var provider FailurePayloadProvider
	if !errors.As(cause, &provider) {
		return
	}
	for key, value := range provider.FailurePayload() {
		if !allowedFailurePayloadKey(key) || value == nil {
			continue
		}
		payload[key] = value
	}
}

func allowedFailurePayloadKey(key string) bool {
	switch key {
	case "agent_usage",
		"failed_surface",
		"agent_session_id",
		"previous_agent_session_id",
		"returned_agent_session_id",
		"tool_session_id",
		"resumed":
		return true
	default:
		return false
	}
}

func DraftRequestFromPendingEvent(event app.LedgerEvent) (DraftRequest, error) {
	var payload struct {
		Title                        string `json:"title"`
		AgentExecutor                string `json:"agent_executor"`
		AgentModel                   string `json:"agent_model"`
		AgentReasoningEffort         string `json:"agent_reasoning_effort"`
		MCPMode                      string `json:"mcp_mode"`
		RigorLevel                   string `json:"rigor_level"`
		RigorLabel                   string `json:"rigor_label"`
		ReportMode                   string `json:"report_mode"`
		ReportSessionPolicy          string `json:"report_session_policy"`
		ReportSessionPolicySelection string `json:"report_session_policy_selection"`
		PostReportHumanize           string `json:"post_report_humanize"`
		GenerationGuidanceProfile    string `json:"generation_guidance_profile"`
		GenerationGuidanceSHA256     string `json:"generation_guidance_sha256"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return DraftRequest{}, fmt.Errorf("%w: invalid report pending payload", app.ErrInvalidInput)
	}
	return normalizeDraftRequest(DraftRequest{
		Title:                        firstNonEmpty(payload.Title, "Mission report"),
		AgentExecutor:                firstNonEmpty(payload.AgentExecutor, "codex"),
		AgentModel:                   payload.AgentModel,
		AgentReasoningEffort:         payload.AgentReasoningEffort,
		MCPMode:                      firstNonEmpty(payload.MCPMode, "auto"),
		RigorLevel:                   payload.RigorLevel,
		RigorLabel:                   payload.RigorLabel,
		ReportMode:                   payload.ReportMode,
		ReportSessionPolicy:          payload.ReportSessionPolicy,
		ReportSessionPolicySelection: payload.ReportSessionPolicySelection,
		PostReportHumanize:           payload.PostReportHumanize,
		GenerationGuidanceProfile:    payload.GenerationGuidanceProfile,
		GenerationGuidanceSHA256:     payload.GenerationGuidanceSHA256,
	}), nil
}

func DesignRequestFromPendingEvent(event app.LedgerEvent) (DesignRequest, error) {
	var payload struct {
		SourceArtifactID     string `json:"source_artifact_id"`
		SourceMediaType      string `json:"source_media_type"`
		Title                string `json:"title"`
		AgentExecutor        string `json:"agent_executor"`
		AgentModel           string `json:"agent_model"`
		AgentReasoningEffort string `json:"agent_reasoning_effort"`
		RendererVersion      string `json:"renderer_version"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return DesignRequest{}, fmt.Errorf("%w: invalid designed HTML pending payload", app.ErrInvalidInput)
	}
	return DesignRequest{
		SourceArtifactID:     strings.TrimSpace(payload.SourceArtifactID),
		SourceMediaType:      strings.TrimSpace(payload.SourceMediaType),
		Title:                firstNonEmpty(payload.Title, "Mission report"),
		AgentExecutor:        firstNonEmpty(payload.AgentExecutor, "codex"),
		AgentModel:           strings.TrimSpace(payload.AgentModel),
		AgentReasoningEffort: strings.TrimSpace(payload.AgentReasoningEffort),
		RendererVersion:      strings.TrimSpace(payload.RendererVersion),
	}, nil
}

func HumanizeRequestFromPendingEvent(event app.LedgerEvent) (HumanizeRequest, error) {
	var payload humanizePendingPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return HumanizeRequest{}, fmt.Errorf("%w: invalid H5 humanize pending payload", app.ErrInvalidInput)
	}
	return normalizeHumanizeRequest(HumanizeRequest{
		SourceArtifactID:       payload.SourceArtifactID,
		SourceArtifactSHA256:   payload.SourceArtifactSHA256,
		SourceMediaType:        payload.SourceMediaType,
		Title:                  payload.Title,
		AgentExecutor:          payload.AgentExecutor,
		AgentModel:             payload.AgentModel,
		AgentReasoningEffort:   payload.AgentReasoningEffort,
		MCPMode:                payload.MCPMode,
		PreviousAgentSessionID: payload.PreviousSessionID,
		ToolSessionID:          payload.ToolSessionID,
		ReportMode:             payload.ReportMode,
		ReportPendingEventID:   payload.ReportPendingEventID,
	}), nil
}

type humanizePendingPayload struct {
	Target               string `json:"target"`
	Profile              string `json:"profile"`
	PendingEventID       string `json:"pending_event_id"`
	ReportPendingEventID string `json:"report_pending_event_id"`
	Title                string `json:"title"`
	SourceArtifactID     string `json:"source_artifact_id"`
	SourceArtifactSHA256 string `json:"source_artifact_sha256"`
	SourceMediaType      string `json:"source_media_type"`
	AgentExecutor        string `json:"agent_executor"`
	AgentModel           string `json:"agent_model"`
	AgentReasoningEffort string `json:"agent_reasoning_effort"`
	PreviousSessionID    string `json:"previous_agent_session_id"`
	ToolSessionID        string `json:"tool_session_id"`
	MCPMode              string `json:"mcp_mode"`
	ReportMode           string `json:"report_mode"`
	ReportModeLabel      string `json:"report_mode_label"`
	HumanizeTransport    string `json:"humanize_transport"`
}

func humanizePendingPayloadFromEvent(event app.LedgerEvent) humanizePendingPayload {
	var payload humanizePendingPayload
	_ = json.Unmarshal(event.Payload, &payload)
	return payload
}

func PatchRequestFromPendingEvent(event app.LedgerEvent) (PatchRequest, error) {
	var payload struct {
		BaseArtifactID               string `json:"base_artifact_id"`
		Instruction                  string `json:"instruction"`
		Title                        string `json:"title"`
		AgentExecutor                string `json:"agent_executor"`
		AgentModel                   string `json:"agent_model"`
		AgentReasoningEffort         string `json:"agent_reasoning_effort"`
		MCPMode                      string `json:"mcp_mode"`
		ReportSessionID              string `json:"report_session_id"`
		PreviousAgentSessionID       string `json:"previous_agent_session_id"`
		ForkSourceAgentSessionID     string `json:"fork_source_agent_session_id"`
		ReportSessionPolicy          string `json:"report_session_policy"`
		ReportSessionPolicySelection string `json:"report_session_policy_selection"`
		SessionChainKind             string `json:"session_chain_kind"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return PatchRequest{}, fmt.Errorf("%w: invalid report patch pending payload", app.ErrInvalidInput)
	}
	return normalizePatchRequest(PatchRequest{
		BaseArtifactID:               payload.BaseArtifactID,
		Instruction:                  payload.Instruction,
		Title:                        payload.Title,
		AgentExecutor:                payload.AgentExecutor,
		AgentModel:                   payload.AgentModel,
		AgentReasoningEffort:         payload.AgentReasoningEffort,
		MCPMode:                      payload.MCPMode,
		ReportSessionID:              payload.ReportSessionID,
		PreviousAgentSessionID:       payload.PreviousAgentSessionID,
		ForkSourceAgentSessionID:     payload.ForkSourceAgentSessionID,
		ReportSessionPolicy:          payload.ReportSessionPolicy,
		ReportSessionPolicySelection: payload.ReportSessionPolicySelection,
		SessionChainKind:             payload.SessionChainKind,
	}), nil
}

func CompletedPendingEventIDs(events []app.LedgerEvent) map[string]struct{} {
	completed := map[string]struct{}{}
	for _, event := range events {
		switch event.EventType {
		case "report.drafted", "report.artifact.created", "report.artifact.exported":
			if pendingEventID := pendingEventID(event); pendingEventID != "" {
				completed[pendingEventID] = struct{}{}
			}
		case "report.draft.failed", "report.design.failed", "report.humanize.failed", "report.humanize.skipped", "report.patch.failed":
			var payload struct {
				PendingEventID string `json:"pending_event_id"`
			}
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				continue
			}
			if pendingEventID := strings.TrimSpace(payload.PendingEventID); pendingEventID != "" {
				completed[pendingEventID] = struct{}{}
			}
		}
	}
	return completed
}

func (runner Runner) hasTerminalEvent(ctx context.Context, missionID string, pendingEventID string) bool {
	events, err := runner.Service.ListEvents(ctx, missionID)
	if err != nil {
		return false
	}
	_, ok := CompletedPendingEventIDs(events)[strings.TrimSpace(pendingEventID)]
	return ok
}

func (runner Runner) isSamePendingAlreadyRunning(missionID string, pendingEventID string) bool {
	if runner.InFlight == nil {
		return false
	}
	current, ok := runner.InFlight.PendingEventID(missionID)
	return ok && strings.TrimSpace(current) == strings.TrimSpace(pendingEventID)
}

func (runner Runner) id(prefix string) string {
	if runner.NewID == nil {
		return prefix + "_report"
	}
	return runner.NewID(prefix)
}

func normalizeDraftRequest(req DraftRequest) DraftRequest {
	req.Title = firstNonEmpty(req.Title, "Mission report")
	req.AgentExecutor = firstNonEmpty(req.AgentExecutor, "codex")
	req.AgentModel = strings.TrimSpace(req.AgentModel)
	req.AgentReasoningEffort = strings.TrimSpace(req.AgentReasoningEffort)
	req.MCPMode = firstNonEmpty(req.MCPMode, "auto")
	req.RigorLevel = strings.TrimSpace(req.RigorLevel)
	req.RigorLabel = strings.TrimSpace(req.RigorLabel)
	mode, err := NormalizeMode(req.ReportMode)
	if err != nil {
		mode = DefaultMode
	}
	req.ReportMode = mode
	policy, err := NormalizeSessionPolicy(req.ReportSessionPolicy)
	if err != nil {
		policy = DefaultSessionPolicy
	}
	req.ReportSessionPolicy = policy
	req.ReportSessionPolicySelection = strings.TrimSpace(req.ReportSessionPolicySelection)
	req.PostReportHumanize = normalizePostReportHumanize(req.PostReportHumanize)
	req.GenerationGuidanceProfile = normalizeGenerationGuidanceProfile(req.GenerationGuidanceProfile)
	req.GenerationGuidanceSHA256 = strings.TrimSpace(req.GenerationGuidanceSHA256)
	return req
}

func normalizePostReportHumanize(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "enabled", "enable", "true", "yes", "on", "1":
		return "enabled"
	case "", "disabled", "disable", "false", "no", "off", "0":
		return "disabled"
	default:
		return "disabled"
	}
}

func normalizeGenerationGuidanceProfile(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", "g2", "h5-g2", "substance-preserving-korean", "substance_preserving_korean":
		return "g2"
	case "none", "off", "disabled", "disable", "false", "0":
		return "none"
	default:
		return strings.TrimSpace(value)
	}
}

func normalizePatchRequest(req PatchRequest) PatchRequest {
	req.BaseArtifactID = strings.TrimSpace(req.BaseArtifactID)
	req.Instruction = strings.TrimSpace(req.Instruction)
	req.Title = firstNonEmpty(req.Title, "Patched report")
	req.AgentExecutor = firstNonEmpty(req.AgentExecutor, "codex")
	req.AgentModel = strings.TrimSpace(req.AgentModel)
	req.AgentReasoningEffort = strings.TrimSpace(req.AgentReasoningEffort)
	req.MCPMode = firstNonEmpty(req.MCPMode, "auto")
	req.ReportSessionID = strings.TrimSpace(req.ReportSessionID)
	req.PreviousAgentSessionID = firstNonEmpty(req.PreviousAgentSessionID, req.ReportSessionID)
	req.ForkSourceAgentSessionID = strings.TrimSpace(req.ForkSourceAgentSessionID)
	req.ReportSessionPolicy = strings.TrimSpace(req.ReportSessionPolicy)
	req.ReportSessionPolicySelection = strings.TrimSpace(req.ReportSessionPolicySelection)
	req.SessionChainKind = firstNonEmpty(req.SessionChainKind, "report_patch_session")
	return req
}

func normalizeHumanizeRequest(req HumanizeRequest) HumanizeRequest {
	req.SourceArtifactID = strings.TrimSpace(req.SourceArtifactID)
	req.SourceArtifactSHA256 = strings.TrimSpace(req.SourceArtifactSHA256)
	req.SourceMediaType = strings.TrimSpace(req.SourceMediaType)
	req.Title = firstNonEmpty(req.Title, "Humanized report")
	req.AgentExecutor = firstNonEmpty(req.AgentExecutor, "codex")
	req.AgentModel = strings.TrimSpace(req.AgentModel)
	req.AgentReasoningEffort = strings.TrimSpace(req.AgentReasoningEffort)
	req.MCPMode = firstNonEmpty(req.MCPMode, "auto")
	req.PreviousAgentSessionID = strings.TrimSpace(req.PreviousAgentSessionID)
	req.ToolSessionID = strings.TrimSpace(req.ToolSessionID)
	mode, err := NormalizeMode(req.ReportMode)
	if err != nil {
		mode = DefaultMode
	}
	req.ReportMode = mode
	req.ReportPendingEventID = strings.TrimSpace(req.ReportPendingEventID)
	return req
}

func pendingEventID(event app.LedgerEvent) string {
	var payload struct {
		PendingEventID string `json:"pending_event_id"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.PendingEventID)
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

func validAgentExecutorOrEmpty(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	normalized, err := app.NormalizeAgentExecutorName(value)
	if err != nil {
		return ""
	}
	return normalized
}
