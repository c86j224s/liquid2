package web

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/conversation"
	plasmamcp "github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/sources/localpath"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestMissionDetailActiveWorkIsMissionScoped(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	server := httptest.NewServer(NewServer(service, Options{}))
	defer server.Close()

	activeMission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Active"})
	activeMissionID := nestedString(t, activeMission, "projection", "mission_id")
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_active_report", MissionID: activeMissionID, EventType: "report.draft.pending",
		Producer: app.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{"title": "Active report"}),
	}); err != nil {
		t.Fatal(err)
	}
	idleMission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Idle"})
	idleMissionID := nestedString(t, idleMission, "projection", "mission_id")

	activeDetail := getJSON(t, server.URL+"/api/missions/"+activeMissionID)
	activeWork := nestedMap(t, activeDetail, "active_work")
	blocks, ok := activeWork["blocks"].([]any)
	if !ok || len(blocks) != 1 {
		t.Fatalf("expected one active mission block, got %#v", activeWork)
	}
	block, ok := blocks[0].(map[string]any)
	if !ok || block["reason_code"] != app.BlockingReasonReport {
		t.Fatalf("expected active mission report block, got %#v", activeDetail["active_work"])
	}
	idleDetail := getJSON(t, server.URL+"/api/missions/"+idleMissionID)
	idleBlocks := nestedMap(t, idleDetail, "active_work")["blocks"]
	if idleBlocks != nil {
		blocks, ok := idleBlocks.([]any)
		if !ok || len(blocks) != 0 {
			t.Fatalf("idle mission must not inherit another mission active work: %#v", idleDetail["active_work"])
		}
	}
}

func TestMissionListIncludesMissionActivitySummary(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	server := httptest.NewServer(NewServer(service, Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Activity"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_list_pending", MissionID: missionID, EventType: "turn.agent.pending",
		Producer: app.Producer{Type: "agent", ID: "test"}, Payload: mustJSON(map[string]any{"user_event_id": "evt_user"}),
	}); err != nil {
		t.Fatal(err)
	}

	listed := getJSON(t, server.URL+"/api/missions")
	missions, ok := listed["missions"].([]any)
	if !ok || len(missions) != 1 {
		t.Fatalf("mission list = %#v", listed)
	}
	item, ok := missions[0].(map[string]any)
	if !ok {
		t.Fatalf("mission item = %#v", missions[0])
	}
	activity := nestedMap(t, item, "activity")
	if activity["last_sequence"] != float64(2) {
		t.Fatalf("activity last sequence = %#v", activity)
	}
	activeWork := nestedMap(t, activity, "active_work")
	items, ok := activeWork["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("activity active work = %#v", activeWork)
	}
}

func TestMissionListActivitySummaryExposesAgentFailure(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	server := httptest.NewServer(NewServer(service, Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Failure activity"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_list_error_pending", MissionID: missionID, EventType: "turn.agent.pending",
		Producer: app.Producer{Type: "agent", ID: "codex"}, Payload: mustJSON(map[string]any{"user_event_id": "evt_list_error_user", "agent_executor": "codex"}),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_list_error", MissionID: missionID, EventType: "turn.agent.response",
		Producer: app.Producer{Type: "agent", ID: "codex"}, Payload: mustJSON(map[string]any{"kind": "agent_error", "user_event_id": "evt_list_error_user", "agent_executor": "codex"}),
	}); err != nil {
		t.Fatal(err)
	}

	listed := getJSON(t, server.URL+"/api/missions")
	missions, ok := listed["missions"].([]any)
	if !ok || len(missions) != 1 {
		t.Fatalf("mission list = %#v", listed)
	}
	item, ok := missions[0].(map[string]any)
	if !ok {
		t.Fatalf("mission item = %#v", missions[0])
	}
	activity := nestedMap(t, item, "activity")
	latest := nestedMap(t, activity, "latest_terminal_activity")
	if activity["last_sequence"] != float64(3) || latest["kind"] != string(app.TerminalActivityTurn) || latest["outcome"] != string(app.TerminalActivityFailed) {
		t.Fatalf("activity = %#v", activity)
	}
	activityResponse := getJSON(t, server.URL+"/api/missions/"+missionID+"/activity")
	polled := nestedMap(t, activityResponse, "activity")
	polledLatest := nestedMap(t, polled, "latest_terminal_activity")
	if polled["last_sequence"] != float64(3) || polledLatest["outcome"] != string(app.TerminalActivityFailed) {
		t.Fatalf("polled activity = %#v", activityResponse)
	}
	cursor := nestedMap(t, activityResponse, "cursor")
	if cursor["schema"] != missionActivityCursorSchema || cursor["sequence"] != float64(3) || cursor["server_id"] == "" {
		t.Fatalf("activity cursor = %#v", cursor)
	}
	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	detailCursor := nestedMap(t, detail, "activity_cursor")
	if detailCursor["schema"] != missionActivityCursorSchema || detailCursor["sequence"] != float64(3) || detailCursor["server_id"] != cursor["server_id"] {
		t.Fatalf("detail activity cursor = %#v, activity cursor = %#v", detailCursor, cursor)
	}
}

func TestMissionArchiveRestoreHidesListAndKeepsDetailData(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	server := httptest.NewServer(NewServer(service, Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Archive me"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/text", map[string]any{
		"title": "Kept source", "content": "Archived missions keep source data.",
	})

	archived := postJSON(t, server.URL+"/api/missions/"+missionID+"/archive", map[string]any{"reason": "finished"})
	if nestedString(t, archived, "event", "EventType") != app.MissionArchivedEvent {
		t.Fatalf("archive result = %#v", archived)
	}
	if nestedString(t, archived, "projection", "lifecycle_state") != app.MissionLifecycleArchived {
		t.Fatalf("archive projection = %#v", archived["projection"])
	}

	defaultList := getJSON(t, server.URL+"/api/missions")
	if missions := defaultList["missions"].([]any); len(missions) != 0 {
		t.Fatalf("default list must hide archived mission: %#v", defaultList)
	}
	archivedList := getJSON(t, server.URL+"/api/missions?include_archived=true")
	missions := archivedList["missions"].([]any)
	if len(missions) != 1 {
		t.Fatalf("include archived list = %#v", archivedList)
	}
	item := missions[0].(map[string]any)
	if item["lifecycle_state"] != app.MissionLifecycleArchived {
		t.Fatalf("archived list item = %#v", item)
	}
	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	if nestedString(t, detail, "projection", "lifecycle_state") != app.MissionLifecycleArchived {
		t.Fatalf("detail projection = %#v", detail["projection"])
	}
	if sources := detail["sources"].([]any); len(sources) != 1 {
		t.Fatalf("archived detail must retain sources: %#v", detail["sources"])
	}

	restored := postJSON(t, server.URL+"/api/missions/"+missionID+"/restore", map[string]any{})
	if nestedString(t, restored, "event", "EventType") != app.MissionRestoredEvent {
		t.Fatalf("restore result = %#v", restored)
	}
	defaultList = getJSON(t, server.URL+"/api/missions")
	if missions := defaultList["missions"].([]any); len(missions) != 1 {
		t.Fatalf("restored mission must return to default list: %#v", defaultList)
	}
}

func TestMissionHardDeletePreviewAndDelete(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	server := httptest.NewServer(NewServer(service, Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Delete me"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	keepMission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Keep me"})
	keepMissionID := nestedString(t, keepMission, "projection", "mission_id")
	source := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/text", map[string]any{
		"title": "Deleted source", "content": "Deleted source body.",
	})
	snapshotID := nestedString(t, source, "snapshot", "SnapshotID")

	status, failure := deleteJSONBodyFailure(t, server.URL+"/api/missions/"+missionID, map[string]any{"confirm_mission_id": missionID})
	if status != http.StatusConflict || !strings.Contains(nestedString(t, failure, "error", "message"), "eligible") {
		t.Fatalf("expected active hard delete conflict, got %d %#v", status, failure)
	}

	postJSON(t, server.URL+"/api/missions/"+missionID+"/archive", map[string]any{"reason": "finished"})
	preview := getJSON(t, server.URL+"/api/missions/"+missionID+"/hard_delete_preview")
	if eligible, _ := preview["eligible"].(bool); !eligible {
		t.Fatalf("expected archived mission to be eligible, got %#v", preview)
	}
	impact := nestedMap(t, preview, "impact")
	if impact["raw_artifacts"] != float64(1) || impact["source_snapshots"] != float64(1) || impact["source_snapshot_artifact_links"] != float64(1) {
		t.Fatalf("unexpected hard delete impact: %#v", impact)
	}

	status, failure = deleteJSONBodyFailure(t, server.URL+"/api/missions/"+missionID, map[string]any{"confirm_mission_id": keepMissionID})
	if status != http.StatusBadRequest || !strings.Contains(nestedString(t, failure, "error", "message"), "confirmation") {
		t.Fatalf("expected confirmation mismatch failure, got %d %#v", status, failure)
	}
	deleted := deleteJSONBody(t, server.URL+"/api/missions/"+missionID, map[string]any{"confirm_mission_id": missionID})
	if deleted["deleted"] != true || nestedString(t, deleted, "mission_id") != missionID {
		t.Fatalf("hard delete result = %#v", deleted)
	}

	list := getJSON(t, server.URL+"/api/missions?include_archived=true")
	missions := list["missions"].([]any)
	if len(missions) != 1 || missions[0].(map[string]any)["MissionID"] != keepMissionID {
		t.Fatalf("deleted mission remained in list: %#v", list)
	}
	status, failure = getJSONFailure(t, server.URL+"/api/missions/"+missionID)
	if status != http.StatusNotFound {
		t.Fatalf("expected deleted mission detail 404, got %d %#v", status, failure)
	}
	status, failure = getJSONFailure(t, server.URL+"/api/missions/"+missionID+"/hard_delete_preview")
	if status != http.StatusNotFound {
		t.Fatalf("expected deleted mission preview 404, got %d %#v", status, failure)
	}
	status, failure = getJSONFailure(t, server.URL+"/api/missions/"+missionID+"/sources/"+snapshotID+"/read")
	if status != http.StatusNotFound {
		t.Fatalf("expected deleted source read 404, got %d %#v", status, failure)
	}
}

func TestMissionArchiveRejectsOpenActiveWork(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	server := httptest.NewServer(NewServer(service, Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Busy"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_busy_turn", MissionID: missionID, EventType: "turn.agent.pending",
		Producer: app.Producer{Type: "agent", ID: "test"}, Payload: mustJSON(map[string]any{"user_event_id": "evt_user"}),
	}); err != nil {
		t.Fatal(err)
	}
	status, failure := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/archive", map[string]any{})
	if status != http.StatusBadRequest || !strings.Contains(nestedString(t, failure, "error", "message"), "agent turn") {
		t.Fatalf("archive active work failure status=%d body=%#v", status, failure)
	}
}

func TestWorkspaceFlow(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	agent := &fakeAgentExecutor{rejectDeadline: true, responses: []AgentResult{
		{Text: "first answer", SessionID: "agent-session-1"},
		{Text: agentReportPlanJSON(agentReportPlan{
			Summary: "Cover HTTPS DNS records from the approved source.",
			Sections: []agentReportSection{{
				Title:   "HTTPS DNS records",
				Purpose: "Explain the approved source-backed facts.",
			}},
			CoverageNotes: []string{"Approved HTTPS DNS source cluster."},
		}), SessionID: "agent-session-1"},
		{Text: "# DNS report\n\nHTTPS DNS records should be explained through the pinned source.", SessionID: "agent-session-1"},
	}}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{
		"title":     "DNS test",
		"objective": "Check browser workspace flow",
	})
	missionID := nestedString(t, mission, "projection", "mission_id")

	source := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/text", map[string]any{
		"title":        "IANA note",
		"external_uri": "https://www.iana.org/assignments/dns-parameters/",
		"content":      "HTTPS is DNS RR type 65. SVCB is DNS RR type 64.",
	})
	snapshotID := nestedString(t, source, "snapshot", "SnapshotID")
	artifactID := nestedString(t, source, "artifact", "ArtifactID")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{
		"text": "Summarize HTTPS DNS records.",
	})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	proposal := postJSON(t, server.URL+"/api/missions/"+missionID+"/candidates/evidence", map[string]any{
		"summary":     "HTTPS is a DNS resource record type used for HTTPS service binding metadata.",
		"snapshot_id": snapshotID,
		"artifact_id": artifactID,
	})
	proposalID := nestedString(t, proposal, "Proposal", "proposal_id")
	evidenceID := nestedString(t, proposal, "Evidence", "evidence_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/proposals/"+proposalID+"/approve", map[string]any{})
	claimID := "clm_workspace_report"
	claimProposalID := "prp_workspace_report_claim"
	if _, err := svc.CreateClaimProposal(ctx, app.CreateClaimProposalRequest{
		ClaimEvent: app.AppendEventRequest{
			EventID:   "evt_workspace_report_claim",
			MissionID: missionID,
			EventType: "claim.proposed",
			Producer:  app.Producer{Type: "agent_session", ID: "ses_workspace_test"},
			Payload: mustJSON(map[string]any{
				"claim_id":    claimID,
				"proposal_id": claimProposalID,
			}),
		},
		Claim: app.CreateClaimRecordRequest{
			ClaimID:               claimID,
			MissionID:             missionID,
			State:                 "proposed",
			Text:                  "HTTPS DNS records should be explained as HTTPS service binding metadata.",
			ClaimType:             "descriptive",
			SupportingEvidenceIDs: []string{evidenceID},
			Confidence:            app.Confidence{Level: "medium"},
			Approval:              app.Approval{State: "pending", Required: true},
			CreatedEventID:        "evt_workspace_report_claim",
		},
		ProposalEvent: app.AppendEventRequest{
			EventID:   "evt_workspace_report_claim_proposal",
			MissionID: missionID,
			EventType: "proposal.submitted",
			Producer:  app.Producer{Type: "agent_session", ID: "ses_workspace_test"},
			Payload: mustJSON(map[string]any{
				"proposal_id": claimProposalID,
			}),
		},
		Proposal: app.CreateProposalBundleRequest{
			ProposalID:        claimProposalID,
			MissionID:         missionID,
			Title:             "Review claim",
			ObjectRefs:        []app.ObjectRef{{ObjectKind: app.ClaimRecordObjectKind, ObjectID: claimID}},
			RequestedDecision: "approve",
			CreatedEventID:    "evt_workspace_report_claim_proposal",
		},
	}); err != nil {
		t.Fatal(err)
	}
	postJSON(t, server.URL+"/api/missions/"+missionID+"/proposals/"+claimProposalID+"/approve", map[string]any{})

	reportStart := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":       "DNS report",
		"rigor_level": "strict",
		"report_mode": "planned",
	})
	if pendingType := nestedString(t, reportStart, "pending_event", "EventType"); pendingType != "report.draft.pending" {
		t.Fatalf("expected report draft pending event, got %q", pendingType)
	}
	if rigor := nestedString(t, reportStart, "pending_event", "Payload", "rigor_level"); rigor != "strict" {
		t.Fatalf("expected strict report rigor in pending event, got %q", rigor)
	}
	detailWithReport := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if len(agent.requests) < 3 {
		t.Fatalf("expected report agent request, got %#v", agent.requests)
	}
	planReq := agent.requests[1]
	if planReq.PreviousSessionID != "agent-session-1" {
		t.Fatalf("report planning must continue the mission session, got %q", planReq.PreviousSessionID)
	}
	for _, expected := range []string{
		"Create a user-visible Korean report generation plan",
		"prior user turns, agent responses, controller questions",
		"Treat repeated or explicit user questions as coverage signals",
		"target_refs should name only approved records",
		"mission_id " + missionID,
	} {
		if !strings.Contains(planReq.Prompt, expected) {
			t.Fatalf("expected report plan prompt to contain %q:\n%s", expected, planReq.Prompt)
		}
	}
	reportReq := agent.requests[2]
	if reportReq.PreviousSessionID != "agent-session-1" {
		t.Fatalf("report generation must continue the planning session, got %q", reportReq.PreviousSessionID)
	}
	if reportReq.ToolSessionID == "" {
		t.Fatal("report generation should receive a tool session id")
	}
	for _, expected := range []string{
		"plasma.research.outline",
		"plasma.research.list",
		"plasma.research.read",
		"plasma.research.references",
		"mission_id " + missionID,
		"not a thin stitched summary",
		"Use prior investigation answers, normal conversation, and controller questions as working memory only",
		"visible generation plan below was created in the previous step",
		"coverage contract",
		"Visible generation plan",
		"Level: strict",
		"검증형",
		"temporary paths, or working directories",
	} {
		if !strings.Contains(reportReq.Prompt, expected) {
			t.Fatalf("expected report prompt to contain %q:\n%s", expected, reportReq.Prompt)
		}
	}
	for _, forbidden := range []string{"Mission recall:", "plasma.agent_recall_preview", "Sources\":", "Evidence\":", "Claims\":"} {
		if strings.Contains(reportReq.Prompt, forbidden) {
			t.Fatalf("report prompt contains forbidden recall/source payload marker %q:\n%s", forbidden, reportReq.Prompt)
		}
	}
	if countEvents(detailWithReport, "report.drafted") != 0 {
		t.Fatalf("default report path must not create legacy AST draft events, got %#v", detailWithReport["events"])
	}
	if countEvents(detailWithReport, "report.plan.created") != 1 {
		t.Fatalf("expected one visible markdown report plan event, got %#v", detailWithReport["events"])
	}
	reportPayload := lastEventPayload(t, detailWithReport, "report.artifact.created")
	if reportPayload["rigor_level"] != "strict" || reportPayload["rigor_label"] != "검증형" {
		t.Fatalf("expected strict report artifact rigor payload, got %#v", reportPayload)
	}
	if reportPayload["plan_event_id"] == "" || reportPayload["plan_tool_session_id"] == "" {
		t.Fatalf("expected report artifact to link to plan event and plan tool session, got %#v", reportPayload)
	}
	planPayload := lastEventPayload(t, detailWithReport, "report.plan.created")
	if planPayload["pending_event_id"] != reportPayload["pending_event_id"] || planPayload["artifact_id"] != reportPayload["artifact_id"] {
		t.Fatalf("expected report plan to be linked to artifact, plan=%#v artifact=%#v", planPayload, reportPayload)
	}
	artifact, err := svc.GetRawArtifact(ctx, reportPayload["artifact_id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if artifact.MediaType != "text/markdown; charset=utf-8" || !strings.Contains(string(artifact.Content), "HTTPS DNS records") {
		t.Fatalf("expected markdown report artifact, got %#v", artifact)
	}
	artifactRead := getJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifact.ArtifactID)
	if got := nestedString(t, artifactRead, "artifact", "artifact_id"); got != artifact.ArtifactID {
		t.Fatalf("expected artifact read id %q, got %q", artifact.ArtifactID, got)
	}
	if got := nestedString(t, artifactRead, "artifact", "mission_id"); got != missionID {
		t.Fatalf("expected artifact read mission %q, got %q", missionID, got)
	}
	if content, _ := artifactRead["content"].(string); !strings.Contains(content, "HTTPS DNS records") {
		t.Fatalf("expected artifact read content, got %#v", artifactRead)
	}
	downloadResp, err := http.Get(server.URL + "/api/missions/" + missionID + "/artifacts/" + artifact.ArtifactID + "/download")
	if err != nil {
		t.Fatal(err)
	}
	defer downloadResp.Body.Close()
	if downloadResp.StatusCode != http.StatusOK {
		t.Fatalf("expected artifact download 200, got %d", downloadResp.StatusCode)
	}
	if got := downloadResp.Header.Get("Content-Type"); got != "text/markdown; charset=utf-8" {
		t.Fatalf("expected markdown download content-type, got %q", got)
	}
	body, err := io.ReadAll(downloadResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "HTTPS DNS records") {
		t.Fatalf("expected artifact download body, got %q", string(body))
	}
	status, bodyJSON := getJSONFailure(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID)
	if status != http.StatusNotFound {
		t.Fatalf("expected source artifact read 404, got %d: %#v", status, bodyJSON)
	}
	otherMission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Other mission"})
	otherMissionID := nestedString(t, otherMission, "projection", "mission_id")
	status, bodyJSON = getJSONFailure(t, server.URL+"/api/missions/"+otherMissionID+"/artifacts/"+artifact.ArtifactID)
	if status != http.StatusNotFound {
		t.Fatalf("expected cross-mission artifact read 404, got %d: %#v", status, bodyJSON)
	}
	if len(agent.requests) != 3 {
		t.Fatalf("expected answer turn, report planning turn, and markdown report generation turn, got %d", len(agent.requests))
	}
	if strings.Contains(agent.requests[0].Prompt, "Source excerpts") {
		t.Fatal("agent prompt must not contain source excerpts")
	}
	if strings.Contains(agent.requests[0].Prompt, "HTTPS is DNS RR type 65") {
		t.Fatal("agent prompt must not paste source body content")
	}
	if !strings.Contains(agent.requests[0].Prompt, "Mission reminder") {
		t.Fatal("expected agent prompt to contain a mission reminder")
	}
	if !strings.Contains(agent.requests[1].Prompt, "Create a user-visible Korean report generation plan") {
		t.Fatalf("expected C1 markdown report planning prompt, got %q", agent.requests[1].Prompt)
	}
	if !strings.Contains(agent.requests[2].Prompt, "Markdown artifact") ||
		strings.Contains(agent.requests[2].Prompt, "structured AST JSON") ||
		strings.Contains(agent.requests[2].Prompt, "evidence_ids") {
		t.Fatalf("expected C1 markdown report prompt, got %q", agent.requests[2].Prompt)
	}
	if agent.requests[0].MissionID != missionID {
		t.Fatalf("expected agent request mission binding %q, got %q", missionID, agent.requests[0].MissionID)
	}
	if !strings.HasPrefix(agent.requests[0].ToolSessionID, "ses_") {
		t.Fatalf("expected tool session id, got %q", agent.requests[0].ToolSessionID)
	}
	if !strings.Contains(agent.requests[0].Prompt, "Plasma tool binding") ||
		!strings.Contains(agent.requests[0].Prompt, agent.requests[0].ToolSessionID) {
		t.Fatalf("expected prompt to include tool binding, got %q", agent.requests[0].Prompt)
	}
	for _, req := range agent.requests {
		if strings.Contains(req.Prompt, "Create review proposals for the latest answer") {
			t.Fatalf("default C1 flow must not run proposal extraction, got %q", req.Prompt)
		}
	}
}

func TestConfluenceSourceAPIWorkflow(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)

	var authHeaders []string
	var metadataQuery string
	pageVersion := 7
	oldDefaultClient := http.DefaultClient
	fallbackTransport := oldDefaultClient.Transport
	if fallbackTransport == nil {
		fallbackTransport = http.DefaultTransport
	}
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Hostname() != "docs.atlassian.net" {
			return fallbackTransport.RoundTrip(r)
		}
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		body := ""
		switch r.URL.Path {
		case "/wiki/rest/api/search":
			body = `{
				"results": [{
					"content": {
						"id": "123",
						"title": "Roadmap",
						"space": {"id": "987", "key": "ENG"},
						"version": {"when": "2026-07-02T05:10:00.000Z", "number": 7},
						"_links": {"webui": "/spaces/ENG/pages/123/Roadmap"}
					},
					"excerpt": "<p>roadmap result</p>",
					"url": "/spaces/ENG/pages/123/Roadmap"
				}],
				"_links": {"base": "https://docs.atlassian.net/wiki"}
			}`
		case "/wiki/api/v2/spaces":
			body = `{
				"results": [{"id":"987","key":"ENG","name":"Engineering","_links":{"webui":"/spaces/ENG"}}],
				"_links": {"base":"https://docs.atlassian.net/wiki"}
			}`
		case "/wiki/api/v2/spaces/987/pages":
			body = `{
				"results": [{"id":"123","title":"Roadmap","spaceId":"987","version":{"createdAt":"2026-07-02T05:10:00.000Z","number":7},"_links":{"webui":"/spaces/ENG/pages/123/Roadmap"}}],
				"_links": {"base":"https://docs.atlassian.net/wiki"}
			}`
		case "/wiki/api/v2/pages/123/children":
			body = `{
				"results": [{"id":"456","title":"Child page","spaceId":"987","version":{"number":1},"_links":{"webui":"/spaces/ENG/pages/456/Child"}}],
				"_links": {"base":"https://docs.atlassian.net/wiki"}
			}`
		case "/wiki/api/v2/pages/123":
			if r.URL.Query().Get("body-format") == "" {
				metadataQuery = r.URL.RawQuery
			}
			body = fmt.Sprintf(`{
				"id": "123",
				"title": "Roadmap",
				"spaceId": "987",
				"version": {"createdAt": "2026-07-03T02:00:00.000Z", "number": %d},
				"body": {"storage": {"value": "<p>Hello version %d</p>", "representation": "storage"}},
				"_links": {"base": "https://docs.atlassian.net/wiki", "webui": "/spaces/ENG/pages/123/Roadmap"}
			}`, pageVersion, pageVersion)
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`)), Request: r}, nil
		}
		header := make(http.Header)
		header.Set("Content-Type", "application/json")
		return &http.Response{StatusCode: http.StatusOK, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	defer func() { http.DefaultClient = oldDefaultClient }()
	server := httptest.NewServer(NewServer(svc, Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Confluence API"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	connection := postJSON(t, server.URL+"/api/settings/connectors/confluence/connections", map[string]any{
		"connection_id": "cnf_web",
		"display_name":  "Docs",
		"auth_type":     app.ConfluenceAuthTypeAPIToken,
		"account_name":  "person@example.com",
		"api_token":     "secret-api-token",
		"sites": []map[string]any{{
			"url": "https://docs.atlassian.net/wiki",
		}},
	})
	connectionRaw := mustMarshalTestJSON(t, connection)
	if strings.Contains(connectionRaw, "secret-api-token") {
		t.Fatalf("connection response leaked token: %s", connectionRaw)
	}
	cloudID := "site_docs.atlassian.net"

	search := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/search", map[string]any{
		"connection_id": "cnf_web",
		"cloud_id":      cloudID,
		"query":         "roadmap",
	})
	if !strings.Contains(mustMarshalTestJSON(t, search), "Roadmap") {
		t.Fatalf("expected Roadmap candidate, got %#v", search)
	}

	spaces := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/spaces", map[string]any{
		"connection_id": "cnf_web",
		"cloud_id":      cloudID,
	})
	if !strings.Contains(mustMarshalTestJSON(t, spaces), `"space_id":"987"`) {
		t.Fatalf("expected browsed space, got %#v", spaces)
	}
	pages := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/space-pages", map[string]any{
		"connection_id": "cnf_web",
		"cloud_id":      cloudID,
		"space_id":      "987",
	})
	if !strings.Contains(mustMarshalTestJSON(t, pages), `"page_id":"123"`) {
		t.Fatalf("expected browsed page, got %#v", pages)
	}
	children := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/children", map[string]any{
		"connection_id": "cnf_web",
		"cloud_id":      cloudID,
		"page_id":       "123",
	})
	if !strings.Contains(mustMarshalTestJSON(t, children), "Child page") {
		t.Fatalf("expected child page, got %#v", children)
	}

	preview := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/preview", map[string]any{
		"connection_id":    "cnf_web",
		"cloud_id":         cloudID,
		"page_id":          "123",
		"expected_version": 7,
		"max_body_bytes":   8,
	})
	previewRaw := mustMarshalTestJSON(t, preview)
	if !strings.Contains(previewRaw, `"full_body_too_large":true`) || !strings.Contains(previewRaw, `"range_options"`) {
		t.Fatalf("expected large page preview with ranges, got %s", previewRaw)
	}

	status, failure := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/snapshot", map[string]any{
		"connection_id": "cnf_web",
		"cloud_id":      cloudID,
		"page_id":       "123",
	})
	if status != http.StatusBadRequest || !strings.Contains(nestedString(t, failure, "error", "message"), "page version") {
		t.Fatalf("expected missing snapshot version rejection, got %d %#v", status, failure)
	}

	rangeSnapshot := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/snapshot", map[string]any{
		"connection_id":    "cnf_web",
		"cloud_id":         cloudID,
		"page_id":          "123",
		"expected_version": 7,
		"max_body_bytes":   8,
		"range_content_id": "plain_text",
		"range_start":      6,
		"range_end":        13,
	})
	if !strings.Contains(mustMarshalTestJSON(t, rangeSnapshot), "confluence_page_range") {
		t.Fatalf("expected range snapshot locator, got %#v", rangeSnapshot)
	}

	snapshot := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/snapshot", map[string]any{
		"connection_id":    "cnf_web",
		"cloud_id":         cloudID,
		"page_id":          "123",
		"expected_version": 7,
	})
	snapshotID := nestedString(t, snapshot, "Snapshot", "SnapshotID")
	if snapshotID == "" {
		t.Fatalf("expected snapshot id, got %#v", snapshot)
	}
	accessAfterSnapshot := getJSON(t, server.URL+"/api/missions/"+missionID+"/connector-access/confluence")
	if enabled, _ := nestedValue(t, accessAfterSnapshot, "access", "enabled").(bool); enabled {
		t.Fatalf("source snapshot attachment must not enable Confluence agent search grant: %#v", accessAfterSnapshot)
	}

	pageVersion = 8
	check := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/check-update", map[string]any{
		"connection_id": "cnf_web",
		"snapshot_id":   snapshotID,
	})
	checkRaw := mustMarshalTestJSON(t, check)
	if metadataQuery != "" {
		t.Fatalf("metadata request should not request body, got query %q", metadataQuery)
	}
	if !strings.Contains(checkRaw, `"UpdateAvailable":true`) {
		t.Fatalf("expected update available, got %s", checkRaw)
	}
	if strings.Contains(checkRaw, "Hello version 8") {
		t.Fatalf("check-update leaked page body: %s", checkRaw)
	}
	updatePreview := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/update-preview", map[string]any{
		"connection_id":    "cnf_web",
		"snapshot_id":      snapshotID,
		"expected_version": 8,
	})
	if !strings.Contains(mustMarshalTestJSON(t, updatePreview), "Hello version 8") {
		t.Fatalf("expected explicit update preview body, got %#v", updatePreview)
	}

	status, failure = postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/update", map[string]any{
		"connection_id": "cnf_web",
		"snapshot_id":   snapshotID,
	})
	if status != http.StatusBadRequest || !strings.Contains(nestedString(t, failure, "error", "message"), "page version") {
		t.Fatalf("expected missing update version rejection, got %d %#v", status, failure)
	}

	updated := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/update", map[string]any{
		"connection_id":    "cnf_web",
		"snapshot_id":      snapshotID,
		"expected_version": 8,
	})
	newSnapshotID := nestedString(t, updated, "Snapshot", "SnapshotID")
	if newSnapshotID == "" || newSnapshotID == snapshotID {
		t.Fatalf("expected new snapshot, got %#v", updated)
	}
	renameBody := bytes.NewBufferString(`{"display_name":"Docs renamed"}`)
	renameReq, err := http.NewRequest(http.MethodPatch, server.URL+"/api/settings/connectors/confluence/connections/cnf_web", renameBody)
	if err != nil {
		t.Fatal(err)
	}
	renameReq.Header.Set("Content-Type", "application/json")
	renameResp, err := http.DefaultClient.Do(renameReq)
	if err != nil {
		t.Fatal(err)
	}
	renameResp.Body.Close()
	if renameResp.StatusCode != http.StatusOK {
		t.Fatalf("expected rename status 200, got %d", renameResp.StatusCode)
	}
	revoke := postJSON(t, server.URL+"/api/settings/connectors/confluence/connections/cnf_web/revoke", map[string]any{})
	if !strings.Contains(mustMarshalTestJSON(t, revoke), `"revoked":true`) {
		t.Fatalf("expected revoke result, got %#v", revoke)
	}
	readOld := getJSON(t, server.URL+"/api/missions/"+missionID+"/sources/"+snapshotID+"/read")
	if !strings.Contains(mustMarshalTestJSON(t, readOld), "Hello version 7") {
		t.Fatalf("expected old snapshot readable after revoke, got %#v", readOld)
	}
	deleteJSON(t, server.URL+"/api/settings/connectors/confluence/connections/cnf_web")
	readAfterDelete := getJSON(t, server.URL+"/api/missions/"+missionID+"/sources/"+snapshotID+"/read")
	if !strings.Contains(mustMarshalTestJSON(t, readAfterDelete), "Hello version 7") {
		t.Fatalf("expected old snapshot readable after connection delete, got %#v", readAfterDelete)
	}
	for _, auth := range authHeaders {
		if auth != "" && !strings.HasPrefix(auth, "Basic ") {
			t.Fatalf("unexpected auth header %q", auth)
		}
	}
}

func TestConfluenceSettingsRoutesAndLegacyLifecycleDeprecation(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	created := postJSON(t, server.URL+"/api/settings/connectors/confluence/connections", map[string]any{
		"connection_id": "cnf_settings",
		"display_name":  "Docs",
		"auth_type":     app.ConfluenceAuthTypeAPIToken,
		"account_name":  "person@example.com",
		"api_token":     "secret-api-token",
		"sites":         []map[string]any{{"url": "https://docs.atlassian.net/wiki/"}},
	})
	rawCreated := mustMarshalTestJSON(t, created)
	if strings.Contains(rawCreated, "secret-api-token") {
		t.Fatalf("settings create response leaked API token: %s", rawCreated)
	}
	list := getJSON(t, server.URL+"/api/settings/connectors/confluence/connections")
	if !strings.Contains(mustMarshalTestJSON(t, list), "cnf_settings") {
		t.Fatalf("expected settings connection list, got %#v", list)
	}

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Legacy compatibility"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	compatList := getJSON(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/connections")
	if !strings.Contains(mustMarshalTestJSON(t, compatList), "cnf_settings") {
		t.Fatalf("expected legacy list wrapper to remain safe, got %#v", compatList)
	}
	for _, tc := range []struct {
		name string
		call func() (int, map[string]any)
	}{
		{
			name: "legacy create",
			call: func() (int, map[string]any) {
				return postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/connections", map[string]any{})
			},
		},
		{
			name: "legacy rename",
			call: func() (int, map[string]any) {
				body := bytes.NewBufferString(`{"display_name":"Old path"}`)
				req, err := http.NewRequest(http.MethodPatch, server.URL+"/api/missions/"+missionID+"/sources/confluence/connections/cnf_settings", body)
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Content-Type", "application/json")
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				defer resp.Body.Close()
				var payload map[string]any
				if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				return resp.StatusCode, payload
			},
		},
		{
			name: "legacy oauth start",
			call: func() (int, map[string]any) {
				return postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/oauth/start", map[string]any{})
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, payload := tc.call()
			if status != http.StatusGone || !strings.Contains(nestedString(t, payload, "error", "message"), "/api/settings/connectors/confluence") {
				t.Fatalf("expected legacy lifecycle deprecation, got %d %#v", status, payload)
			}
		})
	}
}

func TestRuntimeEndpointReturnsEnvironmentLabel(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := httptest.NewServer(NewServer(app.NewService(store), Options{EnvironmentLabel: "DEV"}))
	defer server.Close()

	runtime := getJSON(t, server.URL+"/api/runtime")
	if got := nestedString(t, runtime, "environment_label"); got != "DEV" {
		t.Fatalf("expected runtime environment label, got %#v", runtime)
	}
}

func TestMissionMetadataPatchUpdatesSharedProjection(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Original", "objective": "Keep"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	updated := patchJSON(t, server.URL+"/api/missions/"+missionID, map[string]any{"title": " Updated ", "scope": map[string]any{"included": []string{" A ", ""}, "excluded": []string{}}})
	if nestedString(t, updated, "projection", "title") != "Updated" {
		t.Fatalf("unexpected update: %#v", updated)
	}
	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	if nestedString(t, detail, "projection", "objective") != "Keep" {
		t.Fatalf("partial update cleared objective: %#v", detail)
	}
	status, _ := patchJSONFailure(t, server.URL+"/api/missions/"+missionID, map[string]any{})
	if status != http.StatusBadRequest {
		t.Fatalf("no-op status = %d", status)
	}
	request, _ := http.NewRequest(http.MethodPost, server.URL+"/api/missions/"+missionID, strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("wrong method status = %d", response.StatusCode)
	}
}

func TestConfluenceConnectorAccessAPIUsesLedgerAndDoesNotAttachSources(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Confluence grant"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/settings/connectors/confluence/connections", map[string]any{
		"connection_id": "cnf_grant",
		"display_name":  "Docs",
		"auth_type":     app.ConfluenceAuthTypeAPIToken,
		"account_name":  "person@example.com",
		"api_token":     "secret-api-token",
		"sites":         []map[string]any{{"url": "https://docs.atlassian.net/wiki/"}},
	})

	defaultAccess := getJSON(t, server.URL+"/api/missions/"+missionID+"/connector-access/confluence")
	if enabled, _ := nestedValue(t, defaultAccess, "access", "enabled").(bool); enabled {
		t.Fatalf("expected default-off connector access, got %#v", defaultAccess)
	}
	status, failure := putJSONFailure(t, server.URL+"/api/missions/"+missionID+"/connector-access/confluence", map[string]any{
		"enabled":       true,
		"connection_id": "cnf_grant",
		"cloud_id":      "site_docs.atlassian.net",
		"reason":        "Bearer should-not-enter-ledger",
	})
	if status != http.StatusBadRequest || !strings.Contains(nestedString(t, failure, "error", "message"), "unknown field") {
		t.Fatalf("expected reason field rejection, got %d %#v", status, failure)
	}
	beforeSources := getJSON(t, server.URL+"/api/missions/"+missionID+"/sources")
	enabled := putJSON(t, server.URL+"/api/missions/"+missionID+"/connector-access/confluence", map[string]any{
		"enabled":       true,
		"connection_id": "cnf_grant",
		"cloud_id":      "site_docs.atlassian.net",
		"space_key":     "ENG",
	})
	if !nestedBool(t, enabled, "access", "enabled") || nestedString(t, enabled, "event", "EventType") != app.ConnectorAccessEventEnabled {
		t.Fatalf("expected enabled grant event, got %#v", enabled)
	}
	afterSources := getJSON(t, server.URL+"/api/missions/"+missionID+"/sources")
	if mustMarshalTestJSON(t, beforeSources) != mustMarshalTestJSON(t, afterSources) {
		t.Fatalf("enabling grant must not attach or mutate sources: before=%#v after=%#v", beforeSources, afterSources)
	}
	events := getJSON(t, server.URL+"/api/missions/"+missionID+"/events")
	rawEvents := mustMarshalTestJSON(t, events)
	for _, leaked := range []string{"secret-api-token", "Authorization", "Bearer"} {
		if strings.Contains(rawEvents, leaked) {
			t.Fatalf("connector access ledger leaked %q: %s", leaked, rawEvents)
		}
	}
	if !strings.Contains(rawEvents, app.ConnectorAccessEventEnabled) || !strings.Contains(rawEvents, `"connector_id":"confluence"`) {
		t.Fatalf("expected connector access ledger event, got %s", rawEvents)
	}

	postJSON(t, server.URL+"/api/settings/connectors/confluence/connections/cnf_grant/revoke", map[string]any{})
	invalid := getJSON(t, server.URL+"/api/missions/"+missionID+"/connector-access/confluence")
	if nestedString(t, invalid, "access", "status") != app.ConnectorAccessStatusInvalid {
		t.Fatalf("expected revoked connection to invalidate projected grant, got %#v", invalid)
	}
	disabled := deleteJSON(t, server.URL+"/api/missions/"+missionID+"/connector-access/confluence")
	if nestedBool(t, disabled, "access", "enabled") {
		t.Fatalf("expected disabled connector access, got %#v", disabled)
	}
}

func TestFileUploadSourceStoresReadableTextAndDeduplicatesArtifact(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	server := httptest.NewServer(NewServer(svc, Options{}))
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Upload source"})
	missionID := nestedString(t, mission, "projection", "mission_id")

	first := postMultipartFile(t, server.URL+"/api/missions/"+missionID+"/sources/upload", "notes.md", "text/markdown", []byte("# Notes\n\nUploaded body."), "Uploaded notes")
	if nestedString(t, first, "snapshot", "Connector", "ConnectorType") != app.SourceConnectorTypeFileUpload {
		t.Fatalf("expected file_upload connector, got %#v", first)
	}
	firstArtifactID := nestedString(t, first, "artifact", "ArtifactID")
	firstSnapshotID := nestedString(t, first, "snapshot", "SnapshotID")
	if _, leaked := nestedMap(t, first, "artifact")["Content"]; leaked {
		t.Fatalf("upload response must not include artifact content: %#v", first["artifact"])
	}
	read := getJSON(t, server.URL+"/api/missions/"+missionID+"/sources/"+firstSnapshotID+"/read?max_bytes=8")
	if nestedString(t, read, "content") != "# Notes\n" || nestedFloat(t, read, "next_offset") == 0 {
		t.Fatalf("expected bounded uploaded text read, got %#v", read)
	}
	locators := mustMarshalTestJSON(t, nestedValue(t, first, "snapshot", "Locators"))
	for _, expected := range []string{`"locator_type":"full_document"`, "notes.md", "text/markdown", "text"} {
		if !strings.Contains(locators, expected) {
			t.Fatalf("expected locator to contain %q, got %s", expected, locators)
		}
	}
	if strings.Contains(locators, `"kind":"file_upload"`) {
		t.Fatalf("uploaded text locator must not use file_upload as locator kind: %s", locators)
	}

	second := postMultipartFile(t, server.URL+"/api/missions/"+missionID+"/sources/upload", "copy.md", "text/markdown", []byte("# Notes\n\nUploaded body."), "")
	if nestedString(t, second, "artifact", "ArtifactID") != firstArtifactID {
		t.Fatalf("expected duplicate upload to reuse artifact %s, got %#v", firstArtifactID, second)
	}
	if nestedString(t, second, "snapshot", "SnapshotID") == firstSnapshotID {
		t.Fatalf("expected duplicate upload to create a new snapshot, got %#v", second)
	}
	if existing, ok := second["existing"].(bool); !ok || !existing {
		t.Fatalf("expected duplicate upload response to mark existing artifact, got %#v", second)
	}
}

func TestFileUploadImageReadReturnsMetadataOnly(t *testing.T) {
	store, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Upload image"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	uploaded := postMultipartFile(t, server.URL+"/api/missions/"+missionID+"/sources/upload", "pixel.png", "image/png", testPNGBytes(), "Pixel")
	locators := mustMarshalTestJSON(t, nestedValue(t, uploaded, "snapshot", "Locators"))
	for _, expected := range []string{`"locator_type":"media"`, `"media_kind":"image"`, `"content_kind":"image"`, `"mime_type":"image/png"`} {
		if !strings.Contains(locators, expected) {
			t.Fatalf("expected image upload locator to contain %q, got %s", expected, locators)
		}
	}
	if strings.Contains(locators, `"kind":"file_upload"`) {
		t.Fatalf("uploaded image locator must not use file_upload as locator kind: %s", locators)
	}
	snapshotID := nestedString(t, uploaded, "snapshot", "SnapshotID")
	read := getJSON(t, server.URL+"/api/missions/"+missionID+"/sources/"+snapshotID+"/read")
	if metadataOnly, ok := read["metadata_only"].(bool); !ok || !metadataOnly {
		t.Fatalf("expected image read to be metadata-only, got %#v", read)
	}
	if content, ok := read["content"].(string); !ok || content != "" {
		t.Fatalf("image read must not dump binary content, got %q", content)
	}
	if nestedString(t, read, "artifact", "read_kind") != "metadata" {
		t.Fatalf("expected metadata read kind, got %#v", read)
	}
}

func TestFileUploadPDFReadsExtractedTextAndRejectsUnknownBinary(t *testing.T) {
	store, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Upload PDF"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	pdfBytes := testPDFBytes(t, []string{"Uploaded PDF Source", "Visible extracted text."})
	uploaded := postMultipartFile(t, server.URL+"/api/missions/"+missionID+"/sources/upload", "paper.pdf", "application/pdf", pdfBytes, "Paper")
	locators := mustMarshalTestJSON(t, nestedValue(t, uploaded, "snapshot", "Locators"))
	for _, expected := range []string{`"locator_type":"pdf_document"`, "paper.pdf", `"content_kind":"pdf"`, `"mime_type":"application/pdf"`} {
		if !strings.Contains(locators, expected) {
			t.Fatalf("expected PDF upload locator to contain %q, got %s", expected, locators)
		}
	}
	if strings.Contains(locators, `"kind":"file_upload"`) {
		t.Fatalf("uploaded PDF locator must not use file_upload as locator kind: %s", locators)
	}
	snapshotID := nestedString(t, uploaded, "snapshot", "SnapshotID")
	read := getJSON(t, server.URL+"/api/missions/"+missionID+"/sources/"+snapshotID+"/read?max_bytes=40")
	if content := nestedString(t, read, "content"); !strings.Contains(content, "Uploaded PDF") || strings.Contains(content, "%PDF-") {
		t.Fatalf("expected extracted PDF text without raw bytes, got %#v", read)
	}
	if extraction := nestedMap(t, read, "extraction"); extraction["type"] != "pdf_text" {
		t.Fatalf("expected pdf_text extraction, got %#v", read)
	}

	status, failure := postMultipartFileFailure(t, server.URL+"/api/missions/"+missionID+"/sources/upload", "blob.bin", []byte{0x00, 0x01, 0x02, 0x03})
	if status != http.StatusBadRequest || !strings.Contains(nestedString(t, failure, "error", "message"), "unsupported uploaded file") {
		t.Fatalf("expected unsupported binary rejection, got %d %#v", status, failure)
	}
}

func TestFileUploadPDFInspectsWithoutExtractingTextAtUpload(t *testing.T) {
	store, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "PDF inspect only"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	uploaded := postMultipartFile(t, server.URL+"/api/missions/"+missionID+"/sources/upload", "inspect-only.pdf", "application/pdf", testPDFBytesWithInvalidContentStream(t), "Inspect-only PDF")
	if nestedString(t, uploaded, "snapshot", "Connector", "ConnectorType") != app.SourceConnectorTypeFileUpload {
		t.Fatalf("expected uploaded PDF source, got %#v", uploaded)
	}
}

func TestFileUploadRejectsDisguisedExtensions(t *testing.T) {
	store, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Upload validation"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	uploadURL := server.URL + "/api/missions/" + missionID + "/sources/upload"

	cases := []struct {
		filename string
		content  []byte
		want     string
	}{
		{filename: "fake.pdf", content: []byte("not actually a pdf"), want: "PDF"},
		{filename: "fake.png", content: []byte("not actually a png"), want: "image"},
		{filename: "fake.txt", content: []byte{0x00, 0x01, 't', 'x', 't'}, want: "unsupported uploaded file"},
	}
	for _, tc := range cases {
		status, failure := postMultipartFileFailure(t, uploadURL, tc.filename, tc.content)
		message := nestedString(t, failure, "error", "message")
		if status != http.StatusBadRequest || !strings.Contains(message, tc.want) {
			t.Fatalf("expected %s rejection containing %q, got %d %#v", tc.filename, tc.want, status, failure)
		}
	}
}

func TestFileUploadDoesNotDedupeAgainstNonUploadArtifact(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	server := httptest.NewServer(NewServer(svc, Options{}))
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Upload provenance"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	content := []byte("same bytes but already stored as a result")
	if _, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_existing_result_same_sha",
		MissionID:  missionID,
		MediaType:  "text/plain; charset=utf-8",
		Filename:   "result.txt",
		Producer:   app.Producer{Type: "agent", ID: "codex"},
		Content:    content,
	}); err != nil {
		t.Fatal(err)
	}
	status, failure := postMultipartFileFailure(t, server.URL+"/api/missions/"+missionID+"/sources/upload", "source.txt", content)
	message := nestedString(t, failure, "error", "message")
	if status != http.StatusConflict || !strings.Contains(message, "existing non-upload artifact") {
		t.Fatalf("expected non-upload same-sha conflict, got %d %#v", status, failure)
	}
}

func TestReportArtifactPreviewReturnsFullContent(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	handler := NewServer(svc, Options{})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Report preview"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	content := "# Long Report\n\n" + strings.Repeat("0123456789", 3000)
	artifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_long_report_preview",
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   "long-report.md",
		Producer:   app.Producer{Type: "agent", ID: "codex"},
		Content:    []byte(content),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := appendTestEvent(t, webServer, ctx, missionID, "report.artifact.created", map[string]any{
		"kind":        "markdown_report_artifact",
		"artifact_id": artifact.ArtifactID,
		"media_type":  artifact.MediaType,
	}, app.Producer{Type: "agent", ID: "codex"}); err != nil {
		t.Fatal(err)
	}
	read := getJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifact.ArtifactID)
	if got := nestedString(t, read, "content"); got != content {
		t.Fatalf("expected full report preview content length %d, got %d", len(content), len(got))
	}
	if truncated, ok := read["truncated"].(bool); !ok || truncated {
		t.Fatalf("expected untruncated report preview, got %#v", read)
	}
}

func TestConversationExportCreatesReadableMarkdownArtifact(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	handler := NewServer(svc, Options{})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Conversation export"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	if _, err := appendTestEvent(t, webServer, ctx, missionID, "turn.user", map[string]any{
		"kind":            "user_turn",
		"text":            "기술면접 Q&A를 그대로 뽑아줘",
		"tool_session_id": "ses_private",
	}, app.Producer{Type: "user", ID: "plasma-ui"}); err != nil {
		t.Fatalf("append user turn returned error: %v", err)
	}
	if _, err := appendTestEvent(t, webServer, ctx, missionID, "turn.agent.response", map[string]any{
		"kind":             "agent_response",
		"text":             "Q. HTTP 캐시는 무엇인가?\n\nA. 응답 재사용을 제어하는 메커니즘입니다.",
		"agent_session_id": "ses_private",
		"user_event_id":    "evt_user",
	}, app.Producer{Type: "agent", ID: "codex"}); err != nil {
		t.Fatalf("append agent response returned error: %v", err)
	}

	export := postJSON(t, server.URL+"/api/missions/"+missionID+"/conversation_exports", map[string]any{"title": "Q&A 원문"})
	artifactID := nestedString(t, export, "artifact", "artifact_id")
	if artifactID == "" || nestedString(t, export, "event", "EventType") != app.ConversationExportedEvent {
		t.Fatalf("unexpected conversation export response: %#v", export)
	}
	content, _ := export["content"].(string)
	if !strings.Contains(content, "# Q&A 원문") ||
		!strings.Contains(content, "기술면접 Q&A를 그대로 뽑아줘") ||
		!strings.Contains(content, "Q. HTTP 캐시는 무엇인가?") {
		t.Fatalf("conversation export content missing visible turns:\n%s", content)
	}
	if strings.Contains(content, "ses_private") || strings.Contains(content, "tool_session_id") {
		t.Fatalf("conversation export leaked internal fields:\n%s", content)
	}

	read := getJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID)
	if got := nestedString(t, read, "artifact", "artifact_id"); got != artifactID {
		t.Fatalf("expected readable conversation export artifact %q, got %#v", artifactID, read)
	}
	if readContent, _ := read["content"].(string); !strings.Contains(readContent, "Q. HTTP 캐시는 무엇인가?") {
		t.Fatalf("expected exported content read, got %#v", read)
	}
}

func TestReportPatchPromptDoesNotAdvertiseUnavailableResearchTools(t *testing.T) {
	prompt := agentReportPatchPrompt("Patch", "mis_prompt", "ses_prompt", "evt_pending", "art_base", "Fix wording.", reporting.PatchRequest{
		AgentExecutor:                "codex",
		AgentModel:                   "gpt-test",
		AgentReasoningEffort:         "medium",
		MCPMode:                      "auto",
		ReportSessionID:              "report-session-1",
		PreviousAgentSessionID:       "report-session-1",
		ReportSessionPolicy:          reportSessionPolicySameSession,
		ReportSessionPolicySelection: "test",
		SessionChainKind:             "test",
	})
	if strings.Contains(prompt, "normal Plasma read/search tools") {
		t.Fatalf("patch-only prompt must not advertise unavailable research tools: %s", prompt)
	}
	if !strings.Contains(prompt, "only exposes report patch tools") ||
		!strings.Contains(prompt, "stop and explain the blocker") {
		t.Fatalf("patch-only prompt should tell the agent to stop on unverifiable changes, got %s", prompt)
	}
}

func TestConfluenceOAuthStartCallbackIsDisabled(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	server := httptest.NewServer(NewServer(svc, Options{}))
	defer server.Close()

	connectionList := getJSON(t, server.URL+"/api/settings/connectors/confluence/connections")
	if configured, ok := connectionList["oauth_configured"].(bool); !ok || configured {
		t.Fatalf("expected OAuth to be disabled in connections response, got %#v", connectionList)
	}
	if apiTokenOnly, ok := connectionList["api_token_only"].(bool); !ok || !apiTokenOnly {
		t.Fatalf("expected api_token_only response, got %#v", connectionList)
	}
	status, failure := postJSONFailure(t, server.URL+"/api/settings/connectors/confluence/oauth/start", map[string]any{
		"connection_id": "cnf_oauth",
		"display_name":  "Docs OAuth",
	})
	if status != http.StatusBadRequest || !strings.Contains(nestedString(t, failure, "error", "message"), "API token") {
		t.Fatalf("expected OAuth disabled failure, got %d %#v", status, failure)
	}
}

func TestConfluenceWebRejectsRequestLevelEndpointOverrides(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Confluence overrides"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	for _, tc := range []struct {
		name string
		path string
		body map[string]any
	}{
		{
			name: "oauth start token url",
			path: "/api/settings/connectors/confluence/oauth/start",
			body: map[string]any{"connection_id": "cnf_oauth", "token_url": "https://evil.example/token"},
		},
		{
			name: "site discovery url",
			path: "/api/settings/connectors/confluence/connections/cnf_web/sites/refresh",
			body: map[string]any{"discovery_url": "https://evil.example"},
		},
		{
			name: "api base url",
			path: "/sources/confluence/search",
			body: map[string]any{"connection_id": "cnf_web", "cloud_id": "cloud_1", "api_base_url": "https://evil.example/wiki"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			url := server.URL + tc.path
			if strings.HasPrefix(tc.path, "/sources/") {
				url = server.URL + "/api/missions/" + missionID + tc.path
			}
			status, failure := postJSONFailure(t, url, tc.body)
			message := nestedString(t, failure, "error", "message")
			if tc.name == "oauth start token url" {
				if status != http.StatusBadRequest || !strings.Contains(message, "API token") {
					t.Fatalf("expected OAuth disabled rejection, got %d %#v", status, failure)
				}
				return
			}
			if status != http.StatusBadRequest || !strings.Contains(message, "unknown field") {
				t.Fatalf("expected unknown field rejection, got %d %#v", status, failure)
			}
		})
	}
}

func TestConfluenceWebRejectsUnsafeAPITokenSiteURL(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	for _, siteURL := range []string{
		"http://docs.atlassian.net",
		"https://evil.example",
		"https://person:secret@docs.atlassian.net/wiki",
		"https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap",
	} {
		status, failure := postJSONFailure(t, server.URL+"/api/settings/connectors/confluence/connections", map[string]any{
			"display_name": "Unsafe",
			"auth_type":    app.ConfluenceAuthTypeAPIToken,
			"account_name": "person@example.com",
			"api_token":    "secret-api-token",
			"sites": []map[string]any{{
				"cloud_id": "cloud_1",
				"name":     "Unsafe",
				"url":      siteURL,
			}},
		})
		message := nestedString(t, failure, "error", "message")
		if status != http.StatusBadRequest || !strings.Contains(message, "site URL") {
			t.Fatalf("expected unsafe API token URL %q rejection, got %d %#v", siteURL, status, failure)
		}
		if strings.Contains(mustMarshalTestJSON(t, failure), "secret-api-token") {
			t.Fatalf("failure leaked API token: %#v", failure)
		}
		if strings.Contains(mustMarshalTestJSON(t, failure), "person:secret") {
			t.Fatalf("failure leaked URL credentials: %#v", failure)
		}
	}
}

func TestConfluenceWebRejectsManualOAuthConnection(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	status, failure := postJSONFailure(t, server.URL+"/api/settings/connectors/confluence/connections", map[string]any{
		"display_name": "Forged OAuth",
		"auth_type":    app.ConfluenceAuthTypeOAuth,
		"access_token": "secret-oauth-token",
		"scopes":       []string{"read:page:confluence"},
		"sites": []map[string]any{{
			"cloud_id": "cloud_forged",
			"url":      "https://docs.atlassian.net/wiki",
			"scopes":   []string{"read:page:confluence"},
		}},
	})
	message := nestedString(t, failure, "error", "message")
	if status != http.StatusBadRequest || !strings.Contains(message, "API token") {
		t.Fatalf("expected OAuth disabled rejection, got %d %#v", status, failure)
	}
	raw := mustMarshalTestJSON(t, failure)
	for _, leaked := range []string{"secret-oauth-token", "cloud_forged"} {
		if strings.Contains(raw, leaked) {
			t.Fatalf("failure leaked request value %q: %#v", leaked, failure)
		}
	}
}

func TestConfluenceWebRejectsAPITokenCloudIDMismatch(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	status, failure := postJSONFailure(t, server.URL+"/api/settings/connectors/confluence/connections", map[string]any{
		"display_name": "Mismatch",
		"auth_type":    app.ConfluenceAuthTypeAPIToken,
		"account_name": "person@example.com",
		"api_token":    "secret-api-token",
		"sites": []map[string]any{{
			"cloud_id": "cloud_1",
			"url":      "https://docs.atlassian.net/wiki/",
		}},
	})
	message := nestedString(t, failure, "error", "message")
	if status != http.StatusBadRequest || !strings.Contains(message, "cloud id must match the site URL") {
		t.Fatalf("expected API token cloud id mismatch rejection, got %d %#v", status, failure)
	}
	if strings.Contains(mustMarshalTestJSON(t, failure), "secret-api-token") {
		t.Fatalf("failure leaked API token: %#v", failure)
	}
}

func TestConfluenceWebDerivesAPITokenSiteCloudID(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	result := postJSON(t, server.URL+"/api/settings/connectors/confluence/connections", map[string]any{
		"display_name": "Docs",
		"auth_type":    app.ConfluenceAuthTypeAPIToken,
		"account_name": "person@example.com",
		"api_token":    "secret-api-token",
		"sites": []map[string]any{{
			"url": "https://docs.atlassian.net/wiki/",
		}},
	})
	raw := mustMarshalTestJSON(t, result)
	for _, expected := range []string{"site_docs.atlassian.net", "docs.atlassian.net", "https://docs.atlassian.net/wiki"} {
		if !strings.Contains(raw, expected) {
			t.Fatalf("expected API token site metadata %q in %#v", expected, result)
		}
	}
	if strings.Contains(raw, "secret-api-token") {
		t.Fatalf("response leaked API token: %#v", result)
	}
}

func TestConfluenceUpdateConnectionCloudIDForSnapshotRestrictsCrossAuthMapping(t *testing.T) {
	apiTokenConnection := app.ConfluenceConnection{
		AuthType: app.ConfluenceAuthTypeAPIToken,
		Sites: []app.ConfluenceSite{{
			CloudID: "site_docs.atlassian.net",
			URL:     "https://docs.atlassian.net/wiki",
		}},
	}
	got, err := webConfluenceConnectionCloudIDForSnapshot(apiTokenConnection, webConfluenceSnapshotSiteIdentity{
		CloudID: "cloud_1",
		SiteURL: "https://docs.atlassian.net/wiki",
	})
	if err != nil || got != "site_docs.atlassian.net" {
		t.Fatalf("expected API-token transport cloud id site_docs.atlassian.net, got %q err=%v", got, err)
	}

	oauthConnection := app.ConfluenceConnection{
		AuthType: app.ConfluenceAuthTypeOAuth,
		Sites: []app.ConfluenceSite{{
			CloudID: "cloud_1",
			URL:     "https://docs.atlassian.net/wiki",
			Scopes:  []string{"read:page:confluence"},
		}},
	}
	got, err = webConfluenceConnectionCloudIDForSnapshot(oauthConnection, webConfluenceSnapshotSiteIdentity{
		CloudID: "site_docs.atlassian.net",
		SiteURL: "https://docs.atlassian.net/wiki",
	})
	if err != nil || got != "cloud_1" {
		t.Fatalf("expected OAuth transport cloud id cloud_1, got %q err=%v", got, err)
	}

	unverifiedOAuthConnection := app.ConfluenceConnection{
		AuthType: app.ConfluenceAuthTypeOAuth,
		Sites: []app.ConfluenceSite{{
			CloudID: "cloud_unverified",
			URL:     "https://docs.atlassian.net/wiki",
		}},
	}
	_, err = webConfluenceConnectionCloudIDForSnapshot(unverifiedOAuthConnection, webConfluenceSnapshotSiteIdentity{
		CloudID: "site_docs.atlassian.net",
		SiteURL: "https://docs.atlassian.net/wiki",
	})
	if err == nil || !strings.Contains(err.Error(), "selected connection") {
		t.Fatalf("expected unverified OAuth same-host site error, got %v", err)
	}

	_, err = webConfluenceConnectionCloudIDForSnapshot(oauthConnection, webConfluenceSnapshotSiteIdentity{
		CloudID: "cloud_1",
		SiteURL: "https://docs.atlassian.net/wiki",
	})
	if err != nil {
		t.Fatalf("expected exact OAuth cloud id to remain valid, got %v", err)
	}

	_, err = webConfluenceConnectionCloudIDForSnapshot(app.ConfluenceConnection{
		AuthType: app.ConfluenceAuthTypeOAuth,
		Sites: []app.ConfluenceSite{{
			CloudID: "cloud_2",
			URL:     "https://docs.atlassian.net/wiki",
			Scopes:  []string{"read:page:confluence"},
		}},
	}, webConfluenceSnapshotSiteIdentity{
		CloudID: "cloud_1",
		SiteURL: "https://docs.atlassian.net/wiki",
	})
	if err == nil || !strings.Contains(err.Error(), "selected connection") {
		t.Fatalf("expected same-host different-cloud OAuth site error, got %v", err)
	}

	_, err = webConfluenceConnectionCloudIDForSnapshot(oauthConnection, webConfluenceSnapshotSiteIdentity{
		CloudID: "site_other.atlassian.net",
		SiteURL: "https://other.atlassian.net/wiki",
	})
	if err == nil || !strings.Contains(err.Error(), "selected connection") {
		t.Fatalf("expected mismatched connection site error, got %v", err)
	}
}

func TestConfluenceSnapshotSiteIdentityRecoversLegacySiteURLFromArtifact(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: "mis_confluence_legacy", Title: "Legacy Confluence"}); err != nil {
		t.Fatal(err)
	}
	artifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_confluence_legacy",
		MissionID:  "mis_confluence_legacy",
		MediaType:  app.ConfluenceSnapshotMediaType,
		Producer:   app.Producer{Type: "test", ID: "test"},
		Content: []byte(`{
			"schema_version":"plasma.confluence.snapshot.v1",
			"page":{"cloud_id":"cloud_1","site_url":"https://docs.atlassian.net/wiki","page_id":"123"}
		}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := svc.CreateSourceSnapshot(ctx, app.CreateSourceSnapshotRequest{
		SnapshotID: "src_confluence_legacy",
		MissionID:  "mis_confluence_legacy",
		Connector: app.ConnectorRef{
			ConnectorID:      app.ConfluenceConnectorID,
			ConnectorType:    app.ConfluenceConnectorType,
			ExternalSourceID: app.ConfluenceExternalSourceID("cloud_1", "123"),
			ExternalURI:      app.ConfluenceExternalURI("cloud_1", "123"),
		},
		Title:       "Legacy Confluence",
		ArtifactIDs: []string{artifact.ArtifactID},
		Locators:    json.RawMessage(`[{"cloud_id":"cloud_1","page_id":"123"}]`),
		Access:      app.SourceAccess{RetrievalPolicy: app.SourceRetrievalPolicySnapshotOnly},
	})
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(svc, Options{}).(*Server)
	identity, err := server.confluenceSnapshotSiteIdentity(ctx, "mis_confluence_legacy", snapshot.SnapshotID)
	if err != nil {
		t.Fatal(err)
	}
	if identity.CloudID != "cloud_1" || identity.SiteURL != "https://docs.atlassian.net/wiki" {
		t.Fatalf("expected legacy site identity from artifact payload, got %#v", identity)
	}
	_, err = server.confluenceSnapshotSiteIdentity(ctx, "mis_other", snapshot.SnapshotID)
	if err == nil || !strings.Contains(err.Error(), "another mission") {
		t.Fatalf("expected cross-mission snapshot identity rejection, got %v", err)
	}
}

func TestConfluenceSnapshotSiteIdentityRecoversLegacySyntheticSiteURL(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: "mis_confluence_synthetic", Title: "Synthetic Confluence"}); err != nil {
		t.Fatal(err)
	}
	artifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_confluence_synthetic",
		MissionID:  "mis_confluence_synthetic",
		MediaType:  "text/plain",
		Producer:   app.Producer{Type: "test", ID: "test"},
		Content:    []byte("legacy confluence snapshot without site url"),
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := svc.CreateSourceSnapshot(ctx, app.CreateSourceSnapshotRequest{
		SnapshotID: "src_confluence_synthetic",
		MissionID:  "mis_confluence_synthetic",
		Connector: app.ConnectorRef{
			ConnectorID:      app.ConfluenceConnectorID,
			ConnectorType:    app.ConfluenceConnectorType,
			ExternalSourceID: app.ConfluenceExternalSourceID("site_docs.atlassian.net", "123"),
			ExternalURI:      app.ConfluenceExternalURI("site_docs.atlassian.net", "123"),
		},
		Title:       "Synthetic Confluence",
		ArtifactIDs: []string{artifact.ArtifactID},
		Locators:    json.RawMessage(`[{"cloud_id":"site_docs.atlassian.net","page_id":"123"}]`),
		Access:      app.SourceAccess{RetrievalPolicy: app.SourceRetrievalPolicySnapshotOnly},
	})
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(svc, Options{}).(*Server)
	identity, err := server.confluenceSnapshotSiteIdentity(ctx, "mis_confluence_synthetic", snapshot.SnapshotID)
	if err != nil {
		t.Fatal(err)
	}
	if identity.CloudID != "site_docs.atlassian.net" || identity.SiteURL != "https://docs.atlassian.net/wiki" {
		t.Fatalf("expected synthetic legacy site URL from cloud id, got %#v", identity)
	}
}

func TestConfluenceIdentityMappingConnectorPreservesSnapshotIdentity(t *testing.T) {
	delegate := &recordingConfluenceSourceConnector{
		cloudID: "site_docs.atlassian.net",
		siteURL: "https://docs.atlassian.net/wiki",
	}
	connector := &webConfluenceIdentityMappingConnector{
		delegate:          delegate,
		snapshotCloudID:   "cloud_1",
		snapshotSiteURL:   "https://docs.atlassian.net/wiki",
		connectionCloudID: "site_docs.atlassian.net",
	}
	ctx := context.Background()

	version, err := connector.GetConfluenceSourceVersion(ctx, app.ConfluenceSourceReadRequest{
		CloudID: "cloud_1",
		PageID:  "123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if delegate.versionReq.CloudID != "site_docs.atlassian.net" {
		t.Fatalf("expected delegate version request to use connection cloud id, got %#v", delegate.versionReq)
	}
	if version.CloudID != "cloud_1" ||
		version.Connector.ExternalSourceID != app.ConfluenceExternalSourceID("cloud_1", "123") ||
		version.Connector.ExternalURI != app.ConfluenceExternalURI("cloud_1", "123") {
		t.Fatalf("expected version response mapped to snapshot identity, got %#v", version)
	}

	page, err := connector.ReadConfluenceSource(ctx, app.ConfluenceSourceReadRequest{
		CloudID: "cloud_1",
		PageID:  "123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if delegate.readReq.CloudID != "site_docs.atlassian.net" {
		t.Fatalf("expected delegate read request to use connection cloud id, got %#v", delegate.readReq)
	}
	if page.CloudID != "cloud_1" ||
		page.Connector.ExternalSourceID != app.ConfluenceExternalSourceID("cloud_1", "123") ||
		page.Connector.ExternalURI != app.ConfluenceExternalURI("cloud_1", "123") {
		t.Fatalf("expected page response mapped to snapshot identity, got %#v", page)
	}
	var metadata map[string]string
	if err := json.Unmarshal(page.Metadata, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata["cloud_id"] != "cloud_1" || metadata["other"] != "ok" {
		t.Fatalf("expected page metadata cloud id mapped to snapshot identity, got %#v", metadata)
	}

	search, err := connector.SearchConfluenceSources(ctx, app.ConfluenceSourceSearchRequest{
		CloudID: "cloud_1",
		Query:   "roadmap",
	})
	if err != nil {
		t.Fatal(err)
	}
	if delegate.searchReq.CloudID != "site_docs.atlassian.net" {
		t.Fatalf("expected delegate search request to use connection cloud id, got %#v", delegate.searchReq)
	}
	candidate := search.Candidates[0]
	if search.CloudID != "cloud_1" ||
		candidate.CloudID != "cloud_1" ||
		candidate.Connector.ExternalSourceID != app.ConfluenceExternalSourceID("cloud_1", "123") {
		t.Fatalf("expected search response mapped to snapshot identity, got %#v", search)
	}
}

func TestConfluenceIdentityMappingConnectorRejectsMismatchedResponseSite(t *testing.T) {
	delegate := &recordingConfluenceSourceConnector{
		cloudID: "site_other.atlassian.net",
		siteURL: "https://other.atlassian.net/wiki",
	}
	connector := &webConfluenceIdentityMappingConnector{
		delegate:          delegate,
		snapshotCloudID:   "cloud_1",
		snapshotSiteURL:   "https://docs.atlassian.net/wiki",
		connectionCloudID: "site_other.atlassian.net",
	}
	_, err := connector.GetConfluenceSourceVersion(context.Background(), app.ConfluenceSourceReadRequest{
		CloudID: "cloud_1",
		PageID:  "123",
	})
	if err == nil || !strings.Contains(err.Error(), "snapshot site") {
		t.Fatalf("expected response site mismatch error, got %v", err)
	}
}

type recordingConfluenceSourceConnector struct {
	cloudID    string
	siteURL    string
	searchReq  app.ConfluenceSourceSearchRequest
	readReq    app.ConfluenceSourceReadRequest
	versionReq app.ConfluenceSourceReadRequest
}

func (connector *recordingConfluenceSourceConnector) SearchConfluenceSources(_ context.Context, req app.ConfluenceSourceSearchRequest) (app.ConfluenceSourceSearchResult, error) {
	connector.searchReq = req
	cloudID := connector.responseCloudID()
	return app.ConfluenceSourceSearchResult{
		CloudID: cloudID,
		Candidates: []app.ConfluenceSourceCandidate{{
			Connector: app.ConnectorRef{
				ConnectorID:      app.ConfluenceConnectorID,
				ExternalSourceID: app.ConfluenceExternalSourceID(cloudID, "123"),
				ExternalURI:      app.ConfluenceExternalURI(cloudID, "123"),
			},
			CloudID: cloudID,
			Title:   "Roadmap",
		}},
	}, nil
}

func (connector *recordingConfluenceSourceConnector) ReadConfluenceSource(_ context.Context, req app.ConfluenceSourceReadRequest) (app.ConfluenceSourcePage, error) {
	connector.readReq = req
	cloudID := connector.responseCloudID()
	return app.ConfluenceSourcePage{
		Connector: app.ConnectorRef{
			ConnectorID:      app.ConfluenceConnectorID,
			ExternalSourceID: app.ConfluenceExternalSourceID(cloudID, "123"),
			ExternalURI:      app.ConfluenceExternalURI(cloudID, "123"),
		},
		CloudID:  cloudID,
		SiteURL:  connector.siteURL,
		PageID:   "123",
		Title:    "Roadmap",
		Metadata: json.RawMessage(`{"cloud_id":"` + cloudID + `","other":"ok"}`),
	}, nil
}

func (connector *recordingConfluenceSourceConnector) GetConfluenceSourceVersion(_ context.Context, req app.ConfluenceSourceReadRequest) (app.ConfluenceSourceVersion, error) {
	connector.versionReq = req
	cloudID := connector.responseCloudID()
	return app.ConfluenceSourceVersion{
		Connector: app.ConnectorRef{
			ConnectorID:      app.ConfluenceConnectorID,
			ExternalSourceID: app.ConfluenceExternalSourceID(cloudID, "123"),
			ExternalURI:      app.ConfluenceExternalURI(cloudID, "123"),
		},
		CloudID: cloudID,
		SiteURL: connector.siteURL,
		PageID:  "123",
		Title:   "Roadmap",
	}, nil
}

func (connector *recordingConfluenceSourceConnector) responseCloudID() string {
	if strings.TrimSpace(connector.cloudID) != "" {
		return strings.TrimSpace(connector.cloudID)
	}
	return "cloud_1"
}

func TestConfluenceClientRejectsStoredUnsafeAPITokenSiteURLWithAPIBaseOverride(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.UpsertConfluenceConnection(ctx, app.ConfluenceConnection{
		ConnectionID: "cnf_unsafe",
		DisplayName:  "Unsafe",
		AuthType:     app.ConfluenceAuthTypeAPIToken,
		AccountName:  "person@example.com",
		AccessToken:  "secret-api-token",
		Sites: []app.ConfluenceSite{{
			CloudID: "cloud_1",
			Name:    "Unsafe",
			URL:     "https://person:secret@docs.atlassian.net/wiki",
		}},
	}); err != nil {
		t.Fatal(err)
	}
	server := NewServer(app.NewService(store), Options{
		ConfluenceAPIBaseURL: "https://api-proxy.example/wiki",
	}).(*Server)
	_, err = server.confluenceClient(ctx, "cnf_unsafe", "cloud_1")
	if err == nil || !strings.Contains(err.Error(), "site URL") {
		t.Fatalf("expected stored unsafe API token site URL rejection, got %v", err)
	}
	if strings.Contains(err.Error(), "secret-api-token") {
		t.Fatalf("failure leaked API token: %v", err)
	}
	if strings.Contains(err.Error(), "person:secret") {
		t.Fatalf("failure leaked URL credentials: %v", err)
	}
}

func TestConfluenceClientRejectsUnsafeAPITokenAPIBaseURLWithSafeSite(t *testing.T) {
	ctx := context.Background()
	for _, apiBaseURL := range []string{
		"https://attacker.example/wiki",
		"http://docs.atlassian.net/wiki",
		"https://other.atlassian.net/wiki",
	} {
		t.Run(apiBaseURL, func(t *testing.T) {
			store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			if err := store.UpsertConfluenceConnection(ctx, app.ConfluenceConnection{
				ConnectionID: "cnf_api_base",
				DisplayName:  "Docs API",
				AuthType:     app.ConfluenceAuthTypeAPIToken,
				AccountName:  "person@example.com",
				AccessToken:  "secret-api-token",
				Sites: []app.ConfluenceSite{{
					CloudID: "cloud_1",
					Name:    "Docs",
					URL:     "https://docs.atlassian.net/wiki",
				}},
			}); err != nil {
				t.Fatal(err)
			}
			server := NewServer(app.NewService(store), Options{
				ConfluenceAPIBaseURL: apiBaseURL,
			}).(*Server)
			_, err = server.confluenceClient(ctx, "cnf_api_base", "cloud_1")
			if err == nil || !strings.Contains(err.Error(), "API base URL") {
				t.Fatalf("expected unsafe API base URL rejection, got %v", err)
			}
			if strings.Contains(err.Error(), "secret-api-token") {
				t.Fatalf("failure leaked API token: %v", err)
			}
		})
	}
}

func TestConfluenceWebRejectsOAuthConnectionBeforeUse(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	_, err = svc.UpsertConfluenceConnection(ctx, app.UpsertConfluenceConnectionRequest{
		ConnectionID:   "cnf_refresh",
		DisplayName:    "Docs",
		AuthType:       app.ConfluenceAuthTypeOAuth,
		AccessToken:    "expired-oauth-token",
		RefreshToken:   "refresh-secret",
		TokenExpiresAt: time.Now().UTC().Add(-time.Minute),
		Sites: []app.ConfluenceSite{{
			CloudID: "cloud_1",
			Name:    "Docs",
			URL:     "https://docs.atlassian.net/wiki",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(NewServer(svc, Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Confluence refresh"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	status, failure := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/search", map[string]any{
		"connection_id": "cnf_refresh",
		"cloud_id":      "cloud_1",
		"query":         "roadmap",
	})
	if status != http.StatusBadRequest || !strings.Contains(nestedString(t, failure, "error", "message"), "API token") {
		t.Fatalf("expected OAuth disabled rejection, got %d %#v", status, failure)
	}
}

func TestReportArtifactHTMLExportInlinesImageMediaSources(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	handler := NewServer(svc, Options{})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "HTML report export"})
	missionID := nestedString(t, mission, "projection", "mission_id")

	reportArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_html_report_md",
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   "odyssey-report.md",
		Producer:   app.Producer{Type: "agent", ID: "codex"},
		Content:    []byte("# 오디세이 리포트\n\n본문 \\(E=mc^2\\) 입니다.\n\n| 항목 | 값 |\n| --- | --- |\n| 수식 | \\(x+1\\) |\n\n```mermaid\nflowchart TD\n  A[시작] --> B[검토]\n```\n\n\\[x^2+y^2=z^2\\]\n\n잘못된 \\(\\notacommand{\\) 수식입니다.\n\n`\\(code\\)`와 $dollar$는 그대로입니다.\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := appendTestEvent(t, webServer, ctx, missionID, "report.artifact.created", map[string]any{
		"kind":        "markdown_report_artifact",
		"artifact_id": reportArtifact.ArtifactID,
		"media_type":  reportArtifact.MediaType,
	}, app.Producer{Type: "agent", ID: "codex"}); err != nil {
		t.Fatal(err)
	}

	locators, err := json.Marshal([]app.MediaLocator{{
		LocatorType:    app.SourceLocatorTypeMedia,
		MediaKind:      app.MediaKindImage,
		Provider:       "media_url",
		CanonicalURL:   "https://example.com/odyssey.png",
		DirectMediaURL: "https://example.com/odyssey.png",
		MIMEType:       "image/png",
		ByteSize:       int64(len("fake-png-bytes")),
		Width:          640,
		Height:         480,
		Title:          "Odyssey still",
		Attribution:    "Example archive",
		License:        "CC-BY",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateSourceSnapshotWithEvent(ctx, app.CreateSourceSnapshotWithEventRequest{
		Artifact: app.CreateRawArtifactRequest{
			ArtifactID: "art_html_report_image",
			MissionID:  missionID,
			MediaType:  "image/png",
			Filename:   "odyssey.png",
			Producer:   app.Producer{Type: "user", ID: "plasma-ui"},
			Content:    []byte("fake-png-bytes"),
		},
		Snapshot: app.CreateSourceSnapshotRequest{
			SnapshotID: "src_html_report_image",
			MissionID:  missionID,
			Connector: app.ConnectorRef{
				ConnectorID:      "media_url",
				ConnectorType:    app.SourceConnectorTypeMediaURL,
				ExternalSourceID: "https://example.com/odyssey.png",
				ExternalURI:      "https://example.com/odyssey.png",
			},
			Title:    "Odyssey still",
			Locators: locators,
			Access: app.SourceAccess{
				License:         "CC-BY",
				RetrievalPolicy: app.SourceRetrievalPolicySnapshotOnly,
			},
		},
		Event: app.AppendEventRequest{
			EventID:   "evt_html_report_image_source",
			MissionID: missionID,
			EventType: "source.snapshotted",
			Producer:  app.Producer{Type: "user", ID: "plasma-ui"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	legacyArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_legacy_html_export",
		MissionID:  missionID,
		MediaType:  "text/html; charset=utf-8",
		Filename:   "legacy.html",
		Producer:   app.Producer{Type: "plasma", ID: "html-export"},
		Content:    []byte("<!doctype html><html><body>legacy</body></html>"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := appendTestEvent(t, webServer, ctx, missionID, "report.artifact.exported", map[string]any{
		"kind": reporting.ExportKindSelfContainedHTML, "source_artifact_id": reportArtifact.ArtifactID,
		"artifact_id": legacyArtifact.ArtifactID, "target": reporting.ExportTargetSelfContainedHTML,
	}, app.Producer{Type: "plasma", ID: "html-export"}); err != nil {
		t.Fatal(err)
	}
	uploadLocators, err := json.Marshal([]app.UploadedFileLocator{{
		LocatorType:       app.SourceLocatorTypeMedia,
		MediaKind:         app.MediaKindImage,
		OriginalFilename:  "uploaded-still.png",
		SanitizedFilename: "uploaded-still.png",
		MIMEType:          "image/png",
		ByteSize:          int64(len("uploaded-png-bytes")),
		SHA256:            "uploaded-image-sha",
		ContentKind:       app.UploadedContentKindImage,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateSourceSnapshotWithEvent(ctx, app.CreateSourceSnapshotWithEventRequest{
		Artifact: app.CreateRawArtifactRequest{
			ArtifactID: "art_html_report_uploaded_image",
			MissionID:  missionID,
			MediaType:  "image/png",
			Filename:   "uploaded-still.png",
			Producer:   app.Producer{Type: "user", ID: "plasma-ui"},
			Content:    []byte("uploaded-png-bytes"),
		},
		Snapshot: app.CreateSourceSnapshotRequest{
			SnapshotID: "src_html_report_uploaded_image",
			MissionID:  missionID,
			Connector: app.ConnectorRef{
				ConnectorID:      "file_upload",
				ConnectorType:    app.SourceConnectorTypeFileUpload,
				ExternalSourceID: "file_upload:uploaded-image-sha",
				ExternalURI:      "file-upload://uploaded-image-sha",
			},
			Title:    "Uploaded still",
			Locators: uploadLocators,
			Access: app.SourceAccess{
				RetrievalPolicy: app.SourceRetrievalPolicySnapshotOnly,
			},
		},
		Event: app.AppendEventRequest{
			EventID:   "evt_html_report_uploaded_image_source",
			MissionID: missionID,
			EventType: "source.snapshotted",
			Producer:  app.Producer{Type: "user", ID: "plasma-ui"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	export := postJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+reportArtifact.ArtifactID+"/html_export", map[string]any{})
	exportArtifactID := nestedString(t, export, "artifact", "artifact_id")
	if exportArtifactID == legacyArtifact.ArtifactID {
		t.Fatal("versionless legacy HTML export was reused")
	}
	content, _ := export["content"].(string)
	for _, expected := range []string{
		"<!doctype html>",
		"오디세이 리포트",
		"data:image/png;base64,ZmFrZS1wbmctYnl0ZXM=",
		"Odyssey still",
		"data:image/png;base64,dXBsb2FkZWQtcG5nLWJ5dGVz",
		"Uploaded still",
		"이 HTML은 보고서 내용을 다시 생성하지 않고 저장된 Markdown artifact를 렌더링했습니다.",
		`<pre class="report-markdown-raw">`,
		`id="report-markdown" type="application/json"`,
		"data:font/woff2;base64,",
		`version:"0.17.0"`,
		"Mermaid 그래프",
		"securityLevel",
		"renderPlasmaMermaid(root)",
		"renderPlasmaMarkdown(target,JSON.parse(source.textContent))",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected HTML export to contain %q:\n%s", expected, content)
		}
	}
	previewURL := nestedString(t, export, "preview_url")
	expectedPreviewURL := "/api/missions/" + missionID + "/artifacts/" + exportArtifactID + "/preview"
	if previewURL != expectedPreviewURL {
		t.Fatalf("expected preview URL %q, got %q", expectedPreviewURL, previewURL)
	}
	previewResp, err := http.Get(server.URL + previewURL)
	if err != nil {
		t.Fatal(err)
	}
	defer previewResp.Body.Close()
	if previewResp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTML preview 200, got %d", previewResp.StatusCode)
	}
	if got := previewResp.Header.Get("Content-Disposition"); !strings.HasPrefix(got, "inline") {
		t.Fatalf("expected inline HTML preview disposition, got %q", got)
	}
	if got := previewResp.Header.Get("Content-Security-Policy"); !strings.Contains(got, "sandbox allow-scripts") || strings.Contains(got, "allow-same-origin") {
		t.Fatalf("expected sandboxed HTML preview CSP without same-origin, got %q", got)
	}
	previewBody, err := io.ReadAll(previewResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(previewBody), "<!doctype html>") || !strings.Contains(string(previewBody), "renderPlasmaMarkdown") {
		t.Fatalf("expected rendered HTML artifact preview body, got %q", string(previewBody))
	}
	status, failure := getJSONFailure(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+reportArtifact.ArtifactID+"/preview")
	if status != http.StatusUnsupportedMediaType || !strings.Contains(nestedString(t, failure, "error", "message"), "previewable as HTML") {
		t.Fatalf("expected markdown artifact preview rejection, got %d %#v", status, failure)
	}
	metadataOnly := postJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+reportArtifact.ArtifactID+"/html_export", map[string]any{"include_content": false})
	if got := nestedString(t, metadataOnly, "artifact", "artifact_id"); got != exportArtifactID {
		t.Fatalf("expected cached metadata-only HTML artifact %q, got %q", exportArtifactID, got)
	}
	if _, ok := metadataOnly["content"]; ok {
		t.Fatalf("metadata-only HTML export should not include content")
	}
	emptyReq, err := http.NewRequest(http.MethodPost, server.URL+"/api/missions/"+missionID+"/artifacts/"+reportArtifact.ArtifactID+"/html_export", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	emptyReq.Header.Set("Content-Type", "application/json")
	emptyResp, err := http.DefaultClient.Do(emptyReq)
	if err != nil {
		t.Fatal(err)
	}
	defer emptyResp.Body.Close()
	if emptyResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(emptyResp.Body)
		t.Fatalf("expected empty-body HTML export to remain compatible, got %d %s", emptyResp.StatusCode, body)
	}
	var emptyPost map[string]any
	if err := json.NewDecoder(emptyResp.Body).Decode(&emptyPost); err != nil {
		t.Fatal(err)
	}
	if got := nestedString(t, emptyPost, "artifact", "artifact_id"); got != exportArtifactID {
		t.Fatalf("expected empty-body HTML export to reuse %q, got %q", exportArtifactID, got)
	}
	if !strings.Contains(content, `<pre class="report-markdown-raw"># 오디세이 리포트`) || !strings.Contains(content, "$dollar$") {
		t.Fatalf("basic export parsed math on the backend: %s", content)
	}
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	var currentVersionFound bool
	for _, event := range events {
		var payload map[string]any
		_ = json.Unmarshal(event.Payload, &payload)
		if payload["artifact_id"] == exportArtifactID && payload["renderer_version"] == selfContainedReportRendererVersion {
			currentVersionFound = true
		}
	}
	if !currentVersionFound {
		t.Fatal("current basic renderer version was not recorded")
	}
	sourceAfter, err := svc.GetRawArtifact(ctx, reportArtifact.ArtifactID)
	if err != nil || !bytes.Equal(sourceAfter.Content, reportArtifact.Content) {
		t.Fatalf("source Markdown changed during export: %v", err)
	}

	second := postJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+reportArtifact.ArtifactID+"/html_export", map[string]any{})
	if got := nestedString(t, second, "artifact", "artifact_id"); got != exportArtifactID {
		t.Fatalf("expected cached HTML artifact %q, got %q", exportArtifactID, got)
	}
	htmlRead := getJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+exportArtifactID)
	if got := nestedString(t, htmlRead, "artifact", "media_type"); got != "text/html; charset=utf-8" {
		t.Fatalf("expected HTML artifact to be readable, got %#v", htmlRead)
	}
}

func TestReportArtifactDesignedHTMLExportCreatesCachedArtifact(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	contentModel := `{
	  "kicker": "Plasma Designed Report",
	  "title": "오디세이 리포트",
	  "subtitle": "소스와 조사 내용을 읽기 좋은 구조로 재배열한 리포트입니다.",
	  "thesis": "오디세이는 제작 방식, 음악 선택, 고전 원전의 현대적 해석이 서로 맞물린 프로젝트입니다.",
	  "markers": [
	    {"label": "관점", "value": "3", "note": "제작, 음악, 해석을 나누어 읽습니다."}
	  ],
	  "hero_visual": {
	    "title": "핵심 관계",
	    "left_label": "조사",
	    "right_label": "리포트",
	    "nodes": [
	      {"label": "원전", "body": "호메로스 서사시가 비교 기준을 제공합니다.", "tone": "accent"},
	      {"label": "제작", "body": "현대 영화 제작 조건이 해석의 폭을 만듭니다.", "tone": "neutral"}
	    ]
	  },
	  "visual_units": [
	    {"title": "읽는 순서", "kind": "flow", "question": "어떤 순서로 읽어야 하는가", "nodes": [
	      {"label": "맥락", "body": "원전과 제작 맥락을 먼저 잡습니다.", "tone": "accent"},
	      {"label": "쟁점", "body": "음악과 연출 선택을 비교합니다.", "tone": "warn"},
	      {"label": "판단", "body": "확인된 내용과 추정을 분리합니다.", "tone": "good"}
	    ], "caption": "세 단계로 읽으면 보고서의 중심축이 보입니다."}
	  ],
	  "tabs": [
	    {"label": "개요", "question": "무엇을 봐야 하는가", "summary": "오디세이 리포트의 주요 관점을 요약합니다.", "takeaway": "사실과 해석을 분리해 읽어야 합니다.", "sections": [
	      {"heading": "보고서의 중심", "body": ["첫 문단은 작품을 둘러싼 핵심 맥락과 \\(E=mc^2\\)를 설명합니다.", "| 구분 | 의미 |\\n| --- | --- |\\n| 원전 | 비교 기준 |\\n| 영화 | 현대적 해석 |", "\u0060\u0060\u0060mermaid\\nflowchart TD\\n  A[원전] --> B[영화]\\n\u0060\u0060\u0060"], "bullets": ["원전과 영화의 차이를 구분합니다.", "음악 선택은 별도 쟁점으로 봅니다."], "component": "analysis", "table": {"columns": ["관계", "식"], "rows": [["피타고라스", "\\[x^2+y^2=z^2\\]"]]}, "images": [{"image_ref": "image_1", "caption": "오디세이 시각 자료를 제작 맥락 옆에 둡니다.", "placement": "after_body"}, {"image_ref": "image_1", "caption": "중복 배치는 한 번만 렌더링되어야 합니다.", "placement": "after_body"}], "source_note": "원본 Markdown 리포트 기반"}
	    ]}
	  ],
	  "sources": [{"label": "원본 Markdown", "note": "저장된 리포트 artifact"}],
	  "caveats": ["이 HTML은 원본 리포트의 범위를 넘지 않습니다."],
	  "glossary": [{"term": "artifact", "definition": "Plasma에 저장된 산출물입니다."}]
	}`
	contentModelWithSecondImage := strings.Replace(contentModel, `"title": "오디세이 리포트"`, `"title": "오디세이 리포트 확장"`, 1)
	agent := &fakeAgentExecutor{rejectDeadline: true, responses: []AgentResult{
		{Text: contentModel, SessionID: "agent-session-designed"},
		{Text: contentModelWithSecondImage, SessionID: "agent-session-designed-2"},
	}}
	svc := app.NewService(store)
	handler := NewServer(svc, Options{AgentExecutor: agent})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Designed HTML report export"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	unsafeImageDataURI := "data:image/png;base64," + strings.Repeat("A", 320)
	unsafeWrappedBase64Payload := strings.Repeat("B", 76) + "\n" + strings.Repeat("C", 76)
	unsafeWrappedImageDataURI := "data:image/png;base64," + unsafeWrappedBase64Payload
	unsafeParameterizedImageDataURI := "data:image/png;charset=utf-8;base64," + strings.Repeat("D", 320)
	unsafeCaseImageDataURI := "DATA:IMAGE/PNG;BASE64," + strings.Repeat("E", 320)
	unsafeBase64Payload := strings.Repeat("QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVo", 8)
	reportArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_designed_report_md",
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   "odyssey-report.md",
		Producer:   app.Producer{Type: "agent", ID: "codex"},
		Content: []byte("# 오디세이 리포트\n\n소스 기반 장문 리포트입니다.\n\n" +
			"상대성 식은 \\(E=mc^2\\)이고 표의 관계는 \\[x^2+y^2=z^2\\]입니다.\n\n" +
			"![inline payload](" + unsafeImageDataURI + ")\n\n" +
			"![wrapped payload](" + unsafeWrappedImageDataURI + ")\n\n" +
			"![parameterized payload](" + unsafeParameterizedImageDataURI + ")\n\n" +
			"![case payload](" + unsafeCaseImageDataURI + ")\n\n" +
			"payload: " + unsafeBase64Payload + "\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	sourceContent := bytes.Clone(reportArtifact.Content)
	if _, err := appendTestEvent(t, webServer, ctx, missionID, "report.artifact.created", map[string]any{
		"kind":        "markdown_report_artifact",
		"artifact_id": reportArtifact.ArtifactID,
		"media_type":  reportArtifact.MediaType,
	}, app.Producer{Type: "agent", ID: "codex"}); err != nil {
		t.Fatal(err)
	}
	addDesignedReportImageSource(t, ctx, svc, missionID, "src_designed_report_image", "art_designed_report_image", "evt_designed_report_image_source", "Odyssey still", "https://example.com/odyssey.png", []byte("fake-designed-png-bytes"))

	start := postJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+reportArtifact.ArtifactID+"/designed_html_export", map[string]any{"agent_executor": "codex"})
	if got, _ := start["status"].(string); got != "pending" {
		t.Fatalf("expected pending designed HTML response, got %#v", start)
	}
	pendingPayload, ok := nestedValue(t, start, "pending_event", "Payload").(map[string]any)
	if !ok || pendingPayload["agent_model"] != "" || pendingPayload["agent_reasoning_effort"] != "" {
		t.Fatalf("designed HTML pending event must preserve raw defaults, got %#v", pendingPayload)
	}
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.exported")
	if len(agent.requests) != 1 || agent.requests[0].Model != "gpt-5.5" || agent.requests[0].ReasoningEffort != "medium" {
		t.Fatalf("designed HTML execution must resolve GPT-5.5/medium, got %#v", agent.requests)
	}
	for _, expected := range []string{
		"Put the strongest visual unit first",
		"visual_identity",
		"composition_shape",
		"connected SVG relationship map",
		"Available report_images",
		`"image_ref": "image_1"`,
		"Odyssey still",
		"Use only \\(...\\) for inline math and \\[...\\] for display math.",
		`\(E=mc^2\)`,
		`\[x^2+y^2=z^2\]`,
	} {
		if !strings.Contains(agent.requests[0].Prompt, expected) {
			t.Fatalf("expected designed HTML content model prompt to contain %q:\n%s", expected, agent.requests[0].Prompt)
		}
	}
	if strings.Contains(agent.requests[0].Prompt, "data:image") {
		t.Fatalf("designed HTML content model prompt must not include image bytes:\n%s", agent.requests[0].Prompt)
	}
	if strings.Contains(strings.ToLower(agent.requests[0].Prompt), "data:image") {
		t.Fatalf("designed HTML content model prompt must not include case-varied image data URIs:\n%s", agent.requests[0].Prompt)
	}
	for _, notExpected := range []string{
		unsafeImageDataURI,
		unsafeWrappedImageDataURI,
		unsafeWrappedBase64Payload,
		unsafeParameterizedImageDataURI,
		unsafeCaseImageDataURI,
		strings.Repeat("D", 320),
		strings.Repeat("E", 320),
		unsafeBase64Payload,
	} {
		if strings.Contains(agent.requests[0].Prompt, notExpected) {
			t.Fatalf("designed HTML content model prompt must redact unsafe markdown payload %q:\n%s", notExpected, agent.requests[0].Prompt)
		}
	}
	for _, expected := range []string{"[redacted inline image data URI]", "[redacted long base64-like payload]"} {
		if !strings.Contains(agent.requests[0].Prompt, expected) {
			t.Fatalf("expected designed HTML content model prompt to contain redaction marker %q:\n%s", expected, agent.requests[0].Prompt)
		}
	}
	exportPayload := latestEventPayload(t, detail, "report.artifact.exported", "designed_html_report_artifact")
	designedArtifactID, _ := exportPayload["artifact_id"].(string)
	modelArtifactID, _ := exportPayload["content_model_artifact_id"].(string)
	if designedArtifactID == "" || modelArtifactID == "" {
		t.Fatalf("expected designed and model artifact ids in payload: %#v", exportPayload)
	}
	if exportPayload["renderer_version"] != designedReportRendererVersion || exportPayload["content_model_contract"] != reporting.DesignedContentModelContract {
		t.Fatalf("expected current visual grammar renderer metadata, got %#v", exportPayload)
	}
	if _, err := svc.GetRawArtifact(ctx, designedArtifactID); err != nil {
		t.Fatalf("designed artifact %q was not stored: %v; payload=%#v", designedArtifactID, err, exportPayload)
	}
	if ok, err := webServer.isReportArtifact(ctx, missionID, designedArtifactID); err != nil || !ok {
		t.Fatalf("designed artifact %q was not recognized as report artifact, ok=%v err=%v payload=%#v", designedArtifactID, ok, err, exportPayload)
	}
	htmlRead := getJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+designedArtifactID)
	content := nestedString(t, htmlRead, "content")
	for _, expected := range []string{
		"<!doctype html>",
		"오디세이 리포트",
		"visual-map-hero",
		"hero-map-svg",
		"relationship-svg",
		"시각 단위",
		"보고서의 중심",
		"Plasma Designed Report",
		"inline-report-image",
		"data:image/png;base64,",
		"오디세이 시각 자료를 제작 맥락 옆에 둡니다.",
		"모든 이미지는 관련 본문 섹션 안에 배치되었습니다.",
		`\(E=mc^2\)`,
		`\[x^2+y^2=z^2\]`,
		`data-designed-markdown`,
		`renderPlasmaMarkdown(node,JSON.parse(source.textContent))`,
		`renderPlasmaMermaid(root)`,
		`Mermaid 그래프`,
		"renderDesignedTextMath(document.body)",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected designed HTML to contain %q:\n%s", expected, content)
		}
	}
	if got := strings.Count(content, `class="inline-report-image"`); got != 1 {
		t.Fatalf("expected duplicate image_ref to render once, got %d instances:\n%s", got, content)
	}
	if strings.Contains(content, "중복 배치는 한 번만 렌더링되어야 합니다.") {
		t.Fatalf("expected duplicate inline image placement to be skipped:\n%s", content)
	}
	modelArtifact, err := svc.GetRawArtifact(ctx, modelArtifactID)
	if err != nil {
		t.Fatalf("content model artifact %q was not stored: %v", modelArtifactID, err)
	}
	if got := modelArtifact.MediaType; got != "application/json; charset=utf-8" {
		t.Fatalf("expected content model artifact media type, got %q", got)
	}
	modelContent := string(modelArtifact.Content)
	for _, expected := range []string{`"visual_identity"`, `"composition_shape"`, `"style_key": "atlas"`, `"image_ref": "image_1"`, `\\(E=mc^2\\)`, `\\[x^2+y^2=z^2\\]`} {
		if !strings.Contains(modelContent, expected) {
			t.Fatalf("expected normalized content model to contain %q:\n%s", expected, modelContent)
		}
	}
	storedSource, err := svc.GetRawArtifact(ctx, reportArtifact.ArtifactID)
	if err != nil {
		t.Fatalf("source report artifact was not readable after designed export: %v", err)
	}
	if !bytes.Equal(storedSource.Content, sourceContent) {
		t.Fatal("designed export mutated the source Markdown artifact")
	}

	second := postJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+reportArtifact.ArtifactID+"/designed_html_export", map[string]any{"agent_executor": "codex"})
	if got, _ := second["status"].(string); got != "completed" {
		t.Fatalf("expected cached designed HTML response, got %#v", second)
	}
	if got := nestedString(t, second, "artifact", "artifact_id"); got != designedArtifactID {
		t.Fatalf("expected cached designed artifact %q, got %q", designedArtifactID, got)
	}
	addDesignedReportImageSource(t, ctx, svc, missionID, "src_designed_report_image_2", "art_designed_report_image_2", "evt_designed_report_image_source_2", "Odyssey map", "https://example.com/odyssey-map.png", []byte("fake-designed-map-png-bytes"))
	third := postJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+reportArtifact.ArtifactID+"/designed_html_export", map[string]any{"agent_executor": "codex"})
	if got, _ := third["status"].(string); got != "pending" {
		t.Fatalf("expected changed image set to miss cache and start new designed HTML response, got %#v", third)
	}
	detail = waitForEventTypeCount(t, server.URL, missionID, "report.artifact.exported", 2)
	exportPayload = latestEventPayload(t, detail, "report.artifact.exported", "designed_html_report_artifact")
	if exportPayload["artifact_id"] == designedArtifactID {
		t.Fatalf("expected changed image set to create a new designed artifact, got cached payload %#v", exportPayload)
	}
	if len(agent.requests) != 2 {
		t.Fatalf("expected changed image set to run agent again, got %d requests", len(agent.requests))
	}
	if !strings.Contains(agent.requests[1].Prompt, `"image_ref": "image_2"`) || !strings.Contains(agent.requests[1].Prompt, "Odyssey map") {
		t.Fatalf("expected second designed HTML prompt to include second image metadata:\n%s", agent.requests[1].Prompt)
	}
}

func addDesignedReportImageSource(t *testing.T, ctx context.Context, svc *app.Service, missionID string, snapshotID string, artifactID string, eventID string, title string, sourceURL string, content []byte) {
	t.Helper()
	addDesignedReportMediaSource(t, ctx, svc, missionID, snapshotID, artifactID, eventID, title, sourceURL, "image/png", content)
}

func addDesignedReportMediaSource(t *testing.T, ctx context.Context, svc *app.Service, missionID string, snapshotID string, artifactID string, eventID string, title string, sourceURL string, mediaType string, content []byte) {
	t.Helper()
	imageLocators, err := json.Marshal([]app.MediaLocator{{
		LocatorType:    app.SourceLocatorTypeMedia,
		MediaKind:      app.MediaKindImage,
		Provider:       "media_url",
		CanonicalURL:   sourceURL,
		DirectMediaURL: sourceURL,
		MIMEType:       mediaType,
		ByteSize:       int64(len(content)),
		Width:          640,
		Height:         480,
		Title:          title,
		Attribution:    "Example archive",
		License:        "CC-BY",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateSourceSnapshotWithEvent(ctx, app.CreateSourceSnapshotWithEventRequest{
		Artifact: app.CreateRawArtifactRequest{
			ArtifactID: artifactID,
			MissionID:  missionID,
			MediaType:  mediaType,
			Filename:   safeFilename(title, ".png"),
			Producer:   app.Producer{Type: "user", ID: "plasma-ui"},
			Content:    content,
		},
		Snapshot: app.CreateSourceSnapshotRequest{
			SnapshotID: snapshotID,
			MissionID:  missionID,
			Connector: app.ConnectorRef{
				ConnectorID:      "media_url",
				ConnectorType:    app.SourceConnectorTypeMediaURL,
				ExternalSourceID: sourceURL,
				ExternalURI:      sourceURL,
			},
			Title:    title,
			Locators: imageLocators,
			Access: app.SourceAccess{
				License:         "CC-BY",
				RetrievalPolicy: app.SourceRetrievalPolicySnapshotOnly,
			},
		},
		Event: app.AppendEventRequest{
			EventID:   eventID,
			MissionID: missionID,
			EventType: "source.snapshotted",
			Producer:  app.Producer{Type: "user", ID: "plasma-ui"},
		},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestReportArtifactDesignedHTMLExportSkipsUnsupportedImageArtifact(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	contentModel := `{
	  "title": "SVG 제외 리포트",
	  "tabs": [{"label": "검증", "sections": [{"heading": "이미지 검증", "body": ["비허용 이미지는 본문에 들어가지 않아야 합니다."], "images": [{"image_ref": "image_1", "caption": "이 이미지는 없어야 합니다.", "placement": "after_body"}]}]}]
	}`
	agent := &fakeAgentExecutor{responses: []AgentResult{{Text: contentModel, SessionID: "agent-session-designed-svg"}}}
	svc := app.NewService(store)
	handler := NewServer(svc, Options{AgentExecutor: agent})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Designed HTML unsupported image"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	reportArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_designed_svg_report_md",
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   "svg-report.md",
		Producer:   app.Producer{Type: "agent", ID: "codex"},
		Content:    []byte("# SVG 제외 리포트\n\n이미지 필터를 검증합니다.\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := appendTestEvent(t, webServer, ctx, missionID, "report.artifact.created", map[string]any{
		"kind":        "markdown_report_artifact",
		"artifact_id": reportArtifact.ArtifactID,
		"media_type":  reportArtifact.MediaType,
	}, app.Producer{Type: "agent", ID: "codex"}); err != nil {
		t.Fatal(err)
	}
	addDesignedReportMediaSource(t, ctx, svc, missionID, "src_designed_svg_image", "art_designed_svg_image", "evt_designed_svg_image_source", "Unsafe SVG", "https://example.com/unsafe.svg", "image/svg+xml", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`))

	start := postJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+reportArtifact.ArtifactID+"/designed_html_export", map[string]any{"agent_executor": "codex"})
	if got, _ := start["status"].(string); got != "pending" {
		t.Fatalf("expected pending designed HTML response, got %#v", start)
	}
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.exported")
	if !strings.Contains(agent.requests[0].Prompt, "Available report_images") || !strings.Contains(agent.requests[0].Prompt, "\n[]\n\nMarkdown report artifact:") {
		t.Fatalf("expected unsupported SVG source to leave an empty designed HTML image inventory:\n%s", agent.requests[0].Prompt)
	}
	if strings.Contains(agent.requests[0].Prompt, "Unsafe SVG") {
		t.Fatalf("unsupported SVG source must not be exposed to designed HTML content model prompt:\n%s", agent.requests[0].Prompt)
	}
	exportPayload := latestEventPayload(t, detail, "report.artifact.exported", "designed_html_report_artifact")
	designedArtifactID, _ := exportPayload["artifact_id"].(string)
	htmlRead := getJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+designedArtifactID)
	content := nestedString(t, htmlRead, "content")
	for _, unexpected := range []string{"data:image/svg+xml", `<figure class="inline-report-image"`} {
		if strings.Contains(content, unexpected) {
			t.Fatalf("expected unsupported SVG to be skipped, but found %q:\n%s", unexpected, content)
		}
	}
	if !strings.Contains(content, "Unsafe SVG 이미지는 지원하지 않는 이미지 형식이라 HTML에 포함하지 않았습니다.") {
		t.Fatalf("expected skipped image note in designed HTML:\n%s", content)
	}
}

func TestReportArtifactDesignedHTMLExportIgnoresStaleRendererVersion(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	contentModel := `{
	  "kicker": "Designed Report",
	  "title": "새 리포트",
	  "subtitle": "새 렌더러로 만든다.",
	  "thesis": "예전 HTML artifact를 재사용하지 않아야 한다.",
	  "visual_units": [
	    {"title": "핵심 관계", "kind": "flow", "question": "무엇이 바뀌나", "nodes": [
	      {"label": "이전", "body": "예전 renderer artifact가 존재합니다.", "tone": "warn"},
	      {"label": "현재", "body": "새 renderer version으로 다시 만듭니다.", "tone": "good"}
	    ], "caption": "캐시 경계"}
	  ],
	  "tabs": [
	    {"label": "확인", "summary": "캐시 무효화 확인", "sections": [
	      {"heading": "새 artifact", "body": ["새 renderer version으로 생성합니다."]}
	    ]}
	  ]
	}`
	agent := &fakeAgentExecutor{responses: []AgentResult{{Text: contentModel, SessionID: "agent-session-designed"}}}
	svc := app.NewService(store)
	handler := NewServer(svc, Options{AgentExecutor: agent})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Designed HTML stale cache"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	reportArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_stale_source_md",
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   "report.md",
		Producer:   app.Producer{Type: "agent", ID: "codex"},
		Content:    []byte("# 리포트\n\n새 HTML export가 필요합니다.\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := appendTestEvent(t, webServer, ctx, missionID, "report.artifact.created", map[string]any{
		"kind":        "markdown_report_artifact",
		"artifact_id": reportArtifact.ArtifactID,
		"media_type":  reportArtifact.MediaType,
	}, app.Producer{Type: "agent", ID: "codex"}); err != nil {
		t.Fatal(err)
	}
	staleArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_stale_designed_html",
		MissionID:  missionID,
		MediaType:  "text/html; charset=utf-8",
		Filename:   "old.html",
		Producer:   app.Producer{Type: "agent", ID: "codex"},
		Content:    []byte("<!doctype html><html><body>old</body></html>"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := appendTestEvent(t, webServer, ctx, missionID, "report.artifact.exported", map[string]any{
		"kind":               reporting.ExportKindDesignedHTML,
		"source_artifact_id": reportArtifact.ArtifactID,
		"artifact_id":        staleArtifact.ArtifactID,
		"target":             reporting.ExportTargetDesignedHTML,
		"renderer_version":   "dh27-katex-math-20260713",
	}, app.Producer{Type: "agent", ID: "codex"}); err != nil {
		t.Fatal(err)
	}

	start := postJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+reportArtifact.ArtifactID+"/designed_html_export", map[string]any{"agent_executor": "codex"})
	if got, _ := start["status"].(string); got != "pending" {
		t.Fatalf("expected stale cache to be ignored and new export to start, got %#v", start)
	}
	detail := waitForEventTypeCount(t, server.URL, missionID, "report.artifact.exported", 2)
	exportPayload := latestEventPayload(t, detail, "report.artifact.exported", reporting.ExportKindDesignedHTML)
	if exportPayload["artifact_id"] == staleArtifact.ArtifactID {
		t.Fatalf("expected new designed artifact instead of stale cache, got %#v", exportPayload)
	}
	if exportPayload["renderer_version"] != designedReportRendererVersion {
		t.Fatalf("expected current renderer version, got %#v", exportPayload)
	}
	storedStaleArtifact, err := svc.GetRawArtifact(ctx, staleArtifact.ArtifactID)
	if err != nil {
		t.Fatalf("stale designed artifact was removed: %v", err)
	}
	if !bytes.Equal(storedStaleArtifact.Content, staleArtifact.Content) {
		t.Fatal("stale designed artifact was mutated during version cache miss")
	}
	if len(agent.requests) != 1 {
		t.Fatalf("expected agent to run for stale cache miss, got %d requests", len(agent.requests))
	}
}

func TestStaleAgentTurnAutoClosesBeforeDesignedHTMLExport(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	contentModel := `{"kicker":"Designed Report","title":"stale turn export","tabs":[{"label":"overview","sections":[{"heading":"export","body":["designed HTML export started after stale turn cleanup."]}]}]}`
	agent := &fakeAgentExecutor{responses: []AgentResult{{Text: contentModel, SessionID: "agent-session-designed"}}}
	svc := app.NewService(store)
	handler := NewServer(svc, Options{AgentExecutor: agent})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Designed HTML stale agent turn"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	reportArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_designed_stale_agent_md",
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   "report.md",
		Producer:   app.Producer{Type: "agent", ID: "codex"},
		Content:    []byte("# stale turn export\n\n본문입니다.\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := appendTestEvent(t, webServer, ctx, missionID, "report.artifact.created", map[string]any{
		"kind":        "markdown_report_artifact",
		"artifact_id": reportArtifact.ArtifactID,
		"media_type":  reportArtifact.MediaType,
	}, app.Producer{Type: "agent", ID: "codex"}); err != nil {
		t.Fatal(err)
	}
	appendStaleAgentPending(t, ctx, svc, missionID, "evt_stale_design_user", "evt_stale_design_pending")

	start := postJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+reportArtifact.ArtifactID+"/designed_html_export", map[string]any{"agent_executor": "codex"})
	if got, _ := start["status"].(string); got != "pending" {
		t.Fatalf("expected stale agent turn to be closed and designed HTML export to start, got %#v", start)
	}
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.exported")
	payload := firstEventPayload(t, detail, "turn.agent.response")
	if payload["kind"] != "agent_canceled" || payload["user_event_id"] != "evt_stale_design_user" {
		t.Fatalf("expected stale agent turn to be auto-canceled before designed HTML export, got %#v", payload)
	}
	if len(agent.requests) != 1 {
		t.Fatalf("expected designed HTML export to run once after stale cleanup, got %#v", agent.requests)
	}
}

func TestReportArtifactDesignedHTMLPendingBlocksNormalTurn(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	release := make(chan struct{})
	agent := blockingAgentExecutor{
		release: release,
		result:  AgentResult{Text: `{"title":"느린 리포트","tabs":[{"label":"개요","sections":[{"heading":"느린 생성","body":["완료되었습니다."]}]}]}`, SessionID: "agent-session-designed"},
	}
	svc := app.NewService(store)
	handler := NewServer(svc, Options{AgentExecutor: agent})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Designed HTML pending"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	reportArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_designed_pending_md",
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   "pending-report.md",
		Producer:   app.Producer{Type: "agent", ID: "codex"},
		Content:    []byte("# 느린 리포트\n\n본문입니다.\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := appendTestEvent(t, webServer, ctx, missionID, "report.artifact.created", map[string]any{
		"kind":        "markdown_report_artifact",
		"artifact_id": reportArtifact.ArtifactID,
		"media_type":  reportArtifact.MediaType,
	}, app.Producer{Type: "agent", ID: "codex"}); err != nil {
		t.Fatal(err)
	}

	postJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+reportArtifact.ArtifactID+"/designed_html_export", map[string]any{"agent_executor": "codex"})
	waitForEventType(t, server.URL, missionID, "report.design.pending")
	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "normal turn"})
	if status != http.StatusConflict && status != http.StatusBadRequest {
		t.Fatalf("expected active report design to block normal turn, got %d %#v", status, body)
	}
	if !strings.Contains(nestedString(t, body, "error", "message"), "report draft is already running") {
		t.Fatalf("expected report-running message, got %#v", body)
	}
	close(release)
	waitForEventType(t, server.URL, missionID, "report.artifact.exported")
}

func TestReportArtifactDesignedHTMLStalePendingResumes(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	contentModel := `{"kicker":"Designed Report","title":"재개 리포트","summary":"재개된 HTML입니다.","tabs":[{"label":"개요","sections":[{"heading":"재개","body":["기존 pending에서 이어졌습니다."]}]}]}`
	agent := &fakeAgentExecutor{responses: []AgentResult{{Text: contentModel, SessionID: "agent-session-designed"}}}
	svc := app.NewService(store)
	handler := NewServer(svc, Options{AgentExecutor: agent})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Designed HTML resume"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	reportArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_designed_resume_md",
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   "resume-report.md",
		Producer:   app.Producer{Type: "agent", ID: "codex"},
		Content:    []byte("# 재개 리포트\n\n본문입니다.\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := appendTestEvent(t, webServer, ctx, missionID, "report.artifact.created", map[string]any{
		"kind":        "markdown_report_artifact",
		"artifact_id": reportArtifact.ArtifactID,
		"media_type":  reportArtifact.MediaType,
	}, app.Producer{Type: "agent", ID: "codex"}); err != nil {
		t.Fatal(err)
	}
	pending, err := appendTestEvent(t, webServer, ctx, missionID, "report.design.pending", map[string]any{
		"kind":               "designed_html_report_pending",
		"source_artifact_id": reportArtifact.ArtifactID,
		"source_media_type":  reportArtifact.MediaType,
		"title":              reportArtifactTitle(reportArtifact),
		"agent_executor":     "codex",
		"target":             reporting.ExportTargetDesignedHTML,
		"renderer_version":   designedReportRendererVersion,
	}, app.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		t.Fatal(err)
	}

	server.Close()
	server = httptest.NewServer(NewServer(svc, Options{AgentExecutor: agent}))
	defer server.Close()
	status, _ := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+reportArtifact.ArtifactID+"/designed_html_export", map[string]any{"agent_executor": "codex"})
	if status != http.StatusConflict {
		t.Fatalf("expected fresh request to conflict while stale pending resumes, got %d", status)
	}
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.exported")
	if countEvents(detail, "report.design.failed") != 0 {
		t.Fatalf("stale designed HTML pending should resume without failure, got %#v", detail["events"])
	}
	payload := latestEventPayload(t, detail, "report.artifact.exported", reporting.ExportKindDesignedHTML)
	if payload["pending_event_id"] != pending.EventID || payload["source_artifact_id"] != reportArtifact.ArtifactID {
		t.Fatalf("expected export to close stale designed pending for source artifact, got %#v", payload)
	}
	if len(agent.requests) != 1 {
		t.Fatalf("expected stale designed export to run once, got %#v", agent.requests)
	}
}

func TestReportDraftDefaultCreatesPlannedMarkdownArtifact(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	agent := &fakeAgentExecutor{responses: []AgentResult{
		{Text: "mission answer", SessionID: "agent-session-1"},
		{Text: agentReportPlanJSON(agentReportPlan{
			Summary: "Use the current mission session for a planned report.",
			Sections: []agentReportSection{{
				Title:   "Mission summary",
				Purpose: "Summarize the mission.",
			}},
		}), SessionID: "agent-session-1"},
		{Text: "# Planned report\n\nA planned report can use the existing mission session.", SessionID: "agent-session-1"},
	}}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{
		"title":     "Quick report test",
		"objective": "Check one-take report generation",
	})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{
		"text": "Prepare a quick report later.",
	})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")

	response := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":          "Quick report",
		"rigor_level":    "exploratory",
		"direction_hint": "Emphasize operational trade-offs.",
	})
	if hint := nestedString(t, response, "pending_event", "Payload", "direction_hint"); hint != "Emphasize operational trade-offs." {
		t.Fatalf("expected pending event to retain report direction, got %q", hint)
	}
	if mode := nestedString(t, response, "pending_event", "Payload", "report_mode"); mode != reportModePlanned {
		t.Fatalf("expected planned report mode in pending event, got %q", mode)
	}
	if policy := nestedString(t, response, "pending_event", "Payload", "report_session_policy"); policy != reportSessionPolicySameSession {
		t.Fatalf("expected same-session report policy in pending event, got %q", policy)
	}
	if selection := nestedString(t, response, "pending_event", "Payload", "report_session_policy_selection"); selection != reportSessionPolicySelectionAutoSameSessionNoForker {
		t.Fatalf("expected non-forking executor selection in pending event, got %q", selection)
	}
	if humanize := nestedString(t, response, "pending_event", "Payload", "post_report_humanize"); humanize != "disabled" {
		t.Fatalf("expected default report draft to leave H5 disabled, got %q", humanize)
	}
	if profile := nestedString(t, response, "pending_event", "Payload", "generation_guidance_profile"); profile != reportGenerationGuidanceProfileVisualPlan {
		t.Fatalf("expected default report draft to use visual-plan generation guidance, got %q", profile)
	}
	if sha := nestedString(t, response, "pending_event", "Payload", "generation_guidance_sha256"); sha == "" {
		t.Fatalf("expected default report draft pending event to record generation guidance sha")
	}
	sourceContext, ok := nestedValue(t, response, "pending_event", "Payload", "source_context").(map[string]any)
	if !ok || sourceContext["schema_version"] != "plasma.report_source_context.v1" {
		t.Fatalf("Web report did not use shared source context contract: %#v", response)
	}
	if sources, ok := sourceContext["confluence_sources"].([]any); !ok || len(sources) != 0 {
		t.Fatalf("source-free Web report context changed: %#v", sourceContext)
	}
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if countEvents(detail, "report.plan.created") != 1 {
		t.Fatalf("planned report must create one plan event, got %#v", detail["events"])
	}
	if countEvents(detail, "report.humanize.pending") != 0 {
		t.Fatalf("default report draft must not enqueue H5 humanize work, got %#v", detail["events"])
	}
	if len(agent.requests) != 3 {
		t.Fatalf("expected answer turn, plan request, and planned report request, got %d", len(agent.requests))
	}
	planReq := agent.requests[1]
	if planReq.PreviousSessionID != "agent-session-1" {
		t.Fatalf("planned report should continue the mission session while planning, got %q", planReq.PreviousSessionID)
	}
	assertReportMCPToolSurface(t, planReq, plasmamcp.ToolReportPlanSubmit, plasmamcp.ToolSourcesRead)
	if !strings.Contains(planReq.Prompt, "Emphasize operational trade-offs.") || !strings.Contains(planReq.Prompt, reporting.DirectionAdvisory) {
		t.Fatalf("planned report direction did not reach planning prompt:\n%s", planReq.Prompt)
	}
	if !strings.Contains(planReq.Prompt, "Visual-aid planning guidance:") {
		t.Fatalf("planned report default visual guidance did not reach planning prompt:\n%s", planReq.Prompt)
	}
	reportReq := agent.requests[2]
	if reportReq.PreviousSessionID != "agent-session-1" {
		t.Fatalf("planned report should continue planning session, got %q", reportReq.PreviousSessionID)
	}
	assertReportMCPToolSurface(t, reportReq, plasmamcp.ToolSourcesRead)
	if !strings.Contains(reportReq.Prompt, "Emphasize operational trade-offs.") || !strings.Contains(reportReq.Prompt, reporting.DirectionAdvisory) {
		t.Fatalf("planned report direction did not reach writing prompt:\n%s", reportReq.Prompt)
	}
	for _, expected := range []string{
		"Plasma report as a Markdown artifact",
		"Visible generation plan",
		"Return only the Markdown report body",
		"Level: exploratory",
		"mission_id " + missionID,
		"Report writing guidance:",
		"Report visual-aid guidance:",
		"never improve fluency by dropping concrete source details",
		"Follow the generation plan's visual-aid intent",
		"use only \\(...\\) for inline math and \\[...\\] for display math",
	} {
		if !strings.Contains(reportReq.Prompt, expected) {
			t.Fatalf("expected one-take report prompt to contain %q:\n%s", expected, reportReq.Prompt)
		}
	}
	if strings.Contains(reportReq.Prompt, "Long-form human-writer guidance:") {
		t.Fatalf("planned report prompt must not include long-form writing guidance:\n%s", reportReq.Prompt)
	}
	payload := lastEventPayload(t, detail, "report.artifact.created")
	if payload["report_mode"] != reportModePlanned || payload["report_mode_label"] != reportModeLabelPlan {
		t.Fatalf("expected planned report payload, got %#v", payload)
	}
	if payload["composition_strategy"] != "planned_markdown" || payload["plan_review_state"] != "auto_accepted" {
		t.Fatalf("expected planned composition metadata, got %#v", payload)
	}
	if payload["post_report_humanize"] != "disabled" || payload["humanize_enabled"] != false ||
		payload["generation_guidance_profile"] != reportGenerationGuidanceProfileVisualPlan || strings.TrimSpace(fmt.Sprint(payload["generation_guidance_sha256"])) == "" {
		t.Fatalf("expected default visual-plan and disabled H5 metadata, got %#v", payload)
	}
	if payload["report_session_policy"] != reportSessionPolicySameSession ||
		payload["session_chain_kind"] != "same_session_report" ||
		payload["pre_report_research_session_id"] != "agent-session-1" ||
		payload["report_plan_session_id"] != "agent-session-1" ||
		payload["report_session_id"] != "agent-session-1" ||
		payload["fork_source_agent_session_id"] != "" ||
		payload["post_report_research_session_id"] != "" ||
		payload["report_session_policy_selection"] != reportSessionPolicySelectionAutoSameSessionNoForker {
		t.Fatalf("expected same-session report chain metadata, got %#v", payload)
	}
	artifact, err := svc.GetRawArtifact(ctx, payload["artifact_id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if artifact.MediaType != "text/markdown; charset=utf-8" || !strings.Contains(string(artifact.Content), "Planned report") {
		t.Fatalf("expected planned markdown report artifact, got %#v", artifact)
	}
}

func TestReportDraftDefaultUsesForkedReportSessionWhenAvailable(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	agent := &fakeForkingAgentExecutor{
		fakeAgentExecutor: fakeAgentExecutor{rejectDeadline: true, responses: []AgentResult{
			{Text: "mission answer", SessionID: "research-session-1"},
			{Text: agentReportPlanJSON(agentReportPlan{
				Summary: "Use an isolated report session for report work.",
				Sections: []agentReportSection{{
					Title:   "Mission summary",
					Purpose: "Summarize the mission.",
				}},
			}), SessionID: "report-fork-1", Resumed: true},
			{Text: "# Isolated report\n\nA report generated in a forked session.", SessionID: "report-fork-1", Resumed: true},
		}},
		forkSessionID: "report-fork-1",
	}
	handler := NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{
		"title":     "Isolated report test",
		"objective": "Check report session isolation",
	})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{
		"text": "Prepare an isolated report later.",
	})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")

	response := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":       "Isolated report",
		"report_mode": "planned",
	})
	if policy := nestedString(t, response, "pending_event", "Payload", "report_session_policy"); policy != reportSessionPolicyIsolatedFork {
		t.Fatalf("expected isolated fork report policy in pending event, got %q", policy)
	}
	if selection := nestedString(t, response, "pending_event", "Payload", "report_session_policy_selection"); selection != reportSessionPolicySelectionAutoIsolatedFork {
		t.Fatalf("expected automatic isolated fork selection in pending event, got %q", selection)
	}
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if len(agent.forkSources) != 1 || agent.forkSources[0] != "research-session-1" {
		t.Fatalf("expected one fork from research session, got %#v", agent.forkSources)
	}
	if len(agent.requests) != 3 {
		t.Fatalf("expected answer turn, plan request, and planned report request, got %d", len(agent.requests))
	}
	if got := agent.requests[1].PreviousSessionID; got != "report-fork-1" {
		t.Fatalf("expected report plan to resume forked session, got %q", got)
	}
	if got := agent.requests[2].PreviousSessionID; got != "report-fork-1" {
		t.Fatalf("expected report body to resume forked session, got %q", got)
	}

	payload := lastEventPayload(t, detail, "report.artifact.created")
	if payload["report_session_policy"] != reportSessionPolicyIsolatedFork ||
		payload["session_chain_kind"] != "isolated_fork_report" ||
		payload["pre_report_research_session_id"] != "research-session-1" ||
		payload["report_plan_session_id"] != "report-fork-1" ||
		payload["report_session_id"] != "report-fork-1" ||
		payload["fork_source_agent_session_id"] != "research-session-1" ||
		payload["post_report_research_session_id"] != "" ||
		payload["report_session_policy_selection"] != reportSessionPolicySelectionAutoIsolatedFork {
		t.Fatalf("expected isolated report chain metadata, got %#v", payload)
	}
	if got := webServer.latestAgentSessionID(ctx, missionID, "codex"); got != "research-session-1" {
		t.Fatalf("expected isolated report not to replace research session, got %q", got)
	}
}

func TestReportDraftLongFormUsesForkedReportSessionWhenAvailable(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	agent := &fakeForkingAgentExecutor{
		fakeAgentExecutor: fakeAgentExecutor{rejectDeadline: true, responses: []AgentResult{
			{Text: "mission answer", SessionID: "research-session-1"},
			{Text: agentReportAnyJSON(agentSectionalReportPlan{
				Summary: "Use an isolated session for the long-form report.",
				Parts: []agentReportPart{{
					Title:   "Core Part",
					Purpose: "Write one preserved section.",
					Sections: []agentReportSection{{
						Title:   "Core Section",
						Purpose: "Draft the section in the report fork.",
					}},
				}},
			}), SessionID: "report-fork-1", Resumed: true},
			{Text: "Forked long-form section body.", SessionID: "report-fork-1", Resumed: true},
			{Text: `{"intro":"Forked part intro.","transitions":[],"closing":"Forked part closing."}`, SessionID: "report-fork-1", Resumed: true},
			{Text: `{"front_matter":"# Forked Long Report\n\nForked opening.","closing":"Forked final closing."}`, SessionID: "report-fork-1", Resumed: true},
		}},
		forkSessionID: "report-fork-1",
	}
	handler := NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{
		"title":     "Long-form isolated report test",
		"objective": "Check long-form report session isolation",
	})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{
		"text": "Prepare a long-form report later.",
	})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")

	response := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":       "Forked Long Report",
		"report_mode": "long_form",
	})
	if policy := nestedString(t, response, "pending_event", "Payload", "report_session_policy"); policy != reportSessionPolicyIsolatedFork {
		t.Fatalf("expected isolated fork policy for long-form pending event, got %q", policy)
	}
	if selection := nestedString(t, response, "pending_event", "Payload", "report_session_policy_selection"); selection != reportSessionPolicySelectionAutoIsolatedFork {
		t.Fatalf("expected automatic isolated fork selection for long-form pending event, got %q", selection)
	}
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if len(agent.forkSources) != 1 || agent.forkSources[0] != "research-session-1" {
		t.Fatalf("expected one fork from research session, got %#v", agent.forkSources)
	}
	if len(agent.requests) != 5 {
		t.Fatalf("expected answer, plan, section, part, and frame requests, got %d", len(agent.requests))
	}
	for index := 1; index < len(agent.requests); index++ {
		if got := agent.requests[index].PreviousSessionID; got != "report-fork-1" {
			t.Fatalf("expected long-form report request %d to resume forked session, got %q", index, got)
		}
	}
	payload := lastEventPayload(t, detail, "report.artifact.created")
	if payload["report_mode"] != reportModeLongForm ||
		payload["report_session_policy"] != reportSessionPolicyIsolatedFork ||
		payload["session_chain_kind"] != "isolated_fork_report" ||
		payload["pre_report_research_session_id"] != "research-session-1" ||
		payload["report_plan_session_id"] != "report-fork-1" ||
		payload["report_session_id"] != "report-fork-1" ||
		payload["fork_source_agent_session_id"] != "research-session-1" ||
		payload["report_session_policy_selection"] != reportSessionPolicySelectionAutoIsolatedFork {
		t.Fatalf("expected isolated long-form report chain metadata, got %#v", payload)
	}
	if got := webServer.latestAgentSessionID(ctx, missionID, "codex"); got != "research-session-1" {
		t.Fatalf("expected isolated long-form report not to replace research session, got %q", got)
	}
}

func TestReportDraftLongFormSectionFanoutUsesForkedStageSessions(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	agent := &fakeForkingAgentExecutor{
		fakeAgentExecutor: fakeAgentExecutor{rejectDeadline: true, responses: []AgentResult{
			{Text: "mission answer", SessionID: "research-session-1"},
			{Text: agentReportAnyJSON(agentSectionalReportPlan{
				Summary: "Use section fanout for one preserved section.",
				Parts: []agentReportPart{{
					Title:   "Fanout Part",
					Purpose: "Write one section through the fanout strategy.",
					Sections: []agentReportSection{{
						Title:   "Fanout Section",
						Purpose: "Draft the independent section.",
					}},
				}},
			}), SessionID: "report-fork-1", Resumed: true},
			{Text: "Fanout section body.", SessionID: "report-fork-1", Resumed: true},
			{Text: `{"intro":"Fanout part intro.","transitions":[],"closing":"Fanout part closing."}`, SessionID: "report-fork-1", Resumed: true},
			{Text: `{"front_matter":"# Fanout Long Report\n\nFanout opening.","closing":"Fanout final closing."}`, SessionID: "report-fork-1", Resumed: true},
		}},
		forkSessionID: "report-fork-1",
	}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{
		"title":     "Section fanout report test",
		"objective": "Check section fanout report execution",
	})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{
		"text": "Prepare fanout report context.",
	})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")

	response := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":              "Fanout Long Report",
		"report_mode":        "long_form",
		"execution_strategy": "section_fanout",
	})
	if strategy := nestedString(t, response, "pending_event", "Payload", "execution_strategy"); strategy != reportExecutionStrategySectionFanout {
		t.Fatalf("expected pending event to preserve section fanout strategy, got %q", strategy)
	}
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if countEvents(detail, "report.plan.created") != 1 || countEvents(detail, "report.section.started") != 1 || countEvents(detail, "report.section.created") != 1 || countEvents(detail, "report.part.created") != 1 {
		t.Fatalf("expected plan, section start, section, and part events, got %#v", detail["events"])
	}
	if len(agent.requests) != 5 {
		t.Fatalf("expected answer, plan, section, part, and frame requests, got %d", len(agent.requests))
	}
	for index := 1; index < len(agent.requests); index++ {
		if got := agent.requests[index].PreviousSessionID; got != "report-fork-1" {
			t.Fatalf("expected fanout report request %d to resume a forked session, got %q", index, got)
		}
	}
	if len(agent.forkSources) != 4 || agent.forkSources[0] != "research-session-1" || agent.forkSources[1] != "report-fork-1" || agent.forkSources[2] != "report-fork-1" || agent.forkSources[3] != "report-fork-1" {
		t.Fatalf("expected report, section, part, and final forks, got %#v", agent.forkSources)
	}
	payload := lastEventPayload(t, detail, "report.artifact.created")
	if payload["report_mode"] != reportModeLongForm ||
		payload["session_chain_kind"] != "section_fanout_report" ||
		payload["composition_strategy"] != "sectional_preserve_markdown" ||
		payload["assembly_strategy"] != "c4_normalized_section_headings" {
		t.Fatalf("expected section fanout long-form metadata, got %#v", payload)
	}
	artifact, err := svc.GetRawArtifact(ctx, payload["artifact_id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if content := string(artifact.Content); !strings.Contains(content, "Fanout section body.") || !strings.Contains(content, "Fanout final closing.") {
		t.Fatalf("expected final Markdown to preserve section and closing:\n%s", content)
	}
}

func TestReportPatchUsesPreviousReportSessionNotLatestConversationSession(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	agent := &fakeForkingAgentExecutor{
		fakeAgentExecutor: fakeAgentExecutor{responses: []AgentResult{
			{Text: "newer research answer", SessionID: "research-session-2"},
			{Text: "patch attempted", SessionID: "report-patch-fork", Resumed: true},
		}},
		forkSessionID: "report-patch-fork",
	}
	handler := NewServer(svc, Options{AgentExecutor: agent})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{
		"title":     "Patch report session test",
		"objective": "Ensure report patch does not use current conversation session",
	})
	missionID := nestedString(t, mission, "projection", "mission_id")
	reportArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_patch_base_report",
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   "base-report.md",
		Producer:   app.Producer{Type: "agent_session", ID: "report-session-1"},
		Content:    []byte("# Base Report\n\nNeeds a patch.\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := appendTestEvent(t, webServer, ctx, missionID, "report.artifact.created", map[string]any{
		"kind":                            "markdown_report_artifact",
		"title":                           "Base Report",
		"artifact_id":                     reportArtifact.ArtifactID,
		"media_type":                      reportArtifact.MediaType,
		"agent_executor":                  "codex",
		"agent_session_id":                "report-session-1",
		"report_session_id":               "report-session-1",
		"report_session_policy":           reportSessionPolicyIsolatedFork,
		"report_session_policy_selection": reportSessionPolicySelectionAutoIsolatedFork,
	}, app.Producer{Type: "agent_session", ID: "report-session-1"}); err != nil {
		t.Fatal(err)
	}

	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{
		"text": "Continue normal research after the report.",
	})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	if got := webServer.latestAgentSessionID(ctx, missionID, "codex"); got != "research-session-2" {
		t.Fatalf("expected newer conversation session before patch, got %q", got)
	}

	response := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports/patch", map[string]any{
		"base_artifact_id": reportArtifact.ArtifactID,
		"instruction":      "Make the report clearer.",
	})
	if sessionID := nestedString(t, response, "pending_event", "Payload", "report_session_id"); sessionID != "report-patch-fork" {
		t.Fatalf("expected patch pending event to use report fork session, got %q", sessionID)
	}
	if policy := nestedString(t, response, "pending_event", "Payload", "report_session_policy"); policy != reportSessionPolicyIsolatedFork {
		t.Fatalf("expected patch to prefer isolated fork, got %q", policy)
	}
	waitForEventType(t, server.URL, missionID, "report.patch.failed")
	if len(agent.forkSources) != 1 || agent.forkSources[0] != "report-session-1" {
		t.Fatalf("expected patch fork from previous report session, got %#v", agent.forkSources)
	}
	if len(agent.requests) != 2 {
		t.Fatalf("expected normal turn plus patch request, got %d", len(agent.requests))
	}
	patchReq := agent.requests[1]
	if patchReq.PreviousSessionID != "report-patch-fork" {
		t.Fatalf("expected patch request to resume report fork, got %q", patchReq.PreviousSessionID)
	}
	if patchReq.PreviousSessionID == "research-session-2" {
		t.Fatalf("patch must not resume latest conversation session")
	}
	if patchReq.Model != "" || patchReq.ReasoningEffort != "" {
		t.Fatalf("legacy report patch must preserve empty report settings, got %#v", patchReq)
	}
	if !strings.Contains(patchReq.Prompt, reportArtifact.ArtifactID) || !strings.Contains(patchReq.Prompt, "report-patch-fork") {
		t.Fatalf("expected patch prompt to include base artifact and report session:\n%s", patchReq.Prompt)
	}
	hasFinalizeTool := false
	for _, tool := range patchReq.ExtraMCPTools {
		if tool == "plasma.report.patch.finalize" {
			hasFinalizeTool = true
			break
		}
	}
	if !hasFinalizeTool {
		t.Fatalf("expected patch request to enable report patch MCP tools, got %#v", patchReq.ExtraMCPTools)
	}
	if !patchReq.ReplaceMCPTools {
		t.Fatalf("expected patch request to replace default MCP tools, got %#v", patchReq)
	}
	if patchReq.ReportPatch == nil {
		t.Fatalf("expected patch request to bind report patch MCP context")
	}
	if patchReq.ReportPatch.BaseArtifactID != reportArtifact.ArtifactID ||
		patchReq.ReportPatch.ReportSessionID != "report-patch-fork" ||
		patchReq.ReportPatch.PendingEventID == "" {
		t.Fatalf("expected bound report patch MCP context, got %#v", patchReq.ReportPatch)
	}
}

func TestReportPatchDoesNotPromoteFinalizedPatchWhenAgentSessionValidationFails(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	var webServer *Server
	agent := &fakeAgentExecutor{
		responses: []AgentResult{{Text: "patch finalized in tool", SessionID: "wrong-session"}},
		onRun: func(runCtx context.Context, req AgentRequest) {
			if req.ReportPatch == nil {
				return
			}
			artifact, err := svc.CreateRawArtifact(runCtx, app.CreateRawArtifactRequest{
				ArtifactID: "art_patch_wrong_session",
				MissionID:  req.MissionID,
				MediaType:  "text/markdown; charset=utf-8",
				Filename:   "patched.md",
				Producer:   app.Producer{Type: "mcp_tool", ID: "plasma.report.patch.finalize"},
				Content:    []byte("# Patched Report\n\nThis should not be promoted.\n"),
			})
			if err != nil {
				t.Errorf("create provisional artifact: %v", err)
				return
			}
			if _, err := appendTestEvent(t, webServer, runCtx, req.MissionID, "report.patch.finalized", map[string]any{
				"kind":                            "markdown_report_patch_finalized",
				"pending_event_id":                req.ReportPatch.PendingEventID,
				"title":                           "Patched Report",
				"artifact_id":                     artifact.ArtifactID,
				"media_type":                      artifact.MediaType,
				"agent_executor":                  "codex",
				"agent_session_id":                "report-session-1",
				"report_session_id":               "report-session-1",
				"report_session_policy":           reportSessionPolicySameSession,
				"report_session_policy_selection": reporting.SessionPolicySelectionExplicitSameSession,
				"tool_session_id":                 req.ToolSessionID,
				"composition_strategy":            "mcp_patch_markdown",
			}, app.Producer{Type: "mcp_tool", ID: "plasma.report.patch.finalize"}); err != nil {
				t.Errorf("append provisional finalize event: %v", err)
			}
		},
	}
	handler := NewServer(svc, Options{AgentExecutor: agent})
	webServer = handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{
		"title":     "Patch validation test",
		"objective": "Do not publish report patches from the wrong session",
	})
	missionID := nestedString(t, mission, "projection", "mission_id")
	reportArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_patch_validation_base",
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   "base-report.md",
		Producer:   app.Producer{Type: "agent_session", ID: "report-session-1"},
		Content:    []byte("# Base Report\n\nNeeds a patch.\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := appendTestEvent(t, webServer, ctx, missionID, "report.artifact.created", map[string]any{
		"kind":                            "markdown_report_artifact",
		"title":                           "Base Report",
		"artifact_id":                     reportArtifact.ArtifactID,
		"media_type":                      reportArtifact.MediaType,
		"agent_executor":                  "codex",
		"agent_session_id":                "report-session-1",
		"report_session_id":               "report-session-1",
		"report_session_policy":           reportSessionPolicySameSession,
		"report_session_policy_selection": reporting.SessionPolicySelectionExplicitSameSession,
	}, app.Producer{Type: "agent_session", ID: "report-session-1"}); err != nil {
		t.Fatal(err)
	}

	response := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports/patch", map[string]any{
		"base_artifact_id":      reportArtifact.ArtifactID,
		"instruction":           "Make the report clearer.",
		"report_session_policy": reportSessionPolicySameSession,
	})
	pendingEventID := nestedString(t, response, "pending_event", "EventID")
	detail := waitForEventType(t, server.URL, missionID, "report.patch.failed")
	if !hasEventForPending(detail, "report.patch.finalized", pendingEventID) {
		t.Fatalf("expected provisional finalized patch event for pending %s", pendingEventID)
	}
	if hasEventForPending(detail, "report.artifact.created", pendingEventID) {
		t.Fatalf("must not promote finalized patch when agent session validation fails")
	}
}

func TestReportDraftOneTakeKeepsSameSessionEvenWhenForkIsAvailable(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	agent := &fakeForkingAgentExecutor{
		fakeAgentExecutor: fakeAgentExecutor{rejectDeadline: true, responses: []AgentResult{
			{Text: "mission answer", SessionID: "research-session-1"},
			{Text: "# Quick Report\n\nSame-session quick report.", SessionID: "research-session-1", Resumed: true},
		}},
		forkSessionID: "report-fork-1",
	}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{
		"title":     "One-take report test",
		"objective": "Check one-take report session policy",
	})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{
		"text": "Prepare a quick report later.",
	})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")

	response := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":       "Quick Report",
		"report_mode": "one_take",
	})
	if policy := nestedString(t, response, "pending_event", "Payload", "report_session_policy"); policy != reportSessionPolicySameSession {
		t.Fatalf("expected one-take pending event to use same session, got %q", policy)
	}
	if selection := nestedString(t, response, "pending_event", "Payload", "report_session_policy_selection"); selection != reportSessionPolicySelectionAutoSameSessionOneTake {
		t.Fatalf("expected one-take same-session selection, got %q", selection)
	}
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if len(agent.forkSources) != 0 {
		t.Fatalf("one-take report must not fork, got %#v", agent.forkSources)
	}
	if len(agent.requests) != 2 || agent.requests[1].PreviousSessionID != "research-session-1" {
		t.Fatalf("expected one-take report to resume research session, got %#v", agent.requests)
	}
	if !strings.Contains(agent.requests[1].Prompt, "use only \\(...\\) for inline math and \\[...\\] for display math") {
		t.Fatalf("one-take report prompt missing canonical math syntax:\n%s", agent.requests[1].Prompt)
	}
	payload := lastEventPayload(t, detail, "report.artifact.created")
	if payload["report_mode"] != reportModeOneTake ||
		payload["report_session_policy"] != reportSessionPolicySameSession ||
		payload["report_session_policy_selection"] != reportSessionPolicySelectionAutoSameSessionOneTake ||
		payload["report_session_id"] != "research-session-1" ||
		payload["pre_report_research_session_id"] != "research-session-1" {
		t.Fatalf("expected one-take same-session report metadata, got %#v", payload)
	}
}

func TestReportDraftCreatesHumanizedMarkdownArtifact(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	agent := &fakeAgentExecutor{responses: []AgentResult{
		{Text: "research ready", SessionID: "research-session-1"},
		{Text: "# Report\n\n수행되어야 한다.", SessionID: "research-session-1"},
		{Text: "H5 patch finalized.", SessionID: "research-session-1"},
	}}
	agent.onRun = fakeHumanizePatchFinalizer(t, svc, "# Report\n\n수행해야 한다.")
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Humanize test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "Prepare report context."})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":                "Humanized Report",
		"report_mode":          "one_take",
		"post_report_humanize": "enabled",
	})

	detail := waitForEventType(t, server.URL, missionID, "report.artifact.exported")
	if countEvents(detail, "report.humanize.pending") != 1 {
		t.Fatalf("expected one humanize pending event, got %#v", detail["events"])
	}
	pendingEvent := lastEvent(t, detail, "report.humanize.pending")
	pendingEventID, _ := pendingEvent["EventID"].(string)
	pendingPayload, _ := pendingEvent["Payload"].(map[string]any)
	if pendingEventID == "" || pendingPayload["pending_event_id"] != pendingEventID {
		t.Fatalf("expected humanize pending payload to identify itself, event=%#v payload=%#v", pendingEvent, pendingPayload)
	}
	sourcePayload := lastEventPayload(t, detail, "report.artifact.created")
	humanizedPayload := latestEventPayload(t, detail, "report.artifact.exported", reporting.ExportKindHumanizedMarkdown)
	if humanizedPayload["target"] != reporting.ExportTargetHumanizedMarkdown ||
		humanizedPayload["source_artifact_id"] != sourcePayload["artifact_id"] ||
		humanizedPayload["relationship"] != "post_report_tone_pass_of_source_artifact" ||
		humanizedPayload["humanize_transport"] != reporting.HumanizeTransportPatch ||
		humanizedPayload["pending_event_id"] != pendingEventID ||
		humanizedPayload["report_pending_event_id"] != sourcePayload["pending_event_id"] {
		t.Fatalf("expected explicit humanized artifact relationship, got %#v", humanizedPayload)
	}
	if hasOpenReportDraftDetail(detail) {
		t.Fatalf("expected humanize pending to close after export")
	}
	source, err := svc.GetRawArtifact(ctx, sourcePayload["artifact_id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	humanized, err := svc.GetRawArtifact(ctx, humanizedPayload["artifact_id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if string(source.Content) != "# Report\n\n수행되어야 한다." || string(humanized.Content) != "# Report\n\n수행해야 한다." {
		t.Fatalf("expected source preserved and humanized artifact separate, source=%q humanized=%q", string(source.Content), string(humanized.Content))
	}
	humanizedArtifactID := humanizedPayload["artifact_id"].(string)
	readResp := getJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+humanizedArtifactID)
	if readResp["content"] != "# Report\n\n수행해야 한다." {
		t.Fatalf("expected humanized artifact read content, got %#v", readResp)
	}
	downloadResp, err := http.Get(server.URL + "/api/missions/" + missionID + "/artifacts/" + humanizedArtifactID + "/download")
	if err != nil {
		t.Fatal(err)
	}
	defer downloadResp.Body.Close()
	if downloadResp.StatusCode != http.StatusOK {
		t.Fatalf("expected humanized artifact download 200, got %d", downloadResp.StatusCode)
	}
	downloadBody, err := io.ReadAll(downloadResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(downloadBody) != "# Report\n\n수행해야 한다." {
		t.Fatalf("expected humanized artifact download body, got %q", string(downloadBody))
	}
	if len(agent.requests) != 3 ||
		agent.requests[2].DisableTools ||
		!agent.requests[2].ReplaceMCPTools ||
		agent.requests[2].PreviousSessionID != "research-session-1" ||
		agent.requests[2].ReportPatch == nil {
		t.Fatalf("expected humanize request to use report patch MCP on the report session, got %#v", agent.requests)
	}
	if strings.Contains(agent.requests[2].Prompt, "# Report\n\n수행되어야 한다.") {
		t.Fatalf("humanize prompt must not include the full Markdown body: %q", agent.requests[2].Prompt)
	}
}

func TestReportDraftSkipsHumanizeWhenPatchSessionMakesNoSafeChanges(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	agent := &fakeAgentExecutor{responses: []AgentResult{
		{Text: "research ready", SessionID: "research-session-1"},
		{Text: "# Report\n\n이미 자연스러운 문장입니다.", SessionID: "research-session-1"},
		{Text: "검토했지만 안전하게 바꿀 문장이 없습니다.", SessionID: "research-session-1"},
	}}
	agent.onRun = fakeHumanizePatchReader(t, svc)
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Humanize no-op test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "Prepare report context."})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":                "Natural Report",
		"report_mode":          "one_take",
		"post_report_humanize": "enabled",
	})

	detail := waitForEventType(t, server.URL, missionID, "report.humanize.skipped")
	if countEvents(detail, "report.humanize.pending") != 1 ||
		countEvents(detail, "report.humanize.skipped") != 1 ||
		countEvents(detail, "report.humanize.failed") != 0 ||
		countEvents(detail, "report.artifact.exported") != 0 {
		t.Fatalf("expected no-op H5 patch session to skip without failure or export, got %#v", detail["events"])
	}
	if hasOpenReportDraftDetail(detail) {
		t.Fatalf("expected skipped humanize pending to close")
	}
	if len(agent.requests) != 3 ||
		agent.requests[2].ReportPatch == nil ||
		!agent.requests[2].ReplaceMCPTools {
		t.Fatalf("expected no-op humanize to use report patch MCP, got %#v", agent.requests)
	}

	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	var sawStart, sawRead bool
	for _, event := range events {
		if event.EventType != "mcp.tool.called" {
			continue
		}
		var payload struct {
			ToolName      string `json:"tool_name"`
			ToolSessionID string `json:"tool_session_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.ToolSessionID != agent.requests[2].ToolSessionID {
			continue
		}
		sawStart = sawStart || payload.ToolName == plasmamcp.ToolReportPatchStart
		sawRead = sawRead || payload.ToolName == plasmamcp.ToolReportPatchRead
		if payload.ToolName == plasmamcp.ToolReportPatchApply || payload.ToolName == plasmamcp.ToolReportPatchFinalize {
			t.Fatalf("no-op H5 session must not apply or finalize, got %s", payload.ToolName)
		}
	}
	if !sawStart || !sawRead {
		t.Fatalf("expected H5 no-op session to start and read, sawStart=%v sawRead=%v", sawStart, sawRead)
	}
}

func TestReportArtifactHumanizeRetryUsesExistingReportSession(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	agent := &fakeAgentExecutor{responses: []AgentResult{
		{Text: "research ready", SessionID: "research-session-1"},
		{Text: "# Report\n\n수행되어야 한다.", SessionID: "research-session-1"},
		{Text: "forgot to finalize", SessionID: "research-session-1"},
		{Text: "H5 patch finalized.", SessionID: "research-session-1"},
	}}
	finalize := fakeHumanizePatchFinalizer(t, svc, "# Report\n\n수행해야 한다.")
	humanizeRuns := 0
	agent.onRun = func(runCtx context.Context, req AgentRequest) {
		if req.ReportPatch == nil {
			return
		}
		humanizeRuns++
		if humanizeRuns == 2 {
			finalize(runCtx, req)
		}
	}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Humanize retry test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "Prepare report context."})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":                "Humanized Retry Report",
		"report_mode":          "one_take",
		"post_report_humanize": "enabled",
	})
	detail := waitForEventType(t, server.URL, missionID, "report.humanize.failed")
	sourcePayload := lastEventPayload(t, detail, "report.artifact.created")
	sourceArtifactID, _ := sourcePayload["artifact_id"].(string)
	if sourceArtifactID == "" {
		t.Fatalf("expected source report artifact, got %#v", sourcePayload)
	}
	if countEvents(detail, "report.humanize.pending") != 1 || countEvents(detail, "report.humanize.failed") != 1 {
		t.Fatalf("expected initial humanize failure to close the first pending, got %#v", detail["events"])
	}

	start := postJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+sourceArtifactID+"/humanized_markdown_export", map[string]any{})
	if start["status"] != "pending" {
		t.Fatalf("expected retry to start pending humanize job, got %#v", start)
	}
	detail = waitForEventTypeCount(t, server.URL, missionID, "report.artifact.exported", 1)
	if countEvents(detail, "report.humanize.pending") != 2 || countEvents(detail, "report.humanize.failed") != 1 {
		t.Fatalf("expected retry humanize pending to complete without new failure, got %#v", detail["events"])
	}
	humanizedPayload := latestEventPayload(t, detail, "report.artifact.exported", reporting.ExportKindHumanizedMarkdown)
	if humanizedPayload["source_artifact_id"] != sourceArtifactID ||
		humanizedPayload["target"] != reporting.ExportTargetHumanizedMarkdown ||
		humanizedPayload["previous_agent_session_id"] != "research-session-1" {
		t.Fatalf("expected retry to humanize the original report session artifact, got %#v", humanizedPayload)
	}
	if len(agent.requests) != 4 ||
		agent.requests[3].PreviousSessionID != "research-session-1" ||
		agent.requests[3].ReportPatch == nil ||
		!agent.requests[3].ReplaceMCPTools {
		t.Fatalf("expected retry to use report patch MCP on the existing report session, got %#v", agent.requests)
	}
	if strings.Contains(agent.requests[3].Prompt, "# Report\n\n수행되어야 한다.") {
		t.Fatalf("retry prompt must not include the full Markdown body: %q", agent.requests[3].Prompt)
	}
}

func TestReportDraftKeepsOriginalWhenHumanizeGuardFails(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	agent := &fakeAgentExecutor{responses: []AgentResult{
		{Text: "research ready", SessionID: "research-session-1"},
		{Text: "# Report\n\n첫 문단입니다.\n\n둘째 문단입니다.", SessionID: "research-session-1"},
		{Text: "H5 patch finalized.", SessionID: "research-session-1"},
	}}
	agent.onRun = fakeHumanizePatchFinalizer(t, svc, "# Report\n\n첫 문단입니다.")
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Humanize guard test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "Prepare report context."})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":                "Guarded Report",
		"report_mode":          "one_take",
		"post_report_humanize": "enabled",
	})

	detail := waitForEventType(t, server.URL, missionID, "report.humanize.failed")
	if countEvents(detail, "report.humanize.pending") != 1 {
		t.Fatalf("expected one humanize pending event, got %#v", detail["events"])
	}
	if countEvents(detail, "report.artifact.exported") != 0 {
		t.Fatalf("guard failure must not create a humanized artifact, got %#v", detail["events"])
	}
	if countEvents(detail, "report.patch.rejected") != 1 {
		t.Fatalf("guard failure must reject the finalized patch artifact, got %#v", detail["events"])
	}
	pendingEvent := lastEvent(t, detail, "report.humanize.pending")
	pendingEventID, _ := pendingEvent["EventID"].(string)
	sourcePayload := lastEventPayload(t, detail, "report.artifact.created")
	patchPayload := lastEventPayload(t, detail, "report.patch.finalized")
	rejectedPayload := lastEventPayload(t, detail, "report.patch.rejected")
	failurePayload := lastEventPayload(t, detail, "report.humanize.failed")
	if failurePayload["source_artifact_id"] != sourcePayload["artifact_id"] ||
		failurePayload["preserved_original_markdown"] != true ||
		failurePayload["pending_event_id"] != pendingEventID ||
		failurePayload["report_pending_event_id"] != sourcePayload["pending_event_id"] {
		t.Fatalf("expected failure to point at preserved source artifact, got %#v", failurePayload)
	}
	if patchPayload["artifact_id"] == "" || rejectedPayload["artifact_id"] != patchPayload["artifact_id"] {
		t.Fatalf("expected rejected patch artifact to match finalized patch artifact, finalized=%#v rejected=%#v", patchPayload, rejectedPayload)
	}
	if hasOpenReportDraftDetail(detail) {
		t.Fatalf("expected humanize pending to close after failure")
	}
	source, err := svc.GetRawArtifact(ctx, sourcePayload["artifact_id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if string(source.Content) != "# Report\n\n첫 문단입니다.\n\n둘째 문단입니다." {
		t.Fatalf("expected original artifact to remain unchanged, got %q", string(source.Content))
	}
	patchArtifactID := patchPayload["artifact_id"].(string)
	page, err := svc.ListMissionObjects(ctx, missionID, app.ResearchIDEObjectRawArtifact, 20, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range page.Items {
		if item.ObjectID == patchArtifactID {
			t.Fatalf("rejected H5 patch artifact must not be listed in research raw artifacts: %#v", page.Items)
		}
	}
	if _, err := svc.ReadMissionObject(ctx, app.ResearchIDEReadRequest{MissionID: missionID, ObjectKind: app.ResearchIDEObjectRawArtifact, ObjectID: patchArtifactID}); err == nil {
		t.Fatalf("rejected H5 patch artifact must not be readable through research raw artifacts")
	}
}

func TestReportDraftRejectsHumanizePatchArtifactWhenTerminalRaceWins(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	finalize := fakeHumanizePatchFinalizer(t, svc, "# Report\n\n수행해야 한다.")
	agent := &fakeAgentExecutor{responses: []AgentResult{
		{Text: "research ready", SessionID: "research-session-1"},
		{Text: "# Report\n\n수행되어야 한다.", SessionID: "research-session-1"},
		{Text: "H5 patch finalized.", SessionID: "research-session-1"},
	}}
	agent.onRun = func(runCtx context.Context, req AgentRequest) {
		finalize(runCtx, req)
		if req.ReportPatch == nil {
			return
		}
		if _, err := svc.AppendEvent(runCtx, app.AppendEventRequest{
			EventID:   newID("evt"),
			MissionID: req.MissionID,
			EventType: "report.humanize.failed",
			Producer:  app.Producer{Type: "user", ID: "plasma-ui"},
			Payload: mustJSON(map[string]any{
				"kind":             "humanized_markdown_report_canceled",
				"pending_event_id": req.ReportPatch.PendingEventID,
				"canceled":         true,
			}),
		}); err != nil {
			t.Errorf("append terminal race event: %v", err)
		}
	}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Humanize terminal race test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "Prepare report context."})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":                "Terminal Race Report",
		"report_mode":          "one_take",
		"post_report_humanize": "enabled",
	})

	detail := waitForEventType(t, server.URL, missionID, "report.patch.rejected")
	if countEvents(detail, "report.artifact.exported") != 0 {
		t.Fatalf("terminal race must not export the humanized artifact, got %#v", detail["events"])
	}
	patchPayload := lastEventPayload(t, detail, "report.patch.finalized")
	rejectedPayload := lastEventPayload(t, detail, "report.patch.rejected")
	if rejectedPayload["reason"] != "terminal_already_closed" ||
		rejectedPayload["artifact_id"] != patchPayload["artifact_id"] {
		t.Fatalf("expected terminal race to reject finalized patch artifact, finalized=%#v rejected=%#v", patchPayload, rejectedPayload)
	}
	patchArtifactID := patchPayload["artifact_id"].(string)
	page, err := svc.ListMissionObjects(ctx, missionID, app.ResearchIDEObjectRawArtifact, 20, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range page.Items {
		if item.ObjectID == patchArtifactID {
			t.Fatalf("terminal-race rejected artifact must not be listed in research raw artifacts: %#v", page.Items)
		}
	}
	if _, err := svc.ReadMissionObject(ctx, app.ResearchIDEReadRequest{MissionID: missionID, ObjectKind: app.ResearchIDEObjectRawArtifact, ObjectID: patchArtifactID}); err == nil {
		t.Fatalf("terminal-race rejected artifact must not be readable through research raw artifacts")
	}
}

func TestReportDraftClosesHumanizePendingWhenOutputUnchanged(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	agent := &fakeAgentExecutor{responses: []AgentResult{
		{Text: "research ready", SessionID: "research-session-1"},
		{Text: "# Report\n\n이미 충분히 자연스럽다.", SessionID: "research-session-1"},
		{Text: "NO_H5_CHANGES", SessionID: "research-session-1"},
	}}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Humanize no-op test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "Prepare report context."})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":                "No-op Report",
		"report_mode":          "one_take",
		"post_report_humanize": "enabled",
	})

	detail := waitForEventType(t, server.URL, missionID, "report.humanize.skipped")
	if countEvents(detail, "report.humanize.pending") != 1 || countEvents(detail, "report.artifact.exported") != 0 {
		t.Fatalf("expected pending to close with skipped and no export, got %#v", detail["events"])
	}
	pendingEvent := lastEvent(t, detail, "report.humanize.pending")
	pendingEventID, _ := pendingEvent["EventID"].(string)
	sourcePayload := lastEventPayload(t, detail, "report.artifact.created")
	payload := lastEventPayload(t, detail, "report.humanize.skipped")
	if payload["preserved_original_markdown"] != true ||
		payload["relationship"] != "no_change_post_report_tone_pass_of_source_artifact" ||
		payload["pending_event_id"] != pendingEventID ||
		payload["report_pending_event_id"] != sourcePayload["pending_event_id"] {
		t.Fatalf("expected explicit no-change relationship, got %#v", payload)
	}
	if hasOpenReportDraftDetail(detail) {
		t.Fatalf("expected humanize pending to close after skipped event")
	}
	if len(agent.requests) != 3 ||
		agent.requests[2].DisableTools ||
		!agent.requests[2].ReplaceMCPTools ||
		agent.requests[2].PreviousSessionID != "research-session-1" ||
		agent.requests[2].ReportPatch == nil {
		t.Fatalf("expected humanize request to use report patch MCP, got %#v", agent.requests)
	}
}

func TestHumanizeMarkdownReportClosesPendingAfterContextCancellation(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	mission, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: "mis_humanize_cancel", Title: "Humanize cancel test"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_humanize_cancel_source",
		MissionID:  mission.MissionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   "cancel-source.md",
		Producer:   app.Producer{Type: "agent_session", ID: "report-session-1"},
		Content:    []byte("# Report\n\n취소되어야 합니다."),
	})
	if err != nil {
		t.Fatal(err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	executor := &cancelingHumanizeExecutor{cancel: cancel}
	result, err := HumanizeMarkdownReport(runCtx, svc, newID, mission.MissionID, ReportHumanizeInput{
		Title:             "Canceled Humanize",
		Markdown:          string(source.Content),
		SourceArtifact:    source,
		ExecutorName:      "codex",
		MCPMode:           "auto",
		PreviousSessionID: "report-session-1",
		ReportMode:        "planned",
		PendingEventID:    "evt_report_pending",
	}, executor)
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied {
		t.Fatalf("canceled H5 pass must not create a humanized artifact: %#v", result)
	}
	if len(executor.requests) != 1 ||
		executor.requests[0].DisableTools ||
		!executor.requests[0].ReplaceMCPTools ||
		executor.requests[0].PreviousSessionID != "report-session-1" ||
		executor.requests[0].ReportPatch == nil {
		t.Fatalf("H5 pass must use report patch MCP on the source report session, got %#v", executor.requests)
	}

	events, err := svc.ListEvents(ctx, mission.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	if countLedgerEvents(events, "report.humanize.pending") != 1 ||
		countLedgerEvents(events, "report.humanize.failed") != 1 ||
		countLedgerEvents(events, "report.artifact.exported") != 0 {
		t.Fatalf("expected canceled humanize pending to close with failed event, got %#v", events)
	}
}

func TestReportDraftRejectsUnavailableIsolatedForkPolicy(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{
		"title":     "Report session policy test",
		"objective": "Check isolated report session preflight behavior",
	})
	missionID := nestedString(t, mission, "projection", "mission_id")

	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":                 "Isolated report",
		"report_mode":           "planned",
		"report_session_policy": "isolated_fork",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected unavailable isolated fork policy to return 400, got %d %#v", status, body)
	}
	if !strings.Contains(nestedString(t, body, "error", "message"), "unavailable") {
		t.Fatalf("expected unavailable error message, got %#v", body)
	}
	if len(agent.requests) != 0 {
		t.Fatalf("isolated fork preflight must not start an agent request, got %#v", agent.requests)
	}
	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	if countEvents(detail, "report.draft.pending") != 0 || countEvents(detail, "report.artifact.created") != 0 {
		t.Fatalf("unavailable isolated fork policy must not create report events, got %#v", detail["events"])
	}
}

func TestReportDraftRejectsExplicitIsolatedForkWithoutResearchSession(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeForkingAgentExecutor{forkSessionID: "report-fork-1"}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{
		"title":     "Report fork preflight test",
		"objective": "Check isolated report fork preflight",
	})
	missionID := nestedString(t, mission, "projection", "mission_id")

	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":                 "Isolated report",
		"report_mode":           "planned",
		"report_session_policy": "isolated_fork",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected unavailable isolated fork policy to return 400, got %d %#v", status, body)
	}
	if !strings.Contains(nestedString(t, body, "error", "message"), "requires a pre-report research session") {
		t.Fatalf("expected pre-report session error message, got %#v", body)
	}
	if len(agent.requests) != 0 || len(agent.forkSources) != 0 {
		t.Fatalf("isolated fork preflight must not start agent or fork work, got requests=%#v forks=%#v", agent.requests, agent.forkSources)
	}
	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	if countEvents(detail, "report.draft.pending") != 0 || countEvents(detail, "report.artifact.created") != 0 {
		t.Fatalf("disabled isolated fork policy must not create report events, got %#v", detail["events"])
	}
}

func TestReportDraftRequestFromPendingEventPreservesSessionPolicy(t *testing.T) {
	payload, err := json.Marshal(map[string]any{
		"title":                           "Recover isolated report",
		"agent_executor":                  "codex",
		"mcp_mode":                        "auto",
		"rigor_level":                     "balanced",
		"report_mode":                     "planned",
		"report_session_policy":           reportSessionPolicyIsolatedFork,
		"report_session_policy_selection": reportSessionPolicySelectionAutoIsolatedFork,
		"post_report_humanize":            "enabled",
		"generation_guidance_profile":     "none",
		"generation_guidance_sha256":      "sha-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := reportDraftRequestFromPendingEvent(app.LedgerEvent{Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	if req.ReportSessionPolicy != reportSessionPolicyIsolatedFork ||
		req.ReportSessionPolicySelection != reportSessionPolicySelectionAutoIsolatedFork {
		t.Fatalf("expected recovered session policy metadata, got %#v", req)
	}
	if req.PostReportHumanize != "enabled" ||
		req.GenerationGuidanceProfile != "none" ||
		req.GenerationGuidanceSHA256 != "sha-test" {
		t.Fatalf("expected recovered report generation metadata, got %#v", req)
	}
}

func TestReportDraftLongFormCreatesSectionalPreservedMarkdownArtifact(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	sectionOne := "첫 번째 섹션 본문입니다. 구체적 사건과 맥락을 길게 보존합니다."
	sectionTwo := "두 번째 섹션 본문입니다. 반대 관점과 불확실성을 그대로 남깁니다."
	agent := &fakeAgentExecutor{responses: []AgentResult{
		{Text: agentReportAnyJSON(agentSectionalReportPlan{
			Summary: "섹션을 보존하는 장문 보고서를 만든다.",
			Parts: []agentReportPart{{
				Title:   "핵심 파트",
				Purpose: "두 섹션을 보존해 하나의 파트로 조립한다.",
				Sections: []agentReportSection{
					{Title: "사건과 맥락", Purpose: "첫 번째 섹션을 작성한다."},
					{Title: "긴장과 불확실성", Purpose: "두 번째 섹션을 작성한다."},
				},
			}},
		}), SessionID: "report-session-1"},
		{Text: sectionOne, SessionID: "report-session-1"},
		{Text: sectionTwo, SessionID: "report-session-1"},
		{Text: `{"intro":"파트 도입부입니다.","transitions":[{"after_section_index":1,"markdown":"두 섹션을 이어주는 전환문입니다."}],"closing":"파트 마무리입니다."}`, SessionID: "report-session-1"},
		{Text: `{"front_matter":"# 보존형 장문 보고서\n\n읽기 안내입니다.","closing":"## 결론\n\n최종 결론입니다."}`, SessionID: "report-session-1", Usage: agentusage.New("openai", "codex", "gpt-5.5", "high", "finalize").WithProviderUsage(agentusage.ProviderUsage{InputTokens: 11, OutputTokens: 7}, "fixture")},
	}}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{
		"title":     "Sectional report test",
		"objective": "Check section-first report generation",
	})
	missionID := nestedString(t, mission, "projection", "mission_id")
	response := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":                  "Sectional report",
		"rigor_level":            "balanced",
		"report_mode":            "long_form",
		"agent_model":            "gpt-5.5",
		"agent_reasoning_effort": "high",
	})
	if mode := nestedString(t, response, "pending_event", "Payload", "report_mode"); mode != reportModeLongForm {
		t.Fatalf("expected long-form report mode in pending event, got %q", mode)
	}
	if policy := nestedString(t, response, "pending_event", "Payload", "report_session_policy"); policy != reportSessionPolicySameSession {
		t.Fatalf("expected non-forking long-form report to use same session, got %q", policy)
	}
	if selection := nestedString(t, response, "pending_event", "Payload", "report_session_policy_selection"); selection != reportSessionPolicySelectionAutoSameSessionNoForker {
		t.Fatalf("expected non-forking long-form selection, got %q", selection)
	}
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if countEvents(detail, "report.plan.created") != 1 || countEvents(detail, "report.section.created") != 2 || countEvents(detail, "report.part.created") != 1 {
		t.Fatalf("expected plan, section, and part events, got %#v", detail["events"])
	}
	if len(agent.requests) != 5 {
		t.Fatalf("expected plan, two sections, part assembly, and frame requests, got %d", len(agent.requests))
	}
	for _, request := range agent.requests {
		if request.Model != "gpt-5.5" || request.ReasoningEffort != "high" {
			t.Fatalf("long-form selection not propagated: %#v", request)
		}
	}
	assertReportMCPToolSurface(t, agent.requests[0], plasmamcp.ToolReportPlanSubmit, plasmamcp.ToolSourcesRead)
	assertReportMCPToolSurface(t, agent.requests[1], plasmamcp.ToolSourcesRead)
	assertReportMCPToolSurface(t, agent.requests[2], plasmamcp.ToolSourcesRead)
	assertReportMCPToolSurface(t, agent.requests[3],
		plasmamcp.ToolReportPartAssemblyStart,
		plasmamcp.ToolReportPartAssemblyRead,
		plasmamcp.ToolReportPartAssemblyPatch,
		plasmamcp.ToolReportPartAssemblySubmit,
	)
	assertReportMCPToolSurface(t, agent.requests[4], plasmamcp.ToolReportLongFormFinalize)
	for _, eventType := range []string{"report.plan.created", "report.section.created", "report.part.created", "report.artifact.created"} {
		selectionPayload := lastEventPayload(t, detail, eventType)
		if selectionPayload["agent_model"] != "gpt-5.5" || selectionPayload["agent_reasoning_effort"] != "high" || selectionPayload["agent_selection_source"] != reporting.AgentSelectionSourceExplicitRequest {
			t.Fatalf("%s frozen selection mismatch: %#v", eventType, selectionPayload)
		}
	}
	if strings.Contains(agent.requests[0].Prompt, "Report writing guidance:") {
		t.Fatalf("report planning prompt must not include generation writing guidance:\n%s", agent.requests[0].Prompt)
	}
	if strings.Contains(agent.requests[0].Prompt, "Long-form human-writer guidance:") {
		t.Fatalf("report planning prompt must not include long-form writing guidance:\n%s", agent.requests[0].Prompt)
	}
	for index := 1; index < len(agent.requests); index++ {
		if !strings.Contains(agent.requests[index].Prompt, "Report writing guidance:") ||
			!strings.Contains(agent.requests[index].Prompt, "use only \\(...\\) for inline math and \\[...\\] for display math") ||
			!strings.Contains(agent.requests[index].Prompt, "Long-form human-writer guidance:") ||
			!strings.Contains(agent.requests[index].Prompt, "not as a system reporting that it inspected a session") {
			t.Fatalf("expected writing request %d to include generation guidance:\n%s", index, agent.requests[index].Prompt)
		}
	}
	if !strings.Contains(agent.requests[3].Prompt, "Section bodies are immutable") || !strings.Contains(agent.requests[4].Prompt, "will be mechanically preserved") {
		t.Fatalf("expected preservation prompts, got part prompt:\n%s\nframe prompt:\n%s", agent.requests[3].Prompt, agent.requests[4].Prompt)
	}
	payload := lastEventPayload(t, detail, "report.artifact.created")
	if payload["report_mode"] != reportModeLongForm ||
		payload["composition_strategy"] != "sectional_preserve_markdown" ||
		payload["assembly_strategy"] != "c4_normalized_section_headings" {
		t.Fatalf("expected sectional long-form payload, got %#v", payload)
	}
	if payload["report_session_policy"] != reportSessionPolicySameSession ||
		payload["report_session_policy_selection"] != reportSessionPolicySelectionAutoSameSessionNoForker ||
		payload["session_chain_kind"] != "same_session_report" {
		t.Fatalf("expected same-session long-form report metadata, got %#v", payload)
	}
	if payload["returned_agent_session_id"] != "" || payload["agent_usage"] != nil {
		t.Fatalf("canonical must contain only bound session provenance: %#v", payload)
	}
	if countEvents(detail, "turn.agent.response") != 0 {
		t.Fatalf("long-form finalization must not add conversation telemetry: %#v", detail["events"])
	}
	artifact, err := svc.GetRawArtifact(ctx, payload["artifact_id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	content := string(artifact.Content)
	for _, expected := range []string{sectionOne, sectionTwo, "두 섹션을 이어주는 전환문입니다.", "## 결론"} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected final Markdown to preserve %q:\n%s", expected, content)
		}
	}
	if strings.Contains(content, "sectional long-form") {
		t.Fatalf("final Markdown leaked implementation wording:\n%s", content)
	}
}

func TestRunPartAssemblyAgentUsesMCPToolsForPartAssemblyEditProfile(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: "mis_part_assembly", Title: "Part assembly test"}); err != nil {
		t.Fatal(err)
	}
	server := NewServer(svc, Options{}).(*Server)
	agent := &fakeAgentExecutor{
		responses: []AgentResult{{Text: reporting.PartAssemblySubmittedSentinel, SessionID: "report-session-1"}},
		onRun: func(runCtx context.Context, req AgentRequest) {
			if req.PartAssembly == nil {
				t.Fatalf("expected part assembly binding on agent request: %#v", req)
			}
			assertReportMCPToolSurface(t, req,
				plasmamcp.ToolReportPartAssemblyStart,
				plasmamcp.ToolReportPartAssemblyRead,
				plasmamcp.ToolReportPartAssemblyPatch,
				plasmamcp.ToolReportPartAssemblySubmit,
			)
			for _, expected := range []string{
				plasmamcp.ToolReportPartAssemblyStart,
				plasmamcp.ToolReportPartAssemblyRead,
				plasmamcp.ToolReportPartAssemblyPatch,
				plasmamcp.ToolReportPartAssemblySubmit,
			} {
				if !slices.Contains(req.ExtraMCPTools, expected) {
					t.Fatalf("expected %s in part assembly tools: %#v", expected, req.ExtraMCPTools)
				}
			}
			if _, err := svc.AppendEvent(runCtx, reporting.BuildPartAssemblySubmittedAppendRequest(reporting.PartAssemblySubmittedEventRequest{
				EventID: "evt_part_assembly_submitted",
				Binding: *req.PartAssembly,
				Assembly: reporting.PartAssembly{
					Intro: "파트 도입입니다.",
					Transitions: []reporting.PartTransition{{
						AfterSectionIndex: 1,
						Markdown:          "섹션 전환입니다.",
					}},
					Closing: "파트 마무리입니다.",
				},
			})); err != nil {
				t.Fatal(err)
			}
		},
	}
	assembly, result, returnedSessionID, err := server.runPartAssemblyAgent(ctx, reportPartAssemblyAgentRequest{
		title:                     "Report",
		missionID:                 "mis_part_assembly",
		toolSessionID:             "ses_part_assembly",
		previousSessionID:         "report-session-1",
		pendingEventID:            "evt_pending",
		planEventID:               "evt_plan",
		executorName:              "codex",
		agentModel:                "gpt-5.5",
		agentReasoningEffort:      "medium",
		agentSelectionSource:      reporting.AgentSelectionSourceExplicitRequest,
		mcpMode:                   "auto",
		rigor:                     reportRigorProfiles["balanced"],
		plan:                      agentSectionalReportPlan{Summary: "Plan"},
		part:                      agentReportPart{Title: "Part", Sections: []agentReportSection{{Title: "First"}, {Title: "Second"}}},
		drafts:                    []sectionalReportDraft{{Title: "First", Markdown: "첫 섹션 본문입니다.", WordCount: 2}, {Title: "Second", Markdown: "둘째 섹션 본문입니다.", WordCount: 2}},
		generationGuidanceProfile: reportGenerationGuidanceProfilePartAssemblyEditTools,
	}, agent)
	if err != nil {
		t.Fatal(err)
	}
	if result.SessionID != "report-session-1" || returnedSessionID != "report-session-1" {
		t.Fatalf("unexpected session ids: result=%q returned=%q", result.SessionID, returnedSessionID)
	}
	if assembly.Intro != "파트 도입입니다." || assembly.Closing != "파트 마무리입니다." || len(assembly.Transitions) != 1 {
		t.Fatalf("expected submitted MCP connective tissue, got %#v", assembly)
	}
	if len(agent.requests) != 1 || !strings.Contains(agent.requests[0].Prompt, "Required tool sequence:") {
		t.Fatalf("expected edit-tools part assembly prompt, got %#v", agent.requests)
	}
}

func TestReportDraftLongFormCanonicalSurvivesAcknowledgementAnomaly(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	delegate := &fakeAgentExecutor{responses: []AgentResult{
		{Text: agentReportAnyJSON(agentSectionalReportPlan{Summary: "Plan", Parts: []agentReportPart{{Title: "Part", Sections: []agentReportSection{{Title: "Section"}}}}}), SessionID: "report-session-1"},
		{Text: "Section body.", SessionID: "report-session-1"},
		{Text: `{"intro":"Intro","transitions":[],"closing":"Close"}`, SessionID: "report-session-1"},
		{Text: `{"front_matter":"# Report","closing":"## Close"}`, SessionID: "report-session-1"},
		{Text: `{"front_matter":"# Report","closing":"## Close"}`, SessionID: "report-session-1"},
	}}
	executor := &ackAnomalyExecutor{delegate: withReportPlanSubmissionFixture(svc, delegate)}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: executor}))
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Ack anomaly"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{"title": "Report", "report_mode": "long_form", "post_report_humanize": "disabled"})
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	time.Sleep(50 * time.Millisecond)
	detail = getJSON(t, server.URL+"/api/missions/"+missionID)
	if countEvents(detail, "report.artifact.created") != 1 || countEvents(detail, "report.draft.failed") != 0 || executor.finalCalls != 2 {
		t.Fatalf("canonical acknowledgement anomaly must remain successful: calls=%d events=%#v", executor.finalCalls, detail["events"])
	}
	canonical := lastEventPayload(t, detail, "report.artifact.created")
	if canonical["agent_session_id"] != "report-session-1" || canonical["returned_agent_session_id"] != "" || countEvents(detail, "turn.agent.response") != 0 {
		t.Fatalf("bound canonical provenance must remain separate without conversation telemetry: canonical=%#v events=%#v", canonical, detail["events"])
	}
}

func TestLongFormFinalObservationLogIsRedacted(t *testing.T) {
	var output bytes.Buffer
	previousWriter, previousFlags, previousPrefix := log.Writer(), log.Flags(), log.Prefix()
	log.SetOutput(&output)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(previousWriter)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
	})
	usage := agentusage.New("openai", "codex", "model", "high", "sensitive prompt").WithProviderUsage(agentusage.ProviderUsage{InputTokens: 11, OutputTokens: 7}, "fixture")
	logLongFormFinalObservation("mis_redacted", "evt_pending", "evt_plan", 2, "sensitive-bound-session", AgentResult{
		Text: "sensitive report body", SessionID: "sensitive-returned-session", Log: "sensitive provider response", Usage: usage,
	}, 25)
	logged := output.String()
	for _, forbidden := range []string{"sensitive-bound-session", "sensitive-returned-session", "sensitive report body", "sensitive provider response", "sensitive prompt"} {
		if strings.Contains(logged, forbidden) {
			t.Fatalf("redacted final observation leaked %q: %s", forbidden, logged)
		}
	}
	for _, expected := range []string{"returned_session_present=true", "returned_session_matches_bound=false", "usage_available=true", "input_tokens=11", "output_tokens=7", "total_tokens=18", "duration_ms=25"} {
		if !strings.Contains(logged, expected) {
			t.Fatalf("redacted final observation missing %q: %s", expected, logged)
		}
	}
}

func TestWebWorkflowStartStatusStop(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	release := make(chan struct{})
	svc := app.NewService(store)
	agent := &sequenceBlockingAgentExecutor{
		release: release,
		responses: []AgentResult{{
			Text:      "workflow step result\nPLASMA_WORKFLOW_CONTROL: {\"decision\":\"continue\",\"reason\":\"more\"}",
			SessionID: "agent-session-1",
		}},
	}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Workflow stop"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	start := postJSON(t, server.URL+"/api/missions/"+missionID+"/workflows", map[string]any{
		"step_instruction_mode": "layered",
		"user_instruction_raw":  "다각도로 조사",
		"run_goal":              "넓게 가능성을 확인한다.",
		"instruction":           "Run one workflow step",
		"agent_executor":        "codex",
		"mcp_mode":              "auto",
		"max_steps":             3,
		"max_duration_ms":       60000,
	})
	runID := nestedString(t, start, "workflow_run", "workflow_run_id")
	if raw := nestedString(t, start, "workflow_run", "user_instruction_raw"); raw != "다각도로 조사" {
		t.Fatalf("expected raw workflow instruction in response, got %q", raw)
	}
	if goal := nestedString(t, start, "workflow_run", "run_goal"); goal != "넓게 가능성을 확인한다." {
		t.Fatalf("expected workflow goal in response, got %q", goal)
	}
	if mode := nestedString(t, start, "workflow_run", "step_instruction_mode"); mode != "layered" {
		t.Fatalf("expected layered step instruction mode in response, got %q", mode)
	}
	detail := waitForEventType(t, server.URL, missionID, app.WorkflowStepStartedEvent)
	if status := workflowRunStatus(t, detail, runID); status != app.WorkflowStatusRunning {
		t.Fatalf("expected running workflow, got %q", status)
	}
	var request AgentRequest
	for i := 0; i < 50; i++ {
		agent.mu.Lock()
		if len(agent.requests) > 0 {
			request = agent.requests[0]
			agent.mu.Unlock()
			break
		}
		agent.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(request.Prompt, "다각도로 조사") || !strings.Contains(request.Prompt, "넓게 가능성을 확인한다.") {
		t.Fatalf("expected workflow prompt to carry raw request and goal, got %#v", request)
	}

	stop := postJSON(t, server.URL+"/api/missions/"+missionID+"/workflows/"+runID+"/stop", map[string]any{})
	if status := nestedString(t, stop, "workflow_run", "status"); status != app.WorkflowStatusStopped {
		t.Fatalf("expected stopped workflow after stop request, got %q", status)
	}
	detail = waitForEventType(t, server.URL, missionID, app.WorkflowRunStoppedEvent)
	if status := workflowRunStatus(t, detail, runID); status != app.WorkflowStatusStopped {
		t.Fatalf("expected stopped workflow, got %q", status)
	}
	if hasOpenPendingDetail(detail) {
		t.Fatal("expected workflow stop to close the open agent pending event")
	}
}

func TestCancelWorkflowTurnStopsWorkflowRun(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	release := make(chan struct{})
	agent := &sequenceBlockingAgentExecutor{
		release: release,
		responses: []AgentResult{{
			Text:      "workflow step result\nPLASMA_WORKFLOW_CONTROL: {\"decision\":\"continue\",\"reason\":\"more\"}",
			SessionID: "agent-session-1",
		}},
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Workflow turn cancel"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	start := postJSON(t, server.URL+"/api/missions/"+missionID+"/workflows", map[string]any{
		"instruction":     "Run a cancelable workflow step",
		"agent_executor":  "codex",
		"mcp_mode":        "auto",
		"max_steps":       3,
		"max_duration_ms": 60000,
	})
	runID := nestedString(t, start, "workflow_run", "workflow_run_id")
	waitForEventType(t, server.URL, missionID, app.WorkflowStepStartedEvent)

	cancel := postJSON(t, server.URL+"/api/missions/"+missionID+"/turns/cancel", map[string]any{"agent_executor": "codex"})
	if canceled, _ := cancel["canceled"].(bool); !canceled {
		t.Fatalf("expected workflow turn cancellation response, got %#v", cancel)
	}
	detail := waitForEventType(t, server.URL, missionID, app.WorkflowRunStoppedEvent)
	if status := workflowRunStatus(t, detail, runID); status != app.WorkflowStatusStopped {
		t.Fatalf("expected canceled workflow turn to stop workflow run, got %q", status)
	}
	payload := firstEventPayload(t, detail, "turn.agent.response")
	if payload["kind"] != "agent_canceled" || payload["workflow_run_id"] != runID {
		t.Fatalf("expected workflow agent_canceled response, got %#v", payload)
	}
	if hasOpenPendingDetail(detail) {
		t.Fatal("expected workflow turn cancel to close the open agent pending event")
	}
}

func TestWorkflowReconcileStopsCanceledPendingRun(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: &fakeAgentExecutor{}}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Workflow repair"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	run, err := svc.RequestWorkflowRun(ctx, app.RequestWorkflowRunRequest{
		MissionID:          missionID,
		RequestedBySurface: app.WorkflowSurfaceWeb,
		AgentExecutor:      "codex",
		MCPMode:            "auto",
		Instruction:        "Repair old stuck workflow",
		MaxSteps:           3,
	})
	if err != nil {
		t.Fatalf("RequestWorkflowRun returned error: %v", err)
	}
	run, claimed, err := svc.ClaimWorkflowRunStart(ctx, missionID, run.WorkflowRunID, time.Now())
	if err != nil || !claimed {
		t.Fatalf("ClaimWorkflowRunStart returned claimed=%v err=%v", claimed, err)
	}
	stepID := "wfs_repair"
	userEventID := "evt_repair_user"
	if _, err := svc.AppendEvents(ctx, missionID, []app.AppendEventRequest{
		{
			EventID:   "evt_repair_step_started",
			MissionID: missionID,
			EventType: app.WorkflowStepStartedEvent,
			Producer:  app.Producer{Type: "workflow", ID: run.WorkflowRunID},
			Payload: mustJSON(app.WorkflowStepStartedPayload{
				WorkflowRunID:  run.WorkflowRunID,
				MissionID:      missionID,
				WorkflowStepID: stepID,
				Instruction:    "Repair old stuck workflow",
				StepIndex:      1,
				StartedAt:      time.Now().UTC().Format(time.RFC3339Nano),
				ToolSessionID:  "ses_repair",
			}),
		},
		{
			EventID:   userEventID,
			MissionID: missionID,
			EventType: "turn.user",
			Producer:  app.Producer{Type: "user", ID: "test"},
			Payload: mustJSON(map[string]any{
				"kind":             "workflow_step_user",
				"text":             "Repair old stuck workflow",
				"workflow_run_id":  run.WorkflowRunID,
				"workflow_step_id": stepID,
			}),
		},
		{
			EventID:   "evt_repair_pending",
			MissionID: missionID,
			EventType: "turn.agent.pending",
			Producer:  app.Producer{Type: "agent", ID: "codex"},
			Payload: mustJSON(map[string]any{
				"kind":             "agent_pending",
				"agent_executor":   "codex",
				"text":             "워크플로우 단계의 에이전트 응답을 기다리는 중입니다.",
				"user_event_id":    userEventID,
				"workflow_run_id":  run.WorkflowRunID,
				"workflow_step_id": stepID,
				"started_at":       time.Now().UTC().Format(time.RFC3339Nano),
			}),
		},
		{
			EventID:   "evt_repair_agent_canceled",
			MissionID: missionID,
			EventType: "turn.agent.response",
			Producer:  app.Producer{Type: "agent", ID: "codex"},
			Payload: mustJSON(map[string]any{
				"kind":             "agent_canceled",
				"agent_executor":   "codex",
				"text":             "브라우저에서 끊긴 오래된 대기 상태를 취소했습니다.",
				"user_event_id":    userEventID,
				"workflow_run_id":  run.WorkflowRunID,
				"workflow_step_id": stepID,
				"canceled_at":      time.Now().UTC().Format(time.RFC3339Nano),
			}),
		},
	}); err != nil {
		t.Fatalf("AppendEvents returned error: %v", err)
	}
	if _, err := svc.RequestWorkflowStop(ctx, app.RequestWorkflowStopRequest{
		WorkflowRunID:      run.WorkflowRunID,
		MissionID:          missionID,
		RequestedBySurface: app.WorkflowSurfaceWeb,
		Reason:             "사용자가 웹에서 워크플로우 정지를 요청했습니다.",
	}); err != nil {
		t.Fatalf("RequestWorkflowStop returned error: %v", err)
	}
	server.Close()
	server = httptest.NewServer(NewServer(svc, Options{AgentExecutor: &fakeAgentExecutor{}}))
	defer server.Close()
	runs, err := svc.ListWorkflowRuns(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	status := ""
	for _, view := range runs {
		if view.WorkflowRunID == run.WorkflowRunID {
			status = view.Status
			break
		}
	}
	if status != app.WorkflowStatusStopping {
		t.Fatalf("server construction reconciled workflow before detail GET, got %q", status)
	}
	activity := getJSON(t, server.URL+"/api/missions/"+missionID+"/activity")
	if nestedMap(t, activity, "activity")["last_sequence"] == nil {
		t.Fatalf("activity response missing summary: %#v", activity)
	}
	runs, err = svc.ListWorkflowRuns(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	for _, view := range runs {
		if view.WorkflowRunID == run.WorkflowRunID && view.Status != app.WorkflowStatusStopping {
			t.Fatalf("activity read reconciled workflow before detail GET, got %q", view.Status)
		}
	}

	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	if status := workflowRunStatus(t, detail, run.WorkflowRunID); status != app.WorkflowStatusStopped {
		t.Fatalf("expected reconcile to stop old canceled-pending workflow, got %q", status)
	}
	if countEvents(detail, app.WorkflowRunStoppedEvent) != 1 {
		t.Fatalf("expected one workflow stopped event, got %#v", detail["events"])
	}
	if hasOpenPendingDetail(detail) {
		t.Fatal("expected repaired workflow to have no open pending event")
	}
}

func TestStaleAgentTurnAutoClosesBeforeWorkflowStart(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{responses: []AgentResult{{
		Text:      "workflow step result\nPLASMA_WORKFLOW_CONTROL: {\"decision\":\"stop\",\"reason\":\"done\"}",
		SessionID: "agent-session-workflow",
	}}}
	svc := app.NewService(store)
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Workflow stale agent turn"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	appendStaleAgentPending(t, ctx, svc, missionID, "evt_stale_workflow_user", "evt_stale_workflow_pending")

	start := postJSON(t, server.URL+"/api/missions/"+missionID+"/workflows", map[string]any{
		"instruction":    "Run workflow after stale cleanup",
		"agent_executor": "codex",
		"mcp_mode":       "auto",
		"max_steps":      1,
	})
	if runID := nestedString(t, start, "workflow_run", "workflow_run_id"); runID == "" {
		t.Fatalf("expected workflow to start after stale cleanup, got %#v", start)
	}
	detail := waitForEventType(t, server.URL, missionID, app.WorkflowRunCompletedEvent)
	payload := firstEventPayload(t, detail, "turn.agent.response")
	if payload["kind"] != "agent_canceled" || payload["user_event_id"] != "evt_stale_workflow_user" {
		t.Fatalf("expected stale agent turn to be auto-canceled before workflow start, got %#v", payload)
	}
	if len(agent.requests) != 1 {
		t.Fatalf("expected workflow to call agent once after stale cleanup, got %#v", agent.requests)
	}
}

func TestWebWorkflowGoalDraftUsesConfiguredDraftModel(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{responses: []AgentResult{{
		Text:      `{"run_goal":"원문을 넓게 보존한 목표","step_instruction":"첫 번째로 기존 소스를 훑는다"}`,
		SessionID: "draft-session-1",
	}}}
	agent.onRun = func(ctx context.Context, _ AgentRequest) {
		if deadline, ok := ctx.Deadline(); ok {
			t.Errorf("workflow goal draft must not inherit a deadline, got %v", deadline)
		}
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent, WorkflowGoalReasoningEffort: "low"}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{
		"title":     "Workflow goal draft",
		"objective": "자율진행 목표 초안 테스트",
	})
	missionID := nestedString(t, mission, "projection", "mission_id")
	response := postJSON(t, server.URL+"/api/missions/"+missionID+"/workflows/goal_draft", map[string]any{
		"user_instruction_raw": "다각도로 조사해줘",
		"agent_executor":       "codex",
	})
	if got := nestedString(t, response, "workflow_goal_draft", "user_instruction_raw"); got != "다각도로 조사해줘" {
		t.Fatalf("expected raw instruction echo, got %q", got)
	}
	if got := nestedString(t, response, "workflow_goal_draft", "run_goal"); got != "원문을 넓게 보존한 목표" {
		t.Fatalf("expected run goal, got %q", got)
	}
	if got := nestedString(t, response, "workflow_goal_draft", "step_instruction"); got != "첫 번째로 기존 소스를 훑는다" {
		t.Fatalf("expected step instruction, got %q", got)
	}
	if got := nestedString(t, response, "workflow_goal_draft", "reasoning_effort"); got != "low" {
		t.Fatalf("expected reasoning effort in response, got %q", got)
	}
	if len(agent.requests) != 1 {
		t.Fatalf("expected one draft request, got %d", len(agent.requests))
	}
	req := agent.requests[0]
	if req.Model != "" || req.ReasoningEffort != "low" || req.MissionID != missionID || !strings.Contains(req.Prompt, "Do not research the topic") || !strings.Contains(req.Prompt, "다각도로 조사해줘") {
		t.Fatalf("unexpected draft agent request: %#v", req)
	}
}

func TestWebSettingsModelDefaultsRoundTrip(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := httptest.NewServer(NewServer(app.NewService(store), Options{WorkflowGoalReasoningEffort: "low"}))
	defer server.Close()

	initial := getJSON(t, server.URL+"/api/settings/model-defaults")
	if got := nestedString(t, initial, "effective", "workflow_goal_reasoning_effort"); got != "low" {
		t.Fatalf("expected server config fallback, got %q", got)
	}
	saved := patchJSON(t, server.URL+"/api/settings/model-defaults", map[string]any{
		"workflow_goal_model":            "gpt-5.5",
		"workflow_goal_reasoning_effort": "high",
	})
	if got := nestedString(t, saved, "model_defaults", "workflow_goal_model"); got != "gpt-5.5" {
		t.Fatalf("expected saved workflow goal model, got %q", got)
	}
	if got := nestedString(t, saved, "model_defaults", "workflow_goal_reasoning_effort"); got != "high" {
		t.Fatalf("expected saved workflow goal effort, got %q", got)
	}
	if got := nestedString(t, saved, "display", "workflow_goal_source"); got != "settings" {
		t.Fatalf("expected settings source, got %q", got)
	}
	status, failure := patchJSONFailure(t, server.URL+"/api/settings/model-defaults", map[string]any{
		"workflow_goal_model":            "gpt-5.6-luna",
		"workflow_goal_reasoning_effort": "ultra",
	})
	if status != http.StatusBadRequest || !strings.Contains(nestedString(t, failure, "error", "message"), "unsupported reasoning effort") {
		t.Fatalf("expected invalid model effort response, got %d %#v", status, failure)
	}
}

func TestWebWorkflowGoalDraftUsesSavedModelDefaults(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := app.NewService(store).SaveModelDefaults(ctx, app.ModelDefaults{
		WorkflowGoalModel:           "gpt-5.5",
		WorkflowGoalReasoningEffort: "high",
	}); err != nil {
		t.Fatal(err)
	}
	agent := &fakeAgentExecutor{responses: []AgentResult{{
		Text:      `{"run_goal":"저장 설정 기반 목표","step_instruction":"저장 설정을 쓴다"}`,
		SessionID: "draft-session-settings",
	}}}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{
		AgentExecutor:               agent,
		WorkflowGoalReasoningEffort: "low",
	}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Workflow goal settings"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	response := postJSON(t, server.URL+"/api/missions/"+missionID+"/workflows/goal_draft", map[string]any{
		"user_instruction_raw": "설정 모델로 초안",
		"agent_executor":       "codex",
	})
	if got := nestedString(t, response, "workflow_goal_draft", "model"); got != "gpt-5.5" {
		t.Fatalf("expected stored model in response, got %q", got)
	}
	if got := nestedString(t, response, "workflow_goal_draft", "reasoning_effort"); got != "high" {
		t.Fatalf("expected stored effort in response, got %q", got)
	}
	if len(agent.requests) != 1 {
		t.Fatalf("expected one draft request, got %d", len(agent.requests))
	}
	if got := agent.requests[0].Model; got != "gpt-5.5" {
		t.Fatalf("expected stored model in request, got %q", got)
	}
	if got := agent.requests[0].ReasoningEffort; got != "high" {
		t.Fatalf("expected stored effort in request, got %q", got)
	}
}

func TestNormalConversationHasNoDeadlineWhenAgentTimeoutIsUnset(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	agent := &fakeAgentExecutor{responses: []AgentResult{{Text: "answer", SessionID: "session-1"}}}
	agent.onRun = func(runCtx context.Context, _ AgentRequest) {
		if deadline, ok := runCtx.Deadline(); ok {
			t.Errorf("normal conversation must not inherit a deadline, got %v", deadline)
		}
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "No deadline conversation"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "hello"})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	if len(agent.requests) != 1 {
		t.Fatalf("expected one agent request, got %#v", agent.requests)
	}
}

func TestWebWorkflowGoalDraftRejectsOversizedInstruction(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Workflow goal draft limit"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/workflows/goal_draft", map[string]any{
		"user_instruction_raw": strings.Repeat("x", app.WorkflowInstructionLimit+1),
		"agent_executor":       "codex",
	})
	if status != http.StatusBadRequest || !strings.Contains(nestedString(t, body, "error", "message"), "longer than") {
		t.Fatalf("expected oversized draft request rejection, got %d %#v", status, body)
	}
	if len(agent.requests) != 0 {
		t.Fatalf("oversized draft request should not call agent, got %#v", agent.requests)
	}
}

func TestWebWorkflowBlocksAgentSessionReset(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	release := make(chan struct{})
	agent := &sequenceBlockingAgentExecutor{
		release: release,
		responses: []AgentResult{{
			Text:      "workflow step result\nPLASMA_WORKFLOW_CONTROL: {\"decision\":\"continue\",\"reason\":\"more\"}",
			SessionID: "agent-session-1",
		}},
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Workflow reset conflict"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/workflows", map[string]any{
		"instruction": "Run workflow before reset",
		"max_steps":   2,
	})
	waitForEventType(t, server.URL, missionID, app.WorkflowStepStartedEvent)

	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/agent_sessions/reset", map[string]any{"agent_executor": "codex"})
	if status != http.StatusConflict {
		t.Fatalf("expected reset rejection while workflow active, got %d %#v", status, body)
	}
	close(release)
}

func TestWebWorkflowStartQueuesWhileTurnPendingThenDrains(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	release := make(chan struct{})
	agent := &sequenceBlockingAgentExecutor{
		release: release,
		responses: []AgentResult{
			{Text: "normal answer", SessionID: "agent-session-1"},
			{Text: "workflow result\nPLASMA_WORKFLOW_CONTROL: {\"decision\":\"stop\",\"reason\":\"done\"}", SessionID: "agent-session-1"},
		},
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Workflow queued"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "hold the session"})
	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	if countEvents(detail, "turn.agent.pending") != 1 {
		t.Fatalf("expected pending turn, got %#v", detail["events"])
	}
	start := postJSON(t, server.URL+"/api/missions/"+missionID+"/workflows", map[string]any{
		"instruction": "Continue after current turn",
		"max_steps":   1,
	})
	runID := nestedString(t, start, "workflow_run", "workflow_run_id")
	if status := nestedString(t, start, "workflow_run", "status"); status != app.WorkflowStatusQueued {
		t.Fatalf("expected queued workflow, got %q", status)
	}
	detail = getJSON(t, server.URL+"/api/missions/"+missionID)
	if countEvents(detail, app.WorkflowRunStartedEvent) != 0 {
		t.Fatal("queued workflow should not start before current turn terminal event")
	}

	close(release)
	detail = waitForEventType(t, server.URL, missionID, app.WorkflowRunCompletedEvent)
	if status := workflowRunStatus(t, detail, runID); status != app.WorkflowStatusCompleted {
		t.Fatalf("expected drained workflow completion, got %q", status)
	}
	if len(agent.requests) < 2 || agent.requests[1].PreviousSessionID != "agent-session-1" {
		t.Fatalf("expected workflow to resume same provider session, requests=%#v", agent.requests)
	}
}

func TestWebWorkflowRejectsDuplicateStartWhileQueued(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	release := make(chan struct{})
	agent := &sequenceBlockingAgentExecutor{
		release: release,
		responses: []AgentResult{
			{Text: "normal answer", SessionID: "agent-session-1"},
			{Text: "workflow result\nPLASMA_WORKFLOW_CONTROL: {\"decision\":\"stop\",\"reason\":\"done\"}", SessionID: "agent-session-1"},
		},
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Workflow duplicate queue"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "hold the session"})
	postJSON(t, server.URL+"/api/missions/"+missionID+"/workflows", map[string]any{
		"instruction": "Continue after current turn",
		"max_steps":   1,
	})

	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/workflows", map[string]any{
		"instruction": "Start a second queued workflow",
		"max_steps":   1,
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected duplicate queued workflow rejection, got %d %#v", status, body)
	}
	close(release)
}

func TestWebWorkflowRejectsNormalTurnWhileQueued(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: &sequenceBlockingAgentExecutor{release: make(chan struct{})}}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Workflow queued conflict"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_user_queued",
		MissionID: missionID,
		EventType: "turn.user",
		Producer:  app.Producer{Type: "user", ID: "test"},
		Payload:   mustJSON(map[string]any{"kind": "user_turn", "text": "hold"}),
	}); err != nil {
		t.Fatalf("append turn.user returned error: %v", err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_pending_queued",
		MissionID: missionID,
		EventType: "turn.agent.pending",
		Producer:  app.Producer{Type: "agent", ID: "codex"},
		Payload:   mustJSON(map[string]any{"user_event_id": "evt_user_queued", "agent_executor": "codex"}),
	}); err != nil {
		t.Fatalf("append turn.agent.pending returned error: %v", err)
	}
	if _, err := svc.RequestWorkflowRun(ctx, app.RequestWorkflowRunRequest{
		MissionID:          missionID,
		RequestedBySurface: app.WorkflowSurfaceMCP,
		AgentExecutor:      "codex",
		MCPMode:            "auto",
		Instruction:        "Queued external workflow",
		MaxSteps:           1,
		StartAfterEventID:  "evt_user_queued",
	}); err != nil {
		t.Fatalf("RequestWorkflowRun returned error: %v", err)
	}

	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "normal turn"})
	if status != http.StatusConflict {
		t.Fatalf("expected queued workflow conflict, got %d %#v", status, body)
	}
}

func TestWebWorkflowRejectsNormalTurnWhileRunning(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	release := make(chan struct{})
	agent := &sequenceBlockingAgentExecutor{
		release: release,
		responses: []AgentResult{{
			Text:      "workflow step result\nPLASMA_WORKFLOW_CONTROL: {\"decision\":\"continue\",\"reason\":\"more\"}",
			SessionID: "agent-session-1",
		}},
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Workflow conflict"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/workflows", map[string]any{
		"instruction": "Run a workflow step",
		"max_steps":   3,
	})
	waitForEventType(t, server.URL, missionID, app.WorkflowStepStartedEvent)
	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "normal turn"})
	if status != http.StatusConflict {
		t.Fatalf("expected conflict, got %d %#v", status, body)
	}
	close(release)
}

func TestWebWorkflowStartRequiresConfiguredAgent(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "No agent workflow"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	status, _ := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/workflows", map[string]any{
		"instruction": "Cannot run",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected validation error, got %d", status)
	}
	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	if countEvents(detail, app.WorkflowRunRequestedEvent) != 0 {
		t.Fatal("workflow should not be recorded when no agent executor is configured")
	}
}

func TestReportDraftCreatesMarkdownArtifactWithoutASTRepair(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	agent := &fakeAgentExecutor{}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{
		"title":     "Report repair test",
		"objective": "Ensure report refs are repaired before save",
	})
	missionID := nestedString(t, mission, "projection", "mission_id")
	source := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/text", map[string]any{
		"title":        "Approved source",
		"external_uri": "https://example.com/approved",
		"content":      "Approved report evidence.",
	})
	snapshotID := nestedString(t, source, "snapshot", "SnapshotID")
	artifactID := nestedString(t, source, "artifact", "ArtifactID")
	proposal := postJSON(t, server.URL+"/api/missions/"+missionID+"/candidates/evidence", map[string]any{
		"summary":       "Approved report evidence.",
		"snapshot_id":   snapshotID,
		"artifact_id":   artifactID,
		"evidence_type": "fact",
	})
	evidenceProposalID := nestedString(t, proposal, "Proposal", "proposal_id")
	evidenceID := nestedString(t, proposal, "Evidence", "evidence_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/proposals/"+evidenceProposalID+"/approve", map[string]any{})

	createClaimForReportRepairTest(t, ctx, svc, missionID, "clm_report_repair_approved", "prp_report_repair_claim_ok", "evt_report_repair_claim_ok", "Approved report claim.", []string{evidenceID})
	postJSON(t, server.URL+"/api/missions/"+missionID+"/proposals/prp_report_repair_claim_ok/approve", map[string]any{})
	createClaimForReportRepairTest(t, ctx, svc, missionID, "clm_report_repair_proposed", "prp_report_repair_claim_pending", "evt_report_repair_claim_pending", "Proposed report claim.", []string{evidenceID})

	agent.responses = []AgentResult{
		{Text: agentReportPlanJSON(agentReportPlan{
			Summary: "Use approved source material for the repair report.",
			Sections: []agentReportSection{{
				Title:   "Approved source",
				Purpose: "Cover the approved evidence without exposing internal IDs.",
			}},
		}), SessionID: "report-session-1"},
		{Text: "# Repair report\n\nThis is a Markdown report based on source reads.", SessionID: "report-session-1"},
	}

	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":       "Repair report",
		"rigor_level": "balanced",
		"report_mode": "planned",
	})
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if countEvents(detail, "report.repair.requested") != 0 || countEvents(detail, "report.drafted") != 0 {
		t.Fatalf("default report path must not create AST repair/draft events, got %#v", detail["events"])
	}
	if countEvents(detail, "report.plan.created") != 1 {
		t.Fatalf("expected one markdown report plan event, got %#v", detail["events"])
	}
	if len(agent.requests) != 2 {
		t.Fatalf("expected plan and markdown report requests, got %d", len(agent.requests))
	}
	if agent.requests[0].PreviousSessionID != "" {
		t.Fatalf("first report may start a new session when no mission session exists, got %q", agent.requests[0].PreviousSessionID)
	}
	if agent.requests[1].PreviousSessionID != "report-session-1" {
		t.Fatalf("markdown report should continue planning session, got %q", agent.requests[1].PreviousSessionID)
	}
	payload := lastEventPayload(t, detail, "report.artifact.created")
	artifact, err := svc.GetRawArtifact(ctx, payload["artifact_id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if artifact.MediaType != "text/markdown; charset=utf-8" || strings.Contains(string(artifact.Content), "clm_report_repair_") || strings.Contains(string(artifact.Content), evidenceID) {
		t.Fatalf("markdown artifact should not force internal evidence/claim ids as public citations: %#v", artifact)
	}
}

func TestReportScopeIDsHonorPartiallyApprovedProposalDecision(t *testing.T) {
	events := []app.LedgerEvent{{
		EventID:   "evt_partial_decision",
		EventType: "proposal.partially_approved",
		Producer:  app.Producer{Type: "user", ID: "ses_user"},
		Payload: mustJSON(map[string]any{
			"proposal_id":         "prp_partial",
			"approved_object_ids": []string{"evd_ok", "clm_ok"},
			"rejected_object_ids": []string{"evd_rejected", "clm_rejected"},
		}),
	}, {
		EventID:   "evt_partial_stray",
		EventType: "proposal.partially_approved",
		Producer:  app.Producer{Type: "user", ID: "ses_user"},
		Payload: mustJSON(map[string]any{
			"proposal_id":         "prp_partial",
			"approved_object_ids": []string{"evd_rejected", "clm_rejected"},
			"rejected_object_ids": []string{"evd_ok", "clm_ok"},
		}),
	}}
	records := recordsResponse{
		Evidence: []app.EvidenceRecord{
			{EvidenceID: "evd_ok"},
			{EvidenceID: "evd_rejected"},
		},
		Claims: []app.ClaimRecord{
			{ClaimID: "clm_ok"},
			{ClaimID: "clm_rejected"},
		},
		Proposals: []app.ProposalBundle{{
			ProposalID:      "prp_partial",
			State:           "partially_approved",
			DecisionEventID: "evt_partial_decision",
			ObjectRefs: []app.ObjectRef{
				{ObjectKind: app.EvidenceRecordObjectKind, ObjectID: "evd_ok"},
				{ObjectKind: app.EvidenceRecordObjectKind, ObjectID: "evd_rejected"},
				{ObjectKind: app.ClaimRecordObjectKind, ObjectID: "clm_ok"},
				{ObjectKind: app.ClaimRecordObjectKind, ObjectID: "clm_rejected"},
			},
		}},
		approvedObjectIDsByDecisionEventID: approvedObjectIDsByDecisionEventID(events),
	}

	evidenceIDs := approvedEvidenceIDs(records)
	if len(evidenceIDs) != 1 || evidenceIDs[0] != "evd_ok" {
		t.Fatalf("expected only approved evidence id, got %#v", evidenceIDs)
	}
	claimIDs := approvedClaimIDs(records)
	if len(claimIDs) != 1 || claimIDs[0] != "clm_ok" {
		t.Fatalf("expected only approved claim id, got %#v", claimIDs)
	}
}

func TestReportDraftReturnsPendingBeforeBackgroundCompletes(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	release := make(chan struct{})
	svc := app.NewService(store)
	agent := &sequenceBlockingAgentExecutor{
		release: release,
		responses: []AgentResult{
			{Text: agentReportPlanJSON(agentReportPlan{
				Summary: "Plan delayed report.",
				Sections: []agentReportSection{{
					Title:   "Pinned note",
					Purpose: "Use pinned evidence for the background report.",
				}},
			}), SessionID: "agent-session-1"},
			{Text: "# Delayed report\n\nThis report was generated in the background.", SessionID: "agent-session-1"},
		},
	}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Report pending test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	source := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/text", map[string]any{
		"title":   "Pinned note",
		"content": "Pinned evidence for the background report.",
	})
	proposal := postJSON(t, server.URL+"/api/missions/"+missionID+"/candidates/evidence", map[string]any{
		"summary":     "Pinned evidence for the background report.",
		"snapshot_id": nestedString(t, source, "snapshot", "SnapshotID"),
		"artifact_id": nestedString(t, source, "artifact", "ArtifactID"),
	})
	postJSON(t, server.URL+"/api/missions/"+missionID+"/proposals/"+nestedString(t, proposal, "Proposal", "proposal_id")+"/approve", map[string]any{})

	response := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{"title": "Delayed report", "report_mode": "planned"})
	if pendingType := nestedString(t, response, "pending_event", "EventType"); pendingType != "report.draft.pending" {
		t.Fatalf("expected report.draft.pending, got %q", pendingType)
	}
	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	if countEvents(detail, "report.draft.pending") != 1 || countEvents(detail, "report.artifact.created") != 0 {
		t.Fatalf("expected only pending report event before release, got %#v", detail["events"])
	}
	if !hasOpenReportDraftDetail(detail) {
		t.Fatalf("expected report draft pending to be open before release")
	}

	close(release)
	detail = waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if hasOpenReportDraftDetail(detail) {
		t.Fatalf("expected report draft pending to close after report artifact creation")
	}
	if countEvents(detail, "report.drafted") != 0 {
		t.Fatalf("default report path must not create AST report draft events, got %#v", detail["events"])
	}
}

func TestReportDraftPendingBlocksNormalTurn(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	release := make(chan struct{})
	server := httptest.NewServer(NewServer(app.NewService(store), Options{
		AgentExecutor: blockingAgentExecutor{
			release: release,
			result:  AgentResult{Text: "# Delayed report", SessionID: "agent-session-1"},
		},
	}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Report turn conflict"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{"title": "Delayed report"})

	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "try a turn"})
	if status != http.StatusConflict {
		t.Fatalf("expected turn rejection while report active, got %d %#v", status, body)
	}
	close(release)
}

func TestReportDraftPendingBlocksWorkflowStart(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	release := make(chan struct{})
	server := httptest.NewServer(NewServer(app.NewService(store), Options{
		AgentExecutor: blockingAgentExecutor{
			release: release,
			result:  AgentResult{Text: "# Delayed report", SessionID: "agent-session-1"},
		},
	}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Report workflow conflict"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{"title": "Delayed report"})

	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/workflows", map[string]any{"instruction": "try workflow"})
	if status != http.StatusBadRequest {
		t.Fatalf("expected workflow rejection while report active, got %d %#v", status, body)
	}
	close(release)
}

func TestReportDraftRejectsWhileWorkflowActive(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	release := make(chan struct{})
	agent := &sequenceBlockingAgentExecutor{
		release: release,
		responses: []AgentResult{{
			Text:      "workflow step result\nPLASMA_WORKFLOW_CONTROL: {\"decision\":\"continue\",\"reason\":\"more\"}",
			SessionID: "agent-session-1",
		}},
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Report workflow conflict"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/workflows", map[string]any{
		"instruction": "Run workflow before reporting",
		"max_steps":   2,
	})
	waitForEventType(t, server.URL, missionID, app.WorkflowStepStartedEvent)

	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{"title": "Blocked report"})
	if status != http.StatusBadRequest {
		t.Fatalf("expected report draft rejection while workflow active, got %d %#v", status, body)
	}
	close(release)
}

func TestManualEvidenceCandidatePreservesSignalType(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Evidence signal type test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	source := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/text", map[string]any{
		"title":   "Audience note",
		"content": "Audience reactions repeatedly mention the same expectation.",
	})
	proposal := postJSON(t, server.URL+"/api/missions/"+missionID+"/candidates/evidence", map[string]any{
		"summary":       "Audience reactions repeatedly mention the same expectation.",
		"evidence_type": "reaction",
		"snapshot_id":   nestedString(t, source, "snapshot", "SnapshotID"),
		"artifact_id":   nestedString(t, source, "artifact", "ArtifactID"),
	})
	if got := nestedString(t, proposal, "Evidence", "evidence_type"); got != "reaction" {
		t.Fatalf("expected reaction evidence type, got %q", got)
	}
}

func TestReportDraftFailureClosesPending(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{
		AgentExecutor: errorAgentExecutor{err: errors.New("report boom")},
	}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Report failure test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	source := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/text", map[string]any{
		"title":   "Pinned note",
		"content": "Pinned evidence for the failed report.",
	})
	proposal := postJSON(t, server.URL+"/api/missions/"+missionID+"/candidates/evidence", map[string]any{
		"summary":     "Pinned evidence for the failed report.",
		"snapshot_id": nestedString(t, source, "snapshot", "SnapshotID"),
		"artifact_id": nestedString(t, source, "artifact", "ArtifactID"),
	})
	postJSON(t, server.URL+"/api/missions/"+missionID+"/proposals/"+nestedString(t, proposal, "Proposal", "proposal_id")+"/approve", map[string]any{})

	response := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{"title": "Failed report"})
	if pendingType := nestedString(t, response, "pending_event", "EventType"); pendingType != "report.draft.pending" {
		t.Fatalf("expected report.draft.pending, got %q", pendingType)
	}
	detail := waitForEventType(t, server.URL, missionID, "report.draft.failed")
	if hasOpenReportDraftDetail(detail) {
		t.Fatalf("expected failed report draft to close pending state")
	}
	payload := lastEventPayload(t, detail, "report.draft.failed")
	if !strings.Contains(payload["error"].(string), "report boom") {
		t.Fatalf("expected report failure error in payload, got %#v", payload)
	}
	versions, _ := detail["report_versions"].([]any)
	if len(versions) != 0 {
		t.Fatalf("expected no report version after failure, got %#v", versions)
	}
}

func TestReportDraftCanceledContextStillClosesPending(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	handler := NewServer(svc, Options{})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Report cancel failure test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	pending, err := appendTestEvent(t, webServer, ctx, missionID, "report.draft.pending", map[string]any{
		"kind":  "report_draft_pending",
		"title": "Canceled report",
		"text":  "리포트 초안 생성 중입니다.",
	}, app.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		t.Fatal(err)
	}

	if err := webServer.reportRunner().RunDraft(context.Background(), missionID, reporting.DraftRequest{Title: "Canceled report"}, pending.EventID); err != nil {
		t.Fatal(err)
	}

	var events []app.LedgerEvent
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		events, err = svc.ListEvents(ctx, missionID)
		if err != nil {
			t.Fatal(err)
		}
		failedCount := 0
		for _, event := range events {
			if event.EventType == "report.draft.failed" {
				failedCount++
			}
		}
		if failedCount > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	var failedPayload map[string]any
	for _, event := range events {
		if event.EventType != "report.draft.failed" {
			continue
		}
		if err := json.Unmarshal(event.Payload, &failedPayload); err != nil {
			t.Fatal(err)
		}
	}
	if failedPayload == nil {
		t.Fatalf("expected report.draft.failed after canceled worker context, got %#v", events)
	}
	if failedPayload["pending_event_id"] != pending.EventID {
		t.Fatalf("expected failed event to close pending %q, got %#v", pending.EventID, failedPayload)
	}
}

func TestCancelReportDraftEndpointClosesPending(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	handler := NewServer(svc, Options{})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Report cancel endpoint test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	pending, err := appendTestEvent(t, webServer, ctx, missionID, "report.draft.pending", map[string]any{
		"kind":           "markdown_report_artifact_pending",
		"title":          "Canceled report",
		"agent_executor": "codex",
		"report_mode":    "long_form",
		"text":           "리포트 초안 생성 중입니다.",
	}, app.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		t.Fatal(err)
	}

	response := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports/cancel", map[string]any{})
	if canceled, ok := response["canceled"].(bool); !ok || !canceled {
		t.Fatalf("expected canceled response, got %#v", response)
	}
	payload := nestedMap(t, response, "event", "Payload")
	if payload["kind"] != "report_draft_canceled" || payload["pending_event_id"] != pending.EventID || payload["canceled"] != true {
		t.Fatalf("expected report_draft_canceled payload for %q, got %#v", pending.EventID, payload)
	}
	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	if hasOpenReportDraftDetail(detail) {
		t.Fatal("expected report draft pending to close after cancel")
	}
}

func TestCancelReportDraftEndpointClosesHumanizePending(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	handler := NewServer(svc, Options{})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Humanize cancel endpoint test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	reportPending, err := appendTestEvent(t, webServer, ctx, missionID, "report.draft.pending", map[string]any{
		"kind":           "markdown_report_artifact_pending",
		"title":          "Canceled report",
		"agent_executor": "codex",
		"report_mode":    "long_form",
		"text":           "리포트 초안 생성 중입니다.",
	}, app.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := appendTestEvent(t, webServer, ctx, missionID, "report.artifact.created", map[string]any{
		"kind":             "markdown_report_artifact",
		"pending_event_id": reportPending.EventID,
		"artifact_id":      "art_report",
		"media_type":       "text/markdown; charset=utf-8",
	}, app.Producer{Type: "agent_session", ID: "report-session-1"}); err != nil {
		t.Fatal(err)
	}
	humanizePending, err := appendTestEvent(t, webServer, ctx, missionID, "report.humanize.pending", map[string]any{
		"kind":                    "humanized_markdown_report_pending",
		"target":                  reporting.ExportTargetHumanizedMarkdown,
		"profile":                 reporting.HumanizeProfileH5,
		"pending_event_id":        "evt_humanize_pending",
		"report_pending_event_id": reportPending.EventID,
		"title":                   "Canceled report",
		"source_artifact_id":      "art_report",
		"agent_executor":          "codex",
		"report_mode":             "long_form",
		"text":                    "H5 말투 보정 Markdown artifact를 생성하는 중입니다.",
	}, app.Producer{Type: "agent", ID: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	canceled := false
	if _, ok := webServer.runningReports.Start(missionID, reportPending.EventID, func() {
		canceled = true
		if _, err := appendReportHumanizeFailed(context.Background(), webServer.service, newID, missionID, reportHumanizeInput{}, "ses_cancel", humanizePending.EventID, 1, errors.New("worker observed canceled context")); err != nil {
			t.Errorf("append worker cancellation failure returned error: %v", err)
		}
	}); !ok {
		t.Fatal("expected in-flight report registration")
	}

	response := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports/cancel", map[string]any{})
	if canceledFlag, ok := response["canceled"].(bool); !ok || !canceledFlag {
		t.Fatalf("expected canceled response, got %#v", response)
	}
	if inFlight, ok := response["in_flight"].(bool); !ok || !inFlight || !canceled {
		t.Fatalf("expected cancel to stop original draft in-flight run, response=%#v canceled=%v", response, canceled)
	}
	payload := nestedMap(t, response, "event", "Payload")
	if payload["kind"] != "humanized_markdown_report_canceled" ||
		payload["pending_event_id"] != humanizePending.EventID ||
		payload["report_pending_event_id"] != reportPending.EventID ||
		payload["canceled"] != true {
		t.Fatalf("expected humanize canceled payload for %q, got %#v", humanizePending.EventID, payload)
	}
	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	if hasOpenReportDraftDetail(detail) {
		t.Fatal("expected humanize pending to close after cancel")
	}
	if countEvents(detail, "report.draft.failed") != 0 {
		t.Fatalf("humanize cancel must not fail the preserved original report, got %#v", detail["events"])
	}
	detail = getJSON(t, server.URL+"/api/missions/"+missionID)
	if countEvents(detail, "report.humanize.failed") != 1 {
		t.Fatalf("expected humanize cancel terminal to stay idempotent, got %#v", detail["events"])
	}
}

func TestMissionSourcesHidesSupersededByDefault(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	server := httptest.NewServer(NewServer(svc, Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Superseded source list"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	oldSource := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/text", map[string]any{
		"title":        "Old source",
		"external_uri": "https://example.com/old",
		"content":      "old",
	})
	newSource := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/text", map[string]any{
		"title":        "New source",
		"external_uri": "https://example.com/new",
		"content":      "new",
	})
	oldSnapshotID := nestedString(t, oldSource, "snapshot", "SnapshotID")
	newSnapshotID := nestedString(t, newSource, "snapshot", "SnapshotID")
	if oldSnapshotID == "" || newSnapshotID == "" {
		t.Fatalf("expected source snapshots, old=%#v new=%#v", oldSource, newSource)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_superseded_source_list",
		MissionID: missionID,
		EventType: app.ConfluenceUpdatedEvent,
		Producer:  app.Producer{Type: "user", ID: "test"},
		Payload: mustJSON(map[string]any{
			"old_snapshot_id": oldSnapshotID,
			"new_snapshot_id": newSnapshotID,
		}),
	}); err != nil {
		t.Fatalf("AppendEvent returned error: %v", err)
	}

	defaultList := getJSON(t, server.URL+"/api/missions/"+missionID+"/sources")
	defaultRaw := toJSON(t, defaultList)
	if strings.Contains(defaultRaw, oldSnapshotID) {
		t.Fatalf("default source list should hide superseded source: %s", defaultRaw)
	}
	if !strings.Contains(defaultRaw, newSnapshotID) {
		t.Fatalf("default source list should include current source: %s", defaultRaw)
	}
	auditList := getJSON(t, server.URL+"/api/missions/"+missionID+"/sources?include_superseded=true")
	auditRaw := toJSON(t, auditList)
	if !strings.Contains(auditRaw, oldSnapshotID) || !strings.Contains(auditRaw, `"superseded":true`) {
		t.Fatalf("include_superseded should show superseded source state: %s", auditRaw)
	}
}

func TestDetailGETReconciliationResumesStaleReportDraft(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	agent := &fakeAgentExecutor{responses: []AgentResult{
		{Text: agentReportPlanJSON(agentReportPlan{
			Summary: "Plan the fresh report.",
			Sections: []agentReportSection{{
				Title:   "Pinned note",
				Purpose: "Use the pinned source after stale pending reconciliation.",
			}},
		}), SessionID: "report-session-1"},
		{Text: "# Fresh report\n\nFresh Markdown report.", SessionID: "report-session-1"},
	}}
	fixtureAgent := withReportPlanSubmissionFixture(svc, agent)
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: fixtureAgent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Stale report detail test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	appendStaleReportPending(t, ctx, svc, missionID, "evt_stale_report_pending_detail")
	server.Close()
	server = httptest.NewServer(NewServer(svc, Options{AgentExecutor: fixtureAgent}))
	defer server.Close()

	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if hasReportArtifactCreated(events) {
		t.Fatal("server construction resumed stale report before detail GET")
	}
	getJSON(t, server.URL+"/api/missions/"+missionID+"/activity")
	events, err = svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if hasReportArtifactCreated(events) {
		t.Fatal("activity read resumed stale report before detail GET")
	}
	getJSON(t, server.URL+"/api/missions/"+missionID)
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if hasOpenReportDraftDetail(detail) {
		t.Fatalf("expected resumed report pending to close after artifact creation")
	}
	if countEvents(detail, "report.draft.failed") != 0 {
		t.Fatalf("expected stale report pending to resume, got failure events: %#v", detail["events"])
	}
	payload := lastEventPayload(t, detail, "report.artifact.created")
	if payload["pending_event_id"] != "evt_stale_report_pending_detail" {
		t.Fatalf("expected artifact to close stale pending event, got %#v", payload)
	}
	if len(agent.requests) != 2 {
		t.Fatalf("expected recovered plan and writer requests, got %#v", agent.requests)
	}
	for _, req := range agent.requests {
		if !strings.Contains(req.Prompt, "Preserve the recovered operational focus.") || !strings.Contains(req.Prompt, reporting.DirectionAdvisory) {
			t.Fatalf("recovered report direction did not reach prompt:\n%s", req.Prompt)
		}
	}
}

func hasReportArtifactCreated(events []app.LedgerEvent) bool {
	for _, event := range events {
		if event.EventType == "report.artifact.created" {
			return true
		}
	}
	return false
}

func TestReportDraftStalePendingKeepsFrozenSelection(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	agent := &fakeAgentExecutor{responses: []AgentResult{{Text: agentReportPlanJSON(agentReportPlan{Summary: "Plan", Sections: []agentReportSection{{Title: "Section", Purpose: "Purpose"}}}), SessionID: "report-session"}, {Text: "# Report", SessionID: "report-session"}}}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Frozen recovery"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{EventID: "evt_frozen_pending", MissionID: missionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{
		"kind": "markdown_report_artifact_pending", "title": "Frozen", "agent_executor": "codex", "agent_model": "gpt-5.5", "agent_reasoning_effort": "high", "agent_selection_source": reporting.AgentSelectionSourceExplicitRequest,
		"mcp_mode": "auto", "report_mode": "planned", "report_session_policy": "same_session", "started_at": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano),
	})}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{EventID: "evt_newer_session", MissionID: missionID, EventType: "agent.session.reset", Producer: app.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{
		"agent_executor": "codex", "agent_model": "gpt-5.4", "agent_reasoning_effort": "low",
	})}); err != nil {
		t.Fatal(err)
	}
	getJSON(t, server.URL+"/api/missions/"+missionID)
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if len(agent.requests) != 2 {
		t.Fatalf("expected recovered plan/body, got %#v", agent.requests)
	}
	for _, request := range agent.requests {
		if request.Model != "gpt-5.5" || request.ReasoningEffort != "high" {
			t.Fatalf("recovery consulted newer metadata: %#v", request)
		}
	}
	for _, eventType := range []string{"report.plan.created", "report.artifact.created"} {
		payload := lastEventPayload(t, detail, eventType)
		if payload["agent_selection_source"] != reporting.AgentSelectionSourceExplicitRequest {
			t.Fatalf("%s source mismatch: %#v", eventType, payload)
		}
	}
}

func TestReportDraftStaleHumanizePendingFailsClosedWhenCannotResume(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	handler := NewServer(svc, Options{})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Stale humanize detail test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	reportPending, err := appendTestEvent(t, webServer, ctx, missionID, "report.draft.pending", map[string]any{
		"kind":           "markdown_report_artifact_pending",
		"title":          "Recovered report",
		"agent_executor": "codex",
		"report_mode":    "long_form",
		"text":           "리포트 초안 생성 중입니다.",
	}, app.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := appendTestEvent(t, webServer, ctx, missionID, "report.artifact.created", map[string]any{
		"kind":             "markdown_report_artifact",
		"pending_event_id": reportPending.EventID,
		"artifact_id":      "art_report",
		"media_type":       "text/markdown; charset=utf-8",
	}, app.Producer{Type: "agent_session", ID: "report-session-1"}); err != nil {
		t.Fatal(err)
	}
	humanizePending, err := appendTestEvent(t, webServer, ctx, missionID, "report.humanize.pending", map[string]any{
		"kind":                    "humanized_markdown_report_pending",
		"target":                  reporting.ExportTargetHumanizedMarkdown,
		"profile":                 reporting.HumanizeProfileH5,
		"pending_event_id":        "evt_humanize_stale",
		"report_pending_event_id": reportPending.EventID,
		"title":                   "Recovered report",
		"source_artifact_id":      "art_report",
		"agent_executor":          "codex",
		"report_mode":             "long_form",
		"text":                    "H5 말투 보정 Markdown artifact를 생성하는 중입니다.",
	}, app.Producer{Type: "agent", ID: "codex"})
	if err != nil {
		t.Fatal(err)
	}

	detail := waitForEventType(t, server.URL, missionID, "report.humanize.failed")
	if hasOpenReportDraftDetail(detail) {
		t.Fatal("expected stale humanize pending to close")
	}
	payload := lastEventPayload(t, detail, "report.humanize.failed")
	if payload["kind"] != "humanized_markdown_report_failed" ||
		payload["pending_event_id"] != humanizePending.EventID ||
		payload["source_artifact_id"] != "art_report" ||
		payload["preserved_original_markdown"] != true {
		t.Fatalf("expected stale humanize resume failure to preserve original report, got %#v", payload)
	}
	if countEvents(detail, "report.draft.failed") != 0 {
		t.Fatalf("stale humanize must not fail the preserved original report, got %#v", detail["events"])
	}
}

func TestReportDraftStaleHumanizePendingPromotesFinalizedPatch(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	handler := NewServer(svc, Options{})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Recovered humanize patch test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	source, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_source_report",
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   "source.md",
		Producer:   app.Producer{Type: "agent_session", ID: "report-session-1"},
		Content:    []byte("# Report\n\n수행되어야 한다."),
	})
	if err != nil {
		t.Fatal(err)
	}
	reportPending, err := appendTestEvent(t, webServer, ctx, missionID, "report.draft.pending", map[string]any{
		"kind":           "markdown_report_artifact_pending",
		"title":          "Recovered report",
		"agent_executor": "codex",
		"report_mode":    "long_form",
		"text":           "리포트 초안 생성 중입니다.",
	}, app.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := appendTestEvent(t, webServer, ctx, missionID, "report.artifact.created", map[string]any{
		"kind":             "markdown_report_artifact",
		"pending_event_id": reportPending.EventID,
		"artifact_id":      source.ArtifactID,
		"media_type":       source.MediaType,
	}, app.Producer{Type: "agent_session", ID: "report-session-1"}); err != nil {
		t.Fatal(err)
	}
	humanizePending, err := appendTestEvent(t, webServer, ctx, missionID, "report.humanize.pending", map[string]any{
		"kind":                      "humanized_markdown_report_pending",
		"target":                    reporting.ExportTargetHumanizedMarkdown,
		"profile":                   reporting.HumanizeProfileH5,
		"pending_event_id":          "evt_humanize_stale_finalized",
		"report_pending_event_id":   reportPending.EventID,
		"title":                     "Recovered report",
		"source_artifact_id":        source.ArtifactID,
		"source_artifact_sha256":    source.SHA256,
		"agent_executor":            "codex",
		"previous_agent_session_id": "report-session-1",
		"tool_session_id":           "tool-session-1",
		"mcp_mode":                  "auto",
		"report_mode":               "long_form",
		"text":                      "H5 말투 보정 Markdown artifact를 생성하는 중입니다.",
	}, app.Producer{Type: "agent", ID: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	patch, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_humanized_report",
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   "humanized.md",
		Producer:   app.Producer{Type: "agent_session", ID: "report-session-1"},
		Content:    []byte("# Report\n\n수행해야 한다."),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := appendTestEvent(t, webServer, ctx, missionID, "report.patch.finalized", map[string]any{
		"kind":              "markdown_report_patch_finalized",
		"pending_event_id":  humanizePending.EventID,
		"artifact_id":       patch.ArtifactID,
		"media_type":        patch.MediaType,
		"report_session_id": "report-session-1",
	}, app.Producer{Type: "agent_session", ID: "report-session-1"}); err != nil {
		t.Fatal(err)
	}

	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	if hasOpenReportDraftDetail(detail) {
		t.Fatal("expected stale finalized humanize patch to close")
	}
	if countEvents(detail, "report.humanize.failed") != 0 || countEvents(detail, "report.patch.rejected") != 0 {
		t.Fatalf("valid finalized patch must not be failed or rejected, got %#v", detail["events"])
	}
	payload := latestEventPayload(t, detail, "report.artifact.exported", reporting.ExportKindHumanizedMarkdown)
	if payload["artifact_id"] != patch.ArtifactID ||
		payload["pending_event_id"] != humanizePending.EventID ||
		payload["report_pending_event_id"] != reportPending.EventID ||
		payload["recovered_after_restart"] != true {
		t.Fatalf("expected recovered humanized export from finalized patch, got %#v", payload)
	}
}

func TestReportDraftStartResumesStalePendingBeforeConflict(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	release := make(chan struct{})
	agent := &sequenceBlockingAgentExecutor{release: release, responses: []AgentResult{
		{Text: agentReportPlanJSON(agentReportPlan{
			Summary: "Plan the recovered report.",
			Sections: []agentReportSection{{
				Title:   "Pinned note",
				Purpose: "Use the pinned source after stale pending recovery.",
			}},
		}), SessionID: "report-session-1"},
		{Text: "# Recovered report\n\nRecovered Markdown report.", SessionID: "report-session-1"},
	}}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Stale report start test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	source := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/text", map[string]any{
		"title":   "Pinned note",
		"content": "Pinned evidence for a report after stale pending reconciliation.",
	})
	proposal := postJSON(t, server.URL+"/api/missions/"+missionID+"/candidates/evidence", map[string]any{
		"summary":     "Pinned evidence for a report after stale pending reconciliation.",
		"snapshot_id": nestedString(t, source, "snapshot", "SnapshotID"),
		"artifact_id": nestedString(t, source, "artifact", "ArtifactID"),
	})
	postJSON(t, server.URL+"/api/missions/"+missionID+"/proposals/"+nestedString(t, proposal, "Proposal", "proposal_id")+"/approve", map[string]any{})
	appendStaleReportPending(t, ctx, svc, missionID, "evt_stale_report_pending_start")

	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{"title": "Fresh report", "report_mode": "planned"})
	if status != http.StatusConflict {
		t.Fatalf("expected fresh report request to conflict with recovered stale report, got %d %#v", status, body)
	}
	close(release)
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if hasOpenReportDraftDetail(detail) {
		t.Fatalf("expected no open report pending after recovered report draft completes")
	}
	if countEvents(detail, "report.draft.failed") != 0 {
		t.Fatalf("expected stale pending to recover without failure, got %#v", detail["events"])
	}
	if countEvents(detail, "report.draft.pending") != 1 {
		t.Fatalf("expected only recovered stale pending event, got %#v", detail["events"])
	}
}

func TestReportDraftResumesPartialLongFormSections(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	plan := agentSectionalReportPlan{
		Summary: "Recover a sectional report.",
		Parts: []agentReportPart{{
			Title:   "Recovered Part",
			Purpose: "Show that completed sections are durable.",
			Sections: []agentReportSection{{
				Title:   "Already written",
				Purpose: "This section was completed before restart.",
			}, {
				Title:   "Still needed",
				Purpose: "This section must be generated after restart.",
			}},
		}},
	}
	agent := &fakeAgentExecutor{responses: []AgentResult{
		{Text: "Generated second section body.", SessionID: "report-session-1"},
		{Text: `{"intro":"Part intro.","transitions":[],"closing":"Part closing."}`, SessionID: "report-session-1"},
		{Text: `{"front_matter":"# Recovered long report\n\nRecovered opening.","closing":"Recovered closing."}`, SessionID: "report-session-1"},
	}}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Partial long form resume test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	pendingID := "evt_partial_long_pending"
	finalArtifactID := "art_partial_long_final"
	sectionArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_partial_section_1",
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   "section-1.md",
		Producer:   app.Producer{Type: "agent_session", ID: "report-session-1"},
		Content:    []byte("Existing first section body."),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   pendingID,
		MissionID: missionID,
		EventType: "report.draft.pending",
		Producer:  app.Producer{Type: "user", ID: "plasma-ui"},
		Payload: mustJSON(map[string]any{
			"kind":           "markdown_report_artifact_pending",
			"title":          "Recovered long report",
			"agent_executor": "codex",
			"mcp_mode":       "auto",
			"rigor_level":    "exploratory",
			"report_mode":    reportModeLongForm,
			"text":           "리포트 초안 생성 중입니다.",
			"started_at":     time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano),
		}),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_partial_long_plan",
		MissionID: missionID,
		EventType: "report.plan.created",
		Producer:  app.Producer{Type: "agent_session", ID: "report-session-1"},
		Payload: mustJSON(map[string]any{
			"kind":                 "sectional_markdown_report_plan",
			"pending_event_id":     pendingID,
			"title":                "Recovered long report",
			"artifact_id":          finalArtifactID,
			"agent_executor":       "codex",
			"agent_session_id":     "report-session-1",
			"mcp_mode":             "auto",
			"report_mode":          reportModeLongForm,
			"composition_strategy": "sectional_preserve_markdown",
			"plan":                 plan,
			"text":                 "섹션별 장문 Markdown 리포트 생성 계획을 만들었습니다.",
		}),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_partial_long_section_1",
		MissionID: missionID,
		EventType: "report.section.created",
		Producer:  app.Producer{Type: "agent_session", ID: "report-session-1"},
		Payload: mustJSON(map[string]any{
			"kind":                 "sectional_markdown_report_section",
			"pending_event_id":     pendingID,
			"plan_event_id":        "evt_partial_long_plan",
			"title":                "Already written",
			"artifact_id":          sectionArtifact.ArtifactID,
			"media_type":           sectionArtifact.MediaType,
			"agent_executor":       "codex",
			"agent_session_id":     "report-session-1",
			"part_index":           1,
			"section_index":        1,
			"word_count":           reportWordCount(string(sectionArtifact.Content)),
			"composition_strategy": "sectional_preserve_markdown",
			"text":                 "장문 리포트 섹션 Markdown을 생성했습니다.",
		}),
	}); err != nil {
		t.Fatal(err)
	}

	getJSON(t, server.URL+"/api/missions/"+missionID)
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if countEvents(detail, "report.plan.created") != 1 {
		t.Fatalf("expected recovered report to reuse existing plan, got %#v", detail["events"])
	}
	if countEvents(detail, "report.section.created") != 2 {
		t.Fatalf("expected only missing section to be generated, got %#v", detail["events"])
	}
	if len(agent.requests) != 3 {
		t.Fatalf("expected section, part, and frame agent calls only, got %#v", agent.requests)
	}
	if !strings.Contains(agent.requests[0].UserText, "draft section 1.2") {
		t.Fatalf("expected first recovered call to draft missing section 1.2, got %#v", agent.requests[0])
	}
	payload := lastEventPayload(t, detail, "report.artifact.created")
	if payload["artifact_id"] != finalArtifactID || payload["pending_event_id"] != pendingID {
		t.Fatalf("expected final artifact to use recovered ids, got %#v", payload)
	}
	artifact, err := svc.GetRawArtifact(ctx, finalArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	content := string(artifact.Content)
	for _, expected := range []string{"Existing first section body.", "Generated second section body.", "Recovered closing."} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected recovered final report to contain %q:\n%s", expected, content)
		}
	}
}

func TestReportDraftResumesPartialLongFormParts(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	plan := agentSectionalReportPlan{
		Summary: "Recover a completed part.",
		Parts: []agentReportPart{{
			Title:   "Existing Part",
			Purpose: "Show that completed part artifacts are durable.",
			Sections: []agentReportSection{{
				Title:   "Existing Section",
				Purpose: "This section already exists inside the part.",
			}},
		}},
	}
	agent := &fakeAgentExecutor{responses: []AgentResult{
		{Text: `{"front_matter":"# Recovered from part\n\nRecovered opening.","closing":"Recovered part closing."}`, SessionID: "report-session-1"},
	}}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Partial part resume test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	pendingID := "evt_partial_part_pending"
	planEventID := "evt_partial_part_plan"
	finalArtifactID := "art_partial_part_final"
	partArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_partial_part_1",
		MissionID:  missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   "part-1.md",
		Producer:   app.Producer{Type: "agent_session", ID: "report-session-1"},
		Content:    []byte("# Part 1. Existing Part\n\nExisting preserved part body."),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   pendingID,
		MissionID: missionID,
		EventType: "report.draft.pending",
		Producer:  app.Producer{Type: "user", ID: "plasma-ui"},
		Payload: mustJSON(map[string]any{
			"kind":           "markdown_report_artifact_pending",
			"title":          "Recovered from part",
			"agent_executor": "codex",
			"mcp_mode":       "auto",
			"rigor_level":    "balanced",
			"report_mode":    reportModeLongForm,
			"text":           "리포트 초안 생성 중입니다.",
			"started_at":     time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano),
		}),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   planEventID,
		MissionID: missionID,
		EventType: "report.plan.created",
		Producer:  app.Producer{Type: "agent_session", ID: "report-session-1"},
		Payload: mustJSON(map[string]any{
			"kind":                 "sectional_markdown_report_plan",
			"pending_event_id":     pendingID,
			"title":                "Recovered from part",
			"artifact_id":          finalArtifactID,
			"agent_executor":       "codex",
			"agent_session_id":     "report-session-1",
			"mcp_mode":             "auto",
			"report_mode":          reportModeLongForm,
			"composition_strategy": "sectional_preserve_markdown",
			"plan":                 plan,
			"text":                 "섹션별 장문 Markdown 리포트 생성 계획을 만들었습니다.",
		}),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_partial_part_1",
		MissionID: missionID,
		EventType: "report.part.created",
		Producer:  app.Producer{Type: "agent_session", ID: "report-session-1"},
		Payload: mustJSON(map[string]any{
			"kind":                 "sectional_markdown_report_part",
			"pending_event_id":     pendingID,
			"plan_event_id":        planEventID,
			"title":                "Existing Part",
			"artifact_id":          partArtifact.ArtifactID,
			"media_type":           partArtifact.MediaType,
			"agent_executor":       "codex",
			"agent_session_id":     "report-session-1",
			"part_index":           1,
			"section_count":        1,
			"word_count":           reportWordCount(string(partArtifact.Content)),
			"composition_strategy": "sectional_preserve_markdown",
			"text":                 "장문 리포트 파트 Markdown을 보존 조립했습니다.",
		}),
	}); err != nil {
		t.Fatal(err)
	}

	getJSON(t, server.URL+"/api/missions/"+missionID)
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if countEvents(detail, "report.section.created") != 0 || countEvents(detail, "report.part.created") != 1 {
		t.Fatalf("expected recovered report to reuse existing part without section writes, got %#v", detail["events"])
	}
	if len(agent.requests) != 1 {
		t.Fatalf("expected only frame agent call, got %#v", agent.requests)
	}
	payload := lastEventPayload(t, detail, "report.artifact.created")
	if payload["artifact_id"] != finalArtifactID {
		t.Fatalf("expected final artifact to reuse planned artifact id, got %#v", payload)
	}
	artifact, err := svc.GetRawArtifact(ctx, finalArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	content := string(artifact.Content)
	for _, expected := range []string{"Existing preserved part body.", "Recovered part closing."} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected recovered report to contain %q:\n%s", expected, content)
		}
	}
}

func TestClaimConfidenceUpdateAppearsInMissionDetail(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	server := httptest.NewServer(NewServer(svc, Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{
		"title":     "Confidence test",
		"objective": "Track changing claim confidence",
	})
	missionID := nestedString(t, mission, "projection", "mission_id")
	source := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/text", map[string]any{
		"title":   "Pinned note",
		"content": "A new pinned note strongly supports the claim.",
	})
	proposal := postJSON(t, server.URL+"/api/missions/"+missionID+"/candidates/evidence", map[string]any{
		"summary":     "The pinned note strongly supports the claim.",
		"snapshot_id": nestedString(t, source, "snapshot", "SnapshotID"),
		"artifact_id": nestedString(t, source, "artifact", "ArtifactID"),
	})
	evidenceID := nestedString(t, proposal, "Evidence", "evidence_id")
	claimID := "clm_confidence_web"
	if _, err := svc.CreateClaimProposal(ctx, app.CreateClaimProposalRequest{
		ClaimEvent: app.AppendEventRequest{
			EventID:   "evt_confidence_web_claim",
			MissionID: missionID,
			EventType: "claim.proposed",
			Producer:  app.Producer{Type: "agent_session", ID: "ses_confidence_web"},
			Payload: mustJSON(map[string]any{
				"claim_id":    claimID,
				"proposal_id": "prp_confidence_web",
			}),
		},
		Claim: app.CreateClaimRecordRequest{
			ClaimID:               claimID,
			MissionID:             missionID,
			Text:                  "The claim can be reassessed as evidence improves.",
			ClaimType:             "descriptive",
			SupportingEvidenceIDs: []string{evidenceID},
			Confidence:            app.Confidence{Level: "low", Rationale: "Initial support was weak."},
			Approval:              app.Approval{State: "pending", Required: true},
			CreatedEventID:        "evt_confidence_web_claim",
		},
		ProposalEvent: app.AppendEventRequest{
			EventID:   "evt_confidence_web_proposal",
			MissionID: missionID,
			EventType: "proposal.submitted",
			Producer:  app.Producer{Type: "agent_session", ID: "ses_confidence_web"},
			Payload: mustJSON(map[string]any{
				"proposal_id": "prp_confidence_web",
			}),
		},
		Proposal: app.CreateProposalBundleRequest{
			ProposalID:        "prp_confidence_web",
			MissionID:         missionID,
			Title:             "Review claim",
			ObjectRefs:        []app.ObjectRef{{ObjectKind: app.ClaimRecordObjectKind, ObjectID: claimID}},
			RequestedDecision: "approve",
			CreatedEventID:    "evt_confidence_web_proposal",
		},
	}); err != nil {
		t.Fatal(err)
	}

	response := postJSON(t, server.URL+"/api/missions/"+missionID+"/claims/"+claimID+"/confidence", map[string]any{
		"level":              "high",
		"rationale":          "The accepted source-backed evidence now directly supports the claim.",
		"basis_evidence_ids": []string{evidenceID},
	})
	if got := nestedString(t, response, "event", "EventType"); got != app.ClaimConfidenceUpdatedEvent {
		t.Fatalf("unexpected event type %q", got)
	}
	detail := response["detail"].(map[string]any)
	records := detail["records"].(map[string]any)
	views := records["claim_confidence"].([]any)
	if len(views) != 1 {
		t.Fatalf("expected one confidence view, got %#v", views)
	}
	view := views[0].(map[string]any)
	if view["claim_id"] != claimID || view["direction"] != "up" {
		t.Fatalf("unexpected confidence view: %#v", view)
	}
	current := view["current_confidence"].(map[string]any)
	if current["level"] != "high" {
		t.Fatalf("unexpected current confidence: %#v", current)
	}
}

func TestNormalizeMCPModeDefaultsToAuto(t *testing.T) {
	mode, err := normalizeMCPMode("")
	if err != nil {
		t.Fatal(err)
	}
	if mode != "auto" {
		t.Fatalf("expected auto default, got %q", mode)
	}
}

func TestMediaURLSourceStoresImageArtifactAndAudioLiveReference(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	server := httptest.NewServer(NewServer(svc, Options{
		mediaFetcher: func(_ context.Context, rawURL string) (fetchedMediaSource, error) {
			if strings.Contains(rawURL, "sound") {
				return fetchedMediaSource{
					MediaType: "audio/mpeg",
					MediaKind: app.MediaKindAudio,
					ByteSize:  12345,
				}, nil
			}
			return fetchedMediaSource{
				Content:   []byte("fake-png-bytes"),
				MediaType: "image/png",
				MediaKind: app.MediaKindImage,
				ByteSize:  int64(len("fake-png-bytes")),
				Width:     640,
				Height:    480,
			}, nil
		},
	}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Media source"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	image := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/media_url", map[string]any{
		"url":         "https://example.com/image.png",
		"title":       "Example image",
		"license":     "CC-BY",
		"attribution": "Example author",
	})
	if nestedString(t, image, "artifact", "MediaType") != "image/png" {
		t.Fatalf("expected image artifact, got %#v", image)
	}
	imageRead := getJSON(t, server.URL+"/api/missions/"+missionID+"/sources/"+nestedString(t, image, "snapshot", "SnapshotID")+"/read")
	if _, ok := imageRead["content"]; ok {
		t.Fatalf("media source read must not return binary content: %#v", imageRead)
	}
	media := imageRead["media"].(map[string]any)
	if media["media_kind"] != app.MediaKindImage || media["inspection_support"] != "metadata_only_until_vision_engine_configured" {
		t.Fatalf("unexpected image media locator: %#v", media)
	}

	audio := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/media_url", map[string]any{
		"url":   "https://example.com/sound.mp3",
		"title": "Example audio",
	})
	if _, ok := audio["artifact"]; ok {
		t.Fatalf("audio live reference must not create an artifact: %#v", audio)
	}
	audioSnapshot := audio["snapshot"].(map[string]any)
	access := audioSnapshot["Access"].(map[string]any)
	if access["RetrievalPolicy"] != app.SourceRetrievalPolicyLiveReference {
		t.Fatalf("expected audio live reference, got %#v", audioSnapshot)
	}
}

func TestMediaLocatorFromJSONValidatesLocatorTypeAndLegacyKind(t *testing.T) {
	if _, err := mediaLocatorFromJSON(json.RawMessage(`{"locator_type":"pdf_document","media_kind":"image"}`)); err == nil {
		t.Fatal("expected non-media locator_type to be rejected")
	}
	legacy, err := mediaLocatorFromJSON(json.RawMessage(`{"kind":"media","media_kind":"image","mime_type":"image/png"}`))
	if err != nil {
		t.Fatalf("expected legacy media kind fallback: %v", err)
	}
	if legacy.LocatorType != app.SourceLocatorTypeMedia || legacy.Kind != "" || legacy.MediaKind != app.MediaKindImage {
		t.Fatalf("expected normalized legacy media locator, got %#v", legacy)
	}
	uploadedLegacy, err := mediaLocatorFromJSON(json.RawMessage(`{"kind":"file_upload","content_kind":"image","sanitized_filename":"legacy-pixel.png","media_type":"image/png","byte_size":123}`))
	if err != nil {
		t.Fatalf("expected legacy uploaded image fallback: %v", err)
	}
	if uploadedLegacy.LocatorType != app.SourceLocatorTypeMedia || uploadedLegacy.Provider != app.SourceConnectorTypeFileUpload || uploadedLegacy.Title != "legacy-pixel.png" || uploadedLegacy.MIMEType != "image/png" {
		t.Fatalf("expected normalized legacy uploaded image locator, got %#v", uploadedLegacy)
	}
}

func TestPDFURLSourceStoresOriginalAndReadsExtractedText(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	pdfBytes := testPDFBytes(t, []string{
		"Plasma PDF URL Source",
		"Alpha code is 53.",
		"Reading policy is extracted text only.",
	})
	server := httptest.NewServer(NewServer(app.NewService(store), Options{
		pdfFetcher: func(context.Context, string) (fetchedPDFSource, error) {
			return fetchedPDFSource{
				Content:    pdfBytes,
				MediaType:  "application/pdf",
				Title:      "PDF URL Source",
				ByteSize:   int64(len(pdfBytes)),
				PageCount:  1,
				TextLength: 80,
			}, nil
		},
	}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "PDF source"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	source := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/pdf_url", map[string]any{
		"url": "https://example.com/source.pdf",
	})
	if nestedString(t, source, "artifact", "MediaType") != "application/pdf" {
		t.Fatalf("expected PDF artifact, got %#v", source)
	}
	read := getJSON(t, server.URL+"/api/missions/"+missionID+"/sources/"+nestedString(t, source, "snapshot", "SnapshotID")+"/read?max_bytes=20000")
	content, _ := read["content"].(string)
	if !strings.Contains(content, "Plasma PDF URL Source") || strings.Contains(content, "%PDF-") {
		t.Fatalf("expected extracted PDF text without raw PDF bytes, got %#v", read)
	}
	extraction := read["extraction"].(map[string]any)
	if extraction["type"] != "pdf_text" {
		t.Fatalf("expected pdf_text extraction metadata, got %#v", read)
	}
}

func TestPDFURLSourceReusesStagedSourceCandidate(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	server := httptest.NewServer(NewServer(svc, Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "PDF staged candidate"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	pdfBytes := testPDFBytes(t, []string{
		"Staged PDF URL Source",
		"Alpha code is 94.",
	})
	artifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_candidate_pdf",
		MissionID:  missionID,
		MediaType:  "application/pdf",
		Filename:   "candidate.pdf",
		Producer:   app.Producer{Type: "agent", ID: "codex"},
		Content:    pdfBytes,
	})
	if err != nil {
		t.Fatalf("CreateRawArtifact returned error: %v", err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_candidate_staged",
		MissionID: missionID,
		EventType: "source.candidate.staged",
		Producer:  app.Producer{Type: "agent", ID: "codex"},
		Payload: mustJSON(map[string]any{
			"url":                "https://example.com/candidate.pdf",
			"proposal_event_id":  "evt_candidate_proposed",
			"artifact_id":        artifact.ArtifactID,
			"approval_state":     "unapproved_candidate",
			"not_report_default": true,
		}),
	}); err != nil {
		t.Fatalf("AppendEvent returned error: %v", err)
	}

	source := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/pdf_url", map[string]any{
		"url": "https://example.com/candidate.pdf",
	})
	if reused, _ := source["reused_source_candidate"].(bool); !reused {
		t.Fatalf("expected staged PDF candidate artifact reuse, got %#v", source)
	}
	if nestedString(t, source, "snapshot", "Connector", "ConnectorType") != app.SourceConnectorTypePDFURL {
		t.Fatalf("expected pdf_url source snapshot, got %#v", source)
	}
	read := getJSON(t, server.URL+"/api/missions/"+missionID+"/sources/"+nestedString(t, source, "snapshot", "SnapshotID")+"/read?max_bytes=20000")
	content, _ := read["content"].(string)
	if !strings.Contains(content, "Staged PDF URL Source") || strings.Contains(content, "%PDF-") {
		t.Fatalf("expected extracted staged PDF text without raw PDF bytes, got %#v", read)
	}
}

func TestFetchPDFSourceInspectsWithoutExtractingText(t *testing.T) {
	pdfBytes := testPDFBytes(t, []string{
		"Plasma PDF URL Source",
		"Alpha code is 53.",
		"Reading policy is extracted text only.",
	})
	pdfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(pdfBytes)
	}))
	defer pdfServer.Close()

	fetched, err := fetchPDFSourceWithClient(context.Background(), pdfServer.URL+"/source.pdf", pdfServer.Client())
	if err != nil {
		t.Fatalf("fetchPDFSourceWithClient returned error: %v", err)
	}
	if fetched.PageCount != 1 || fetched.TextLength != 0 || fetched.TextLengthKnown {
		t.Fatalf("expected PDF fetch to inspect only, got %#v", fetched)
	}
}

func TestAgentProposalExtractionIgnoresQuestionOnlyEvents(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	mission, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: "mis_question_only", Title: "Question only"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateSourceSnapshotWithEvent(ctx, app.CreateSourceSnapshotWithEventRequest{
		Artifact: app.CreateRawArtifactRequest{
			ArtifactID: "art_question_only",
			MissionID:  mission.MissionID,
			MediaType:  "text/plain",
			Producer:   app.Producer{Type: "user", ID: "plasma-ui"},
			Content:    []byte("source body"),
		},
		Snapshot: app.CreateSourceSnapshotRequest{
			SnapshotID:  "src_question_only",
			MissionID:   mission.MissionID,
			Connector:   app.ConnectorRef{ConnectorID: "test", ConnectorType: "test", ExternalSourceID: "test://source"},
			Title:       "Question only source",
			ArtifactIDs: []string{"art_question_only"},
		},
		Event: app.AppendEventRequest{
			EventID:   "evt_question_only_source",
			MissionID: mission.MissionID,
			EventType: "source.snapshotted",
			Producer:  app.Producer{Type: "user", ID: "plasma-ui"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateQuestionProposal(ctx, app.CreateQuestionProposalRequest{
		QuestionEvent: app.AppendEventRequest{
			EventID:   "evt_question_only",
			MissionID: mission.MissionID,
			EventType: "question.proposed",
			Producer:  app.Producer{Type: "agent_session", ID: "ses_question_only"},
			Payload: mustJSON(map[string]any{
				"question_id": "qst_question_only",
				"proposal_id": "prp_question_only",
			}),
		},
		Question: app.CreateQuestionRecordRequest{
			QuestionID:     "qst_question_only",
			MissionID:      mission.MissionID,
			State:          "open",
			Text:           "What evidence is missing?",
			Priority:       "medium",
			CreatedEventID: "evt_question_only",
		},
		ProposalEvent: app.AppendEventRequest{
			EventID:   "evt_question_only_proposal",
			MissionID: mission.MissionID,
			EventType: "proposal.submitted",
			Producer:  app.Producer{Type: "agent_session", ID: "ses_question_only"},
			Payload: mustJSON(map[string]any{
				"proposal_id": "prp_question_only",
			}),
		},
		Proposal: app.CreateProposalBundleRequest{
			ProposalID:        "prp_question_only",
			MissionID:         mission.MissionID,
			Title:             "Review question",
			ObjectRefs:        []app.ObjectRef{{ObjectKind: app.QuestionRecordObjectKind, ObjectID: "qst_question_only"}},
			RequestedDecision: "approve",
			CreatedEventID:    "evt_question_only_proposal",
		},
	}); err != nil {
		t.Fatal(err)
	}

	server := &Server{service: svc}
	if server.hasAgentKnowledgeProposalEvents(ctx, mission.MissionID, "ses_question_only") {
		t.Fatal("question-only proposals must not satisfy source-backed evidence/claim extraction")
	}
}

func TestAgentProposalExtractionRunsWhenMainTurnAlreadyCreatedOneProposal(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	mission, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: "mis_existing_proposal", Title: "Existing proposal"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateSourceSnapshotWithEvent(ctx, app.CreateSourceSnapshotWithEventRequest{
		Artifact: app.CreateRawArtifactRequest{
			ArtifactID: "art_existing_proposal",
			MissionID:  mission.MissionID,
			MediaType:  "text/plain",
			Producer:   app.Producer{Type: "user", ID: "plasma-ui"},
			Content:    []byte("source body with multiple useful facts and reactions"),
		},
		Snapshot: app.CreateSourceSnapshotRequest{
			SnapshotID:  "src_existing_proposal",
			MissionID:   mission.MissionID,
			Connector:   app.ConnectorRef{ConnectorID: "test", ConnectorType: "test", ExternalSourceID: "test://existing"},
			Title:       "Existing proposal source",
			ArtifactIDs: []string{"art_existing_proposal"},
		},
		Event: app.AppendEventRequest{
			EventID:   "evt_existing_proposal_source",
			MissionID: mission.MissionID,
			EventType: "source.snapshotted",
			Producer:  app.Producer{Type: "user", ID: "plasma-ui"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateEvidenceProposal(ctx, app.CreateEvidenceProposalRequest{
		EvidenceEvent: app.AppendEventRequest{
			EventID:   "evt_existing_evidence",
			MissionID: mission.MissionID,
			EventType: "evidence.proposed",
			Producer:  app.Producer{Type: "agent_session", ID: "ses_existing"},
			Payload: mustJSON(map[string]any{
				"evidence_id": "evd_existing",
				"proposal_id": "prp_existing",
			}),
		},
		Evidence: app.CreateEvidenceRecordRequest{
			EvidenceID:   "evd_existing",
			MissionID:    mission.MissionID,
			State:        "proposed",
			Summary:      "One existing proposal should not stop extraction.",
			EvidenceType: "fact",
			SnapshotRefs: []app.SnapshotRef{{
				SnapshotID: "src_existing_proposal",
				ArtifactID: "art_existing_proposal",
			}},
			Producer:       app.Producer{Type: "agent_session", ID: "ses_existing"},
			CreatedEventID: "evt_existing_evidence",
		},
		ProposalEvent: app.AppendEventRequest{
			EventID:   "evt_existing_proposal",
			MissionID: mission.MissionID,
			EventType: "proposal.submitted",
			Producer:  app.Producer{Type: "agent_session", ID: "ses_existing"},
			Payload: mustJSON(map[string]any{
				"proposal_id": "prp_existing",
			}),
		},
		Proposal: app.CreateProposalBundleRequest{
			ProposalID:        "prp_existing",
			MissionID:         mission.MissionID,
			Title:             "Review existing evidence",
			ObjectRefs:        []app.ObjectRef{{ObjectKind: app.EvidenceRecordObjectKind, ObjectID: "evd_existing"}},
			RequestedDecision: "approve",
			CreatedEventID:    "evt_existing_proposal",
		},
	}); err != nil {
		t.Fatal(err)
	}

	agent := &proposalWritingAgentExecutor{
		service:   svc,
		missionID: mission.MissionID,
		sessionID: "ses_existing",
	}
	server := &Server{service: svc}
	status := server.ensureAgentProposals(ctx, mission.MissionID, "evt_user_existing", recallPreview{
		Mission: recallMission{MissionID: mission.MissionID, Title: mission.Title, Objective: "Extract useful evidence"},
	}, agent, "codex", "auto", "ses_existing", AgentResult{Text: "answer from inspected source", SessionID: "agent-session-1"})

	if status["attempted"] != true || status["main_turn_created_proposals"] != true || status["created_proposals"] != true {
		t.Fatalf("expected proposal extraction to run despite existing proposal, got %#v", status)
	}
	if len(agent.requests) != 1 {
		t.Fatalf("expected proposal extraction agent call, got %d", len(agent.requests))
	}
	if agent.duplicateErr == nil {
		t.Fatal("expected duplicate evidence proposal attempt to fail")
	}
	if !strings.Contains(agent.requests[0].Prompt, "add missing non-duplicate proposals") {
		t.Fatalf("expected prompt to ask for missing non-duplicate proposals, got %q", agent.requests[0].Prompt)
	}
	if got := server.countAgentKnowledgeProposalEvents(ctx, mission.MissionID, "ses_existing"); got != 2 {
		t.Fatalf("expected only existing plus one distinct missing evidence proposal, got %d", got)
	}
}

func TestAgentProposalExtractionCountsCreatedProposalsDespiteEmptyResponse(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	mission, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: "mis_empty_proposal_response", Title: "Empty proposal response"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateSourceSnapshotWithEvent(ctx, app.CreateSourceSnapshotWithEventRequest{
		Artifact: app.CreateRawArtifactRequest{
			ArtifactID: "art_existing_proposal",
			MissionID:  mission.MissionID,
			MediaType:  "text/plain",
			Producer:   app.Producer{Type: "user", ID: "plasma-ui"},
			Content:    []byte("source body"),
		},
		Snapshot: app.CreateSourceSnapshotRequest{
			SnapshotID:  "src_existing_proposal",
			MissionID:   mission.MissionID,
			Connector:   app.ConnectorRef{ConnectorID: "test", ConnectorType: "test", ExternalSourceID: "test://source"},
			Title:       "Empty response source",
			ArtifactIDs: []string{"art_existing_proposal"},
		},
		Event: app.AppendEventRequest{
			EventID:   "evt_empty_proposal_response_source",
			MissionID: mission.MissionID,
			EventType: "source.snapshotted",
			Producer:  app.Producer{Type: "user", ID: "plasma-ui"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	agent := &proposalWritingAgentExecutor{
		service:       svc,
		missionID:     mission.MissionID,
		sessionID:     "ses_empty_response",
		returnErr:     errors.New("agent returned an empty response"),
		returnLog:     "proposal tools completed before empty response",
		skipDuplicate: true,
	}
	server := &Server{service: svc}
	status := server.ensureAgentProposals(ctx, mission.MissionID, "evt_user_empty_response", recallPreview{
		Mission: recallMission{MissionID: mission.MissionID, Title: mission.Title, Objective: "Extract useful evidence"},
	}, agent, "codex", "auto", "ses_empty_response", AgentResult{Text: "answer from inspected source", SessionID: "agent-session-1"})

	if status["attempted"] != true || status["created_proposals"] != true {
		t.Fatalf("expected created proposals to make extraction successful, got %#v", status)
	}
	if _, exists := status["error"]; exists {
		t.Fatalf("expected no extraction error when proposals were created, got %#v", status)
	}
	if status["warning"] != "agent returned an empty response" {
		t.Fatalf("expected warning to preserve empty response issue, got %#v", status)
	}
	if got := server.countAgentKnowledgeProposalEvents(ctx, mission.MissionID, "ses_empty_response"); got != 1 {
		t.Fatalf("expected one created evidence proposal, got %d", got)
	}
}

func TestInvestigationRequestDoesNotRequirePreapprovalForSourceSearch(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{responses: []AgentResult{{Text: "investigated", SessionID: "agent-session-1"}}}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Investigation request"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "보조모니터 케이블 구매 조사"})
	detail := waitForEventType(t, server.URL, missionID, "turn.agent.response")

	if countEvents(detail, "investigation.authorized") != 0 {
		t.Fatalf("source search should not require a preapproval event, got %#v", detail["events"])
	}
	if len(agent.requests) != 1 {
		t.Fatalf("expected one agent request, got %d", len(agent.requests))
	}
	if !strings.Contains(agent.requests[0].Prompt, "without asking for separate pre-approval") ||
		!strings.Contains(agent.requests[0].Prompt, "plasma.sources.candidates.propose") ||
		!strings.Contains(agent.requests[0].Prompt, "Source candidates are not sources and are not saved source snapshots") ||
		!strings.Contains(agent.requests[0].Prompt, "Plasma may also record those links as review candidates only") {
		t.Fatalf("expected prompt to allow search without preapproval while keeping C1 source attachment boundary, got %q", agent.requests[0].Prompt)
	}
}

func TestControllerSteeringUsesSameTurnPathWithoutKnowledgeRecords(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{responses: []AgentResult{{Text: "controller-steered answer", SessionID: "agent-session-1"}}}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Controller steering"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{
		"text":       "다음에는 반대 사례를 확인해줘.",
		"controller": true,
	})
	detail := waitForEventType(t, server.URL, missionID, "turn.agent.response")
	if len(agent.requests) != 1 {
		t.Fatalf("expected one shared turn agent request, got %d", len(agent.requests))
	}
	userPayload := firstEventPayload(t, detail, "turn.user")
	if userPayload["kind"] != "controller_steering" {
		t.Fatalf("expected controller steering event kind, got %#v", userPayload)
	}
	events := detail["events"].([]any)
	for _, raw := range events {
		event := raw.(map[string]any)
		switch event["EventType"] {
		case "source.candidate.proposed", "evidence.proposed", "claim.proposed":
			t.Fatalf("controller steering must not create source/evidence/claim records: %#v", event)
		}
	}
}

func TestAgentTurnRecordsControllerStrategySelection(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{responses: []AgentResult{{Text: "broad answer", SessionID: "agent-session-1"}}}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Controller strategy"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{
		"text":                "반대 관점과 대안을 넓게 비교해줘.",
		"controller_strategy": "auto",
	})
	detail := waitForEventType(t, server.URL, missionID, "turn.agent.response")

	if len(agent.requests) != 1 {
		t.Fatalf("expected one agent request, got %d", len(agent.requests))
	}
	if !strings.Contains(agent.requests[0].Prompt, "Controller strategy") ||
		!strings.Contains(agent.requests[0].Prompt, "v3") ||
		!strings.Contains(agent.requests[0].Prompt, "Use repeated lens shifts") {
		t.Fatalf("expected V3 controller guidance in prompt, got %q", agent.requests[0].Prompt)
	}
	payload := firstEventPayload(t, detail, "controller.strategy.selected")
	if payload["strategy_id"] != "v3" || payload["user_event_id"] == "" || payload["tool_session_id"] == "" {
		t.Fatalf("unexpected controller strategy payload: %#v", payload)
	}
	responsePayload := firstEventPayload(t, detail, "turn.agent.response")
	if responsePayload["strategy_id"] != "v3" {
		t.Fatalf("expected agent response to carry strategy id, got %#v", responsePayload)
	}
}

func TestAgentTurnRespectsExplicitControllerStrategy(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{responses: []AgentResult{{Text: "narrow answer", SessionID: "agent-session-1"}}}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Controller strategy override"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{
		"text":                "반대 관점과 대안을 넓게 비교해줘.",
		"controller_strategy": "v2",
	})
	detail := waitForEventType(t, server.URL, missionID, "turn.agent.response")

	if len(agent.requests) != 1 {
		t.Fatalf("expected one agent request, got %d", len(agent.requests))
	}
	if !strings.Contains(agent.requests[0].Prompt, "Controller strategy") ||
		!strings.Contains(agent.requests[0].Prompt, "v2") ||
		!strings.Contains(agent.requests[0].Prompt, "Stay close to the user's latest request") {
		t.Fatalf("expected explicit V2 controller guidance in prompt, got %q", agent.requests[0].Prompt)
	}
	if strings.Contains(agent.requests[0].Prompt, "Use repeated lens shifts") {
		t.Fatalf("explicit V2 request should not use V3 guidance: %q", agent.requests[0].Prompt)
	}
	payload := firstEventPayload(t, detail, "controller.strategy.selected")
	if payload["strategy_id"] != "v2" || payload["requested_strategy"] != "v2" {
		t.Fatalf("unexpected controller strategy payload: %#v", payload)
	}
}

func TestContextualSearchApprovalIsNotRequiredForMissionInvestigation(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{responses: []AgentResult{
		{Text: "소스 검색 승인이 필요합니다.", SessionID: "agent-session-1"},
		{Text: "approved", SessionID: "agent-session-1", Resumed: true},
	}}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Contextual investigation approval"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "pLTV 개념 설명"})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "승인이요 승인"})
	detail := waitForEventTypeCount(t, server.URL, missionID, "turn.agent.response", 2)

	if countEvents(detail, "investigation.authorized") != 0 {
		t.Fatalf("contextual approval should not create a search preapproval event, got %#v", detail["events"])
	}
	if len(agent.requests) != 2 {
		t.Fatalf("expected two agent requests, got %d", len(agent.requests))
	}
	if !strings.Contains(agent.requests[1].Prompt, "without asking for separate pre-approval") {
		t.Fatalf("expected contextual turn prompt to allow source discovery without preapproval, got %q", agent.requests[1].Prompt)
	}
}

func TestPlainQuestionDoesNotEnableMissionInvestigation(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{responses: []AgentResult{{Text: "2", SessionID: "agent-session-1"}}}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Plain question"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "1+1"})
	detail := waitForEventType(t, server.URL, missionID, "turn.agent.response")

	if countEvents(detail, "investigation.authorized") != 0 {
		t.Fatalf("plain question must not enable investigation, got %#v", detail["events"])
	}
	if len(agent.requests) != 1 {
		t.Fatalf("expected one agent request, got %d", len(agent.requests))
	}
	if !strings.Contains(agent.requests[0].Prompt, "source discovery: allowed when useful") {
		t.Fatalf("expected prompt to state the durable source discovery policy, got %q", agent.requests[0].Prompt)
	}
}

func TestPriorSearchApprovalTurnDoesNotNeedBackfilledPermission(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	agent := &fakeAgentExecutor{responses: []AgentResult{{Text: "resumed", SessionID: "agent-session-1"}}}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Backfill search approval"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_old_search_approval",
		MissionID: missionID,
		EventType: "turn.user",
		Producer:  app.Producer{Type: "user", ID: "plasma-ui"},
		Payload: mustJSON(map[string]any{
			"text":           "검색 승인",
			"agent_executor": "codex",
			"mcp_mode":       "auto",
		}),
	}); err != nil {
		t.Fatal(err)
	}

	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "이제 조사해줘"})
	detail := waitForEventType(t, server.URL, missionID, "turn.agent.response")

	if countEvents(detail, "investigation.authorized") != 0 {
		t.Fatalf("prior approval must not be backfilled into a search preapproval event, got %#v", detail["events"])
	}
	if len(agent.requests) != 1 {
		t.Fatalf("expected one agent request, got %d", len(agent.requests))
	}
	if !strings.Contains(agent.requests[0].Prompt, "without asking for separate pre-approval") {
		t.Fatalf("expected prompt to allow search without backfilled permission, got %q", agent.requests[0].Prompt)
	}
}

func TestAgentTurnResumesPreviousCodexSession(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{
		responses: []AgentResult{
			{Text: "first answer", SessionID: "codex-session-1"},
			{Text: "second answer", SessionID: "codex-session-1", Resumed: true},
		},
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Resume test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "first"})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "second"})
	waitForEventTypeCount(t, server.URL, missionID, "turn.agent.response", 2)

	if len(agent.requests) != 2 {
		t.Fatalf("expected two agent requests, got %d", len(agent.requests))
	}
	if agent.requests[1].PreviousSessionID != "codex-session-1" {
		t.Fatalf("expected resume session id, got %q", agent.requests[1].PreviousSessionID)
	}
	if !strings.Contains(agent.requests[1].Prompt, "Mission reminder") {
		t.Fatal("expected resumed prompt to contain only a short mission reminder")
	}
	if !strings.Contains(agent.requests[1].Prompt, "second") {
		t.Fatal("expected resumed prompt to contain latest user turn")
	}
}

func TestAgentTurnDoesNotAutoCompactOnResumeFailure(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{
		responses: []AgentResult{
			{Text: "first answer", SessionID: "codex-session-1"},
			{Log: "resume failed log tail"},
		},
		errors: []error{
			nil,
			errors.New("resume exploded"),
		},
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Resume failure test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "first"})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "second"})
	detail := waitForEventTypeCount(t, server.URL, missionID, "turn.agent.response", 2)

	if len(agent.requests) != 2 {
		t.Fatalf("expected initial and failed resume only, got %d", len(agent.requests))
	}
	if agent.requests[1].PreviousSessionID != "codex-session-1" {
		t.Fatalf("expected failed request to resume same session, got %q", agent.requests[1].PreviousSessionID)
	}
	if agent.requests[1].Compaction {
		t.Fatalf("resume failure must not trigger automatic compaction, got %#v", agent.requests[1])
	}
	if countEvents(detail, "turn.agent.compacted") != 0 {
		t.Fatalf("unexpected automatic compaction event: %#v", detail["events"])
	}
	payload := lastEventPayload(t, detail, "turn.agent.response")
	if payload["kind"] != "agent_error" {
		t.Fatalf("expected resume failure to be reported as agent_error, got %#v", payload)
	}
	if payload["previous_agent_session_id"] != "codex-session-1" {
		t.Fatalf("expected previous session marker, got %#v", payload)
	}
	if _, ok := payload["compaction_attempted"]; ok {
		t.Fatalf("resume failure must not mark compaction_attempted, got %#v", payload)
	}
}

func TestAgentTurnAutoCompactsAndRetriesWhenContextWindowIsFull(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{
		responses: []AgentResult{
			{Text: "first answer", SessionID: "codex-session-1"},
			{Log: "ERROR: Codex ran out of room in the model's context window. Start a new thread or clear earlier history before retrying."},
			{Text: "compact summary", SessionID: "codex-session-1", Resumed: true},
			{Text: "second answer after compact", SessionID: "codex-session-1", Resumed: true},
		},
		errors: []error{
			nil,
			errors.New("agent command failed: exit status 1"),
			nil,
			nil,
		},
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Auto compact test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "first"})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "second"})
	detail := waitForEventTypeCount(t, server.URL, missionID, "turn.agent.response", 2)

	if len(agent.requests) != 4 {
		t.Fatalf("expected first run, failed resume, compaction, and retry, got %d", len(agent.requests))
	}
	if agent.requests[1].PreviousSessionID != "codex-session-1" || agent.requests[1].Compaction {
		t.Fatalf("expected second request to be failed normal resume, got %#v", agent.requests[1])
	}
	if agent.requests[2].PreviousSessionID != "codex-session-1" || !agent.requests[2].Compaction {
		t.Fatalf("expected third request to compact same session, got %#v", agent.requests[2])
	}
	if agent.requests[3].PreviousSessionID != "codex-session-1" || agent.requests[3].Compaction {
		t.Fatalf("expected fourth request to retry same session, got %#v", agent.requests[3])
	}
	if countEvents(detail, "turn.agent.compacted") != 1 {
		t.Fatalf("expected one automatic compaction event, got %#v", detail["events"])
	}
	payload := lastEventPayload(t, detail, "turn.agent.response")
	if payload["kind"] != "agent_response" || payload["text"] != "second answer after compact" {
		t.Fatalf("expected successful retry response, got %#v", payload)
	}
	if payload["compaction_attempted"] != true || payload["retry_after_compacted"] != true {
		t.Fatalf("expected retry metadata after compaction, got %#v", payload)
	}
}

func TestAgentTurnRetryFailureAfterAutoCompactionRecordsTurnUsage(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	retryUsage := agentusage.New("codex", "codex", "", "", "retry prompt").
		WithProviderUsage(agentusage.ProviderUsage{InputTokens: 120, CachedInputTokens: 80, OutputTokens: 4}, "test")
	agent := &fakeAgentExecutor{
		responses: []AgentResult{
			{Text: "first answer", SessionID: "codex-session-1"},
			{Log: "ERROR: Codex ran out of room in the model's context window. Start a new thread or clear earlier history before retrying."},
			{Text: "compact summary", SessionID: "codex-session-1", Resumed: true},
			{SessionID: "codex-session-1", Resumed: true, Usage: retryUsage},
		},
		errors: []error{
			nil,
			errors.New("agent command failed: exit status 1"),
			nil,
			errors.New("retry failed"),
		},
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Auto compact retry failure test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "first"})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "second"})
	detail := waitForEventTypeCount(t, server.URL, missionID, "turn.agent.response", 2)

	if countEvents(detail, "turn.agent.compacted") != 1 {
		t.Fatalf("expected one automatic compaction event, got %#v", detail["events"])
	}
	payload := lastEventPayload(t, detail, "turn.agent.response")
	if payload["kind"] != "agent_error" || payload["compaction_attempted"] != true {
		t.Fatalf("expected retry failure after compaction, got %#v", payload)
	}
	if _, ok := payload["agent_usage_surface"]; ok {
		t.Fatalf("internal usage surface override must not be stored, got %#v", payload)
	}
	usage, ok := payload["agent_usage"].(map[string]any)
	if !ok {
		t.Fatalf("expected agent_usage on retry failure, got %#v", payload)
	}
	if usage["surface"] != "turn" {
		t.Fatalf("retry failure usage should be attributed to turn, got %#v", usage)
	}
	session, ok := usage["session"].(map[string]any)
	if !ok || session["compaction_attempted"] != true {
		t.Fatalf("expected usage session to keep compaction marker, got %#v", usage)
	}
}

func TestManualCompactCommandCompactsExistingSessionOnly(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{
		responses: []AgentResult{
			{Text: "first answer", SessionID: "codex-session-1"},
			{Text: "manual compact summary", SessionID: "codex-session-1", Resumed: true},
		},
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Manual compact test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "first"})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "/compact"})
	detail := waitForEventTypeCount(t, server.URL, missionID, "turn.agent.response", 2)

	if len(agent.requests) != 2 {
		t.Fatalf("expected normal turn and manual compaction only, got %d", len(agent.requests))
	}
	if !agent.requests[1].Compaction {
		t.Fatalf("expected second request to be compaction, got %#v", agent.requests[1])
	}
	if agent.requests[1].PreviousSessionID != "codex-session-1" {
		t.Fatalf("expected compaction to resume existing session, got %q", agent.requests[1].PreviousSessionID)
	}
	if countEvents(detail, "turn.agent.compacted") != 1 {
		t.Fatalf("expected compaction event, got %#v", detail["events"])
	}
	payload := lastEventPayload(t, detail, "turn.agent.response")
	if payload["kind"] != "agent_compacted" {
		t.Fatalf("expected terminal compact response, got %#v", payload)
	}
	if payload["agent_session_id"] != "codex-session-1" {
		t.Fatalf("expected same session id, got %#v", payload)
	}
}

func TestManualCompactCommandWithoutSessionSkipsCleanly(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Manual compact no session test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "/compact"})
	detail := waitForEventType(t, server.URL, missionID, "turn.agent.response")

	if len(agent.requests) != 0 {
		t.Fatalf("expected no agent call without prior session, got %d", len(agent.requests))
	}
	payload := firstEventPayload(t, detail, "turn.agent.response")
	if payload["kind"] != "agent_compaction_skipped" {
		t.Fatalf("expected compaction skipped response, got %#v", payload)
	}
}

func TestAgentTurnStopsWhenSuccessfulResumeReturnsDifferentSession(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{
		responses: []AgentResult{
			{Text: "first answer", SessionID: "codex-session-1"},
			{Text: "wrong session answer", SessionID: "different-session", Resumed: true},
			{Text: "third answer", SessionID: "codex-session-1", Resumed: true},
		},
	}
	handler := NewServer(app.NewService(store), Options{AgentExecutor: agent})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Resume mismatch test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "first"})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "second"})
	detail := waitForEventTypeCount(t, server.URL, missionID, "turn.agent.response", 2)

	payload := lastEventPayload(t, detail, "turn.agent.response")
	if payload["kind"] != "agent_error" {
		t.Fatalf("expected agent_error on resumed session mismatch, got %#v", payload)
	}
	if payload["returned_agent_session_id"] != "different-session" {
		t.Fatalf("expected returned session marker, got %#v", payload)
	}
	if _, ok := payload["agent_session_id"]; ok {
		t.Fatalf("mismatch error must not store canonical agent_session_id, got %#v", payload)
	}
	if got := webServer.latestAgentSessionID(ctx, missionID, "codex"); got != "codex-session-1" {
		t.Fatalf("expected canonical session to remain previous id, got %q", got)
	}
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "third"})
	waitForEventTypeCount(t, server.URL, missionID, "turn.agent.response", 3)
	if agent.requests[2].PreviousSessionID != "codex-session-1" {
		t.Fatalf("expected third turn to resume previous session, got %q", agent.requests[2].PreviousSessionID)
	}
}

func TestAgentTurnRejectsSuccessfulResumeWithoutSessionID(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{
		responses: []AgentResult{
			{Text: "first answer", SessionID: "codex-session-1"},
			{Text: "second answer", Resumed: true},
		},
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Resume blank id test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "first"})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "second"})
	detail := waitForEventTypeCount(t, server.URL, missionID, "turn.agent.response", 2)

	payload := lastEventPayload(t, detail, "turn.agent.response")
	if payload["kind"] != "agent_error" {
		t.Fatalf("expected agent_error, got %#v", payload)
	}
	if payload["previous_agent_session_id"] != "codex-session-1" ||
		payload["returned_agent_session_id"] != "" ||
		!strings.Contains(payload["error"].(string), "did not return a session id") {
		t.Fatalf("expected missing session id error to preserve previous lineage, got %#v", payload)
	}
	if _, ok := payload["agent_session_id"]; ok {
		t.Fatalf("missing session id error must not store canonical session id, got %#v", payload)
	}
}

func TestAgentSessionResetStartsNextTurnWithoutResume(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{
		responses: []AgentResult{
			{Text: "first answer", SessionID: "codex-session-1"},
			{Text: "new session answer", SessionID: "codex-session-2"},
		},
	}
	handler := NewServer(app.NewService(store), Options{AgentExecutor: agent})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Reset session test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "first"})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")

	reset := postJSON(t, server.URL+"/api/missions/"+missionID+"/agent_sessions/reset", map[string]any{"agent_executor": "codex", "agent_model": "gpt-5.5", "agent_reasoning_effort": "high"})
	if reset["previous_agent_session_id"] != "codex-session-1" {
		t.Fatalf("expected reset to report previous session, got %#v", reset)
	}
	if reset["agent_model"] != "gpt-5.5" {
		t.Fatalf("expected reset to report agent model, got %#v", reset)
	}
	if reset["agent_reasoning_effort"] != "high" {
		t.Fatalf("expected reset to report agent reasoning effort, got %#v", reset)
	}
	if got := webServer.latestAgentSessionID(ctx, missionID, "codex"); got != "" {
		t.Fatalf("expected reset to clear latest codex session, got %q", got)
	}
	if got := webServer.latestAgentSessionModel(ctx, missionID, "codex"); got != "gpt-5.5" {
		t.Fatalf("expected reset model to be stored, got %q", got)
	}
	if got := webServer.latestAgentReasoningEffort(ctx, missionID, "codex"); got != "high" {
		t.Fatalf("expected reset reasoning effort to be stored, got %q", got)
	}

	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "second"})
	detail := waitForEventTypeCount(t, server.URL, missionID, "turn.agent.response", 2)
	if len(agent.requests) != 2 {
		t.Fatalf("expected two agent requests, got %d", len(agent.requests))
	}
	if agent.requests[1].PreviousSessionID != "" {
		t.Fatalf("expected next turn to start without resume, got %q", agent.requests[1].PreviousSessionID)
	}
	if agent.requests[1].Model != "gpt-5.5" {
		t.Fatalf("expected next turn to use reset model, got %q", agent.requests[1].Model)
	}
	if agent.requests[1].ReasoningEffort != "high" {
		t.Fatalf("expected next turn to use reset reasoning effort, got %q", agent.requests[1].ReasoningEffort)
	}
	if countEvents(detail, "agent.session.reset") != 1 {
		t.Fatalf("expected reset event, got %#v", detail["events"])
	}
	payload := lastEventPayload(t, detail, "turn.agent.response")
	if payload["agent_session_id"] != "codex-session-2" {
		t.Fatalf("expected new session id to be stored, got %#v", payload)
	}
	if payload["agent_model"] != "gpt-5.5" {
		t.Fatalf("expected response to keep reset model, got %#v", payload)
	}
	if payload["agent_reasoning_effort"] != "high" {
		t.Fatalf("expected response to keep reset reasoning effort, got %#v", payload)
	}
}

func TestAgentSessionResetDefaultKeepsRawSelectionAndUsesCurrentDefaultForNewTurn(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	agent := &fakeAgentExecutor{responses: []AgentResult{{Text: "new session answer", SessionID: "codex-session-1"}}}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Default reset selection"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	reset := postJSON(t, server.URL+"/api/missions/"+missionID+"/agent_sessions/reset", map[string]any{"agent_executor": "codex"})
	if reset["agent_model"] != "" || reset["agent_reasoning_effort"] != "" {
		t.Fatalf("default reset must preserve raw empty selection, got %#v", reset)
	}
	resetPayload, ok := nestedValue(t, reset, "event", "Payload").(map[string]any)
	if !ok || resetPayload["agent_model"] != "" || resetPayload["agent_reasoning_effort"] != "" {
		t.Fatalf("default reset event must store raw empty selection, got %#v", resetPayload)
	}

	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "start"})
	detail := waitForEventType(t, server.URL, missionID, "turn.agent.response")
	if len(agent.requests) != 1 || agent.requests[0].Model != "gpt-5.5" || agent.requests[0].ReasoningEffort != "medium" {
		t.Fatalf("expected new turn to resolve GPT-5.5/medium, got %#v", agent.requests)
	}
	payload := lastEventPayload(t, detail, "turn.agent.response")
	if payload["agent_model"] != "gpt-5.5" || payload["agent_reasoning_effort"] != "medium" {
		t.Fatalf("expected response metadata to record effective default, got %#v", payload)
	}
}

func TestStaleAgentTurnAutoClosesBeforeAgentSessionReset(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: &fakeAgentExecutor{}}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Reset stale agent turn"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	appendStaleAgentPending(t, ctx, svc, missionID, "evt_stale_reset_user", "evt_stale_reset_pending")

	reset := postJSON(t, server.URL+"/api/missions/"+missionID+"/agent_sessions/reset", map[string]any{
		"agent_executor":         "codex",
		"agent_model":            "gpt-5.5",
		"agent_reasoning_effort": "medium",
	})
	if reset["agent_model"] != "gpt-5.5" || reset["agent_reasoning_effort"] != "medium" {
		t.Fatalf("expected reset to succeed after stale cleanup, got %#v", reset)
	}
	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	payload := firstEventPayload(t, detail, "turn.agent.response")
	if payload["kind"] != "agent_canceled" || payload["user_event_id"] != "evt_stale_reset_user" {
		t.Fatalf("expected stale agent turn to be auto-canceled before reset, got %#v", payload)
	}
	if countEvents(detail, "agent.session.reset") != 1 {
		t.Fatalf("expected reset event after stale cleanup, got %#v", detail["events"])
	}
}

func TestWorkflowUsesResetAgentModelAndReasoningEffort(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{
		responses: []AgentResult{
			{Text: "first answer", SessionID: "codex-session-1"},
			{Text: "workflow result\nPLASMA_WORKFLOW_CONTROL: {\"decision\":\"stop\",\"reason\":\"done\"}", SessionID: "codex-session-2"},
		},
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Workflow model reset test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "first"})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/agent_sessions/reset", map[string]any{
		"agent_executor":         "codex",
		"agent_model":            "gpt-5.5",
		"agent_reasoning_effort": "high",
	})
	postJSON(t, server.URL+"/api/missions/"+missionID+"/workflows", map[string]any{
		"instruction": "다각도로 조사",
		"max_steps":   1,
	})
	detail := waitForEventType(t, server.URL, missionID, app.WorkflowRunCompletedEvent)
	if len(agent.requests) != 2 {
		t.Fatalf("expected initial turn and workflow request, got %#v", agent.requests)
	}
	workflowReq := agent.requests[1]
	if workflowReq.Model != "gpt-5.5" || workflowReq.ReasoningEffort != "high" {
		t.Fatalf("expected workflow to use reset model and effort, got %#v", workflowReq)
	}
	payload := lastEventPayload(t, detail, "turn.agent.response")
	if payload["agent_model"] != "gpt-5.5" || payload["agent_reasoning_effort"] != "high" {
		t.Fatalf("expected workflow response payload to keep model metadata, got %#v", payload)
	}
}

func TestWorkflowPreservesEmptySettingsForLegacySession(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	agent := &fakeAgentExecutor{responses: []AgentResult{{Text: "done\nPLASMA_WORKFLOW_CONTROL: {\"decision\":\"stop\",\"reason\":\"done\"}", SessionID: "legacy-session", Resumed: true}}}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: agent}))
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Legacy workflow settings"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	appendLegacyCodexSession(t, ctx, svc, missionID, "legacy-session")

	postJSON(t, server.URL+"/api/missions/"+missionID+"/workflows", map[string]any{"instruction": "조사", "max_steps": 1})
	waitForEventType(t, server.URL, missionID, app.WorkflowRunCompletedEvent)
	if len(agent.requests) != 1 || agent.requests[0].Model != "" || agent.requests[0].ReasoningEffort != "" {
		t.Fatalf("legacy workflow must preserve empty settings, got %#v", agent.requests)
	}
}

func TestReportDraftSettingsRespectLegacyResumeAndNewMissionDefault(t *testing.T) {
	for _, test := range []struct {
		name       string
		legacy     bool
		wantModel  string
		wantEffort string
	}{
		{name: "legacy resume", legacy: true, wantModel: "gpt-5.5", wantEffort: "medium"},
		{name: "new mission", wantModel: "gpt-5.5", wantEffort: "medium"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			svc := app.NewService(store)
			sessionID := "report-session"
			agent := &fakeAgentExecutor{responses: []AgentResult{
				{Text: agentReportPlanJSON(agentReportPlan{Summary: "Plan", Sections: []agentReportSection{{Title: "Section", Purpose: "Purpose"}}}), SessionID: sessionID, Resumed: test.legacy},
				{Text: "# Report", SessionID: sessionID, Resumed: test.legacy},
			}}
			server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
			defer server.Close()
			mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": test.name})
			missionID := nestedString(t, mission, "projection", "mission_id")
			if test.legacy {
				appendLegacyCodexSession(t, ctx, svc, missionID, sessionID)
			}
			start := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{"title": "Report", "report_mode": "planned"})
			pendingPayload, ok := nestedValue(t, start, "pending_event", "Payload").(map[string]any)
			if !ok {
				t.Fatalf("expected report pending payload, got %#v", start)
			}
			if pendingPayload["agent_model"] != "gpt-5.5" || pendingPayload["agent_reasoning_effort"] != "medium" || pendingPayload["agent_selection_source"] == "" {
				t.Fatalf("fresh report pending state must freeze effective defaults, got %#v", pendingPayload)
			}
			waitForEventType(t, server.URL, missionID, "report.artifact.created")
			if len(agent.requests) != 2 {
				t.Fatalf("expected plan and report requests, got %#v", agent.requests)
			}
			for _, request := range agent.requests {
				if request.Model != test.wantModel || request.ReasoningEffort != test.wantEffort {
					t.Fatalf("unexpected report settings: %#v", request)
				}
			}
		})
	}
}

func TestReportDraftUsesResetAgentModelAndReasoningEffort(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	agent := &fakeAgentExecutor{
		responses: []AgentResult{
			{Text: "first answer", SessionID: "codex-session-1"},
			{Text: agentReportPlanJSON(agentReportPlan{
				Summary: "Use reset model settings for the report.",
				Sections: []agentReportSection{{
					Title:   "Reset settings",
					Purpose: "Confirm report calls inherit selected agent settings.",
				}},
			}), SessionID: "report-session-1"},
			{Text: "# Reset report\n\nThe report used selected agent settings.", SessionID: "report-session-1"},
		},
	}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Report model reset test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "first"})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/agent_sessions/reset", map[string]any{
		"agent_executor":         "codex",
		"agent_model":            "gpt-5.5",
		"agent_reasoning_effort": "high",
	})
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":       "Reset report",
		"report_mode": "planned",
	})
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if len(agent.requests) != 3 {
		t.Fatalf("expected initial turn, plan request, and report request, got %#v", agent.requests)
	}
	for index := 1; index < len(agent.requests); index++ {
		if agent.requests[index].Model != "gpt-5.5" || agent.requests[index].ReasoningEffort != "high" {
			t.Fatalf("expected report request %d to use reset model and effort, got %#v", index, agent.requests[index])
		}
	}
	planPayload := lastEventPayload(t, detail, "report.plan.created")
	if planPayload["agent_model"] != "gpt-5.5" || planPayload["agent_reasoning_effort"] != "high" {
		t.Fatalf("expected report plan payload to keep model metadata, got %#v", planPayload)
	}
	reportPayload := lastEventPayload(t, detail, "report.artifact.created")
	if reportPayload["agent_model"] != "gpt-5.5" || reportPayload["agent_reasoning_effort"] != "high" {
		t.Fatalf("expected report artifact payload to keep model metadata, got %#v", reportPayload)
	}
}

func TestMissionDetailIncludesAgentDefaultModelMetadata(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{
		AgentExecutors: map[string]AgentExecutor{
			"codex":  CodexExecutor{},
			"claude": ClaudeExecutor{Model: "sonnet"},
		},
	}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Agent metadata test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	statuses, ok := detail["agent_executors"].([]any)
	if !ok {
		t.Fatalf("expected agent_executors in detail, got %#v", detail["agent_executors"])
	}
	codex := agentStatusByName(t, statuses, "codex")
	if codex["default_model"] != "gpt-5.5" || codex["default_model_label"] != "GPT-5.5" || codex["default_model_version"] != "gpt-5.5" {
		t.Fatalf("unexpected codex model metadata: %#v", codex)
	}
	if codex["reasoning_effort_supported"] != true || codex["default_reasoning_effort"] != "medium" {
		t.Fatalf("unexpected codex reasoning metadata: %#v", codex)
	}
	models, ok := codex["models"].([]any)
	if !ok || len(models) < 3 {
		t.Fatalf("expected Codex model catalog, got %#v", codex["models"])
	}
	if model := agentModelByName(t, models, "gpt-5.6-luna"); len(stringSliceFromMap(t, model, "reasoning_efforts")) != 5 {
		t.Fatalf("unexpected Luna capabilities: %#v", model)
	}
	claude := agentStatusByName(t, statuses, "claude")
	if claude["default_model"] != "sonnet" || claude["default_model_label"] != "Claude Sonnet" || claude["default_model_version"] != "sonnet" {
		t.Fatalf("unexpected claude model metadata: %#v", claude)
	}
	if claude["reasoning_effort_supported"] != false || !strings.Contains(nestedStringFromMap(t, claude, "reasoning_effort_note"), "지원하지 않습니다") {
		t.Fatalf("unexpected claude reasoning metadata: %#v", claude)
	}
	claudeModels, ok := claude["models"].([]any)
	if !ok {
		t.Fatalf("missing Claude model catalog: %#v", claude)
	}
	for _, name := range []string{"haiku", "sonnet", "opus"} {
		agentModelByName(t, claudeModels, name)
	}
}

func TestReportDraftRequestSelectionIntegration(t *testing.T) {
	for _, test := range []struct {
		name       string
		body       map[string]any
		wantEffort string
	}{
		{name: "explicit pair", body: map[string]any{"agent_model": "gpt-5.5", "agent_reasoning_effort": "high"}, wantEffort: "high"},
		{name: "explicit model default", body: map[string]any{"agent_model": "gpt-5.5"}, wantEffort: "medium"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			svc := app.NewService(store)
			agent := &fakeAgentExecutor{responses: []AgentResult{{Text: agentReportPlanJSON(agentReportPlan{Summary: "Plan", Sections: []agentReportSection{{Title: "Section", Purpose: "Purpose"}}}), SessionID: "report-session"}, {Text: "# Report", SessionID: "report-session"}}}
			server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, agent)}))
			defer server.Close()
			mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": test.name})
			missionID := nestedString(t, mission, "projection", "mission_id")
			body := map[string]any{"title": "Report", "report_mode": "planned"}
			for key, value := range test.body {
				body[key] = value
			}
			start := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", body)
			pending, ok := nestedValue(t, start, "pending_event", "Payload").(map[string]any)
			if !ok {
				t.Fatalf("missing pending payload: %#v", start)
			}
			if pending["agent_model"] != "gpt-5.5" || pending["agent_reasoning_effort"] != test.wantEffort || pending["agent_selection_source"] != reporting.AgentSelectionSourceExplicitRequest {
				t.Fatalf("pending selection mismatch: %#v", pending)
			}
			detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
			if len(agent.requests) != 2 {
				t.Fatalf("expected plan/body, got %#v", agent.requests)
			}
			for _, req := range agent.requests {
				if req.Model != "gpt-5.5" || req.ReasoningEffort != test.wantEffort {
					t.Fatalf("agent selection mismatch: %#v", req)
				}
			}
			for _, eventType := range []string{"report.plan.created", "report.artifact.created"} {
				payload := lastEventPayload(t, detail, eventType)
				if payload["agent_selection_source"] != reporting.AgentSelectionSourceExplicitRequest {
					t.Fatalf("%s source mismatch: %#v", eventType, payload)
				}
			}
		})
	}
}

func TestReportDraftInvalidSelectionHasNoSideEffects(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	agent := &fakeAgentExecutor{}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "invalid selection"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	status, _ := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{"title": "Report", "report_mode": "planned", "agent_model": "gpt-5.6-luna", "agent_reasoning_effort": "ultra"})
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d", status)
	}
	if len(agent.requests) != 0 {
		t.Fatalf("agent called: %#v", agent.requests)
	}
	events, err := app.NewService(store).ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.EventType == "report.draft.pending" {
			t.Fatalf("invalid selection appended pending: %#v", event)
		}
	}
}

func TestAgentSessionResetRequiresLockedExecutor(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	codex := &fakeAgentExecutor{responses: []AgentResult{{Text: "codex first", SessionID: "codex-session-1"}}}
	claude := &fakeAgentExecutor{responses: []AgentResult{{Text: "claude first", SessionID: "claude-session-1"}}}
	handler := NewServer(app.NewService(store), Options{
		AgentExecutors: map[string]AgentExecutor{
			"codex":  codex,
			"claude": claude,
		},
	})
	webServer := handler.(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Provider reset test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "codex", "agent_executor": "codex"})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")

	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/agent_sessions/reset", map[string]any{"agent_executor": "claude"})
	if status != http.StatusBadRequest {
		t.Fatalf("expected provider switch reset to fail with 400, got %d %#v", status, body)
	}
	if !strings.Contains(nestedString(t, body, "error", "message"), "already using codex") {
		t.Fatalf("expected locked-provider error, got %#v", body)
	}

	postJSON(t, server.URL+"/api/missions/"+missionID+"/agent_sessions/reset", map[string]any{"agent_executor": "codex"})
	if got := webServer.latestAgentSessionID(ctx, missionID, "codex"); got != "" {
		t.Fatalf("expected codex reset to clear codex session, got %q", got)
	}
	if len(claude.requests) != 0 {
		t.Fatalf("expected locked mission not to invoke claude, got %d requests", len(claude.requests))
	}
}

func TestAgentSessionResetValidatesGPT56ReasoningEffort(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: &fakeAgentExecutor{}}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "GPT-5.6 reset validation"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	status, _ := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/agent_sessions/reset", map[string]any{
		"agent_executor": "codex", "agent_model": "gpt-5.6-luna", "agent_reasoning_effort": "ultra",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected Luna ultra rejection, got %d", status)
	}
	reset := postJSON(t, server.URL+"/api/missions/"+missionID+"/agent_sessions/reset", map[string]any{
		"agent_executor": "codex", "agent_model": "gpt-5.6-sol", "agent_reasoning_effort": "ultra",
	})
	if reset["agent_model"] != "gpt-5.6-sol" || reset["agent_reasoning_effort"] != "ultra" {
		t.Fatalf("expected Sol ultra reset, got %#v", reset)
	}
}

func TestMissionRejectsProviderSwitchAfterFirstTurn(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	codex := &fakeAgentExecutor{
		responses: []AgentResult{
			{Text: "codex first", SessionID: "codex-session-1"},
			{Text: "codex second", SessionID: "codex-session-1", Resumed: true},
		},
	}
	claude := &fakeAgentExecutor{
		responses: []AgentResult{{Text: "claude first", SessionID: "claude-session-1"}},
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{
		AgentExecutors: map[string]AgentExecutor{
			"codex":  codex,
			"claude": claude,
		},
	}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Provider session test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "codex first", "agent_executor": "codex"})
	detail := waitForEventType(t, server.URL, missionID, "turn.agent.response")

	if detail["locked_agent_executor"] != "codex" {
		t.Fatalf("expected mission to lock to codex, got %#v", detail["locked_agent_executor"])
	}
	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "claude first", "agent_executor": "claude"})
	if status != http.StatusBadRequest {
		t.Fatalf("expected provider switch turn to fail with 400, got %d %#v", status, body)
	}
	if !strings.Contains(nestedString(t, body, "error", "message"), "already using codex") {
		t.Fatalf("expected locked-provider error, got %#v", body)
	}
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "codex second", "agent_executor": "codex"})
	waitForEventTypeCount(t, server.URL, missionID, "turn.agent.response", 2)

	if len(claude.requests) != 0 {
		t.Fatalf("expected locked mission not to invoke claude, got %d requests", len(claude.requests))
	}
	if codex.requests[1].PreviousSessionID != "codex-session-1" {
		t.Fatalf("expected codex to resume codex session, got %q", codex.requests[1].PreviousSessionID)
	}
}

func TestSeparateMissionsCanUseDifferentExecutors(t *testing.T) {
	store, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	codex := &fakeAgentExecutor{responses: []AgentResult{{Text: "codex first", SessionID: "codex-session-1"}}}
	claude := &fakeAgentExecutor{responses: []AgentResult{{Text: "claude first", SessionID: "claude-session-1"}}}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{
		AgentExecutors: map[string]AgentExecutor{
			"codex":  codex,
			"claude": claude,
		},
	}))
	defer server.Close()

	codexMission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Codex mission"})
	codexMissionID := nestedString(t, codexMission, "projection", "mission_id")
	claudeMission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Claude mission"})
	claudeMissionID := nestedString(t, claudeMission, "projection", "mission_id")

	postJSON(t, server.URL+"/api/missions/"+codexMissionID+"/turns", map[string]any{"text": "codex", "agent_executor": "codex"})
	codexDetail := waitForEventType(t, server.URL, codexMissionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+claudeMissionID+"/turns", map[string]any{"text": "claude", "agent_executor": "claude"})
	claudeDetail := waitForEventType(t, server.URL, claudeMissionID, "turn.agent.response")

	if codexDetail["locked_agent_executor"] != "codex" {
		t.Fatalf("expected codex mission lock, got %#v", codexDetail["locked_agent_executor"])
	}
	if claudeDetail["locked_agent_executor"] != "claude" {
		t.Fatalf("expected claude mission lock, got %#v", claudeDetail["locked_agent_executor"])
	}
}

func TestProviderLockIgnoresEventsWithoutExplicitExecutor(t *testing.T) {
	events := []app.LedgerEvent{
		{EventType: "turn.agent.response", Payload: []byte(`{"kind":"agent_response","text":"legacy"}`)},
		{EventType: "report.draft.pending", Payload: []byte(`{"kind":"report_draft_pending"}`)},
	}
	if got := app.LockedAgentExecutorFromEvents(events); got != "" {
		t.Fatalf("expected no lock from events without agent_executor, got %q", got)
	}

	events = append(events, app.LedgerEvent{EventType: "turn.agent.response", Payload: []byte(`{"kind":"agent_response","agent_executor":"claude"}`)})
	if got := app.LockedAgentExecutorFromEvents(events); got != "claude" {
		t.Fatalf("expected explicit claude lock, got %q", got)
	}
}

func TestUntaggedAgentSessionOnlyResumesForCodexCompatibility(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	handler := NewServer(app.NewService(store), Options{})
	server := handler.(*Server)
	mission, err := server.createMission(ctx, createMissionRequest{Title: "Legacy session test"})
	if err != nil {
		t.Fatal(err)
	}
	missionID := mission.Projection.MissionID
	if _, err := appendTestEvent(t, server, ctx, missionID, "turn.agent.response", map[string]any{
		"kind":             "agent_response",
		"agent_session_id": "legacy-codex-session",
		"text":             "legacy response",
		"user_event_id":    "evt_user",
	}, app.Producer{Type: "agent", ID: "codex"}); err != nil {
		t.Fatal(err)
	}

	if got := server.latestAgentSessionID(ctx, missionID, "codex"); got != "legacy-codex-session" {
		t.Fatalf("expected codex to reuse legacy untagged session, got %q", got)
	}
	if got := server.latestAgentSessionID(ctx, missionID, "claude"); got != "" {
		t.Fatalf("expected claude not to reuse legacy codex session, got %q", got)
	}
}

func TestReportArtifactSessionContributesToLatestAgentSession(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	handler := NewServer(app.NewService(store), Options{})
	server := handler.(*Server)
	mission, err := server.createMission(ctx, createMissionRequest{Title: "Report session test"})
	if err != nil {
		t.Fatal(err)
	}
	missionID := mission.Projection.MissionID
	if _, err := appendTestEvent(t, server, ctx, missionID, "report.artifact.created", map[string]any{
		"kind":             "markdown_report_artifact",
		"artifact_id":      "art_report_session",
		"agent_executor":   "codex",
		"agent_session_id": "report-session-1",
	}, app.Producer{Type: "agent", ID: "codex"}); err != nil {
		t.Fatal(err)
	}

	if got := server.latestAgentSessionID(ctx, missionID, "codex"); got != "report-session-1" {
		t.Fatalf("expected report artifact session to become latest session, got %q", got)
	}
}

func TestIsolatedReportArtifactDoesNotReplaceResearchSession(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	handler := NewServer(app.NewService(store), Options{})
	server := handler.(*Server)
	mission, err := server.createMission(ctx, createMissionRequest{Title: "Report isolation session test"})
	if err != nil {
		t.Fatal(err)
	}
	missionID := mission.Projection.MissionID
	if _, err := appendTestEvent(t, server, ctx, missionID, "turn.agent.response", map[string]any{
		"kind":             "agent_response",
		"agent_executor":   "codex",
		"agent_session_id": "research-session-1",
	}, app.Producer{Type: "agent", ID: "codex"}); err != nil {
		t.Fatal(err)
	}
	if _, err := appendTestEvent(t, server, ctx, missionID, "report.artifact.created", map[string]any{
		"kind":                           "markdown_report_artifact",
		"artifact_id":                    "art_isolated_report_session",
		"agent_executor":                 "codex",
		"agent_session_id":               "report-session-1",
		"report_session_policy":          reportSessionPolicyIsolatedFork,
		"pre_report_research_session_id": "research-session-1",
	}, app.Producer{Type: "agent", ID: "codex"}); err != nil {
		t.Fatal(err)
	}

	if got := server.latestAgentSessionID(ctx, missionID, "codex"); got != "research-session-1" {
		t.Fatalf("expected isolated report artifact to preserve research session, got %q", got)
	}
}

func TestAgentRecallDoesNotReinjectTurnHistory(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{
		responses: []AgentResult{
			{Text: "first answer must stay inside codex session only", SessionID: "codex-session-1"},
			{Text: "second answer", SessionID: "codex-session-1", Resumed: true},
		},
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Recall test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "first"})
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "second"})
	detail := waitForEventTypeCount(t, server.URL, missionID, "turn.agent.response", 2)

	if len(agent.requests) != 2 {
		t.Fatalf("expected two agent requests, got %d", len(agent.requests))
	}
	if strings.Contains(agent.requests[1].Prompt, "first answer must stay inside codex session only") {
		t.Fatal("second prompt must not reinject previous agent output")
	}
	if strings.Contains(agent.requests[1].Prompt, "recent_turns") {
		t.Fatal("mission recall must not contain recent turn history")
	}

	for _, value := range detail["events"].([]any) {
		event := value.(map[string]any)
		if event["EventType"] != "turn.agent.response" {
			continue
		}
		payload := event["Payload"].(map[string]any)
		if _, ok := payload["recall_preview"]; ok {
			t.Fatal("agent response payload must not store recall_preview")
		}
	}
}

func TestAgentTurnRegistersExplicitSourceCandidatesFromResponse(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	agent := &fakeAgentExecutor{responses: []AgentResult{{
		Text:      "소스 후보: https://example.com/report\n채택 의견: 이 자료는 제조사 기준 문서로 케이블 요구사항의 전력 조건과 호환성 기준을 확인하는 근거가 됩니다.\nDuplicate: https://example.com/report",
		SessionID: "codex-session-1",
	}}}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Source candidate test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "research"})
	detail := waitForEventType(t, server.URL, missionID, "source.candidate.proposed")

	if countEvents(detail, "source.candidate.proposed") != 1 {
		t.Fatalf("expected one source candidate event, got %#v", detail["events"])
	}
	responsePayload := lastEventPayload(t, detail, "turn.agent.response")
	candidatePayload := lastEventPayload(t, detail, "source.candidate.proposed")
	responseEventID := ""
	for _, raw := range detail["events"].([]any) {
		event := raw.(map[string]any)
		if event["EventType"] == "turn.agent.response" {
			responseEventID = event["EventID"].(string)
		}
	}
	if responseEventID == "" || candidatePayload["agent_event_id"] != responseEventID || candidatePayload["user_event_id"] != responsePayload["user_event_id"] {
		t.Fatalf("candidate payload should point at agent response: candidate=%#v response=%#v", candidatePayload, responsePayload)
	}
	candidates := candidatePayload["candidates"].([]any)
	if len(candidates) != 1 {
		t.Fatalf("expected duplicate candidate URLs to collapse, got %#v", candidates)
	}
	candidate := candidates[0].(map[string]any)
	if candidate["url"] != "https://example.com/report" || !strings.Contains(candidate["reason"].(string), "케이블 요구사항") {
		t.Fatalf("unexpected source candidate payload: %#v", candidatePayload)
	}
}

func TestAgentTurnStagesConfluenceSourceCandidateWithMissionAccess(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	oldDefaultClient := http.DefaultClient
	fallbackTransport := oldDefaultClient.Transport
	if fallbackTransport == nil {
		fallbackTransport = http.DefaultTransport
	}
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Hostname() != "docs.atlassian.net" {
			return fallbackTransport.RoundTrip(r)
		}
		body := `{
			"id": "123",
			"title": "Roadmap",
			"spaceId": "987",
			"version": {"createdAt": "2026-07-03T02:00:00.000Z", "number": 7},
			"body": {"storage": {"value": "<p>Confluence body</p><p>Stage me before approval.</p>", "representation": "storage"}},
			"_links": {"base": "https://docs.atlassian.net/wiki", "webui": "/spaces/ENG/pages/123/Roadmap"}
		}`
		header := make(http.Header)
		header.Set("Content-Type", "application/json")
		return &http.Response{StatusCode: http.StatusOK, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	defer func() { http.DefaultClient = oldDefaultClient }()

	svc := app.NewService(store)
	agent := &fakeAgentExecutor{responses: []AgentResult{{
		Text:      "소스 후보: https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap\n채택 의견: 이 Confluence 문서는 내부 로드맵의 세부 일정과 의사결정 배경을 확인하는 데 필요합니다.",
		SessionID: "codex-session-1",
	}}}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Confluence candidate staging"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	cloudID, err := app.ConfluenceAPITokenSiteCloudID("https://docs.atlassian.net/wiki")
	if err != nil {
		t.Fatal(err)
	}
	postJSON(t, server.URL+"/api/settings/connectors/confluence/connections", map[string]any{
		"connection_id": "cnf_stage",
		"display_name":  "Docs",
		"auth_type":     app.ConfluenceAuthTypeAPIToken,
		"account_name":  "person@example.com",
		"api_token":     "secret-api-token",
		"sites":         []map[string]any{{"url": "https://docs.atlassian.net/wiki"}},
	})
	putJSON(t, server.URL+"/api/missions/"+missionID+"/connector-access/confluence", map[string]any{
		"enabled":       true,
		"connection_id": "cnf_stage",
		"cloud_id":      cloudID,
	})

	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "research"})
	detail := waitForEventType(t, server.URL, missionID, "source.candidate.staged")
	stagedPayload := lastEventPayload(t, detail, "source.candidate.staged")
	if stagedPayload["candidate_kind"] != "confluence_url" || stagedPayload["approval_state"] != "unapproved_candidate" || stagedPayload["not_report_default"] != true {
		t.Fatalf("expected unapproved Confluence staged candidate payload, got %#v", stagedPayload)
	}
	artifactID, _ := stagedPayload["artifact_id"].(string)
	if artifactID == "" {
		t.Fatalf("expected staged artifact id, got %#v", stagedPayload)
	}
	artifact, err := store.GetRawArtifact(ctx, artifactID)
	if err != nil {
		t.Fatal(err)
	}
	if artifact.MediaType != "text/plain; charset=utf-8" || !strings.Contains(string(artifact.Content), "Stage me before approval") {
		t.Fatalf("expected staged Confluence plain text artifact, got type=%q content=%q", artifact.MediaType, string(artifact.Content))
	}
	sources, err := svc.ListSourceSnapshotsWithState(ctx, app.ListSourceSnapshotsRequest{MissionID: missionID})
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 0 {
		t.Fatalf("candidate staging must not create accepted source snapshots: %#v", sources)
	}
}

func TestSourceCandidateOpinionRejectsDomainFragments(t *testing.T) {
	candidates := sourceCandidatesFromText("com https://example.com")
	if len(candidates) != 0 {
		t.Fatalf("domain fragment must not create a candidate: %#v", candidates)
	}
}

func TestSourceCandidateRequiresAcceptanceOpinion(t *testing.T) {
	candidates := sourceCandidatesFromText("Reference: https://example.com/report")
	if len(candidates) != 0 {
		t.Fatalf("URL-only text must not create a source candidate: %#v", candidates)
	}
}

func TestSourceCandidateRequiresExplicitCandidateLabel(t *testing.T) {
	candidates := sourceCandidatesFromText("Reference: https://example.com/report\n채택 의견: 이 자료는 제조사 공식 사양으로 USB-C 케이블의 전력과 영상 출력 조건을 확인할 수 있습니다.")
	if len(candidates) != 0 {
		t.Fatalf("acceptance opinion without source-candidate label must not create a source candidate: %#v", candidates)
	}
}

func TestSourceCandidateDoesNotInferAcceptanceOpinionFromProse(t *testing.T) {
	candidates := sourceCandidatesFromText("Use https://example.com/report because it is the vendor reference for cable requirements.")
	if len(candidates) != 0 {
		t.Fatalf("prose around a URL must not create a source candidate: %#v", candidates)
	}
}

func TestSourceCandidateExtractsExplicitAcceptanceOpinion(t *testing.T) {
	candidates := sourceCandidatesFromText("소스 후보: https://example.com/spec\n채택 의견: 이 자료는 제조사 공식 사양으로 USB-C 케이블의 전력과 영상 출력 조건을 확인할 수 있습니다.")
	if len(candidates) != 1 {
		t.Fatalf("expected one candidate, got %#v", candidates)
	}
	if !strings.Contains(candidates[0].Reason, "제조사 공식 사양") {
		t.Fatalf("expected explicit source candidate acceptance opinion, got %#v", candidates[0])
	}
}

func TestURLSourceSnapshotFetchesOriginalMaterial(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{
		urlFetcher: func(_ context.Context, rawURL string) (fetchedURLSource, error) {
			if rawURL != "https://example.com/paper" {
				t.Fatalf("expected normalized fetch URL, got %q", rawURL)
			}
			return fetchedURLSource{
				Content:         []byte("<!doctype html><title>Source Title</title><main>original source body</main>"),
				MediaType:       "text/html; charset=utf-8",
				Title:           "Source Title",
				ExternalVersion: "etag=source-v1",
			}, nil
		},
	}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "URL source test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	result := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/url", map[string]any{
		"url": "https://example.com/paper#fragment",
	})

	if got := nestedString(t, result, "snapshot", "Connector", "ExternalURI"); got != "https://example.com/paper" {
		t.Fatalf("expected normalized URL without fragment, got %q", got)
	}
	if got := nestedString(t, result, "snapshot", "Title"); got != "Source Title" {
		t.Fatalf("expected HTML title, got %q", got)
	}
	artifactID := nestedString(t, result, "artifact", "ArtifactID")
	artifact, err := store.GetRawArtifact(ctx, artifactID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(artifact.Content), "original source body") {
		t.Fatalf("expected fetched source content, got %q", string(artifact.Content))
	}
}

func TestURLSourceSnapshotRoutesConfluencePageURLToConnector(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	oldDefaultClient := http.DefaultClient
	fallbackTransport := oldDefaultClient.Transport
	if fallbackTransport == nil {
		fallbackTransport = http.DefaultTransport
	}
	var authHeaders []string
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Hostname() != "docs.atlassian.net" {
			return fallbackTransport.RoundTrip(r)
		}
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		body := `{
			"id": "123",
			"title": "Roadmap",
			"spaceId": "987",
			"version": {"createdAt": "2026-07-03T02:00:00.000Z", "number": 7},
			"body": {"storage": {"value": "<p>Confluence body</p>", "representation": "storage"}},
			"_links": {"base": "https://docs.atlassian.net/wiki", "webui": "/spaces/ENG/pages/123/Roadmap"}
		}`
		header := make(http.Header)
		header.Set("Content-Type", "application/json")
		return &http.Response{StatusCode: http.StatusOK, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	defer func() { http.DefaultClient = oldDefaultClient }()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Confluence URL source test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/settings/connectors/confluence/connections", map[string]any{
		"connection_id": "cnf_web",
		"display_name":  "Docs",
		"auth_type":     app.ConfluenceAuthTypeAPIToken,
		"account_name":  "person@example.com",
		"api_token":     "secret-api-token",
		"sites": []map[string]any{{
			"url": "https://docs.atlassian.net/wiki",
		}},
	})

	result := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/url", map[string]any{
		"url": "https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap#ignored",
	})
	if got := nestedString(t, result, "Snapshot", "Connector", "ConnectorID"); got != app.ConfluenceConnectorID {
		t.Fatalf("expected Confluence snapshot connector, got %q result=%#v", got, result)
	}
	if got := nestedString(t, result, "Snapshot", "Connector", "ExternalSourceID"); got != "site_docs.atlassian.net:123" {
		t.Fatalf("expected Confluence page identity, got %q", got)
	}
	reused := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/url", map[string]any{
		"url":           "https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap",
		"connection_id": "cnf_web",
		"cloud_id":      "site_docs.atlassian.net",
	})
	if existing, _ := nestedValue(t, reused, "existing").(bool); !existing {
		t.Fatalf("expected repeated Confluence URL source approval to reuse existing source, got %#v", reused)
	}
	if got := nestedString(t, reused, "snapshot", "SnapshotID"); got != nestedString(t, result, "Snapshot", "SnapshotID") {
		t.Fatalf("expected existing Confluence snapshot reuse, got %q want %q", got, nestedString(t, result, "Snapshot", "SnapshotID"))
	}
	artifactID := nestedString(t, result, "Artifact", "ArtifactID")
	artifact, err := store.GetRawArtifact(ctx, artifactID)
	if err != nil {
		t.Fatal(err)
	}
	if artifact.MediaType != app.ConfluenceSnapshotMediaType || !strings.Contains(string(artifact.Content), "Confluence body") {
		t.Fatalf("expected Confluence snapshot artifact, got type=%q content=%q", artifact.MediaType, string(artifact.Content))
	}
	for _, auth := range authHeaders {
		if auth != "" && !strings.HasPrefix(auth, "Basic ") {
			t.Fatalf("unexpected auth header %q", auth)
		}
	}
}

func TestConfluenceURLSourceRouteSnapshotsPageURL(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	oldDefaultClient := http.DefaultClient
	fallbackTransport := oldDefaultClient.Transport
	if fallbackTransport == nil {
		fallbackTransport = http.DefaultTransport
	}
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Hostname() != "docs.atlassian.net" {
			return fallbackTransport.RoundTrip(r)
		}
		body := `{
			"id": "123",
			"title": "Roadmap",
			"spaceId": "987",
			"version": {"createdAt": "2026-07-03T02:00:00.000Z", "number": 7},
			"body": {"storage": {"value": "<p>Confluence body</p>", "representation": "storage"}},
			"_links": {"base": "https://docs.atlassian.net/wiki", "webui": "/spaces/ENG/pages/123/Roadmap"}
		}`
		header := make(http.Header)
		header.Set("Content-Type", "application/json")
		return &http.Response{StatusCode: http.StatusOK, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	defer func() { http.DefaultClient = oldDefaultClient }()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Confluence URL route test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	cloudID, err := app.ConfluenceAPITokenSiteCloudID("https://docs.atlassian.net/wiki")
	if err != nil {
		t.Fatal(err)
	}
	postJSON(t, server.URL+"/api/settings/connectors/confluence/connections", map[string]any{
		"connection_id": "cnf_web",
		"display_name":  "Docs",
		"auth_type":     app.ConfluenceAuthTypeAPIToken,
		"account_name":  "person@example.com",
		"api_token":     "secret-api-token",
		"sites": []map[string]any{{
			"url": "https://docs.atlassian.net/wiki",
		}},
	})

	result := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/url", map[string]any{
		"url":           "https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap",
		"connection_id": "cnf_web",
		"cloud_id":      cloudID,
	})
	if got := nestedString(t, result, "Snapshot", "Connector", "ConnectorID"); got != app.ConfluenceConnectorID {
		t.Fatalf("expected Confluence snapshot connector, got %q result=%#v", got, result)
	}
}

func TestConfluenceURLSourceRouteUsesRequestTitleAsFallback(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	oldDefaultClient := http.DefaultClient
	fallbackTransport := oldDefaultClient.Transport
	if fallbackTransport == nil {
		fallbackTransport = http.DefaultTransport
	}
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Hostname() != "docs.atlassian.net" {
			return fallbackTransport.RoundTrip(r)
		}
		body := `{
			"id": "123",
			"title": "",
			"spaceId": "987",
			"version": {"createdAt": "2026-07-03T02:00:00.000Z", "number": 7},
			"body": {"storage": {"value": "<p>Confluence body</p>", "representation": "storage"}},
			"_links": {"base": "https://docs.atlassian.net/wiki", "webui": "/spaces/ENG/pages/123/Roadmap"}
		}`
		header := make(http.Header)
		header.Set("Content-Type", "application/json")
		return &http.Response{StatusCode: http.StatusOK, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	defer func() { http.DefaultClient = oldDefaultClient }()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Confluence URL title fallback test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	cloudID, err := app.ConfluenceAPITokenSiteCloudID("https://docs.atlassian.net/wiki")
	if err != nil {
		t.Fatal(err)
	}
	postJSON(t, server.URL+"/api/settings/connectors/confluence/connections", map[string]any{
		"connection_id": "cnf_web",
		"display_name":  "Docs",
		"auth_type":     app.ConfluenceAuthTypeAPIToken,
		"account_name":  "person@example.com",
		"api_token":     "secret-api-token",
		"sites": []map[string]any{{
			"url": "https://docs.atlassian.net/wiki",
		}},
	})

	result := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/url", map[string]any{
		"url":           "https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap",
		"title":         "Candidate Roadmap",
		"connection_id": "cnf_web",
		"cloud_id":      cloudID,
	})
	if got := nestedString(t, result, "Snapshot", "Title"); got != "Candidate Roadmap" {
		t.Fatalf("expected request title fallback, got %q result=%#v", got, result)
	}
}

func TestConfluenceURLSourceRouteRejectsSelectedSiteMismatch(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Confluence URL selected site mismatch test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	otherCloudID, err := app.ConfluenceAPITokenSiteCloudID("https://other.atlassian.net/wiki")
	if err != nil {
		t.Fatal(err)
	}
	postJSON(t, server.URL+"/api/settings/connectors/confluence/connections", map[string]any{
		"connection_id": "cnf_web",
		"display_name":  "Docs",
		"auth_type":     app.ConfluenceAuthTypeAPIToken,
		"account_name":  "person@example.com",
		"api_token":     "secret-api-token",
		"sites": []map[string]any{{
			"url": "https://docs.atlassian.net/wiki",
		}},
	})

	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/url", map[string]any{
		"url":           "https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap",
		"connection_id": "cnf_web",
		"cloud_id":      otherCloudID,
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %#v", status, body)
	}
	if message := nestedString(t, body, "error", "message"); !strings.Contains(message, "일치하지 않습니다") {
		t.Fatalf("expected selected site mismatch guidance, got %q", message)
	}
}

func TestURLSourceSnapshotRejectsConfluenceURLWithoutPageID(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Confluence URL source reject test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/sources/url", map[string]any{
		"url": "https://docs.atlassian.net/wiki",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %#v", status, body)
	}
	if message := nestedString(t, body, "error", "message"); !strings.Contains(message, "page id") {
		t.Fatalf("expected page id guidance, got %q", message)
	}
}

func TestURLSourceSnapshotRejectsConfluencePageIDMismatch(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	oldDefaultClient := http.DefaultClient
	fallbackTransport := oldDefaultClient.Transport
	if fallbackTransport == nil {
		fallbackTransport = http.DefaultTransport
	}
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Hostname() != "docs.atlassian.net" {
			return fallbackTransport.RoundTrip(r)
		}
		body := `{
			"id": "998",
			"title": "Wrong page",
			"spaceId": "987",
			"version": {"createdAt": "2026-07-03T02:00:00.000Z", "number": 7},
			"body": {"storage": {"value": "<p>wrong body</p>", "representation": "storage"}},
			"_links": {"base": "https://docs.atlassian.net/wiki", "webui": "/spaces/ENG/pages/998/Wrong"}
		}`
		header := make(http.Header)
		header.Set("Content-Type", "application/json")
		return &http.Response{StatusCode: http.StatusOK, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	defer func() { http.DefaultClient = oldDefaultClient }()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Confluence URL mismatch test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/settings/connectors/confluence/connections", map[string]any{
		"connection_id": "cnf_web",
		"display_name":  "Docs",
		"auth_type":     app.ConfluenceAuthTypeAPIToken,
		"account_name":  "person@example.com",
		"api_token":     "secret-api-token",
		"sites": []map[string]any{{
			"url": "https://docs.atlassian.net/wiki",
		}},
	})

	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/sources/url", map[string]any{
		"url": "https://docs.atlassian.net/wiki/spaces/ENG/pages/999/Roadmap",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %#v", status, body)
	}
	if message := nestedString(t, body, "error", "message"); !strings.Contains(message, "page id") && !strings.Contains(message, "URL") {
		t.Fatalf("expected page id mismatch guidance, got %q", message)
	}
}

func TestURLSourceSnapshotRejectsAmbiguousConfluenceConnections(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Confluence URL ambiguous connection test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	for _, connectionID := range []string{"cnf_web_one", "cnf_web_two"} {
		postJSON(t, server.URL+"/api/settings/connectors/confluence/connections", map[string]any{
			"connection_id": connectionID,
			"display_name":  connectionID,
			"auth_type":     app.ConfluenceAuthTypeAPIToken,
			"account_name":  "person@example.com",
			"api_token":     "secret-api-token",
			"sites": []map[string]any{{
				"url": "https://docs.atlassian.net/wiki",
			}},
		})
	}

	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/sources/confluence/url", map[string]any{
		"url": "https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %#v", status, body)
	}
	if message := nestedString(t, body, "error", "message"); !strings.Contains(message, "둘 이상") {
		t.Fatalf("expected ambiguous connection guidance, got %q", message)
	}
}

func TestURLSourceSnapshotReportsUnauthorizedAsActionableSourceFailure(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer sourceServer.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{
		urlFetcher: func(ctx context.Context, rawURL string) (fetchedURLSource, error) {
			return fetchURLSourceWithClient(ctx, rawURL, sourceServer.Client())
		},
	}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "URL source unauthorized test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/sources/url", map[string]any{
		"url": sourceServer.URL + "/locked",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %#v", status, body)
	}
	message := nestedString(t, body, "error", "message")
	if strings.Contains(message, "invalid input") {
		t.Fatalf("message must not expose implementation prefix: %q", message)
	}
	for _, expected := range []string{"인증", "HTTP 401", "텍스트 소스"} {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected %q in actionable message, got %q", expected, message)
		}
	}
	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	if countEvents(detail, "source.snapshot_failed") != 1 {
		t.Fatalf("expected one source snapshot failure event, got %#v", detail["events"])
	}
	payload := lastEventPayload(t, detail, "source.snapshot_failed")
	if payload["source_kind"] != "url" || payload["url"] != sourceServer.URL+"/locked" {
		t.Fatalf("unexpected source failure payload: %#v", payload)
	}
	if !strings.Contains(payload["message"].(string), "HTTP 401") {
		t.Fatalf("expected failure message to include HTTP status, got %#v", payload)
	}
}

func TestURLSourceSnapshotReusesExistingContentHash(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{
		urlFetcher: func(_ context.Context, _ string) (fetchedURLSource, error) {
			return fetchedURLSource{
				Content:   []byte("same original source body"),
				MediaType: "text/plain; charset=utf-8",
				Title:     "Same source",
			}, nil
		},
	}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Same content URL source"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	first := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/url", map[string]any{
		"url": "https://example.com/source-a",
	})
	second := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/url", map[string]any{
		"url": "https://example.org/source-b",
	})

	if existing, _ := second["existing"].(bool); !existing {
		t.Fatalf("expected second same-content URL to reuse existing source, got %#v", second)
	}
	if nestedString(t, second, "snapshot", "SnapshotID") != nestedString(t, first, "snapshot", "SnapshotID") {
		t.Fatalf("expected same snapshot to be returned, first=%#v second=%#v", first, second)
	}
	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	if countEvents(detail, "source.snapshotted") != 1 {
		t.Fatalf("expected only one source snapshot event, got %#v", detail["events"])
	}
}

func TestURLSourceSnapshotRejectsBlockedNetworks(t *testing.T) {
	for _, rawURL := range []string{
		"http://127.0.0.1:1/",
		"http://10.0.0.1/",
		"http://172.16.0.1/",
		"http://192.168.0.1/",
		"http://169.254.169.254/latest/meta-data/",
		"http://100.64.0.1/",
		"http://[::1]/",
	} {
		t.Run(rawURL, func(t *testing.T) {
			_, err := fetchURLSource(context.Background(), rawURL)
			if err == nil {
				t.Fatal("expected blocked network error")
			}
			if !strings.Contains(err.Error(), "blocked address") {
				t.Fatalf("expected blocked address error, got %v", err)
			}
		})
	}
}

func TestURLSourceSnapshotAllowsReasonableLargeTextSource(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	content := strings.Repeat("a", (1<<20)+128)
	sourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(content))
	}))
	defer sourceServer.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{
		urlFetcher: func(ctx context.Context, rawURL string) (fetchedURLSource, error) {
			return fetchURLSourceWithClient(ctx, rawURL, sourceServer.Client())
		},
	}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Large URL source test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	result := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/url", map[string]any{
		"url": sourceServer.URL + "/large",
	})
	artifactID := nestedString(t, result, "artifact", "ArtifactID")
	artifact, err := store.GetRawArtifact(ctx, artifactID)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifact.Content) != len(content) {
		t.Fatalf("expected %d bytes, got %d", len(content), len(artifact.Content))
	}
}

func TestURLSourceSnapshotRejectsTooLargeTextSource(t *testing.T) {
	sourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Length", "20971521")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("x", 128)))
	}))
	defer sourceServer.Close()

	_, err := fetchURLSourceWithClient(context.Background(), sourceServer.URL+"/too-large", sourceServer.Client())
	if err == nil {
		t.Fatal("expected too-large source error")
	}
	if !strings.Contains(err.Error(), "larger than 20 MiB") {
		t.Fatalf("expected 20 MiB error, got %v", err)
	}
}

func TestURLSourceSnapshotAcceptsPDFCandidateContent(t *testing.T) {
	pdfBytes := testPDFBytes(t, []string{"PDF candidate content", "Alpha code is 93."})
	sourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(pdfBytes)
	}))
	defer sourceServer.Close()

	fetched, err := fetchURLSourceWithClient(context.Background(), sourceServer.URL+"/paper.pdf", sourceServer.Client())
	if err != nil {
		t.Fatalf("fetchURLSourceWithClient returned error: %v", err)
	}
	if fetched.MediaType != "application/pdf" || fetched.PageCount != 1 || fetched.ByteSize != int64(len(pdfBytes)) {
		t.Fatalf("expected PDF candidate metadata, got %#v", fetched)
	}
}

func TestURLSourceSnapshotReportsTimeoutClearly(t *testing.T) {
	sourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("slow source"))
	}))
	defer sourceServer.Close()

	client := &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: 10 * time.Millisecond,
		},
	}
	_, err := fetchURLSourceWithClient(context.Background(), sourceServer.URL+"/slow", client)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "URL 원문 응답이 제한 시간 내 도착하지 않았습니다") {
		t.Fatalf("expected clear timeout message, got %v", err)
	}
}

func TestURLSourceSnapshotReusesExistingNormalizedURL(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	var fetches atomic.Int64
	server := httptest.NewServer(NewServer(app.NewService(store), Options{
		urlFetcher: func(_ context.Context, rawURL string) (fetchedURLSource, error) {
			fetches.Add(1)
			return fetchedURLSource{
				Content:   []byte("same source"),
				MediaType: "text/plain; charset=utf-8",
				Title:     rawURL,
			}, nil
		},
	}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Duplicate URL source test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/url", map[string]any{
		"url": "https://example.com/source#first",
	})
	second := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/url", map[string]any{
		"url": "https://example.com/source#second",
	})
	if existing, ok := second["existing"].(bool); !ok || !existing {
		t.Fatalf("expected existing response, got %#v", second)
	}
	if fetches.Load() != 1 {
		t.Fatalf("expected one fetch, got %d", fetches.Load())
	}
	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	if sources := detail["sources"].([]any); len(sources) != 1 {
		t.Fatalf("expected one source snapshot, got %d", len(sources))
	}
}

func TestURLSourceSnapshotConcurrentDuplicateUsesOneFetch(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	var fetches atomic.Int64
	server := httptest.NewServer(NewServer(app.NewService(store), Options{
		urlFetcher: func(_ context.Context, rawURL string) (fetchedURLSource, error) {
			fetches.Add(1)
			time.Sleep(80 * time.Millisecond)
			return fetchedURLSource{
				Content:   []byte("same source"),
				MediaType: "text/plain; charset=utf-8",
				Title:     rawURL,
			}, nil
		},
	}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Concurrent duplicate URL source test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	statuses := make([]int, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range statuses {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			statuses[index] = postJSONStatus(t, server.URL+"/api/missions/"+missionID+"/sources/url", map[string]any{
				"url": "https://example.com/source#fragment",
			})
		}(i)
	}
	close(start)
	wg.Wait()

	for _, status := range statuses {
		if status != http.StatusCreated && status != http.StatusOK {
			t.Fatalf("expected 200 or 201, got statuses %#v", statuses)
		}
	}
	if fetches.Load() != 1 {
		t.Fatalf("expected one fetch, got %d", fetches.Load())
	}
	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	if sources := detail["sources"].([]any); len(sources) != 1 {
		t.Fatalf("expected one source snapshot, got %d", len(sources))
	}
}

func TestRejectSourceCandidateRecordsDecisionEvent(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Reject candidate test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	result := postJSON(t, server.URL+"/api/missions/"+missionID+"/candidates/sources/reject", map[string]any{
		"url":    "HTTPS://Example.com/path#part",
		"reason": "이미 더 좋은 공식 문서를 소스로 붙였습니다.",
	})
	payload := nestedMap(t, result, "event", "Payload")
	if got := payload["url"]; got != "https://example.com/path" {
		t.Fatalf("expected normalized rejected URL, got %#v", got)
	}
	if got := payload["reason"]; got != "이미 더 좋은 공식 문서를 소스로 붙였습니다." {
		t.Fatalf("expected custom rejection reason, got %#v", got)
	}

	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	if countEvents(detail, "source.candidate.rejected") != 1 {
		t.Fatalf("expected one rejected candidate event, got %#v", detail["events"])
	}
}

func TestRestoreSourceCandidateRecordsDecisionEvent(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Restore candidate test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/candidates/sources/reject", map[string]any{
		"url": "https://example.com/source",
	})
	result := postJSON(t, server.URL+"/api/missions/"+missionID+"/candidates/sources/restore", map[string]any{
		"url": "https://example.com/source#ignored",
	})
	payload := nestedMap(t, result, "event", "Payload")
	if got := payload["url"]; got != "https://example.com/source" {
		t.Fatalf("expected normalized restored URL, got %#v", got)
	}
	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	if countEvents(detail, "source.candidate.restored") != 1 {
		t.Fatalf("expected one restored candidate event, got %#v", detail["events"])
	}
}

func TestLocalPathSourceWebWorkflow(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "notes.md"), []byte("hello local path source\nsecond line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	engine, err := localpath.New(localpath.Config{Roots: []localpath.RootConfig{{RootID: "workspace", Path: root, Alias: "Workspace"}}})
	if err != nil {
		t.Fatal(err)
	}
	svc := app.NewServiceWithLocalPathEngine(store, engine)
	server := httptest.NewServer(NewServer(svc, Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Local path source"})
	missionID := nestedString(t, mission, "projection", "mission_id")

	roots := getJSON(t, server.URL+"/api/local_path/roots")
	encodedRoots, _ := json.Marshal(roots)
	if !strings.Contains(string(encodedRoots), "workspace") || strings.Contains(string(encodedRoots), root) {
		t.Fatalf("roots output should expose root id only, got %s", string(encodedRoots))
	}

	tree := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/local_path/tree", map[string]any{
		"root_id":       "workspace",
		"relative_path": "docs",
		"depth":         1,
		"limit":         20,
	})
	encodedTree, _ := json.Marshal(tree)
	if !strings.Contains(string(encodedTree), "docs/notes.md") || strings.Contains(string(encodedTree), root) {
		t.Fatalf("tree output should use relative paths only, got %s", string(encodedTree))
	}

	status, body := postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/sources/local_path/tree", map[string]any{
		"root_id":       "workspace",
		"relative_path": root,
	})
	if status != http.StatusBadRequest || strings.Contains(toJSON(t, body), root) {
		t.Fatalf("expected absolute path rejection without root leak, got %d %#v", status, body)
	}

	attach := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/local_path", map[string]any{
		"root_id":       "workspace",
		"relative_path": "docs/notes.md",
		"title":         "Notes",
	})
	snapshotID := nestedString(t, attach, "snapshot", "SnapshotID")
	if policy := nestedString(t, attach, "snapshot", "Access", "RetrievalPolicy"); policy != app.SourceRetrievalPolicyLiveReference {
		t.Fatalf("expected live reference policy, got %q", policy)
	}
	encodedAttach, _ := json.Marshal(attach)
	if strings.Contains(string(encodedAttach), root) {
		t.Fatalf("attach output leaked absolute root: %s", string(encodedAttach))
	}

	duplicate := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/local_path", map[string]any{
		"root_id":       "workspace",
		"relative_path": "docs/notes.md",
	})
	if existing, _ := duplicate["existing"].(bool); !existing {
		t.Fatalf("expected duplicate active attach to return existing, got %#v", duplicate)
	}

	read := getJSON(t, server.URL+"/api/missions/"+missionID+"/sources/"+snapshotID+"/read?max_bytes=5")
	if content := nestedString(t, read, "content"); content != "hello" {
		t.Fatalf("expected bounded live read content, got %q", content)
	}
	if eventID := nestedString(t, read, "observation_event_id"); eventID == "" {
		t.Fatalf("expected observation event id, got %#v", read)
	}
	encodedRead, _ := json.Marshal(read)
	if strings.Contains(string(encodedRead), root) {
		t.Fatalf("read output leaked absolute root: %s", string(encodedRead))
	}

	remove := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/"+snapshotID+"/remove", map[string]any{"reason": "test removal"})
	if removed := nestedString(t, remove, "snapshot", "State", "state"); removed != app.SourceStateRemoved {
		t.Fatalf("expected removed source state, got %#v", remove)
	}
	listDefault := getJSON(t, server.URL+"/api/missions/"+missionID+"/sources")
	if strings.Contains(toJSON(t, listDefault), snapshotID) {
		t.Fatalf("removed source should be hidden by default: %#v", listDefault)
	}
	listRemoved := getJSON(t, server.URL+"/api/missions/"+missionID+"/sources?include_removed=true")
	if !strings.Contains(toJSON(t, listRemoved), snapshotID) {
		t.Fatalf("include_removed should show removed source: %#v", listRemoved)
	}
	status, _ = getJSONFailure(t, server.URL+"/api/missions/"+missionID+"/sources/"+snapshotID+"/read")
	if status != http.StatusBadRequest {
		t.Fatalf("expected removed source read rejection, got %d", status)
	}
	status, body = postJSONFailure(t, server.URL+"/api/missions/"+missionID+"/sources/local_path", map[string]any{
		"root_id":       "workspace",
		"relative_path": "docs/notes.md",
	})
	if status != http.StatusConflict || !strings.Contains(nestedString(t, body, "error", "message"), "restore") {
		t.Fatalf("expected restore-required conflict, got %d %#v", status, body)
	}
	restored := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/"+snapshotID+"/restore", map[string]any{})
	if state := nestedString(t, restored, "snapshot", "State", "state"); state != app.SourceStateActive {
		t.Fatalf("expected restored active state, got %#v", restored)
	}

	dirAttach := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/local_path", map[string]any{
		"root_id":       "workspace",
		"relative_path": "docs",
		"title":         "Docs directory",
	})
	dirSnapshotID := nestedString(t, dirAttach, "snapshot", "SnapshotID")
	dirRead := getJSON(t, server.URL+"/api/missions/"+missionID+"/sources/"+dirSnapshotID+"/read?depth=1&limit=20")
	if !strings.Contains(toJSON(t, dirRead), "docs/notes.md") || nestedString(t, dirRead, "observation_event_id") == "" {
		t.Fatalf("expected directory source read to return tree observation, got %#v", dirRead)
	}
}

func TestAgentTurnReturnsPendingBeforeBackgroundCompletes(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	release := make(chan struct{})
	agent := blockingAgentExecutor{
		release: release,
		result:  AgentResult{Text: "background answer", SessionID: "codex-session-1"},
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Background test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	response := postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "slow"})
	if pendingType := nestedString(t, response, "pending_event", "EventType"); pendingType != "turn.agent.pending" {
		t.Fatalf("expected pending event, got %q", pendingType)
	}

	detail := getJSON(t, server.URL+"/api/missions/"+missionID)
	if countEvents(detail, "turn.agent.pending") != 1 {
		t.Fatalf("expected one pending event, got %#v", detail["events"])
	}
	if countEvents(detail, "turn.agent.response") != 0 {
		t.Fatalf("expected no agent response before release, got %#v", detail["events"])
	}

	close(release)
	waitForEventType(t, server.URL, missionID, "turn.agent.response")
}

func TestConcurrentAgentTurnsRejectSecondRequest(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	release := make(chan struct{})
	agent := blockingAgentExecutor{
		release: release,
		result:  AgentResult{Text: "done", SessionID: "codex-session-1"},
	}
	server := httptest.NewServer(NewServer(app.NewService(store), Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Concurrent test"})
	missionID := nestedString(t, mission, "projection", "mission_id")

	var accepted atomic.Int64
	var conflict atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status := postJSONStatus(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "slow"})
			switch status {
			case http.StatusAccepted:
				accepted.Add(1)
			case http.StatusConflict:
				conflict.Add(1)
			default:
				t.Errorf("unexpected status %d", status)
			}
		}()
	}
	wg.Wait()
	close(release)
	waitForEventType(t, server.URL, missionID, "turn.agent.response")

	if accepted.Load() != 1 || conflict.Load() != 1 {
		t.Fatalf("expected one accepted and one conflict, got accepted=%d conflict=%d", accepted.Load(), conflict.Load())
	}
}

func TestAgentFailureClosesPendingTurn(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{
		AgentExecutor: errorAgentExecutor{err: errors.New("boom")},
	}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Error closes pending"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "first"})
	detail := waitForEventType(t, server.URL, missionID, "turn.agent.response")
	payload := firstEventPayload(t, detail, "turn.agent.response")
	if payload["kind"] != "agent_error" {
		t.Fatalf("expected agent_error response, got %#v", payload)
	}
	if hasOpenPendingDetail(detail) {
		t.Fatal("expected pending turn to be closed by agent_error")
	}
}

func TestStaleAgentTurnAutoClosesBeforeNewTurn(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	agent := &fakeAgentExecutor{responses: []AgentResult{{Text: "new answer", SessionID: "agent-session-1"}}}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Stale agent turn"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	appendStaleAgentPending(t, ctx, svc, missionID, "evt_stale_user", "evt_stale_pending")

	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "continue"})
	detail := waitForEventTypeCount(t, server.URL, missionID, "turn.agent.response", 2)
	firstPayload := firstEventPayload(t, detail, "turn.agent.response")
	if firstPayload["kind"] != "agent_canceled" || firstPayload["user_event_id"] != "evt_stale_user" {
		t.Fatalf("expected stale turn to be auto-canceled before new turn, got %#v", firstPayload)
	}
	lastPayload := lastEventPayload(t, detail, "turn.agent.response")
	if lastPayload["kind"] != "agent_response" {
		t.Fatalf("expected new turn response after stale cleanup, got %#v", lastPayload)
	}
	if hasOpenPendingDetail(detail) {
		t.Fatal("expected no open pending turn after stale cleanup and new response")
	}
}

func TestStaleAgentTurnAutoClosesBeforeReportDraft(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	agent := &fakeAgentExecutor{responses: []AgentResult{{Text: "# Report\n\nStale turn no longer blocks report.", SessionID: "agent-session-1"}}}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: agent}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Stale turn report"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	appendStaleAgentPending(t, ctx, svc, missionID, "evt_stale_report_user", "evt_stale_report_pending")

	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":          "Stale turn report",
		"agent_executor": "codex",
		"mcp_mode":       "auto",
		"report_mode":    "one_take",
	})
	detail := waitForEventType(t, server.URL, missionID, "report.draft.pending")
	payload := firstEventPayload(t, detail, "turn.agent.response")
	if payload["kind"] != "agent_canceled" || payload["user_event_id"] != "evt_stale_report_user" {
		t.Fatalf("expected stale turn to be auto-canceled before report draft, got %#v", payload)
	}
}

func TestCancelAgentTurnClosesPendingTurn(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	release := make(chan struct{})
	server := httptest.NewServer(NewServer(app.NewService(store), Options{
		AgentExecutor: blockingAgentExecutor{release: release, result: AgentResult{Text: "late"}},
	}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Cancel test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "slow"})
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns/cancel", map[string]any{"agent_executor": "claude"})
	detail := waitForEventType(t, server.URL, missionID, "turn.agent.response")
	payload := firstEventPayload(t, detail, "turn.agent.response")
	if payload["kind"] != "agent_canceled" {
		t.Fatalf("expected agent_canceled response, got %#v", payload)
	}
	if hasOpenPendingDetail(detail) {
		t.Fatal("expected pending turn to be closed by cancel")
	}
	close(release)
}

func TestAgentFailureLogExcerptKeepsTail(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{
		AgentExecutor: errorAgentExecutor{
			result: AgentResult{Log: strings.Repeat("head", 1500) + "TAIL_CAUSE"},
			err:    errors.New("boom"),
		},
	}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Log excerpt test"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{"text": "first"})
	detail := waitForEventType(t, server.URL, missionID, "turn.agent.response")
	payload := firstEventPayload(t, detail, "turn.agent.response")
	excerpt, ok := payload["log_excerpt"].(string)
	if !ok || !strings.Contains(excerpt, "TAIL_CAUSE") {
		t.Fatalf("expected log excerpt to keep tail, got %#v", payload)
	}
}

func TestCrossOriginWriteRejected(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	body := bytes.NewBufferString(`{"title":"blocked"}`)
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/missions", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.test")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestJSONContentTypeRequiredForWrites(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/missions", "text/plain", bytes.NewBufferString(`{"title":"blocked"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", resp.StatusCode)
	}
}

func TestWorkspaceResponsesDisableBrowserCaching(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	for _, path := range []string{"/api/missions", "/", "/static/app.js", "/static/confluence.js"} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if got := resp.Header.Get("Cache-Control"); got != "no-store" {
			t.Fatalf("expected Cache-Control no-store for %s, got %q", path, got)
		}
	}
}

func TestJSONContentTypeRequiredForProposalDecision(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := httptest.NewServer(NewServer(app.NewService(store), Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{
		"title": "DNS test",
	})
	missionID := nestedString(t, mission, "projection", "mission_id")
	source := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/text", map[string]any{
		"title":   "IANA note",
		"content": "HTTPS is DNS RR type 65.",
	})
	proposal := postJSON(t, server.URL+"/api/missions/"+missionID+"/candidates/evidence", map[string]any{
		"summary":     "HTTPS is DNS RR type 65.",
		"snapshot_id": nestedString(t, source, "snapshot", "SnapshotID"),
		"artifact_id": nestedString(t, source, "artifact", "ArtifactID"),
	})
	proposalID := nestedString(t, proposal, "Proposal", "proposal_id")

	resp, err := http.Post(server.URL+"/api/missions/"+missionID+"/proposals/"+proposalID+"/approve", "text/plain", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", resp.StatusCode)
	}
}

func postJSON(t *testing.T, url string, body any) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("POST %s returned %d: %#v", url, resp.StatusCode, decoded)
	}
	return decoded
}

func postJSONStatus(t *testing.T, url string, body any) int {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Errorf("marshal body: %v", err)
		return 0
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(encoded))
	if err != nil {
		t.Errorf("POST %s: %v", url, err)
		return 0
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func postJSONFailure(t *testing.T, url string, body any) (int, map[string]any) {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.Fatalf("POST %s unexpectedly returned %d: %#v", url, resp.StatusCode, decoded)
	}
	return resp.StatusCode, decoded
}

func deleteJSON(t *testing.T, url string) map[string]any {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("DELETE %s returned %d: %#v", url, resp.StatusCode, decoded)
	}
	return decoded
}

func deleteJSONBody(t *testing.T, url string, body any) map[string]any {
	t.Helper()
	status, decoded := deleteJSONBodyFailure(t, url, body)
	if status < 200 || status >= 300 {
		t.Fatalf("DELETE %s returned %d: %#v", url, status, decoded)
	}
	return decoded
}

func deleteJSONBodyFailure(t *testing.T, url string, body any) (int, map[string]any) {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, decoded
}

func patchJSON(t *testing.T, url string, payload any) map[string]any {
	status, result := patchJSONFailure(t, url, payload)
	if status != http.StatusOK {
		t.Fatalf("PATCH %s returned %d: %#v", url, status, result)
	}
	return result
}

func patchJSONFailure(t *testing.T, url string, payload any) (int, map[string]any) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var result map[string]any
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, result
}

func putJSON(t *testing.T, url string, payload any) map[string]any {
	t.Helper()
	body := mustMarshalTestJSON(t, payload)
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("PUT %s returned %d: %#v", url, resp.StatusCode, decoded)
	}
	return decoded
}

func putJSONFailure(t *testing.T, url string, payload any) (int, map[string]any) {
	t.Helper()
	body := mustMarshalTestJSON(t, payload)
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.Fatalf("PUT %s unexpectedly succeeded: %#v", url, decoded)
	}
	return resp.StatusCode, decoded
}

func postMultipartFile(t *testing.T, url string, filename string, contentType string, content []byte, title string) map[string]any {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if title != "" {
		if err := writer.WriteField("title", title); err != nil {
			t.Fatal(err)
		}
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Test-Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("POST multipart %s returned %d: %#v", url, resp.StatusCode, decoded)
	}
	return decoded
}

func postMultipartFileFailure(t *testing.T, url string, filename string, content []byte) (int, map[string]any) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.Fatalf("POST multipart %s unexpectedly returned %d: %#v", url, resp.StatusCode, decoded)
	}
	return resp.StatusCode, decoded
}

func testPNGBytes() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xdd, 0x8d,
		0xb0, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}
}

func getJSON(t *testing.T, url string) map[string]any {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("GET %s returned %d: %#v", url, resp.StatusCode, decoded)
	}
	return decoded
}

func getJSONFailure(t *testing.T, url string) (int, map[string]any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.Fatalf("GET %s unexpectedly returned %d: %#v", url, resp.StatusCode, decoded)
	}
	return resp.StatusCode, decoded
}

func appendLegacyCodexSession(t *testing.T, ctx context.Context, svc *app.Service, missionID string, sessionID string) {
	t.Helper()
	_, err := svc.AppendEvent(ctx, conversation.BuildTurnAgentResponseAppendRequest(conversation.TurnAgentResponseEventRequest{
		EventID:               newID("evt"),
		MissionID:             missionID,
		Kind:                  "agent_response",
		AgentExecutor:         "codex",
		Text:                  "legacy response",
		AgentSessionID:        sessionID,
		IncludeAgentSessionID: true,
		Producer:              app.Producer{Type: "agent", ID: "codex"},
	}))
	if err != nil {
		t.Fatal(err)
	}
}

func agentStatusByName(t *testing.T, statuses []any, name string) map[string]any {
	t.Helper()
	for _, value := range statuses {
		status, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if status["name"] == name {
			return status
		}
	}
	t.Fatalf("missing agent status %q in %#v", name, statuses)
	return nil
}

func agentModelByName(t *testing.T, models []any, name string) map[string]any {
	t.Helper()
	for _, raw := range models {
		model, ok := raw.(map[string]any)
		if ok && model["name"] == name {
			return model
		}
	}
	t.Fatalf("agent model %q not found in %#v", name, models)
	return nil
}

func stringSliceFromMap(t *testing.T, values map[string]any, key string) []string {
	t.Helper()
	raw, ok := values[key].([]any)
	if !ok {
		t.Fatalf("expected %q list in %#v", key, values)
	}
	result := make([]string, len(raw))
	for i, value := range raw {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("expected string at %q[%d], got %#v", key, i, value)
		}
		result[i] = text
	}
	return result
}

func nestedStringFromMap(t *testing.T, values map[string]any, key string) string {
	t.Helper()
	value, ok := values[key].(string)
	if !ok {
		t.Fatalf("missing string %s in %#v", key, values)
	}
	return value
}

func firstEventPayload(t *testing.T, detail map[string]any, eventType string) map[string]any {
	t.Helper()
	values, ok := detail["events"].([]any)
	if !ok {
		t.Fatalf("missing events in %#v", detail)
	}
	for _, value := range values {
		event, ok := value.(map[string]any)
		if !ok || event["EventType"] != eventType {
			continue
		}
		payload, ok := event["Payload"].(map[string]any)
		if !ok {
			t.Fatalf("missing payload in %#v", event)
		}
		return payload
	}
	t.Fatalf("missing event type %s in %#v", eventType, detail)
	return nil
}

func lastEventPayload(t *testing.T, detail map[string]any, eventType string) map[string]any {
	t.Helper()
	event := lastEvent(t, detail, eventType)
	payload, ok := event["Payload"].(map[string]any)
	if !ok {
		t.Fatalf("missing payload in %#v", event)
	}
	return payload
}

func lastEvent(t *testing.T, detail map[string]any, eventType string) map[string]any {
	t.Helper()
	values, ok := detail["events"].([]any)
	if !ok {
		t.Fatalf("missing events in %#v", detail)
	}
	for i := len(values) - 1; i >= 0; i-- {
		event, ok := values[i].(map[string]any)
		if !ok || event["EventType"] != eventType {
			continue
		}
		return event
	}
	t.Fatalf("missing event type %s in %#v", eventType, detail)
	return nil
}

func appendStaleReportPending(t *testing.T, ctx context.Context, svc *app.Service, missionID string, eventID string) {
	t.Helper()
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   eventID,
		MissionID: missionID,
		EventType: "report.draft.pending",
		Producer:  app.Producer{Type: "user", ID: "plasma-ui"},
		Payload: mustJSON(map[string]any{
			"kind":           "report_draft_pending",
			"title":          "Stale report",
			"direction_hint": "Preserve the recovered operational focus.",
			"agent_executor": "codex",
			"mcp_mode":       "auto",
			"text":           "리포트 초안 생성 중입니다.",
			"started_at":     time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano),
		}),
	}); err != nil {
		t.Fatal(err)
	}
}

func appendStaleAgentPending(t *testing.T, ctx context.Context, svc *app.Service, missionID string, userEventID string, pendingEventID string) {
	t.Helper()
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   userEventID,
		MissionID: missionID,
		EventType: "turn.user",
		Producer:  app.Producer{Type: "user", ID: "test"},
		Payload: mustJSON(map[string]any{
			"kind": "user_turn",
			"text": "stale turn",
		}),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   pendingEventID,
		MissionID: missionID,
		EventType: "turn.agent.pending",
		Producer:  app.Producer{Type: "agent", ID: "codex"},
		Payload: mustJSON(map[string]any{
			"kind":           "agent_pending",
			"agent_executor": "codex",
			"text":           "에이전트 응답을 기다리는 중입니다.",
			"user_event_id":  userEventID,
			"started_at":     time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano),
		}),
	}); err != nil {
		t.Fatal(err)
	}
}

func hasOpenPendingDetail(detail map[string]any) bool {
	values, ok := detail["events"].([]any)
	if !ok {
		return false
	}
	completed := map[string]struct{}{}
	for _, value := range values {
		event, ok := value.(map[string]any)
		if !ok || event["EventType"] != "turn.agent.response" {
			continue
		}
		payload, ok := event["Payload"].(map[string]any)
		if !ok {
			continue
		}
		if userEventID, ok := payload["user_event_id"].(string); ok && userEventID != "" {
			completed[userEventID] = struct{}{}
		}
	}
	for _, value := range values {
		event, ok := value.(map[string]any)
		if !ok || event["EventType"] != "turn.agent.pending" {
			continue
		}
		payload, ok := event["Payload"].(map[string]any)
		if !ok {
			continue
		}
		userEventID, ok := payload["user_event_id"].(string)
		if ok && userEventID != "" {
			if _, done := completed[userEventID]; !done {
				return true
			}
		}
	}
	return false
}

func hasOpenReportDraftDetail(detail map[string]any) bool {
	values, ok := detail["events"].([]any)
	if !ok {
		return false
	}
	completed := map[string]struct{}{}
	for _, value := range values {
		event, ok := value.(map[string]any)
		if !ok {
			continue
		}
		payload, ok := event["Payload"].(map[string]any)
		if !ok {
			continue
		}
		switch event["EventType"] {
		case "report.drafted":
			pendingEventID := ""
			if generation, ok := payload["generation"].(map[string]any); ok {
				pendingEventID, _ = generation["pending_event_id"].(string)
			}
			if pendingEventID != "" {
				completed[pendingEventID] = struct{}{}
			}
		case "report.artifact.created", "report.artifact.exported", "report.draft.failed", "report.design.failed", "report.humanize.failed", "report.humanize.skipped", "report.patch.failed":
			pendingEventID, _ := payload["pending_event_id"].(string)
			if pendingEventID != "" {
				completed[pendingEventID] = struct{}{}
			}
		}
	}
	for _, value := range values {
		event, ok := value.(map[string]any)
		if !ok {
			continue
		}
		switch event["EventType"] {
		case "report.draft.pending", "report.design.pending", "report.humanize.pending", "report.patch.pending":
		default:
			continue
		}
		eventID, _ := event["EventID"].(string)
		if eventID == "" {
			continue
		}
		if _, done := completed[eventID]; !done {
			return true
		}
	}
	return false
}

func waitForEventType(t *testing.T, baseURL string, missionID string, eventType string) map[string]any {
	t.Helper()
	return waitForEventTypeCount(t, baseURL, missionID, eventType, 1)
}

func waitForEventTypeCount(t *testing.T, baseURL string, missionID string, eventType string, count int) map[string]any {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var detail map[string]any
	for time.Now().Before(deadline) {
		detail = getJSON(t, baseURL+"/api/missions/"+missionID)
		if countEvents(detail, eventType) >= count {
			return detail
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d %s event(s), last detail: %#v", count, eventType, detail)
	return nil
}

func countEvents(detail map[string]any, eventType string) int {
	values, ok := detail["events"].([]any)
	if !ok {
		return 0
	}
	var count int
	for _, value := range values {
		event, ok := value.(map[string]any)
		if ok && event["EventType"] == eventType {
			count++
		}
	}
	return count
}

func hasEventForPending(detail map[string]any, eventType string, pendingEventID string) bool {
	values, ok := detail["events"].([]any)
	if !ok {
		return false
	}
	for _, value := range values {
		event, ok := value.(map[string]any)
		if !ok || event["EventType"] != eventType {
			continue
		}
		payload, ok := event["Payload"].(map[string]any)
		if !ok {
			continue
		}
		if payload["pending_event_id"] == pendingEventID {
			return true
		}
	}
	return false
}

func countLedgerEvents(events []app.LedgerEvent, eventType string) int {
	var count int
	for _, event := range events {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}

func latestEventPayload(t *testing.T, detail map[string]any, eventType string, kind string) map[string]any {
	t.Helper()
	values, ok := detail["events"].([]any)
	if !ok {
		t.Fatalf("expected events in detail: %#v", detail)
	}
	for index := len(values) - 1; index >= 0; index-- {
		event, ok := values[index].(map[string]any)
		if !ok || event["EventType"] != eventType {
			continue
		}
		payload, ok := event["Payload"].(map[string]any)
		if !ok {
			continue
		}
		if kind == "" || payload["kind"] == kind {
			return payload
		}
	}
	t.Fatalf("event %q kind %q not found in %#v", eventType, kind, values)
	return nil
}

func workflowRunStatus(t *testing.T, detail map[string]any, workflowRunID string) string {
	t.Helper()
	values, ok := detail["workflow_runs"].([]any)
	if !ok {
		t.Fatalf("expected workflow_runs in detail: %#v", detail)
	}
	for _, value := range values {
		run, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if run["workflow_run_id"] == workflowRunID {
			status, _ := run["status"].(string)
			return status
		}
	}
	t.Fatalf("workflow run %q not found in %#v", workflowRunID, values)
	return ""
}

func nestedString(t *testing.T, value map[string]any, path ...string) string {
	t.Helper()
	var current any = value
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("expected object at %s in %#v", key, current)
		}
		current = object[key]
	}
	text, ok := current.(string)
	if !ok || text == "" {
		t.Fatalf("expected string at %v in %#v", path, current)
	}
	return text
}

func nestedValue(t *testing.T, value map[string]any, path ...string) any {
	t.Helper()
	var current any = value
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("expected object at %s in %#v", key, current)
		}
		current = object[key]
	}
	return current
}

func nestedBool(t *testing.T, value map[string]any, path ...string) bool {
	t.Helper()
	current := nestedValue(t, value, path...)
	boolean, ok := current.(bool)
	if !ok {
		t.Fatalf("expected bool at %v in %#v", path, current)
	}
	return boolean
}

func nestedFloat(t *testing.T, value map[string]any, path ...string) float64 {
	t.Helper()
	current := nestedValue(t, value, path...)
	number, ok := current.(float64)
	if !ok {
		t.Fatalf("expected number at %v in %#v", path, current)
	}
	return number
}

func toJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func nestedMap(t *testing.T, value map[string]any, path ...string) map[string]any {
	t.Helper()
	var current any = value
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("expected object at %s in %#v", key, current)
		}
		current = object[key]
	}
	object, ok := current.(map[string]any)
	if !ok {
		t.Fatalf("expected object at %v in %#v", path, current)
	}
	return object
}

func fakeHumanizePatchFinalizer(t *testing.T, svc *app.Service, content string) func(context.Context, AgentRequest) {
	t.Helper()
	return func(ctx context.Context, req AgentRequest) {
		if req.ReportPatch == nil {
			return
		}
		artifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
			ArtifactID: newID("art"),
			MissionID:  req.MissionID,
			MediaType:  "text/markdown; charset=utf-8",
			Filename:   "humanized.md",
			Producer:   app.Producer{Type: "mcp_tool", ID: plasmamcp.ToolReportPatchFinalize},
			Content:    []byte(content),
		})
		if err != nil {
			t.Errorf("create H5 patch artifact: %v", err)
			return
		}
		if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
			EventID:       newID("evt"),
			MissionID:     req.MissionID,
			EventType:     "report.patch.finalized",
			Producer:      app.Producer{Type: "mcp_tool", ID: plasmamcp.ToolReportPatchFinalize},
			CorrelationID: req.ToolSessionID,
			Payload: mustJSON(map[string]any{
				"kind":                            "markdown_report_patch_finalized",
				"pending_event_id":                req.ReportPatch.PendingEventID,
				"title":                           "Humanized report",
				"artifact_id":                     artifact.ArtifactID,
				"media_type":                      artifact.MediaType,
				"base_artifact_id":                req.ReportPatch.BaseArtifactID,
				"agent_executor":                  req.ReportPatch.AgentExecutor,
				"agent_model":                     req.ReportPatch.AgentModel,
				"agent_reasoning_effort":          req.ReportPatch.AgentReasoningEffort,
				"agent_session_id":                req.ReportPatch.AgentSessionID,
				"previous_agent_session_id":       req.ReportPatch.PreviousAgentSessionID,
				"returned_agent_session_id":       req.ReportPatch.ReportSessionID,
				"report_session_id":               req.ReportPatch.ReportSessionID,
				"report_session_policy":           req.ReportPatch.ReportSessionPolicy,
				"report_session_policy_selection": req.ReportPatch.ReportSessionPolicySelection,
				"tool_session_id":                 req.ToolSessionID,
				"mcp_mode":                        req.ReportPatch.MCPMode,
				"composition_strategy":            "mcp_patch_markdown",
				"session_chain_kind":              req.ReportPatch.SessionChainKind,
			}),
		}); err != nil {
			t.Errorf("append H5 patch finalized event: %v", err)
		}
	}
}

func fakeHumanizePatchReader(t *testing.T, svc *app.Service) func(context.Context, AgentRequest) {
	t.Helper()
	return func(ctx context.Context, req AgentRequest) {
		if req.ReportPatch == nil {
			return
		}
		for _, toolName := range []string{plasmamcp.ToolReportPatchStart, plasmamcp.ToolReportPatchRead} {
			if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
				EventID:       newID("evt"),
				MissionID:     req.MissionID,
				EventType:     "mcp.tool.called",
				Producer:      app.Producer{Type: "agent_session", ID: req.ToolSessionID},
				CorrelationID: req.ToolSessionID,
				Payload: mustJSON(map[string]any{
					"agent_session_id": req.ToolSessionID,
					"tool_session_id":  req.ToolSessionID,
					"tool_name":        toolName,
					"mission_id":       req.MissionID,
					"success":          true,
					"result": map[string]any{
						"mission_id": req.MissionID,
						"success":    true,
					},
				}),
			}); err != nil {
				t.Errorf("append H5 MCP %s event: %v", toolName, err)
				return
			}
		}
	}
}

func createClaimForReportRepairTest(
	t *testing.T,
	ctx context.Context,
	svc *app.Service,
	missionID string,
	claimID string,
	proposalID string,
	eventID string,
	text string,
	evidenceIDs []string,
) {
	t.Helper()
	if _, err := svc.CreateClaimProposal(ctx, app.CreateClaimProposalRequest{
		ClaimEvent: app.AppendEventRequest{
			EventID:   eventID,
			MissionID: missionID,
			EventType: "claim.proposed",
			Producer:  app.Producer{Type: "agent_session", ID: "ses_report_repair_test"},
			Payload: mustJSON(map[string]any{
				"claim_id":    claimID,
				"proposal_id": proposalID,
			}),
		},
		Claim: app.CreateClaimRecordRequest{
			ClaimID:               claimID,
			MissionID:             missionID,
			State:                 "proposed",
			Text:                  text,
			ClaimType:             "descriptive",
			SupportingEvidenceIDs: evidenceIDs,
			Confidence:            app.Confidence{Level: "medium"},
			Approval:              app.Approval{State: "pending", Required: true},
			CreatedEventID:        eventID,
		},
		ProposalEvent: app.AppendEventRequest{
			EventID:   eventID + "_proposal",
			MissionID: missionID,
			EventType: "proposal.submitted",
			Producer:  app.Producer{Type: "agent_session", ID: "ses_report_repair_test"},
			Payload: mustJSON(map[string]any{
				"proposal_id": proposalID,
			}),
		},
		Proposal: app.CreateProposalBundleRequest{
			ProposalID:        proposalID,
			MissionID:         missionID,
			Title:             "Review claim",
			ObjectRefs:        []app.ObjectRef{{ObjectKind: app.ClaimRecordObjectKind, ObjectID: claimID}},
			RequestedDecision: "approve",
			CreatedEventID:    eventID + "_proposal",
		},
	}); err != nil {
		t.Fatal(err)
	}
}

type fakeAgentExecutor struct {
	requests       []AgentRequest
	responses      []AgentResult
	errors         []error
	onRun          func(context.Context, AgentRequest)
	rejectDeadline bool
}

type reportPlanFixtureExecutor struct {
	delegate AgentExecutor
	service  *app.Service
	sequence atomic.Int64
}

type reportPlanFixtureForkExecutor struct {
	*reportPlanFixtureExecutor
	forker    AgentSessionForker
	readiness AgentSessionForkReadiness
}

type ackAnomalyExecutor struct {
	delegate   AgentExecutor
	finalCalls int
}

func (executor *ackAnomalyExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	result, err := executor.delegate.Run(ctx, req)
	if req.LongFormFinalize != nil && err == nil {
		executor.finalCalls++
		result.Text = "ACK_NOT_EXACT"
		result.SessionID = "returned-session-other"
	}
	return result, err
}

func withReportPlanSubmissionFixture(service *app.Service, delegate AgentExecutor) AgentExecutor {
	base := &reportPlanFixtureExecutor{delegate: delegate, service: service}
	forker, canFork := delegate.(AgentSessionForker)
	readiness, canCheck := delegate.(AgentSessionForkReadiness)
	if canFork && canCheck {
		return &reportPlanFixtureForkExecutor{reportPlanFixtureExecutor: base, forker: forker, readiness: readiness}
	}
	return base
}

func (executor *reportPlanFixtureExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	result, err := executor.delegate.Run(ctx, req)
	if err != nil {
		return result, err
	}
	if req.PartAssembly != nil {
		var assembly reporting.PartAssembly
		if parseErr := json.Unmarshal([]byte(result.Text), &assembly); parseErr != nil {
			return result, parseErr
		}
		_, submitErr := executor.service.AppendEvent(ctx, reporting.BuildPartAssemblySubmittedAppendRequest(reporting.PartAssemblySubmittedEventRequest{
			EventID:  fmt.Sprintf("evt_part_assembly_fixture_%d", executor.sequence.Add(1)),
			Binding:  *req.PartAssembly,
			Assembly: assembly,
		}))
		if submitErr != nil {
			return result, submitErr
		}
		result.Text = reporting.PartAssemblySubmittedSentinel
		return result, nil
	}
	if req.LongFormFinalize != nil {
		frame, parseErr := parseFixtureSectionalFrame(result.Text)
		if parseErr != nil {
			return result, parseErr
		}
		_, finalizeErr := reporting.FinalizeLongForm(ctx, executor.service, reporting.LongFormFinalizeRequest{
			Binding: *req.LongFormFinalize, EventID: fmt.Sprintf("evt_final_fixture_%d", executor.sequence.Add(1)), OpeningMarkdown: frame.FrontMatter, ClosingMarkdown: frame.Closing,
		})
		if finalizeErr != nil {
			return result, finalizeErr
		}
		result.Text = "REPORT_FINALIZED"
		return result, nil
	}
	if req.ReportPlan == nil {
		return result, nil
	}
	var plan any
	if req.ReportPlan.ReportMode == reportModePlanned {
		var value reporting.ReportPlan
		if json.Unmarshal([]byte(result.Text), &value) != nil {
			return result, err
		}
		normalized, normalizeErr := reporting.NormalizeReportPlan(value)
		if normalizeErr != nil {
			return result, err
		}
		plan = normalized
	} else {
		var value reporting.SectionalReportPlan
		if json.Unmarshal([]byte(result.Text), &value) != nil {
			return result, err
		}
		normalized, normalizeErr := reporting.NormalizeSectionalReportPlan(value)
		if normalizeErr != nil {
			return result, err
		}
		plan = normalized
	}
	hash, encoded, hashErr := reporting.ReportPlanHash(plan)
	if hashErr != nil {
		return result, hashErr
	}
	_, submitErr := executor.service.SubmitReportPlan(ctx, app.ReportPlanSubmissionRequest{
		EventID: fmt.Sprintf("evt_plan_fixture_%d", executor.sequence.Add(1)), MissionID: req.MissionID, PendingEventID: req.ReportPlan.PendingEventID,
		ReportMode: req.ReportPlan.ReportMode, ToolSessionID: req.ToolSessionID, PreviousProviderSessionID: req.ReportPlan.PreviousProviderSessionID, AgentExecutor: req.AgentExecutor,
		AgentModel: req.ReportPlan.AgentModel, AgentReasoningEffort: req.ReportPlan.AgentReasoningEffort,
		IdempotencyKey: req.ReportPlan.IdempotencyKey, ArgumentsHash: "fixture-arguments", PlanHash: hash, Plan: encoded, Attempt: 1,
		ToolProducer: app.Producer{Type: "agent_session", ID: req.ToolSessionID},
	})
	if submitErr != nil {
		return result, submitErr
	}
	result.Text = reporting.ReportPlanSubmittedSentinel
	return result, nil
}

type fixtureSectionalFrame struct {
	FrontMatter string `json:"front_matter"`
	Closing     string `json:"closing"`
}

func parseFixtureSectionalFrame(text string) (fixtureSectionalFrame, error) {
	var frame fixtureSectionalFrame
	err := json.Unmarshal([]byte(text), &frame)
	return frame, err
}

func assertReportMCPToolSurface(t *testing.T, req AgentRequest, expected ...string) {
	t.Helper()
	if req.DisableTools {
		t.Fatalf("report request disabled tools instead of using report MCP tools: %#v", req)
	}
	if !req.ReplaceMCPTools {
		t.Fatalf("report request must replace the default MCP tool surface: %#v", req)
	}
	for _, forbidden := range []string{
		plasmamcp.ToolSourcesSearch,
		plasmamcp.ToolSourceCandidatesPropose,
		plasmamcp.ToolSourceCandidatesRead,
	} {
		if slices.Contains(req.ExtraMCPTools, forbidden) {
			t.Fatalf("report request exposed conversation/source-candidate tool %s: %#v", forbidden, req.ExtraMCPTools)
		}
	}
	for _, tool := range expected {
		if !slices.Contains(req.ExtraMCPTools, tool) {
			t.Fatalf("report request missing required MCP tool %s: %#v", tool, req.ExtraMCPTools)
		}
	}
}

func (executor *reportPlanFixtureForkExecutor) ForkSession(ctx context.Context, sessionID string) (AgentSessionForkResult, error) {
	return executor.forker.ForkSession(ctx, sessionID)
}

func (executor *reportPlanFixtureForkExecutor) CheckForkSession(ctx context.Context, sessionID string) error {
	return executor.readiness.CheckForkSession(ctx, sessionID)
}

func (executor *fakeAgentExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	executor.requests = append(executor.requests, req)
	if _, ok := ctx.Deadline(); ok && executor.rejectDeadline {
		return AgentResult{}, errors.New("unexpected agent execution deadline")
	}
	if executor.onRun != nil {
		executor.onRun(ctx, req)
	}
	var err error
	if len(executor.errors) > 0 {
		err = executor.errors[0]
		executor.errors = executor.errors[1:]
	}
	if len(executor.responses) == 0 {
		return AgentResult{Text: "fake answer", SessionID: "fake-session"}, err
	}
	response := executor.responses[0]
	executor.responses = executor.responses[1:]
	return response, err
}

type fakeForkingAgentExecutor struct {
	fakeAgentExecutor
	forkSessionID string
	forkSources   []string
	forkErr       error
}

func (executor *fakeForkingAgentExecutor) ForkSession(_ context.Context, sourceSessionID string) (AgentSessionForkResult, error) {
	executor.forkSources = append(executor.forkSources, sourceSessionID)
	if executor.forkErr != nil {
		return AgentSessionForkResult{}, executor.forkErr
	}
	sessionID := strings.TrimSpace(executor.forkSessionID)
	if sessionID == "" {
		sessionID = "forked-session"
	}
	return AgentSessionForkResult{
		SessionID:       sessionID,
		SourceSessionID: sourceSessionID,
		SourceHash:      "source-hash",
		CloneHash:       "clone-hash",
		SourceSizeBytes: 100,
		CloneSizeBytes:  100,
	}, nil
}

func (executor *fakeForkingAgentExecutor) CheckForkSession(_ context.Context, sourceSessionID string) error {
	if executor.forkErr != nil {
		return executor.forkErr
	}
	if strings.TrimSpace(sourceSessionID) == "" {
		return fmt.Errorf("source session id is required")
	}
	return nil
}

type cancelingHumanizeExecutor struct {
	cancel   context.CancelFunc
	requests []AgentRequest
}

func (executor *cancelingHumanizeExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	executor.requests = append(executor.requests, req)
	if executor.cancel != nil {
		executor.cancel()
	}
	if err := ctx.Err(); err != nil {
		return AgentResult{Log: "context canceled"}, err
	}
	return AgentResult{Log: "context canceled"}, context.Canceled
}

type sequenceBlockingAgentExecutor struct {
	mu        sync.Mutex
	release   <-chan struct{}
	requests  []AgentRequest
	responses []AgentResult
}

func (executor *sequenceBlockingAgentExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	executor.mu.Lock()
	executor.requests = append(executor.requests, req)
	executor.mu.Unlock()
	select {
	case <-executor.release:
	case <-ctx.Done():
		return AgentResult{Log: "context canceled"}, ctx.Err()
	}
	executor.mu.Lock()
	defer executor.mu.Unlock()
	if len(executor.responses) == 0 {
		return AgentResult{Text: "done", SessionID: req.PreviousSessionID, Resumed: req.PreviousSessionID != ""}, nil
	}
	response := executor.responses[0]
	executor.responses = executor.responses[1:]
	if response.SessionID == "" {
		response.SessionID = req.PreviousSessionID
	}
	response.Resumed = req.PreviousSessionID != ""
	return response, nil
}

type proposalWritingAgentExecutor struct {
	requests      []AgentRequest
	service       *app.Service
	missionID     string
	sessionID     string
	duplicateErr  error
	returnErr     error
	returnLog     string
	skipDuplicate bool
}

func (executor *proposalWritingAgentExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	executor.requests = append(executor.requests, req)
	if !executor.skipDuplicate {
		executor.duplicateErr = executor.createEvidenceProposal(ctx, "evd_existing", "prp_duplicate_existing", "evt_duplicate_existing", "evt_duplicate_existing_proposal", "Duplicate proposal should fail")
		if executor.duplicateErr == nil {
			return AgentResult{}, errors.New("duplicate evidence proposal unexpectedly succeeded")
		}
	}
	if err := executor.createEvidenceProposal(ctx, "evd_missing", "prp_missing", "evt_missing_evidence", "evt_missing_proposal", "Distinct missing evidence should be added"); err != nil {
		return AgentResult{}, err
	}
	result := AgentResult{Text: "added missing proposal", SessionID: req.PreviousSessionID, Resumed: true, Log: executor.returnLog}
	if executor.returnErr != nil {
		return result, executor.returnErr
	}
	return result, nil
}

func (executor *proposalWritingAgentExecutor) createEvidenceProposal(ctx context.Context, evidenceID, proposalID, evidenceEventID, proposalEventID, summary string) error {
	_, err := executor.service.CreateEvidenceProposal(ctx, app.CreateEvidenceProposalRequest{
		EvidenceEvent: app.AppendEventRequest{
			EventID:   evidenceEventID,
			MissionID: executor.missionID,
			EventType: "evidence.proposed",
			Producer:  app.Producer{Type: "agent_session", ID: executor.sessionID},
			Payload: mustJSON(map[string]any{
				"evidence_id": evidenceID,
				"proposal_id": proposalID,
			}),
		},
		Evidence: app.CreateEvidenceRecordRequest{
			EvidenceID:   evidenceID,
			MissionID:    executor.missionID,
			State:        "proposed",
			Summary:      summary,
			EvidenceType: "reaction",
			SnapshotRefs: []app.SnapshotRef{{
				SnapshotID: "src_existing_proposal",
				ArtifactID: "art_existing_proposal",
			}},
			Producer:       app.Producer{Type: "agent_session", ID: executor.sessionID},
			CreatedEventID: evidenceEventID,
		},
		ProposalEvent: app.AppendEventRequest{
			EventID:   proposalEventID,
			MissionID: executor.missionID,
			EventType: "proposal.submitted",
			Producer:  app.Producer{Type: "agent_session", ID: executor.sessionID},
			Payload: mustJSON(map[string]any{
				"proposal_id": proposalID,
			}),
		},
		Proposal: app.CreateProposalBundleRequest{
			ProposalID:        proposalID,
			MissionID:         executor.missionID,
			Title:             "Review extracted evidence",
			ObjectRefs:        []app.ObjectRef{{ObjectKind: app.EvidenceRecordObjectKind, ObjectID: evidenceID}},
			RequestedDecision: "approve",
			CreatedEventID:    proposalEventID,
		},
	})
	return err
}

type blockingAgentExecutor struct {
	release <-chan struct{}
	result  AgentResult
}

func (executor blockingAgentExecutor) Run(ctx context.Context, _ AgentRequest) (AgentResult, error) {
	select {
	case <-executor.release:
		return executor.result, nil
	case <-ctx.Done():
		return AgentResult{Log: "context canceled"}, ctx.Err()
	}
}

type errorAgentExecutor struct {
	result AgentResult
	err    error
}

func (executor errorAgentExecutor) Run(context.Context, AgentRequest) (AgentResult, error) {
	return executor.result, executor.err
}

func testPDFBytes(t *testing.T, lines []string) []byte {
	t.Helper()
	var stream bytes.Buffer
	stream.WriteString("BT\n/F1 12 Tf\n72 720 Td\n")
	for i, line := range lines {
		if i > 0 {
			stream.WriteString("0 -18 Td\n")
		}
		fmt.Fprintf(&stream, "(%s) Tj\n", escapeTestPDFString(line))
	}
	stream.WriteString("ET\n")
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(stream.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d /Filter /FlateDecode >>\nstream\n%s\nendstream", compressed.Len(), compressed.String()),
	}
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, out.Len())
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return out.Bytes()
}

func testPDFBytesWithInvalidContentStream(t *testing.T) []byte {
	t.Helper()
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> /Contents 4 0 R >>",
		"<< /Length 18 /Filter /FlateDecode >>\nstream\nnot flate content\nendstream",
	}
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, out.Len())
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return out.Bytes()
}

func mustMarshalTestJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal test JSON: %v", err)
	}
	return string(encoded)
}

func appendTestEvent(t *testing.T, server *Server, ctx context.Context, missionID string, eventType string, payload any, producer app.Producer) (app.LedgerEvent, error) {
	t.Helper()
	return server.service.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   newID("evt"),
		MissionID: missionID,
		EventType: eventType,
		Producer:  producer,
		Payload:   mustJSON(payload),
	})
}

func escapeTestPDFString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `(`, `\(`)
	value = strings.ReplaceAll(value, `)`, `\)`)
	return value
}
