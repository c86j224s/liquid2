package web

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type finalWriterV2TraceRow struct {
	stage          string
	label          string
	providerStage  string
	tools          []string
	requiredEvents []string
	forkFrom       string
	sourceArtifact string
	canonicalizes  bool
}

var finalWriterV2CitationTokenPattern = regexp.MustCompile(`\[(?:WANG|RAFT|T)-[0-9]+\]`)

func writeFinalWriterV2BlindPacks(t *testing.T, archive string, runs []finalWriterV2ExperimentRun) {
	t.Helper()
	byPair := map[string]map[string]finalWriterV2ExperimentRun{}
	for _, run := range runs {
		if byPair[run.PairID] == nil {
			byPair[run.PairID] = map[string]finalWriterV2ExperimentRun{}
		}
		byPair[run.PairID][run.Arm] = run
	}
	mapping := loadOrCreateFinalWriterV2BlindMapping(t, archive)
	for _, pair := range finalWriterV2ExperimentPairs() {
		lines := []string{
			"# Blind Pair: " + pair.TopicTitle,
			"",
			"- Pair ID: `" + pair.PairID + "`",
			"- Rigor: `" + pair.Rigor + "`",
			"- Read both Markdown reports directly before scoring.",
			"",
		}
		for _, label := range []string{"report_1", "report_2"} {
			arm := mapping[pair.PairID][label]
			body, err := os.ReadFile(byPair[pair.PairID][arm].ReportPath)
			if err != nil {
				t.Fatal(err)
			}
			lines = append(lines, "## "+strings.Title(strings.ReplaceAll(label, "_", " ")), "", strings.TrimSpace(string(body)), "")
		}
		pack := strings.Join(lines, "\n")
		if finalWriterV2LeaksArmIdentity(pack) {
			t.Fatalf("blind pack leaks arm identity for %s", pair.PairID)
		}
		path := filepath.Join(archive, "reading-packs", finalWriterV2ExperimentRunNamespace, "blind", pair.PairID+".md")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(strings.TrimSpace(pack)+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	index := []string{"# Blind Reading Pack Index", ""}
	for _, pair := range finalWriterV2ExperimentPairs() {
		index = append(index, fmt.Sprintf("- [%s](%s.md)", pair.PairID, pair.PairID))
	}
	if err := os.WriteFile(filepath.Join(archive, "reading-packs", finalWriterV2ExperimentRunNamespace, "blind", "README.md"), []byte(strings.Join(index, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := finalWriterV2BlindSeal(archive); err != nil {
		t.Fatal(err)
	}
}

func loadOrCreateFinalWriterV2BlindMapping(t *testing.T, archive string) map[string]map[string]string {
	t.Helper()
	path := filepath.Join(archive, "control", "blind_mapping."+finalWriterV2ExperimentRunNamespace+".json")
	content, err := os.ReadFile(path)
	if err == nil {
		var mapping map[string]map[string]string
		if err := json.Unmarshal(content, &mapping); err != nil {
			t.Fatalf("decode existing blind mapping: %v", err)
		}
		if err := validateFinalWriterV2BlindMapping(mapping); err != nil {
			t.Fatalf("validate existing blind mapping: %v", err)
		}
		return mapping
	}
	if !os.IsNotExist(err) {
		t.Fatal(err)
	}

	mapping := map[string]map[string]string{}
	for _, pair := range finalWriterV2ExperimentPairs() {
		arms := []string{"A", "B"}
		if finalWriterV2RandomBit(t) == 1 {
			arms[0], arms[1] = arms[1], arms[0]
		}
		mapping[pair.PairID] = map[string]string{"report_1": arms[0], "report_2": arms[1]}
	}
	if err := validateFinalWriterV2BlindMapping(mapping); err != nil {
		t.Fatal(err)
	}
	writeFinalWriterV2JSON(t, path, mapping)
	return mapping
}

func validateFinalWriterV2BlindMapping(mapping map[string]map[string]string) error {
	expected := finalWriterV2ExperimentPairs()
	if len(mapping) != len(expected) {
		return fmt.Errorf("blind mapping pair count=%d, want %d", len(mapping), len(expected))
	}
	for _, pair := range expected {
		labels, ok := mapping[pair.PairID]
		if !ok || len(labels) != 2 || labels["report_1"] == labels["report_2"] ||
			!slices.Contains([]string{"A", "B"}, labels["report_1"]) ||
			!slices.Contains([]string{"A", "B"}, labels["report_2"]) {
			return fmt.Errorf("blind mapping is invalid for %s", pair.PairID)
		}
	}
	return nil
}

func finalWriterV2BlindSeal(archive string) (string, map[string]string, error) {
	mappingPath := filepath.Join(archive, "control", "blind_mapping."+finalWriterV2ExperimentRunNamespace+".json")
	if err := finalWriterV2PathInsideArchive(archive, mappingPath); err != nil {
		return "", nil, err
	}
	mappingSHA := finalWriterV2SHA256FileNoErr(mappingPath)
	if mappingSHA == "" {
		return "", nil, fmt.Errorf("blind mapping digest is missing")
	}
	packDir := filepath.Join(archive, "reading-packs", finalWriterV2ExperimentRunNamespace, "blind")
	packSHA := map[string]string{}
	for _, pair := range finalWriterV2ExperimentPairs() {
		path := filepath.Join(packDir, pair.PairID+".md")
		if err := finalWriterV2PathInsideArchive(archive, path); err != nil {
			return "", nil, err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return "", nil, err
		}
		if finalWriterV2LeaksArmIdentity(string(content)) {
			return "", nil, fmt.Errorf("blind pack leaks arm identity for %s", pair.PairID)
		}
		packSHA[pair.PairID] = sha256Hex(content)
	}
	return mappingSHA, packSHA, nil
}

func finalWriterV2StageTrace(ctx context.Context, svc *app.Service, req longFormReaderStyleGatePipelineRequest, arm string, requests []AgentRequest) ([]map[string]any, []string) {
	styleEnabled := req.postReportHumanize == reporting.FinalEditHumanizeEnabled
	pipeline := finalWriterV2PipelineForArm(arm)
	events, err := svc.ListEvents(ctx, req.missionID)
	if err != nil {
		return nil, []string{err.Error()}
	}
	stageEvents := map[string][]string{}
	for _, event := range events {
		switch event.EventType {
		case reporting.FinalEditAssemblyCreatedEventType:
			stageEvents["final_assembly"] = append(stageEvents["final_assembly"], event.EventType)
		case reporting.FinalEditWriterStartedEventType, reporting.FinalEditWriterSubmittedEventType:
			stageEvents["final_write"] = append(stageEvents["final_write"], event.EventType)
		case reporting.FinalEditReaderStartedEventType, reporting.FinalEditReaderSubmittedEventType:
			stageEvents["reader_edit"] = append(stageEvents["reader_edit"], event.EventType)
		case reporting.FinalEditStyleStartedEventType, reporting.FinalEditStyleSubmittedEventType:
			stageEvents["style_edit"] = append(stageEvents["style_edit"], event.EventType)
		case reporting.FinalEditGateStartedEventType, reporting.FinalEditGateSubmittedEventType:
			stageEvents["corrective_gate"] = append(stageEvents["corrective_gate"], event.EventType)
		}
	}
	stageRequests := map[string]AgentRequest{}
	for _, request := range requests {
		if request.FinalEditStage != nil {
			stageRequests[request.FinalEditStage.Stage] = request
		}
	}
	stages := finalWriterV2ExpectedTraceRows(arm, styleEnabled)
	trace := make([]map[string]any, 0, len(stages))
	errors := []string{}
	for _, stage := range stages {
		row := map[string]any{
			"stage": stage.stage, "label": stage.label, "pipeline": pipeline, "tools": finalWriterV2StringSlice(stage.tools),
			"events": finalWriterV2StringSlice(stageEvents[stage.stage]), "fork_from": stage.forkFrom,
			"source_artifact": stage.sourceArtifact, "canonicalizes": stage.canonicalizes,
		}
		if stage.providerStage != "" {
			request, ok := stageRequests[stage.providerStage]
			if !ok || request.FinalEditStage == nil {
				errors = append(errors, "missing provider request for "+stage.stage)
			} else {
				binding := request.FinalEditStage
				row["provider_stage"] = binding.Stage
				row["source_artifact_id"] = binding.SourceArtifactID
				row["provider_session_id"] = binding.ProviderSessionID
				row["previous_provider_session_id"] = binding.PreviousProviderSessionID
				row["fork_source_agent_session_id"] = binding.ForkSourceAgentSessionID
				if !slices.Equal(request.ExtraMCPTools, stage.tools) || !request.ReplaceMCPTools {
					errors = append(errors, "wrong tools for "+stage.stage)
				}
				if err := finalWriterV2ValidateStageAncestry(req, binding, stage.forkFrom); err != nil {
					errors = append(errors, stage.stage+": "+err.Error())
				}
			}
		}
		for _, eventType := range stage.requiredEvents {
			if !slices.Contains(stageEvents[stage.stage], eventType) {
				errors = append(errors, "missing event "+eventType+" for "+stage.stage)
			}
		}
		trace = append(trace, row)
	}
	return trace, errors
}

func finalWriterV2ValidateStoredStageTrace(trace []map[string]any, arm string, styleEnabled bool) []string {
	expected := finalWriterV2ExpectedTraceRows(arm, styleEnabled)
	if len(trace) != len(expected) {
		return []string{fmt.Sprintf("stage trace row count=%d, want %d", len(trace), len(expected))}
	}
	pipeline := finalWriterV2PipelineForArm(arm)
	errors := []string{}
	for index, want := range expected {
		row := trace[index]
		if got, _ := row["pipeline"].(string); got != pipeline {
			errors = append(errors, fmt.Sprintf("%s pipeline=%q, want %q", want.stage, got, pipeline))
		}
		if got, _ := row["stage"].(string); got != want.stage {
			errors = append(errors, fmt.Sprintf("stage %d=%q, want %q", index, got, want.stage))
		}
		if got, _ := row["label"].(string); got != want.label {
			errors = append(errors, fmt.Sprintf("%s label=%q, want %q", want.stage, got, want.label))
		}
		if got, _ := row["fork_from"].(string); got != want.forkFrom {
			errors = append(errors, fmt.Sprintf("%s fork_from=%q, want %q", want.stage, got, want.forkFrom))
		}
		if got, _ := row["source_artifact"].(string); got != want.sourceArtifact {
			errors = append(errors, fmt.Sprintf("%s source_artifact=%q, want %q", want.stage, got, want.sourceArtifact))
		}
		if got, _ := row["canonicalizes"].(bool); got != want.canonicalizes {
			errors = append(errors, fmt.Sprintf("%s canonicalizes=%v, want %v", want.stage, got, want.canonicalizes))
		}
		if !slices.Equal(finalWriterV2StringsFromAny(row["tools"]), want.tools) {
			errors = append(errors, "wrong tools for "+want.stage)
		}
		events := finalWriterV2StringsFromAny(row["events"])
		for _, eventType := range want.requiredEvents {
			if !slices.Contains(events, eventType) {
				errors = append(errors, "missing event "+eventType+" for "+want.stage)
			}
		}
		if want.providerStage == "" {
			continue
		}
		if got, _ := row["provider_stage"].(string); got != want.providerStage {
			errors = append(errors, fmt.Sprintf("%s provider_stage=%q, want %q", want.stage, got, want.providerStage))
		}
		for _, key := range []string{"source_artifact_id", "provider_session_id", "fork_source_agent_session_id"} {
			if got, _ := row[key].(string); strings.TrimSpace(got) == "" {
				errors = append(errors, want.stage+" missing "+key)
			}
		}
	}
	return errors
}

func TestFinalWriterV2ExperimentAdapterRejectsStoredStageTraceWithMissingOrMismatchedPipeline(t *testing.T) {
	tests := []struct {
		name   string
		mutate func([]map[string]any)
	}{
		{
			name: "missing",
			mutate: func(trace []map[string]any) {
				delete(trace[0], "pipeline")
			},
		},
		{
			name: "mismatched",
			mutate: func(trace []map[string]any) {
				trace[0]["pipeline"] = finalWriterV2PipelineForArm("B")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			trace := finalWriterV2ValidStoredStageTraceForTest("A", false)
			tc.mutate(trace)
			if errors := finalWriterV2ValidateStoredStageTrace(trace, "A", false); len(errors) == 0 {
				t.Fatal("accepted stored stage trace with invalid pipeline")
			}
		})
	}
}

func finalWriterV2ValidStoredStageTraceForTest(arm string, styleEnabled bool) []map[string]any {
	pipeline := finalWriterV2PipelineForArm(arm)
	trace := []map[string]any{}
	for _, row := range finalWriterV2ExpectedTraceRows(arm, styleEnabled) {
		item := map[string]any{
			"stage": row.stage, "label": row.label, "pipeline": pipeline, "tools": finalWriterV2StringSlice(row.tools),
			"events": finalWriterV2StringSlice(row.requiredEvents), "fork_from": row.forkFrom,
			"source_artifact": row.sourceArtifact, "canonicalizes": row.canonicalizes,
		}
		if row.providerStage != "" {
			item["provider_stage"] = row.providerStage
			item["source_artifact_id"] = "artifact_" + row.stage
			item["provider_session_id"] = "session_" + row.stage
			item["fork_source_agent_session_id"] = "fork_" + row.stage
		}
		trace = append(trace, item)
	}
	return trace
}

func finalWriterV2ExpectedTraceRows(arm string, styleEnabled bool) []finalWriterV2TraceRow {
	var rows []finalWriterV2TraceRow
	if arm == "B" {
		rows = append(rows,
			finalWriterV2TraceRow{stage: "final_assembly", label: "최종 조립", requiredEvents: []string{reporting.FinalEditAssemblyCreatedEventType}, forkFrom: "none", sourceArtifact: "reviewed_part_artifacts"},
			finalWriterV2TraceRow{stage: "final_write", label: "최종 작성", providerStage: reporting.FinalEditStageWriter, tools: reportFinalEditWriterMCPTools(), requiredEvents: []string{reporting.FinalEditWriterStartedEventType, reporting.FinalEditWriterSubmittedEventType}, forkFrom: "report_plan_session", sourceArtifact: "final_assembly"},
		)
	} else {
		rows = append(rows, finalWriterV2TraceRow{stage: "reader_source_assembly", label: "reader source assembly", forkFrom: "none", sourceArtifact: "reviewed_part_artifacts"})
	}
	readerSource := "reader_source_assembly"
	if arm == "B" {
		readerSource = "final_write"
	}
	rows = append(rows, finalWriterV2TraceRow{stage: "reader_edit", label: "reader editor", providerStage: reporting.FinalEditStageReader, tools: reportFinalEditReaderMCPTools(), requiredEvents: []string{reporting.FinalEditReaderStartedEventType, reporting.FinalEditReaderSubmittedEventType}, forkFrom: "report_plan_session", sourceArtifact: readerSource})
	if styleEnabled {
		rows = append(rows, finalWriterV2TraceRow{stage: "style_edit", label: "optional style editor", providerStage: reporting.FinalEditStageStyle, tools: reportFinalEditStyleMCPTools(), requiredEvents: []string{reporting.FinalEditStyleStartedEventType, reporting.FinalEditStyleSubmittedEventType}, forkFrom: "reader_provider_session", sourceArtifact: "reader_edit"})
	}
	gateSource := "reader_edit"
	if styleEnabled {
		gateSource = "style_edit"
	}
	rows = append(rows, finalWriterV2TraceRow{stage: "corrective_gate", label: "corrective gate", providerStage: reporting.FinalEditStageGate, tools: reportFinalEditGateMCPTools(), requiredEvents: []string{reporting.FinalEditGateStartedEventType, reporting.FinalEditGateSubmittedEventType}, forkFrom: "report_plan_session", sourceArtifact: gateSource, canonicalizes: true})
	return rows
}

func finalWriterV2ValidateStageAncestry(req longFormReaderStyleGatePipelineRequest, binding *reporting.FinalEditStageBinding, forkFrom string) error {
	switch forkFrom {
	case "none":
		return nil
	case "report_plan_session":
		if binding.ForkSourceAgentSessionID != req.reportPlanSessionID {
			return fmt.Errorf("fork source %q, want report plan session %q", binding.ForkSourceAgentSessionID, req.reportPlanSessionID)
		}
	case "reader_provider_session":
		if binding.Stage != reporting.FinalEditStageStyle {
			return fmt.Errorf("reader child fork assigned to %s", binding.Stage)
		}
		if strings.TrimSpace(binding.ForkSourceAgentSessionID) == "" || binding.ForkSourceAgentSessionID == req.reportPlanSessionID {
			return fmt.Errorf("style fork source did not use reader provider session")
		}
	default:
		return fmt.Errorf("unknown fork contract %q", forkFrom)
	}
	return nil
}

func finalWriterV2ProtectedCitationTokens(markdown string) []string {
	seen := map[string]bool{}
	tokens := []string{}
	for _, token := range finalWriterV2CitationTokenPattern.FindAllString(markdown, -1) {
		if seen[token] {
			continue
		}
		seen[token] = true
		tokens = append(tokens, token)
	}
	return tokens
}

func finalWriterV2InformationLossCount(pair finalWriterV2ExperimentPair, markdown string) int {
	switch pair.TopicID {
	case "wang-anshi-northern-song":
		return finalWriterV2MissingConceptGroups(markdown,
			[]string{"new policies", "신법"}, []string{"shenzong", "신종"}, []string{"sima guang", "사마광"},
			[]string{"green sprouts", "청묘", "청묘법"}, []string{"labor-service", "노역", "부역", "역역"},
			[]string{"factional", "파당", "파벌", "당파", "정파"},
		)
	case "go-raft-implementation-roadmap":
		return finalWriterV2MissingConceptGroups(markdown,
			[]string{"term", "임기"}, []string{"vote", "투표"}, []string{"log", "로그"},
			[]string{"randomized", "무작위", "랜덤"},
			[]string{"appendentries"}, []string{"snapshot", "스냅샷", "스냅숏"}, []string{"membership", "구성", "멤버십"},
			[]string{"simulator", "시뮬레이터", "모의"}, []string{"storage", "저장", "스토리지"},
			[]string{"transport", "전송", "트랜스포트"}, []string{"observability", "관측", "가시성"},
		)
	default:
		return 1
	}
}

func finalWriterV2RequirementLossCount(pair finalWriterV2ExperimentPair, markdown string) int {
	switch pair.TopicID {
	case "wang-anshi-northern-song":
		return finalWriterV2MissingConceptGroups(markdown,
			[]string{"finance", "재정"}, []string{"frontier defense", "국방", "방위", "방어", "변방", "변경", "군사"},
			[]string{"official recruitment", "관료 충원", "관료 선발", "관료 등용", "관료 형성", "인재 선발", "인재 운용", "관리 선발", "관리 교육", "관리 체계"},
			[]string{"balanced", "균형"}, []string{"implementation risk", "implementation burden", "risk", "risky", "burden", "시행 위험", "실행 위험", "집행 위험", "지방 부담", "부담", "한계"},
		)
	case "go-raft-implementation-roadmap":
		return finalWriterV2MissingConceptGroups(markdown,
			[]string{"safety", "안전", "안전성"}, []string{"testable", "테스트", "검증"},
			[]string{"milestone", "마일스톤", "단계", "이정표"}, []string{"ordering", "순서", "순차", "우선순위"}, []string{"risk", "위험", "리스크"},
		)
	default:
		return 1
	}
}

func finalWriterV2MissingConceptGroups(markdown string, groups ...[]string) int {
	folded := strings.ToLower(markdown)
	missing := 0
	for _, group := range groups {
		found := false
		for _, token := range group {
			if strings.Contains(folded, strings.ToLower(token)) {
				found = true
				break
			}
		}
		if !found {
			missing++
		}
	}
	return missing
}

func finalWriterV2LeaksArmIdentity(markdown string) bool {
	folded := strings.ToLower(markdown)
	for _, term := range []string{
		reporting.FinalEditPipelineReaderStyleGateV1,
		reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2,
		"final_write", "final writer", "current v1", "writer v2", "arm a", "arm b",
	} {
		if strings.Contains(folded, strings.ToLower(term)) {
			return true
		}
	}
	return false
}

func finalWriterV2TraceStyleEnabled(trace []map[string]any) bool {
	for _, row := range trace {
		if row["stage"] == "style_edit" {
			return true
		}
	}
	return false
}

func finalWriterV2PassFail(ok bool) string {
	if ok {
		return "pass"
	}
	return "fail"
}

func finalWriterV2StringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func finalWriterV2StringsFromAny(value any) []string {
	switch values := value.(type) {
	case []string:
		return append([]string(nil), values...)
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			text, ok := value.(string)
			if !ok {
				return nil
			}
			out = append(out, text)
		}
		return out
	default:
		return nil
	}
}

func finalWriterV2IntFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}

func ensureFinalWriterV2ArchiveOutsideRepo(archive string) error {
	archive, err := filepath.Abs(archive)
	if err != nil {
		return err
	}
	repo, err := filepath.Abs(filepath.Clean(filepath.Join("..", "..")))
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(repo, archive)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("experiment archive must stay outside repository: %s", archive)
	}
	return nil
}

func finalWriterV2PathInsideArchive(archive string, path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("archive path is empty")
	}
	archiveAbs, err := filepath.Abs(filepath.Clean(archive))
	if err != nil {
		return err
	}
	pathAbs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(archiveAbs, pathAbs)
	if err != nil {
		return err
	}
	if rel == "." {
		return fmt.Errorf("path points to archive root: %s", path)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("path escapes archive: %s", path)
	}
	return nil
}

func writeFinalWriterV2JSON(t *testing.T, path string, value any) {
	t.Helper()
	if err := writeFinalWriterV2JSONFilePath(path, value); err != nil {
		t.Fatal(err)
	}
}

func writeFinalWriterV2JSONFile(root string, rel string, value any) error {
	return writeFinalWriterV2JSONFilePath(filepath.Join(root, rel), value)
}

func writeFinalWriterV2RawFile(root string, rel string, content string) error {
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

func finalWriterV2MustJSON(value any) json.RawMessage {
	encoded, _ := json.Marshal(value)
	return encoded
}

func finalWriterV2SHA256FileNoErr(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return sha256Hex(content)
}

func finalWriterV2RandomBit(t *testing.T) byte {
	t.Helper()
	var buf [1]byte
	if _, err := rand.Read(buf[:]); err != nil {
		t.Fatal(err)
	}
	return buf[0] & 1
}
