// Package agentmodels owns Plasma's product-level Codex model policy.
package agentmodels

import (
	"fmt"
	"strings"
)

const (
	DefaultModel           = "gpt-5.6-terra"
	DefaultReasoningEffort = "medium"
)

// Model describes a selectable Codex model and its supported reasoning efforts.
type Model struct {
	Name                   string   `json:"name"`
	Label                  string   `json:"label"`
	ReasoningEfforts       []string `json:"reasoning_efforts"`
	DefaultReasoningEffort string   `json:"default_reasoning_effort"`
}

var catalog = []Model{
	{Name: "gpt-5.6-sol", Label: "GPT-5.6 Sol", ReasoningEfforts: []string{"low", "medium", "high", "xhigh", "max", "ultra"}, DefaultReasoningEffort: "medium"},
	{Name: "gpt-5.6-terra", Label: "GPT-5.6 Terra", ReasoningEfforts: []string{"low", "medium", "high", "xhigh", "max", "ultra"}, DefaultReasoningEffort: "medium"},
	{Name: "gpt-5.6-luna", Label: "GPT-5.6 Luna", ReasoningEfforts: []string{"low", "medium", "high", "xhigh", "max"}, DefaultReasoningEffort: "medium"},
	{Name: "gpt-5.5", Label: "GPT-5.5", ReasoningEfforts: []string{"low", "medium", "high", "xhigh"}, DefaultReasoningEffort: "medium"},
	{Name: "gpt-5.4", Label: "GPT-5.4", ReasoningEfforts: []string{"low", "medium", "high", "xhigh"}, DefaultReasoningEffort: "medium"},
	{Name: "gpt-5.4-mini", Label: "GPT-5.4 mini", ReasoningEfforts: []string{"low", "medium", "high", "xhigh"}, DefaultReasoningEffort: "medium"},
	{Name: "gpt-5.3-codex-spark", Label: "GPT-5.3 Codex Spark", ReasoningEfforts: []string{"low", "medium", "high", "xhigh"}, DefaultReasoningEffort: "medium"},
}

var genericReasoningEfforts = []string{"low", "medium", "high", "xhigh"}

// Catalog returns a copy suitable for publishing through a transport layer.
func Catalog() []Model {
	result := make([]Model, len(catalog))
	for i, model := range catalog {
		result[i] = clone(model)
	}
	return result
}

// Default returns the product's default Codex model metadata.
func Default() Model {
	model, _ := lookup(DefaultModel)
	return clone(model)
}

// Resolve applies product defaults and validates known model capabilities.
// Unknown models remain compatible with future Codex releases and use the
// previously supported generic reasoning efforts.
func Resolve(model, effort string) (string, string, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		model = DefaultModel
	}
	effort = strings.ToLower(strings.TrimSpace(effort))
	if effort == "" {
		effort = DefaultReasoningEffort
		if known, ok := lookup(model); ok {
			effort = known.DefaultReasoningEffort
		}
	}
	allowed := genericReasoningEfforts
	if known, ok := lookup(model); ok {
		allowed = known.ReasoningEfforts
	}
	if !contains(allowed, effort) {
		return "", "", fmt.Errorf("unsupported reasoning effort %q for model %q", effort, model)
	}
	return model, effort, nil
}

// ResolveForSession preserves legacy resumed sessions that did not record
// either setting. Defaults are only applied when starting a new session.
func ResolveForSession(model, effort, previousSessionID string) (string, string, error) {
	if strings.TrimSpace(previousSessionID) != "" && strings.TrimSpace(model) == "" && strings.TrimSpace(effort) == "" {
		return "", "", nil
	}
	return Resolve(model, effort)
}

func lookup(name string) (Model, bool) {
	for _, model := range catalog {
		if model.Name == name {
			return model, true
		}
	}
	return Model{}, false
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func clone(model Model) Model {
	model.ReasoningEfforts = append([]string(nil), model.ReasoningEfforts...)
	return model
}
