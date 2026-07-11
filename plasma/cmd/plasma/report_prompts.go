package main

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/web"
)

func cliReportPlanPrompt(title string, missionID string, toolSessionID string) string {
	return fmt.Sprintf(`You are planning a Plasma Markdown report.

Do not write the report yet. Use Plasma read tools when useful, starting with plasma.research.outline.

Mission ID: %s
Report title: %s
Tool session ID: %s

Rules:
- Sources are original materials. Prior answers and reports are results, not sources.
- Do not create evidence, claims, confidence updates, source candidates, proposal bundles, report blocks, or report AST JSON.
- Return a concise Korean generation plan as Markdown bullets.`, strings.TrimSpace(missionID), strings.TrimSpace(title), strings.TrimSpace(toolSessionID))
}

func cliReportGenerationGuidanceSelection(profile string) (string, string, error) {
	return web.SelectReportGenerationGuidance(profile)
}

func cliPostReportHumanizeFlag(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func cliNormalizePostReportHumanize(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "enabled", "enable", "true", "yes", "on", "1":
		return "enabled"
	case "", "disabled", "disable", "false", "no", "off", "0":
		return "disabled"
	default:
		return "disabled"
	}
}

func cliRequestedReportSessionPolicy(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", "auto", "automatic":
		return ""
	default:
		return strings.TrimSpace(value)
	}
}

func cliReportGenerationGuidance(profile string) string {
	return web.ReportGenerationGuidance(profile)
}

func cliReportPrompt(title string, missionID string, toolSessionID string, reportMode string, planEventID string, generationGuidanceProfile string) string {
	modeLine := "This is an explicit one-take compatibility report. Do not create a separate plan."
	if reportMode != reporting.ModeOneTake {
		modeLine = fmt.Sprintf("Follow the auto-accepted generation plan recorded as %s. Re-read original source material before writing; do not treat the plan itself as a source.", strings.TrimSpace(planEventID))
	}
	guidance := strings.TrimSpace(cliReportGenerationGuidance(generationGuidanceProfile))
	if guidance != "" {
		guidance = "\n" + guidance + "\n"
	}
	return fmt.Sprintf(`You are writing a Plasma report as a Markdown artifact.

Write the public-facing report in Korean Markdown.

Mission ID: %s
Report title: %s
Tool session ID: %s
Report mode: %s
%s
%s

Rules:
- Use MCP/source read tools to inspect original materials. Do not assume source bodies are present in this prompt.
- Start with plasma.research.outline, then use plasma.research.list, plasma.research.grep, plasma.research.read, plasma.sources.read, plasma.sources.tree, plasma.sources.grep, and plasma.research.references as needed.
- PDF sources are original documents; read them through Plasma tools, which return extracted text and metadata rather than raw PDF bytes.
- For live_reference local_path sources, use explicit source observations and cite observation_event_id, observed_at, relative_path, sha256, and git metadata when available.
- Sources are original materials. This Markdown report is an output artifact, not a source.
- Do not create evidence, claims, confidence updates, proposal bundles, report blocks, or report AST JSON.
- Return only the Markdown report body.`, strings.TrimSpace(missionID), strings.TrimSpace(title), strings.TrimSpace(toolSessionID), strings.TrimSpace(reportMode), modeLine, guidance)
}

func cliReportCompositionStrategy(reportMode string) string {
	switch reportMode {
	case reporting.ModeOneTake:
		return "cli_one_take"
	default:
		return "cli_planned_markdown"
	}
}

func cliValidatedSessionID(returnedSessionID string, previousSessionID string) (string, error) {
	returnedSessionID = strings.TrimSpace(returnedSessionID)
	previousSessionID = strings.TrimSpace(previousSessionID)
	if previousSessionID == "" {
		if returnedSessionID == "" {
			return "", fmt.Errorf("%w: agent did not return a session id", app.ErrInvalidInput)
		}
		return returnedSessionID, nil
	}
	if returnedSessionID == "" {
		return previousSessionID, nil
	}
	if returnedSessionID != previousSessionID {
		return "", fmt.Errorf("agent returned a different session id")
	}
	return returnedSessionID, nil
}

func safeCLIReportFilename(title string) string {
	base := strings.ToLower(strings.TrimSpace(title))
	if base == "" {
		base = "mission-report"
	}
	var builder strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-' || r == '_':
			builder.WriteRune(r)
		case r == ' ' || r == '.':
			builder.WriteRune('-')
		}
	}
	clean := strings.Trim(builder.String(), "-_")
	if clean == "" {
		clean = "mission-report"
	}
	return clean + ".md"
}
