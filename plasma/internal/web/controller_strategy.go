package web

import (
	"strings"
)

const (
	controllerStrategyAuto = "auto"
	controllerStrategyV2   = "v2"
	controllerStrategyV3   = "v3"
)

type controllerStrategyDecision struct {
	ID       string
	Label    string
	Reason   string
	Guidance string
}

func selectControllerStrategy(requested string, userText string, recall recallPreview, resumed bool) controllerStrategyDecision {
	normalized := normalizeControllerStrategy(requested)
	if normalized == controllerStrategyAuto {
		normalized = inferControllerStrategy(userText, recall, resumed)
	}
	switch normalized {
	case controllerStrategyV3:
		return controllerStrategyDecision{
			ID:     controllerStrategyV3,
			Label:  "V3 broadening",
			Reason: "The turn asks for broader exploration, competing angles, or repeated reframing.",
			Guidance: strings.Join([]string{
				"Use this as steering guidance, not as a source.",
				"Use repeated lens shifts when they help: first map the current path, then inspect a contrasting or complementary angle, then recover into an actionable synthesis.",
				"Do not keep diverging after the useful breadth is found; return to the user's mission and state what changed.",
			}, "\n"),
		}
	default:
		return controllerStrategyDecision{
			ID:     controllerStrategyV2,
			Label:  "V2 conservative",
			Reason: "The turn can be handled by staying close to the current mission and recovering direction after at most one reframing.",
			Guidance: strings.Join([]string{
				"Use this as steering guidance, not as a source.",
				"Stay close to the user's latest request. If the current path is stuck or too shallow, make one useful reframing, then recover to a concrete answer or next investigation step.",
				"Prefer direct progress over broad expansion unless the user explicitly asks to widen the search.",
			}, "\n"),
		}
	}
}

func normalizeControllerStrategy(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", controllerStrategyAuto:
		return controllerStrategyAuto
	case controllerStrategyV2, "conservative", "narrow":
		return controllerStrategyV2
	case controllerStrategyV3, "broad", "broadening", "divergent":
		return controllerStrategyV3
	default:
		return controllerStrategyAuto
	}
}

func inferControllerStrategy(userText string, recall recallPreview, resumed bool) string {
	text := strings.ToLower(strings.Join([]string{
		userText,
		recall.Mission.Title,
		recall.Mission.Objective,
	}, "\n"))
	if containsAny(text, []string{
		"넓게",
		"폭넓",
		"다각",
		"다른 관점",
		"반대",
		"상충",
		"충돌",
		"논쟁",
		"갈등",
		"비교",
		"대안",
		"trade-off",
		"tradeoff",
		"broaden",
		"diverge",
		"alternative",
		"conflict",
	}) {
		return controllerStrategyV3
	}
	if resumed && containsAny(text, []string{
		"정체",
		"막혔",
		"부족",
		"얕",
		"풍부",
		"깊게",
		"deep",
		"stuck",
	}) {
		return controllerStrategyV3
	}
	return controllerStrategyV2
}

func containsAny(text string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func controllerStrategyPromptBlock(decision controllerStrategyDecision) string {
	if strings.TrimSpace(decision.ID) == "" {
		return ""
	}
	return strings.Join([]string{
		"Controller strategy:",
		"- id: " + strings.TrimSpace(decision.ID),
		"- label: " + strings.TrimSpace(decision.Label),
		"- selection reason: " + strings.TrimSpace(decision.Reason),
		"- guidance:",
		indentLines(strings.TrimSpace(decision.Guidance), "  "),
	}, "\n")
}

func indentLines(text string, prefix string) string {
	if text == "" {
		return prefix
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
