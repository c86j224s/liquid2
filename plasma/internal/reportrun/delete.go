package reportrun

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportusage"
)

const (
	BlockerActiveRun          = "active_run"
	BlockerInFlight           = "in_flight"
	BlockerOpenPending        = "open_pending"
	BlockerAmbiguousLineage   = "ambiguous_lineage"
	BlockerMalformedReference = "malformed_reference"
	BlockerPurged             = "purged"
	BlockerNotCompleted       = "not_completed"
	BlockerCompletionPending  = "completion_pending"
)

// PreviewDelete applies report-run product deletion rules to a storage facts
// snapshot. activePendingEventID is process-local state supplied by the web
// adapter; durable pending/terminal checks are derived from member events.
func PreviewDelete(facts DeleteFacts, activePendingEventID string) DeletePreview {
	usage := AggregateUsage(facts.Events)
	eventIDs, artifactIDs, artifactBytes := deletableMembers(facts)
	blockers := deleteBlockers(facts, activePendingEventID)
	return DeletePreview{
		RunID:                  facts.Run.RunID,
		State:                  facts.Run.LifecycleState,
		Revision:               facts.Run.Revision,
		Eligible:               len(blockers) == 0,
		DeletableEventCount:    int64(len(eventIDs)),
		DeletableArtifactCount: int64(len(artifactIDs)),
		DeletableArtifactBytes: artifactBytes,
		SharedArtifactCount:    int64(len(facts.SharedArtifacts)),
		SharedArtifacts:        append([]SharedArtifact(nil), facts.SharedArtifacts...),
		Blockers:               blockers,
		Usage:                  usage,
		DeleteFactsHash:        DeleteFactsHash(facts),
	}
}

// BuildDeleteDecision returns the exact member IDs that a transaction may
// delete, or blockers explaining why it must not mutate state.
func BuildDeleteDecision(facts DeleteFacts, activePendingEventID string, purgedByType string, purgedByID string, now time.Time) DeleteDecision {
	preview := PreviewDelete(facts, activePendingEventID)
	eventIDs, artifactIDs, _ := deletableMembers(facts)
	return DeleteDecision{
		Preview:           preview,
		DeleteEventIDs:    eventIDs,
		DeleteArtifactIDs: artifactIDs,
		RetainedUsage:     preview.Usage,
		PurgedAt:          now,
		PurgedByType:      strings.TrimSpace(purgedByType),
		PurgedByID:        strings.TrimSpace(purgedByID),
	}
}

// DeleteFactsHash returns the lowercase SHA-256 binding between a preview and
// the exact external facts used by a later delete transaction.
func DeleteFactsHash(facts DeleteFacts) string {
	eventIDs, artifacts, _ := deletableMemberDetails(facts)
	if eventIDs == nil {
		eventIDs = []string{}
	}
	if artifacts == nil {
		artifacts = []deleteFactsHashArtifact{}
	}
	shared := make([]deleteFactsHashSharedArtifact, 0, len(facts.SharedArtifacts))
	for _, item := range facts.SharedArtifacts {
		reasons := append([]string(nil), item.Reasons...)
		sort.Strings(reasons)
		shared = append(shared, deleteFactsHashSharedArtifact{
			ArtifactID: item.ArtifactID,
			ByteSize:   item.ByteSize,
			Reasons:    reasons,
		})
	}
	if shared == nil {
		shared = []deleteFactsHashSharedArtifact{}
	}
	sort.Slice(shared, func(i, j int) bool { return shared[i].ArtifactID < shared[j].ArtifactID })
	payload := deleteFactsHashPayload{
		Version:            "report_delete_facts.v1",
		MissionID:          facts.Run.MissionID,
		RunID:              facts.Run.RunID,
		FinalArtifactID:    facts.Run.FinalArtifactID,
		LifecycleState:     facts.Run.LifecycleState,
		Revision:           facts.Run.Revision,
		MalformedReference: facts.MalformedReference,
		DeleteEventIDs:     eventIDs,
		DeleteArtifacts:    artifacts,
		SharedArtifacts:    shared,
	}
	encoded, _ := json.Marshal(payload)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

type deleteFactsHashPayload struct {
	Version            string                          `json:"version"`
	MissionID          string                          `json:"mission_id"`
	RunID              string                          `json:"run_id"`
	FinalArtifactID    string                          `json:"final_artifact_id"`
	LifecycleState     string                          `json:"lifecycle_state"`
	Revision           int64                           `json:"revision"`
	MalformedReference bool                            `json:"malformed_reference"`
	DeleteEventIDs     []string                        `json:"delete_event_ids"`
	DeleteArtifacts    []deleteFactsHashArtifact       `json:"delete_artifacts"`
	SharedArtifacts    []deleteFactsHashSharedArtifact `json:"shared_artifacts"`
}

type deleteFactsHashArtifact struct {
	ArtifactID string `json:"artifact_id"`
	ByteSize   int64  `json:"byte_size"`
}

type deleteFactsHashSharedArtifact struct {
	ArtifactID string   `json:"artifact_id"`
	ByteSize   int64    `json:"byte_size"`
	Reasons    []string `json:"reasons"`
}

func deletableMembers(facts DeleteFacts) ([]string, []string, int64) {
	eventIDs, artifacts, bytes := deletableMemberDetails(facts)
	artifactIDs := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		artifactIDs = append(artifactIDs, artifact.ArtifactID)
	}
	return eventIDs, artifactIDs, bytes
}

func deletableMemberDetails(facts DeleteFacts) ([]string, []deleteFactsHashArtifact, int64) {
	eventSeen := map[string]bool{}
	var eventIDs []string
	for _, member := range facts.Events {
		if member.Event.EventID == "" || eventSeen[member.Event.EventID] {
			continue
		}
		eventSeen[member.Event.EventID] = true
		eventIDs = append(eventIDs, member.Event.EventID)
	}
	sort.Strings(eventIDs)
	shared := map[string]bool{}
	for _, item := range facts.SharedArtifacts {
		shared[item.ArtifactID] = true
	}
	artifactSeen := map[string]bool{}
	var artifacts []deleteFactsHashArtifact
	var bytes int64
	for _, member := range facts.Artifacts {
		if !artifactDeleteCandidate(member) || shared[member.Artifact.ArtifactID] || artifactSeen[member.Artifact.ArtifactID] {
			continue
		}
		artifactSeen[member.Artifact.ArtifactID] = true
		artifacts = append(artifacts, deleteFactsHashArtifact{ArtifactID: member.Artifact.ArtifactID, ByteSize: member.Artifact.ByteSize})
		bytes += member.Artifact.ByteSize
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].ArtifactID < artifacts[j].ArtifactID })
	return eventIDs, artifacts, bytes
}

func artifactDeleteCandidate(member MemberArtifact) bool {
	return member.Membership.Ownership == OwnershipCreated && member.Membership.ArtifactRole != ArtifactRoleInput
}

func deleteBlockers(facts DeleteFacts, activePendingEventID string) []Blocker {
	var blockers []Blocker
	add := func(code, message string) {
		blockers = append(blockers, Blocker{ReasonCode: code, Message: message})
	}
	switch facts.Run.LifecycleState {
	case LifecyclePurged:
		add(BlockerPurged, "이미 삭제된 보고서 실행입니다.")
	case LifecycleAmbiguous:
		add(BlockerAmbiguousLineage, "보고서 실행 계보가 모호해 삭제할 수 없습니다.")
	case LifecycleActive:
		add(BlockerActiveRun, "보고서 실행이 아직 진행 중입니다.")
	case LifecycleCompleted, LifecycleFailed, LifecycleCanceled:
	default:
		add(BlockerAmbiguousLineage, "보고서 실행 상태를 확인할 수 없습니다.")
	}
	if facts.Run.LifecycleState != LifecycleCompleted {
		add(BlockerNotCompleted, "완료된 Markdown 보고서만 삭제할 수 있습니다.")
	}
	if facts.Run.LifecycleState == LifecycleCompleted && !validCompletionMember(facts) {
		add(BlockerCompletionPending, "보고서 실행 완료 bookkeeping이 아직 확정되지 않았습니다.")
	}
	if facts.MalformedReference {
		add(BlockerMalformedReference, "외부 artifact 참조를 확인할 수 없는 장부 payload가 있습니다.")
	}
	if activePendingEventID != "" && memberEventID(facts.Events, activePendingEventID) {
		add(BlockerInFlight, "현재 프로세스에서 이 보고서 실행 작업이 진행 중입니다.")
	}
	for _, pendingID := range openPendingIDs(facts.Events) {
		add(BlockerOpenPending, "종료 이벤트가 없는 보고서 작업이 남아 있습니다: "+pendingID)
	}
	return blockers
}

type completionDeleteTarget struct {
	event   Event
	pending string
	meta    reportusage.Target
}

func validCompletionMember(facts DeleteFacts) bool {
	for _, member := range facts.Events {
		event := member.Event
		if event.EventType != "report.run.completed" || event.EventID != completionDeleteEventID(facts.Run.RunID) ||
			event.MissionID != facts.Run.MissionID || event.Producer != (ledger.Producer{Type: "system", ID: "report-completion"}) ||
			event.CorrelationID != facts.Run.RunID || member.Membership.EventID != event.EventID ||
			member.Membership.MissionID != facts.Run.MissionID || member.Membership.RunID != facts.Run.RunID ||
			member.Membership.EventRole != "completion" {
			continue
		}
		var payload struct {
			Kind             string `json:"kind"`
			SchemaVersion    string `json:"schema_version"`
			RunID            string `json:"run_id"`
			ArtifactID       string `json:"artifact_id"`
			CanonicalEventID string `json:"canonical_event_id"`
			PendingEventID   string `json:"pending_event_id"`
			TargetCount      int    `json:"delayed_usage_target_count"`
			Recorded         int    `json:"usage_recorded_count"`
			Unavailable      int    `json:"usage_unavailable_count"`
		}
		if json.Unmarshal(event.Payload, &payload) != nil || payload.Kind != "report_run_completed" ||
			payload.SchemaVersion != "plasma.report_run_completion.v1" || payload.RunID != facts.Run.RunID ||
			payload.ArtifactID == "" || payload.CanonicalEventID == "" || payload.PendingEventID == "" ||
			payload.TargetCount < 0 || payload.Recorded < 0 || payload.Unavailable < 0 ||
			payload.Recorded+payload.Unavailable != payload.TargetCount || member.Membership.AttemptEventID != payload.PendingEventID {
			continue
		}
		if !validRootPendingMember(facts) {
			continue
		}
		canonical, ok := exactReportMember(facts, payload.CanonicalEventID, "report.artifact.created", "final", payload.PendingEventID)
		if !ok {
			continue
		}
		var artifactPayload struct {
			PendingEventID string `json:"pending_event_id"`
			ArtifactID     string `json:"artifact_id"`
		}
		if json.Unmarshal(canonical.Event.Payload, &artifactPayload) != nil || artifactPayload.PendingEventID != payload.PendingEventID || artifactPayload.ArtifactID != payload.ArtifactID || event.CausationEventID != canonical.Event.EventID {
			continue
		}
		if _, ok := exactReportMember(facts, payload.PendingEventID, "report.draft.pending", "draft_pending", payload.PendingEventID); !ok {
			continue
		}
		if validCompletionUsageOutcomes(facts, payload.TargetCount, payload.Recorded, payload.Unavailable) {
			return true
		}
	}
	return false
}

func completionDeleteEventID(runID string) string {
	return "evt_report_run_completed_" + strings.TrimPrefix(strings.TrimSpace(runID), "evt_")
}

func validRootPendingMember(facts DeleteFacts) bool {
	member, ok := exactReportMember(facts, facts.Run.RootPendingEventID, "report.draft.pending", "draft_pending", facts.Run.RootPendingEventID)
	return ok && member.Event.MissionID == facts.Run.MissionID
}

func exactReportMember(facts DeleteFacts, eventID, eventType, role, attempt string) (MemberEvent, bool) {
	for _, member := range facts.Events {
		if member.Event.EventID != eventID || member.Event.EventType != eventType || member.Event.MissionID != facts.Run.MissionID ||
			member.Membership.EventID != member.Event.EventID || member.Membership.MissionID != facts.Run.MissionID ||
			member.Membership.RunID != facts.Run.RunID || member.Membership.EventRole != role || member.Membership.AttemptEventID != attempt {
			continue
		}
		return member, true
	}
	return MemberEvent{}, false
}

func validCompletionUsageOutcomes(facts DeleteFacts, targetCount, recorded, unavailable int) bool {
	targets, ok := completionDeleteTargets(facts)
	if !ok || len(targets) != targetCount {
		return false
	}
	seenUsage := map[string]int{}
	counts := [2]int{}
	for _, member := range facts.Events {
		if member.Event.EventType != "report.agent_usage.recorded" {
			continue
		}
		if member.Event.MissionID != facts.Run.MissionID || member.Membership.EventID != member.Event.EventID ||
			member.Membership.MissionID != facts.Run.MissionID || member.Membership.RunID != facts.Run.RunID ||
			member.Membership.EventRole != "stage" {
			return false
		}
		var payload struct {
			Kind                     string                `json:"kind"`
			PendingEventID           string                `json:"pending_event_id"`
			CorrelationEventID       string                `json:"correlation_event_id"`
			ForkSourceAgentSessionID string                `json:"fork_source_agent_session_id"`
			AgentUsage               agentusage.AgentUsage `json:"agent_usage"`
		}
		if json.Unmarshal(member.Event.Payload, &payload) != nil {
			return false
		}
		target, ok := targets[payload.CorrelationEventID]
		if !ok {
			return false
		}
		if member.Event.EventID != "evt_report_usage_"+strings.TrimPrefix(target.event.EventID, "evt_") ||
			member.Event.CausationEventID != target.event.EventID || member.Event.CorrelationID != target.pending ||
			payload.Kind != "report_agent_usage" || payload.PendingEventID != target.pending ||
			payload.CorrelationEventID != target.event.EventID || member.Membership.AttemptEventID != target.pending ||
			target.meta.AgentSessionID == "" || member.Event.Producer != (ledger.Producer{Type: "agent_session", ID: target.meta.AgentSessionID}) {
			return false
		}
		usage := payload.AgentUsage
		if usage.SchemaVersion != agentusage.SchemaVersion || usage.Empty() || usage.Session.AgentSessionID != target.meta.AgentSessionID ||
			usage.Session.PreviousAgentSessionID != target.meta.PreviousAgentSessionID || usage.Surface != target.meta.Surface ||
			usage.Executor != target.meta.AgentExecutor || usage.Model != target.meta.AgentModel ||
			usage.ReasoningEffort != target.meta.AgentReasoningEffort || payload.ForkSourceAgentSessionID != target.meta.ForkSourceAgentSessionID ||
			(usage.ProviderUsage == nil && !usage.UsageUnavailable) || (usage.ProviderUsage != nil && usage.UsageUnavailable) || (usage.UsageUnavailable && strings.TrimSpace(usage.UsageUnavailableReason) == "") {
			return false
		}
		seenUsage[member.Event.EventID]++
		if usage.ProviderUsage != nil {
			counts[0]++
		} else {
			counts[1]++
		}
	}
	if len(seenUsage) != len(targets) || counts[0] != recorded || counts[1] != unavailable {
		return false
	}
	for targetID := range targets {
		if seenUsage["evt_report_usage_"+strings.TrimPrefix(targetID, "evt_")] != 1 {
			return false
		}
	}
	return true
}

func targetPending(event Event) string {
	var payload struct {
		PendingEventID string `json:"pending_event_id"`
	}
	if json.Unmarshal(event.Payload, &payload) != nil {
		return ""
	}
	return strings.TrimSpace(payload.PendingEventID)
}

func completionDeleteTargets(facts DeleteFacts) (map[string]completionDeleteTarget, bool) {
	targets := map[string]completionDeleteTarget{}
	for _, member := range facts.Events {
		if member.Event.MissionID != facts.Run.MissionID || member.Membership.EventID != member.Event.EventID || member.Membership.MissionID != facts.Run.MissionID || member.Membership.RunID != facts.Run.RunID {
			return nil, false
		}
		converted := ledger.Event{EventID: member.Event.EventID, MissionID: member.Event.MissionID, Sequence: member.Event.Sequence, EventType: member.Event.EventType, Producer: member.Event.Producer, CausationEventID: member.Event.CausationEventID, CorrelationID: member.Event.CorrelationID, Payload: member.Event.Payload, CreatedAt: member.Event.CreatedAt}
		meta, recognized, err := reportusage.TargetForEvent(converted)
		if err != nil {
			return nil, false
		}
		if !recognized {
			continue
		}
		if member.Membership.EventRole != "stage" {
			return nil, false
		}
		pendingID := targetPending(member.Event)
		if pendingID == "" {
			return nil, false
		}
		pending, ok := exactReportMember(facts, pendingID, "report.draft.pending", "draft_pending", pendingID)
		if !ok || member.Membership.AttemptEventID != pending.Event.EventID {
			return nil, false
		}
		if _, duplicate := targets[member.Event.EventID]; duplicate {
			return nil, false
		}
		targets[member.Event.EventID] = completionDeleteTarget{event: member.Event, pending: pending.Event.EventID, meta: meta}
	}
	return targets, true
}

func memberEventID(events []MemberEvent, eventID string) bool {
	for _, member := range events {
		if member.Event.EventID == eventID {
			return true
		}
	}
	return false
}

func openPendingIDs(events []MemberEvent) []string {
	pending := map[string]bool{}
	for _, member := range events {
		switch member.Event.EventType {
		case "report.draft.pending", "report.design.pending", "report.humanize.pending", "report.patch.pending":
			pending[member.Event.EventID] = true
		}
	}
	for _, member := range events {
		if !terminalEventClosesPending(member.Event.EventType) {
			continue
		}
		pendingID := terminalPendingID(member.Event)
		if pendingID != "" {
			delete(pending, pendingID)
		}
	}
	var ids []string
	for id := range pending {
		ids = append(ids, id)
	}
	return ids
}

func terminalEventClosesPending(eventType string) bool {
	switch eventType {
	case "report.artifact.created", "report.artifact.exported",
		"report.draft.failed", "report.design.failed", "report.humanize.failed",
		"report.humanize.skipped", "report.patch.failed":
		return true
	default:
		return false
	}
}

func terminalPendingID(event Event) string {
	var payload struct {
		PendingID  string `json:"pending_event_id"`
		Generation struct {
			PendingID string `json:"pending_event_id"`
		} `json:"generation"`
	}
	if json.Unmarshal(event.Payload, &payload) != nil {
		return ""
	}
	if payload.PendingID != "" {
		return strings.TrimSpace(payload.PendingID)
	}
	return strings.TrimSpace(payload.Generation.PendingID)
}
