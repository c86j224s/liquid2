package web

import "github.com/c86j224s/liquid2/plasma/internal/reporting/reportdocument"

import (
	"encoding/json"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/researchproposal"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

type sourceCandidate struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
	State  string `json:"state"`
}

type missionDetailResponse struct {
	Projection          mission.Projection              `json:"projection"`
	ActivityCursor      missionActivityCursor           `json:"activity_cursor"`
	Events              []ledger.Event                  `json:"events"`
	Sources             []sourcecontract.Snapshot       `json:"sources"`
	Records             recordsResponse                 `json:"records"`
	Reports             []reportdocument.Report         `json:"reports"`
	ReportVersions      []reportdocument.ReportVersion  `json:"report_versions"`
	WorkflowRuns        []workflowstate.WorkflowRunView `json:"workflow_runs"`
	Recall              recallPreview                   `json:"recall"`
	AgentExecutors      []agentExecutorStatus           `json:"agent_executors"`
	LockedAgentExecutor string                          `json:"locked_agent_executor,omitempty"`
	ActiveWork          mission.ActiveWorkState         `json:"active_work"`
	ReportProgress      app.ReportProgress              `json:"report_progress"`
}

type recordsResponse struct {
	Evidence        []researchrecords.EvidenceRecord  `json:"evidence"`
	Claims          []researchrecords.ClaimRecord     `json:"claims"`
	ClaimConfidence []claimConfidenceView             `json:"claim_confidence"`
	Questions       []researchrecords.QuestionRecord  `json:"questions"`
	Options         []researchrecords.OptionRecord    `json:"options"`
	Proposals       []researchproposal.ProposalBundle `json:"proposals"`

	approvedObjectIDsByDecisionEventID map[string]map[string]struct{}
}

type claimConfidenceView struct {
	ClaimID           string                        `json:"claim_id"`
	InitialConfidence researchrecords.Confidence    `json:"initial_confidence"`
	CurrentConfidence researchrecords.Confidence    `json:"current_confidence"`
	CurrentEventID    string                        `json:"current_event_id,omitempty"`
	Direction         string                        `json:"direction"`
	UpdatedAt         string                        `json:"updated_at,omitempty"`
	History           []claimConfidenceHistoryEntry `json:"history"`
	HistoryTruncated  bool                          `json:"history_truncated,omitempty"`
}

type claimConfidenceHistoryEntry struct {
	EventID           string          `json:"event_id"`
	Sequence          int64           `json:"sequence"`
	PreviousLevel     string          `json:"previous_level,omitempty"`
	Level             string          `json:"level"`
	Direction         string          `json:"direction"`
	Rationale         string          `json:"rationale"`
	OpenRisks         []string        `json:"open_risks,omitempty"`
	NeedsVerification bool            `json:"needs_verification"`
	BasisEvidenceIDs  []string        `json:"basis_evidence_ids,omitempty"`
	Origin            string          `json:"origin"`
	Producer          ledger.Producer `json:"producer"`
	CreatedAt         string          `json:"created_at,omitempty"`
}

type recallPreview struct {
	SchemaVersion        string                           `json:"schema_version"`
	Mission              recallMission                    `json:"mission"`
	Sources              []sourcecontract.Snapshot        `json:"sources"`
	OpenQuestionIDs      []string                         `json:"open_question_ids"`
	SavedEvidence        []researchrecords.EvidenceRecord `json:"saved_evidence"`
	SavedClaims          []researchrecords.ClaimRecord    `json:"saved_claims"`
	AllowedTools         []string                         `json:"allowed_tools"`
	InvestigationAllowed bool                             `json:"investigation_allowed"`
	SourceSearchAllowed  bool                             `json:"source_search_allowed"`
}

type agentExecutorStatus struct {
	Name                     string                 `json:"name"`
	Label                    string                 `json:"label"`
	Configured               bool                   `json:"configured"`
	DefaultModel             string                 `json:"default_model,omitempty"`
	DefaultModelLabel        string                 `json:"default_model_label,omitempty"`
	DefaultModelVersion      string                 `json:"default_model_version,omitempty"`
	ReasoningEffortSupported bool                   `json:"reasoning_effort_supported"`
	DefaultReasoningEffort   string                 `json:"default_reasoning_effort,omitempty"`
	ReasoningEffortNote      string                 `json:"reasoning_effort_note,omitempty"`
	Models                   []agentModelCapability `json:"models,omitempty"`
}

type agentModelCapability struct {
	Name                   string   `json:"name"`
	Label                  string   `json:"label"`
	ReasoningEfforts       []string `json:"reasoning_efforts"`
	DefaultReasoningEffort string   `json:"default_reasoning_effort"`
}

type recallMission struct {
	MissionID string        `json:"mission_id"`
	Title     string        `json:"title"`
	Objective string        `json:"objective"`
	Scope     mission.Scope `json:"scope"`
}

func approvedEvidence(records recordsResponse) []researchrecords.EvidenceRecord {
	ids := approvedEvidenceIDs(records)
	if len(ids) == 0 {
		return nil
	}
	allowed := map[string]struct{}{}
	for _, id := range ids {
		allowed[id] = struct{}{}
	}
	var approved []researchrecords.EvidenceRecord
	for _, record := range records.Evidence {
		if _, ok := allowed[record.EvidenceID]; ok {
			approved = append(approved, record)
		}
	}
	return approved
}

func approvedEvidenceIDs(records recordsResponse) []string {
	ids := []string{}
	for _, record := range records.Evidence {
		if record.State == "approved" {
			addUnique(&ids, record.EvidenceID)
		}
	}
	for _, proposal := range records.Proposals {
		for _, ref := range proposal.ObjectRefs {
			if ref.ObjectKind == researchrecords.EvidenceRecordObjectKind {
				addApprovedProposalRefID(&ids, proposal, ref.ObjectID, records.approvedObjectIDsByDecisionEventID)
			}
		}
	}
	return ids
}

func approvedClaimsByProposal(records recordsResponse) []researchrecords.ClaimRecord {
	ids := approvedClaimIDs(records)
	if len(ids) == 0 {
		return nil
	}
	allowed := map[string]struct{}{}
	for _, id := range ids {
		allowed[id] = struct{}{}
	}
	var approved []researchrecords.ClaimRecord
	for _, claim := range records.Claims {
		if _, ok := allowed[claim.ClaimID]; ok {
			approved = append(approved, claim)
		}
	}
	return approved
}

func approvedClaimIDs(records recordsResponse) []string {
	ids := []string{}
	for _, claim := range records.Claims {
		if claim.State == "approved" {
			addUnique(&ids, claim.ClaimID)
		}
	}
	for _, proposal := range records.Proposals {
		for _, ref := range proposal.ObjectRefs {
			if ref.ObjectKind == researchrecords.ClaimRecordObjectKind {
				addApprovedProposalRefID(&ids, proposal, ref.ObjectID, records.approvedObjectIDsByDecisionEventID)
			}
		}
	}
	return ids
}

func addApprovedProposalRefID(ids *[]string, proposal researchproposal.ProposalBundle, objectID string, approvedByDecisionEventID map[string]map[string]struct{}) {
	objectID = strings.TrimSpace(objectID)
	if objectID == "" {
		return
	}
	switch proposal.State {
	case "approved", "partially_approved":
		decisionEventID := strings.TrimSpace(proposal.DecisionEventID)
		if decisionEventID == "" {
			return
		}
		if approvedIDs, ok := approvedByDecisionEventID[decisionEventID]; ok {
			if _, approved := approvedIDs[objectID]; approved {
				addUnique(ids, objectID)
			}
		}
	}
}

func approvedObjectIDsByDecisionEventID(events []ledger.Event) map[string]map[string]struct{} {
	approvedByEvent := map[string]map[string]struct{}{}
	for _, event := range events {
		if event.EventType != "proposal.approved" && event.EventType != "proposal.partially_approved" {
			continue
		}
		if !isProposalDecisionProducer(event.Producer) {
			continue
		}
		var payload struct {
			ProposalID        string   `json:"proposal_id"`
			ApprovedObjectIDs []string `json:"approved_object_ids"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		eventID := strings.TrimSpace(event.EventID)
		if eventID == "" || strings.TrimSpace(payload.ProposalID) == "" {
			continue
		}
		if _, ok := approvedByEvent[eventID]; !ok {
			approvedByEvent[eventID] = map[string]struct{}{}
		}
		for _, objectID := range payload.ApprovedObjectIDs {
			objectID = strings.TrimSpace(objectID)
			if objectID != "" {
				approvedByEvent[eventID][objectID] = struct{}{}
			}
		}
	}
	return approvedByEvent
}

func isProposalDecisionProducer(producer ledger.Producer) bool {
	return producer.Type == "user" || producer.Type == "steering_chat"
}

func addUnique(values *[]string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	for _, existing := range *values {
		if existing == value {
			return
		}
	}
	*values = append(*values, value)
}
