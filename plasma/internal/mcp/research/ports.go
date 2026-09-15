package research

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/researchcatalog"
	"github.com/c86j224s/liquid2/plasma/internal/researchinspection"
	"github.com/c86j224s/liquid2/plasma/internal/researchproposal"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

// Reader is the consumer-side port required by the current research read tools.
type Reader interface {
	OutlineMission(context.Context, string) (app.ResearchIDEOutline, error)
	ListMissionChanges(context.Context, app.ResearchIDEChangesRequest) (app.ResearchIDEChanges, error)
	ListMissionObjects(context.Context, string, string, int, string) (researchcatalog.Page, error)
	ReadMissionObject(context.Context, researchinspection.ReadRequest) (researchinspection.ObjectRead, error)
	GrepMissionObjects(context.Context, string, string, int, string) (researchinspection.GrepResult, error)
	ListObjectReferences(context.Context, string, string, string, int, string) (researchcatalog.References, error)
}

// LegacyReader is the optional port used only when legacy research reads are
// explicitly enabled by the root MCP server.
type LegacyReader interface {
	OutlineMissionLegacy(context.Context, string) (app.ResearchIDEOutline, error)
	ListMissionObjectsLegacy(context.Context, string, string, int, string) (researchcatalog.Page, error)
	GrepMissionObjectsLegacy(context.Context, string, string, int, string) (researchinspection.GrepResult, error)
	ListObjectReferencesLegacy(context.Context, string, string, string, int, string) (researchcatalog.References, error)
}

// ProposalWriter is the consumer-side port required by legacy research proposal
// mutation tools. Idempotency and session binding are enforced by the root MCP
// server before these methods are called.
type ProposalWriter interface {
	CreateEvidenceProposal(context.Context, researchproposal.CreateEvidenceProposalRequest) (researchproposal.EvidenceProposalResult, error)
	CreateQuestionProposal(context.Context, researchproposal.CreateQuestionProposalRequest) (researchproposal.QuestionProposalResult, error)
	CreateClaimProposal(context.Context, researchproposal.CreateClaimProposalRequest) (researchproposal.ClaimProposalResult, error)
	UpdateClaimConfidence(context.Context, researchrecords.UpdateClaimConfidenceRequest) (ledger.Event, error)
	SubmitProposal(context.Context, app.SubmitProposalRequest) (app.SubmitProposalResult, error)
}
