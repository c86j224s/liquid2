package reportilphase0

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentcapability"
	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/pdfdocument"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

// ProductConfig configures one request-local IL product run. The provider is
// called only with the isolated codex request contract.
type ProductConfig struct {
	MissionID         string
	MissionObjective  string
	Title             string
	Direction         string
	TargetLanguage    string
	AuthoringMode     string
	ValidationProfile string
	ChromePath        string
	NewID             func(string) string
	Sources           SourceReader
	Provider          Provider
	Progress          func(stage, status string) error
	LongFormProgress  func(LongFormProgressEvent) error
	PendingEventID    string
	VerifySourceRead  agentexec.ReportILSourceReadVerifier
	AuthorDocuments   agentexec.ReportILAuthorDocumentReader
	ImageFetch        ProductImageFetcher
	Resume            *reportilcontract.ResumeCheckpoint
	Checkpoint        func(reportilcontract.ProductCheckpoint) error
	flowReadSpans     []reportilcontract.SourceReadSpan
}

type LongFormProgressEvent struct {
	Kind         string
	Status       string
	Plan         *reportilcontract.LongFormPlan
	PartIndex    int
	SectionIndex int
	Title        string
}

// ProductArtifact is one durable logical output before app persistence.
type ProductArtifact struct {
	ID        string
	Kind      string
	MediaType string
	Content   []byte
	Role      string
	Filename  string
	SHA256    string
	ByteSize  int
	AssetID   string
}

// ProductBundle is the complete core output, optional image artifacts, and final lineage.
type ProductBundle struct {
	Artifacts          []ProductArtifact
	Narrative          Narrative
	Document           Document
	Markdown           []byte
	HTML               []byte
	PDF                []byte
	ProjectionSHA      string
	CatalogSHA         string
	SourceSelection    SourceSelectionReceipt
	ManifestContent    []byte
	Manifest           ProductManifest
	Usage              TerminalUsageReceipt
	ReaderFinalization ReaderFinalizationReceipt
}

// ProviderUsageReceipt is a safe product-stage receipt. It excludes provider
// responses, prompts, logs, and session identifiers.
type ProviderUsageReceipt struct {
	Stage             string                `json:"stage"`
	Usage             agentusage.AgentUsage `json:"usage"`
	UsageUnavailable  bool                  `json:"usage_unavailable"`
	UnavailableReason string                `json:"unavailable_reason,omitempty"`
}

func (r ProviderUsageReceipt) safe() ProviderUsageReceipt {
	r.Usage.Prompt = agentusage.PromptMetrics{}
	r.Usage.Session = agentusage.SessionMetrics{}
	return r
}

// TerminalUsageReceipt is the truthful aggregate across all provider stages
// used by this run. Missing provider usage remains explicitly unavailable.
type TerminalUsageReceipt struct {
	Stages            []ProviderUsageReceipt `json:"stages"`
	DurationMS        int64                  `json:"duration_ms"`
	UsageUnavailable  bool                   `json:"usage_unavailable"`
	UnavailableReason string                 `json:"unavailable_reason,omitempty"`
}

func (r *TerminalUsageReceipt) addResult(stage string, result agentexec.AgentResult) {
	r.add(stage, result)
}

func (r *TerminalUsageReceipt) add(stage string, results ...agentexec.AgentResult) {
	for _, result := range results {
		receipt := ProviderUsageReceipt{Stage: stage, Usage: result.Usage}
		if result.Usage.ProviderUsage == nil {
			receipt.UsageUnavailable = true
			receipt.UnavailableReason = "provider usage was not emitted"
		}
		r.Stages = append(r.Stages, receipt.safe())
		r.DurationMS += result.Usage.DurationMS
		if receipt.UsageUnavailable {
			r.UsageUnavailable = true
			if r.UnavailableReason == "" {
				r.UnavailableReason = receipt.UnavailableReason
			}
		}
	}
}

func (r TerminalUsageReceipt) normalized() TerminalUsageReceipt {
	for i := range r.Stages {
		r.Stages[i] = r.Stages[i].safe()
	}
	return r
}

func (r TerminalUsageReceipt) failureAttempts() []reportexecution.ProviderAttemptUsageReceipt {
	attempts := make([]reportexecution.ProviderAttemptUsageReceipt, 0, len(r.Stages))
	stageAttempts := map[string]int{}
	for _, stage := range r.normalized().Stages {
		stageAttempts[stage.Stage]++
		attempts = append(attempts, reportexecution.NewProviderAttemptUsageReceipt(stage.Stage, stageAttempts[stage.Stage], stage.Usage))
	}
	return attempts
}

type flowResponse struct {
	Document    Document        `json:"document"`
	Attestation FlowAttestation `json:"flow_attestation"`
}

// RunProduct freezes a source catalog, authors one complete report, compiles it
// into target-neutral IL, and renders all required targets without archive
// filesystem use or a second prose rewrite.
func RunProduct(ctx context.Context, config ProductConfig) (ProductBundle, error) {
	if config.Provider == nil || config.Sources == nil || config.NewID == nil || config.VerifySourceRead == nil || config.AuthorDocuments == nil || !strings.HasPrefix(strings.TrimSpace(config.PendingEventID), "evt_") {
		return ProductBundle{}, fmt.Errorf("report IL product dependencies are incomplete")
	}
	config.AuthoringMode = normalizeAuthoringMode(config.AuthoringMode)
	config.ValidationProfile = normalizeValidationProfile(config.ValidationProfile)
	config.TargetLanguage = strings.TrimSpace(config.TargetLanguage)
	if config.TargetLanguage == "" {
		config.TargetLanguage = "ko"
	}
	stagePlan := validationPlan(config.ValidationProfile)
	if config.Resume != nil && config.Resume.Stage == "il_long_form_final" {
		// A failed publication pass must restart with a fresh reader receipt. Earlier
		// accepted publication/continuity edits belong to a later checkpoint.
		config.Resume.ReaderFinalization = reportilcontract.CheckpointReaderFinalization{}
	}
	usage := TerminalUsageReceipt{}
	progress := func(stage, status string) error {
		if config.Progress == nil {
			return nil
		}
		return config.Progress(stage, status)
	}
	checkpoint := func(value reportilcontract.ProductCheckpoint) error {
		if config.Checkpoint == nil {
			return nil
		}
		return config.Checkpoint(value)
	}
	if config.Resume != nil && config.AuthoringMode != AuthoringModeLongForm {
		return ProductBundle{}, fmt.Errorf("report IL checkpoint resume requires long-form authoring")
	}
	if err := progress("source_packet", "started"); err != nil {
		return ProductBundle{}, reportexecution.NewStageFailure("source_packet", "", -1, -1, err)
	}
	catalogBuild, err := BuildSourceCatalogForSelection(ctx, config.Sources, config.MissionID)
	if err != nil {
		return ProductBundle{}, reportexecution.NewStageFailure("source_packet", "", -1, -1, err)
	}
	if err := progress("source_packet", "completed"); err != nil {
		return ProductBundle{}, reportexecution.NewStageFailure("source_packet", "", -1, -1, err)
	}

	var selection SourceSelectionResult
	var sourceSelection SourceSelectionReceipt
	var authorCatalog reportilcontract.SourceCatalog
	var memory reportilcontract.EditorialMemory
	var memoryReceipt reportilcontract.EditorialMemoryReceipt
	var narrative Narrative
	var document Document
	var authorWorkspace reportilcontract.AuthorWorkspaceReceipt
	var longFormAuthoring LongFormAuthoringReceipt
	if config.Resume != nil {
		resume := *config.Resume
		if err := reportilcontract.ValidateProductCheckpoint(resume.ProductCheckpoint, config.MissionID); err != nil {
			return ProductBundle{}, reportexecution.NewStageFailure("source_packet", "", -1, -1, err)
		}
		if resume.CandidateCatalogSHA256 != catalogBuild.Catalog.SHA256 {
			return ProductBundle{}, reportexecution.NewStageFailure("source_packet", "", -1, -1, fmt.Errorf("report IL checkpoint source catalog changed"))
		}
		selection = SourceSelectionResult{Catalog: resume.AuthorCatalog, AuthorCatalog: resume.AuthorCatalog}
		sourceSelection = sourceSelectionFromCheckpoint(resume.SourceSelection)
		authorCatalog = resume.AuthorCatalog
		memory = resume.EditorialMemory
		memoryReceipt = resume.ProductCheckpoint.EditorialMemory.Receipt()
		authorWorkspace = resume.AuthorWorkspace
		longFormAuthoring = longFormFromCheckpoint(resume.LongFormAuthoring)
		if resume.Stage == "il_long_form_parts" {
			plan := resume.LongFormPlan
			partArtifacts := make(map[string]reportilcontract.AuthorWorkspaceReceipt, len(plan.Parts))
			for index, part := range plan.Parts {
				artifact := resume.LongFormAuthoring.PartArtifacts[index]
				partArtifacts[part.PartKey] = reportilcontract.AuthorWorkspaceReceipt{ArtifactID: artifact.ArtifactID, SHA256: artifact.SHA256, ByteSize: artifact.ByteSize, Stage: artifact.Stage}
			}
			planReceipt := reportilcontract.LongFormPlanReceipt{ArtifactID: resume.LongFormAuthoring.Plan.ArtifactID, SHA256: resume.LongFormAuthoring.Plan.SHA256, ByteSize: resume.LongFormAuthoring.Plan.ByteSize, Parts: resume.LongFormAuthoring.Parts, Sections: resume.LongFormAuthoring.Sections}
			if err := progress("il_long_form_final", "started"); err != nil {
				return ProductBundle{}, reportexecution.NewStageFailure("il_long_form_final", "", -1, -1, err)
			}
			authored, workspace, finalResult, edits, finalErr := runLongFormFinalEditStage(ctx, config, authorCatalog, memory, memoryReceipt, plan, planReceipt, partArtifacts)
			usage.addResult("il_long_form_final", finalResult)
			if finalErr != nil {
				return ProductBundle{}, providerStageFailure("il_long_form_final", finalErr, usage)
			}
			authorWorkspace = workspace
			longFormAuthoring.FinalEdits = edits
			appendLongFormFinalization(&longFormAuthoring, "il_long_form_final", authorWorkspace)
			if err := progress("il_long_form_final", "completed"); err != nil {
				return ProductBundle{}, reportexecution.NewStageFailure("il_long_form_final", "", -1, -1, err)
			}
			narrative, document, err = compileLongFormAuthorDocument(authored, config.NewID("narrative"), config.NewID("doc"), config.MissionObjective, authorCatalog, editorialMemorySourceReadReceipt(authorCatalog), catalogBuild.citationCatalog(authorCatalog))
			if err != nil {
				return ProductBundle{}, semanticProviderStageFailure("il_long_form_final", err, usage)
			}
			for _, stage := range []string{"il_source_selection", "il_editorial_memory", "il_narrative", "il_long_form_plan", "il_long_form_sections", "il_long_form_parts"} {
				if err := progress(stage, "reused"); err != nil {
					return ProductBundle{}, reportexecution.NewStageFailure(stage, "", -1, -1, err)
				}
			}
		} else {
			if err := validateLongFormDocumentBindings(resume.AuthorDocument, resume.EditorialMemory, resume.AuthorCatalog, true); err != nil {
				return ProductBundle{}, reportexecution.NewStageFailure("il_long_form_final", "", -1, -1, err)
			}
			narrative, document, err = compileLongFormAuthorDocument(resume.AuthorDocument, config.NewID("narrative"), config.NewID("doc"), config.MissionObjective, authorCatalog, editorialMemorySourceReadReceipt(authorCatalog), catalogBuild.citationCatalog(authorCatalog))
			if err != nil {
				return ProductBundle{}, semanticProviderStageFailure("il_long_form_final", err, usage)
			}
			for _, stage := range []string{"il_source_selection", "il_editorial_memory", "il_narrative", "il_long_form_plan", "il_long_form_sections", "il_long_form_parts", "il_long_form_final"} {
				if err := progress(stage, "reused"); err != nil {
					return ProductBundle{}, reportexecution.NewStageFailure(stage, "", -1, -1, err)
				}
			}
		}
		if resume.Stage == "il_reader" {
			if err := progress("il_reader", "reused"); err != nil {
				return ProductBundle{}, reportexecution.NewStageFailure("il_reader", "", -1, -1, err)
			}
		}
		if resume.Stage == "il_continuity" {
			for _, stage := range []string{"il_reader", "il_continuity"} {
				if err := progress(stage, "reused"); err != nil {
					return ProductBundle{}, reportexecution.NewStageFailure(stage, "", -1, -1, err)
				}
			}
		}
	} else {
		selectionNeeded := catalogBuild.HasExclusions() || !sourceCatalogFitsSelectionTarget(catalogBuild.Catalog)
		if selectionNeeded {
			if err := progress("il_source_selection", "started"); err != nil {
				return ProductBundle{}, reportexecution.NewStageFailure("il_source_selection", "", -1, -1, err)
			}
		}
		var selectionResults []agentexec.AgentResult
		selection, sourceSelection, selectionResults, err = selectSourceCatalog(ctx, config, catalogBuild)
		usage.add("il_source_selection", selectionResults...)
		if err != nil {
			return ProductBundle{}, providerStageFailure("il_source_selection", err, usage)
		}
		if selectionNeeded {
			if err := progress("il_source_selection", "completed"); err != nil {
				return ProductBundle{}, reportexecution.NewStageFailure("il_source_selection", "", -1, -1, err)
			}
		}
		authorCatalog = selection.AuthorCatalog
		if stagePlan.EditorialMemory {
			if err := progress("il_editorial_memory", "started"); err != nil {
				return ProductBundle{}, reportexecution.NewStageFailure("il_editorial_memory", "", -1, -1, err)
			}
			var memoryResults []agentexec.AgentResult
			memory, memoryReceipt, memoryResults, err = runEditorialMemoryStage(ctx, config, authorCatalog)
			usage.add("il_editorial_memory", memoryResults...)
			if err != nil {
				return ProductBundle{}, providerStageFailure("il_editorial_memory", err, usage)
			}
			if err := progress("il_editorial_memory", "completed"); err != nil {
				return ProductBundle{}, reportexecution.NewStageFailure("il_editorial_memory", "", -1, -1, err)
			}
		}
		if err := progress("il_narrative", "started"); err != nil {
			return ProductBundle{}, reportexecution.NewStageFailure("il_narrative", "", -1, -1, err)
		}
		citationCatalog := catalogBuild.citationCatalog(authorCatalog)
		var authorResults []agentexec.AgentResult
		if config.AuthoringMode == AuthoringModeLongForm {
			var longFormResults []longFormAgentResult
			partsCompleted := func(planReceipt reportilcontract.LongFormPlanReceipt, sectionArtifacts, partArtifacts []reportilcontract.AuthorWorkspaceReceipt, parts, sections int) error {
				if !stagePlan.EditorialMemory {
					return nil
				}
				return checkpoint(NewPartsCheckpoint(
					config, catalogBuild.Catalog.SHA256, authorCatalog, catalogBuild.ImageCatalog,
					sourceSelection, memoryReceipt, planReceipt, sectionArtifacts, partArtifacts, parts, sections,
				))
			}
			narrative, document, authorWorkspace, longFormAuthoring, longFormResults, err = runLongFormAuthoringGraph(
				ctx, config, authorCatalog, memory, memoryReceipt, citationCatalog, sourceSelection.Applied, progress, partsCompleted,
			)
			for _, stageResult := range longFormResults {
				usage.addResult(stageResult.Stage, stageResult.Result)
			}
		} else if stagePlan.EditorialMemory {
			narrative, document, authorWorkspace, authorResults, err = runReportFirstAuthorStage(
				ctx, config, selection.Catalog, memory, memoryReceipt, citationCatalog, sourceSelection.Applied,
			)
			usage.add("il_narrative", authorResults...)
		} else {
			narrative, document, authorWorkspace, authorResults, err = runDirectSourceAuthorStage(
				ctx, config, selection.Catalog, citationCatalog, sourceSelection.Applied,
			)
			usage.add("il_narrative", authorResults...)
		}
		if err != nil {
			var longFormFailure *longFormStageError
			if errors.As(err, &longFormFailure) {
				return ProductBundle{}, providerStageFailure(longFormFailure.Stage, longFormFailure.Cause, usage)
			}
			return ProductBundle{}, providerStageFailure("il_narrative", err, usage)
		}
		if err := progress("il_narrative", "completed"); err != nil {
			return ProductBundle{}, reportexecution.NewStageFailure("il_narrative", "", -1, -1, err)
		}
	}
	if err := validateCompleteSourceCatalogBudget(selection.Catalog); err != nil {
		return ProductBundle{}, reportexecution.NewStageFailure("il_source_selection", "", -1, -1, err)
	}
	if err := validateCompleteSourceCatalogBudget(authorCatalog); err != nil {
		return ProductBundle{}, reportexecution.NewStageFailure("il_source_selection", "", -1, -1, err)
	}
	authorCatalog = selection.AuthorCatalog
	if config.Resume != nil {
		authorCatalog = config.Resume.AuthorCatalog
	}
	citationCatalog := catalogBuild.citationCatalog(authorCatalog)
	if config.AuthoringMode == AuthoringModeLongForm && config.Resume == nil {
		appendLongFormFinalization(&longFormAuthoring, "il_long_form_final", authorWorkspace)
	}
	if config.AuthoringMode == AuthoringModeLongForm && (config.Resume == nil || config.Resume.Stage == "il_long_form_parts") && stagePlan.EditorialMemory {
		checkpointValue := newProductCheckpoint(
			config, "il_long_form_final", catalogBuild.Catalog.SHA256, authorCatalog,
			catalogBuild.ImageCatalog, sourceSelection, memoryReceipt, authorWorkspace,
			longFormAuthoring, ReaderFinalizationReceipt{},
		)
		if err := checkpoint(checkpointValue); err != nil {
			return ProductBundle{}, reportexecution.NewStageFailure("il_long_form_final", "", -1, -1, err)
		}
	}

	readerFinalization := ReaderFinalizationReceipt{}
	if config.Resume != nil {
		readerFinalization = readerFromCheckpoint(config.Resume.ReaderFinalization)
	}
	publicationWorkspace := authorWorkspace
	if stagePlan.PublicationRead && (config.Resume == nil || config.Resume.Stage == "il_long_form_parts" || config.Resume.Stage == "il_long_form_final") {
		if err := progress("il_reader", "started"); err != nil {
			return ProductBundle{}, reportexecution.NewStageFailure("il_reader", "", -1, -1, err)
		}
		var publicationFinalization ReaderFinalizationReceipt
		var readerResults []agentexec.AgentResult
		document, publicationFinalization, publicationWorkspace, readerResults, err = runReaderPatchStage(
			ctx, config, authorCatalog, memory, memoryReceipt, document, citationCatalog, authorWorkspace,
		)
		usage.add("il_reader", readerResults...)
		if err != nil {
			return ProductBundle{}, providerStageFailure("il_reader", err, usage)
		}
		readerFinalization = mergeReaderFinalization(readerFinalization, publicationFinalization)
		if config.AuthoringMode == AuthoringModeLongForm {
			appendLongFormFinalization(
				&longFormAuthoring,
				"il_reader",
				publicationWorkspace,
			)
		}
		if err := progress("il_reader", "completed"); err != nil {
			return ProductBundle{}, reportexecution.NewStageFailure("il_reader", "", -1, -1, err)
		}
		if config.AuthoringMode == AuthoringModeLongForm && stagePlan.EditorialMemory {
			checkpointValue := newProductCheckpoint(
				config, "il_reader", catalogBuild.Catalog.SHA256, authorCatalog,
				catalogBuild.ImageCatalog, sourceSelection, memoryReceipt, publicationWorkspace,
				longFormAuthoring, readerFinalization,
			)
			if err := checkpoint(checkpointValue); err != nil {
				return ProductBundle{}, reportexecution.NewStageFailure("il_reader", "", -1, -1, err)
			}
		}
	}

	if stagePlan.Continuity && (config.Resume == nil || config.Resume.Stage != "il_continuity") {
		if err := progress("il_continuity", "started"); err != nil {
			return ProductBundle{}, reportexecution.NewStageFailure("il_continuity", "", -1, -1, err)
		}
		var continuityFinalization ReaderFinalizationReceipt
		var continuityWorkspace reportilcontract.AuthorWorkspaceReceipt
		var continuityResults []agentexec.AgentResult
		document, continuityFinalization, continuityWorkspace, continuityResults, err = runContinuityPatchStage(
			ctx, config, authorCatalog, memory, memoryReceipt, document, citationCatalog, publicationWorkspace,
		)
		usage.add("il_continuity", continuityResults...)
		if err != nil {
			return ProductBundle{}, providerStageFailure("il_continuity", err, usage)
		}
		readerFinalization = mergeReaderFinalization(readerFinalization, continuityFinalization)
		if config.AuthoringMode == AuthoringModeLongForm {
			appendLongFormFinalization(
				&longFormAuthoring,
				"il_continuity",
				continuityWorkspace,
			)
		}
		if err := progress("il_continuity", "completed"); err != nil {
			return ProductBundle{}, reportexecution.NewStageFailure("il_continuity", "", -1, -1, err)
		}
		if config.AuthoringMode == AuthoringModeLongForm && stagePlan.EditorialMemory {
			checkpointValue := newProductCheckpoint(
				config, "il_continuity", catalogBuild.Catalog.SHA256, authorCatalog,
				catalogBuild.ImageCatalog, sourceSelection, memoryReceipt, continuityWorkspace,
				longFormAuthoring, readerFinalization,
			)
			if err := checkpoint(checkpointValue); err != nil {
				return ProductBundle{}, reportexecution.NewStageFailure("il_continuity", "", -1, -1, err)
			}
		}
	}
	narrative, err = finalizeNarrativeFromDocument(
		narrative, document, config.MissionObjective, nil,
	)
	if err != nil {
		return ProductBundle{}, semanticProviderStageFailure("il_narrative", err, usage)
	}

	if err := progress("il_images", "started"); err != nil {
		return ProductBundle{}, reportexecution.NewStageFailure("il_images", "", -1, -1, err)
	}
	imageCandidates, err := collectProductImageCandidates(ctx, config.Sources, catalogBuild.ImageCatalog, config.ImageFetch)
	if err != nil {
		return ProductBundle{}, reportexecution.NewStageFailure("il_images", "", -1, -1, err)
	}
	var imageArtifacts []ProductArtifact
	var manifestImages []ManifestImage
	if len(imageCandidates) > 0 {
		placements, imageResults, placementErr := runImagePlacementStage(ctx, config, document, imageCandidates)
		usage.add("il_images", imageResults...)
		if placementErr != nil {
			return ProductBundle{}, providerStageFailure("il_images", placementErr, usage)
		}
		document, imageArtifacts, manifestImages, err = joinProductImages(document, catalogBuild.ImageCatalog, imageCandidates, placements.Placements, config.NewID)
		if err != nil {
			return ProductBundle{}, semanticProviderStageFailure("il_images", err, usage)
		}
	}
	if err := progress("il_images", "completed"); err != nil {
		return ProductBundle{}, reportexecution.NewStageFailure("il_images", "", -1, -1, err)
	}

	if err := progress("il_document", "started"); err != nil {
		return ProductBundle{}, reportexecution.NewStageFailure("il_document", "", -1, -1, err)
	}
	if err := validateReaderFinalizationReceipt(readerFinalization); err != nil {
		return ProductBundle{}, semanticProviderStageFailure("il_continuity", err, usage)
	}
	if err := validateProductDocument(narrative, document); err != nil {
		return ProductBundle{}, semanticProviderStageFailure("il_document", err, usage)
	}
	document.RevisionID = config.NewID("rev")
	bundle := Bundle{
		SchemaVersion: BundleSchemaVersion,
		Arm:           "T1",
		Narrative:     &narrative,
		Document:      document,
	}
	if err := ValidateBundle(bundle); err != nil {
		return ProductBundle{}, semanticProviderStageFailure("il_document", err, usage)
	}
	if err := progress("il_document", "completed"); err != nil {
		return ProductBundle{}, reportexecution.NewStageFailure("il_document", "", -1, -1, err)
	}
	narrativeBytes := mustMarshal(narrative)

	if err := progress("il_render", "started"); err != nil {
		return ProductBundle{}, reportexecution.NewStageFailure("il_render", "", -1, -1, err)
	}
	markdown, _, err := RenderMarkdown(document)
	if err != nil {
		return ProductBundle{}, reportexecution.NewStageFailure("il_render", "", -1, -1, err)
	}
	html, _, err := RenderHTML(document)
	if err != nil {
		return ProductBundle{}, reportexecution.NewStageFailure("il_render", "", -1, -1, err)
	}
	pdf, err := RenderPDF(ctx, html, config.ChromePath)
	if err != nil {
		return ProductBundle{}, reportexecution.NewStageFailure("il_render", "", -1, -1, err)
	}
	if _, err := pdfdocument.Inspect(pdf.Content); err != nil {
		return ProductBundle{}, reportexecution.NewStageFailure("il_render", "", -1, -1, err)
	}
	if err := progress("il_render", "completed"); err != nil {
		return ProductBundle{}, reportexecution.NewStageFailure("il_render", "", -1, -1, err)
	}

	// Build the core outputs and optional image artifacts first. The manifest
	// intentionally excludes its own entry; its hash and byte size are computed
	// only after serialization.
	manifestID := config.NewID("art")
	narrativeID, documentID := config.NewID("art"), config.NewID("art")
	markdownID, htmlID, pdfID := config.NewID("art"), config.NewID("art"), config.NewID("art")
	finalDocumentBytes := mustMarshal(document)
	manifest := ProductManifest{
		SchemaVersion: ProductManifestSchemaVersion, CompilerVersion: CompilerVersion, PipelineFamily: PipelineFamily,
		AuthoringMode: config.AuthoringMode, ValidationProfile: config.ValidationProfile,
		MissionID: config.MissionID, CatalogSHA256: authorCatalog.SHA256, Provider: "codex", Model: "gpt-5.6-luna", Effort: "xhigh",
		DocumentID: document.DocumentID, RevisionID: document.RevisionID,
		NarrativeSHA256: SHA256(narrativeBytes), DocumentSHA256: SHA256(finalDocumentBytes), ProjectionSHA256: SHA256(markdown),
		SourceSelection:    sourceSelection,
		ReaderFinalization: readerFinalization,
		Images:             sortManifestImages(manifestImages),
		Usage:              usage.normalized(),
	}
	if stagePlan.EditorialMemory {
		manifest.EditorialMemory = &ManifestEditorialMemory{
			ArtifactID: memoryReceipt.ArtifactID,
			SHA256:     memoryReceipt.SHA256,
			ByteSize:   memoryReceipt.ByteSize,
			Revision:   memoryReceipt.Revision,
			Accounts:   memoryReceipt.Accounts,
		}
	}
	if config.AuthoringMode == AuthoringModeLongForm {
		manifest.LongFormAuthoring = &longFormAuthoring
	}
	manifest.Artifacts = []ManifestArtifact{
		{ArtifactID: narrativeID, Kind: "narrative", MediaType: "application/json", SHA256: SHA256(narrativeBytes), ByteSize: len(narrativeBytes), Role: "intermediate", Filename: "narrative.json"},
		{ArtifactID: documentID, Kind: "semantic_il", MediaType: "application/json", SHA256: SHA256(finalDocumentBytes), ByteSize: len(finalDocumentBytes), Role: "intermediate", Filename: "semantic-il.json"},
		{ArtifactID: markdownID, Kind: "markdown", MediaType: "text/markdown; charset=utf-8", SHA256: SHA256(markdown), ByteSize: len(markdown), Role: "final", Filename: "report.md"},
		{ArtifactID: htmlID, Kind: "html", MediaType: "text/html; charset=utf-8", SHA256: SHA256(html), ByteSize: len(html), Role: "derivative", Filename: "report.html"},
		{ArtifactID: pdfID, Kind: "pdf", MediaType: pdfdocument.MediaType, SHA256: SHA256(pdf.Content), ByteSize: len(pdf.Content), Role: "derivative", Filename: "report.pdf"},
	}
	for _, image := range imageArtifacts {
		manifest.Artifacts = append(manifest.Artifacts, ManifestArtifact{ArtifactID: image.ID, Kind: image.Kind, MediaType: image.MediaType, SHA256: image.SHA256, ByteSize: image.ByteSize, Role: image.Role, Filename: image.Filename, AssetID: image.AssetID})
	}
	manifestBytes := mustMarshal(manifest)

	artifacts := []ProductArtifact{
		{ID: narrativeID, Kind: "narrative", MediaType: "application/json", Content: narrativeBytes, Role: "intermediate", Filename: "narrative.json"},
		{ID: documentID, Kind: "semantic_il", MediaType: "application/json", Content: finalDocumentBytes, Role: "intermediate", Filename: "semantic-il.json"},
		{ID: markdownID, Kind: "markdown", MediaType: "text/markdown; charset=utf-8", Content: markdown, Role: "final", Filename: "report.md"},
		{ID: htmlID, Kind: "html", MediaType: "text/html; charset=utf-8", Content: html, Role: "derivative", Filename: "report.html"},
		{ID: pdfID, Kind: "pdf", MediaType: pdfdocument.MediaType, Content: pdf.Content, Role: "derivative", Filename: "report.pdf"},
	}
	artifacts = append(artifacts, imageArtifacts...)
	artifacts = append(artifacts, ProductArtifact{ID: manifestID, Kind: "manifest", MediaType: "application/json", Content: manifestBytes, Role: "intermediate", Filename: "manifest.json"})
	for i := range artifacts {
		artifacts[i].SHA256 = SHA256(artifacts[i].Content)
		artifacts[i].ByteSize = len(artifacts[i].Content)
	}
	usage = usage.normalized()
	return ProductBundle{Artifacts: artifacts, Narrative: narrative, Document: document, Markdown: markdown, HTML: html, PDF: pdf.Content, ProjectionSHA: SHA256(markdown), CatalogSHA: authorCatalog.SHA256, SourceSelection: sourceSelection, ManifestContent: manifestBytes, Manifest: manifest, Usage: usage, ReaderFinalization: readerFinalization}, nil
}

// ProductManifest is safe durable lineage metadata with no paths or responses.
type ProductManifest struct {
	SchemaVersion      string                    `json:"schema_version"`
	CompilerVersion    string                    `json:"compiler_version"`
	PipelineFamily     string                    `json:"pipeline_family"`
	AuthoringMode      string                    `json:"authoring_mode"`
	ValidationProfile  string                    `json:"validation_profile"`
	MissionID          string                    `json:"mission_id"`
	CatalogSHA256      string                    `json:"source_catalog_sha256"`
	Provider           string                    `json:"provider"`
	Model              string                    `json:"model"`
	Effort             string                    `json:"effort"`
	DocumentID         string                    `json:"document_id"`
	RevisionID         string                    `json:"revision_id"`
	NarrativeSHA256    string                    `json:"narrative_sha256"`
	DocumentSHA256     string                    `json:"document_sha256"`
	ProjectionSHA256   string                    `json:"projection_sha256"`
	SourceSelection    SourceSelectionReceipt    `json:"source_selection"`
	EditorialMemory    *ManifestEditorialMemory  `json:"editorial_memory,omitempty"`
	LongFormAuthoring  *LongFormAuthoringReceipt `json:"long_form_authoring,omitempty"`
	ReaderFinalization ReaderFinalizationReceipt `json:"reader_finalization"`
	Images             []ManifestImage           `json:"images,omitempty"`
	Usage              TerminalUsageReceipt      `json:"usage"`
	Artifacts          []ManifestArtifact        `json:"artifacts"`
}

type ManifestEditorialMemory struct {
	ArtifactID string `json:"artifact_id,omitempty"`
	SHA256     string `json:"sha256,omitempty"`
	ByteSize   int    `json:"byte_size,omitempty"`
	Revision   int    `json:"revision,omitempty"`
	Accounts   int    `json:"accounts,omitempty"`
}

type ManifestArtifact struct {
	ArtifactID string `json:"artifact_id"`
	Kind       string `json:"kind"`
	MediaType  string `json:"media_type"`
	SHA256     string `json:"sha256"`
	ByteSize   int    `json:"byte_size"`
	Role       string `json:"role"`
	Filename   string `json:"filename"`
	AssetID    string `json:"asset_id,omitempty"`
}

func runAuthorStage(ctx context.Context, config ProductConfig, catalog reportilcontract.SourceCatalog, citations ...map[int]SourceCitation) (Narrative, Document, []agentexec.AgentResult, error) {
	citationCatalog := map[int]SourceCitation{}
	if len(citations) > 0 && citations[0] != nil {
		citationCatalog = citations[0]
	}
	narrative, document, _, results, err := runAuthorStageWithTerminology(
		ctx, config, catalog, citationCatalog, nil,
	)
	return narrative, document, results, err
}

func runAuthorStageWithTerminology(
	ctx context.Context,
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
	citations map[int]SourceCitation,
	readableByAcceptedOrdinal map[int]string,
) (Narrative, Document, []terminologyDecision, []agentexec.AgentResult, error) {
	contractID, documentID := config.NewID("narrative"), config.NewID("doc")
	authorSchema := providerAuthorSchemaBytes(catalog)
	originalPrompt := stagePrompt("il_narrative", authorPrompt(config, catalog))
	results := make([]agentexec.AgentResult, 0, 2)

	firstDraft, firstResult, firstReceipt, err := runAuthorSourceAttempt[authorDraft](
		ctx, config, catalog, 1, originalPrompt, authorSchema,
	)
	results = append(results, firstResult)
	if err != nil {
		var stageErr *providerStageError
		if !errors.As(err, &stageErr) || stageErr.reason != reportexecution.ProviderFailureReasonDecode {
			return Narrative{}, Document{}, nil, results, err
		}
		return runFullAuthorRepair(
			ctx, config, catalog, citations, readableByAcceptedOrdinal,
			contractID, documentID, authorSchema,
			sourceRepairPrompt(originalPrompt, "il_narrative", "decode", "", "", ""),
			results,
		)
	}

	firstNarrative, firstDocument, firstTerminology, validationErr := compileAuthorStageDraft(
		firstDraft, contractID, documentID, config.MissionObjective, catalog,
		firstReceipt, citations, readableByAcceptedOrdinal,
	)
	if validationErr == nil {
		return firstNarrative, firstDocument, firstTerminology, results, nil
	}
	if !isEvidencePacketValidationCode(providerValidationCode(validationErr)) {
		return runFullAuthorRepairForValidation(
			ctx, config, catalog, citations, readableByAcceptedOrdinal,
			contractID, documentID, originalPrompt, authorSchema,
			firstDraft, validationErr, results,
		)
	}
	if bindingErr := validateFrozenAuthorEvidenceBindings(firstDraft); bindingErr != nil {
		return runFullAuthorRepairForValidation(
			ctx, config, catalog, citations, readableByAcceptedOrdinal,
			contractID, documentID, originalPrompt, authorSchema,
			firstDraft, bindingErr, results,
		)
	}
	if frozenErr := validateFrozenAuthorDraftWithoutEvidence(
		firstDraft, contractID, documentID, config.MissionObjective, catalog,
		firstReceipt, citations, readableByAcceptedOrdinal,
	); frozenErr != nil {
		return runFullAuthorRepairForValidation(
			ctx, config, catalog, citations, readableByAcceptedOrdinal,
			contractID, documentID, originalPrompt, authorSchema,
			firstDraft, frozenErr, results,
		)
	}
	if safetyErr := validateAuthorEvidenceRepairPromptSafety(firstDraft); safetyErr != nil {
		return runFullAuthorRepairForValidation(
			ctx, config, catalog, citations, readableByAcceptedOrdinal,
			contractID, documentID, originalPrompt, authorSchema,
			firstDraft, safetyErr, results,
		)
	}
	if len(authorEvidenceBindingSlots(firstDraft)) > reportilcontract.MaxSourceReadSpans {
		return runFullAuthorRepairForValidation(
			ctx, config, catalog, citations, readableByAcceptedOrdinal,
			contractID, documentID, originalPrompt, authorSchema,
			firstDraft, withValidationCode(
				reportexecution.ProviderValidationCodeEvidencePacketInventory,
				fmt.Errorf("frozen evidence binding inventory exceeds the repair ceiling"),
			), results,
		)
	}

	repairPrompt := stagePrompt(
		"il_narrative",
		authorEvidenceRepairPrompt(firstDraft, catalog),
	)
	repairDraft, repairResult, repairReceipt, err := runAuthorSourceAttempt[authorEvidenceRepairDraft](
		ctx, config, catalog, 2, repairPrompt,
		providerAuthorEvidenceRepairSchemaBytes(firstDraft),
	)
	results = append(results, repairResult)
	if err != nil {
		return Narrative{}, Document{}, nil, results, err
	}
	if err := validateAuthorEvidenceRepair(firstDraft, repairDraft); err != nil {
		return Narrative{}, Document{}, nil, results, &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause:  err,
		}
	}
	repairedDraft := applyAuthorEvidenceRepair(firstDraft, repairDraft)
	narrative, document, terminology, err := compileAuthorStageDraft(
		repairedDraft, contractID, documentID, config.MissionObjective, catalog,
		repairReceipt, citations, readableByAcceptedOrdinal,
	)
	if err != nil {
		return Narrative{}, Document{}, nil, results, &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause:  err,
		}
	}
	return narrative, document, terminology, results, nil
}

func runAuthorSourceAttempt[T any](
	ctx context.Context,
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
	attempt int,
	prompt string,
	schema []byte,
) (T, agentexec.AgentResult, reportilcontract.SourceReadReceipt, error) {
	var zero T
	request := sourceIsolatedRequest(
		config, catalog, "il_narrative", attempt, prompt, schema, 0,
	)
	result, err := config.Provider.Run(ctx, request)
	if err != nil {
		return zero, result, reportilcontract.SourceReadReceipt{}, &providerStageError{
			reason: reportexecution.ProviderFailureReasonTransport,
			cause:  err,
		}
	}
	receipt, err := config.VerifySourceRead.VerifyReportILSourceRead(
		ctx, config.MissionID, request.ToolSessionID, "il_narrative", catalog,
	)
	if err != nil {
		return zero, result, reportilcontract.SourceReadReceipt{}, &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause: withValidationCode(
				reportexecution.ProviderValidationCodeSourceReadContract,
				err,
			),
		}
	}
	value, err := strictDecode[T]([]byte(result.Text))
	if err != nil {
		return zero, result, receipt, &providerStageError{
			reason: reportexecution.ProviderFailureReasonDecode,
			cause:  err,
		}
	}
	return value, result, receipt, nil
}

func runFullAuthorRepairForValidation(
	ctx context.Context,
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
	citations map[int]SourceCitation,
	readableByAcceptedOrdinal map[int]string,
	contractID,
	documentID,
	originalPrompt string,
	authorSchema []byte,
	firstDraft authorDraft,
	validationErr error,
	results []agentexec.AgentResult,
) (Narrative, Document, []terminologyDecision, []agentexec.AgentResult, error) {
	return runFullAuthorRepair(
		ctx, config, catalog, citations, readableByAcceptedOrdinal,
		contractID, documentID, authorSchema,
		sourceRepairPrompt(
			originalPrompt, "il_narrative", "semantic_validation",
			authorRepairContextWithValidation(firstDraft, validationErr),
			providerValidationCode(validationErr), providerValidationTermAlias(validationErr),
		),
		results,
	)
}

func runFullAuthorRepair(
	ctx context.Context,
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
	citations map[int]SourceCitation,
	readableByAcceptedOrdinal map[int]string,
	contractID,
	documentID string,
	authorSchema []byte,
	repairPrompt string,
	results []agentexec.AgentResult,
) (Narrative, Document, []terminologyDecision, []agentexec.AgentResult, error) {
	draft, result, receipt, err := runAuthorSourceAttempt[authorDraft](
		ctx, config, catalog, 2, repairPrompt, authorSchema,
	)
	results = append(results, result)
	if err != nil {
		return Narrative{}, Document{}, nil, results, err
	}
	narrative, document, terminology, err := compileAuthorStageDraft(
		draft, contractID, documentID, config.MissionObjective, catalog,
		receipt, citations, readableByAcceptedOrdinal,
	)
	if err != nil {
		return Narrative{}, Document{}, nil, results, &providerStageError{
			reason: reportexecution.ProviderFailureReasonSemanticValidation,
			cause:  err,
		}
	}
	return narrative, document, terminology, results, nil
}

func compileAuthorStageDraft(
	draft authorDraft,
	contractID,
	documentID,
	missionObjective string,
	catalog reportilcontract.SourceCatalog,
	receipt reportilcontract.SourceReadReceipt,
	citations map[int]SourceCitation,
	readableByAcceptedOrdinal map[int]string,
) (Narrative, Document, []terminologyDecision, error) {
	if err := validateCompleteSourceRead(catalog, receipt); err != nil {
		return Narrative{}, Document{}, nil, withValidationCode(
			reportexecution.ProviderValidationCodeSourceReadContract,
			err,
		)
	}
	narrative, document, terminology, err := compileAuthorDraftWithTerminology(
		draft, contractID, documentID, missionObjective, catalog, receipt,
		citations, readableByAcceptedOrdinal,
	)
	if err != nil {
		return Narrative{}, Document{}, nil, err
	}
	if err := validateProductEvidencePackets(narrative, document); err != nil {
		return Narrative{}, Document{}, nil, withValidationCode(
			reportexecution.ProviderValidationCodeEvidencePacketBinding,
			err,
		)
	}
	return narrative, document, terminology, nil
}

func isEvidencePacketValidationCode(code reportexecution.ProviderValidationCode) bool {
	switch code {
	case reportexecution.ProviderValidationCodeEvidencePacketInventory,
		reportexecution.ProviderValidationCodeEvidencePacketTarget,
		reportexecution.ProviderValidationCodeEvidencePacketSource,
		reportexecution.ProviderValidationCodeEvidencePacketBinding,
		reportexecution.ProviderValidationCodeEvidencePacketCoverage:
		return true
	default:
		return false
	}
}

func validateFrozenAuthorDraftWithoutEvidence(
	draft authorDraft,
	contractID,
	documentID,
	missionObjective string,
	catalog reportilcontract.SourceCatalog,
	receipt reportilcontract.SourceReadReceipt,
	citations map[int]SourceCitation,
	readableByAcceptedOrdinal map[int]string,
) error {
	withoutEvidence := draft
	withoutEvidence.EvidencePackets = nil
	_, _, _, err := compileAuthorDraftWithTerminology(
		withoutEvidence, contractID, documentID, missionObjective, catalog, receipt,
		citations, readableByAcceptedOrdinal,
	)
	return err
}

func validateFrozenAuthorEvidenceBindings(draft authorDraft) error {
	for _, section := range draft.Sections {
		for _, block := range section.Blocks {
			if len(block.EvidenceSourceKeys) == 0 {
				return withValidationCode(
					reportexecution.ProviderValidationCodeDocumentContract,
					fmt.Errorf("author content leaf has no frozen source binding"),
				)
			}
		}
	}
	return nil
}

func validateAuthorEvidenceRepair(
	frozen authorDraft,
	repair authorEvidenceRepairDraft,
) error {
	slots := authorEvidenceBindingSlots(frozen)
	if len(repair.Bindings) != len(slots) {
		return withValidationCode(
			reportexecution.ProviderValidationCodeEvidencePacketInventory,
			fmt.Errorf("author evidence repair binding inventory changed"),
		)
	}
	packetCount := 0
	for _, slot := range slots {
		packets, ok := repair.Bindings[slot.Alias]
		if !ok || len(packets) == 0 {
			return withValidationCode(
				reportexecution.ProviderValidationCodeEvidencePacketInventory,
				fmt.Errorf("author evidence repair binding inventory changed"),
			)
		}
		packetCount += len(packets)
		if packetCount > reportilcontract.MaxSourceReadSpans {
			return withValidationCode(
				reportexecution.ProviderValidationCodeEvidencePacketInventory,
				fmt.Errorf("author evidence repair packet inventory exceeds the ceiling"),
			)
		}
	}
	return nil
}

func applyAuthorEvidenceRepair(
	frozen authorDraft,
	repair authorEvidenceRepairDraft,
) authorDraft {
	slots := authorEvidenceBindingSlots(frozen)
	packets := make([]authorEvidenceDraft, 0)
	for _, slot := range slots {
		for _, packet := range repair.Bindings[slot.Alias] {
			packets = append(packets, authorEvidenceDraft{
				SectionKey:    slot.SectionKey,
				BlockKey:      slot.BlockKey,
				Claim:         packet.Claim,
				SourceKey:     slot.SourceKey,
				SourceReceipt: packet.SourceReceipt,
				SourceExcerpt: packet.SourceExcerpt,
				Preserve:      packet.Preserve,
			})
		}
	}
	frozen.EvidencePackets = packets
	return frozen
}

func validateAuthorEvidenceRepairPromptSafety(draft authorDraft) error {
	for _, section := range draft.Sections {
		for _, block := range section.Blocks {
			for _, value := range draftBlockReaderValues(block) {
				if readerFacingSourceMetadataPattern.MatchString(value) ||
					repairContextLocatorPattern.MatchString(value) ||
					repairContextCredentialPattern.MatchString(value) ||
					validateProviderContent(value) != nil {
					return withValidationCode(
						reportexecution.ProviderValidationCodeReaderFacingContent,
						fmt.Errorf("author manuscript cannot enter evidence repair context"),
					)
				}
			}
		}
	}
	return nil
}

func authorEvidenceRepairPrompt(
	frozen authorDraft,
	catalog reportilcontract.SourceCatalog,
) string {
	var out strings.Builder
	out.WriteString(`Rebuild only the complete evidence packet content for the server-frozen manuscript below. The manuscript, section and block topology, block kinds, reader-facing text, terminology, language review, and source bindings cannot be changed.

EVIDENCE PACKET CONTRACT:
- Start a fresh source-tool session and read every selected source before returning.
- Return every schema-required binding_NNN slot exactly once. The server fixes each slot to one frozen leaf and one opaque source; do not return or choose section_key, block_key, or source_key.
- Return one or more claim/source_receipt/preserve packets inside every binding slot.
- Each claim must be an exact contiguous substring of its slot's frozen leaf. Claims across all slots for a leaf must jointly cover all factual reader-facing text; for a table, cover every data cell while its caption and column labels are presentation metadata.
- For every packet, call plasma.report_il.sources.quote with the source fixed to that slot and a nonempty exact quote of at most 512 UTF-8 bytes, then return only the opaque source_receipt from that successful tool result. Never copy a source excerpt into the response.
- Use the fewest nonredundant packets that preserve complete factual-text coverage. Set preserve=true only for exact high-value, nonredundant detail whose loss would materially weaken the answer; there is no minimum preservation count.
- The total packet count across all binding slots must not exceed 128.
- Do not return manuscript fields, terminology, language review, source bodies, locators, validator prose, or any explanation outside the strict JSON response.

FROZEN READER-FACING LEAVES AND SERVER-FIXED BINDINGS:
`)
	slots := authorEvidenceBindingSlots(frozen)
	slotsByBlock := map[string][]authorEvidenceBindingSlot{}
	for _, slot := range slots {
		key := slot.SectionKey + "/" + slot.BlockKey
		slotsByBlock[key] = append(slotsByBlock[key], slot)
	}
	for sectionIndex, section := range frozen.Sections {
		for blockIndex, block := range section.Blocks {
			sectionKey := fmt.Sprintf("section_%02d", sectionIndex+1)
			blockKey := fmt.Sprintf("block_%02d", blockIndex+1)
			blockSlots := slotsByBlock[sectionKey+"/"+blockKey]
			aliases := make([]string, 0, len(blockSlots))
			for _, slot := range blockSlots {
				aliases = append(aliases, slot.Alias)
			}
			fmt.Fprintf(
				&out, "\n[%s/%s kind=%s bindings=%s]\n",
				sectionKey, blockKey, block.Kind, strings.Join(aliases, ","),
			)
			for _, slot := range blockSlots {
				fmt.Fprintf(
					&out, "- %s is server-fixed to %s; read that source and return packet content only\n",
					slot.Alias, slot.SourceKey,
				)
			}
			for _, value := range draftBlockReaderValues(block) {
				out.WriteString(value)
				out.WriteByte('\n')
			}
		}
	}
	fmt.Fprintf(
		&out, "\nFrozen source catalog SHA-256: %s\n%s",
		catalog.SHA256, authorEvidenceRepairSourceToolContract(),
	)
	return strings.TrimSpace(out.String())
}

func authorEvidenceRepairSourceToolContract() string {
	return `SOURCE ACCESS CONTRACT:
- Source content is intentionally absent from this prompt.
- First call plasma.report_il.sources.list with {}.
- Then call plasma.report_il.sources.read with {} until remaining_sources is zero, then stop. The server returns catalog-ordered batches and continues large sources across batches within the per-call ceiling.
- The tools expose only the run-frozen accepted source set. Do not use or request any other tool, connector, URL, path, or ambient knowledge.
- The strict response schema exposes only server-fixed binding_NNN slots. Do not return or choose source keys, section keys, or block keys.
- After all sources are completely read, call plasma.report_il.sources.quote separately for every packet using that slot's source key and an exact supporting quote of at most 512 UTF-8 bytes. Return each successful tool result's opaque source_receipt in the corresponding packet; one receipt may be used only once.
- Never return source excerpts, source bodies, tool metadata, or source-reading process in the structured response. The server reconstructs exact bytes from verified request-local receipts.`
}

func validateProductEvidencePackets(narrative Narrative, document Document) error {
	packetsByNode := map[string]int{}
	for _, packet := range narrative.EvidencePackets {
		packetsByNode[packet.AuthorNodeID]++
	}
	for _, block := range document.Blocks {
		if block.Kind == "section" {
			continue
		}
		if len(documentNodeEvidenceOrdinals(document, block.NodeID)) == 0 ||
			packetsByNode[block.NodeID] == 0 {
			return fmt.Errorf("author evidence packet is missing from a reader-facing content leaf")
		}
	}
	return nil
}

func authorRepairContext(draft authorDraft) string {
	return authorRepairContextWithValidation(draft, nil)
}

func authorRepairContextWithValidation(draft authorDraft, validationErr error) string {
	unsafeTerminology := draftTerminologyValues(draft.Terminology)
	var out strings.Builder
	fmt.Fprintf(&out, "TITLE: %s\nLANGUAGE: %s\nREADER TAKEAWAY: %s\nTHROUGHLINE: %s\nVOICE AND TONE: %s\n",
		safeRepairText(draft.Title, unsafeTerminology), safeRepairText(draft.Language, unsafeTerminology),
		safeRepairText(draft.ReaderTakeaway, unsafeTerminology), safeRepairText(draft.Throughline, unsafeTerminology),
		safeRepairText(draft.VoiceAndTone, unsafeTerminology),
	)
	for sectionIndex, section := range draft.Sections {
		fmt.Fprintf(&out, "\n[section_%02d] %s\n", sectionIndex+1, safeRepairText(section.Title, unsafeTerminology))
		for blockIndex, block := range section.Blocks {
			fmt.Fprintf(&out, "[block_%02d kind=%s]\n", blockIndex+1, block.Kind)
			for _, value := range draftBlockReaderValues(block) {
				if value = safeRepairText(value, unsafeTerminology); value != "" {
					out.WriteString(value)
					out.WriteByte('\n')
				}
			}
		}
	}
	packetAliases := providerValidationPacketAliases(validationErr)
	if len(packetAliases) > 0 {
		out.WriteString("\nINVALID EVIDENCE PACKETS TO REPAIR:\n")
		for _, alias := range packetAliases {
			index, ok := evidencePacketAliasIndex(alias)
			if !ok || index >= len(draft.EvidencePackets) {
				continue
			}
			packet := draft.EvidencePackets[index]
			if !providerValidationSectionAliasPattern.MatchString(packet.SectionKey) ||
				!providerValidationBlockAliasPattern.MatchString(packet.BlockKey) ||
				!providerValidationSourceKeyPattern.MatchString(packet.SourceKey) {
				continue
			}
			fmt.Fprintf(
				&out, "- %s: %s/%s from %s\n",
				alias, packet.SectionKey, packet.BlockKey, packet.SourceKey,
			)
		}
	}
	// Draft terminology and values using its reader forms are intentionally
	// omitted. Until source validation passes, neither alleged source forms nor
	// their proposed reader forms are source-safe facts. Raw packet claims and
	// excerpts are also omitted; only closed request-local aliases and bindings
	// may identify a rejected packet for the fresh repair attempt.
	return strings.TrimSpace(out.String())
}

func evidencePacketAliasIndex(alias string) (int, bool) {
	if !providerValidationPacketAliasPattern.MatchString(alias) {
		return 0, false
	}
	index, err := strconv.Atoi(strings.TrimPrefix(alias, "evidence_"))
	return index - 1, err == nil && index > 0
}

func draftTerminologyValues(drafts []authorTermDraft) []string {
	values := make([]string, 0, len(drafts)*3)
	for _, draft := range drafts {
		for _, value := range []string{draft.SourceForm, optionalProviderString(draft.SourceReading), draft.ReaderForm} {
			if value = strings.TrimSpace(value); value != "" {
				values = append(values, value)
			}
		}
	}
	return values
}

func safeRepairText(value string, unsafeTerminology []string) string {
	value = strings.TrimSpace(value)
	if value == "" || readerFacingSourceMetadataPattern.MatchString(value) || repairContextLocatorPattern.MatchString(value) || repairContextCredentialPattern.MatchString(value) || validateProviderContent(value) != nil {
		return ""
	}
	for _, term := range unsafeTerminology {
		if strings.Contains(value, term) {
			return ""
		}
	}
	return value
}

func draftBlockReaderValues(block documentBlockDraft) []string {
	switch block.Kind {
	case "prose", "quote", "callout":
		return []string{block.Prose}
	case "list":
		return append([]string(nil), block.Items...)
	case "code":
		return []string{block.Code}
	case "table":
		if block.Table == nil {
			return nil
		}
		values := []string{}
		if block.Table.Caption != nil {
			values = append(values, *block.Table.Caption)
		}
		values = append(values, block.Table.Column1, block.Table.Column2)
		if block.Table.Column3 != nil {
			values = append(values, *block.Table.Column3)
		}
		if block.Table.Column4 != nil {
			values = append(values, *block.Table.Column4)
		}
		for _, row := range block.Table.Rows {
			values = append(values, row.Cell1, row.Cell2)
			if row.Cell3 != nil {
				values = append(values, *row.Cell3)
			}
			if row.Cell4 != nil {
				values = append(values, *row.Cell4)
			}
		}
		return values
	default:
		return nil
	}
}

func authorPrompt(config ProductConfig, catalog reportilcontract.SourceCatalog) string {
	objective := strings.TrimSpace(config.MissionObjective)
	if objective == "" {
		objective = strings.TrimSpace(config.Title)
	}
	direction := strings.TrimSpace(config.Direction)
	if direction == "" {
		direction = "No additional direction. Answer the mission objective directly."
	}
	requestedTitle := strings.TrimSpace(config.Title)
	if !reportMachinerySubjectPattern.MatchString(objective) &&
		readerFacingInternalTermPattern.MatchString(requestedTitle) {
		requestedTitle = "Choose a natural reader-facing title from the mission objective and evidence."
	}
	return fmt.Sprintf(`Write the complete report a real reader asked for. The output is the manuscript itself, not a plan, audit, validation record, source tour, or description of how the report was produced.

MISSION OBJECTIVE:
%s

REQUESTED TITLE:
%s

ADDITIONAL DIRECTION:
%s

READER CONTRACT:
- Read every selected source before writing, then decide the strongest useful answer the evidence supports.
- Every reader-facing content leaf must cite at least one opaque source key, including transitions and conclusions; no factual prose may sit outside packet review. Return one evidence packet for every source binding on every content leaf. Bind it to the exact section_key and block_key aliases in the response and provide an exact contiguous claim from that leaf. For every packet, after reading the complete catalog, call plasma.report_il.sources.quote with its source_key and a nonempty exact supporting quote of at most 512 UTF-8 bytes copied from that source; place only the successful tool result's opaque source_receipt in the packet, never the source quote itself. Use each source_receipt once. If one leaf cites multiple sources, include one packet for each cited source. The claims across that leaf's packets must jointly cover all reader-facing factual text; for a table they must jointly cover every data cell while its caption and column labels are presentation metadata. A claim may cover a whole leaf when one registered quote supports it. Set preserve true only when the exact claim is a high-value, nonredundant fact, chronology, mechanism, example, distinction, tension, context, or evidence boundary whose exact loss would materially weaken the answer; otherwise set preserve false. There is no minimum preservation count and no reward for inventory size; do not split facts beyond what source support requires, restate them, or add generic background to satisfy this contract.
- Explain the subject itself. Never mention this generation system, its stages, IL, schemas, validation, selection, coverage, prompts, or source-reading process.
- Open briefly with the subject, the central question, and the main answer or evidence boundary. After that, let each section follow the subject and the reader's next question.
- Use concrete facts, mechanisms, examples, comparisons, and uncertainty where they matter. Do not organize the report source by source.
- Write a self-contained general report whose body develops the strongest useful answer far enough that a reader does not need the source material to supply missing context, chronology, mechanisms, examples, contrasts, or implications.
- Preserve high-value, nonredundant source-backed detail. Never improve fluency or brevity by replacing concrete supported facts, differences, tensions, or uncertainty with a generic summary. Synthesize repeated material instead of touring sources or counting citations.
- A thin outline, stitched summary, or safe overview that omits useful source-backed detail is not a complete report. Be rich and specific where the material supports it, but never pad the manuscript with repetition, generic background, or unsupported detail.
- Write idiomatic prose in the mission's requested language. For foreign names, places, institutions, technical terms, and official designations, use the conventional target-language form supported by the sources; do not mechanically pronounce source-script characters in the target language, invent a translated proper noun, or turn an official designation into a similar-looking common noun. If a specialist term has no familiar target-language form, retain or transliterate it consistently and explain it briefly on first use.
- Do not announce what a later section will discuss, narrate the outline, instruct the reader how to check the report, or repeat a conclusion as a checklist unless the mission explicitly asks for a procedure.
- State an important limitation once, next to the claim it qualifies. Do not turn caveats into repeated warnings.
- Every section must add a distinct claim, relationship, explanation, example, or implication. The conclusion must sharpen the answer rather than replay the section map.
- Author one coherent manuscript with a natural title and two to twelve reader-facing sections. Use prose by default; use a list, code block, quote, callout, or compact two-to-four-column table only when it genuinely improves comprehension.
- For each content leaf, list only the opaque source keys that directly support its factual content. Read keys are metadata only: never reproduce a source key or generic source label in authored text.
- Do not force every selected source into the report. Relevance and reader value decide what appears; unused selected sources remain part of the run receipt, not the prose. Before omitting a selected source, determine whether it contributes any material, nonredundant fact, example, chronology, contrast, context, or evidence boundary; preserve that information when it improves the reader's understanding.
- Use the terminology inventory only for reader-facing foreign proper names or official designations whose source form or source-supplied reading actually appears in the manuscript, and for every genuinely unfamiliar specialist term or foreign unit that needs explanation or conversion. A conventional target-language place, institution, ordinary measurement, or naturally translated historical term needs no inventory entry when you use only the natural reader form and omit source-script or romanized aliases. Express ordinary SI units directly in the target language; do not reproduce source notation merely to define m, km, or another familiar unit. Never add source-script forms, readings, romanizations, or dictionary-like explanations merely to satisfy the inventory. For each retained term, copy the exact source form; include source_reading only when a selected source explicitly supplies it next to that form; and choose the exact reader-facing form. The server derives terminology grounding across the selected sources; do not add source keys to terminology entries. Use preserve only when reader_form exactly preserves source_form and needs no explanation. Use preserve_and_explain only when the manuscript genuinely benefits from identifying or explaining a different source form, a source-supplied reading, or an unfamiliar concept. Use convert_and_explain only for a genuinely foreign unit whose converted value helps the reader.
- Explain every retained unfamiliar specialist term and genuinely foreign unit briefly in the same sentence, list item, or table cell as its first reader_form occurrence in body content. A title or heading may use the natural reader form without carrying the explanation only when the explanatory first body use also exists; otherwise remove that heading-only term. A retained proper name or official designation with explanatory handling follows the same rule. The server validates that sentence directly, so do not duplicate it in terminology metadata. Do not inventory ordinary words, familiar target-language names used without source-script aliases, familiar SI units, natural translations that need no source-form display, or terms that the report does not need.
- Treat every explicitly named subject, place, institution, relationship, and requested comparison in ADDITIONAL DIRECTION as a reader-facing content obligation. Answer it with source-bounded specificity or state the evidence limit next to it; never silently generalize a named entity into an unnamed category.
- Before returning, re-read the complete manuscript against the terminology inventory and specifically for natural target-language prose, conventional names and official terms, mechanical translation, unexplained units or specialist vocabulary, and a nonduplicative opening. Set every language_review field to true only after the manuscript actually passes that check; a false field rejects the draft.

%s

Frozen source catalog SHA-256: %s
%s`, objective, requestedTitle, direction, authorSemanticBoundaries(), catalog.SHA256, sourceToolContract(true))
}

func runReaderStage(
	ctx context.Context,
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
	document Document,
) (flowResponse, []agentexec.AgentResult, error) {
	flow, _, results, err := runReaderStageWithTerminology(ctx, config, catalog, document, nil)
	return flow, results, err
}

func runReaderStageWithTerminology(
	ctx context.Context,
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
	document Document,
	terminology []terminologyDecision,
	evidenceArguments ...[]EvidencePacket,
) (flowResponse, []terminologyDecision, []agentexec.AgentResult, error) {
	evidencePackets := []EvidencePacket{}
	if len(evidenceArguments) > 0 {
		evidencePackets = evidenceArguments[0]
	}
	schema := providerReaderSchemaBytes(document, terminology, evidencePackets)
	prompt := stagePrompt(
		"il_flow",
		readerPrompt(config, document, catalog, terminology, evidencePackets)+"\n\nFrozen source catalog SHA-256: "+catalog.SHA256+"\n"+readerSourceToolContract(len(evidencePackets)),
	)
	draft, results, err := runJSONStageWithSourceSchema(
		ctx,
		config,
		catalog,
		"il_flow",
		prompt,
		schema,
		func(value readerDraft, receipt reportilcontract.SourceReadReceipt) error {
			if len(evidencePackets) > 0 {
				if err := validateEvidenceSourceReads(catalog, evidencePackets, receipt); err != nil {
					return withValidationCode(
						reportexecution.ProviderValidationCodeSourceReadContract,
						err,
					)
				}
			} else if err := validateCompleteSourceRead(catalog, receipt); err != nil {
				return withValidationCode(
					reportexecution.ProviderValidationCodeSourceReadContract,
					err,
				)
			}
			_, _, _, err := compileReaderDraftWithTerminology(
				document,
				value,
				config.MissionObjective,
				terminology,
				evidencePackets,
			)
			return err
		},
	)
	if err != nil {
		return flowResponse{}, nil, results, err
	}
	edited, attestation, finalTerminology, err := compileReaderDraftWithTerminology(
		document, draft, config.MissionObjective, terminology, evidencePackets,
	)
	if err != nil {
		return flowResponse{}, nil, results, &providerStageError{reason: reportexecution.ProviderFailureReasonSemanticValidation, cause: err}
	}
	return flowResponse{Document: edited, Attestation: attestation}, finalTerminology, results, nil
}

func readerPrompt(config ProductConfig, document Document, arguments ...any) string {
	catalog := reportilcontract.SourceCatalog{}
	terms := []terminologyDecision{}
	evidencePackets := []EvidencePacket{}
	for _, argument := range arguments {
		switch value := argument.(type) {
		case reportilcontract.SourceCatalog:
			catalog = value
		case []terminologyDecision:
			terms = value
		case []EvidencePacket:
			evidencePackets = value
		}
	}
	objective := strings.TrimSpace(config.MissionObjective)
	if objective == "" {
		objective = strings.TrimSpace(config.Title)
	}
	finalSectionAlias := "(none)"
	if sections := readerSections(document); len(sections) > 0 {
		finalSectionAlias = readerSectionKey(len(sections) - 1)
	}
	return fmt.Sprintf(`Edit this complete manuscript as its final reader-author. Return the complete title, every section/block value, and every terminology decision in the exact schema inventory.

Your job is to make the report clearer, more useful, more natural, and more compelling while independently checking the original manuscript against the compact source evidence spans bound to the evidence packet. Read every bound span before editing. You may not use ambient knowledge or add facts.

- Distinguish three evidence levels while editing: a direct source statement, a bounded inference that combines established facts without claiming more, and an unsupported operational or causal claim. Check every relationship, causation, governance, administration, operational-control, surveillance, transport, protection, and traffic-control claim against the original manuscript's evidence boundaries and the bound source spans. Remove unsupported claims. When a requested relationship needs an evidence boundary, state that boundary naturally once, in the same paragraph as the relationship; do not repeat it in the opening, later sections, and conclusion.
- ADDITIONAL DIRECTION creates a reader-facing obligation, not evidence. A request to explain a relationship does not prove direct operation, administration, surveillance, protection, transport control, or causation. Never turn co-occurrence, geographic visibility, ownership by one lord, or a military corridor into a stronger operational relationship unless a selected source directly supports that stronger claim. Visibility alone does not support saying that movement was watched, monitored, protected, or controlled, even as a possibility.
- Style editing must never increase claim strength. Preserve useful named entities and supported relationships. Prefer direct subject prose built from established facts; use an explicit bounded inference only where it adds reader value.
- Keep or create one brief report-level opening whose first paragraph directly states the subject and the main historical answer. Do not answer with an inventory of surviving remains, a research-status statement, or an evidence disclaimer. Move any necessary evidence boundary to the affected relationship later in the manuscript.
- Remove report-process narration, section roadmaps, repeated previews and recaps, validation/checking instructions, formulaic transitions, generic source-tour language, duplicate synthesis, and any citation markers copied from the rendered manuscript.
- The source review is private work, not the report's voice. Use a nearby evidence boundary only when a distinct claim genuinely needs one, and state each boundary once beside that claim. Do not repeat the same limitation elsewhere or turn several limitations into a general research-status voice. Otherwise do not narrate what sources, materials, records, evidence, or the report can show, explain, confirm, establish, reconstruct, or prove. State the supported subject facts directly.
- Express familiar SI measurements naturally. Never define an elevation such as 353 meters by explaining that it is measured from sea level, and do not add dictionary-like explanations for meters, kilometers, or other ordinary units.
- Read the title and entire manuscript once specifically for language quality. Correct mechanical translation, unnatural target-language compounds, unsupported source-script character pronunciations presented as translated names, inconsistent transliteration, and mistranslated official designations. Preserve source-bounded meaning. A source-supplied reading is explicitly identified below when one exists. When none exists for a source-script specialist term, do not invent or infer a pronunciation from outside knowledge: retain the exact source form with a brief first-use explanation, or remove the term.
- Independently inventory every historical route, administrative institution, period-specific office, specialist concept, and other term an ordinary target-language reader may not know, including naturalized or transliterated forms that contain no source script and were omitted from the author's terminology decisions. At its first body use, either explain what it is in the same sentence or replace it with ordinary wording. Do not assume that a target-language spelling makes a specialist term familiar. A concise heading is allowed only when the explained first body use also exists.
- The terminology decisions below are source-safe. Each source form and each explicitly supplied reading is source-verified; an author reader form is not evidence of a source-supplied reading. For every term alias, echo the exact receipt and make three independent reader decisions. reader_form is the one final natural form, or null after removing every known form from the manuscript. Null removes the terminology notation, not necessarily the supported fact: keep useful meaning by rewriting it in ordinary target-language wording that contains none of the known forms. retain_as_terminology is true only when the final reader still needs this item enforced as an unfamiliar or continuity-sensitive term; set it false when the natural reader_form may remain as ordinary wording without a terminology receipt. present_source_aliases is true only when showing the exact source form or source-supplied reading adds real reader value; otherwise remove every source form and reading from the title and manuscript and set it false. Never set present_source_aliases false while keeping a reader_form equal to the source form or source-supplied reading; that form is visibly a source alias. A non-null term with retain_as_terminology false must also set present_source_aliases false and may not be a specialist term or foreign unit; retain a useful specialist term or genuinely foreign unit with its brief first-use explanation, or remove all known forms and rewrite the fact in ordinary target-language wording. Except for a source-script specialist term explicitly marked as having no source-supplied reading, the reader_form may be improved using linguistic judgment but must remain one consistent form. A source-script specialist term with no source-supplied reading may use only its exact source form with retain_as_terminology true and present_source_aliases true, or reader_form null after removing the exact notation; do not bypass this by turning retention off. In Korean, the schema requires null for such a term unless the mission explicitly names that exact source form. Normally remove the unfamiliar notation and state any useful supported fact naturally without inventing a pronunciation; when the mission names the exact form, preserve it exactly and explain it without adding a pronunciation. Put the actual explanation for every retained specialist term, genuinely foreign unit, or genuinely unfamiliar proper name or official designation directly in the same sentence, list item, or table cell as first use in body content. Remove low-value source aliases for familiar target-language names, ordinary SI measurements, natural translations of common historical terms, and foreign romanizations that do not help the reader. Preserve the natural target-language fact or wording when it remains useful. Titles and headings may stay concise only when a retained explanatory term also has an explained first body use; otherwise remove that heading-only term. The server derives that first-use sentence from the edited manuscript and validates it, so do not duplicate the explanation in terminology metadata. Remove superseded forms elsewhere.
- Preserve every explicitly named subject, place, institution, relationship, and requested comparison in ADDITIONAL DIRECTION when the source-bounded manuscript supports it. Never replace a requested named entity with a generic category merely to shorten or smooth the report.
- Return one evidence_support_coverage entry for every SERVER-BOUND EVIDENCE PACKET. Echo its exact receipt and edited section/block alias. Use direct_source_statement only when the compact source excerpt directly supports the final wording. Use bounded_inference only when the final wording is a cautious inference from that excerpt and the original manuscript does not state a stronger relationship. For either supported level, coverage_quote must be an exact nonempty contiguous quote copied from that packet's final edited block; quote the complete final wording this packet supports, so the union of supported quotes covers every factual leaf. A non-preserved authored claim may be naturally paraphrased when its final wording stays within the same evidence boundary. Use unsupported_removed only when the excerpt does not support the claim, remove that exact claim from the edited block, and return coverage_quote as null. A preserve=true packet may never be marked unsupported_removed or paraphrased; its coverage_quote must equal the exact authored claim. If its excerpt does not support it, reject the draft rather than laundering it through preservation coverage.
- Before returning, enforce the complete terminology contract yourself: a null term leaves none of its known forms; a non-retained proper name or official designation keeps only its natural reader form and leaves no source form or reading; a rename leaves no superseded reader form; every retained explanatory reader form occurs in body content; every presented source form and source-supplied reading occurs only in that first explained body sentence; and no retained canonical or alias equals, contains, or is contained by another term's form. In Korean, remove or account for every remaining Han, Hiragana, or Katakana span.
- Preserve or improve supported explanation of mechanisms, causal links, examples, comparisons, conditions, stakes, and uncertainty. The server-bound preservation obligations below are concrete source-grounded statements that already occur in the author manuscript. Preserve each exact statement in a final block that keeps its bound evidence; return that location in preservation_coverage. You may improve surrounding prose, but do not generalize, shorten, or paraphrase away these details.
- Judge the edited manuscript as a self-contained general report, not as a safe abstract. The evidence packet covers every authored source binding; preserve=true marks only the concrete high-value statements that must survive exactly. Develop natural relationships, context, and implications without arbitrary length, repetition, source tours, forced citations, or unsupported additions.
- Let transitions follow the subject or the reader's next question. Do not optimize for brevity alone. The report-level opening must not be followed by a second paragraph that restates the same answer before advancing the subject.
- Treat %s as the final conclusion section. Make it sharpen one final subject-level judgment. Do not recap the report by listing, naming, or walking back through the subjects of earlier sections, even if the final sentence then offers a judgment.
- Set every language_review and reader_review field to true only after the complete edited manuscript actually passes that check. unsupported_pronunciation_absent means no source-script pronunciation survives unless the terminology decision explicitly identifies it as source-supplied. low_value_source_aliases_absent means no source-script form, source reading, foreign romanization, or source unit notation survives unless it provides concrete reader value beyond the natural target-language wording. conclusion_not_recap means %s makes a final judgment without replaying the earlier section map. claim_strength_bounded means you checked every factual and inferential claim against the original manuscript and the compact source evidence spans, removed unsupported operational or causal claims, and kept every remaining inference no stronger than that evidence permits. subject_first_answer_present means the first paragraph directly gives the report's subject-level historical answer rather than an evidence inventory or disclaimer. audit_record_prose_absent means the source review remains invisible except for a necessary evidence boundary stated once beside each distinct claim that needs one. low_value_si_explanations_absent means ordinary measurements are stated naturally without dictionary-like definitions. unfamiliar_terms_explained means every historical route, administrative institution, period-specific office, specialist concept, and other unfamiliar term—including a naturalized or transliterated term absent from the terminology decisions—is explained in its first body sentence or replaced with ordinary wording. Detail preservation is accepted only through the schema-bound preservation_coverage receipts and exact final manuscript content, never through a self-asserted boolean. A false review field is a rejection, not permission to leave the defect in place.
- The ordered manuscript labels are the exact response aliases. Return section_01 for section_01 and block_01 for block_01 within that section. Echo each schema-required section_receipt and original_sha256 constant exactly; these bind every edit to its original location. Never move, exchange, or associate text with a different alias.
- Keep the exact section and block inventory. You may rewrite the report title, section titles, prose, list items, code, and table text, but may not add, remove, reorder, or change block kinds. The server preserves citations separately from the editable text.
- Never mention IL, pipelines, stages, schemas, validation, source selection, source coverage, frozen catalogs, prompts, or the writing process unless the mission itself is explicitly about that subject.

MISSION OBJECTIVE:
%s

ADDITIONAL DIRECTION:
%s

SOURCE-SAFE TERMINOLOGY DECISIONS:
%s

SERVER-BOUND EVIDENCE PACKETS:
%s

ORDERED COMPLETE MANUSCRIPT:
%s`, finalSectionAlias, finalSectionAlias, objective, readerDirection(config), readerPromptTerminology(terms), readerPromptEvidence(document, catalog, evidencePackets), readerPromptManuscript(document))
}

func readerDirection(config ProductConfig) string {
	direction := strings.TrimSpace(config.Direction)
	if direction == "" {
		return "No additional direction. Preserve the strongest source-bounded answer to the mission objective."
	}
	return direction
}

func finalizeNarrativeFromDocument(
	narrative Narrative,
	document Document,
	missionObjective string,
	terminology []terminologyDecision,
) (Narrative, error) {
	sections := readerSections(document)
	if len(sections) < 2 {
		return Narrative{}, fmt.Errorf("final Narrative requires at least two sections")
	}
	opening := readerSectionBoundaryText(sections[0], false)
	conclusion := readerSectionBoundaryText(sections[len(sections)-1], true)
	if opening == "" || conclusion == "" {
		return Narrative{}, fmt.Errorf("final Narrative requires opening and conclusion text")
	}
	centralQuestion := strings.TrimSpace(missionObjective)
	if centralQuestion == "" {
		centralQuestion = strings.TrimSpace(document.Title)
	}
	if centralQuestion == "" {
		return Narrative{}, fmt.Errorf("final Narrative central question is empty")
	}

	narrative.CentralQuestion = centralQuestion
	narrative.ReaderTakeaway = conclusion
	narrative.Throughline = opening
	narrative.ReaderJourney = make([]string, 0, len(sections))
	narrative.ArgumentArc = make([]string, 0, len(sections))
	narrative.SectionRoles = make([]SectionRole, 0, len(sections))
	for index, section := range sections {
		title := strings.TrimSpace(section.Title)
		if title == "" {
			return Narrative{}, fmt.Errorf("final Narrative section title is empty")
		}
		narrative.ReaderJourney = append(narrative.ReaderJourney, title)
		narrative.ArgumentArc = append(narrative.ArgumentArc, title)
		narrative.SectionRoles = append(narrative.SectionRoles, SectionRole{
			SectionID:        section.SectionID,
			Role:             authorSectionRole(index, len(sections)),
			QuestionAnswered: title,
		})
	}
	narrative.DependencyEdges = nil
	narrative.TransitionObligations = nil
	narrative.OpenLoops = nil
	narrative.Callbacks = nil
	narrative.RepetitionPolicies = nil
	narrative.ContinuityTerms = continuityTerms(terminology)
	narrative.ConclusionObligations = []string{conclusion}
	if err := validateNarrativeSchema(narrative); err != nil {
		return Narrative{}, err
	}
	if err := validateNarrative(&narrative, document); err != nil {
		return Narrative{}, err
	}
	return narrative, nil
}

func readerSectionBoundaryText(section readerSectionContext, reverse bool) string {
	if reverse {
		for index := len(section.Blocks) - 1; index >= 0; index-- {
			if value := readerBlockBoundaryText(section.Blocks[index], true); value != "" {
				return value
			}
		}
		return ""
	}
	for _, block := range section.Blocks {
		if value := readerBlockBoundaryText(block, false); value != "" {
			return value
		}
	}
	return ""
}

func readerBlockBoundaryText(block Block, reverse bool) string {
	switch block.Kind {
	case "prose", "quote", "callout":
		return strings.TrimSpace(block.Prose)
	case "list":
		if len(block.Items) == 0 {
			return ""
		}
		if reverse {
			return strings.TrimSpace(block.Items[len(block.Items)-1])
		}
		return strings.TrimSpace(block.Items[0])
	case "table":
		if block.Table != nil {
			return strings.TrimSpace(block.Table.Caption)
		}
	case "code":
		return strings.TrimSpace(block.Code)
	case "equation":
		if block.Equation != nil {
			return strings.TrimSpace(block.Equation.Expression)
		}
	}
	return ""
}

func markRequiredSourceForms(terms []terminologyDecision, objective, direction string) {
	requirements := objective + "\n" + direction
	for index := range terms {
		terms[index].PreserveSourceForm = strings.Contains(requirements, terms[index].SourceForm)
	}
}

func readerPromptTerminology(terms []terminologyDecision) string {
	if len(terms) == 0 {
		return "(none declared)"
	}
	var out strings.Builder
	for index, term := range terms {
		fmt.Fprintf(&out, "[%s] category=%s handling=%s\n", readerTermKey(index), term.Category, term.Handling)
		fmt.Fprintf(&out, "source form: %s\n", term.SourceForm)
		if term.SourceReading != "" {
			fmt.Fprintf(&out, "source-supplied reading: %s\n", term.SourceReading)
		} else if requiresSourceFormReaderForm(term) {
			out.WriteString("source-supplied reading: none; do not invent a pronunciation\n")
		}
		if term.PreserveSourceForm {
			out.WriteString("mission requirement: preserve the exact source form\n")
		}
		fmt.Fprintf(&out, "author reader form: %s\n", term.ReaderForm)
	}
	return strings.TrimSpace(out.String())
}

func readerPromptEvidence(
	document Document,
	catalog reportilcontract.SourceCatalog,
	packets []EvidencePacket,
) string {
	if len(packets) == 0 {
		return "(none declared)"
	}
	aliases := map[string]string{}
	for sectionIndex, section := range readerSections(document) {
		for blockIndex, block := range section.Blocks {
			aliases[block.NodeID] = fmt.Sprintf(
				"%s/%s",
				readerSectionKey(sectionIndex),
				readerBlockKey(blockIndex),
			)
		}
	}
	var out strings.Builder
	preservedIndex := 0
	for index, packet := range packets {
		fmt.Fprintf(
			&out,
			"[evidence_%03d] receipt=%s source=%s manuscript=%s source_offset=%d source_byte_size=%d source_excerpt_sha256=%s preserve=%t\n",
			index+1,
			evidenceReceipt(packet),
			sourceKeyByAcceptedOrdinal(catalog, packet.AcceptedOrdinal),
			aliases[packet.AuthorNodeID],
			packet.SourceOffset,
			packet.SourceByteSize,
			packet.SourceExcerptSHA,
			packet.Preserve,
		)
		fmt.Fprintf(&out, "exact authored claim to check: %s\n", packet.Claim)
		fmt.Fprintf(&out, "bound authored block receipt: %s\n", packet.AuthorNodeSHA)
		if packet.Preserve {
			preservedIndex++
			fmt.Fprintf(
				&out,
				"preservation alias: preserve_%02d; required exact statement: %s\n",
				preservedIndex,
				packet.Claim,
			)
		}
	}
	return strings.TrimSpace(out.String())
}

func readerPromptManuscript(document Document) string {
	var out strings.Builder
	fmt.Fprintf(&out, "TITLE: %s\n", document.Title)
	for sectionIndex, section := range readerSections(document) {
		fmt.Fprintf(&out, "\n[%s] %s\n", readerSectionKey(sectionIndex), section.Title)
		for blockIndex, block := range section.Blocks {
			fmt.Fprintf(&out, "[%s kind=%s]\n", readerBlockKey(blockIndex), block.Kind)
			switch block.Kind {
			case "prose", "quote", "callout":
				out.WriteString(block.Prose)
				out.WriteByte('\n')
			case "list":
				for _, item := range block.Items {
					fmt.Fprintf(&out, "- %s\n", item)
				}
			case "code":
				out.WriteString(block.Code)
				out.WriteByte('\n')
			case "equation":
				if block.Equation != nil {
					out.WriteString(block.Equation.Expression)
					out.WriteByte('\n')
				}
			case "table":
				if block.Table.Caption != "" {
					out.WriteString(block.Table.Caption)
					out.WriteByte('\n')
				}
				out.WriteString(strings.Join(block.Table.Columns, " | "))
				out.WriteByte('\n')
				for _, row := range block.Table.Rows {
					out.WriteString(strings.Join(row, " | "))
					out.WriteByte('\n')
				}
			}
		}
	}
	return out.String()
}

// runNarrativeStage and runDocumentStage remain compatibility seams for the
// focused contract tests. Product execution uses runAuthorStage so writing is
// source-aware once rather than split between planner and author.
func runNarrativeStage(ctx context.Context, config ProductConfig, catalog reportilcontract.SourceCatalog) (Narrative, []agentexec.AgentResult, error) {
	narrative, _, results, err := runAuthorStage(ctx, config, catalog)
	return narrative, results, err
}

func runDocumentStage(ctx context.Context, config ProductConfig, catalog reportilcontract.SourceCatalog, narrative Narrative) (Document, []agentexec.AgentResult, error) {
	narrativeBytes := mustMarshal(narrative)
	schema := providerDocumentSchemaBytes(narrative, catalog)
	prompt := stagePrompt("il_document", fmt.Sprintf("Use this exact Narrative Contract without changing its meaning:\n%s\nReport title: %s\nFrozen source catalog SHA-256: %s\n%s", narrativeBytes, config.Title, catalog.SHA256, completeSingleSourceToolContract()))
	var acceptedReadReceipt reportilcontract.SourceReadReceipt
	draft, results, err := runJSONStageWithSourceSchema(ctx, config, catalog, "il_document", prompt, schema, func(value documentDraft, receipt reportilcontract.SourceReadReceipt) error {
		if err := validateCompleteSourceRead(catalog, receipt); err != nil {
			return err
		}
		document, err := compileDocumentDraft(config.Title, narrative, catalog, receipt, value)
		if err != nil {
			return err
		}
		if err := validateDocumentSourceCoverage(document, catalog); err != nil {
			return err
		}
		acceptedReadReceipt = receipt
		return nil
	})
	if err != nil {
		return Document{}, results, err
	}
	document, err := compileDocumentDraft(config.Title, narrative, catalog, acceptedReadReceipt, draft)
	if err != nil {
		return Document{}, results, &providerStageError{reason: reportexecution.ProviderFailureReasonSemanticValidation, cause: err}
	}
	return document, results, nil
}

func validateCompleteSourceCatalogBudget(catalog reportilcontract.SourceCatalog) error {
	readableBytes := 0
	for _, entry := range catalog.Sources {
		if entry.ReadableBytes > reportilcontract.DefaultSourceAttemptReadBytes-readableBytes {
			return fmt.Errorf("frozen accepted source text exceeds the complete-read attempt ceiling")
		}
		readableBytes += entry.ReadableBytes
	}
	return nil
}

func validateCompleteSourceRead(catalog reportilcontract.SourceCatalog, receipt reportilcontract.SourceReadReceipt) error {
	if err := receipt.Validate(catalog); err != nil {
		return err
	}
	for _, entry := range catalog.Sources {
		if !receipt.IncludesEntireSource(entry.SourceKey) {
			return fmt.Errorf("report IL attempt did not read every byte of every frozen accepted source")
		}
	}
	return nil
}

func validateDocumentSourceCoverage(document Document, catalog reportilcontract.SourceCatalog) error {
	coveredOrdinals := make(map[int]bool, len(catalog.Sources))
	for _, ref := range document.References {
		if ref.Kind != "footnote" || !strings.HasPrefix(ref.Target, "accepted-source:") {
			continue
		}
		ordinal, err := strconv.Atoi(strings.TrimPrefix(ref.Target, "accepted-source:"))
		if err != nil || !catalogHasAcceptedOrdinal(catalog, ordinal) {
			return fmt.Errorf("document evidence reference is invalid")
		}
		coveredOrdinals[ordinal] = true
	}
	if len(coveredOrdinals) == 0 {
		return fmt.Errorf("document does not cite any frozen accepted source")
	}
	return nil
}

func catalogHasAcceptedOrdinal(catalog reportilcontract.SourceCatalog, ordinal int) bool {
	for _, entry := range catalog.Sources {
		if entry.AcceptedOrdinal == ordinal {
			return true
		}
	}
	return false
}

func catalogEntryByAcceptedOrdinal(catalog reportilcontract.SourceCatalog, ordinal int) (reportilcontract.SourceCatalogEntry, bool) {
	for _, entry := range catalog.Sources {
		if entry.AcceptedOrdinal == ordinal {
			return entry, true
		}
	}
	return reportilcontract.SourceCatalogEntry{}, false
}

func validateProductDocument(narrative Narrative, value Document) error {
	issues := productDocumentIssues(narrative, value)
	if len(issues) != 0 {
		return fmt.Errorf("semantic document violations:\n- %s", strings.Join(issues, "\n- "))
	}
	return nil
}

func productDocumentIssues(narrative Narrative, value Document) []string {
	issues := make([]string, 0)
	if err := validateProductDocumentShape(value); err != nil {
		issues = append(issues, "document shape: "+err.Error())
	}
	if err := validateCanonicalDocumentSchema(value); err != nil {
		issues = append(issues, "canonical document schema: "+err.Error())
	}
	if value.NarrativeContractID != narrative.ContractID || value.DocumentID != narrative.DocumentID {
		issues = append(issues, "Narrative binding: document_id and narrative_contract_id must match the server-assigned Narrative")
	}
	if value.RevisionID != "draft" {
		issues = append(issues, `revision binding: revision_id must be "draft"`)
	}
	for _, ref := range value.References {
		if ref.Kind != "footnote" || !strings.HasPrefix(ref.Target, "accepted-source:") || !strings.HasPrefix(ref.RefID, "ref.accepted_source.") || strings.TrimSpace(ref.VisibleLabel) == "" || !validServerCitationLocator(ref.Locator) {
			issues = append(issues, "initial profile: references must be server-owned accepted-source footnotes")
			break
		}
	}
	if len(value.Coverage) != 0 || len(value.Extensions) != 0 {
		issues = append(issues, "initial profile: coverage and extensions must be empty")
	}
	for _, asset := range value.Assets {
		if asset.LicenseStatus != "private_source" || asset.ArtifactID == "" || asset.SourceSnapshotReceipt == "" {
			issues = append(issues, "initial profile: figures require server-owned private-source assets")
			break
		}
	}
	for index, block := range value.Blocks {
		if len(block.RequirementRefs) != 0 {
			issues = append(issues, fmt.Sprintf("block %d %q: requirement_refs must be empty", index, block.NodeID))
		}
	}
	if err := validateDocument(value); err != nil {
		issues = append(issues, "document semantics: "+err.Error())
	}
	if err := validateNarrative(&narrative, value); err != nil {
		issues = append(issues, "Narrative realization: "+err.Error())
	}
	return uniqueIssues(issues)
}

func validServerCitationLocator(locator string) bool {
	return locator == "Frozen source catalog" || publicCitationURL(locator) == locator
}

func uniqueIssues(issues []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(issues))
	for _, issue := range issues {
		if issue = strings.TrimSpace(issue); issue != "" && !seen[issue] {
			seen[issue] = true
			result = append(result, issue)
		}
	}
	return result
}

func runFlowStage(ctx context.Context, config ProductConfig, catalog reportilcontract.SourceCatalog, narrative Narrative, document Document) (flowResponse, []agentexec.AgentResult, error) {
	flowSchema := providerFlowSchemaBytes(document)
	reviewContext, err := flowReviewContext(narrative, document, catalog)
	if err != nil {
		return flowResponse{}, nil, err
	}
	prompt := stagePrompt("il_flow", fmt.Sprintf("Read the complete ordered review manuscript before editing. Return one edited string for every schema-required prose node key. Preserve all source-bounded meaning and citations while improving whole-manuscript transitions, continuity, cadence, and repetition. The server owns and preserves identifiers, sections, lists, code, tables, evidence, provenance, revision, and projection hash. Re-read every selected source completely; do not invent source facts.\n%s\nFrozen source catalog SHA-256: %s\n%s", reviewContext, catalog.SHA256, sourceToolContract(true)))
	draft, results, err := runJSONStageWithSourceSchema(ctx, config, catalog, "il_flow", prompt, flowSchema, func(value flowDraft, receipt reportilcontract.SourceReadReceipt) error {
		if err := validateFlowEvidenceReads(document, catalog, receipt); err != nil {
			return err
		}
		_, err := compileFlowDraft(document, value)
		return err
	})
	if err != nil {
		return flowResponse{}, results, err
	}
	flow, err := compileFlowDraft(document, draft)
	if err != nil {
		return flowResponse{}, results, &providerStageError{reason: reportexecution.ProviderFailureReasonSemanticValidation, cause: err}
	}
	return flow, results, nil
}

func flowReviewContext(narrative Narrative, document Document, catalog reportilcontract.SourceCatalog) (string, error) {
	refOrdinals := make(map[string]int, len(document.References))
	refLabels := make(map[int]string, len(document.References))
	for _, ref := range document.References {
		if ref.Kind != "footnote" || !strings.HasPrefix(ref.Target, "accepted-source:") {
			continue
		}
		ordinal, err := strconv.Atoi(strings.TrimPrefix(ref.Target, "accepted-source:"))
		if err != nil || !catalogHasAcceptedOrdinal(catalog, ordinal) || strings.TrimSpace(ref.VisibleLabel) == "" {
			return "", fmt.Errorf("flow evidence reference is invalid")
		}
		refOrdinals[ref.RefID] = ordinal
		refLabels[ordinal] = ref.VisibleLabel
	}
	var out strings.Builder
	out.WriteString("FLOW OBLIGATIONS:\n")
	fmt.Fprintf(&out, "- Central question: %s\n- Reader takeaway: %s\n- Throughline: %s\n- Voice and tone: %s\n", narrative.CentralQuestion, narrative.ReaderTakeaway, narrative.Throughline, narrative.VoiceAndTone)
	for _, transition := range narrative.TransitionObligations {
		fmt.Fprintf(&out, "- Transition %s -> %s: %s\n", transition.FromSectionID, transition.ToSectionID, transition.Obligation)
	}
	for _, obligation := range narrative.ConclusionObligations {
		fmt.Fprintf(&out, "- Conclusion: %s\n", obligation)
	}
	if len(refLabels) > 0 {
		out.WriteString("EVIDENCE LEGEND:\n")
		ordinals := make([]int, 0, len(refLabels))
		for ordinal := range refLabels {
			ordinals = append(ordinals, ordinal)
		}
		sort.Ints(ordinals)
		for _, ordinal := range ordinals {
			fmt.Fprintf(&out, "- %d: %s\n", ordinal, refLabels[ordinal])
		}
	}
	out.WriteString("ORDERED REVIEW MANUSCRIPT:\n")
	for _, block := range document.Blocks {
		switch block.Kind {
		case "section":
			fmt.Fprintf(&out, "\n[SECTION %s] %s\n", block.NodeID, block.Title)
		case "prose", "quote", "callout":
			fmt.Fprintf(&out, "[%s %s]", strings.ToUpper(block.Kind), block.NodeID)
			if len(block.EvidenceRefs) > 0 {
				out.WriteString(" [evidence:")
				for index, refID := range block.EvidenceRefs {
					if index > 0 {
						out.WriteByte(',')
					}
					ordinal, ok := refOrdinals[refID]
					if !ok {
						return "", fmt.Errorf("flow evidence reference is invalid")
					}
					fmt.Fprintf(&out, "%d", ordinal)
				}
				out.WriteByte(']')
			}
			out.WriteByte('\n')
			out.WriteString(block.Prose)
			out.WriteByte('\n')
		case "list":
			out.WriteString("[LIST]\n")
			for _, item := range block.Items {
				fmt.Fprintf(&out, "- %s\n", item)
			}
		case "code":
			fmt.Fprintf(&out, "[CODE %s]\n%s\n", block.Language, block.Code)
		case "table":
			out.WriteString("[TABLE]\n")
			if block.Table.Caption != "" {
				out.WriteString(block.Table.Caption)
				out.WriteByte('\n')
			}
			out.WriteString(strings.Join(block.Table.Columns, " | "))
			out.WriteByte('\n')
			for _, row := range block.Table.Rows {
				out.WriteString(strings.Join(row, " | "))
				out.WriteByte('\n')
			}
		case "equation":
			out.WriteString("[EQUATION latex]\n")
			if block.Equation != nil {
				out.WriteString(block.Equation.Expression)
				out.WriteByte('\n')
			}
		default:
			return "", fmt.Errorf("flow review context does not support block kind %q", block.Kind)
		}
	}
	return out.String(), nil
}

func validateFlowEvidenceReads(document Document, catalog reportilcontract.SourceCatalog, receipt reportilcontract.SourceReadReceipt) error {
	if err := validateCompleteSourceRead(catalog, receipt); err != nil {
		return err
	}
	refOrdinals := make(map[string]int, len(document.References))
	for _, ref := range document.References {
		if ref.Kind != "footnote" || !strings.HasPrefix(ref.Target, "accepted-source:") {
			continue
		}
		ordinal, err := strconv.Atoi(strings.TrimPrefix(ref.Target, "accepted-source:"))
		if err != nil || !catalogHasAcceptedOrdinal(catalog, ordinal) {
			return fmt.Errorf("flow evidence reference is invalid")
		}
		refOrdinals[ref.RefID] = ordinal
	}
	for _, block := range document.Blocks {
		if !authoredProseKind(block.Kind) {
			continue
		}
		for _, refID := range block.EvidenceRefs {
			ordinal, ok := refOrdinals[refID]
			entry, available := catalogEntryByAcceptedOrdinal(catalog, ordinal)
			if !ok || !available || !receipt.IncludesEntireSource(entry.SourceKey) {
				return fmt.Errorf("flow attempt did not read every byte of every source cited by edited prose")
			}
		}
	}
	return nil
}

func flowResponseSchemaText() string {
	raw, err := schemaFiles.ReadFile("schemas/report-flow-response.experimental.v2.schema.json")
	if err != nil {
		panic(fmt.Sprintf("read flow response schema: %v", err))
	}
	return string(raw)
}

func validateFlowResponseSchema(value flowResponse) error {
	loaded, err := loadSchemas()
	if err != nil {
		return err
	}
	compiled := loaded.flowResponse
	generic, err := schemaValue(value)
	if err != nil {
		return err
	}
	if err := compiled.Validate(generic); err != nil {
		return fmt.Errorf("%s schema: %w", flowResponseSchemaURL, err)
	}
	return nil
}

func narrativeSchemaText() string {
	raw, _ := schemaFiles.ReadFile("schemas/narrative-contract.experimental.v1.schema.json")
	return string(raw)
}
func stagePrompt(stage, contextText string) string {
	return fmt.Sprintf("Generate only strict JSON for %s using the request-local Structured Output schema. Additional properties are forbidden, and all required fields, enums, and block shapes must be honored. Cross-object binding rules are enforced by the server.\nCONTEXT:\n%s", stage, contextText)
}

func completeSingleSourceToolContract() string {
	return fmt.Sprintf(`SOURCE ACCESS CONTRACT:
- Source content is intentionally absent from this prompt.
- First call plasma.report_il.sources.list with {}.
- Read every byte of every frozen accepted source exactly once using plasma.report_il.sources.read with source_key, offset, and max_bytes. Start each source at offset 0 and continue only at the exact returned next_offset until truncated is false. Never restart or reread a completed source.
- Use up to %d max_bytes per read when the remaining source is that large, so complete coverage needs fewer non-overlapping tool turns.
- The tools expose only the run-frozen accepted source set. Do not use or request any other tool, connector, URL, path, or ambient knowledge.
- Do not reproduce tool metadata or source identifiers in authored content. Use opaque source keys only in a schema-required evidence_source_keys field. Ground every factual claim in source text you actually read.`, reportilcontract.DefaultSourceReadMaxBytes)
}

func readerSourceToolContract(packetCount ...int) string {
	if len(packetCount) > 0 && packetCount[0] > 0 {
		return fmt.Sprintf(`SOURCE ACCESS CONTRACT:
- Source content is intentionally absent from this prompt.
- First call plasma.report_il.sources.list with {}.
- Then call plasma.report_il.sources.read with {} until remaining_sources is zero, then stop. The server returns compact source evidence spans derived from %d bound evidence packets; complete selected source bodies are unavailable in this stage.
- Use these spans to independently check every authored source binding and any claim-strength issue in the original manuscript. Do not add facts from ambient knowledge.
- The tools expose only the run-frozen evidence packet. Do not use or request any other tool, connector, URL, path, or ambient knowledge.
- Never reproduce source keys, tool metadata, private locators, source excerpts, or source-reading process in the edited manuscript.`, packetCount[0])
	}
	return `SOURCE ACCESS CONTRACT:
- Source content is intentionally absent from this prompt.
- First call plasma.report_il.sources.list with {}.
- Then call plasma.report_il.sources.read with {} until remaining_sources is zero, then stop. The server returns the complete catalog in order and continues large sources across batches within the per-call ceiling.
- Read all selected sources before judging or editing the manuscript. A source cited by the author is not automatically sufficient for the claim attached to it.
- The tools expose only the run-frozen selected source set. Do not use or request any other tool, connector, URL, path, or ambient knowledge.
- Never reproduce source keys, tool metadata, private locators, or source-reading process in the edited manuscript.`
}

func sourceToolContract(requireCompleteCatalog bool) string {
	if requireCompleteCatalog {
		return `SOURCE ACCESS CONTRACT:
- Source content is intentionally absent from this prompt.
- First call plasma.report_il.sources.list with {}.
- Then call plasma.report_il.sources.read with {} until remaining_sources is zero, then stop. The server returns catalog-ordered batches and continues large sources across batches within the per-call ceiling.
- The tools expose only the run-frozen accepted source set. Do not use or request any other tool, connector, URL, path, or ambient knowledge.
- Do not reproduce tool metadata or source identifiers in authored content. Use opaque source keys only in a schema-required evidence_source_keys field. Ground every factual claim in source text you actually read.`
	}
	return fmt.Sprintf(`SOURCE ACCESS CONTRACT:
- Source content is intentionally absent from this prompt.
- First call plasma.report_il.sources.list with {}.
- Read every source cited by edited prose exactly once using plasma.report_il.sources.read. Start each required source at offset 0, continue only at the exact returned next_offset until truncated is false, and never restart a completed source.
- Use up to %d max_bytes per read when the remaining source is that large, so source coverage needs fewer non-overlapping tool turns. Continue only with returned next_offset when more text is needed.
- The tools expose only the run-frozen accepted source set. Do not use or request any other tool, connector, URL, path, or ambient knowledge.
- Do not reproduce tool metadata or source identifiers in authored content. Ground every factual claim in source text you actually read.`, reportilcontract.DefaultSourceReadMaxBytes)
}

func validateNarrativeSchema(value Narrative) error {
	compiled, err := loadSchemas()
	if err != nil {
		return err
	}
	generic, err := schemaValue(value)
	if err != nil {
		return err
	}
	if err := compiled.narrative.Validate(generic); err != nil {
		return fmt.Errorf("%s schema: %w", narrativeSchemaURL, err)
	}
	return nil
}

func runJSONStage[T any](ctx context.Context, config ProductConfig, stage, prompt string, validate func(T) error) (T, []agentexec.AgentResult, error) {
	return runJSONStageWithSchema(ctx, config, stage, prompt, nil, validate)
}

type providerStageError struct {
	reason reportexecution.ProviderFailureReason
	cause  error
}

func (err *providerStageError) Error() string { return "provider stage failed" }
func (err *providerStageError) Unwrap() error { return err.cause }

func providerStageFailure(stage string, cause error, usage TerminalUsageReceipt) *reportexecution.StageFailureError {
	failure := reportexecution.NewStageFailure(stage, "", -1, -1, cause)
	var providerFailure *providerStageError
	if !errors.As(cause, &providerFailure) || !providerFailure.reason.Valid() {
		return failure
	}
	receipt := reportexecution.NewProviderFailureUsageReceipt(usage.failureAttempts())
	failure.SafeFailureReason = providerFailure.reason
	if providerFailure.reason == reportexecution.ProviderFailureReasonSemanticValidation {
		failure.SafeValidationCode = providerValidationCode(providerFailure.cause)
	}
	failure.ProviderUsage = &receipt
	return failure
}

func semanticProviderStageFailure(stage string, cause error, usage TerminalUsageReceipt) *reportexecution.StageFailureError {
	return providerStageFailure(stage, &providerStageError{
		reason: reportexecution.ProviderFailureReasonSemanticValidation,
		cause:  cause,
	}, usage)
}

func runJSONStageWithSchema[T any](ctx context.Context, config ProductConfig, stage, prompt string, schema []byte, validate func(T) error) (T, []agentexec.AgentResult, error) {
	return runJSONStageAttempts(ctx, config, reportilcontract.SourceCatalog{}, stage, prompt, schema, false, 0, func(value T, _ reportilcontract.SourceReadReceipt) error {
		return validate(value)
	})
}

func runJSONStageWithSourceSchema[T any](ctx context.Context, config ProductConfig, catalog reportilcontract.SourceCatalog, stage, prompt string, schema []byte, validate func(T, reportilcontract.SourceReadReceipt) error, repairContext ...func(T, error) string) (T, []agentexec.AgentResult, error) {
	return runJSONStageWithSourceLimit(ctx, config, catalog, stage, prompt, schema, 0, validate, repairContext...)
}

func runJSONStageWithSourceLimit[T any](ctx context.Context, config ProductConfig, catalog reportilcontract.SourceCatalog, stage, prompt string, schema []byte, maxSourceReadBytes int, validate func(T, reportilcontract.SourceReadReceipt) error, repairContext ...func(T, error) string) (T, []agentexec.AgentResult, error) {
	return runJSONStageAttempts(ctx, config, catalog, stage, prompt, schema, true, maxSourceReadBytes, validate, repairContext...)
}

func runJSONStageAttempts[T any](ctx context.Context, config ProductConfig, catalog reportilcontract.SourceCatalog, stage, prompt string, schema []byte, requireSourceRead bool, maxSourceReadBytes int, validate func(T, reportilcontract.SourceReadReceipt) error, repairContext ...func(T, error) string) (T, []agentexec.AgentResult, error) {
	var zero T
	results := make([]agentexec.AgentResult, 0, 2)
	for attempt := 0; attempt < 2; attempt++ {
		var request agentexec.AgentRequest
		if requireSourceRead {
			request = sourceIsolatedRequest(config, catalog, stage, attempt+1, prompt, schema, maxSourceReadBytes)
		} else {
			request = isolatedRequest(stage, prompt, schema)
		}
		result, err := config.Provider.Run(ctx, request)
		results = append(results, result)
		if err != nil {
			return zero, results, &providerStageError{reason: reportexecution.ProviderFailureReasonTransport, cause: err}
		}
		var sourceReadReceipt reportilcontract.SourceReadReceipt
		if requireSourceRead {
			sourceReadReceipt, err = config.VerifySourceRead.VerifyReportILSourceRead(ctx, config.MissionID, request.ToolSessionID, stage, catalog)
			if err != nil {
				return zero, results, &providerStageError{
					reason: reportexecution.ProviderFailureReasonSemanticValidation,
					cause: withValidationCode(
						reportexecution.ProviderValidationCodeSourceReadContract,
						err,
					),
				}
			}
		}
		value, decodeErr := strictDecode[T]([]byte(result.Text))
		if decodeErr == nil {
			if validationErr := validate(value, sourceReadReceipt); validationErr == nil {
				return value, results, nil
			} else if attempt == 0 {
				if requireSourceRead {
					context := ""
					if len(repairContext) > 0 && repairContext[0] != nil {
						context = repairContext[0](value, validationErr)
					}
					prompt = sourceRepairPrompt(
						prompt, stage, "semantic_validation", context,
						providerValidationCode(validationErr), providerValidationTermAlias(validationErr),
						providerValidationPacketAliases(validationErr)...,
					)
				} else {
					prompt = sourceFreeRepairPrompt(prompt, stage, "semantic_validation", providerValidationCode(validationErr))
				}
				continue
			} else {
				return zero, results, &providerStageError{reason: reportexecution.ProviderFailureReasonSemanticValidation, cause: validationErr}
			}
		} else if attempt == 0 {
			if requireSourceRead {
				prompt = sourceRepairPrompt(prompt, stage, "decode", "", "", "")
			} else {
				prompt = sourceFreeRepairPrompt(prompt, stage, "decode", "")
			}
			continue
		} else {
			return zero, results, &providerStageError{reason: reportexecution.ProviderFailureReasonDecode, cause: decodeErr}
		}
	}
	return zero, results, &providerStageError{reason: reportexecution.ProviderFailureReasonSemanticValidation, cause: fmt.Errorf("provider stage failed")}
}

func sourceAttemptTools(stage string) []string {
	switch stage {
	case "il_source_selection", "il_editorial_memory", "il_flow":
		return []string{reportilcontract.SourceListTool, reportilcontract.SourceReadTool}
	case "il_narrative":
		// Compatibility-only author paths still inspect their historical request
		// contract. Active MCP author paths replace this list before dispatch.
		return []string{
			reportilcontract.SourceListTool,
			reportilcontract.SourceReadTool,
			reportilcontract.SourceQuoteRegisterTool,
		}
	default:
		return nil
	}
}

func sourceIsolatedRequest(config ProductConfig, catalog reportilcontract.SourceCatalog, stage string, attempt int, prompt string, schema []byte, maxSourceReadBytes int) agentexec.AgentRequest {
	profile := agentcapability.ReportILSource()
	maxReadBytes := 0
	maxCallBytes := 0
	if stage == "il_source_selection" {
		maxReadBytes = reportilcontract.DefaultSourceReadMaxBytes
		maxCallBytes = reportilcontract.DefaultSourceReadMaxBytes
	} else if stage == "il_editorial_memory" || stage == "il_flow" ||
		(stage == "il_narrative" || stage == "il_long_form_plan" || stage == "il_long_form_section") &&
			normalizeValidationProfile(config.ValidationProfile) == ValidationProfileUnverified {
		maxReadBytes = reportilcontract.DefaultSourceAttemptReadBytes
		maxCallBytes = reportilcontract.DefaultSourceReadMaxBytes
	}
	binding := reportilcontract.SourceAccessBinding{
		PendingEventID:     config.PendingEventID,
		Stage:              stage,
		Attempt:            attempt,
		MaxReadBytes:       maxReadBytes,
		MaxCallBytes:       maxCallBytes,
		MaxSourceReadBytes: maxSourceReadBytes,
		ReadSpans:          nil,
		Catalog:            catalog,
	}
	if stage == "il_long_form_plan" || stage == "il_long_form_section" || stage == "il_long_form_part" || stage == "il_long_form_final" {
		binding.LongFormTargetLanguage = config.TargetLanguage
	}
	if stage == "il_flow" {
		binding.ReadSpans = append([]reportilcontract.SourceReadSpan(nil), config.flowReadSpans...)
		if len(binding.ReadSpans) > 0 {
			binding.MaxReadBytes = 0
			for _, span := range binding.ReadSpans {
				binding.MaxReadBytes += span.ByteSize
			}
			if binding.MaxReadBytes < binding.MaxCallBytes {
				binding.MaxCallBytes = binding.MaxReadBytes
			}
		}
	}
	return agentexec.AgentRequest{
		UserText:          "report IL " + stage,
		Prompt:            prompt,
		Model:             "gpt-5.6-luna",
		ReasoningEffort:   "xhigh",
		MissionID:         config.MissionID,
		ToolSessionID:     config.NewID("ses"),
		AgentExecutor:     "codex",
		MCPMode:           "source_read_only",
		CapabilityProfile: profile.ID,
		ProfileRevision:   profile.Revision,
		DisableTools:      false,
		IgnoreUserConfig:  true,
		EphemeralSession:  true,
		ReplaceMCPTools:   true,
		ExtraMCPTools:     sourceAttemptTools(stage),
		OutputJSONSchema:  append([]byte(nil), schema...),
		ReportILSources:   &binding,
	}
}

func longFormIsolatedRequest(config ProductConfig, catalog reportilcontract.SourceCatalog, stage, prompt string, schema []byte, memoryReceipt reportilcontract.EditorialMemoryReceipt) agentexec.AgentRequest {
	request := sourceIsolatedRequest(config, catalog, stage, 1, prompt, schema, 0)
	if memoryReceipt.ArtifactID != "" {
		request.ReportILSources.EditorialMemoryArtifactID = memoryReceipt.ArtifactID
		request.ReportILSources.EditorialMemorySHA256 = memoryReceipt.SHA256
	}
	return request
}

func isolatedRequest(stage, prompt string, schemas ...[]byte) agentexec.AgentRequest {
	profile := agentcapability.ReportIL()
	var schema []byte
	if len(schemas) > 0 {
		schema = schemas[0]
	}
	return agentexec.AgentRequest{
		UserText:          "report IL " + stage,
		Prompt:            prompt,
		Model:             "gpt-5.6-luna",
		ReasoningEffort:   "xhigh",
		AgentExecutor:     "codex",
		MCPMode:           "disabled",
		CapabilityProfile: profile.ID,
		ProfileRevision:   profile.Revision,
		DisableTools:      true,
		IgnoreUserConfig:  true,
		EphemeralSession:  true,
		ReplaceMCPTools:   true,
		OutputJSONSchema:  append([]byte(nil), schema...),
	}
}

func sourceRepairPrompt(original, stage, failureClass, repairContext string, code reportexecution.ProviderValidationCode, termAlias string, packetAliases ...string) string {
	guidance := "Return only strict JSON matching the exact schema."
	if failureClass == "semantic_validation" {
		switch stage {
		case "il_source_selection":
			guidance = "Return every listed source key exactly once in ranked_source_keys with no duplicates or omissions. Return only strict JSON matching the exact schema."
		case "il_narrative":
			guidance = authorSemanticBoundaries() + "\n" + authorRepairGuidance(code, termAlias) + " Return only strict JSON matching the exact schema."
		case "il_flow":
			guidance = readerRepairGuidance(code, packetAliases...) + " Return only strict JSON matching the exact schema."
		}
	}
	repair := original + "\nREPAIR ATTEMPT. The prior output was rejected (" + failureClass + "). Start a fresh source-tool session and return a corrected complete value. Raw provider output and validator prose are intentionally omitted; use only the safe manuscript context below when present. " + guidance
	if failureClass == "semantic_validation" && stage == "il_narrative" && repairContext != "" {
		repair += "\nSAFE REQUEST-LOCAL MANUSCRIPT TO REPAIR:\n" + repairContext
	}
	return repair
}

func authorRepairGuidance(code reportexecution.ProviderValidationCode, termAlias string) string {
	target := "the rejected terminology entry"
	if providerValidationTermAliasPattern.MatchString(termAlias) {
		target = termAlias
	}
	switch code {
	case reportexecution.ProviderValidationCodeSourceReadContract:
		return "Read every byte of every frozen source again before returning, and cite only keys read in this attempt."
	case reportexecution.ProviderValidationCodeLanguageReview:
		return "Re-read the full manuscript and correct every language-review defect before setting all four review fields true."
	case reportexecution.ProviderValidationCodeTerminologyInventory:
		return "Reconcile " + target + " and the complete terminology array with every term actually used in the title, headings, and manuscript; remove unused entries and add missing entries."
	case reportexecution.ProviderValidationCodeTerminologySourceGrounding:
		return "Reconstruct the complete terminology array from the selected sources. Use only exact source forms present in those sources and include a reading only when it appears next to that form."
	case reportexecution.ProviderValidationCodeTerminologySourceFormGrounding:
		return "Reconstruct " + target + " using an exact source form that appears in a selected source, or remove that entry and its ungrounded forms from the manuscript."
	case reportexecution.ProviderValidationCodeTerminologySourceReadingGrounding:
		return "For " + target + ", keep source_reading only if that exact reading appears next to its exact source_form in a selected source; otherwise return source_reading as null and do not present it as source-supplied."
	case reportexecution.ProviderValidationCodeTerminologyAliasCollision:
		return "Give each terminology entry unique source, reading, and reader forms that do not belong to another entry."
	case reportexecution.ProviderValidationCodeTerminologyScriptCoverage:
		return "Account for every Han, Hiragana, or Katakana span remaining in the Korean manuscript, or replace unnecessary source-script text with the natural Korean reader form."
	case reportexecution.ProviderValidationCodeReaderFacingContent:
		return "Remove reader-facing report-process narration, internal identifiers, locators, and duplicated opening synthesis while preserving the subject matter."
	case reportexecution.ProviderValidationCodeEvidencePacketInventory:
		return "Return a complete evidence_packets inventory within the schema ceiling. Use each referenced source_receipt once. You may register candidate quotes while working and omit unused candidate receipts from the final response; the server discards them. Repeated valid packets with the same block, source, and exact claim are canonicalized to one packet."
	case reportexecution.ProviderValidationCodeEvidencePacketTarget:
		return "Give every evidence packet an existing section_key and block_key from the safe manuscript, using the exact section_NN/block_NN aliases and never targeting a heading or nonexistent leaf."
	case reportexecution.ProviderValidationCodeEvidencePacketSource:
		return "Repair every alias listed under INVALID EVIDENCE PACKETS TO REPAIR. Re-read the complete catalog, register a nonempty exact supporting quote of at most 512 UTF-8 bytes with plasma.report_il.sources.quote under its declared source_key, and replace only its source_receipt with the successful opaque receipt. Recheck every other packet and never return source quote text."
	case reportexecution.ProviderValidationCodeEvidencePacketBinding:
		return "Rebuild evidence_packets so every reader-facing content leaf has at least one packet and every evidence_source_keys entry on that leaf has a packet with the same source_key. Each claim must be an exact contiguous substring of its own section_NN/block_NN leaf."
	case reportexecution.ProviderValidationCodeEvidencePacketCoverage:
		return "For every reader-facing leaf, make the union of its exact packet claims cover all factual prose, quote, callout, list-item, or code text. For tables, cover every data-cell value; caption and column labels need no claim coverage. Prefer one whole-leaf claim when one source excerpt supports the whole leaf; otherwise use the fewest nonoverlapping claims needed."
	case reportexecution.ProviderValidationCodeDocumentContract:
		return "Repair only manuscript shape, block fields, evidence keys, and required nonblank document values without changing supported meaning."
	default:
		return "Reconcile the complete response with every server-checked manuscript boundary."
	}
}

func authorSemanticBoundaries() string {
	return `SERVER-CHECKED MANUSCRIPT BOUNDARIES:
- Include two to twelve nonempty sections, each with at least one valid content block.
- Use only schema-supported block fields and keep every table row aligned with its two-to-four columns.
- Cite only source keys you actually read in this attempt. The server treats repeated keys in one content leaf as one citation.
- Never reproduce source keys, generic source labels, or report-system terminology in reader-facing text.
- Keep all required narrative fields nonblank and use a valid document language tag.
- Return a complete terminology array covering every reader-facing foreign proper name or official designation whose source form or source-supplied reading appears in the manuscript, plus every specialist term and foreign unit that needs explanation. Exact source forms are server-checked across the selected sources; any declared reading must appear next to its source form. Specialist terms and foreign units require a colocated first-use explanation.
- Return language_review with all four fields true only after reviewing the complete manuscript and its terminology array for natural target-language prose, names and terms, mechanical translation, unexplained specialist terms or units, and opening duplication.`
}

func readerRepairGuidance(code reportexecution.ProviderValidationCode, packetAliases ...string) string {
	guidance := "Read every server-bound compact source evidence span again, re-read the complete manuscript, and return every section, block, terminology decision, evidence support receipt, and preservation coverage receipt in the exact schema inventory. Do not introduce source keys, URLs, citation locators, source excerpts, or report-process narration."
	target := "every rejected evidence_support_coverage entry"
	validAliases := make([]string, 0, len(packetAliases))
	seenAliases := map[string]bool{}
	for _, alias := range packetAliases {
		if providerValidationPacketAliasPattern.MatchString(alias) && !seenAliases[alias] {
			validAliases = append(validAliases, alias)
			seenAliases[alias] = true
		}
	}
	if len(validAliases) > 0 {
		target = strings.Join(validAliases, ", ")
	}
	switch code {
	case reportexecution.ProviderValidationCodeSourceReadContract:
		guidance += " Complete every server-ordered source batch until remaining_sources is zero before returning."
	case reportexecution.ProviderValidationCodeClaimStrength:
		guidance += " Recheck every relationship, causal, governance, operational-control, surveillance, transport, protection, and traffic-control claim. Remove unsupported mechanisms or lower them to explicit bounded inferences, then set claim_strength_bounded true only after the full manuscript passes."
	case reportexecution.ProviderValidationCodeEvidenceSupportInventory:
		guidance += " Return every schema-declared evidence_support_coverage entry exactly once with no additions or omissions. Reconstruct the complete inventory; specifically include " + target + "."
	case reportexecution.ProviderValidationCodeEvidenceSupportReceipt:
		guidance += " For " + target + ", echo the exact evidence_receipt already supplied for that alias. Do not invent, truncate, normalize, or move a receipt."
	case reportexecution.ProviderValidationCodeEvidenceSupportTarget:
		guidance += " Place " + target + " at the exact schema-bound section_key and block_key that contain its authored claim. Use only existing section_NN/block_NN aliases."
	case reportexecution.ProviderValidationCodeEvidenceSupportBinding:
		guidance += " Keep " + target + " bound to its original packet node and source-backed manuscript leaf. Do not move it to another block or borrow another block's evidence binding."
	case reportexecution.ProviderValidationCodeEvidenceSupportQuote:
		guidance += " For each supported entry in " + target + ", copy a nonempty coverage_quote exactly from one factual value in its final edited block. A preserve=true entry must keep the exact authored claim as that quote; do not return a stale pre-edit quote."
	case reportexecution.ProviderValidationCodeEvidenceSupportUnsupported:
		guidance += " For each entry in " + target + " marked unsupported_removed, remove the entire authored claim from the final block and return coverage_quote null. Never mark a preserve=true entry unsupported_removed."
	case reportexecution.ProviderValidationCodeEvidenceSupportLevel:
		guidance += " For " + target + ", use exactly one allowed support_level: direct_source_statement, bounded_inference, or unsupported_removed. Choose it only after checking the bound compact source span."
	case reportexecution.ProviderValidationCodeEvidenceSupportCoverage:
		guidance += " For the complete factual leaf containing " + target + ", make the union of all supported exact coverage_quote values cover every non-punctuation factual byte in its final prose, quote, callout, list item, code value, and table data cell. Caption and table headers need no coverage. Use multiple compact exact quotes when one quote cannot cover the leaf."
	case reportexecution.ProviderValidationCodeEvidencePacketInventory,
		reportexecution.ProviderValidationCodeEvidencePacketTarget,
		reportexecution.ProviderValidationCodeEvidencePacketSource,
		reportexecution.ProviderValidationCodeEvidencePacketBinding,
		reportexecution.ProviderValidationCodeEvidencePacketCoverage:
		guidance += " Recheck every evidence support entry against its server-bound packet. For every supported packet, return a nonempty coverage_quote copied exactly from its final edited block; supported quotes must cover the complete final factual leaf. For unsupported_removed, remove the authored claim and return coverage_quote null."
	case reportexecution.ProviderValidationCodeLanguageReview:
		guidance += " Correct every natural-language defect and set all review fields true only after the full manuscript passes."
	case reportexecution.ProviderValidationCodeReaderFacingContent:
		guidance += " Remove process narration, repeated source-audit voice, duplicated opening synthesis, low-value SI definitions, internal metadata, and locators. Keep each necessary evidence boundary once beside the distinct claim it qualifies."
	case reportexecution.ProviderValidationCodeReaderOpening:
		guidance += " Rewrite the first paragraph as a direct subject-level historical answer. Move any evidence boundary to the distinct claim it qualifies later in the manuscript."
	case reportexecution.ProviderValidationCodeReaderAuditVoice:
		guidance += " Replace repeated source-review narration with direct subject facts. Keep each genuinely necessary evidence boundary once beside its distinct claim, and do not repeat that limitation elsewhere."
	case reportexecution.ProviderValidationCodeReaderOrdinarySI:
		guidance += " State ordinary SI measurements directly without defining meters, kilometers, elevation, or sea level."
	case reportexecution.ProviderValidationCodeReaderProcess:
		guidance += " Remove report, section, outline, review, and construction narration; let the subject itself create the transition."
	case reportexecution.ProviderValidationCodeReaderMetadata:
		guidance += " Remove every source key, source label, URL, locator, path, and internal reference from reader-facing values."
	case reportexecution.ProviderValidationCodeReaderInternalMachinery:
		guidance += " Remove IL, schema, validation, pipeline, stage, source-selection, and report-system terminology unless that machinery is the mission subject."
	case reportexecution.ProviderValidationCodeReaderUnexplainedTerm:
		guidance += " Inventory every historical route, administrative institution, period-specific office, specialist concept, and other unfamiliar term, even when it is written only in naturalized or transliterated target-language form. Explain each one in its first body sentence or replace it with ordinary wording, then set unfamiliar_terms_explained true."
	case reportexecution.ProviderValidationCodeSupportedDetail:
		guidance += " Restore each exact server-bound detail statement in a schema-bound final block that retains its evidence, and return the correct preservation_coverage alias and receipt. Improve surrounding prose without paraphrasing, generalizing, shortening, or moving the required statement to a block with different evidence."
	case reportexecution.ProviderValidationCodeReportDepth:
		guidance += " Develop the existing manuscript leaves into a self-contained report around the required evidence packet, with natural context, relationships, examples, and implications. Do not pad, repeat, tour sources, force citations, or add unsupported material."
	case reportexecution.ProviderValidationCodeTerminologyInventory:
		guidance += " Return every schema-bound terminology entry exactly once under its original alias and receipt; do not add, omit, reorder, or reassign an entry."
	case reportexecution.ProviderValidationCodeTerminologyPresentation:
		guidance += " Reconcile every exact terminology receipt with the edited manuscript. Use reader_form null only after removing every known form. For a non-null natural form, set retain_as_terminology true only when the reader still needs an enforced explanation or continuity term, and set present_source_aliases true only when those source aliases add reader value."
	case reportexecution.ProviderValidationCodeTerminologyRemoval:
		guidance += " A term marked with reader_form null still has a known source, reading, or reader form in the manuscript. Remove every occurrence of every known form for each null term, or keep the term with a non-null final reader form."
	case reportexecution.ProviderValidationCodeTerminologyRename:
		guidance += " A renamed term still uses its superseded author reader form. Replace every superseded occurrence with the returned final reader form."
	case reportexecution.ProviderValidationCodeTerminologyReaderForm:
		guidance += " Every non-null reader_form must be a valid nonblank natural form that occurs in the edited manuscript, including a first body occurrence when the term requires explanation."
	case reportexecution.ProviderValidationCodeTerminologyFirstUse:
		guidance += " A retained explanatory term lacks a complete first body use. Put its explanation directly in the same first body sentence, list item, or table cell as its final reader form; a title or heading alone is insufficient."
	case reportexecution.ProviderValidationCodeTerminologySourceForm:
		guidance += " For every retained term whose source form differs from the reader form, include the exact source form in the same explained sentence as the first body use."
	case reportexecution.ProviderValidationCodeTerminologySourceReading:
		guidance += " For every retained term with a source-supplied reading, include that exact reading in the same explained sentence as the first body use."
	case reportexecution.ProviderValidationCodeTerminologyAliasPlacement:
		guidance += " When present_source_aliases is false, remove every source form and source-supplied reading from the entire title and manuscript. When true, each alias may occur only inside the explained first body sentence."
	case reportexecution.ProviderValidationCodeTerminologyAliasCollision:
		guidance += " Give each retained term one unique final reader form. Remove or rename forms so no canonical or alias equals, contains, or is contained by a form owned by another term."
	case reportexecution.ProviderValidationCodeTerminologyScriptCoverage:
		guidance += " Account for every Han, Hiragana, or Katakana span remaining in the Korean manuscript with its exact terminology receipt, or replace unnecessary source-script text with the natural reader form."
	case reportexecution.ProviderValidationCodeDocumentContract:
		guidance += " Preserve every schema-bound section, block kind, receipt, hash, list item, and table cell inventory."
	}
	return guidance
}

func sourceFreeRepairPrompt(original, stage, failureClass string, code reportexecution.ProviderValidationCode) string {
	guidance := "Return only strict JSON matching the exact schema."
	if stage == "il_flow" {
		guidance = readerRepairGuidance(code) + " Return only strict JSON matching the exact schema."
	}
	return original + "\nREPAIR ATTEMPT. The prior output was rejected (" + failureClass + "). Start fresh from the complete context above. " + guidance + " The prior output and validator details are intentionally omitted and must not be reconstructed."
}

func strictDecode[T any](raw []byte) (T, error) {
	var value T
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return value, fmt.Errorf("trailing JSON value")
		}
		return value, fmt.Errorf("malformed trailing JSON: %w", err)
	}
	return value, nil
}

func preserveDocumentStructure(original, edited Document) error {
	if len(original.Blocks) != len(edited.Blocks) {
		return fmt.Errorf("flow edit changed block structure or evidence semantics")
	}
	for i := range original.Blocks {
		if original.Blocks[i].Prose != edited.Blocks[i].Prose && !authoredProseKind(original.Blocks[i].Kind) {
			return fmt.Errorf("flow edit changed non-prose block data")
		}
	}
	left, right := original, edited
	left.Blocks = append([]Block(nil), original.Blocks...)
	right.Blocks = append([]Block(nil), edited.Blocks...)
	left.RevisionID = right.RevisionID
	for i := range left.Blocks {
		left.Blocks[i].Prose = ""
		right.Blocks[i].Prose = ""
	}
	if !reflect.DeepEqual(left, right) {
		return fmt.Errorf("flow edit changed document structure or evidence semantics")
	}
	return nil
}

func authoredProseKind(kind string) bool {
	switch kind {
	case "prose", "quote", "callout":
		return true
	default:
		return false
	}
}

func mustMarshal(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}

// SortProductArtifacts provides stable output ordering for persistence adapters.
func SortProductArtifacts(artifacts []ProductArtifact) []ProductArtifact {
	result := append([]ProductArtifact(nil), artifacts...)
	sort.Slice(result, func(i, j int) bool { return result[i].Kind < result[j].Kind })
	return result
}
