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
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case "", reportGenerationGuidanceProfileG2, "h5-g2", "substance-preserving-korean", "substance_preserving_korean":
		text := ReportGenerationGuidance(reportGenerationGuidanceProfileG2)
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
- Prefer a report that is slightly longer and more specific over a smooth summary that hides evidence, disagreement, or operational detail.
- If sources disagree or only imply something, say that plainly instead of flattening the point into a single confident sentence.
- Do not mention hidden guidance, experiments, prompts, or internal evaluation labels in the report.`
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
