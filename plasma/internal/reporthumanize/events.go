package reporthumanize

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const terminalWriteTimeout = 10 * time.Second

func appendRejectedPatch(ctx context.Context, service Service, idFunc IDFunc, missionID string, input Input, toolSessionID string, humanizePendingEventID string, artifact artifact.Raw, finalized ledger.Event, reason string) (ledger.Event, error) {
	ledgerCtx, cancel := terminalWriteContext(ctx)
	defer cancel()
	return service.AppendEvent(ledgerCtx, reporting.BuildHumanizePatchRejectedAppendRequest(reporting.HumanizePatchRejectedEventRequest{
		HumanizeEventBase: eventBase(idFunc("evt"), missionID, input, toolSessionID, humanizePendingEventID, ledger.Producer{Type: "agent", ID: firstNonEmpty(input.ExecutorName, "plasma")}),
		PatchEventID:      finalized.EventID,
		Artifact:          artifact,
		Reason:            reason,
	}))
}

func appendPending(ctx context.Context, service Service, idFunc IDFunc, missionID string, input Input, toolSessionID string) (ledger.Event, error) {
	eventID := idFunc("evt")
	return service.AppendEvent(ctx, reporting.BuildHumanizePendingAppendRequest(reporting.HumanizePendingEventRequest{
		HumanizeEventBase: eventBase(eventID, missionID, input, toolSessionID, eventID, ledger.Producer{Type: "agent", ID: firstNonEmpty(input.ExecutorName, "plasma")}),
	}))
}

func appendSkipped(ctx context.Context, service Service, idFunc IDFunc, missionID string, input Input, toolSessionID string, humanizePendingEventID string, durationMS int64) (ledger.Event, error) {
	ledgerCtx, cancel := terminalWriteContext(ctx)
	defer cancel()
	event, _, err := appendTerminal(ledgerCtx, service, missionID, humanizePendingEventID, reporting.BuildHumanizeSkippedAppendRequest(reporting.HumanizeSkippedEventRequest{
		HumanizeEventBase: eventBase(idFunc("evt"), missionID, input, toolSessionID, humanizePendingEventID, ledger.Producer{Type: "agent", ID: firstNonEmpty(input.ExecutorName, "plasma")}),
		DurationMS:        durationMS,
	}))
	return event, err
}

func appendFailed(ctx context.Context, service Service, idFunc IDFunc, missionID string, input Input, toolSessionID string, humanizePendingEventID string, durationMS int64, cause error) (ledger.Event, error) {
	ledgerCtx, cancel := terminalWriteContext(ctx)
	defer cancel()
	event, _, err := appendTerminal(ledgerCtx, service, missionID, humanizePendingEventID, reporting.BuildHumanizeFailedAppendRequest(reporting.HumanizeFailedEventRequest{
		HumanizeEventBase: eventBase(idFunc("evt"), missionID, input, toolSessionID, humanizePendingEventID, ledger.Producer{Type: "agent", ID: firstNonEmpty(input.ExecutorName, "plasma")}),
		DurationMS:        durationMS,
		Error:             cause.Error(),
	}))
	if err != nil {
		log.Printf("report_terminal_write_failed mission_id=%q pending_event_id=%q report_type=%q intended_event_type=%q err=%q", missionID, humanizePendingEventID, "humanize", "report.humanize.failed", err)
	}
	return event, err
}

// AppendFailed closes an H5 pending event with the existing failure payload
// shape. It is exposed for transport wrappers and cancellation tests only.
func AppendFailed(ctx context.Context, service Service, idFunc IDFunc, missionID string, input Input, toolSessionID string, humanizePendingEventID string, durationMS int64, cause error) (ledger.Event, error) {
	return appendFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS, cause)
}

// PendingPayloadFromEvent decodes a report.humanize.pending payload for restart
// ownership and recovery decisions. Invalid payloads decode to the zero value.
func PendingPayloadFromEvent(event ledger.Event) PendingPayload {
	var payload PendingPayload
	_ = json.Unmarshal(event.Payload, &payload)
	return payload
}

// InFlightPendingEventID returns the parent report pending event that owns a
// humanize pending event for cancellation and in-flight bookkeeping.
func InFlightPendingEventID(event ledger.Event) string {
	payload := PendingPayloadFromEvent(event)
	return strings.TrimSpace(payload.ReportPendingEventID)
}

func eventBase(eventID string, missionID string, input Input, toolSessionID string, pendingEventID string, producer ledger.Producer) reporting.HumanizeEventBase {
	return reporting.HumanizeEventBase{
		EventID:                eventID,
		MissionID:              missionID,
		PendingEventID:         pendingEventID,
		ReportPendingEventID:   input.PendingEventID,
		Title:                  input.Title,
		SourceArtifactID:       input.SourceArtifact.ArtifactID,
		SourceArtifactSHA256:   input.SourceArtifact.SHA256,
		AgentExecutor:          input.ExecutorName,
		AgentModel:             input.AgentModel,
		AgentReasoningEffort:   input.ReasoningEffort,
		PreviousAgentSessionID: input.PreviousSessionID,
		ToolSessionID:          toolSessionID,
		MCPMode:                input.MCPMode,
		ReportMode:             input.ReportMode,
		Producer:               producer,
	}
}

func terminalExists(ctx context.Context, service Service, missionID string, pendingEventID string) bool {
	pendingEventID = strings.TrimSpace(pendingEventID)
	if pendingEventID == "" {
		return false
	}
	lister, ok := service.(eventLister)
	if !ok {
		return false
	}
	events, err := lister.ListEvents(ctx, missionID)
	if err != nil {
		return false
	}
	_, ok = reportexecution.CompletedPendingEventIDs(events)[pendingEventID]
	return ok
}

func appendTerminal(ctx context.Context, service Service, missionID, pendingEventID string, req ledger.AppendRequest) (ledger.Event, bool, error) {
	appended, closed, err := service.AppendReportTerminalIfOpen(ctx, missionID, pendingEventID, []ledger.AppendRequest{req})
	if err != nil || !closed {
		return ledger.Event{}, closed, err
	}
	return appended[0], true, nil
}

func terminalWriteContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx != nil && ctx.Err() == nil {
		return ctx, func() {}
	}
	return context.WithTimeout(context.Background(), terminalWriteTimeout)
}
