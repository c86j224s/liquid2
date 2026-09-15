package reportfinaledit

import (
	"context"
	"fmt"
	patchhandler "github.com/c86j224s/liquid2/plasma/internal/mcp/reportpatch"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func validateLongFormStageEditAccess(draft *LongFormStageEditDraft, missionID string, sessionID string, stage string) error {
	if draft == nil || draft.MissionID != strings.TrimSpace(missionID) || draft.SessionID != strings.TrimSpace(sessionID) || draft.Stage != strings.TrimSpace(stage) {
		return fmt.Errorf("%w: long-form stage edit draft is outside this MCP session", producterror.ErrInvalidInput)
	}
	return nil
}

func longFormStageEditFromState(draft LongFormStageEditDraft) map[string]any {
	state := "open"
	if draft.Submitted || draft.StageSubmitted {
		state = "submitted"
	} else if draft.Finalizing {
		state = "finalizing"
	}
	operationCount := len(draft.Operations)
	if draft.StageSubmitted {
		operationCount = draft.StageOperationCount
	}
	return map[string]any{
		"draft_id": draft.DraftID, "stage": draft.Stage, "mission_id": draft.MissionID, "session_id": draft.SessionID,
		"pending_event_id": draft.PendingID, "plan_event_id": draft.PlanEventID,
		"state": state, "content_length": len([]byte(draft.Content)), "operation_count": operationCount,
		"submitted": draft.Submitted || draft.StageSubmitted, "artifact_id": draft.ArtifactID, "event_id": draft.EventID,
	}
}

func (server *Handler) patchLongFormStageEditDraft(call wire.ToolCall, common wire.CommonMutatingInput, input ReportLongFormEditPatchInput, expectedStage string) wire.ToolResult {
	draftID := strings.TrimSpace(input.DraftID)
	if err := server.ValidateID("rfe_", draftID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if !utf8.ValidString(input.Replacement) || len([]byte(input.Replacement)) > patchhandler.ReportPatchMaxApplyBytes {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "long-form stage edit replacement is not bounded UTF-8 text", false, []string{draftID})
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	draft, ok := server.State.StageDrafts[draftID]
	if !ok {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "long-form stage edit draft was not found in this MCP process", false, []string{draftID})
	}
	if err := validateLongFormStageEditAccess(draft, common.MissionID, common.SessionID, expectedStage); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Submitted || draft.StageSubmitted || draft.Finalizing {
		return server.ErrorResult(call.Name, common.MissionID, "conflict", "long-form stage edit draft is no longer editable", false, []string{draftID})
	}
	if len(draft.Operations) >= reportLongFormEditMaxOperations {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "long-form stage edit draft has too many operations", false, []string{draftID})
	}
	category := ""
	reason := ""
	if expectedStage == reporting.FinalEditStageStyle {
		parsedCategory, parsedReason, err := validateStyleStagePatchContract(draft.Content, input)
		if err != nil {
			return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
		}
		category = parsedCategory
		reason = parsedReason
	}
	occurrence := input.Occurrence
	if occurrence <= 0 {
		occurrence = 1
	}
	next, err := patchhandler.ApplyOperation(draft.Content, patchhandler.ReportPatchApplyInput{
		Operation: input.Operation, MatchText: input.MatchText, Replacement: input.Replacement,
		Occurrence: input.Occurrence, ReplaceAll: input.ReplaceAll,
	})
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if strings.TrimSpace(next) == "" || len([]byte(next)) > patchhandler.ReportPatchMaxBytes || !utf8.ValidString(next) {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "long-form stage edit would produce an invalid manuscript", false, []string{draftID})
	}
	draft.Content = next
	draft.Operations = append(draft.Operations, patchhandler.Operation{
		Operation: strings.TrimSpace(input.Operation), Summary: strings.TrimSpace(input.Summary), Bytes: len([]byte(input.Replacement)),
		Category: category, Reason: reason, MatchText: input.MatchText, Replacement: input.Replacement, Occurrence: occurrence,
	})
	draft.UpdatedAt = nowUTC()
	return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, Content: longFormStageEditFromState(*draft)}
}

func validateStyleStagePatchContract(markdown string, input ReportLongFormEditPatchInput) (string, string, error) {
	if strings.TrimSpace(input.Operation) != "replace" || input.ReplaceAll {
		return "", "", fmt.Errorf("%w: style edit patch must use one replace operation without replace_all", producterror.ErrInvalidInput)
	}
	match := strings.TrimSpace(input.MatchText)
	if match == "" {
		return "", "", fmt.Errorf("%w: style edit replacement must stay inside one non-empty Markdown block", producterror.ErrInvalidInput)
	}
	if crossesMarkdownBlankLine(input.MatchText) || crossesMarkdownBlankLine(input.Replacement) {
		return "", "", fmt.Errorf("%w: style edit replacement must stay inside one non-empty Markdown block", producterror.ErrInvalidInput)
	}
	if !stylePatchLeavesOneMarkdownBlockNonEmpty(markdown, input.MatchText, input.Replacement, input.Occurrence) {
		return "", "", fmt.Errorf("%w: style edit match must stay inside one non-empty Markdown block", producterror.ErrInvalidInput)
	}
	return ValidateStyleStagePatchSummary(input.Summary)
}

func ValidateStyleStagePatchSummary(summary string) (string, string, error) {
	summary = strings.TrimSpace(summary)
	const marker = "category:"
	if !strings.HasPrefix(summary, marker) {
		return "", "", fmt.Errorf("%w: style edit summary must use category: <one-known-token>; <concrete issue>", producterror.ErrInvalidInput)
	}
	if strings.Count(summary, marker) != 1 {
		return "", "", fmt.Errorf("%w: style edit summary must contain exactly one category marker", producterror.ErrInvalidInput)
	}
	rest := strings.TrimSpace(strings.TrimPrefix(summary, marker))
	category, issue, ok := strings.Cut(rest, ";")
	if !ok {
		return "", "", fmt.Errorf("%w: style edit summary must separate category and issue with semicolon", producterror.ErrInvalidInput)
	}
	category = strings.TrimSpace(category)
	issue = strings.TrimSpace(issue)
	if issue == "" {
		return "", "", fmt.Errorf("%w: style edit summary must name one known experiment-61 category and one concrete issue", producterror.ErrInvalidInput)
	}
	if err := reporting.ValidateFinalEditStyleDiagnosisCategory(category); err != nil {
		return "", "", fmt.Errorf("%w: style edit summary must name one known experiment-61 category and one concrete issue", producterror.ErrInvalidInput)
	}
	for _, token := range strings.FieldsFunc(issue, func(r rune) bool {
		return (r < 'a' || r > 'z') && r != '_'
	}) {
		if reporting.ValidateFinalEditStyleDiagnosisCategory(token) == nil {
			return "", "", fmt.Errorf("%w: style edit summary issue must not contain extra category tokens", producterror.ErrInvalidInput)
		}
	}
	return category, issue, nil
}

func stylePatchLeavesOneMarkdownBlockNonEmpty(markdown string, matchText string, replacement string, occurrence int) bool {
	if matchText == "" {
		return false
	}
	if occurrence <= 0 {
		occurrence = 1
	}
	index := strings.Index(markdown, matchText)
	for current := 1; current < occurrence && index >= 0; current++ {
		nextStart := index + len(matchText)
		next := strings.Index(markdown[nextStart:], matchText)
		if next < 0 {
			return false
		}
		index = nextStart + next
	}
	if index < 0 {
		return false
	}
	matchStart, matchEnd := index, index+len(matchText)
	for _, blockRange := range markdownNonEmptyBlockByteRanges(markdown) {
		blockStart := blockRange.start
		blockEnd := blockRange.end
		if matchStart >= blockStart && matchEnd <= blockEnd {
			relativeStart := matchStart - blockStart
			relativeEnd := matchEnd - blockStart
			blockText := markdown[blockStart:blockEnd]
			nextBlock := blockText[:relativeStart] + replacement + blockText[relativeEnd:]
			return strings.TrimSpace(nextBlock) != ""
		}
	}
	return false
}

type MarkdownBlockByteRange struct {
	start int
	end   int
}

func markdownNonEmptyBlockByteRanges(markdown string) []MarkdownBlockByteRange {
	ranges := []MarkdownBlockByteRange{}
	blockStart := 0
	for i := 0; i < len(markdown); i++ {
		if markdown[i] != '\n' {
			continue
		}
		j := i + 1
		for j < len(markdown) && (markdown[j] == ' ' || markdown[j] == '\t' || markdown[j] == '\r') {
			j++
		}
		if j >= len(markdown) || markdown[j] != '\n' {
			continue
		}
		blockEnd := i
		if blockEnd > blockStart && markdown[blockEnd-1] == '\r' {
			blockEnd--
		}
		if strings.TrimSpace(markdown[blockStart:blockEnd]) != "" {
			ranges = append(ranges, MarkdownBlockByteRange{start: blockStart, end: blockEnd})
		}
		blockStart = j + 1
		i = j
	}
	blockEnd := len(markdown)
	if blockEnd > blockStart && markdown[blockEnd-1] == '\r' {
		blockEnd--
	}
	if strings.TrimSpace(markdown[blockStart:blockEnd]) != "" {
		ranges = append(ranges, MarkdownBlockByteRange{start: blockStart, end: blockEnd})
	}
	return ranges
}

func crossesMarkdownBlankLine(text string) bool {
	for i := 0; i < len(text); i++ {
		if text[i] != '\n' {
			continue
		}
		j := i + 1
		for j < len(text) && (text[j] == ' ' || text[j] == '\t' || text[j] == '\r') {
			j++
		}
		if j < len(text) && text[j] == '\n' {
			return true
		}
	}
	return false
}

func gateFindingsFromInput(input *[]ReportLongFormGateFindingInput) ([]reporting.FinalEditGateFinding, error) {
	if input == nil {
		return nil, fmt.Errorf("%w: final edit gate findings are required", producterror.ErrInvalidInput)
	}
	findings := make([]reporting.FinalEditGateFinding, 0, len(*input))
	for _, item := range *input {
		statement := strings.TrimSpace(item.Statement)
		classification := strings.TrimSpace(item.Classification)
		action := strings.TrimSpace(item.RepairAction)
		if statement == "" || classification == "" {
			return nil, fmt.Errorf("%w: final edit gate finding is incomplete", producterror.ErrInvalidInput)
		}
		evidenceIDs := make([]string, 0, len(item.EvidenceIDs))
		for _, evidenceID := range item.EvidenceIDs {
			if trimmed := strings.TrimSpace(evidenceID); trimmed != "" {
				evidenceIDs = append(evidenceIDs, trimmed)
			}
		}
		findings = append(findings, reporting.FinalEditGateFinding{
			Statement: statement, Classification: classification, RepairAction: action, EvidenceIDs: evidenceIDs,
		})
	}
	return findings, nil
}

func evidenceGateFindingsFromInput(input *[]ReportLongFormGateFindingInput) ([]reporting.FinalEditGateFinding, error) {
	if input == nil {
		return nil, fmt.Errorf("%w: evidence gate findings are required", producterror.ErrInvalidInput)
	}
	findings := make([]reporting.FinalEditGateFinding, 0, len(*input))
	for _, item := range *input {
		statementSHA := strings.TrimSpace(item.StatementSHA256)
		classification := strings.TrimSpace(item.Classification)
		if statementSHA == "" || classification == "" ||
			strings.TrimSpace(item.Statement) != "" ||
			strings.TrimSpace(item.RepairAction) != "" {
			return nil, fmt.Errorf("%w: evidence gate finding is incomplete", producterror.ErrInvalidInput)
		}
		evidenceIDs := make([]string, 0, len(item.EvidenceIDs))
		for _, evidenceID := range item.EvidenceIDs {
			if trimmed := strings.TrimSpace(evidenceID); trimmed != "" {
				evidenceIDs = append(evidenceIDs, trimmed)
			}
		}
		findings = append(findings, reporting.FinalEditGateFinding{
			StatementSHA256: statementSHA, Classification: classification, EvidenceIDs: evidenceIDs,
		})
	}
	return findings, nil
}

func semanticAcceptanceFromInput(input *[]ReportLongFormSemanticAcceptanceInput) ([]reporting.FinalEditSemanticAcceptance, error) {
	if input == nil {
		return nil, nil
	}
	out := make([]reporting.FinalEditSemanticAcceptance, 0, len(*input))
	for _, item := range *input {
		out = append(out, reporting.FinalEditSemanticAcceptance{
			ParagraphOrdinal:      item.ParagraphOrdinal,
			FinalParagraphOrdinal: item.FinalParagraphOrdinal,
			Verdict:               strings.TrimSpace(item.Verdict),
		})
	}
	return out, nil
}

func (server *Handler) CallReportLongFormStageEditSubmit(ctx context.Context, call wire.ToolCall, expectedStage string) wire.ToolResult {
	var input ReportLongFormStageEditSubmitInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "long-form stage edit submit arguments are invalid", false, nil)
	}
	common, _, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.RequireStageBinding(common, expectedStage)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	if input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID {
		return server.ErrorResult(call.Name, common.MissionID, "binding", "long-form stage edit submit does not match the runner binding", false, nil)
	}
	if binding.Stage == reporting.FinalEditStageGate {
		return server.submitLongFormGateEdit(ctx, call, common, input, binding)
	}
	return server.submitLongFormDurableStageEdit(ctx, call, common, input, binding)
}
