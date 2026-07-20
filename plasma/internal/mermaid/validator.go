package mermaid

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	RendererVersion = "mermaid 11.16.0"
	maxSourceBytes  = 50000
)

type Issue struct {
	Kind        string `json:"kind"`
	Message     string `json:"message"`
	Line        int    `json:"line,omitempty"`
	Column      int    `json:"column,omitempty"`
	Expectation string `json:"expectation,omitempty"`
}

type Result struct {
	OK               bool     `json:"ok"`
	DiagramType      string   `json:"diagram_type"`
	RendererVersion  string   `json:"renderer_version"`
	ValidationMode   string   `json:"validation_mode"`
	CanConfirmRender bool     `json:"can_confirm_render"`
	SourceBytes      int      `json:"source_bytes"`
	SourceLineCount  int      `json:"source_line_count"`
	Errors           []Issue  `json:"errors,omitempty"`
	Warnings         []Issue  `json:"warnings,omitempty"`
	CheckedRules     []string `json:"checked_rules"`
	ExpectedBehavior string   `json:"expected_behavior"`
}

func Validate(source string) Result {
	result := Result{
		RendererVersion:  RendererVersion,
		ValidationMode:   "static_preflight",
		CanConfirmRender: false,
		CheckedRules: []string{
			"source size and UTF-8",
			"diagram type detection",
			"requirementDiagram parse-risk rules",
			"known compatibility warnings",
		},
		ExpectedBehavior: "If ok is false, revise the Mermaid source before showing it to the user. If ok is true, the source passed Plasma's static preflight, but only the browser preview can fully confirm rendering.",
	}
	if !utf8.ValidString(source) {
		result.Errors = append(result.Errors, Issue{Kind: "invalid_utf8", Message: "Mermaid source must be UTF-8 text.", Expectation: "Submit UTF-8 Mermaid source text."})
		result.OK = false
		return result
	}
	normalized, stripped := normalizeSource(source)
	result.SourceBytes = len(normalized)
	result.SourceLineCount = lineCount(normalized)
	if stripped {
		result.Warnings = append(result.Warnings, Issue{Kind: "fence_stripped", Message: "Markdown code fences were removed before validation.", Expectation: "Pass only Mermaid source when possible."})
	}
	if strings.TrimSpace(normalized) == "" {
		result.Errors = append(result.Errors, Issue{Kind: "empty_source", Message: "Mermaid source is empty.", Expectation: "Provide a Mermaid diagram body."})
		result.OK = false
		return result
	}
	if len(normalized) > maxSourceBytes {
		result.Errors = append(result.Errors, Issue{Kind: "source_too_large", Message: fmt.Sprintf("Mermaid source exceeds %d bytes.", maxSourceBytes), Expectation: "Shorten the diagram or split it into multiple diagrams."})
		result.OK = false
		return result
	}
	result.DiagramType = detectDiagramType(normalized)
	if result.DiagramType == "" {
		result.Errors = append(result.Errors, Issue{Kind: "unknown_diagram_type", Message: "Could not detect a supported Mermaid diagram type from the first non-empty line.", Line: firstNonEmptyLineNumber(normalized), Expectation: "Start with a Mermaid diagram directive such as flowchart TD, sequenceDiagram, or requirementDiagram."})
	}
	switch result.DiagramType {
	case "requirementDiagram":
		result.Errors = append(result.Errors, validateRequirementDiagram(normalized)...)
	case "C4Context", "xychart-beta", "sankey-beta", "block-beta", "packet-beta":
		result.Warnings = append(result.Warnings, Issue{Kind: "compatibility_sensitive", Message: result.DiagramType + " is compatibility-sensitive across Mermaid versions.", Expectation: "Keep a simpler fallback or check the Plasma browser preview after validation."})
	}
	result.OK = len(result.Errors) == 0
	return result
}

func normalizeSource(source string) (string, bool) {
	trimmed := strings.TrimSpace(strings.ReplaceAll(source, "\r\n", "\n"))
	if strings.HasPrefix(trimmed, "```") {
		lines := strings.Split(trimmed, "\n")
		if len(lines) >= 3 && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
			return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n")), true
		}
	}
	return trimmed, false
}

func detectDiagramType(source string) string {
	for _, line := range strings.Split(source, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 {
			continue
		}
		first := fields[0]
		switch first {
		case "flowchart", "graph", "sequenceDiagram", "classDiagram", "stateDiagram-v2", "erDiagram", "gantt", "journey", "pie", "quadrantChart", "gitGraph", "mindmap", "timeline", "requirementDiagram", "C4Context", "xychart-beta", "sankey-beta", "block-beta", "packet-beta":
			return first
		}
		return ""
	}
	return ""
}

func validateRequirementDiagram(source string) []Issue {
	var issues []Issue
	for index, raw := range strings.Split(source, "\n") {
		lineNo := index + 1
		line := strings.TrimSpace(raw)
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "id":
			if invalidRequirementID(value) {
				issues = append(issues, Issue{Kind: "requirement_id_token", Message: "requirementDiagram id values should not contain hyphens, spaces, or punctuation unless Mermaid accepts them in this grammar position.", Line: lineNo, Column: strings.Index(raw, value) + 1, Expectation: "Use an identifier such as AUTH_ROOT or a numeric id instead of AUTH-ROOT."})
			}
		case "text":
			if containsComma(value) && !isQuoted(value) {
				issues = append(issues, Issue{Kind: "requirement_text_needs_quotes", Message: "requirementDiagram text values containing commas must be quoted for Mermaid 11.16.0.", Line: lineNo, Column: strings.Index(raw, value) + 1, Expectation: `Use text: "Access decisions must combine identity, policy, and auditability".`})
			}
		}
	}
	return issues
}

func invalidRequirementID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || isQuoted(value) {
		return false
	}
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			continue
		}
		return true
	}
	return false
}

func containsComma(value string) bool {
	return strings.Contains(value, ",")
}

func isQuoted(value string) bool {
	value = strings.TrimSpace(value)
	return len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\''))
}

func firstNonEmptyLineNumber(source string) int {
	for index, line := range strings.Split(source, "\n") {
		if strings.TrimSpace(line) != "" {
			return index + 1
		}
	}
	return 0
}

func lineCount(source string) int {
	if source == "" {
		return 0
	}
	return strings.Count(source, "\n") + 1
}
