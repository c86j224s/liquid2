package reportexecution

import (
	"strings"
)

func normalizeDraftRequest(req DraftRequest) DraftRequest {
	req.DirectionHint = NormalizeDirectionHint(req.DirectionHint)
	req.ExecutionStrategy = strings.TrimSpace(strings.ToLower(req.ExecutionStrategy))
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
