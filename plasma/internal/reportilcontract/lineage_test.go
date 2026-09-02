package reportilcontract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func validLineagePayload(t *testing.T) map[string]any {
	t.Helper()
	entries := []any{}
	for _, item := range []struct{ id, kind, media, file, role string }{{"a1", "narrative", "application/json", "narrative.json", "intermediate"}, {"a2", "semantic_il", "application/json", "semantic-il.json", "intermediate"}, {"a3", "flow_attestation", "application/json", "flow-attestation.json", "intermediate"}, {"a4", "markdown", "text/markdown; charset=utf-8", "report.md", "final"}, {"a5", "html", "text/html; charset=utf-8", "report.html", "derivative"}, {"a6", "pdf", "application/pdf", "report.pdf", "derivative"}, {"a7", "manifest", "application/json", "manifest.json", "intermediate"}} {
		entries = append(entries, map[string]any{"artifact_id": item.id, "kind": item.kind, "media_type": item.media, "filename": item.file, "role": item.role, "sha256": strings.Repeat("a", 64), "byte_size": 1})
	}
	return map[string]any{"kind": "markdown_report_artifact", "pending_event_id": "p1", "pipeline_family": PipelineFamily, "artifact_id": "a4", "artifact_bundle": map[string]any{"pipeline_family": PipelineFamily, "markdown_artifact_id": "a4", "artifacts": entries}}
}

func validCurrentLineagePayload(t *testing.T) map[string]any {
	t.Helper()
	entries := []any{}
	for _, item := range []struct{ id, kind, media, file, role string }{
		{"a1", "narrative", "application/json", "narrative.json", "intermediate"},
		{"a2", "semantic_il", "application/json", "semantic-il.json", "intermediate"},
		{"a3", "markdown", "text/markdown; charset=utf-8", "report.md", "final"},
		{"a4", "html", "text/html; charset=utf-8", "report.html", "derivative"},
		{"a5", "pdf", "application/pdf", "report.pdf", "derivative"},
		{"a6", "manifest", "application/json", "manifest.json", "intermediate"},
	} {
		entries = append(entries, map[string]any{"artifact_id": item.id, "kind": item.kind, "media_type": item.media, "filename": item.file, "role": item.role, "sha256": strings.Repeat("a", 64), "byte_size": 1})
	}
	return map[string]any{"kind": "markdown_report_artifact", "pending_event_id": "p1", "pipeline_family": PipelineFamily, "artifact_id": "a3", "artifact_bundle": map[string]any{"pipeline_family": PipelineFamily, "markdown_artifact_id": "a3", "artifacts": entries}}
}

func validSourceSelectionSummary() map[string]any {
	return map[string]any{
		"applied":                   true,
		"accepted_sources":          23,
		"usable_sources":            18,
		"selected_sources":          12,
		"supplemental_sources":      2,
		"excluded_unusable_sources": 5,
		"excluded_budget_sources":   4,
	}
}

func TestDecodeTerminalPayloadAcceptsImageBearingLineage(t *testing.T) {
	payload := validCurrentLineagePayload(t)
	entries := payload["artifact_bundle"].(map[string]any)["artifacts"].([]any)
	entries = append(entries, map[string]any{
		"artifact_id": "a7", "kind": "image", "media_type": "image/jpeg", "filename": "report-image.jpg",
		"role": "asset", "asset_id": "asset_1", "sha256": strings.Repeat("b", 64), "byte_size": 42,
	})
	payload["artifact_bundle"].(map[string]any)["artifacts"] = entries
	decoded, err := DecodeTerminalPayload(mustJSON(t, payload))
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Bundle.Artifacts) != 7 || decoded.Bundle.Artifacts[6].AssetID != "asset_1" {
		t.Fatalf("image lineage was not preserved: %#v", decoded.Bundle.Artifacts)
	}
}

func TestDecodeTerminalPayloadRejectsUnsupportedImageMetadata(t *testing.T) {
	for name, mutate := range map[string]func(map[string]any){
		"webp":                 func(image map[string]any) { image["media_type"] = "image/webp" },
		"path filename":        func(image map[string]any) { image["filename"] = "../report-image.jpg" },
		"mismatched extension": func(image map[string]any) { image["filename"] = "report-image.png" },
	} {
		t.Run(name, func(t *testing.T) {
			payload := validCurrentLineagePayload(t)
			image := map[string]any{
				"artifact_id": "a7", "kind": "image", "media_type": "image/jpeg", "filename": "report-image.jpg",
				"role": "asset", "asset_id": "asset_1", "sha256": strings.Repeat("b", 64), "byte_size": 42,
			}
			mutate(image)
			entries := payload["artifact_bundle"].(map[string]any)["artifacts"].([]any)
			payload["artifact_bundle"].(map[string]any)["artifacts"] = append(entries, image)
			if _, err := DecodeTerminalPayload(mustJSON(t, payload)); err == nil {
				t.Fatal("unsupported image metadata was accepted")
			}
		})
	}
}

func TestDecodeTerminalPayloadAcceptsCurrentAndLegacyLineage(t *testing.T) {
	for name, payload := range map[string]map[string]any{
		"current six-artifact":  validCurrentLineagePayload(t),
		"legacy seven-artifact": validLineagePayload(t),
	} {
		t.Run(name, func(t *testing.T) {
			decoded, err := DecodeTerminalPayload(mustJSON(t, payload))
			if err != nil {
				t.Fatal(err)
			}
			if got := len(decoded.Bundle.Artifacts); got != len(payload["artifact_bundle"].(map[string]any)["artifacts"].([]any)) {
				t.Fatalf("decoded artifact count = %d", got)
			}
		})
	}
}

func TestDecodeTerminalPayloadValidatesClosedLineage(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(map[string]any)
		wantErr bool
	}{{"valid", nil, false}, {"valid source selection", func(p map[string]any) {
		p["artifact_bundle"].(map[string]any)["source_selection"] = validSourceSelectionSummary()
	}, false}, {"wrong top family", func(p map[string]any) { p["pipeline_family"] = "classic" }, true}, {"wrong bundle family", func(p map[string]any) { p["artifact_bundle"].(map[string]any)["pipeline_family"] = "classic" }, true}, {"duplicate id", func(p map[string]any) {
		p["artifact_bundle"].(map[string]any)["artifacts"].([]any)[1].(map[string]any)["artifact_id"] = "a1"
	}, true}, {"duplicate kind", func(p map[string]any) {
		p["artifact_bundle"].(map[string]any)["artifacts"].([]any)[1].(map[string]any)["kind"] = "narrative"
	}, true}, {"missing kind", func(p map[string]any) {
		delete(p["artifact_bundle"].(map[string]any)["artifacts"].([]any)[1].(map[string]any), "kind")
	}, true}, {"wrong role", func(p map[string]any) {
		p["artifact_bundle"].(map[string]any)["artifacts"].([]any)[3].(map[string]any)["role"] = "intermediate"
	}, true}, {"wrong media", func(p map[string]any) {
		p["artifact_bundle"].(map[string]any)["artifacts"].([]any)[3].(map[string]any)["media_type"] = "text/plain"
	}, true}, {"wrong filename", func(p map[string]any) {
		p["artifact_bundle"].(map[string]any)["artifacts"].([]any)[3].(map[string]any)["filename"] = "x.md"
	}, true}, {"bad hash", func(p map[string]any) {
		p["artifact_bundle"].(map[string]any)["artifacts"].([]any)[0].(map[string]any)["sha256"] = "BAD"
	}, true}, {"zero size", func(p map[string]any) {
		p["artifact_bundle"].(map[string]any)["artifacts"].([]any)[0].(map[string]any)["byte_size"] = 0
	}, true}, {"markdown mismatch", func(p map[string]any) { p["artifact_id"] = "a1" }, true}, {"trailing value", nil, true}, {"malformed trailing", nil, true}, {"unknown bundle", func(p map[string]any) { p["artifact_bundle"].(map[string]any)["unknown"] = true }, true}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := validLineagePayload(t)
			if tc.mutate != nil {
				tc.mutate(p)
			}
			raw, _ := json.Marshal(p)
			if tc.name == "trailing value" {
				raw = append(raw, []byte(` {}`)...)
			}
			if tc.name == "malformed trailing" {
				raw = append(raw, []byte(` {`)...)
			}
			_, err := DecodeTerminalPayload(raw)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestDecodeTerminalPayloadValidatesSourceSelectionSummary(t *testing.T) {
	cases := []struct {
		name    string
		summary map[string]any
		wantErr bool
	}{
		{"selection", validSourceSelectionSummary(), false},
		{"fast path", map[string]any{
			"applied": false, "accepted_sources": 5, "usable_sources": 5,
			"selected_sources": 5, "supplemental_sources": 0, "excluded_unusable_sources": 0,
			"excluded_budget_sources": 0,
		}, false},
		{"supplemented", map[string]any{
			"applied": true, "accepted_sources": 5, "usable_sources": 5,
			"selected_sources": 3, "supplemental_sources": 2, "excluded_unusable_sources": 0,
			"excluded_budget_sources": 0,
		}, false},
		{"inconsistent accepted", map[string]any{
			"applied": true, "accepted_sources": 22, "usable_sources": 18,
			"selected_sources": 12, "supplemental_sources": 0, "excluded_unusable_sources": 5,
			"excluded_budget_sources": 6,
		}, true},
		{"inconsistent applied", map[string]any{
			"applied": false, "accepted_sources": 23, "usable_sources": 18,
			"selected_sources": 12, "supplemental_sources": 0, "excluded_unusable_sources": 5,
			"excluded_budget_sources": 6,
		}, true},
		{"zero selected", map[string]any{
			"applied": true, "accepted_sources": 23, "usable_sources": 18,
			"selected_sources": 0, "supplemental_sources": 0, "excluded_unusable_sources": 5,
			"excluded_budget_sources": 18,
		}, true},
		{"unknown field", map[string]any{
			"applied": true, "accepted_sources": 23, "usable_sources": 18,
			"selected_sources": 12, "supplemental_sources": 0, "excluded_unusable_sources": 5,
			"excluded_budget_sources": 6, "source_key": "source_001",
		}, true},
		{"null", nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := validLineagePayload(t)
			payload["artifact_bundle"].(map[string]any)["source_selection"] = tc.summary
			decoded, err := DecodeTerminalPayload(mustJSON(t, payload))
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v", err)
			}
			if err == nil && decoded.Bundle.SourceSelection == nil {
				t.Fatal("source selection summary was dropped")
			}
		})
	}
}

func TestValidateActualMatchesAllArtifactMetadataAndBytes(t *testing.T) {
	decoded, err := DecodeTerminalPayload(mustJSON(t, validLineagePayload(t)))
	if err != nil {
		t.Fatal(err)
	}
	buildActual := func() []Artifact {
		actual := make([]Artifact, 0, len(decoded.Bundle.Artifacts))
		for _, entry := range decoded.Bundle.Artifacts {
			content := []byte(entry.ArtifactID)
			sumBytes := sha256.Sum256(content)
			sum := hex.EncodeToString(sumBytes[:])
			actual = append(actual, Artifact{
				ArtifactID: entry.ArtifactID,
				MediaType:  entry.MediaType,
				Filename:   entry.Filename,
				ByteSize:   int64(len(content)),
				SHA256:     sum,
				Content:    content,
			})
		}
		return actual
	}
	actual := buildActual()
	for i := range decoded.Bundle.Artifacts {
		decoded.Bundle.Artifacts[i].SHA256 = actual[i].SHA256
		decoded.Bundle.Artifacts[i].ByteSize = int(actual[i].ByteSize)
	}
	if err := ValidateActual(decoded.Bundle, actual); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		mutate func([]Artifact) []Artifact
	}{
		{"artifact ID", func(items []Artifact) []Artifact { items[0].ArtifactID = "other"; return items }},
		{"hash", func(items []Artifact) []Artifact { items[0].SHA256 = strings.Repeat("b", 64); return items }},
		{"size", func(items []Artifact) []Artifact { items[0].ByteSize++; return items }},
		{"media type", func(items []Artifact) []Artifact { items[0].MediaType = "text/plain"; return items }},
		{"filename", func(items []Artifact) []Artifact { items[0].Filename = "other.json"; return items }},
		{"content bytes", func(items []Artifact) []Artifact { items[0].Content = []byte("changed"); return items }},
		{"duplicate ID", func(items []Artifact) []Artifact { items[1].ArtifactID = items[0].ArtifactID; return items }},
		{"missing artifact", func(items []Artifact) []Artifact { return items[:len(items)-1] }},
		{"extra artifact", func(items []Artifact) []Artifact {
			content := []byte("a8")
			sumBytes := sha256.Sum256(content)
			return append(items, Artifact{
				ArtifactID: "a8",
				MediaType:  "application/json",
				Filename:   "extra.json",
				ByteSize:   int64(len(content)),
				SHA256:     hex.EncodeToString(sumBytes[:]),
				Content:    content,
			})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateActual(decoded.Bundle, tc.mutate(buildActual())); err == nil {
				t.Fatal("mismatch accepted")
			}
		})
	}
}
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
