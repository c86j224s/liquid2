package workflow

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidates"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

func (runner Runner) appendSourceCandidateEvent(ctx context.Context, view workflowstate.WorkflowRunView, userEventID string, agentEventID string, stepID string, text string) error {
	candidates := sourceCandidatesFromText(text)
	if len(candidates) == 0 {
		return nil
	}
	proposals := make([]sourcecandidates.WorkflowSourceCandidateProposal, 0, len(candidates))
	for _, candidate := range candidates {
		proposals = append(proposals, sourcecandidates.WorkflowSourceCandidateProposal{
			URL:    candidate.URL,
			Title:  candidate.Title,
			Reason: candidate.Reason,
			State:  candidate.State,
		})
	}
	eventReq, ok, err := sourcecandidates.BuildWorkflowSourceCandidateProposalEventRequest(sourcecandidates.WorkflowSourceCandidateProposalEventRequest{
		EventID:        runner.newID("evt"),
		MissionID:      view.MissionID,
		WorkflowRunID:  view.WorkflowRunID,
		WorkflowStepID: stepID,
		UserEventID:    userEventID,
		AgentEventID:   agentEventID,
		Producer:       ledger.Producer{Type: "agent", ID: view.AgentExecutor},
		Candidates:     proposals,
	})
	if err != nil || !ok {
		return err
	}
	event, err := runner.Service.AppendEvent(ctx, eventReq)
	if err != nil {
		return err
	}
	if runner.SourceCandidateStager != nil {
		runner.SourceCandidateStager(context.Background(), event)
	}
	return nil
}

type sourceCandidate struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
	State  string `json:"state"`
}

func sourceCandidatesFromText(text string) []sourceCandidate {
	parsed := sourcecandidates.Parse(text)
	if len(parsed) == 0 {
		return nil
	}
	candidates := make([]sourceCandidate, 0, len(parsed))
	for _, candidate := range parsed {
		candidates = append(candidates, sourceCandidate{
			URL:    candidate.URL,
			Title:  candidate.Title,
			Reason: candidate.Reason,
			State:  candidate.State,
		})
	}
	return candidates
}
