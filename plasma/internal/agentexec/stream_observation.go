package agentexec

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	observationURIPattern              = regexp.MustCompile(`(?i)\b(?:[a-z][a-z0-9+.-]{1,}:(?://)?|www\.)[^\s<>"')\]}]+`)
	observationDoubleQuotedPathPattern = regexp.MustCompile(`"(?i:(?:/|~/|[A-Za-z]:[\\/]|\.{1,2}[\\/]|[^"\r\n\s]+[\\/])[^"\r\n]*)"`)
	observationSingleQuotedPathPattern = regexp.MustCompile(`'(?i:(?:/|~/|[A-Za-z]:[\\/]|\.{1,2}[\\/]|[^'\r\n\s]+[\\/])[^'\r\n]*)'`)
	observationPathTokenPattern        = regexp.MustCompile(`[^\s<>"')\]}]*[\\/][^\s<>"')\]}]+`)
	observationUUIDPattern             = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	observationLabeledIDPattern        = regexp.MustCompile(`(?i)\b((?:session|event|run|mission|artifact|source|request|call|item|user|tool_use|toolu|tool|thread|message|msg|response|resp|job|token)(?:[\s_-]id)?\s*[:=]\s*)["']?[A-Za-z0-9][A-Za-z0-9_.:-]{5,}\b`)
	observationOpaqueIDPattern         = regexp.MustCompile(`(?i)\b(?:session|event|run|mission|artifact|source|request|call|item|user|toolu|tool|thread|msg|message|resp|response|job|token|ses|evt|mis|art|src|req)_[A-Za-z0-9][A-Za-z0-9_-]{5,}\b`)
)

func observePhase(observer AgentObserver, phase AgentPhase) {
	if observer == nil || phase == "" {
		return
	}
	observer(AgentObservation{Type: AgentObservationPhase, Phase: phase})
}

func observeTool(observer AgentObserver, category AgentToolCategory) {
	if observer == nil || category == "" {
		return
	}
	observer(AgentObservation{Type: AgentObservationTool, ToolCategory: category})
}

func observeAnswer(observer AgentObserver, text string) {
	if observer == nil {
		return
	}
	text = safeAnswerPreview(text)
	if text == "" {
		return
	}
	observer(AgentObservation{Type: AgentObservationAnswer, Text: text})
}

func safeAnswerPreview(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	text = removeUnsafeControlRunes(text)
	text = observationUUIDPattern.ReplaceAllString(text, "[ID]")
	text = observationLabeledIDPattern.ReplaceAllString(text, "${1}[ID]")
	text = observationOpaqueIDPattern.ReplaceAllString(text, "[ID]")
	text = observationURIPattern.ReplaceAllString(text, "[링크]")
	text = observationDoubleQuotedPathPattern.ReplaceAllString(text, "[경로]")
	text = observationSingleQuotedPathPattern.ReplaceAllString(text, "[경로]")
	text = observationPathTokenPattern.ReplaceAllString(text, "[경로]")
	return strings.TrimSpace(limitRunes(text, 4000))
}

func removeUnsafeControlRunes(text string) string {
	var builder strings.Builder
	builder.Grow(len(text))
	for _, r := range text {
		if r == '\n' || r == '\t' {
			builder.WriteRune(r)
			continue
		}
		if unicode.IsControl(r) {
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func limitRunes(text string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(text) <= limit {
		return text
	}
	var builder strings.Builder
	count := 0
	for _, r := range text {
		if count >= limit {
			break
		}
		builder.WriteRune(r)
		count++
	}
	return builder.String()
}

func stableWhitespacePrefix(text string) string {
	last := -1
	for index, r := range text {
		if unicode.IsSpace(r) {
			last = index + utf8.RuneLen(r)
		}
	}
	if last < 0 {
		return ""
	}
	return text[:last]
}

func toolCategoryFromName(name string) AgentToolCategory {
	canonical := canonicalToolName(name)
	compact := strings.ReplaceAll(canonical, ".", "")
	if canonical == "" {
		return AgentToolCategoryUnknown
	}
	if isPlasmaTool(canonical) {
		return plasmaToolCategory(canonical)
	}
	switch {
	case strings.Contains(compact, "websearch"):
		return AgentToolCategoryWebSearch
	case strings.Contains(compact, "webfetch"), strings.Contains(compact, "webread"):
		return AgentToolCategoryWebRead
	case strings.Contains(canonical, "validate"), strings.Contains(canonical, "check"), strings.Contains(canonical, "diagnos"):
		return AgentToolCategoryValidate
	default:
		return AgentToolCategoryUnknown
	}
}

func canonicalToolName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	replacer := strings.NewReplacer("__", ".", "_", ".", "-", ".", ":", ".")
	name = replacer.Replace(name)
	for strings.Contains(name, "..") {
		name = strings.ReplaceAll(name, "..", ".")
	}
	return strings.Trim(name, ".")
}

func isPlasmaTool(name string) bool {
	return strings.Contains(name, "plasma.")
}

func plasmaToolCategory(name string) AgentToolCategory {
	switch {
	case strings.Contains(name, "plasma.sources.candidates.propose"):
		return AgentToolCategorySourcePropose
	case containsAnyToolPart(name, ".validate", ".validation", ".check", ".diagnos", ".test"):
		return AgentToolCategoryValidate
	case containsAnyToolPart(name, "plasma.evidence.", "plasma.questions.", "plasma.claims.", "plasma.proposals."):
		return AgentToolCategoryOrganize
	case containsAnyToolPart(name,
		"plasma.mission.get",
		"plasma.sources.list",
		"plasma.sources.read",
		"plasma.sources.tree",
		"plasma.sources.grep",
		"plasma.sources.search",
		"plasma.sources.candidates.read",
		"plasma.research.outline",
		"plasma.research.list",
		"plasma.research.read",
		"plasma.research.grep",
		"plasma.research.references",
		"plasma.local.path.roots",
		"plasma.local.path.tree",
	):
		return AgentToolCategoryMissionRead
	default:
		return AgentToolCategoryUnknown
	}
}

func containsAnyToolPart(name string, parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(name, part) {
			return true
		}
	}
	return false
}
