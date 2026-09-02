package reportilphase0

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

// PromoteReaderArtifact validates one already-finalized publication workspace
// against a final-author checkpoint and returns a reader-stage checkpoint.
func PromoteReaderArtifact(
	ctx context.Context,
	missionObjective string,
	checkpoint reportilcontract.ResumeCheckpoint,
	readerSessionID string,
	documents interface {
		ReadReportILPublicationDocument(context.Context, string, string, reportilcontract.SourceCatalog) (reportilcontract.AuthorDocument, reportilcontract.AuthorWorkspaceReceipt, error)
	},
) (*reportilcontract.ResumeCheckpoint, error) {
	authored, workspace, err := documents.ReadReportILPublicationDocument(ctx, checkpoint.AuthorCatalog.MissionID, readerSessionID, checkpoint.AuthorCatalog)
	if err != nil {
		return nil, err
	}
	_, original, err := compileLongFormAuthorDocument(
		checkpoint.AuthorDocument, "narrative_promote", "doc_promote", missionObjective,
		checkpoint.AuthorCatalog, editorialMemorySourceReadReceipt(checkpoint.AuthorCatalog), nil,
	)
	if err != nil {
		return nil, err
	}
	if _, err := compilePublicationAuthorDocument(authored, original, AuthoringModeLongForm, missionObjective, checkpoint.AuthorCatalog, nil); err != nil {
		return nil, err
	}
	if workspace.Stage != "il_reader" {
		return nil, fmt.Errorf("publication workspace stage differs")
	}
	promoted := checkpoint
	promoted.Stage = "il_reader"
	promoted.ArtifactID = workspace.ArtifactID
	promoted.AuthorWorkspace = workspace
	promoted.AuthorDocument = authored
	artifact := reportilcontract.CheckpointArtifact{ArtifactID: workspace.ArtifactID, SHA256: workspace.SHA256, ByteSize: workspace.ByteSize, Stage: workspace.Stage}
	promoted.LongFormAuthoring.Final = artifact
	promoted.LongFormAuthoring.Finalizations = append(promoted.LongFormAuthoring.Finalizations, reportilcontract.CheckpointFinalization{Stage: "il_reader", Artifact: artifact})
	promoted.ReaderFinalization = reportilcontract.CheckpointReaderFinalization{PublicationPatches: workspace.Replacements, ProposedPatches: workspace.Replacements, AcceptedPatches: workspace.Replacements, Applied: workspace.Replacements > 0}
	if err := reportilcontract.ValidateProductCheckpoint(promoted.ProductCheckpoint, checkpoint.AuthorCatalog.MissionID); err != nil {
		return nil, err
	}
	return &promoted, nil
}
