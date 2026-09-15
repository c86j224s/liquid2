package sourcecandidate

import "github.com/c86j224s/liquid2/plasma/internal/mcp/wire"

type SourceCandidatesProposeInput struct {
	wire.CommonMutatingInput
	Candidates []SourceCandidateProposalInput `json:"candidates"`
}

type SourceCandidateProposalInput struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
}

type SourceCandidatesProposeOutput struct {
	EventID    string                         `json:"event_id"`
	Candidates []SourceCandidateProposalEvent `json:"candidates"`
	Staging    []SourceCandidateStagingOutput `json:"staging,omitempty"`
}

type SourceCandidateProposalEvent struct {
	URL    string `json:"url"`
	Title  string `json:"title,omitempty"`
	Reason string `json:"reason"`
	State  string `json:"state"`
}

type SourceCandidateStagingOutput struct {
	URL             string `json:"url"`
	ProposalEventID string `json:"proposal_event_id"`
	StagingEventID  string `json:"staging_event_id,omitempty"`
	StagingState    string `json:"staging_state"`
	Message         string `json:"message"`
}

type SourceCandidatesReadInput struct {
	MissionID       string `json:"mission_id"`
	URL             string `json:"url"`
	ProposalEventID string `json:"proposal_event_id"`
	StagingEventID  string `json:"staging_event_id"`
	ArtifactID      string `json:"artifact_id"`
	Offset          int    `json:"offset"`
	MaxBytes        int    `json:"max_bytes"`
}

type SourceCandidatesReadOutput struct {
	ApprovalState      string                       `json:"approval_state"`
	NotReportDefault   bool                         `json:"not_report_default"`
	CandidateURL       string                       `json:"candidate_url,omitempty"`
	ProposalEventID    string                       `json:"proposal_event_id,omitempty"`
	StagingEventID     string                       `json:"staging_event_id,omitempty"`
	StagingState       string                       `json:"staging_state"`
	Message            string                       `json:"message,omitempty"`
	FailureMessage     string                       `json:"failure_message,omitempty"`
	Artifact           *wire.RawArtifactOutput      `json:"artifact,omitempty"`
	Content            string                       `json:"content,omitempty"`
	Offset             int                          `json:"offset,omitempty"`
	NextOffset         int                          `json:"next_offset,omitempty"`
	ContentLength      int                          `json:"content_length,omitempty"`
	ContentLengthKnown bool                         `json:"content_length_known,omitempty"`
	Truncated          bool                         `json:"truncated,omitempty"`
	Extraction         *wire.SourceExtractionOutput `json:"extraction,omitempty"`
}
