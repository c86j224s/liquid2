package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

const (
	c1SourceText   = "A deterministic source fixture for the C1 contract smoke.\n"
	c1TurnText     = "Summarize the source in one sentence."
	c1AgentResult  = "The source fixture is deterministic and suitable for a contract smoke."
	c1ReportTitle  = "C1 contract report"
	c1ReportText   = "# C1 contract report\n\nThe source fixture is deterministic and suitable for a contract smoke.\n"
	c1AgentSession = "c1-agent-session"
)

type c1SmokeAgent struct {
	mu       sync.Mutex
	requests []AgentRequest
}

func (agent *c1SmokeAgent) Run(_ context.Context, req AgentRequest) (AgentResult, error) {
	agent.mu.Lock()
	agent.requests = append(agent.requests, req)
	agent.mu.Unlock()

	switch {
	case req.ReportPlan != nil:
		plan, err := json.Marshal(reporting.ReportPlan{
			Summary: "Explain the deterministic source fixture.",
			Sections: []reporting.ReportPlanSection{{
				Title:   "Deterministic summary",
				Purpose: "State the fixture's stable contract result.",
			}},
		})
		if err != nil {
			return AgentResult{}, err
		}
		return AgentResult{Text: string(plan), SessionID: c1AgentSession}, nil
	case req.UserText == "generate markdown report artifact":
		return AgentResult{Text: c1ReportText, SessionID: c1AgentSession}, nil
	default:
		return AgentResult{Text: c1AgentResult, SessionID: c1AgentSession}, nil
	}
}

func TestC1VerticalContractSmoke(t *testing.T) {
	first := runC1Smoke(t, "first")
	second := runC1Smoke(t, "second")
	if !reflect.DeepEqual(first.normalized, second.normalized) {
		t.Fatalf("normalized repeat-run contract differs:\nfirst:  %s\nsecond: %s", first.normalized, second.normalized)
	}
}

type c1SmokeRun struct {
	normalized []byte
}

func runC1Smoke(t *testing.T, name string) c1SmokeRun {
	t.Helper()
	ctx := context.Background()
	database := filepath.Join(t.TempDir(), "plasma.db")
	store, err := sqlite.Open(ctx, database)
	if err != nil {
		t.Fatalf("%s: open fresh SQLite database: %v", name, err)
	}
	defer store.Close()

	service := app.NewService(store)
	agent := &c1SmokeAgent{}
	server := httptest.NewServer(NewServer(service, Options{
		AgentExecutor: withReportPlanSubmissionFixture(service, agent),
	}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{
		"title":     "C1 smoke mission",
		"objective": "Verify the Web-to-report contract.",
	})
	missionID := nestedString(t, mission, "projection", "mission_id")

	source := postJSON(t, server.URL+"/api/missions/"+missionID+"/sources/text", map[string]any{
		"title":   "C1 pasted fixture",
		"content": c1SourceText,
	})
	sourceArtifact := nestedMap(t, source, "artifact")
	sourceSnapshot := nestedMap(t, source, "snapshot")
	sourceEvent := nestedMap(t, source, "event")
	sourceArtifactID := requiredString(t, sourceArtifact, "ArtifactID")
	sourceSnapshotID := requiredString(t, sourceSnapshot, "SnapshotID")
	sourceContent := []byte(strings.TrimSpace(c1SourceText))
	sourceSHA256 := sha256HexC1(sourceContent)
	sourceStorageURI := requiredString(t, sourceArtifact, "StorageURI")
	if want := c1StorageURI(missionID, sourceSHA256); sourceStorageURI != want {
		t.Fatalf("source artifact storage URI = %q, want %q", sourceStorageURI, want)
	}
	sourceExternalID := requiredString(t, nestedMap(t, sourceSnapshot, "Connector"), "ExternalSourceID")
	if want := "manual:" + sourceSnapshotID; sourceExternalID != want {
		t.Fatalf("source external identity = %q, want %q", sourceExternalID, want)
	}
	assertArtifactMetadata(t, sourceArtifact, sourceArtifactID, "text/plain; charset=utf-8", sourceContent, "c1-pasted-fixture.txt")
	if got := requiredString(t, sourceEvent, "EventType"); got != "source.snapshotted" {
		t.Fatalf("source event type = %q", got)
	}
	sourcePayload := nestedMap(t, sourceEvent, "Payload")
	if got := requiredString(t, sourcePayload, "snapshot_id"); got != sourceSnapshotID {
		t.Fatalf("source event snapshot lineage = %q, want %q", got, sourceSnapshotID)
	}
	if got := stringSlice(t, sourcePayload, "artifact_ids"); !reflect.DeepEqual(got, []string{sourceArtifactID}) {
		t.Fatalf("source event artifact lineage = %#v, want [%q]", got, sourceArtifactID)
	}

	turn := postJSON(t, server.URL+"/api/missions/"+missionID+"/turns", map[string]any{
		"text":           c1TurnText,
		"agent_executor": "codex",
	})
	userEvent := nestedMap(t, turn, "user_event")
	userEventID := requiredString(t, userEvent, "EventID")
	detail := waitForEventType(t, server.URL, missionID, "turn.agent.response")
	responseEvent := lastEvent(t, detail, "turn.agent.response")
	responsePayload := nestedMap(t, responseEvent, "Payload")
	if got := requiredString(t, responsePayload, "user_event_id"); got != userEventID {
		t.Fatalf("agent response user lineage = %q, want %q", got, userEventID)
	}
	if got := requiredString(t, responsePayload, "text"); got != c1AgentResult {
		t.Fatalf("agent result text = %q, want %q", got, c1AgentResult)
	}
	if got := requiredString(t, responsePayload, "kind"); got != "agent_response" {
		t.Fatalf("agent result kind = %q", got)
	}

	reportStart := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title":                c1ReportTitle,
		"report_mode":          reportModePlanned,
		"agent_executor":       "codex",
		"post_report_humanize": "disabled",
	})
	pendingEvent := nestedMap(t, reportStart, "pending_event")
	if got := requiredString(t, pendingEvent, "EventType"); got != "report.draft.pending" {
		t.Fatalf("report pending event type = %q", got)
	}
	pendingEventID := requiredString(t, pendingEvent, "EventID")
	detail = waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if countEvents(detail, "report.draft.failed") != 0 {
		t.Fatalf("planned report failed: %#v", detail["events"])
	}
	if countEvents(detail, "source.snapshotted") != 1 {
		t.Fatalf("agent result or report changed the source snapshot boundary: %#v", detail["events"])
	}
	if countEvents(detail, "report.draft.pending") != 1 || countEvents(detail, "report.plan.submitted") != 1 || countEvents(detail, "report.plan.created") != 1 || countEvents(detail, "report.artifact.created") != 1 {
		t.Fatalf("planned report lifecycle counts are not exactly one: %#v", detail["events"])
	}

	assertC1EventOrder(t, detail,
		"source.snapshotted",
		"turn.user",
		"turn.agent.response",
		"report.draft.pending",
		"report.plan.submitted",
		"report.plan.created",
		"report.artifact.created",
	)
	planSubmission := lastEvent(t, detail, "report.plan.submitted")
	planEvent := lastEvent(t, detail, "report.plan.created")
	planPayload := nestedMap(t, planEvent, "Payload")
	planEventID := requiredString(t, planEvent, "EventID")
	if got := requiredString(t, planPayload, "pending_event_id"); got != pendingEventID {
		t.Fatalf("report plan pending lineage = %q, want %q", got, pendingEventID)
	}
	if got := requiredString(t, planPayload, "report_mode"); got != reportModePlanned {
		t.Fatalf("planned report mode = %q", got)
	}
	planArtifactID := requiredString(t, planPayload, "artifact_id")
	if planArtifactID == sourceArtifactID {
		t.Fatalf("planned report reserved source artifact identity %q", planArtifactID)
	}

	reportEvent := lastEvent(t, detail, "report.artifact.created")
	reportPayload := nestedMap(t, reportEvent, "Payload")
	reportArtifactID := requiredString(t, reportPayload, "artifact_id")
	if reportArtifactID != planArtifactID {
		t.Fatalf("report artifact identity = %q, want plan reservation %q", reportArtifactID, planArtifactID)
	}
	if got := requiredString(t, reportPayload, "pending_event_id"); got != pendingEventID {
		t.Fatalf("report artifact pending lineage = %q, want %q", got, pendingEventID)
	}
	if got := requiredString(t, reportPayload, "plan_event_id"); got != planEventID {
		t.Fatalf("report artifact plan lineage = %q, want %q", got, planEventID)
	}
	if got := requiredString(t, reportPayload, "media_type"); got != "text/markdown; charset=utf-8" {
		t.Fatalf("report artifact event media type = %q", got)
	}
	if reportArtifactID == sourceArtifactID {
		t.Fatalf("report artifact reused source artifact identity %q", reportArtifactID)
	}

	reportArtifactResponse := getJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+reportArtifactID)
	reportArtifact := nestedMap(t, reportArtifactResponse, "artifact")
	reportContent := strings.TrimSpace(c1ReportText)
	reportSHA256 := sha256HexC1([]byte(reportContent))
	reportStorageURI := requiredStringAny(t, reportArtifact, "storage_uri")
	if want := c1StorageURI(missionID, reportSHA256); reportStorageURI != want {
		t.Fatalf("report artifact storage URI = %q, want %q", reportStorageURI, want)
	}
	assertArtifactMetadataAPI(t, reportArtifact, reportArtifactID, "text/markdown; charset=utf-8", []byte(reportContent), "c1-contract-report.md")
	reportBody := getArtifactBody(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+reportArtifactID+"/download")
	if string(reportBody) != reportContent {
		t.Fatalf("report download = %q, want %q", reportBody, reportContent)
	}
	requests := assertC1AgentRequestContract(t, agent, missionID)

	rawContract := map[string]any{
		"source": map[string]any{
			"event":    sourceEvent,
			"artifact": sourceArtifact,
			"snapshot": sourceSnapshot,
		},
		"result": map[string]any{
			"user_event":  userEvent,
			"agent_event": responseEvent,
			"text":        c1AgentResult,
		},
		"report": map[string]any{
			"pending_event":   pendingEvent,
			"plan_submission": planSubmission,
			"plan_event":      planEvent,
			"artifact_event":  reportEvent,
			"artifact":        reportArtifact,
			"markdown":        string(reportBody),
		},
	}
	planSubmissionPayload := nestedMap(t, planSubmission, "Payload")
	replacements := map[string]string{
		missionID:        "<mission_id>",
		sourceArtifactID: "<source_artifact_id>",
		sourceSnapshotID: "<source_snapshot_id>",
		sourceExternalID: "manual:<source_snapshot_id>",
		sourceStorageURI: "plasma-artifact://<mission_id>/<source_sha256_prefix>/<source_sha256>",
		reportStorageURI: "plasma-artifact://<mission_id>/<report_sha256_prefix>/<report_sha256>",
		userEventID:      "<user_event_id>",
		pendingEventID:   "<report_pending_event_id>",
		planArtifactID:   "<report_artifact_id>",
		planEventID:      "<report_plan_event_id>",
		requiredString(t, planSubmission, "EventID"):                "<report_plan_submission_event_id>",
		requiredString(t, planSubmissionPayload, "idempotency_key"): "<report_plan_idempotency_key>",
		requiredString(t, responseEvent, "EventID"):                 "<agent_response_event_id>",
		requiredString(t, sourceEvent, "EventID"):                   "<source_event_id>",
		requiredString(t, reportEvent, "EventID"):                   "<report_artifact_event_id>",
	}
	for index, req := range requests {
		replacements[req.ToolSessionID] = []string{"<turn_tool_session_id>", "<plan_tool_session_id>", "<report_tool_session_id>"}[index]
	}
	normalized := normalizeC1Contract(rawContract, replacements)
	encoded, err := json.Marshal(normalized)
	if err != nil {
		t.Fatalf("marshal normalized contract: %v", err)
	}
	return c1SmokeRun{normalized: encoded}
}

func assertC1EventOrder(t *testing.T, detail map[string]any, eventTypes ...string) {
	t.Helper()
	events, ok := detail["events"].([]any)
	if !ok {
		t.Fatalf("missing events in %#v", detail)
	}
	next := 0
	for _, value := range events {
		event, ok := value.(map[string]any)
		if !ok || next >= len(eventTypes) || event["EventType"] != eventTypes[next] {
			continue
		}
		next++
	}
	if next != len(eventTypes) {
		t.Fatalf("event order stopped before %q in %#v", eventTypes[next], events)
	}
}

func assertC1AgentRequestContract(t *testing.T, agent *c1SmokeAgent, missionID string) []AgentRequest {
	t.Helper()
	agent.mu.Lock()
	requests := append([]AgentRequest(nil), agent.requests...)
	agent.mu.Unlock()
	if len(requests) != 3 {
		t.Fatalf("agent request count = %d, want answer, plan, and report", len(requests))
	}
	for index, req := range requests {
		if req.MissionID != missionID {
			t.Fatalf("agent request %d mission = %q, want %q", index, req.MissionID, missionID)
		}
		if strings.TrimSpace(req.ToolSessionID) == "" {
			t.Fatalf("agent request %d has no tool session binding", index)
		}
	}
	for _, index := range []int{1, 2} {
		req := requests[index]
		if !req.ReplaceMCPTools || !slices.Contains(req.ExtraMCPTools, mcptools.ToolResearchOutline) ||
			!slices.Contains(req.ExtraMCPTools, mcptools.ToolSourcesList) || !slices.Contains(req.ExtraMCPTools, mcptools.ToolSourcesRead) {
			t.Fatalf("report request %d lacks mission source-read tools: %#v", index, req.ExtraMCPTools)
		}
		if !strings.Contains(req.Prompt, "mission_id "+missionID) {
			t.Fatalf("report request %d prompt lacks mission binding", index)
		}
	}
	return requests
}

func assertArtifactMetadata(t *testing.T, value map[string]any, artifactID, mediaType string, content []byte, filename string) {
	t.Helper()
	if got := requiredString(t, value, "ArtifactID"); got != artifactID {
		t.Fatalf("artifact id = %q, want %q", got, artifactID)
	}
	if got := requiredString(t, value, "MediaType"); got != mediaType {
		t.Fatalf("artifact %q media type = %q, want %q", artifactID, got, mediaType)
	}
	if got := requiredString(t, value, "SHA256"); got != sha256HexC1(content) {
		t.Fatalf("artifact %q SHA256 = %q, want %q", artifactID, got, sha256HexC1(content))
	}
	if got := requiredString(t, value, "Filename"); got != filename {
		t.Fatalf("artifact %q filename = %q, want %q", artifactID, got, filename)
	}
}

func assertArtifactMetadataAPI(t *testing.T, value map[string]any, artifactID, mediaType string, content []byte, filename string) {
	t.Helper()
	if got := requiredStringAny(t, value, "artifact_id"); got != artifactID {
		t.Fatalf("artifact id = %q, want %q", got, artifactID)
	}
	if got := requiredStringAny(t, value, "media_type"); got != mediaType {
		t.Fatalf("artifact %q media type = %q, want %q", artifactID, got, mediaType)
	}
	if got := requiredStringAny(t, value, "sha256"); got != sha256HexC1(content) {
		t.Fatalf("artifact %q SHA256 = %q, want %q", artifactID, got, sha256HexC1(content))
	}
	if got := requiredStringAny(t, value, "filename"); got != filename {
		t.Fatalf("artifact %q filename = %q, want %q", artifactID, got, filename)
	}
}

func requiredStringAny(t *testing.T, value map[string]any, key string) string {
	t.Helper()
	got, ok := value[key].(string)
	if !ok || strings.TrimSpace(got) == "" {
		t.Fatalf("missing non-empty %q in %#v", key, value)
	}
	return got
}

func getArtifactBody(t *testing.T, url string) []byte {

	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s returned %d: %s", url, resp.StatusCode, body)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func requiredString(t *testing.T, value map[string]any, key string) string {
	t.Helper()
	got, ok := value[key].(string)
	if !ok || strings.TrimSpace(got) == "" {
		t.Fatalf("missing non-empty %q in %#v", key, value)
	}
	return got
}

func stringSlice(t *testing.T, value map[string]any, key string) []string {
	t.Helper()
	raw, ok := value[key].([]any)
	if !ok {
		t.Fatalf("missing %q string list in %#v", key, value)
	}
	result := make([]string, len(raw))
	for i, item := range raw {
		result[i], ok = item.(string)
		if !ok {
			t.Fatalf("%q[%d] is not a string: %#v", key, i, item)
		}
	}
	return result
}

func sha256HexC1(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func c1StorageURI(missionID, sha256 string) string {
	return "plasma-artifact://" + missionID + "/" + sha256[:2] + "/" + sha256
}

func normalizeC1Contract(value any, replacements map[string]string) any {
	switch value := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(value))
		for key, item := range value {
			if c1DynamicContractKey(key) {
				continue
			}
			result[key] = normalizeC1Contract(item, replacements)
		}
		return result
	case []any:
		result := make([]any, len(value))
		for i, item := range value {
			result[i] = normalizeC1Contract(item, replacements)
		}
		return result
	case string:
		if replacement, ok := replacements[value]; ok {
			return replacement
		}
		return value
	default:
		return value
	}
}

func c1DynamicContractKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "createdat", "capturedat", "externalupdatedat", "created_at", "captured_at", "external_updated_at", "started_at", "duration_ms", "agent_usage_duration_ms", "sequence":
		return true
	default:
		return false
	}
}

var _ AgentExecutor = (*c1SmokeAgent)(nil)
