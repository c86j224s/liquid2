package reportrun

import (
	"strings"
	"testing"
	"time"
)

func closedILBundlePayload(t *testing.T, overrides func(*map[string]any)) map[string]any {
	t.Helper()
	kinds := []struct{ id, kind, media, filename, role string }{{"art_narrative", "narrative", "application/json", "narrative.json", "intermediate"}, {"art_document", "semantic_il", "application/json", "semantic-il.json", "intermediate"}, {"art_flow", "flow_attestation", "application/json", "flow-attestation.json", "intermediate"}, {"art_markdown", "markdown", "text/markdown; charset=utf-8", "report.md", "final"}, {"art_html", "html", "text/html; charset=utf-8", "report.html", "derivative"}, {"art_pdf", "pdf", "application/pdf", "report.pdf", "derivative"}, {"art_manifest", "manifest", "application/json", "manifest.json", "intermediate"}}
	entries := make([]any, 0, 7)
	for _, item := range kinds {
		entries = append(entries, map[string]any{"artifact_id": item.id, "kind": item.kind, "media_type": item.media, "sha256": strings.Repeat("a", 64), "byte_size": 1, "role": item.role, "filename": item.filename})
	}
	payload := map[string]any{"kind": "markdown_report_artifact", "pending_event_id": "evt_pending_il", "pipeline_family": "report_il_experimental", "artifact_id": "art_markdown", "artifact_bundle": map[string]any{"pipeline_family": "report_il_experimental", "markdown_artifact_id": "art_markdown", "artifacts": entries}}
	if overrides != nil {
		overrides(&payload)
	}
	return payload
}

func TestBuildRegistrationRegistersClosedILBundleArtifacts(t *testing.T) {
	registration, err := BuildRegistration(sequencedEvents([]Event{testEvent("evt_pending_il", "report.draft.pending", map[string]any{"title": "IL", "pipeline_family": "report_il_experimental"}), testEvent("evt_final_il", "report.artifact.created", closedILBundlePayload(t, nil))}), RegistrationNative, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if len(registration.Runs) != 1 || registration.Runs[0].FinalArtifactID != "art_markdown" || len(registration.Artifacts) != 7 {
		t.Fatalf("runs/artifacts=%#v/%#v", registration.Runs, registration.Artifacts)
	}
	expectedRoles := map[string]string{"art_narrative": ArtifactRoleIntermediate, "art_document": ArtifactRoleIntermediate, "art_flow": ArtifactRoleIntermediate, "art_markdown": ArtifactRoleFinal, "art_html": ArtifactRoleDerivative, "art_pdf": ArtifactRoleDerivative, "art_manifest": ArtifactRoleIntermediate}
	for _, item := range registration.Artifacts {
		if item.Ownership != OwnershipCreated {
			t.Fatalf("ownership=%#v", item)
		}
		if want := expectedRoles[item.ArtifactID]; want == "" || item.ArtifactRole != want {
			t.Fatalf("artifact %s role=%q want=%q", item.ArtifactID, item.ArtifactRole, want)
		}
	}
	if len(expectedRoles) != len(registration.Artifacts) {
		t.Fatalf("unexpected artifact memberships=%#v", registration.Artifacts)
	}
}

func TestBuildRegistrationRejectsMalformedILBundleCompanions(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*map[string]any)
	}{{"non-il pending", func(p *map[string]any) {}}, {"duplicate kind", func(p *map[string]any) {
		b := (*p)["artifact_bundle"].(map[string]any)
		b["artifacts"].([]any)[1].(map[string]any)["kind"] = "narrative"
	}}, {"missing kind", func(p *map[string]any) {
		b := (*p)["artifact_bundle"].(map[string]any)
		delete(b["artifacts"].([]any)[1].(map[string]any), "kind")
	}}, {"wrong role", func(p *map[string]any) {
		b := (*p)["artifact_bundle"].(map[string]any)
		b["artifacts"].([]any)[3].(map[string]any)["role"] = "intermediate"
	}}, {"bad hash", func(p *map[string]any) {
		b := (*p)["artifact_bundle"].(map[string]any)
		b["artifacts"].([]any)[0].(map[string]any)["sha256"] = "BAD"
	}}, {"markdown mismatch", func(p *map[string]any) { (*p)["artifact_id"] = "other" }}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := closedILBundlePayload(t, tc.mutate)
			pendingFamily := any("report_il_experimental")
			if tc.name == "non-il pending" {
				pendingFamily = "classic"
			}
			reg, err := BuildRegistration(sequencedEvents([]Event{testEvent("evt_pending_il", "report.draft.pending", map[string]any{"title": "IL", "pipeline_family": pendingFamily}), testEvent("evt_final_il", "report.artifact.created", payload)}), RegistrationNative, time.Now().UTC())
			if err != nil {
				t.Fatal(err)
			}
			if len(reg.Artifacts) != 1 {
				t.Fatalf("artifacts=%#v", reg.Artifacts)
			}
		})
	}
}
