package web

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporthumanize"
)

// ReportHumanizeInput는 웹 및 에이전트 어댑터에 전달되는 요청 값이다.
type ReportHumanizeInput = reporthumanize.Input

type reportHumanizeInput = ReportHumanizeInput

// ReportHumanizeResult는 H5 보정 artifact와 agent 실행 metadata를 함께 반환한다.
type ReportHumanizeResult = reporthumanize.Result

type reportHumanizeResult = ReportHumanizeResult

// ReportHumanizeIDFunc는 웹 및 에이전트 어댑터에서 테스트와 실행이 같은 ID 생성 계약을 주입할 수 있게 하는 함수 포트다.
type ReportHumanizeIDFunc = reporthumanize.IDFunc

// ReportHumanizeService는 H5 보정 실행에 필요한 report 조회와 이벤트 기록 기능을 제공한다.
type ReportHumanizeService = reporthumanize.Service

type reportHumanizePendingPayload = reporthumanize.PendingPayload

func (server *Server) humanizeMarkdownReport(ctx context.Context, missionID string, input reportHumanizeInput, executor AgentExecutor) (reportHumanizeResult, error) {
	return reporthumanize.HumanizeMarkdownReport(ctx, server.service, newID, missionID, input, executor)
}

// HumanizeMarkdownReport는 Markdown 원본을 유지한 채 MCP patch 방식으로 말투 보정 artifact를 시도한다.
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
