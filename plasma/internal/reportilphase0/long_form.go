package reportilphase0

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

const longFormILWorkerLimit = 4

type LongFormArtifactReceipt struct {
	ArtifactID string `json:"artifact_id"`
	SHA256     string `json:"sha256"`
	ByteSize   int    `json:"byte_size"`
	Stage      string `json:"stage"`
}

type LongFormFinalizationReceipt struct {
	Stage    string                  `json:"stage"`
	Artifact LongFormArtifactReceipt `json:"artifact"`
	Reused   bool                    `json:"reused"`
}

type LongFormAuthoringReceipt struct {
	Plan             LongFormArtifactReceipt       `json:"plan"`
	SectionArtifacts []LongFormArtifactReceipt     `json:"section_artifacts"`
	PartArtifacts    []LongFormArtifactReceipt     `json:"part_artifacts"`
	Finalizations    []LongFormFinalizationReceipt `json:"finalizations"`
	Final            LongFormArtifactReceipt       `json:"final"`
	Parts            int                           `json:"parts"`
	Sections         int                           `json:"sections"`
	SectionAuthors   int                           `json:"section_authors"`
	PartEditors      int                           `json:"part_editors"`
	FinalEdits       int                           `json:"final_edits"`
}

func longFormArtifactReceipt(receipt reportilcontract.AuthorWorkspaceReceipt) LongFormArtifactReceipt {
	return LongFormArtifactReceipt{
		ArtifactID: receipt.ArtifactID,
		SHA256:     receipt.SHA256,
		ByteSize:   receipt.ByteSize,
		Stage:      receipt.Stage,
	}
}

func longFormPlanArtifactReceipt(receipt reportilcontract.LongFormPlanReceipt) LongFormArtifactReceipt {
	return LongFormArtifactReceipt{
		ArtifactID: receipt.ArtifactID,
		SHA256:     receipt.SHA256,
		ByteSize:   receipt.ByteSize,
		Stage:      "il_long_form_plan",
	}
}

func appendLongFormFinalization(
	receipt *LongFormAuthoringReceipt,
	stage string,
	workspace reportilcontract.AuthorWorkspaceReceipt,
) {
	artifact := longFormArtifactReceipt(workspace)
	reused := len(receipt.Finalizations) > 0 &&
		receipt.Finalizations[len(receipt.Finalizations)-1].Artifact.ArtifactID == artifact.ArtifactID
	if reused {
		artifact = receipt.Finalizations[len(receipt.Finalizations)-1].Artifact
	}
	receipt.Finalizations = append(
		receipt.Finalizations,
		LongFormFinalizationReceipt{
			Stage: stage, Artifact: artifact, Reused: reused,
		},
	)
	receipt.Final = artifact
}

type longFormAgentResult struct {
	Stage  string
	Result agentexec.AgentResult
}

type longFormStageError struct {
	Stage string
	Cause error
}

func (err *longFormStageError) Error() string { return "long-form authoring stage failed" }
func (err *longFormStageError) Unwrap() error { return err.Cause }

func appendLongFormResults(results []longFormAgentResult, stage string, values ...agentexec.AgentResult) []longFormAgentResult {
	for _, value := range values {
		results = append(results, longFormAgentResult{Stage: stage, Result: value})
	}
	return results
}

func failLongFormStage(stage string, err error) error {
	if err == nil {
		return nil
	}
	return &longFormStageError{Stage: stage, Cause: err}
}

func runLongFormAuthoringGraph(
	ctx context.Context,
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
	memory reportilcontract.EditorialMemory,
	memoryReceipt reportilcontract.EditorialMemoryReceipt,
	citations map[int]SourceCitation,
	selectionApplied bool,
	progress func(string, string) error,
	partsCompleted func(reportilcontract.LongFormPlanReceipt, []reportilcontract.AuthorWorkspaceReceipt, []reportilcontract.AuthorWorkspaceReceipt, int, int) error,
) (Narrative, Document, reportilcontract.AuthorWorkspaceReceipt, LongFormAuthoringReceipt, []longFormAgentResult, error) {
	results := []longFormAgentResult{}
	if err := progress("il_long_form_plan", "started"); err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, LongFormAuthoringReceipt{}, results, failLongFormStage("il_long_form_plan", err)
	}
	plan, planReceipt, planResults, err := runLongFormPlanStage(ctx, config, catalog, memory, memoryReceipt, selectionApplied)
	results = appendLongFormResults(results, "il_long_form_plan", planResults...)
	if err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, LongFormAuthoringReceipt{}, results, failLongFormStage("il_long_form_plan", err)
	}
	if err := progress("il_long_form_plan", "completed"); err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, LongFormAuthoringReceipt{}, results, failLongFormStage("il_long_form_plan", err)
	}

	if err := progress("il_long_form_sections", "started"); err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, LongFormAuthoringReceipt{}, results, failLongFormStage("il_long_form_sections", err)
	}
	sectionArtifacts, sectionResults, err := runLongFormSectionStage(ctx, config, catalog, memory, memoryReceipt, plan, planReceipt)
	results = appendLongFormResults(results, "il_long_form_sections", sectionResults...)
	if err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, LongFormAuthoringReceipt{}, results, failLongFormStage("il_long_form_sections", err)
	}
	if err := progress("il_long_form_sections", "completed"); err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, LongFormAuthoringReceipt{}, results, failLongFormStage("il_long_form_sections", err)
	}

	if err := progress("il_long_form_parts", "started"); err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, LongFormAuthoringReceipt{}, results, failLongFormStage("il_long_form_parts", err)
	}
	partArtifacts, partResults, err := runLongFormPartEditStage(ctx, config, catalog, memory, memoryReceipt, plan, planReceipt, sectionArtifacts)
	results = appendLongFormResults(results, "il_long_form_parts", partResults...)
	if err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, LongFormAuthoringReceipt{}, results, failLongFormStage("il_long_form_parts", err)
	}
	if err := progress("il_long_form_parts", "completed"); err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, LongFormAuthoringReceipt{}, results, failLongFormStage("il_long_form_parts", err)
	}
	sectionReceipts := make([]reportilcontract.AuthorWorkspaceReceipt, 0, reportilcontract.LongFormPlanSectionCount(plan))
	partReceipts := make([]reportilcontract.AuthorWorkspaceReceipt, 0, len(plan.Parts))
	for _, part := range plan.Parts {
		for _, section := range part.Sections {
			sectionReceipts = append(sectionReceipts, sectionArtifacts[section.SectionKey])
		}
		partReceipts = append(partReceipts, partArtifacts[part.PartKey])
	}
	if partsCompleted != nil {
		if err := partsCompleted(planReceipt, sectionReceipts, partReceipts, len(plan.Parts), reportilcontract.LongFormPlanSectionCount(plan)); err != nil {
			return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, LongFormAuthoringReceipt{}, results, failLongFormStage("il_long_form_parts", err)
		}
	}

	if err := progress("il_long_form_final", "started"); err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, LongFormAuthoringReceipt{}, results, failLongFormStage("il_long_form_final", err)
	}
	document, authorWorkspace, finalResult, edits, err := runLongFormFinalEditStage(ctx, config, catalog, memory, memoryReceipt, plan, planReceipt, partArtifacts)
	results = appendLongFormResults(results, "il_long_form_final", finalResult)
	if err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, LongFormAuthoringReceipt{}, results, failLongFormStage("il_long_form_final", err)
	}
	if err := progress("il_long_form_final", "completed"); err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, LongFormAuthoringReceipt{}, results, failLongFormStage("il_long_form_final", err)
	}
	contractID, documentID := config.NewID("narrative"), config.NewID("doc")
	narrative, semantic, err := compileLongFormAuthorDocument(
		document, contractID, documentID, config.MissionObjective, catalog,
		editorialMemorySourceReadReceipt(catalog), citations,
	)
	if err != nil {
		return Narrative{}, Document{}, reportilcontract.AuthorWorkspaceReceipt{}, LongFormAuthoringReceipt{}, results, failLongFormStage("il_long_form_final", longFormProviderValidation(err))
	}
	receipt := LongFormAuthoringReceipt{
		Plan:           longFormPlanArtifactReceipt(planReceipt),
		Final:          longFormArtifactReceipt(authorWorkspace),
		Parts:          len(plan.Parts),
		Sections:       reportilcontract.LongFormPlanSectionCount(plan),
		SectionAuthors: reportilcontract.LongFormPlanSectionCount(plan),
		PartEditors:    len(plan.Parts), FinalEdits: edits,
	}
	for _, part := range plan.Parts {
		for _, section := range part.Sections {
			receipt.SectionArtifacts = append(receipt.SectionArtifacts, longFormArtifactReceipt(sectionArtifacts[section.SectionKey]))
		}
		receipt.PartArtifacts = append(receipt.PartArtifacts, longFormArtifactReceipt(partArtifacts[part.PartKey]))
	}
	return narrative, semantic, authorWorkspace, receipt, results, nil
}

func runLongFormPlanStage(
	ctx context.Context,
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
	memory reportilcontract.EditorialMemory,
	memoryReceipt reportilcontract.EditorialMemoryReceipt,
	selectionApplied bool,
) (reportilcontract.LongFormPlan, reportilcontract.LongFormPlanReceipt, []agentexec.AgentResult, error) {
	prompt := longFormPlanPrompt(config, catalog, memory, selectionApplied)
	request := longFormIsolatedRequest(config, catalog, "il_long_form_plan", prompt, nil, memoryReceipt)
	if len(memory.Accounts) > 0 {
		request.ExtraMCPTools = []string{
			reportilcontract.EditorialMemoryReadTool,
			reportilcontract.LongFormPlanSubmitTool,
		}
	} else {
		request.ReportILSources.LongFormSourceKeys = sourceCatalogKeys(catalog)
		request.ExtraMCPTools = []string{
			reportilcontract.SourceListTool,
			reportilcontract.SourceReadTool,
			reportilcontract.LongFormPlanSubmitTool,
		}
	}
	result, err := config.Provider.Run(ctx, request)
	results := []agentexec.AgentResult{result}
	if err != nil {
		return reportilcontract.LongFormPlan{}, reportilcontract.LongFormPlanReceipt{}, results, &providerStageError{reason: reportexecution.ProviderFailureReasonTransport, cause: err}
	}
	plan, receipt, err := config.AuthorDocuments.ReadReportILLongFormPlan(
		ctx, config.MissionID, request.ToolSessionID, catalog,
	)
	if err != nil {
		return reportilcontract.LongFormPlan{}, reportilcontract.LongFormPlanReceipt{}, results, longFormProviderValidation(err)
	}
	var memoryPointer *reportilcontract.EditorialMemory
	if len(memory.Accounts) > 0 {
		memoryPointer = &memory
	}
	if err := reportilcontract.ValidateLongFormPlan(plan, memoryPointer, catalog); err != nil {
		return reportilcontract.LongFormPlan{}, reportilcontract.LongFormPlanReceipt{}, results, longFormProviderValidation(err)
	}
	if plan.Language != config.TargetLanguage {
		return reportilcontract.LongFormPlan{}, reportilcontract.LongFormPlanReceipt{}, results, longFormProviderValidation(fmt.Errorf("long-form plan target language differs from the product request"))
	}
	if err := longFormProgress(config, LongFormProgressEvent{
		Kind: "plan", Status: "completed", Plan: &plan, Title: plan.Title,
	}); err != nil {
		return reportilcontract.LongFormPlan{}, reportilcontract.LongFormPlanReceipt{}, results, err
	}
	return plan, receipt, results, nil
}

func longFormProgress(config ProductConfig, event LongFormProgressEvent) error {
	if config.LongFormProgress == nil {
		return nil
	}
	return config.LongFormProgress(event)
}

func longFormCoordinates(plan reportilcontract.LongFormPlan, partKey, sectionKey string) (int, int) {
	for partIndex, part := range plan.Parts {
		if part.PartKey != partKey {
			continue
		}
		if sectionKey == "" {
			return partIndex + 1, 0
		}
		for sectionIndex, section := range part.Sections {
			if section.SectionKey == sectionKey {
				return partIndex + 1, sectionIndex + 1
			}
		}
		return partIndex + 1, 0
	}
	return 0, 0
}

func runLongFormSectionStage(
	ctx context.Context,
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
	memory reportilcontract.EditorialMemory,
	memoryReceipt reportilcontract.EditorialMemoryReceipt,
	plan reportilcontract.LongFormPlan,
	planReceipt reportilcontract.LongFormPlanReceipt,
) (map[string]reportilcontract.AuthorWorkspaceReceipt, []agentexec.AgentResult, error) {
	type task struct {
		part    reportilcontract.LongFormPart
		section reportilcontract.LongFormSection
	}
	tasks := []task{}
	for _, part := range plan.Parts {
		for _, section := range part.Sections {
			tasks = append(tasks, task{part: part, section: section})
		}
	}
	artifacts := make(map[string]reportilcontract.AuthorWorkspaceReceipt, len(tasks))
	results := make([]agentexec.AgentResult, len(tasks))
	sem := make(chan struct{}, longFormILWorkerLimit)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	for index, item := range tasks {
		index, item := index, item
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			partIndex, sectionIndex := longFormCoordinates(plan, item.part.PartKey, item.section.SectionKey)
			if err := longFormProgress(config, LongFormProgressEvent{
				Kind: "section", Status: "started", PartIndex: partIndex,
				SectionIndex: sectionIndex, Title: item.section.Title,
			}); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			receipt, result, err := runLongFormSectionAuthor(
				ctx, config, catalog, memory, memoryReceipt, plan, planReceipt, item.part, item.section,
			)
			results[index] = result
			status := "completed"
			if err != nil {
				status = "failed"
			}
			progressErr := longFormProgress(config, LongFormProgressEvent{
				Kind: "section", Status: status, PartIndex: partIndex,
				SectionIndex: sectionIndex, Title: item.section.Title,
			})
			mu.Lock()
			defer mu.Unlock()
			if err != nil && firstErr == nil {
				firstErr = err
			}
			if progressErr != nil && firstErr == nil {
				firstErr = progressErr
			}
			if err == nil && progressErr == nil {
				artifacts[item.section.SectionKey] = receipt
			}
		}()
	}
	wg.Wait()
	if firstErr != nil {
		return nil, results, firstErr
	}
	return artifacts, results, nil
}

func runLongFormSectionAuthor(
	ctx context.Context,
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
	memory reportilcontract.EditorialMemory,
	memoryReceipt reportilcontract.EditorialMemoryReceipt,
	plan reportilcontract.LongFormPlan,
	planReceipt reportilcontract.LongFormPlanReceipt,
	part reportilcontract.LongFormPart,
	section reportilcontract.LongFormSection,
) (reportilcontract.AuthorWorkspaceReceipt, agentexec.AgentResult, error) {
	prompt := longFormSectionPrompt(config, plan, part, section, memory)
	request := longFormIsolatedRequest(config, catalog, "il_long_form_section", prompt, nil, memoryReceipt)
	request.ReportILSources.LongFormPlanArtifactID = planReceipt.ArtifactID
	request.ReportILSources.LongFormPlanSHA256 = planReceipt.SHA256
	request.ReportILSources.LongFormPartKey = part.PartKey
	request.ReportILSources.LongFormSectionKey = section.SectionKey
	if len(memory.Accounts) > 0 {
		request.ReportILSources.LongFormSectionAccountKeys = append([]string(nil), section.EditorialAccountKeys...)
		request.ExtraMCPTools = []string{
			reportilcontract.EditorialMemoryReadTool,
			reportilcontract.LongFormPlanReadTool,
			reportilcontract.LongFormDocumentStartTool,
			reportilcontract.LongFormDocumentAppendTool,
			reportilcontract.LongFormDocumentReadTool,
			reportilcontract.LongFormDocumentReplaceTool,
			reportilcontract.LongFormDocumentCorrectBlockTool,
			reportilcontract.LongFormDocumentFinalizeTool,
		}
	} else {
		request.ReportILSources.LongFormSourceKeys = append([]string(nil), section.EvidenceSourceKeys...)
		request.ExtraMCPTools = []string{
			reportilcontract.SourceReadTool,
			reportilcontract.LongFormPlanReadTool,
			reportilcontract.LongFormDocumentStartTool,
			reportilcontract.LongFormDocumentAppendTool,
			reportilcontract.LongFormDocumentReadTool,
			reportilcontract.LongFormDocumentReplaceTool,
			reportilcontract.LongFormDocumentCorrectBlockTool,
			reportilcontract.LongFormDocumentFinalizeTool,
		}
	}
	result, err := config.Provider.Run(ctx, request)
	if err != nil {
		return reportilcontract.AuthorWorkspaceReceipt{}, result, &providerStageError{
			reason: reportexecution.ProviderFailureReasonTransport, cause: err,
		}
	}
	authored, receipt, err := config.AuthorDocuments.ReadReportILLongFormStageDocument(
		ctx, config.MissionID, request.ToolSessionID, "il_long_form_section", catalog,
	)
	if err != nil {
		return reportilcontract.AuthorWorkspaceReceipt{}, result, longFormProviderValidation(err)
	}
	if len(authored.Parts) != 1 || authored.Parts[0].PartKey != part.PartKey ||
		len(authored.Parts[0].Sections) != 1 || authored.Parts[0].Sections[0].SectionKey != section.SectionKey {
		return reportilcontract.AuthorWorkspaceReceipt{}, result, longFormProviderValidation(fmt.Errorf("long-form Section artifact inventory changed"))
	}
	if err := validateLongFormDocumentBindings(authored, memory, catalog, false); err != nil {
		return reportilcontract.AuthorWorkspaceReceipt{}, result, longFormProviderValidation(err)
	}
	if err := reportilcontract.ValidateLongFormRepresentationObligations(
		section,
		authored.Parts[0].Sections[0],
	); err != nil {
		return reportilcontract.AuthorWorkspaceReceipt{}, result, longFormProviderValidation(err)
	}
	if err := validateLongFormSectionAccountRealization(
		section,
		authored.Parts[0].Sections[0],
	); err != nil {
		return reportilcontract.AuthorWorkspaceReceipt{}, result, longFormProviderValidation(err)
	}
	return receipt, result, nil
}

func runLongFormPartEditStage(
	ctx context.Context,
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
	memory reportilcontract.EditorialMemory,
	memoryReceipt reportilcontract.EditorialMemoryReceipt,
	plan reportilcontract.LongFormPlan,
	planReceipt reportilcontract.LongFormPlanReceipt,
	sectionArtifacts map[string]reportilcontract.AuthorWorkspaceReceipt,
) (map[string]reportilcontract.AuthorWorkspaceReceipt, []agentexec.AgentResult, error) {
	results := []agentexec.AgentResult{}
	partArtifacts := make(map[string]reportilcontract.AuthorWorkspaceReceipt, len(plan.Parts))
	for partOffset, part := range plan.Parts {
		partIndex := partOffset + 1
		inputs := make([]reportilcontract.AuthorWorkspaceReceipt, 0, len(part.Sections))
		for _, section := range part.Sections {
			receipt, ok := sectionArtifacts[section.SectionKey]
			if !ok {
				return nil, results, longFormProviderValidation(fmt.Errorf("long-form Section artifact is missing"))
			}
			inputs = append(inputs, receipt)
		}
		if err := longFormProgress(config, LongFormProgressEvent{
			Kind: "part_edit", Status: "started", PartIndex: partIndex, Title: part.Title,
		}); err != nil {
			return nil, results, err
		}
		prompt := longFormPartEditPrompt(config, part)
		request := longFormIsolatedRequest(config, catalog, "il_long_form_part", prompt, nil, memoryReceipt)
		bindLongFormInputs(&request, inputs)
		request.ReportILSources.LongFormPlanArtifactID = planReceipt.ArtifactID
		request.ReportILSources.LongFormPlanSHA256 = planReceipt.SHA256
		request.ReportILSources.LongFormPartKey = part.PartKey
		request.ExtraMCPTools = longFormEditTools(len(memory.Accounts) > 0)
		result, runErr := config.Provider.Run(ctx, request)
		results = append(results, result)
		if runErr != nil {
			providerErr := &providerStageError{reason: reportexecution.ProviderFailureReasonTransport, cause: runErr}
			if progressErr := longFormProgress(config, LongFormProgressEvent{
				Kind: "part_edit", Status: "failed", PartIndex: partIndex, Title: part.Title,
			}); progressErr != nil {
				return nil, results, errors.Join(providerErr, progressErr)
			}
			return nil, results, providerErr
		}
		authored, receipt, err := config.AuthorDocuments.ReadReportILLongFormStageDocument(
			ctx, config.MissionID, request.ToolSessionID, "il_long_form_part", catalog,
		)
		if err != nil {
			providerErr := longFormProviderValidation(err)
			if progressErr := longFormProgress(config, LongFormProgressEvent{
				Kind: "part_edit", Status: "failed", PartIndex: partIndex, Title: part.Title,
			}); progressErr != nil {
				return nil, results, errors.Join(providerErr, progressErr)
			}
			return nil, results, providerErr
		}
		if len(authored.Parts) != 1 || authored.Parts[0].PartKey != part.PartKey || len(authored.Parts[0].Sections) != len(part.Sections) {
			if progressErr := longFormProgress(config, LongFormProgressEvent{
				Kind: "part_edit", Status: "failed", PartIndex: partIndex, Title: part.Title,
			}); progressErr != nil {
				return nil, results, progressErr
			}
			return nil, results, longFormProviderValidation(fmt.Errorf("long-form Part artifact inventory changed"))
		}
		if err := validateLongFormDocumentBindings(authored, memory, catalog, false); err != nil {
			if progressErr := longFormProgress(config, LongFormProgressEvent{
				Kind: "part_edit", Status: "failed", PartIndex: partIndex, Title: part.Title,
			}); progressErr != nil {
				return nil, results, progressErr
			}
			return nil, results, longFormProviderValidation(err)
		}
		if err := validateLongFormPartRepresentationObligations(part, authored.Parts[0]); err != nil {
			if progressErr := longFormProgress(config, LongFormProgressEvent{
				Kind: "part_edit", Status: "failed", PartIndex: partIndex, Title: part.Title,
			}); progressErr != nil {
				return nil, results, progressErr
			}
			return nil, results, longFormProviderValidation(err)
		}
		if err := longFormProgress(config, LongFormProgressEvent{
			Kind: "part_edit", Status: "completed", PartIndex: partIndex, Title: part.Title,
		}); err != nil {
			return nil, results, err
		}
		partArtifacts[part.PartKey] = receipt
	}
	return partArtifacts, results, nil
}

func runLongFormFinalEditStage(
	ctx context.Context,
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
	memory reportilcontract.EditorialMemory,
	memoryReceipt reportilcontract.EditorialMemoryReceipt,
	plan reportilcontract.LongFormPlan,
	planReceipt reportilcontract.LongFormPlanReceipt,
	partArtifacts map[string]reportilcontract.AuthorWorkspaceReceipt,
) (reportilcontract.AuthorDocument, reportilcontract.AuthorWorkspaceReceipt, agentexec.AgentResult, int, error) {
	inputs := make([]reportilcontract.AuthorWorkspaceReceipt, 0, len(plan.Parts))
	for _, part := range plan.Parts {
		receipt, ok := partArtifacts[part.PartKey]
		if !ok {
			return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, agentexec.AgentResult{}, 0, longFormProviderValidation(fmt.Errorf("long-form Part artifact is missing"))
		}
		inputs = append(inputs, receipt)
	}
	prompt := longFormFinalEditPrompt(config)
	request := longFormIsolatedRequest(config, catalog, "il_long_form_final", prompt, nil, memoryReceipt)
	bindLongFormInputs(&request, inputs)
	request.ReportILSources.LongFormPlanArtifactID = planReceipt.ArtifactID
	request.ReportILSources.LongFormPlanSHA256 = planReceipt.SHA256
	request.ExtraMCPTools = longFormEditTools(len(memory.Accounts) > 0)
	result, runErr := config.Provider.Run(ctx, request)
	if runErr != nil {
		return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, result, 0, &providerStageError{reason: reportexecution.ProviderFailureReasonTransport, cause: runErr}
	}
	authored, receipt, err := config.AuthorDocuments.ReadReportILLongFormStageDocument(
		ctx, config.MissionID, request.ToolSessionID, "il_long_form_final", catalog,
	)
	if err != nil {
		return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, result, 0, longFormProviderValidation(err)
	}
	if err := validateLongFormDocumentBindings(authored, memory, catalog, true); err != nil {
		return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, result, 0, longFormProviderValidation(err)
	}
	if err := validateLongFormRepresentationObligations(plan, authored); err != nil {
		return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, result, 0, longFormProviderValidation(err)
	}
	return authored, receipt, result, receipt.Replacements, nil
}

func validateLongFormPartRepresentationObligations(planned reportilcontract.LongFormPart, authored reportilcontract.AuthorPart) error {
	if len(planned.Sections) != len(authored.Sections) {
		return fmt.Errorf("long-form Part representation inventory changed")
	}
	for index, plannedSection := range planned.Sections {
		if err := reportilcontract.ValidateLongFormRepresentationObligations(plannedSection, authored.Sections[index]); err != nil {
			return err
		}
		if err := validateLongFormSectionAccountRealization(plannedSection, authored.Sections[index]); err != nil {
			return err
		}
	}
	return nil
}

func validateLongFormSectionAccountRealization(planned reportilcontract.LongFormSection, authored reportilcontract.AuthorSection) error {
	if len(planned.EditorialAccountKeys) == 0 {
		return nil
	}
	realized := make(map[string]bool, len(planned.EditorialAccountKeys))
	for _, block := range authored.Blocks {
		for _, accountKey := range block.EditorialAccountKeys {
			realized[accountKey] = true
		}
	}
	for _, accountKey := range planned.EditorialAccountKeys {
		if !realized[accountKey] {
			return fmt.Errorf("planned long-form editorial account is missing from its Section")
		}
	}
	return nil
}

func validateLongFormRepresentationObligations(plan reportilcontract.LongFormPlan, authored reportilcontract.AuthorDocument) error {
	if len(plan.Parts) != len(authored.Parts) {
		return fmt.Errorf("long-form representation inventory changed")
	}
	for index, plannedPart := range plan.Parts {
		if err := validateLongFormPartRepresentationObligations(plannedPart, authored.Parts[index]); err != nil {
			return err
		}
	}
	return nil
}

func bindLongFormInputs(request *agentexec.AgentRequest, inputs []reportilcontract.AuthorWorkspaceReceipt) {
	for _, receipt := range inputs {
		request.ReportILSources.LongFormInputArtifactIDs = append(request.ReportILSources.LongFormInputArtifactIDs, receipt.ArtifactID)
		request.ReportILSources.LongFormInputSHA256s = append(request.ReportILSources.LongFormInputSHA256s, receipt.SHA256)
	}
}

func longFormEditTools(withMemory bool) []string {
	tools := []string{}
	if withMemory {
		tools = append(tools, reportilcontract.EditorialMemoryReadTool)
	}
	return append(tools,
		reportilcontract.LongFormPlanReadTool,
		reportilcontract.LongFormDocumentStartTool,
		reportilcontract.LongFormDocumentReadTool,
		reportilcontract.LongFormDocumentReplaceTool,
		reportilcontract.LongFormDocumentFinalizeTool,
	)
}

func sourceCatalogKeys(catalog reportilcontract.SourceCatalog) []string {
	keys := make([]string, 0, len(catalog.Sources))
	for _, entry := range catalog.Sources {
		keys = append(keys, entry.SourceKey)
	}
	return keys
}

func compileLongFormAuthorDocument(
	authored reportilcontract.AuthorDocument,
	contractID,
	documentID,
	missionObjective string,
	catalog reportilcontract.SourceCatalog,
	receipt reportilcontract.SourceReadReceipt,
	citations map[int]SourceCitation,
) (Narrative, Document, error) {
	if err := reportilcontract.ValidateLongFormAuthorDocument(authored, catalog); err != nil {
		return Narrative{}, Document{}, err
	}
	document := Document{
		SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily,
		DocumentID: documentID, RevisionID: "draft", NarrativeContractID: contractID,
		Title: authored.Title, Language: authored.Language,
		Provenance: map[string]string{"source": "accepted mission sources"},
	}
	used := map[string]bool{}
	sectionIDs := []string{}
	sectionTitles := []string{}
	sectionIndex := 0
	for partIndex, part := range authored.Parts {
		partID := serverAuthorPartID(documentID, partIndex, used)
		document.Blocks = append(document.Blocks, Block{NodeID: partID, Kind: "section", Level: 2, Title: part.Title})
		for _, section := range part.Sections {
			sectionID := serverAuthorSectionID(documentID, sectionIndex, section.Title, used)
			sectionIDs = append(sectionIDs, sectionID)
			sectionTitles = append(sectionTitles, section.Title)
			document.Blocks = append(document.Blocks, Block{NodeID: sectionID, Kind: "section", ParentNodeID: partID, Level: 3, Title: section.Title})
			for blockIndex, source := range section.Blocks {
				var block Block
				var err error
				if source.Kind == "equation" {
					block, err = compileAuthorEquationBlock(documentID, sectionID, sectionIndex, blockIndex, source, used)
				} else {
					draft := reportFirstBlockDraft{
						Kind: source.Kind, Prose: source.Prose, Items: source.Items, Code: source.Code,
						Language: source.Language, EditorialAccountKeys: source.EditorialAccountKeys,
						EvidenceSourceKeys: source.EvidenceSourceKeys,
					}
					if source.Table != nil {
						draft.Table = documentTableDraftFromAuthorTable(source.Table)
					}
					block, err = compileDocumentBlockDraft(documentID, sectionID, sectionIndex, blockIndex, draft.documentDraft(), used)
				}
				if err != nil {
					return Narrative{}, Document{}, err
				}
				if err := compileBlockEvidence(&document, &block, catalog, receipt, source.EvidenceSourceKeys, citations); err != nil {
					return Narrative{}, Document{}, err
				}
				document.Blocks = append(document.Blocks, block)
			}
			sectionIndex++
		}
	}
	sections := readerSections(document)
	if len(sections) < reportilcontract.MinLongFormSections {
		return Narrative{}, Document{}, fmt.Errorf("long-form manuscript requires leaf Sections")
	}
	opening := readerSectionBoundaryText(sections[0], false)
	conclusion := readerSectionBoundaryText(sections[len(sections)-1], true)
	if opening == "" || conclusion == "" {
		return Narrative{}, Document{}, fmt.Errorf("long-form manuscript requires opening and conclusion text")
	}
	centralQuestion := strings.TrimSpace(missionObjective)
	if centralQuestion == "" {
		centralQuestion = document.Title
	}
	narrative := Narrative{
		SchemaVersion: NarrativeSchemaVersion, ContractID: contractID, DocumentID: documentID,
		CentralQuestion: centralQuestion, ReaderTakeaway: conclusion, Throughline: opening,
		ConclusionObligations: []string{conclusion}, VoiceAndTone: "Reader-facing authored long-form report",
	}
	for index, title := range sectionTitles {
		narrative.ReaderJourney = append(narrative.ReaderJourney, title)
		narrative.ArgumentArc = append(narrative.ArgumentArc, title)
		narrative.SectionRoles = append(narrative.SectionRoles, SectionRole{SectionID: sectionIDs[index], Role: authorSectionRole(index, len(sectionIDs)), QuestionAnswered: title})
	}
	if err := validateAuthorFacingDocument(document, missionObjective); err != nil {
		return Narrative{}, Document{}, err
	}
	if err := validateProductDocument(narrative, document); err != nil {
		return Narrative{}, Document{}, err
	}
	return narrative, document, nil
}

func serverAuthorPartID(documentID string, index int, used map[string]bool) string {
	return serverDocumentNodeID(documentID, "part", -1, index, used)
}

func validateLongFormDocumentBindings(document reportilcontract.AuthorDocument, memory reportilcontract.EditorialMemory, catalog reportilcontract.SourceCatalog, complete bool) error {
	var err error
	if complete {
		err = reportilcontract.ValidateLongFormAuthorDocument(document, catalog)
	} else {
		err = reportilcontract.ValidateLongFormAuthorFragment(document, catalog)
	}
	if err != nil {
		return err
	}
	if len(memory.Accounts) > 0 {
		if complete {
			return reportilcontract.ValidateAuthorDocumentWithMemory(document, memory, catalog)
		}
		return reportilcontract.ValidateLongFormAuthorFragmentWithMemory(document, memory, catalog)
	}
	return nil
}

func compileAuthorEquationBlock(
	documentID,
	parent string,
	sectionIndex,
	blockIndex int,
	source reportilcontract.AuthorBlock,
	used map[string]bool,
) (Block, error) {
	if source.Equation == nil || strings.TrimSpace(source.Equation.Expression) == "" || source.Equation.Notation != "latex" {
		return Block{}, fmt.Errorf("authored equation block is invalid")
	}
	return Block{
		NodeID:       serverDocumentNodeID(documentID, parent, sectionIndex, blockIndex, used),
		Kind:         "equation",
		ParentNodeID: parent,
		Equation: &Equation{
			Expression: source.Equation.Expression,
			Notation:   source.Equation.Notation,
		},
	}, nil
}

func documentTableDraftFromAuthorTable(table *reportilcontract.AuthorTable) *documentTableDraft {
	if table == nil {
		return nil
	}
	result := &documentTableDraft{Caption: table.Caption}
	if len(table.Columns) > 0 {
		result.Column1 = table.Columns[0]
	}
	if len(table.Columns) > 1 {
		result.Column2 = table.Columns[1]
	}
	if len(table.Columns) > 2 {
		result.Column3 = authorStringPointer(table.Columns[2])
	}
	if len(table.Columns) > 3 {
		result.Column4 = authorStringPointer(table.Columns[3])
	}
	for _, row := range table.Rows {
		item := documentTableRowDraft{}
		if len(row.Cells) > 0 {
			item.Cell1 = row.Cells[0]
		}
		if len(row.Cells) > 1 {
			item.Cell2 = row.Cells[1]
		}
		if len(row.Cells) > 2 {
			item.Cell3 = authorStringPointer(row.Cells[2])
		}
		if len(row.Cells) > 3 {
			item.Cell4 = authorStringPointer(row.Cells[3])
		}
		result.Rows = append(result.Rows, item)
	}
	return result
}

func longFormProviderValidation(err error) error {
	return &providerStageError{reason: reportexecution.ProviderFailureReasonSemanticValidation, cause: withValidationCode(reportexecution.ProviderValidationCodeDocumentContract, err)}
}

func longFormPlanPrompt(config ProductConfig, catalog reportilcontract.SourceCatalog, memory reportilcontract.EditorialMemory, selectionApplied bool) string {
	objective := strings.TrimSpace(config.MissionObjective)
	if objective == "" {
		objective = strings.TrimSpace(config.Title)
	}
	material := "Read the complete source-backed editorial memory through EOF before planning."
	if len(memory.Accounts) == 0 {
		material = "Read every frozen accepted source completely through EOF before planning."
	}
	return fmt.Sprintf(`Plan a genuinely long-form reader report in the server-owned target language %q. Start from the subject and its central answer, then follow the reader's next questions. Build 2-5 Parts and 6-14 independently draftable Sections total. The server assigns role %q to every Section except the final Section and role %q to the final Section, so do not submit a role field. Make the final Section a standalone reader conclusion that ends with a sharper historical or technical judgment, not a source recap, site guide, viewing checklist, or generic summary.

Give each Section exclusive explanatory ownership: its purpose must name the distinct question, relationship, mechanism, episode, or judgment it advances and the already-established facts it must not reintroduce. Shared facts may appear again only when the later Section draws a genuinely new inference from them.

Every essential editorial account must belong to at least one Section. Treat importance as a content-preservation signal, not permission to make every account equally prominent: supporting accounts may be omitted when they add no material reader value, while essential accounts must cross the plan boundary even when a compact integration into another Section is better than another heading. Before submission, reconcile the complete memory account inventory against all Section bindings and repair any omission.

For each Section, submit a representations array containing only source-backed explanatory forms whose actual realization would materially improve understanding: %q for a genuine comparison, %q for executable or illustrative code/API/command/configuration, %q for a meaningful mathematical relationship, %q for a concrete calculation or traced case, %q for measured comparisons with conditions, %q for an operational or decision procedure, and %q for a source-supported flow or structural diagram. Use [] when continuous prose is the best form. This is an obligation-transfer decision, not a quota: do not assign forms merely for visual variety, but do not flatten usable source-native technical material into prose or omit it because prose is easier. When source-backed technical material supports more than one useful form, preserve the forms that answer different reader questions—for example a general cost equation plus a concrete break-even calculation, or an API request pattern plus a provider comparison—rather than treating one form as a receipt for all of them. Prefer executable API, configuration, command, and lifecycle examples over pseudo-code when the memory preserves the real syntax.

Use MCP tools for all source-backed input and plan submission; do not return the plan in your terminal response. The server supplies the target language, so do not submit a language field. Do not expose source tours, coverage receipts, schemas, validators, MCP, IL, providers, prompts, or pipeline stages in reader-facing titles or purposes.

MISSION OBJECTIVE:
%s

ADDITIONAL DIRECTION:
%s

%s

Submit the complete plan exactly once with plasma.report_il.long_form.plan.submit after completing the required reads. Your terminal response may only briefly confirm submission. Source selection was applied: %t. Frozen source count: %d.`,
		config.TargetLanguage,
		reportilcontract.LongFormSectionRoleBody,
		reportilcontract.LongFormSectionRoleConclusion,
		reportilcontract.LongFormRepresentationTable,
		reportilcontract.LongFormRepresentationCode,
		reportilcontract.LongFormRepresentationEquation,
		reportilcontract.LongFormRepresentationWorkedExample,
		reportilcontract.LongFormRepresentationBenchmark,
		reportilcontract.LongFormRepresentationChecklist,
		reportilcontract.LongFormRepresentationDiagram,
		objective,
		strings.TrimSpace(config.Direction),
		material,
		selectionApplied,
		len(catalog.Sources),
	)
}

func longFormSectionPrompt(config ProductConfig, plan reportilcontract.LongFormPlan, part reportilcontract.LongFormPart, section reportilcontract.LongFormSection, memory reportilcontract.EditorialMemory) string {
	materialStep := "Read every bound frozen source completely from offset 0 through EOF."
	if len(memory.Accounts) > 0 {
		materialStep = "Read the complete bound editorial memory through EOF."
	}
	return fmt.Sprintf(`Write one substantial Section in the server-owned target language %q for a long-form reader report. The plan and source-backed material are available only through MCP tools; do not ask for or return the manuscript in your terminal response. Write only the bound Section, not an outline, source summary, or pipeline explanation. Obey the Section purpose as its exclusive explanatory ownership: do not reintroduce facts that earlier Sections own merely to make this Section self-contained. Begin with this Section's new answer, scene, relationship, or mechanism, preserve concrete dates, people, roles, structures, cases, and relevant uncertainty, and end where the next Section can naturally continue.

The bound plan carries source-backed representation obligations. Realize every listed representation in the form that best preserves its explanatory payload:
- %q: use a table block with meaningful dimensions and source-backed cells.
- %q: use a code block that preserves runnable or illustrative syntax and its language when known; explain its role in nearby prose.
- %q: use an equation block with a complete LaTeX expression, then define variables, units, assumptions, and interpretation in nearby prose.
- %q: walk through concrete inputs, steps, and result rather than giving only the takeaway.
- %q: retain measured values together with conditions and comparison basis, normally in a table or compact prose when tabulation would distort it.
- %q: use a list for a genuinely ordered operational or decision procedure and explain the decision logic nearby.
- %q: preserve the source-supported flow with a compact code/diagram block or a clearly ordered list, not an invented decorative image.
Do not add unplanned forms for visual variety. Conversely, do not flatten a planned form into generic prose or omit it merely to finish quickly. Structured blocks are part of the explanation, not appendices.

If the bound Section role is %q, write a standalone conclusion rather than a recap or visitor guide. Synthesize the accumulated report into a sharper historical or technical judgment, explain what that judgment changes in the reader's understanding, and end the report decisively. Do not mention source keys, account keys, MCP, IL, schemas, validators, prompts, providers, or authoring stages in prose.

REPORT: %s
PART: %s
SECTION: %s
SECTION ROLE: %s
PLANNED REPRESENTATIONS: %q
DIRECTION: %s

WORKFLOW:
1. Read the bound Part/Section plan through EOF.
2. %s
3. Start the long-form document workspace exactly once. The server seeds the bound Part and Section identity.
4. Develop the Section through as many complete reader-facing blocks as its distinct explanation and planned representations require. A pair of short prose blocks is not a completion target. Start with prose, use table, code, equation, list, quote, or callout blocks only where they improve the actual explanation, and define every equation's terms in nearby prose. Every bound editorial account must appear in at least one block; preserve its useful syntax, relationship, cases, conditions, and operational detail instead of citing the account while writing only its high-level takeaway. Every block must carry exact evidence_source_keys and, when using editorial memory, exact editorial_account_keys. Never append schema probes, test values, scratch examples, or placeholder blocks merely to learn a tool shape; tool calls mutate the reader document.
5. Write plain reader prose inside prose, quote, and callout blocks. Do not put Markdown emphasis markers, headings, lists, code fences, tables, or display-math delimiters inside prose fields; use the matching structured block instead. A label such as a worked-example heading must be an ordinary complete sentence or part of the surrounding explanation, not literal Markdown asterisk markup.
6. Read the complete assembled Section from offset 0 through EOF. Confirm that its distinct mechanism, evidence, examples, limits, every bound account, and every planned representation are actually present and understandable without an internal source tour. If a block was appended incorrectly, delete or fully replace that exact block with plasma.report_il.long_form.document.correct_block before finalization; do not ask later immutable stages to preserve it.
7. Make only bounded exact replacements or the necessary mistaken-block correction; after any edit, reread through EOF.
8. Finalize exactly once. Your terminal response may only briefly confirm finalization.`,
		config.TargetLanguage,
		reportilcontract.LongFormRepresentationTable,
		reportilcontract.LongFormRepresentationCode,
		reportilcontract.LongFormRepresentationEquation,
		reportilcontract.LongFormRepresentationWorkedExample,
		reportilcontract.LongFormRepresentationBenchmark,
		reportilcontract.LongFormRepresentationChecklist,
		reportilcontract.LongFormRepresentationDiagram,
		reportilcontract.LongFormSectionRoleConclusion,
		plan.Title,
		part.Title,
		section.Title,
		section.Role,
		strings.Join(section.Representations, ", "),
		strings.TrimSpace(config.Direction),
		materialStep,
	)
}

func longFormPartEditPrompt(config ProductConfig, planned reportilcontract.LongFormPart) string {
	return fmt.Sprintf(`Edit one assembled Part of a long-form reader report. The immutable Section artifacts and their assembled Part live only in the MCP workspace; do not ask for or return the manuscript in your terminal response. Preserve every Section and block identity, every source binding, every unique fact, and every table, code example, equation, worked example, benchmark, checklist, or source-supported flow realized from the plan. Never flatten structured explanatory material into generic prose. Improve the Part opening, Section handoffs, chronology, duplicate explanations, and closing judgment with bounded exact replacements only. Remove repeated setup, dates, dimensions, definitions, and viewing advice once the earlier Section has established them; preserve later passages only when they make a distinct inference or advance the reader's understanding. Do not replace a transition with a closing paragraph that restates the preceding paragraph's thesis, and do not add a summary merely because the Section is ending. When the existing final paragraph already reaches the Section's judgment, end there or use the shortest forward handoff that introduces no recap.

PLANNED PART PURPOSE: %s
DIRECTION: %s

WORKFLOW:
1. Read the bound Part plan through EOF.
2. If editorial memory is available, read it through EOF.
3. Start the long-form document workspace exactly once. The server assembles the bound immutable Section artifacts.
4. Read the complete assembled Part from offset 0 through EOF.
5. Apply only exact once-only replacements that improve continuity without changing structure or source bindings. Every edit invalidates the prior full read.
6. After the final edit, reread from offset 0 through EOF.
7. Finalize exactly once. Your terminal response may only briefly confirm finalization.`, planned.Purpose, strings.TrimSpace(config.Direction))
}

func longFormFinalEditPrompt(config ProductConfig) string {
	return fmt.Sprintf(`Perform the final whole-manuscript writing pass on a complete long-form report in target language %q. The immutable edited Part artifacts and assembled authoritative manuscript live only in the MCP workspace; do not ask for or return the manuscript in your terminal response. Preserve every Part, Section, block identity, source binding, unique fact, date, relation, caveat, technical identifier, and planned table, code example, equation, worked example, benchmark, checklist, or source-supported flow. Never flatten structured explanatory material into generic prose. Improve the whole-report opening, cross-Part logic, terminology, chronology, Part transitions, and final judgment. Remove repeated setup, dates, dimensions, definitions, and viewing advice after their first sufficient explanation while preserving every distinct later inference. Within each block, compare adjacent paragraphs and remove or recast a closing paragraph that merely restates the thesis, subjects, and relationships already established in the preceding paragraph. A shorter summary is still repetition when it adds no new inference or handoff. The final Section is the planned standalone conclusion: make its ending a sharper historical or technical judgment, never a source recap, site guide, checklist, or generic summary. Prefer bounded exact replacements over needless rewriting.

DIRECTION: %s

WORKFLOW:
1. Read the complete long-form plan through EOF.
2. If editorial memory is available, read it through EOF.
3. Start the long-form document workspace exactly once. The server assembles the immutable edited Part artifacts.
4. Read the whole assembled manuscript from offset 0 through EOF.
5. Apply only exact once-only reader-facing replacements. Never change structure or source bindings. Every edit invalidates the prior full read.
6. After the final edit, reread the complete manuscript from offset 0 through EOF and check its opening, cross-Part continuity, chronology, and final judgment.
7. Finalize exactly once. Your terminal response may only briefly confirm finalization.`, config.TargetLanguage, strings.TrimSpace(config.Direction))
}
