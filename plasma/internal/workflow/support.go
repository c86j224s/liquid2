package workflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

func (runner Runner) appendWorkflowEvent(ctx context.Context, missionID string, eventType string, payload any, producer ledger.Producer) (ledger.Event, error) {
	return runner.appendEvent(ctx, missionID, eventType, payload, producer)
}

func (runner Runner) appendEvent(ctx context.Context, missionID string, eventType string, payload any, producer ledger.Producer) (ledger.Event, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return ledger.Event{}, err
	}
	return runner.Service.AppendEvent(ctx, ledger.AppendRequest{
		EventID:   runner.newID("evt"),
		MissionID: missionID,
		EventType: eventType,
		Producer:  producer,
		Payload:   encoded,
	})
}

func (runner Runner) now() time.Time {
	if runner.Now != nil {
		return runner.Now().UTC()
	}
	return time.Now().UTC()
}

func (runner Runner) stepTimeout() time.Duration {
	if runner.StepTimeout <= 0 {
		return 25 * time.Minute
	}
	return runner.StepTimeout
}

func agentExecutionError(ctx context.Context, runErr error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return runErr
}

func (runner Runner) newID(prefix string) string {
	if runner.NewID != nil {
		return runner.NewID(prefix)
	}
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s_%s_%s", strings.TrimSuffix(prefix, "_"), time.Now().UTC().Format("20060102150405"), hex.EncodeToString(b[:]))
}

func workflowTerminalStatus(status string) bool {
	switch status {
	case workflowstate.WorkflowStatusCompleted, workflowstate.WorkflowStatusPaused, workflowstate.WorkflowStatusStopped, workflowstate.WorkflowStatusFailed, workflowstate.WorkflowStatusInterrupted:
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func headTailExcerpt(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	headLimit := limit / 2
	tailLimit := limit - headLimit
	return value[:headLimit] + "\n[truncated middle]\n" + value[len(value)-tailLimit:]
}
