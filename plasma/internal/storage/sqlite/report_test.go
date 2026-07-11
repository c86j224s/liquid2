package sqlite

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestCreateReportDraftPersistsImmutableVersionAndClaimLinks(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchTestService(t, store)
	createApprovedClaimFixture(t, ctx, svc)

	result := createReportDraftFixture(t, ctx, svc)
	if result.Report.ActiveVersionID != "rvn_1" || result.Version.RootBlockID == "" {
		t.Fatalf("unexpected report draft: %#v", result)
	}
	blocks, err := svc.ListReportBlocks(ctx, "rvn_1")
	if err != nil {
		t.Fatalf("ListReportBlocks returned error: %v", err)
	}
	claimBlock := findReportBlock(t, blocks, "claim")
	if !containsStringForTest(claimBlock.SourceRefs.ClaimIDs, "clm_1") {
		t.Fatalf("claim block lost claim ref: %#v", claimBlock.SourceRefs)
	}
	if !containsStringForTest(claimBlock.SourceRefs.EvidenceIDs, "evd_1") {
		t.Fatalf("claim block lost evidence ref: %#v", claimBlock.SourceRefs)
	}
	if !containsStringForTest(claimBlock.SourceRefs.SnapshotIDs, "src_1") {
		t.Fatalf("claim block lost snapshot ref: %#v", claimBlock.SourceRefs)
	}

	_, err = svc.CreateReportDraft(ctx, app.CreateReportDraftRequest{
		ReportID:        "rpt_2",
		ReportVersionID: "rvn_1",
		MissionID:       "mis_1",
		Title:           "Duplicate version",
		Scope:           app.ReportEvidenceScope{AcceptedOnly: true, ClaimIDs: []string{"clm_1"}},
		Producer:        app.Producer{Type: "agent_session", ID: "ses_1"},
		CreatedEventID:  "evt_report_drafted_dup",
	})
	if err == nil {
		t.Fatalf("expected duplicate report version failure")
	}
	if _, err := svc.GetReport(ctx, "rpt_2"); err == nil {
		t.Fatalf("duplicate-version transaction left a report behind")
	}
	assertLedgerEventMissing(t, svc, "evt_report_drafted_dup")
}

func TestReportMarkdownAndJSONExportsUseASTFixtures(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchTestService(t, store)
	createApprovedClaimFixture(t, ctx, svc)
	createReportDraftFixture(t, ctx, svc)
	promoteReportFixture(t, ctx, svc, "evt_report_promoted")

	markdown, err := svc.ExportReportVersion(ctx, app.ExportReportVersionRequest{
		ExportID:        "exp_markdown",
		ReportVersionID: "rvn_1",
		Target:          app.ReportExportTargetMarkdown,
		ArtifactID:      "art_report_markdown",
		EventID:         "evt_report_exported_markdown",
		ApprovalEventID: "evt_report_promoted",
		Producer:        app.Producer{Type: "user", ID: "ses_user"},
	})
	if err != nil {
		t.Fatalf("ExportReportVersion markdown returned error: %v", err)
	}
	markdownText := string(markdown.Artifact.Content)
	for _, expected := range []string{
		"# Test Report",
		"Research records must point at pinned evidence. [^1] [^2] [^3]",
		"Pinned source quote. [^2] [^3]",
		"## 각주",
		"[^1]: `clm_1`",
		"[^2]: `evd_1`",
		"[^3]: `src_1`",
	} {
		if !strings.Contains(markdownText, expected) {
			t.Fatalf("markdown export missing %q:\n%s", expected, markdownText)
		}
	}
	if strings.Contains(markdownText, "근거 연결") {
		t.Fatalf("unexpected markdown export:\n%s", string(markdown.Artifact.Content))
	}
	if markdown.Event.EventType != "report.exported" {
		t.Fatalf("unexpected export event: %#v", markdown.Event)
	}

	jsonAST, err := svc.ExportReportVersion(ctx, app.ExportReportVersionRequest{
		ExportID:        "exp_json",
		ReportVersionID: "rvn_1",
		Target:          app.ReportExportTargetJSONAST,
		ArtifactID:      "art_report_json",
		EventID:         "evt_report_exported_json",
		ApprovalEventID: "evt_report_promoted",
		Producer:        app.Producer{Type: "user", ID: "ses_user"},
	})
	if err != nil {
		t.Fatalf("ExportReportVersion json returned error: %v", err)
	}
	jsonText := string(jsonAST.Artifact.Content)
	if !strings.Contains(jsonText, `"schema_version": "plasma.report_ast_export.v1"`) ||
		!strings.Contains(jsonText, `"report_version_id": "rvn_1"`) {
		t.Fatalf("unexpected JSON AST export:\n%s", jsonText)
	}
}

func TestReportDraftCanUseArticleASTAndExportHTML(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchTestService(t, store)
	createApprovedClaimFixture(t, ctx, svc)
	result, err := svc.CreateReportDraft(ctx, app.CreateReportDraftRequest{
		ReportID:        "rpt_article",
		ReportVersionID: "rvn_article",
		MissionID:       "mis_1",
		Title:           "Article Report",
		FormatIntent:    "full_report",
		Scope:           app.ReportEvidenceScope{AcceptedOnly: true, ClaimIDs: []string{"clm_1"}},
		Producer:        app.Producer{Type: "agent_session", ID: "ses_report"},
		CreatedEventID:  "evt_article_report_drafted",
		Generation: map[string]any{
			"mode":             "agent_article_ast",
			"agent_session_id": "agent-session-1",
		},
		Blocks: []app.ReportBlockDraftInput{
			{
				BlockType: "title",
				Content:   []byte(`{"text":"Article Report"}`),
			},
			{
				BlockType: "paragraph",
				Content:   []byte(`{"text":"This is a polished article paragraph."}`),
				SourceRefs: app.ReportBlockSourceRefs{
					ClaimIDs:    []string{"clm_1"},
					EvidenceIDs: []string{"evd_1"},
					SnapshotIDs: []string{"src_1"},
				},
			},
			{
				BlockType: "bullet_list",
				Content:   []byte(`{"items":["First point","Second point"]}`),
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateReportDraft article AST returned error: %v", err)
	}
	if result.Version.ReportVersionID != "rvn_article" {
		t.Fatalf("unexpected article version: %#v", result.Version)
	}
	promoteReportFixtureForVersion(t, ctx, svc, "evt_article_report_promoted", "rvn_article")
	markdown, err := svc.ExportReportVersion(ctx, app.ExportReportVersionRequest{
		ExportID:        "exp_article_markdown",
		ReportVersionID: "rvn_article",
		Target:          app.ReportExportTargetMarkdown,
		ArtifactID:      "art_article_markdown",
		EventID:         "evt_article_report_exported_markdown",
		ApprovalEventID: "evt_article_report_promoted",
		Producer:        app.Producer{Type: "user", ID: "ses_user"},
	})
	if err != nil {
		t.Fatalf("ExportReportVersion markdown returned error: %v", err)
	}
	markdownText := string(markdown.Artifact.Content)
	if !strings.Contains(markdownText, "[^1] [^2] [^3]") ||
		!strings.Contains(markdownText, "[^1]: `clm_1`") ||
		!strings.Contains(markdownText, "[^2]: `evd_1`") ||
		!strings.Contains(markdownText, "[^3]: `src_1`") {
		t.Fatalf("markdown export lost AST refs:\n%s", markdownText)
	}
	html, err := svc.ExportReportVersion(ctx, app.ExportReportVersionRequest{
		ExportID:        "exp_article_html",
		ReportVersionID: "rvn_article",
		Target:          app.ReportExportTargetHTML,
		ArtifactID:      "art_article_html",
		EventID:         "evt_article_report_exported_html",
		ApprovalEventID: "evt_article_report_promoted",
		Producer:        app.Producer{Type: "user", ID: "ses_user"},
	})
	if err != nil {
		t.Fatalf("ExportReportVersion html returned error: %v", err)
	}
	htmlText := string(html.Artifact.Content)
	if html.Artifact.MediaType != "text/html; charset=utf-8" ||
		!strings.Contains(htmlText, "<!doctype html>") ||
		!strings.Contains(htmlText, "This is a polished article paragraph.") ||
		!strings.Contains(htmlText, "class=\"footnotes\"") ||
		!strings.Contains(htmlText, "clm_1") {
		t.Fatalf("unexpected HTML export: media=%q body=%s", html.Artifact.MediaType, htmlText)
	}
}

func TestReportDraftRejectsOutOfScopeArticleASTRefs(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchTestService(t, store)
	createApprovedClaimFixture(t, ctx, svc)
	_, err := svc.CreateReportDraft(ctx, app.CreateReportDraftRequest{
		ReportID:        "rpt_bad_refs",
		ReportVersionID: "rvn_bad_refs",
		MissionID:       "mis_1",
		Title:           "Bad Refs",
		FormatIntent:    "full_report",
		Scope:           app.ReportEvidenceScope{AcceptedOnly: true, ClaimIDs: []string{"clm_1"}},
		Producer:        app.Producer{Type: "agent_session", ID: "ses_report"},
		CreatedEventID:  "evt_bad_refs_report_drafted",
		Blocks: []app.ReportBlockDraftInput{{
			BlockType: "paragraph",
			Content:   []byte(`{"text":"This paragraph cites a missing claim."}`),
			SourceRefs: app.ReportBlockSourceRefs{
				ClaimIDs: []string{"clm_missing"},
			},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "out-of-scope claim") {
		t.Fatalf("expected out-of-scope ref error, got %v", err)
	}
}

func TestPromoteReportVersionRequiresApprovalEvent(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchTestService(t, store)
	createApprovedClaimFixture(t, ctx, svc)
	createReportDraftFixture(t, ctx, svc)

	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_report_promoted_autopilot",
		MissionID: "mis_1",
		EventType: "report.promoted",
		Producer:  app.Producer{Type: "autopilot", ID: "ses_auto"},
		Payload:   []byte(`{"report_version_id":"rvn_1"}`),
	}); err != nil {
		t.Fatalf("AppendEvent bad producer returned error: %v", err)
	}
	if _, err := svc.PromoteReportVersion(ctx, app.PromoteReportVersionRequest{
		ReportVersionID: "rvn_1",
		ApprovalEventID: "evt_report_promoted_autopilot",
	}); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("expected invalid promotion producer, got %v", err)
	}

	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_report_promoted_mismatch",
		MissionID: "mis_1",
		EventType: "report.promoted",
		Producer:  app.Producer{Type: "user", ID: "ses_user"},
		Payload:   []byte(`{"report_version_id":"rvn_other"}`),
	}); err != nil {
		t.Fatalf("AppendEvent mismatch returned error: %v", err)
	}
	if _, err := svc.PromoteReportVersion(ctx, app.PromoteReportVersionRequest{
		ReportVersionID: "rvn_1",
		ApprovalEventID: "evt_report_promoted_mismatch",
	}); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("expected invalid promotion payload, got %v", err)
	}

	promoted := promoteReportFixture(t, ctx, svc, "evt_report_promoted")
	if promoted.State != "export_candidate" {
		t.Fatalf("unexpected promoted version: %#v", promoted)
	}
	report, err := svc.GetReport(ctx, "rpt_1")
	if err != nil {
		t.Fatalf("GetReport returned error: %v", err)
	}
	if report.State != "export_candidate" || report.ActiveVersionID != "rvn_1" {
		t.Fatalf("unexpected promoted report: %#v", report)
	}
}

func TestCreateReportDraftRejectsProposedScope(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchTestService(t, store)
	createApprovedClaimFixture(t, ctx, svc)

	_, err := svc.CreateReportDraft(ctx, app.CreateReportDraftRequest{
		ReportID:        "rpt_proposed",
		ReportVersionID: "rvn_proposed",
		MissionID:       "mis_1",
		Title:           "Proposed records",
		Scope:           app.ReportEvidenceScope{IncludeProposed: true, ClaimIDs: []string{"clm_1"}},
		Producer:        app.Producer{Type: "agent_session", ID: "ses_1"},
		CreatedEventID:  "evt_report_drafted_proposed",
	})
	if !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("expected proposed report scope rejection, got %v", err)
	}
}

func TestCreateReportDraftRejectsUnapprovedEvidence(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchTestService(t, store)
	createResearchSource(t, ctx, svc)

	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_evidence_unapproved",
		MissionID: "mis_1",
		EventType: "evidence.proposed",
		Producer:  app.Producer{Type: "autopilot", ID: "ses_auto"},
		Payload:   []byte(`{"evidence_id":"evd_unapproved","proposal_id":"prp_evidence_unapproved"}`),
	}); err != nil {
		t.Fatalf("AppendEvent evidence returned error: %v", err)
	}
	if _, err := svc.CreateEvidenceRecord(ctx, app.CreateEvidenceRecordRequest{
		EvidenceID:   "evd_unapproved",
		MissionID:    "mis_1",
		Summary:      "Unapproved source quote.",
		EvidenceType: "quote",
		SnapshotRefs: []app.SnapshotRef{{
			SnapshotID: "src_1",
			ArtifactID: "art_1",
			Locator:    []byte(`{"locator_type":"text_quote","exact":"snapshot"}`),
		}},
		Confidence:     app.Confidence{Level: "medium"},
		Producer:       app.Producer{Type: "autopilot", ID: "ses_auto"},
		CreatedEventID: "evt_evidence_unapproved",
	}); err != nil {
		t.Fatalf("CreateEvidenceRecord returned error: %v", err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_claim_unapproved_evidence",
		MissionID: "mis_1",
		EventType: "claim.proposed",
		Producer:  app.Producer{Type: "autopilot", ID: "ses_auto"},
		Payload:   []byte(`{"claim_id":"clm_unapproved_evidence","proposal_id":"prp_claim_unapproved_evidence"}`),
	}); err != nil {
		t.Fatalf("AppendEvent claim returned error: %v", err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_claim_only_approval",
		MissionID: "mis_1",
		EventType: "proposal.approved",
		Producer:  app.Producer{Type: "user", ID: "ses_user"},
		Payload:   []byte(`{"proposal_id":"prp_claim_unapproved_evidence","approved_object_ids":["clm_unapproved_evidence"],"rejected_object_ids":[]}`),
	}); err != nil {
		t.Fatalf("AppendEvent approval returned error: %v", err)
	}
	if _, err := svc.CreateClaimRecord(ctx, app.CreateClaimRecordRequest{
		ClaimID:               "clm_unapproved_evidence",
		MissionID:             "mis_1",
		State:                 "approved",
		Text:                  "This claim points at unapproved evidence.",
		ClaimType:             "descriptive",
		SupportingEvidenceIDs: []string{"evd_unapproved"},
		Confidence:            app.Confidence{Level: "medium"},
		Approval:              app.Approval{State: "approved", ApprovalEventID: "evt_claim_only_approval"},
		CreatedEventID:        "evt_claim_unapproved_evidence",
	}); err != nil {
		t.Fatalf("CreateClaimRecord returned error: %v", err)
	}

	_, err := svc.CreateReportDraft(ctx, app.CreateReportDraftRequest{
		ReportID:        "rpt_unapproved_evidence",
		ReportVersionID: "rvn_unapproved_evidence",
		MissionID:       "mis_1",
		Title:           "Unapproved evidence",
		Scope:           app.ReportEvidenceScope{AcceptedOnly: true, ClaimIDs: []string{"clm_unapproved_evidence"}},
		Producer:        app.Producer{Type: "agent_session", ID: "ses_1"},
		CreatedEventID:  "evt_report_drafted_unapproved_evidence",
	})
	if !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("expected unapproved evidence rejection, got %v", err)
	}
}

func createApprovedClaimFixture(t *testing.T, ctx context.Context, svc *app.Service) {
	t.Helper()
	createResearchSource(t, ctx, svc)
	if _, err := svc.CreateEvidenceRecord(ctx, app.CreateEvidenceRecordRequest{
		EvidenceID:   "evd_1",
		MissionID:    "mis_1",
		Summary:      "Pinned source quote.",
		EvidenceType: "quote",
		SnapshotRefs: []app.SnapshotRef{{
			SnapshotID: "src_1",
			ArtifactID: "art_1",
			Locator:    []byte(`{"locator_type":"text_quote","exact":"snapshot"}`),
		}},
		Confidence:     app.Confidence{Level: "medium", Rationale: "Source snapshot is pinned."},
		Producer:       app.Producer{Type: "autopilot", ID: "ses_auto"},
		CreatedEventID: "evt_evidence",
	}); err != nil {
		t.Fatalf("CreateEvidenceRecord returned error: %v", err)
	}
	if _, err := svc.CreateClaimRecord(ctx, app.CreateClaimRecordRequest{
		ClaimID:               "clm_1",
		MissionID:             "mis_1",
		State:                 "approved",
		Text:                  "Research records must point at pinned evidence.",
		ClaimType:             "decision",
		SupportingEvidenceIDs: []string{"evd_1"},
		Confidence:            app.Confidence{Level: "high"},
		Approval:              app.Approval{State: "approved", ApprovalEventID: "evt_approval"},
		CreatedEventID:        "evt_claim",
	}); err != nil {
		t.Fatalf("CreateClaimRecord returned error: %v", err)
	}
}

func createReportDraftFixture(t *testing.T, ctx context.Context, svc *app.Service) app.ReportDraftResult {
	t.Helper()
	result, err := svc.CreateReportDraft(ctx, app.CreateReportDraftRequest{
		ReportID:        "rpt_1",
		ReportVersionID: "rvn_1",
		MissionID:       "mis_1",
		Title:           "Test Report",
		FormatIntent:    "briefing",
		Scope:           app.ReportEvidenceScope{AcceptedOnly: true, ClaimIDs: []string{"clm_1"}},
		Producer:        app.Producer{Type: "agent_session", ID: "ses_1"},
		CreatedEventID:  "evt_report_drafted",
	})
	if err != nil {
		t.Fatalf("CreateReportDraft returned error: %v", err)
	}
	return result
}

func promoteReportFixture(t *testing.T, ctx context.Context, svc *app.Service, eventID string) app.ReportVersion {
	t.Helper()
	return promoteReportFixtureForVersion(t, ctx, svc, eventID, "rvn_1")
}

func promoteReportFixtureForVersion(t *testing.T, ctx context.Context, svc *app.Service, eventID string, versionID string) app.ReportVersion {
	t.Helper()
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   eventID,
		MissionID: "mis_1",
		EventType: "report.promoted",
		Producer:  app.Producer{Type: "user", ID: "ses_user"},
		Payload:   []byte(`{"report_version_id":"` + versionID + `"}`),
	}); err != nil {
		t.Fatalf("AppendEvent report.promoted returned error: %v", err)
	}
	version, err := svc.PromoteReportVersion(ctx, app.PromoteReportVersionRequest{
		ReportVersionID: versionID,
		ApprovalEventID: eventID,
	})
	if err != nil {
		t.Fatalf("PromoteReportVersion returned error: %v", err)
	}
	return version
}

func findReportBlock(t *testing.T, blocks []app.ReportBlock, blockType string) app.ReportBlock {
	t.Helper()
	for _, block := range blocks {
		if block.BlockType == blockType {
			return block
		}
	}
	t.Fatalf("missing report block type %s in %#v", blockType, blocks)
	return app.ReportBlock{}
}

func containsStringForTest(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
