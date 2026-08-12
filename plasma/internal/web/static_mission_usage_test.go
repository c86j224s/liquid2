package web

import (
	"strings"
	"testing"
)

func TestStaticMissionUsageCardContracts(t *testing.T) {
	index := string(mustReadStatic(t, "static/index.html"))
	usage := string(mustReadStatic(t, "static/plasma/settings_mission_usage.js"))
	tabs := string(mustReadStatic(t, "static/plasma/ui_tabs.js"))
	app := string(mustReadStatic(t, "static/app.js"))
	state := string(mustReadStatic(t, "static/plasma/state.js"))

	for _, expected := range []string{
		"missionUsageDetails", "missionUsageStatusBadge", "missionUsageRefresh",
		"settings_mission_usage.js",
	} {
		if !strings.Contains(index, expected) {
			t.Fatalf("mission usage markup is missing %q", expected)
		}
	}
	for _, expected := range []string{
		"/usage`", "usage_partial", "usage_available_count", "counter_reset_count",
		"failed_call_count", "per_call", "categories", "workflow_runs", "자율 조사 실행별",
		"resumed_call_count", "reasoning_effort", "호출당 평균", "ownsMissionSelection",
		`classList.toggle("hidden", !text)`, `usage.usage_partial ? "부분 집계" : ""`,
	} {
		if !strings.Contains(usage, expected) {
			t.Fatalf("mission usage controller is missing %q", expected)
		}
	}
	if !strings.Contains(tabs, "loadMissionUsage") || !strings.Contains(app, "loadMissionUsage") || !strings.Contains(state, "missionUsageMissionId") {
		t.Fatal("mission usage load lifecycle is not connected")
	}
	if strings.Index(index, `id="missionSettingsDetails"`) > strings.Index(index, `id="missionUsageDetails"`) ||
		strings.Index(index, `id="missionUsageDetails"`) > strings.Index(index, `id="modelDefaultsDetails"`) {
		t.Fatal("mission usage card must follow mission information and precede global defaults")
	}
}
