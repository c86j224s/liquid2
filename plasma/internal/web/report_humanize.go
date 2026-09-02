package web

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporthumanize"
)

// ReportHumanizeInput는 deprecated된 manual/post-canonical H5 Web
// compatibility 요청 값이다.
//
// Deprecated: current long-form reports use the pre-canonical style-edit stage.
type ReportHumanizeInput = reporthumanize.Input

type reportHumanizeInput = ReportHumanizeInput

// ReportHumanizeResult는 legacy H5 compatibility artifact와 agent 실행
// metadata를 함께 반환한다.
//
// Deprecated: current long-form reports use the pre-canonical style-edit stage.
type ReportHumanizeResult = reporthumanize.Result

type reportHumanizeResult = ReportHumanizeResult

// ReportHumanizeIDFunc는 legacy H5 Web compatibility adapter의 ID 생성
// 계약을 주입하는 함수 포트다.
//
// Deprecated: current long-form reports use the pre-canonical style-edit stage.
type ReportHumanizeIDFunc = reporthumanize.IDFunc

// ReportHumanizeService는 legacy H5 compatibility 실행에 필요한 report 조회와
// 이벤트 기록 기능을 제공한다.
//
// Deprecated: current long-form reports use the pre-canonical style-edit stage.
type ReportHumanizeService = reporthumanize.Service

type reportHumanizePendingPayload = reporthumanize.PendingPayload

func (server *Server) humanizeMarkdownReport(ctx context.Context, missionID string, input reportHumanizeInput, executor AgentExecutor) (reportHumanizeResult, error) {
	return reporthumanize.HumanizeMarkdownReport(ctx, server.service, newID, missionID, input, executor)
}

// HumanizeMarkdownReport는 legacy manual/post-canonical H5 compatibility
// artifact를 생성한다. Markdown 원본은 유지한다.
//
// Deprecated: current long-form reports use the pre-canonical style-edit stage.
func HumanizeMarkdownReport(ctx context.Context, service ReportHumanizeService, idFunc ReportHumanizeIDFunc, missionID string, input ReportHumanizeInput, executor AgentExecutor) (ReportHumanizeResult, error) {
	return reporthumanize.HumanizeMarkdownReport(ctx, service, idFunc, missionID, input, executor)
}

func reportHumanizePendingPayloadFromEvent(event ledger.Event) reportHumanizePendingPayload {
	return reporthumanize.PendingPayloadFromEvent(event)
}

func reportHumanizeInFlightPendingEventID(event ledger.Event) string {
	return reporthumanize.InFlightPendingEventID(event)
}

func (server *Server) appendReportHumanizeStaleFailed(ctx context.Context, missionID string, pending ledger.Event) (ledger.Event, error) {
	return reporthumanize.AppendStaleFailed(ctx, server.service, newID, missionID, pending)
}

func (server *Server) recoverStaleReportHumanizeFinalizedPatch(ctx context.Context, missionID string, pending ledger.Event) (bool, error) {
	return reporthumanize.RecoverFinalizedPatch(ctx, server.service, newID, missionID, pending)
}

func reportHumanizeInputFromPendingPayload(payload reportHumanizePendingPayload, sourceArtifact artifact.Raw) reportHumanizeInput {
	return reporthumanize.InputFromPendingPayload(payload, sourceArtifact)
}

func appendReportHumanizeFailed(ctx context.Context, service ReportHumanizeService, idFunc ReportHumanizeIDFunc, missionID string, input reportHumanizeInput, toolSessionID string, humanizePendingEventID string, durationMS int64, cause error) (ledger.Event, error) {
	return reporthumanize.AppendFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS, cause)
}
