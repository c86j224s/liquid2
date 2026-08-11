package partassembly

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

// ParseAgentPartAssembly는 JSON connective response를 기존 permissive object extraction으로 읽는다.
func ParseAgentPartAssembly(text string) (reporting.PartAssembly, error) {
	raw, err := extractJSONObject(text)
	if err != nil {
		return reporting.PartAssembly{}, fmt.Errorf("%w: part assembly agent did not return JSON", producterror.ErrInvalidInput)
	}
	var assembly reporting.PartAssembly
	decoder := json.NewDecoder(strings.NewReader(raw))
	if err := decoder.Decode(&assembly); err != nil {
		return reporting.PartAssembly{}, fmt.Errorf("%w: invalid part assembly JSON: %v", producterror.ErrInvalidInput, err)
	}
	assembly.Intro = strings.TrimSpace(assembly.Intro)
	assembly.Closing = strings.TrimSpace(assembly.Closing)
	transitions := make([]reporting.PartTransition, 0, len(assembly.Transitions))
	for _, transition := range assembly.Transitions {
		transition.Markdown = strings.TrimSpace(transition.Markdown)
		if transition.AfterSectionIndex <= 0 || transition.Markdown == "" {
			continue
		}
		transitions = append(transitions, transition)
	}
	assembly.Transitions = transitions
	return assembly, nil
}

func extractJSONObject(text string) (string, error) {
	raw := strings.TrimSpace(text)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "```"))
	if strings.HasPrefix(raw, "{") {
		return raw, nil
	}
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return "", fmt.Errorf("%w: JSON object not found", producterror.ErrInvalidInput)
	}
	return raw[start : end+1], nil
}
