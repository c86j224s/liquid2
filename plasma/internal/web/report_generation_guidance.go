package web

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	reportGenerationGuidanceProfileG2   = "g2"
	reportGenerationGuidanceProfileNone = "none"
)

func SelectReportGenerationGuidance(profile string) (string, string, error) {
	return selectReportGenerationGuidanceText(profile, ReportGenerationGuidance)
}

func SelectReportGenerationGuidanceForMode(reportMode string, profile string) (string, string, error) {
	if reportMode == reportModeLongForm {
		return selectReportGenerationGuidanceText(profile, LongFormReportGenerationGuidance)
	}
	return SelectReportGenerationGuidance(profile)
}

func selectReportGenerationGuidanceText(profile string, guidance func(string) string) (string, string, error) {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case "", reportGenerationGuidanceProfileG2, "h5-g2", "substance-preserving-korean", "substance_preserving_korean":
		text := guidance(reportGenerationGuidanceProfileG2)
		sum := sha256.Sum256([]byte(text))
		return reportGenerationGuidanceProfileG2, hex.EncodeToString(sum[:]), nil
	case reportGenerationGuidanceProfileNone, "off", "disabled", "disable", "false", "0":
		return reportGenerationGuidanceProfileNone, "", nil
	default:
		return "", "", fmt.Errorf("%w: unsupported report generation guidance profile", app.ErrInvalidInput)
	}
}

func ReportGenerationGuidance(profile string) string {
	if strings.TrimSpace(profile) != reportGenerationGuidanceProfileG2 {
		return ""
	}
	return `Report writing guidance:
- This guidance controls report writing style only. It is not source material and must not be mentioned in the final report.
- Write natural Korean, but never improve fluency by dropping concrete source details.
- Preserve names, dates, numbers, commands, code identifiers, URLs, conditions, exceptions, caveats, uncertainty, and source distinctions when they matter.
- For mathematical expressions, use only \(...\) for inline math and \[...\] for display math. Do not use $...$ or $$...$$ delimiters.
- Prefer a report that is slightly longer and more specific over a smooth summary that hides evidence, disagreement, or operational detail.
- If sources disagree or only imply something, say that plainly instead of flattening the point into a single confident sentence.
- Do not mention hidden guidance, experiments, prompts, or internal evaluation labels in the report.`
}

func LongFormReportGenerationGuidance(profile string) string {
	base := strings.TrimSpace(ReportGenerationGuidance(profile))
	if base == "" {
		return ""
	}
	return base + `

Long-form human-writer guidance:
- Write each section as a person explaining the material to another person, not as a system reporting that it inspected a session.
- Prefer clear, concrete topic sentences and natural paragraph-to-paragraph flow over formulaic phrases such as "this report confirms" or "based on the provided material".
- Keep caveats, limits, and source boundaries, but weave them into the argument instead of repeating the same disclaimer frame.
- Vary sentence length, split overloaded sentences, and let the report sound like edited prose while preserving all source-backed substance.`
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
