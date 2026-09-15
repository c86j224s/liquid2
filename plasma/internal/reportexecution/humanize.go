package reportexecution

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// StartHumanize retains a fail-closed compatibility entrypoint; no work is enqueued.
func (runner Runner) StartHumanize(ctx context.Context, missionID string, req HumanizeRequest, producer ledger.Producer) (ledger.Event, error) {
	return ledger.Event{}, fmt.Errorf("%w: deprecated post-canonical H5 has been removed", producterror.ErrInvalidInput)
}

// ResumeHumanize closes historical work rather than replaying the removed feature.
func (runner Runner) ResumeHumanize(ctx context.Context, missionID string, pending ledger.Event) error {
	return runner.RetireHumanizePending(ctx, missionID, pending)
}

// RunHumanize never launches a provider for a retired H5 operation.
func (runner Runner) RunHumanize(ctx context.Context, missionID string, req HumanizeRequest, pendingEventID string) error {
	_, err := runner.AppendHumanizeFailed(ctx, missionID, pendingEventID, req.AgentExecutor, req.SourceArtifactID, req.ReportMode, humanizeRetiredError{})
	return err
}
