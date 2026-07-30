package reporting

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	ReportRequirementMapSchemaVersion  = "plasma.report_requirement_map.v1"
	ReportRequirementsStartedEventType = "report.requirements.started"
	ReportRequirementsMappedEventType  = "report.requirements.mapped"
	ReportRequirementsMappedSentinel   = "REQUIREMENTS_MAPPED"
)

type ReportRequirementMap struct {
	ReviewedEventIDs []string            `json:"reviewed_event_ids"`
	Requirements     []ReportRequirement `json:"requirements"`
}

type ReportRequirement struct {
	RequirementID  string                  `json:"requirement_id"`
	Instruction    string                  `json:"instruction"`
	SourceEventIDs []string                `json:"source_event_ids"`
	Owner          *ReportRequirementOwner `json:"owner,omitempty"`
	UnmappedReason string                  `json:"unmapped_reason,omitempty"`
}

type ReportRequirementOwner struct {
	PartIndex    int `json:"part_index"`
	SectionIndex int `json:"section_index"`
}

type ReportRequirementMapBinding struct {
	MissionID                 string       `json:"mission_id"`
	PendingEventID            string       `json:"pending_event_id"`
	PlanEventID               string       `json:"plan_event_id"`
	ToolSessionID             string       `json:"tool_session_id"`
	PreviousProviderSessionID string       `json:"previous_provider_session_id"`
	IdempotencyKey            string       `json:"idempotency_key"`
	AgentExecutor             string       `json:"agent_executor"`
	AgentModel                string       `json:"agent_model"`
	AgentReasoningEffort      string       `json:"agent_reasoning_effort"`
	Producer                  app.Producer `json:"producer"`
}

func NormalizeReportRequirementMap(value ReportRequirementMap, plan SectionalReportPlan) (ReportRequirementMap, error) {
	reviewed, err := normalizeUniqueIDs(value.ReviewedEventIDs, 256)
	if err != nil || len(reviewed) == 0 {
		return ReportRequirementMap{}, fmt.Errorf("%w: reviewed report requirement events are required", app.ErrInvalidInput)
	}
	reviewedSet := make(map[string]struct{}, len(reviewed))
	for _, eventID := range reviewed {
		reviewedSet[eventID] = struct{}{}
	}
	if len(value.Requirements) > 64 {
		return ReportRequirementMap{}, fmt.Errorf("%w: too many report requirements", app.ErrInvalidInput)
	}
	requirements := make([]ReportRequirement, 0, len(value.Requirements))
	seen := map[string]struct{}{}
	for _, requirement := range value.Requirements {
		requirement.RequirementID = strings.TrimSpace(requirement.RequirementID)
		requirement.Instruction = strings.TrimSpace(requirement.Instruction)
		requirement.UnmappedReason = strings.TrimSpace(requirement.UnmappedReason)
		if !strings.HasPrefix(requirement.RequirementID, "req_") || requirement.Instruction == "" {
			return ReportRequirementMap{}, fmt.Errorf("%w: report requirement identity and instruction are required", app.ErrInvalidInput)
		}
		if _, exists := seen[requirement.RequirementID]; exists {
			return ReportRequirementMap{}, fmt.Errorf("%w: duplicate report requirement id", app.ErrInvalidInput)
		}
		seen[requirement.RequirementID] = struct{}{}
		sources, sourceErr := normalizeUniqueIDs(requirement.SourceEventIDs, 8)
		if sourceErr != nil || len(sources) == 0 {
			return ReportRequirementMap{}, fmt.Errorf("%w: report requirement source events are required", app.ErrInvalidInput)
		}
		for _, sourceID := range sources {
			if _, ok := reviewedSet[sourceID]; !ok {
				return ReportRequirementMap{}, fmt.Errorf("%w: report requirement source was not reviewed", app.ErrInvalidInput)
			}
		}
		requirement.SourceEventIDs = sources
		if err := validateRequirementDestination(requirement, plan); err != nil {
			return ReportRequirementMap{}, err
		}
		requirements = append(requirements, requirement)
	}
	return ReportRequirementMap{ReviewedEventIDs: reviewed, Requirements: requirements}, nil
}

func ReportRequirementMapHash(value ReportRequirementMap) (string, json.RawMessage, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", nil, fmt.Errorf("%w: report requirement map cannot be encoded", app.ErrInvalidInput)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), encoded, nil
}

func ReportRequirementsForSection(value ReportRequirementMap, partIndex, sectionIndex int) []ReportRequirement {
	result := []ReportRequirement{}
	for _, requirement := range value.Requirements {
		if requirement.Owner != nil && requirement.Owner.PartIndex == partIndex && requirement.Owner.SectionIndex == sectionIndex {
			result = append(result, requirement)
		}
	}
	return result
}

func ReportOwnerBoundRequirements(value ReportRequirementMap) []ReportRequirement {
	result := []ReportRequirement{}
	for _, requirement := range value.Requirements {
		if requirement.Owner != nil {
			result = append(result, requirement)
		}
	}
	return result
}

func ReportRequirementsForPart(value ReportRequirementMap, partIndex int) []ReportRequirement {
	result := []ReportRequirement{}
	for _, requirement := range value.Requirements {
		if requirement.Owner != nil && requirement.Owner.PartIndex == partIndex {
			result = append(result, requirement)
		}
	}
	return result
}

func ValidateReportRequirementMapBinding(binding ReportRequirementMapBinding) error {
	if strings.TrimSpace(binding.MissionID) == "" || strings.TrimSpace(binding.PendingEventID) == "" || strings.TrimSpace(binding.PlanEventID) == "" || strings.TrimSpace(binding.ToolSessionID) == "" || strings.TrimSpace(binding.IdempotencyKey) == "" || strings.TrimSpace(binding.AgentExecutor) == "" {
		return fmt.Errorf("%w: report requirement map binding is incomplete", app.ErrInvalidInput)
	}
	if binding.Producer.Type != "agent_session" || strings.TrimSpace(binding.Producer.ID) != strings.TrimSpace(binding.ToolSessionID) {
		return fmt.Errorf("%w: report requirement map producer binding mismatch", app.ErrInvalidInput)
	}
	return nil
}

func validateRequirementDestination(requirement ReportRequirement, plan SectionalReportPlan) error {
	hasOwner := requirement.Owner != nil
	hasReason := requirement.UnmappedReason != ""
	if hasOwner == hasReason {
		return fmt.Errorf("%w: report requirement needs exactly one owner or unmapped reason", app.ErrInvalidInput)
	}
	if !hasOwner {
		return nil
	}
	partIndex := requirement.Owner.PartIndex
	sectionIndex := requirement.Owner.SectionIndex
	if partIndex < 1 || partIndex > len(plan.Parts) || sectionIndex < 1 || sectionIndex > len(plan.Parts[partIndex-1].Sections) {
		return fmt.Errorf("%w: report requirement owner is outside the fixed outline", app.ErrInvalidInput)
	}
	return nil
}

func normalizeUniqueIDs(values []string, limit int) ([]string, error) {
	if len(values) > limit {
		return nil, fmt.Errorf("%w: too many ids", app.ErrInvalidInput)
	}
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("%w: empty id", app.ErrInvalidInput)
		}
		if _, ok := seen[value]; ok {
			return nil, fmt.Errorf("%w: duplicate id", app.ErrInvalidInput)
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}
