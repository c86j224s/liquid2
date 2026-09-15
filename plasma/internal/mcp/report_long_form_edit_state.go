package mcp

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/reportfinaledit"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"strings"
)

func (server *Server) finalEditHandler() reportfinaledit.Handler {
	return reportfinaledit.Handler{State: server.reportFinalEditState, Mu: &server.mu, Service: server.service, StageBinding: func() reporting.FinalEditStageBinding { return server.finalEditStageBinding }, FinalizeBinding: func() reporting.LongFormFinalizeBinding { return server.longFormFinalizeBinding }, RequireStageBinding: server.requireFinalEditStageBinding, RequireLongFormBinding: server.requireLongFormEditBinding, FinalizeAvailable: func(b reporting.LongFormFinalizeBinding) bool {
		return ValidateLongFormFinalizeBinding(server.binding, b) == nil && server.toolEnabled(ToolReportLongFormFinalize)
	}, MissionID: func() string { return server.binding.MissionID }, Decode: decodeReportPlanJSON, NormalizeInput: normalizeMutatingInput, ErrorResult: errorResult, ErrorFromErr: errorFromErr, ValidateID: validateID, NewID: newMCPID}
}
func (server *Server) requireLongFormEditBinding(common commonMutatingInput) (reporting.LongFormFinalizeBinding, error) {
	if err := server.requireBoundWriteSession(common); err != nil {
		return reporting.LongFormFinalizeBinding{}, err
	}
	binding := server.longFormFinalizeBinding
	if err := ValidateLongFormFinalizeBinding(server.binding, binding); err != nil {
		return reporting.LongFormFinalizeBinding{}, err
	}
	if binding.CompositionStrategy != reporting.LongFormCompositionNarrativeEdit {
		return reporting.LongFormFinalizeBinding{}, fmt.Errorf("%w: long-form final editor is not enabled for this composition strategy", producterror.ErrInvalidInput)
	}
	return binding, nil
}

func (server *Server) longFormEditToolEnabled(name string) bool {
	if server.finalEditConfigErr != nil || server.finalEditStageBindingSet {
		return false
	}
	binding := server.longFormFinalizeBinding
	return server.toolEnabled(name) && binding.CompositionStrategy == reporting.LongFormCompositionNarrativeEdit && ValidateLongFormFinalizeBinding(server.binding, binding) == nil
}

func longFormEditDisabledResult(call ToolCall) ToolResult {
	return errorResult(call.Name, missionIDFromArguments(call.Arguments), "binding", "long-form final editor tools are only enabled for a bound narrative-edit legacy session or corrective gate stage; invalid stage/final binding configurations are closed", false, nil)
}

func (server *Server) finalEditStageMode() string {
	if server.finalEditConfigErr != nil || !server.finalEditStageBindingSet {
		return ""
	}
	return strings.TrimSpace(server.finalEditStageBinding.Stage)
}

func (server *Server) finalEditStageToolEnabled(name string) bool {
	if server.finalEditConfigErr != nil || !server.toolEnabled(name) {
		return false
	}
	stage := server.finalEditStageMode()
	switch name {
	case ToolReportLongFormFinalWriteStart, ToolReportLongFormFinalWriteRead, ToolReportLongFormFinalWritePatch, ToolReportLongFormFinalWriteSubmit:
		return stage == reporting.FinalEditStageWriter
	case ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditRead, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit:
		return stage == reporting.FinalEditStageReader
	case ToolReportLongFormStyleEditStart, ToolReportLongFormStyleEditRead, ToolReportLongFormStyleEditPatch, ToolReportLongFormStyleEditSubmit:
		return stage == reporting.FinalEditStageStyle
	case ToolReportLongFormEditStart, ToolReportLongFormEditRead, ToolReportLongFormEditPatch, ToolReportLongFormEditSubmit:
		return stage == reporting.FinalEditStageGate
	case ToolReportLongFormStyleReviewRead:
		return stage == reporting.FinalEditStageGate && server.finalEditStageBinding.PostReportHumanize == reporting.FinalEditHumanizeEnabled
	case ToolReportLongFormStyleSemanticValidationRead, ToolReportLongFormStyleSemanticValidationSubmit:
		return stage == reporting.FinalEditStageStyleSemanticValidation
	case ToolReportLongFormEvidenceGateRead, ToolReportLongFormEvidenceGateSubmit:
		return stage == reporting.FinalEditStageEvidenceGate
	default:
		return false
	}
}

func (server *Server) requireFinalEditStageBinding(common commonMutatingInput, expectedStage string) (reporting.FinalEditStageBinding, error) {
	if server.finalEditConfigErr != nil {
		return reporting.FinalEditStageBinding{}, fmt.Errorf("%w: final edit MCP binding configuration is closed: %v", producterror.ErrInvalidInput, server.finalEditConfigErr)
	}
	if err := server.requireBoundWriteSession(common); err != nil {
		return reporting.FinalEditStageBinding{}, err
	}
	binding := server.finalEditStageBinding
	if err := ValidateFinalEditStageBinding(server.binding, binding); err != nil {
		return reporting.FinalEditStageBinding{}, err
	}
	if strings.TrimSpace(binding.Stage) != strings.TrimSpace(expectedStage) {
		return reporting.FinalEditStageBinding{}, fmt.Errorf("%w: final edit stage tool does not match the runner binding", producterror.ErrInvalidInput)
	}
	if binding.Stage == reporting.FinalEditStageGate || binding.Stage == reporting.FinalEditStageEvidenceGate {
		if err := ValidateLongFormFinalizeBinding(server.binding, server.longFormFinalizeBinding); err != nil {
			return reporting.FinalEditStageBinding{}, err
		}
	}
	return binding, nil
}

func finalEditStageDisabledResult(call ToolCall) ToolResult {
	return errorResult(call.Name, missionIDFromArguments(call.Arguments), "binding", "final edit stage tools are only enabled for a matching bound stage session; invalid stage/final binding configurations are closed", false, nil)
}

func (server *Server) callReportLongFormFinalize(ctx context.Context, call ToolCall) ToolResult {
	h := server.finalEditHandler()
	return h.CallReportLongFormFinalize(ctx, call)
}
func (server *Server) callReportLongFormEditSubmit(ctx context.Context, call ToolCall) ToolResult {
	h := server.finalEditHandler()
	return h.CallReportLongFormEditSubmit(ctx, call)
}
func (server *Server) callReportLongFormEditStart(ctx context.Context, call ToolCall) ToolResult {
	h := server.finalEditHandler()
	return h.CallReportLongFormEditStart(ctx, call)
}
func (server *Server) callReportLongFormEditRead(ctx context.Context, call ToolCall) ToolResult {
	h := server.finalEditHandler()
	return h.CallReportLongFormEditRead(ctx, call)
}
func (server *Server) callReportLongFormStyleSemanticValidationRead(ctx context.Context, call ToolCall) ToolResult {
	h := server.finalEditHandler()
	return h.CallReportLongFormStyleSemanticValidationRead(ctx, call)
}
func (server *Server) callReportLongFormStyleSemanticValidationSubmit(ctx context.Context, call ToolCall) ToolResult {
	h := server.finalEditHandler()
	return h.CallReportLongFormStyleSemanticValidationSubmit(ctx, call)
}
func (server *Server) callReportLongFormEvidenceGateRead(ctx context.Context, call ToolCall) ToolResult {
	h := server.finalEditHandler()
	return h.CallReportLongFormEvidenceGateRead(ctx, call)
}
func (server *Server) callReportLongFormEvidenceGateSubmit(ctx context.Context, call ToolCall) ToolResult {
	h := server.finalEditHandler()
	return h.CallReportLongFormEvidenceGateSubmit(ctx, call)
}
func (server *Server) callReportLongFormStageEditStart(ctx context.Context, call ToolCall, expectedStage string) ToolResult {
	h := server.finalEditHandler()
	return h.CallReportLongFormStageEditStart(ctx, call, expectedStage)
}
func (server *Server) callReportLongFormStageEditRead(ctx context.Context, call ToolCall, expectedStage string) ToolResult {
	h := server.finalEditHandler()
	return h.CallReportLongFormStageEditRead(ctx, call, expectedStage)
}
func (server *Server) callReportLongFormStageEditPatch(ctx context.Context, call ToolCall, expectedStage string) ToolResult {
	h := server.finalEditHandler()
	return h.CallReportLongFormStageEditPatch(ctx, call, expectedStage)
}
func (server *Server) callReportLongFormStageEditSubmit(ctx context.Context, call ToolCall, expectedStage string) ToolResult {
	h := server.finalEditHandler()
	return h.CallReportLongFormStageEditSubmit(ctx, call, expectedStage)
}
func (server *Server) callReportLongFormStyleReviewRead(ctx context.Context, call ToolCall) ToolResult {
	h := server.finalEditHandler()
	return h.CallReportLongFormStyleReviewRead(ctx, call)
}
func (server *Server) callReportLongFormEditPatch(ctx context.Context, call ToolCall) ToolResult {
	h := server.finalEditHandler()
	return h.CallReportLongFormEditPatch(ctx, call)
}

type longFormEditDraft = reportfinaledit.LongFormEditDraft
type reportLongFormEditStartInput = reportfinaledit.ReportLongFormEditStartInput
type reportLongFormEditReadInput = reportfinaledit.ReportLongFormEditReadInput
type reportLongFormEditPatchInput = reportfinaledit.ReportLongFormEditPatchInput
type reportLongFormEditSubmitInput = reportfinaledit.ReportLongFormEditSubmitInput
type reportLongFormFinalizeInput = reportfinaledit.ReportLongFormFinalizeInput
type readOnlyValidationDraft = reportfinaledit.ReadOnlyValidationDraft
type reportLongFormStyleSemanticValidationSubmitInput = reportfinaledit.ReportLongFormStyleSemanticValidationSubmitInput
type reportLongFormStyleSemanticValidationVerdict = reportfinaledit.ReportLongFormStyleSemanticValidationVerdict
type reportLongFormEvidenceGateSubmitInput = reportfinaledit.ReportLongFormEvidenceGateSubmitInput
type reportLongFormEvidenceGateFindingInput = reportfinaledit.ReportLongFormEvidenceGateFindingInput
type longFormStageEditDraft = reportfinaledit.LongFormStageEditDraft
type reportLongFormStageEditSubmitInput = reportfinaledit.ReportLongFormStageEditSubmitInput
type reportLongFormGateFindingInput = reportfinaledit.ReportLongFormGateFindingInput
type reportLongFormSemanticAcceptanceInput = reportfinaledit.ReportLongFormSemanticAcceptanceInput
type markdownBlockByteRange = reportfinaledit.MarkdownBlockByteRange
