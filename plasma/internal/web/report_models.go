package web

import (
	"errors"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

var errReportDraftRunning = errors.New("report draft is already running for this mission")

const defaultReportRigorLevel = "balanced"

const (
	defaultReportMode   = reporting.DefaultMode
	reportModeOneTake   = reporting.ModeOneTake
	reportModePlanned   = reporting.ModePlanned
	reportModeLongForm  = reporting.ModeLongForm
	reportModeLabelFast = "원테이크 보고서"
	reportModeLabelPlan = "보고서"
	reportModeLabelLong = "장문 보고서"

	reportExecutionStrategySerial        = "serial"
	reportExecutionStrategySectionFanout = "section_fanout"

	reportSessionPolicySameSession  = reporting.SessionPolicySameSession
	reportSessionPolicyIsolatedFork = reporting.SessionPolicyIsolatedFork

	reportSessionPolicySelectionAutoIsolatedFork        = reporting.SessionPolicySelectionAutoIsolatedFork
	reportSessionPolicySelectionAutoSameSessionNoForker = reporting.SessionPolicySelectionAutoSameSessionNoForker
	reportSessionPolicySelectionAutoSameSessionOneTake  = reporting.SessionPolicySelectionAutoSameSessionOneTake
)

type reportRigorProfile struct {
	level        string
	label        string
	description  string
	instructions string
}

type reportHumanizeRequest struct {
	Title                string `json:"title"`
	AgentExecutor        string `json:"agent_executor"`
	AgentModel           string `json:"agent_model"`
	AgentReasoningEffort string `json:"agent_reasoning_effort"`
	MCPMode              string `json:"mcp_mode"`
}

var reportRigorProfiles = map[string]reportRigorProfile{
	"exploratory": {
		level:       "exploratory",
		label:       "탐색적",
		description: "약한 신호도 넓게 활용하되 추정, 반응, 루머, 논쟁, 해석임을 명확히 밝힙니다.",
		instructions: strings.Join([]string{
			"- Use a broad evidence lens. Include useful interpretations, reactions, rumors, controversies, market signals, open questions, code, formulas, and benchmarks when they help the mission.",
			"- Weak or low-confidence material may enrich the article, but it must be explicitly labeled in prose as rumor, reaction, interpretation, unresolved question, or low-confidence signal.",
			"- Do not turn exploratory signals into confirmed facts. Use cautious wording such as \"~라는 해석이 있다\", \"확인되지는 않았지만\", \"반응 신호로는\".",
			"- Prefer richness and coverage over premature pruning, while preserving source references for claims that depend on saved evidence.",
		}, "\n"),
	},
	"balanced": {
		level:       "balanced",
		label:       "균형형",
		description: "검증된 사실을 중심에 두고, 유용한 약한 신호는 맥락과 한계를 붙여 사용합니다.",
		instructions: strings.Join([]string{
			"- Anchor the main storyline on source-backed facts and medium/high-confidence claims.",
			"- Include interpretations, reactions, rumors, controversies, market signals, open questions, code, formulas, and benchmarks when they materially improve understanding.",
			"- Keep weak signals out of the main conclusion unless clearly caveated. Present them as context, competing accounts, or unresolved uncertainty.",
			"- When evidence conflicts, explain the competing accounts and what would be needed to resolve them.",
		}, "\n"),
	},
	"strict": {
		level:       "strict",
		label:       "검증형",
		description: "주요 결론은 강한 근거 중심으로 쓰고, 약한 신호는 배경 또는 미확인 사항으로만 둡니다.",
		instructions: strings.Join([]string{
			"- Base major conclusions only on source-backed facts and medium/high-confidence evidence or claims.",
			"- Do not use rumors, low-confidence reactions, or unsupported interpretations as decision-grade support.",
			"- You may mention weak signals only when they explain uncertainty, public discourse, risk, or a gap; label them clearly as weak, unverified, contested, or contextual.",
			"- Make missing verification visible instead of filling gaps with confident prose.",
		}, "\n"),
	},
}

type sectionalMarkdownNormalization struct {
	DropFirstLeadingHeadingTexts []string
	ConvertHeadingsBold          bool
	ForceHeadingLevel            int
	MaxHeadingLevel              int
	StripBoundaryRules           bool
}

type reportRefViolation struct {
	ObjectKind string `json:"object_kind"`
	ObjectID   string `json:"object_id"`
	State      string `json:"state"`
	Reason     string `json:"reason"`
	BlockIndex int    `json:"block_index"`
	BlockType  string `json:"block_type"`
}

type reportDraftRequest struct {
	Title                        string `json:"title"`
	DirectionHint                string `json:"direction_hint"`
	ExecutionStrategy            string `json:"execution_strategy"`
	AgentExecutor                string `json:"agent_executor"`
	AgentModel                   string `json:"agent_model"`
	AgentReasoningEffort         string `json:"agent_reasoning_effort"`
	AgentSelectionSource         string `json:"agent_selection_source"`
	MCPMode                      string `json:"mcp_mode"`
	RigorLevel                   string `json:"rigor_level"`
	ReportMode                   string `json:"report_mode"`
	ReportSessionPolicy          string `json:"report_session_policy"`
	ReportSessionPolicySelection string `json:"report_session_policy_selection"`
	PostReportHumanize           string `json:"post_report_humanize"`
	GenerationGuidanceProfile    string `json:"generation_guidance_profile"`
	GenerationGuidanceSHA256     string `json:"generation_guidance_sha256"`
}

type reportPatchRequest struct {
	BaseArtifactID       string `json:"base_artifact_id"`
	Instruction          string `json:"instruction"`
	Title                string `json:"title"`
	AgentExecutor        string `json:"agent_executor"`
	AgentModel           string `json:"agent_model"`
	AgentReasoningEffort string `json:"agent_reasoning_effort"`
	MCPMode              string `json:"mcp_mode"`
	ReportSessionPolicy  string `json:"report_session_policy"`
}

type reportRetryRequest struct {
	FailedPendingEventID string `json:"failed_pending_event_id"`
	Strategy             string `json:"strategy"`
	RetryRequestID       string `json:"retry_request_id"`
}

type reportExportRequest struct {
	Target string `json:"target"`
}

type agentReportPlan = reporting.ReportPlan
type agentSectionalReportPlan = reporting.SectionalReportPlan
type agentReportPart = reporting.ReportPlanPart
type agentReportSection = reporting.ReportPlanSection

type sectionalReportDraft struct {
	Title      string
	Markdown   string
	ArtifactID string
	WordCount  int
}

type sectionalReportPartDraft struct {
	Title      string
	Markdown   string
	ArtifactID string
	WordCount  int
}

type agentPartAssembly = reporting.PartAssembly
type agentPartTransition = reporting.PartTransition

type agentReportAST struct {
	Title   string             `json:"title"`
	Summary string             `json:"summary"`
	Blocks  []agentReportBlock `json:"blocks"`
}

type agentReportBlock struct {
	Type       string                    `json:"type"`
	Level      int                       `json:"level,omitempty"`
	Text       string                    `json:"text,omitempty"`
	Items      []string                  `json:"items,omitempty"`
	SourceRefs app.ReportBlockSourceRefs `json:"source_refs,omitempty"`
	Refs       app.ReportBlockSourceRefs `json:"refs,omitempty"`
}
