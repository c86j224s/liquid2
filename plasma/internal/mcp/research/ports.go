package research

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// Reader is the consumer-side port required by the current research read tools.
type Reader interface {
	OutlineMission(context.Context, string) (app.ResearchIDEOutline, error)
	ListMissionChanges(context.Context, app.ResearchIDEChangesRequest) (app.ResearchIDEChanges, error)
	ListMissionObjects(context.Context, string, string, int, string) (app.ResearchIDEPage, error)
	ReadMissionObject(context.Context, app.ResearchIDEReadRequest) (app.ResearchIDEObjectRead, error)
	GrepMissionObjects(context.Context, string, string, int, string) (app.ResearchIDEGrepResult, error)
	ListObjectReferences(context.Context, string, string, string, int, string) (app.ResearchIDEReferences, error)
}

// LegacyReader is the optional port used only when legacy research reads are
// explicitly enabled by the root MCP server.
type LegacyReader interface {
	OutlineMissionLegacy(context.Context, string) (app.ResearchIDEOutline, error)
	ListMissionObjectsLegacy(context.Context, string, string, int, string) (app.ResearchIDEPage, error)
	GrepMissionObjectsLegacy(context.Context, string, string, int, string) (app.ResearchIDEGrepResult, error)
	ListObjectReferencesLegacy(context.Context, string, string, string, int, string) (app.ResearchIDEReferences, error)
}

// ProposalWriter is the consumer-side port required by legacy research proposal
// mutation tools. Idempotency and session binding are enforced by the root MCP
// server before these methods are called.
type ProposalWriter interface {
	CreateEvidenceProposal(context.Context, app.CreateEvidenceProposalRequest) (app.EvidenceProposalResult, error)
	CreateQuestionProposal(context.Context, app.CreateQuestionProposalRequest) (app.QuestionProposalResult, error)
	CreateClaimProposal(context.Context, app.CreateClaimProposalRequest) (app.ClaimProposalResult, error)
	UpdateClaimConfidence(context.Context, app.UpdateClaimConfidenceRequest) (app.LedgerEvent, error)
	SubmitProposal(context.Context, app.SubmitProposalRequest) (app.SubmitProposalResult, error)
}
