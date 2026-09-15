package mcp

import uploadsource "github.com/c86j224s/liquid2/plasma/internal/source"

import (
	experimenthandler "github.com/c86j224s/liquid2/plasma/internal/mcp/reportexperiment"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/reportil"
	patchhandler "github.com/c86j224s/liquid2/plasma/internal/mcp/reportpatch"
	sourcehandler "github.com/c86j224s/liquid2/plasma/internal/mcp/source"
	candidatehandler "github.com/c86j224s/liquid2/plasma/internal/mcp/sourcecandidate"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
)

type commonMutatingInput = CommonMutatingInput

// CommonMutatingInput는 MCP transport 경계에 전달되는 요청 값이다.
type CommonMutatingInput = wire.CommonMutatingInput

type reportILSourcesListInput = reportil.ReportILSourcesListInput
type reportILSourcesListOutput = reportil.ReportILSourcesListOutput
type reportILSourceListItem = reportil.ReportILSourceListItem
type reportILSourcesReadInput = reportil.ReportILSourcesReadInput
type reportILSourceQuoteInput = reportil.ReportILSourceQuoteInput
type reportILSourceQuoteOutput = reportil.ReportILSourceQuoteOutput
type reportILSourcesReadOutput = reportil.ReportILSourcesReadOutput
type reportILSourceBatchItem = reportil.ReportILSourceBatchItem
type reportILSourcesBatchReadOutput = reportil.ReportILSourcesBatchReadOutput

type reportILEditorialMemoryStartInput = reportil.ReportILEditorialMemoryStartInput
type reportILEditorialMemoryAppendInput = reportil.ReportILEditorialMemoryAppendInput
type reportILEditorialMemoryReadInput = reportil.ReportILEditorialMemoryReadInput
type reportILEditorialMemoryFinalizeInput = reportil.ReportILEditorialMemoryFinalizeInput
type reportILEditorialMemoryStateOutput = reportil.ReportILEditorialMemoryStateOutput
type reportILEditorialMemoryReadOutput = reportil.ReportILEditorialMemoryReadOutput

type reportILDocumentStartInput = reportil.ReportILDocumentStartInput
type reportILDocumentOpenInput = reportil.ReportILDocumentOpenInput
type reportILDocumentAppendInput = reportil.ReportILDocumentAppendInput
type reportILDocumentAppendSourceInput = reportil.ReportILDocumentAppendSourceInput
type reportILDocumentTableInput = reportil.ReportILDocumentTableInput
type reportILDocumentTableRowInput = reportil.ReportILDocumentTableRowInput
type reportILDocumentReadInput = reportil.ReportILDocumentReadInput
type reportILDocumentReplaceInput = reportil.ReportILDocumentReplaceInput
type reportILDocumentEditTextInput = reportil.ReportILDocumentEditTextInput
type reportILDocumentReviseBlockInput = reportil.ReportILDocumentReviseBlockInput
type reportILDocumentFinalizeInput = reportil.ReportILDocumentFinalizeInput
type reportILDocumentStateOutput = reportil.ReportILDocumentStateOutput
type reportILDocumentReadOutput = reportil.ReportILDocumentReadOutput
type reportILLongFormDocumentStartInput = reportil.ReportILLongFormDocumentStartInput
type reportILLongFormDocumentAppendInput = reportil.ReportILLongFormDocumentAppendInput
type reportILLongFormDocumentCorrectBlockInput = reportil.ReportILLongFormDocumentCorrectBlockInput
type reportILLongFormBlockReplacementInput = reportil.ReportILLongFormBlockReplacementInput
type reportILDocumentEquationInput = reportil.ReportILDocumentEquationInput

type sourcesListInput = sourcehandler.SourcesListInput

type sourcesListOutput = sourcehandler.SourcesListOutput

type sourcesReadInput = sourcehandler.SourcesReadInput

type sourcesReadOutput = sourcehandler.SourcesReadOutput

type sourceExtractionOutput = wire.SourceExtractionOutput

type sourcesTreeInput = sourcehandler.SourcesTreeInput

type sourcesTreeOutput = sourcehandler.SourcesTreeOutput

type sourcesGrepInput = sourcehandler.SourcesGrepInput

type sourcesGrepOutput = sourcehandler.SourcesGrepOutput

type mediaSourceReadOutput = sourcehandler.MediaSourceReadOutput

type sourcesSearchInput = sourcehandler.SourcesSearchInput

type sourcesSearchOutput = sourcehandler.SourcesSearchOutput

type sourceCandidatesProposeInput = candidatehandler.SourceCandidatesProposeInput

type sourceCandidateProposalInput = candidatehandler.SourceCandidateProposalInput

type sourceCandidatesProposeOutput = candidatehandler.SourceCandidatesProposeOutput

type sourceCandidateProposalEvent = candidatehandler.SourceCandidateProposalEvent

type sourceCandidateStagingOutput = candidatehandler.SourceCandidateStagingOutput

type sourceCandidatesReadInput = candidatehandler.SourceCandidatesReadInput

type sourceCandidatesReadOutput = candidatehandler.SourceCandidatesReadOutput

type mermaidValidateInput struct {
	MissionID string `json:"mission_id"`
	Source    string `json:"source"`
}

type experimentReportCreateInput = experimenthandler.ExperimentReportCreateInput

type experimentReportAppendInput = experimenthandler.ExperimentReportAppendInput

type experimentReportReadInput = experimenthandler.ExperimentReportReadInput

type experimentReportFinalizeInput = experimenthandler.ExperimentReportFinalizeInput

type reportPatchStartInput = patchhandler.ReportPatchStartInput

type reportPatchReadInput = patchhandler.ReportPatchReadInput

type reportPatchApplyInput = patchhandler.ReportPatchApplyInput

type reportPatchFinalizeInput = patchhandler.ReportPatchFinalizeInput

type experimentReportDraftOutput = experimenthandler.ExperimentReportDraftOutput

type experimentReportReadOutput = experimenthandler.ExperimentReportReadOutput

type experimentReportFinalizeOutput = experimenthandler.ExperimentReportFinalizeOutput

type reportPatchOutput = patchhandler.ReportPatchOutput

type reportPatchReadOutput = patchhandler.ReportPatchReadOutput

type reportPatchFinalizeOutput = patchhandler.ReportPatchFinalizeOutput

type experimentReportHumanizeReadyOutput = experimenthandler.ExperimentReportHumanizeReadyOutput

type sourcesSnapshotInput = sourcehandler.SourcesSnapshotInput

type sourcesSnapshotOutput = sourcehandler.SourcesSnapshotOutput

type localPathRootsInput = sourcehandler.LocalPathRootsInput

type localPathRootsOutput = sourcehandler.LocalPathRootsOutput

type localPathTreeInput = sourcehandler.LocalPathTreeInput

type localPathTreeOutput = sourcehandler.LocalPathTreeOutput

type localPathAttachInput = sourcehandler.LocalPathAttachInput

type localPathAttachOutput = sourcehandler.LocalPathAttachOutput

type sourceRemoveInput = sourcehandler.SourceRemoveInput

type sourceRestoreInput = sourcehandler.SourceRestoreInput

type sourceStateChangeOutput = sourcehandler.SourceStateChangeOutput

type connectorRefInput = sourcehandler.ConnectorRefInput

func rawArtifactFromApp(artifact artifactcontract.Raw) rawArtifactOutput {
	output := rawArtifactOutput{
		ArtifactID: artifact.ArtifactID,
		MissionID:  artifact.MissionID,
		MediaType:  artifact.MediaType,
		ByteSize:   artifact.ByteSize,
		SHA256:     artifact.SHA256,
		StorageURI: artifact.StorageURI,
		Filename:   artifact.Filename,
		ReadKind:   uploadsource.UploadedArtifactReadKind(artifact),
	}
	if !artifact.CreatedAt.IsZero() {
		output.CreatedAt = artifact.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return output
}

type sourceSnapshotOutput = sourcehandler.SourceSnapshotOutput

type matchedRangeOutput = sourcehandler.MatchedRangeOutput

type sourceCandidateOutput = sourcehandler.SourceCandidateOutput

type contentRangeInput = sourcehandler.ContentRangeInput

type rawArtifactOutput = wire.RawArtifactOutput
