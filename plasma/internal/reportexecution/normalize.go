package reportexecution

import (
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportpipeline"
)

// NormalizeDraftRequest applies the request-local defaults and the independent
// report IL server contract. Callers may use it before dispatch or recovery;
// it never resolves classic model, session, or capability policy.
func NormalizeDraftRequest(req DraftRequest) DraftRequest {
	req.DirectionHint = NormalizeDirectionHint(req.DirectionHint)
	req.Title = firstNonEmpty(req.Title, "Mission report")
	req.AgentSelectionSource = strings.TrimSpace(req.AgentSelectionSource)
	if family, familyErr := NormalizePipelineFamily(req.PipelineFamily); familyErr == nil {
		req.PipelineFamily = family
	} else {
		req.PipelineFamily = strings.TrimSpace(req.PipelineFamily)
	}
	req.ExecutionStrategy = strings.TrimSpace(strings.ToLower(req.ExecutionStrategy))
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
	req.RetryStrategy = strings.TrimSpace(req.RetryStrategy)
	req.RetryOfPendingEventID = strings.TrimSpace(req.RetryOfPendingEventID)
	req.ResumeStage = strings.TrimSpace(req.ResumeStage)

	switch req.PipelineFamily {
	case reportpipeline.ExperimentalIL:
		req.PipelineGraph = normalizePipelineGraph(req.PipelineFamily, req.PipelineGraph)
		req.ExecutionStrategy = ""
		if req.ReportMode != ModeLongForm {
			req.ReportMode = ModePlanned
		}
		req.AgentExecutor = "codex"
		req.AgentModel = "gpt-5.6-luna"
		req.AgentReasoningEffort = "xhigh"
		req.AgentSelectionSource = "experimental_fixed"
		req.MCPMode = "source_read_only"
		req.RigorLevel, req.RigorLabel = normalizeILValidationProfile(req.RigorLevel)
		req.ReportSessionPolicy = SessionPolicyFreshSession
		req.ReportSessionPolicySelection = "experimental_fixed"
		req.PostReportHumanize = "disabled"
		req.GenerationGuidanceProfile = ""
		req.GenerationGuidanceSHA256 = ""
	case reportpipeline.Unverified:
		req.PipelineGraph = ""
		req.ExecutionStrategy = ""
		req.ReportMode = ModePlanned
		req.AgentExecutor = "codex"
		req.AgentModel = "gpt-5.6-luna"
		req.AgentReasoningEffort = "xhigh"
		req.AgentSelectionSource = "unverified_fixed"
		req.MCPMode = "source_read_only"
		req.RigorLevel = "unverified"
		req.RigorLabel = "무검증형"
		req.ReportSessionPolicy = SessionPolicyFreshSession
		req.ReportSessionPolicySelection = "unverified_fixed"
		req.PostReportHumanize = "disabled"
		req.GenerationGuidanceProfile = ""
		req.GenerationGuidanceSHA256 = ""
	default:
		req.PipelineGraph = ""
	}
	return req
}

func normalizeDraftRequest(req DraftRequest) DraftRequest {
	return NormalizeDraftRequest(req)
}

func normalizePipelineGraph(family, graph string) string {
	if strings.TrimSpace(family) != reportpipeline.ExperimentalIL {
		return ""
	}
	switch strings.TrimSpace(graph) {
	case reportpipeline.ExperimentalILFlowGraph:
		return reportpipeline.ExperimentalILFlowGraph
	case reportpipeline.ExperimentalILReaderGraph:
		return reportpipeline.ExperimentalILReaderGraph
	case reportpipeline.ExperimentalILEditorialGraph:
		return reportpipeline.ExperimentalILEditorialGraph
	case reportpipeline.ExperimentalILEditorialMemoryGraph:
		return reportpipeline.ExperimentalILEditorialMemoryGraph
	default:
		return reportpipeline.ExperimentalILValidationProfilesGraph
	}
}

func normalizeILValidationProfile(value string) (string, string) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "unverified":
		return "unverified", "무검증"
	case "exploratory":
		return "exploratory", "탐색형"
	default:
		return "strict", "검증형"
	}
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
