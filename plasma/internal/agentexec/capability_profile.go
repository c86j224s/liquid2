package agentexec

import (
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/agentcapability"
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

var codexResearchProfileConfig = []string{
	"features.tool_suggest=false",
	"features.recommended_plugins=false",
	"tools.experimental_request_user_input.enabled=false",
	"features.apps=false",
	"features.plugins=false",
	"features.multi_agent=false",
	"features.shell_tool=false",
	"features.view_image=false",
	"tools.update_plan.enabled=false",
	"skills.include_instructions=false",
	"project_doc_max_bytes=0",
}

var codexReportILProfileConfig = append([]string(nil), codexResearchProfileConfig...)

func applyCapabilityProfile(req AgentRequest, provider string) (AgentRequest, error) {
	profile, err := agentcapability.Resolve(req.CapabilityProfile, req.ProfileRevision)
	if err != nil {
		return AgentRequest{}, err
	}
	req.CapabilityProfile = profile.ID
	req.ProfileRevision = profile.Revision

	switch profile.ID {
	case agentcapability.ProfileLegacyV1:
		return req, nil
	case agentcapability.ProfileResearchV1:
		if provider == "claude" {
			req.IgnoreUserConfig = true
		}
		if provider == "codex" {
			req.CodexConfig = appendUniqueStrings(req.CodexConfig, codexResearchProfileConfig...)
		}
		return req, nil
	case agentcapability.ProfileGoalDraftV1:
		req.DisableTools = true
		req.IgnoreUserConfig = provider == "claude"
		req.EphemeralSession = true
		req.ExtraMCPTools = nil
		req.ReplaceMCPTools = true
		if provider == "codex" {
			req.CodexConfig = appendUniqueStrings(req.CodexConfig, codexResearchProfileConfig...)
		}
		return req, nil
	case agentcapability.ProfileReportILV1:
		req.DisableTools = true
		req.IgnoreUserConfig = true
		req.EphemeralSession = true
		req.ExtraMCPTools = nil
		req.ReplaceMCPTools = true
		req.ReportILSources = nil
		if provider == "codex" {
			req.CodexConfig = appendUniqueStrings(req.CodexConfig, codexReportILProfileConfig...)
		}
		return req, nil
	case agentcapability.ProfileReportILSourceV1:
		if provider != "codex" {
			return AgentRequest{}, fmt.Errorf("report IL source profile requires codex")
		}
		if req.ReportILSources == nil {
			return AgentRequest{}, fmt.Errorf("report IL source binding is required")
		}
		if err := reportilcontract.ValidateSourceAccessBinding(*req.ReportILSources); err != nil {
			return AgentRequest{}, err
		}
		req.DisableTools = false
		req.IgnoreUserConfig = true
		req.EphemeralSession = true
		allowedTools := map[string]bool{}
		switch req.ReportILSources.Stage {
		case "il_source_selection", "il_flow":
			allowedTools[reportilcontract.SourceListTool] = true
			allowedTools[reportilcontract.SourceReadTool] = true
		case "il_editorial_memory":
			for _, name := range []string{
				reportilcontract.SourceListTool,
				reportilcontract.SourceReadTool,
				reportilcontract.SourceQuoteRegisterTool,
				reportilcontract.EditorialMemoryStartTool,
				reportilcontract.EditorialMemoryAppendTool,
				reportilcontract.EditorialMemoryReadTool,
				reportilcontract.EditorialMemoryFinalizeTool,
			} {
				allowedTools[name] = true
			}
		case "il_narrative":
			for _, name := range []string{
				reportilcontract.AuthorDocumentStartTool,
				reportilcontract.AuthorDocumentReadTool,
				reportilcontract.AuthorDocumentReplaceTool,
				reportilcontract.AuthorDocumentFinalizeTool,
			} {
				allowedTools[name] = true
			}
			if req.ReportILSources.MaxReadBytes > 0 {
				allowedTools[reportilcontract.SourceListTool] = true
				allowedTools[reportilcontract.SourceReadTool] = true
				allowedTools[reportilcontract.AuthorDocumentAppendSourceTool] = true
			} else {
				allowedTools[reportilcontract.EditorialMemoryReadTool] = true
				allowedTools[reportilcontract.AuthorDocumentAppendTool] = true
			}
		case "il_continuity":
			for _, name := range []string{
				reportilcontract.EditorialMemoryReadTool,
				reportilcontract.AuthorDocumentOpenTool,
				reportilcontract.AuthorDocumentReadTool,
				reportilcontract.AuthorDocumentReviseBlockTool,
				reportilcontract.AuthorDocumentFinalizeTool,
			} {
				allowedTools[name] = true
			}
		case "il_reader":
			for _, name := range []string{
				reportilcontract.EditorialMemoryReadTool,
				reportilcontract.AuthorDocumentOpenTool,
				reportilcontract.AuthorDocumentReadTool,
				reportilcontract.AuthorDocumentEditTextTool,
				reportilcontract.AuthorDocumentFinalizeTool,
			} {
				allowedTools[name] = true
			}
		case "il_long_form_plan":
			if req.ReportILSources.MaxReadBytes > 0 {
				allowedTools[reportilcontract.SourceListTool] = true
				allowedTools[reportilcontract.SourceReadTool] = true
			} else {
				allowedTools[reportilcontract.EditorialMemoryReadTool] = true
			}
			allowedTools[reportilcontract.LongFormPlanSubmitTool] = true
		case "il_long_form_section":
			if req.ReportILSources.MaxReadBytes > 0 {
				allowedTools[reportilcontract.SourceReadTool] = true
			} else {
				allowedTools[reportilcontract.EditorialMemoryReadTool] = true
			}
			for _, name := range []string{
				reportilcontract.LongFormPlanReadTool,
				reportilcontract.LongFormDocumentStartTool,
				reportilcontract.LongFormDocumentAppendTool,
				reportilcontract.LongFormDocumentReadTool,
				reportilcontract.LongFormDocumentReplaceTool,
				reportilcontract.LongFormDocumentCorrectBlockTool,
				reportilcontract.LongFormDocumentFinalizeTool,
			} {
				allowedTools[name] = true
			}
		case "il_long_form_part", "il_long_form_final":
			if req.ReportILSources.EditorialMemoryArtifactID != "" {
				allowedTools[reportilcontract.EditorialMemoryReadTool] = true
			}
			for _, name := range []string{
				reportilcontract.LongFormPlanReadTool,
				reportilcontract.LongFormDocumentStartTool,
				reportilcontract.LongFormDocumentReadTool,
				reportilcontract.LongFormDocumentReplaceTool,
				reportilcontract.LongFormDocumentFinalizeTool,
			} {
				allowedTools[name] = true
			}
		}
		for _, name := range req.ExtraMCPTools {
			if !allowedTools[name] {
				return AgentRequest{}, fmt.Errorf("report IL source profile tool is unavailable")
			}
		}
		if len(req.ExtraMCPTools) == 0 {
			return AgentRequest{}, fmt.Errorf("report IL source profile requires explicit stage tools")
		}
		req.ReplaceMCPTools = true
		req.CodexConfig = appendUniqueStrings(req.CodexConfig, codexReportILProfileConfig...)
		return req, nil
	case agentcapability.ProfileReportUnverifiedV1:
		if provider != "codex" {
			return AgentRequest{}, fmt.Errorf("unverified report profile requires codex")
		}
		req.DisableTools = false
		req.IgnoreUserConfig = true
		req.EphemeralSession = true
		req.PreserveResponseWhitespace = true
		req.ExtraMCPTools = []string{mcptools.ToolSourcesList, mcptools.ToolSourcesRead}
		req.ReplaceMCPTools = true
		req.ReportILSources = nil
		req.CodexConfig = appendUniqueStrings(req.CodexConfig, codexResearchProfileConfig...)
		return req, nil
	default:
		return AgentRequest{}, fmt.Errorf("unsupported agent capability profile %q", profile.ID)
	}
}

func appendUniqueStrings(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values)+len(additions))
	result := make([]string, 0, len(values)+len(additions))
	for _, value := range append(append([]string(nil), values...), additions...) {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
