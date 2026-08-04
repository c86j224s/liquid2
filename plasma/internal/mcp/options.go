package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/sourceretrieval"
)

// Binding은 MCP server instance가 묶인 현재 미션과 agent session의 실행 맥락이다.
//
// tool handler는 요청 argument보다 Binding을 우선해 미션/session 경계를 확인해야
// 한다. 이 값이 비어 있으면 미션에 귀속되는 tool은 노출되거나 실행되면 안 된다.
type Binding struct {
	MissionID          string
	AgentSessionID     string
	CurrentUserEventID string
	AgentExecutor      string
}

// ReportPatchBinding은 기존 report artifact를 MCP patch tool로 수정할 때 필요한
// session lineage와 pending event 경계다.
//
// base artifact와 pending event가 일치해야 patch 결과를 올바른 report generation
// 시도에 귀속시킬 수 있다.
type ReportPatchBinding struct {
	BaseArtifactID               string
	PendingEventID               string
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	MCPMode                      string
	AgentSessionID               string
	PreviousAgentSessionID       string
	ReturnedAgentSessionID       string
	ReportSessionID              string
	ForkSourceAgentSessionID     string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	SessionChainKind             string
}

// ReportPlanBinding은 report planning session이 계획 결과를 장부에 제출할 때의
// 고정 실행 계약이다.
//
// ToolSessionID와 IdempotencyKey는 같은 계획 제출이 중복 기록되지 않도록 caller와
// MCP server 사이에서 공유되는 값이다.
type ReportPlanBinding struct {
	PendingEventID            string
	ReportMode                string
	IdempotencyKey            string
	ToolSessionID             string
	PreviousProviderSessionID string
	AgentExecutor             string
	AgentModel                string
	AgentReasoningEffort      string
	RequireWritingContract    bool
}

type idempotencyEntry struct {
	ArgumentsHash string
	Result        ToolResult
}

// Option은 Server 생성 시 tool surface와 binding을 구성하는 함수다.
type Option func(*Server)

// SourceCandidateFetcher는 URL source 후보 staging에서 쓰는 bounded fetch port다.
type SourceCandidateFetcher func(context.Context, string) (sourceretrieval.Fetched, error)

// ConfluenceConnectorFactory는 tool 호출 시점의 Confluence connection 정보를
// connector adapter로 바꾸는 factory다.
type ConfluenceConnectorFactory func(context.Context, ConfluenceConnectorRequest) (app.ConfluenceSourceConnector, error)

// ConfluenceConnectorRequest는 MCP 요청에서 선택된 Confluence connection 범위를
// factory에 전달하는 값이다.
type ConfluenceConnectorRequest struct {
	ConnectionID string
	CloudID      string
	SpaceKey     string
}

// WithLiquid2Connector는 Liquid2 source search/read tool에 사용할 connector를 등록한다.
func WithLiquid2Connector(connector app.Liquid2SourceConnector) Option {
	return func(server *Server) {
		if connector != nil {
			server.connectors[app.Liquid2ConnectorID] = connector
		}
	}
}

// WithConfluenceConnectorFactory는 Confluence source tool이 connection별 connector를
// 만들 수 있게 한다.
func WithConfluenceConnectorFactory(factory ConfluenceConnectorFactory) Option {
	return func(server *Server) {
		server.confluenceConnectorFactory = factory
	}
}

// WithBinding은 Server의 미션/session binding을 정규화해 저장한다.
func WithBinding(binding Binding) Option {
	return func(server *Server) {
		server.binding = Binding{
			MissionID:          strings.TrimSpace(binding.MissionID),
			AgentSessionID:     strings.TrimSpace(binding.AgentSessionID),
			CurrentUserEventID: strings.TrimSpace(binding.CurrentUserEventID),
			AgentExecutor:      strings.TrimSpace(strings.ToLower(binding.AgentExecutor)),
		}
	}
}

// WithLegacyResearchLoop는 과거 evidence/claim 중심 research tool surface를 켠다.
func WithLegacyResearchLoop() Option {
	return func(server *Server) {
		server.legacyResearchLoop = true
	}
}

// WithExperimentalReportComposition은 실험용 report composition tool surface를 켠다.
func WithExperimentalReportComposition() Option {
	return func(server *Server) {
		server.experimentalReportComposition = true
	}
}

// WithOperatorSourceMutation은 operator 전용 source attach/remove/restore tool을 켠다.
func WithOperatorSourceMutation() Option {
	return func(server *Server) {
		server.operatorSourceMutation = true
	}
}

// WithReportPatch는 Markdown report patch tool surface를 켠다.
func WithReportPatch() Option {
	return func(server *Server) {
		server.reportPatch = true
	}
}

// WithReportPatchBinding은 report patch tool이 사용할 base artifact와 session 경계를
// 정규화해 저장한다.
func WithReportPatchBinding(binding ReportPatchBinding) Option {
	return func(server *Server) {
		server.reportPatchBinding = normalizeReportPatchBinding(binding)
	}
}

// WithReportPlanBinding은 report plan submit tool의 session/idempotency 경계를 저장한다.
func WithReportPlanBinding(binding ReportPlanBinding) Option {
	return func(server *Server) {
		server.reportPlanBinding = normalizeReportPlanBinding(binding)
	}
}

// WithReportRequirementMapBinding은 requirement mapping tool의 runner binding을 저장한다.
func WithReportRequirementMapBinding(binding reporting.ReportRequirementMapBinding) Option {
	return func(server *Server) {
		binding.MissionID = strings.TrimSpace(binding.MissionID)
		binding.PendingEventID = strings.TrimSpace(binding.PendingEventID)
		binding.PlanEventID = strings.TrimSpace(binding.PlanEventID)
		binding.ToolSessionID = strings.TrimSpace(binding.ToolSessionID)
		binding.PreviousProviderSessionID = strings.TrimSpace(binding.PreviousProviderSessionID)
		binding.IdempotencyKey = strings.TrimSpace(binding.IdempotencyKey)
		binding.AgentExecutor = strings.TrimSpace(strings.ToLower(binding.AgentExecutor))
		binding.AgentModel = strings.TrimSpace(binding.AgentModel)
		binding.AgentReasoningEffort = strings.TrimSpace(binding.AgentReasoningEffort)
		binding.Producer.Type = strings.TrimSpace(binding.Producer.Type)
		binding.Producer.ID = strings.TrimSpace(binding.Producer.ID)
		server.reportRequirementMapBinding = binding
	}
}

// WithLongFormFinalizeBinding은 long-form finalizer tool이 assemble할 저장된 part
// 집합을 지정한다.
func WithLongFormFinalizeBinding(binding reporting.LongFormFinalizeBinding) Option {
	return func(server *Server) {
		server.longFormFinalizeBinding = binding
		server.longFormFinalizeBindingSet = true
	}
}

// WithFinalEditStageBinding은 final edit stage tool surface를 특정 writer/reader/style
// 단계에 묶는다.
func WithFinalEditStageBinding(binding reporting.FinalEditStageBinding) Option {
	return func(server *Server) {
		server.finalEditStageBinding = binding
		server.finalEditStageBindingSet = true
	}
}

// WithPartAssemblyBinding은 part assembly tool을 runner가 지정한 part 입력에 묶는다.
func WithPartAssemblyBinding(binding reporting.PartAssemblyBinding) Option {
	return func(server *Server) { server.partAssemblyBinding = binding }
}

// WithPartEditBinding은 part edit tool을 runner가 지정한 assembled part에 묶는다.
func WithPartEditBinding(binding reporting.PartEditBinding) Option {
	return func(server *Server) { server.partEditBinding = binding }
}

// ValidatePartAssemblyBinding은 MCP binding과 part assembly binding이 같은 session
// 경계를 가리키는지 확인한다.
func ValidatePartAssemblyBinding(binding Binding, part reporting.PartAssemblyBinding) error {
	if err := reporting.ValidatePartAssemblyBinding(part); err != nil {
		return fmt.Errorf("part assembly binding is incomplete: %w", err)
	}
	if strings.TrimSpace(part.MissionID) != strings.TrimSpace(binding.MissionID) || strings.TrimSpace(part.ToolSessionID) != strings.TrimSpace(binding.AgentSessionID) || strings.TrimSpace(strings.ToLower(part.AgentExecutor)) != strings.TrimSpace(strings.ToLower(binding.AgentExecutor)) {
		return fmt.Errorf("part assembly binding conflicts with MCP binding")
	}
	return nil
}

// ValidatePartEditBinding은 MCP binding과 part edit binding이 같은 session 경계를
// 가리키는지 확인한다.
func ValidatePartEditBinding(binding Binding, part reporting.PartEditBinding) error {
	if err := reporting.ValidatePartEditBinding(part); err != nil {
		return fmt.Errorf("part edit binding is incomplete: %w", err)
	}
	if strings.TrimSpace(part.MissionID) != strings.TrimSpace(binding.MissionID) || strings.TrimSpace(part.ToolSessionID) != strings.TrimSpace(binding.AgentSessionID) || strings.TrimSpace(strings.ToLower(part.AgentExecutor)) != strings.TrimSpace(strings.ToLower(binding.AgentExecutor)) {
		return fmt.Errorf("part edit binding conflicts with MCP binding")
	}
	return nil
}

// ValidateLongFormFinalizeBinding은 MCP binding과 finalization binding이 같은
// session 경계를 가리키는지 확인한다.
func ValidateLongFormFinalizeBinding(binding Binding, final reporting.LongFormFinalizeBinding) error {
	if err := reporting.ValidateLongFormFinalizeBinding(final); err != nil {
		return fmt.Errorf("long-form finalization binding is incomplete: %w", err)
	}
	if strings.TrimSpace(final.MissionID) != strings.TrimSpace(binding.MissionID) || strings.TrimSpace(final.ToolSessionID) != strings.TrimSpace(binding.AgentSessionID) || strings.TrimSpace(strings.ToLower(final.AgentExecutor)) != strings.TrimSpace(strings.ToLower(binding.AgentExecutor)) {
		return fmt.Errorf("long-form finalization binding conflicts with MCP binding")
	}
	return nil
}

// ValidateFinalEditStageBinding은 MCP binding과 final edit stage binding이 같은
// session 경계를 가리키는지 확인한다.
func ValidateFinalEditStageBinding(binding Binding, stage reporting.FinalEditStageBinding) error {
	if err := reporting.ValidateFinalEditStageBinding(stage); err != nil {
		return fmt.Errorf("final edit stage binding is incomplete: %w", err)
	}
	if strings.TrimSpace(stage.MissionID) != strings.TrimSpace(binding.MissionID) || strings.TrimSpace(stage.ToolSessionID) != strings.TrimSpace(binding.AgentSessionID) || strings.TrimSpace(strings.ToLower(stage.AgentExecutor)) != strings.TrimSpace(strings.ToLower(binding.AgentExecutor)) {
		return fmt.Errorf("final edit stage binding conflicts with MCP binding")
	}
	return nil
}

func (server *Server) validateFinalEditConfiguration() error {
	stageProvided := server.finalEditStageBindingSet
	finalProvided := server.longFormFinalizeBindingSet
	if !stageProvided {
		return nil
	}
	if err := ValidateFinalEditStageBinding(server.binding, server.finalEditStageBinding); err != nil {
		return err
	}
	switch strings.TrimSpace(server.finalEditStageBinding.Stage) {
	case reporting.FinalEditStageWriter:
		if finalProvided {
			return fmt.Errorf("%w: final writer final edit stage MCP server must not carry a final binding", app.ErrInvalidInput)
		}
	case reporting.FinalEditStageReader, reporting.FinalEditStageStyle, reporting.FinalEditStageStyleSemanticValidation:
		if finalProvided {
			return fmt.Errorf("%w: reader/style validation final edit stage MCP servers must not carry a final binding", app.ErrInvalidInput)
		}
	case reporting.FinalEditStageGate, reporting.FinalEditStageEvidenceGate:
		if !finalProvided {
			return fmt.Errorf("%w: final edit gate MCP server requires a final binding", app.ErrInvalidInput)
		}
		if err := ValidateLongFormFinalizeBinding(server.binding, server.longFormFinalizeBinding); err != nil {
			return err
		}
		if err := reporting.ValidateFinalEditGateBindingsCompatible(server.finalEditStageBinding, server.longFormFinalizeBinding); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: unsupported final edit stage", app.ErrInvalidInput)
	}
	return nil
}

func normalizeReportPlanBinding(binding ReportPlanBinding) ReportPlanBinding {
	return ReportPlanBinding{
		PendingEventID: strings.TrimSpace(binding.PendingEventID), ReportMode: strings.TrimSpace(binding.ReportMode),
		IdempotencyKey: strings.TrimSpace(binding.IdempotencyKey), ToolSessionID: strings.TrimSpace(binding.ToolSessionID),
		PreviousProviderSessionID: strings.TrimSpace(binding.PreviousProviderSessionID), AgentExecutor: strings.TrimSpace(strings.ToLower(binding.AgentExecutor)),
		AgentModel: strings.TrimSpace(binding.AgentModel), AgentReasoningEffort: strings.TrimSpace(binding.AgentReasoningEffort),
		RequireWritingContract: binding.RequireWritingContract,
	}
}

func (binding ReportPlanBinding) complete() bool {
	return binding.PendingEventID != "" && (binding.ReportMode == "planned" || binding.ReportMode == "long_form") && binding.IdempotencyKey != "" && binding.ToolSessionID != "" && binding.AgentExecutor != ""
}

// ValidateReportPlanBinding은 report planning tool이 현재 MCP session에서만 제출되게
// binding 충돌을 검사한다.
func ValidateReportPlanBinding(binding Binding, plan ReportPlanBinding) error {
	plan = normalizeReportPlanBinding(plan)
	if !plan.complete() {
		return fmt.Errorf("report plan binding is incomplete")
	}
	if plan.ToolSessionID != strings.TrimSpace(binding.AgentSessionID) || plan.AgentExecutor != strings.TrimSpace(strings.ToLower(binding.AgentExecutor)) {
		return fmt.Errorf("report plan binding conflicts with MCP binding")
	}
	return nil
}

// ValidateReportRequirementMapBinding은 requirement mapping tool의 runner binding과
// MCP binding이 같은 실행을 가리키는지 확인한다.
func ValidateReportRequirementMapBinding(binding Binding, requirements reporting.ReportRequirementMapBinding) error {
	if err := reporting.ValidateReportRequirementMapBinding(requirements); err != nil {
		return fmt.Errorf("report requirement map binding is incomplete: %w", err)
	}
	if strings.TrimSpace(requirements.MissionID) != strings.TrimSpace(binding.MissionID) || strings.TrimSpace(requirements.ToolSessionID) != strings.TrimSpace(binding.AgentSessionID) || strings.TrimSpace(strings.ToLower(requirements.AgentExecutor)) != strings.TrimSpace(strings.ToLower(binding.AgentExecutor)) {
		return fmt.Errorf("report requirement map binding conflicts with MCP binding")
	}
	return nil
}

// WithEnabledTools는 embedding caller가 노출할 MCP tool 목록을 allowlist 방식으로
// 제한한다.
func WithEnabledTools(tools []string) Option {
	return func(server *Server) {
		enabled := map[string]struct{}{}
		for _, tool := range tools {
			tool = strings.TrimSpace(tool)
			if tool != "" {
				enabled[tool] = struct{}{}
			}
		}
		if len(enabled) > 0 {
			server.enabledTools = enabled
		}
	}
}

// WithSourceCandidateFetcher는 URL source 후보 staging에서 사용할 fetch 구현을 주입한다.
func WithSourceCandidateFetcher(fetcher SourceCandidateFetcher) Option {
	return func(server *Server) {
		server.sourceCandidateFetcher = fetcher
	}
}
