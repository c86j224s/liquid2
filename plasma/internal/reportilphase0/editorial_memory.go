package reportilphase0

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

func runEditorialMemoryStage(
	ctx context.Context,
	config ProductConfig,
	catalog reportilcontract.SourceCatalog,
) (reportilcontract.EditorialMemory, reportilcontract.EditorialMemoryReceipt, []agentexec.AgentResult, error) {
	request := sourceIsolatedRequest(
		config, catalog, "il_editorial_memory", 1,
		editorialMemoryPrompt(config, catalog), nil, 0,
	)
	request.ExtraMCPTools = []string{
		reportilcontract.SourceListTool,
		reportilcontract.SourceReadTool,
		reportilcontract.SourceQuoteRegisterTool,
		reportilcontract.EditorialMemoryStartTool,
		reportilcontract.EditorialMemoryAppendTool,
		reportilcontract.EditorialMemoryReadTool,
		reportilcontract.EditorialMemoryFinalizeTool,
	}
	result, err := config.Provider.Run(ctx, request)
	results := []agentexec.AgentResult{result}
	if err != nil {
		return reportilcontract.EditorialMemory{}, reportilcontract.EditorialMemoryReceipt{}, results,
			&providerStageError{reason: reportexecution.ProviderFailureReasonTransport, cause: err}
	}
	sourceReceipt, err := config.VerifySourceRead.VerifyReportILSourceRead(
		ctx, config.MissionID, request.ToolSessionID, "il_editorial_memory", catalog,
	)
	if err != nil {
		return reportilcontract.EditorialMemory{}, reportilcontract.EditorialMemoryReceipt{}, results,
			&providerStageError{
				reason: reportexecution.ProviderFailureReasonSemanticValidation,
				cause:  withValidationCode(reportexecution.ProviderValidationCodeSourceReadContract, err),
			}
	}
	if err := validateCompleteSourceRead(catalog, sourceReceipt); err != nil {
		return reportilcontract.EditorialMemory{}, reportilcontract.EditorialMemoryReceipt{}, results,
			&providerStageError{
				reason: reportexecution.ProviderFailureReasonSemanticValidation,
				cause:  withValidationCode(reportexecution.ProviderValidationCodeSourceReadContract, err),
			}
	}
	if len(sourceReceipt.SourceQuotes) == 0 {
		return reportilcontract.EditorialMemory{}, reportilcontract.EditorialMemoryReceipt{}, results,
			&providerStageError{
				reason: reportexecution.ProviderFailureReasonSemanticValidation,
				cause: withValidationCode(
					reportexecution.ProviderValidationCodeSourceReadContract,
					fmt.Errorf("editorial memory did not register exact frozen-source anchors"),
				),
			}
	}
	memory, receipt, err := config.AuthorDocuments.ReadReportILEditorialMemory(
		ctx, config.MissionID, request.ToolSessionID, catalog,
	)
	if err != nil {
		return reportilcontract.EditorialMemory{}, reportilcontract.EditorialMemoryReceipt{}, results,
			&providerStageError{
				reason: reportexecution.ProviderFailureReasonSemanticValidation,
				cause:  withValidationCode(reportexecution.ProviderValidationCodeDocumentContract, err),
			}
	}
	return memory, receipt, results, nil
}

func editorialMemoryPrompt(config ProductConfig, catalog reportilcontract.SourceCatalog) string {
	objective := strings.TrimSpace(config.MissionObjective)
	if objective == "" {
		objective = strings.TrimSpace(config.Title)
	}
	return fmt.Sprintf(`Build the server-owned editorial memory that the report author will read instead of rereading raw sources. This is not a report outline, evidence ledger, source summary, or final prose. Its only job is to preserve the material factual accounts and relationships that a natural report must be able to write without source compression loss.

MISSION OBJECTIVE:
%s

ADDITIONAL DIRECTION:
%s

AUTHORING DEPTH:
%s

WORKFLOW:
1. Call plasma.report_il.sources.list with {}.
2. Call plasma.report_il.sources.read with {} until remaining_sources is zero. Read every selected source completely.
3. Decide which connected accounts materially improve the reader's answer. A connected account keeps together the actor, role, action, object or relationship, date, duration, concrete case, and uncertainty that a source presents together.
4. Also inventory source-native explanatory material that a later author would otherwise be unable to reconstruct from a high-level paraphrase: comparison rows and dimensions; equations and variable meanings; worked calculations with inputs, assumptions, and results; executable code, API calls, commands, and configuration; benchmark conditions and measurements; operational procedures and decision checklists; and diagram or flow relationships. Preserve literal syntax, units, thresholds, formulas, example values, and enough surrounding explanation to use them correctly. When a source provides a multi-step API lifecycle, stateful/stateless transport alternative, routing constraint, or command-line budget control, preserve the complete distinct operation rather than collapsing it into a general recommendation. Do not invent these materials, force them into every subject, or turn decorative source formatting into an obligation.
5. For each account, call plasma.report_il.sources.quote once per supporting source with the shortest exact contiguous excerpt that still preserves the account's decisive wording and relationship. For source-native explanatory material, the excerpt must also preserve the complete usable payload: for example an entire short equation, code or command example, comparison row, worked calculation, or procedure step sequence when it fits within the quote limit. Include causative, passive, commissioned, attributed, or uncertain wording rather than trimming it to names and dates. Each excerpt must be nonempty, exact, and at most 512 UTF-8 bytes. Keep only the successful opaque source_receipt.
6. Call plasma.report_il.memory.start once with the report language.
7. Call plasma.report_il.memory.append once per material connected account. Write each account as compact natural factual prose, not labels or field-value fragments. When an account preserves source-native explanatory material, retain its exact formula, syntax, dimensions, ordered steps, values, or comparison structure in the account prose instead of replacing it with a general takeaway; one account may contain a compact code, equation, row set, or recipe when that payload belongs together. Mark it essential when losing it would materially damage the answer; otherwise mark it supporting. Include every directly supporting source key and every successful source_receipt for that account. Each declared source must have at least one exact anchor. The server reconstructs the verified excerpts inside memory; never copy quote text into receipt fields.
8. Preserve conflicting or distinct traditions as separate accounts. Never borrow one account's actor, role, or relationship for another account's date or duration, and never silently reconcile disagreement.
9. Normalize source names into the natural report language only when the complete identity is established by the selected sources. If a source itself contains a visibly incomplete, mixed-script, placeholder-like, or malformed personal name and no selected source establishes the complete identity, omit that name from account prose and preserve only the supported role, action, and relationship. Never guess the missing identity.
10. Do not create one account per sentence or source. Combine details only when the source presents them as one connected account; split genuinely independent facts.
11. After assembly, call plasma.report_il.memory.read from offset 0 and continue at exact next_offset until truncated is false. Reread the complete memory, including its exact source anchors, and correct any account whose paraphrase changes who did what to whom, who ordered or caused an action, how certain the source is, presents an incomplete source name as a finished reader-facing identity, or loses the usable code, equation, comparison, calculation, benchmark, or procedure that made the account explanatory.
12. Call plasma.report_il.memory.finalize exactly once. Your terminal response may only briefly confirm finalization.

The author will turn this memory into natural reader-facing prose. Do not prescribe headings, section order, opening language, transitions, or conclusion language. Do not mention tools, source keys, schemas, validators, or stages inside account prose.

Frozen source catalog SHA-256: %s`, objective, readerDirection(config), editorialMemoryDepthGuidance(config.AuthoringMode), catalog.SHA256)
}
