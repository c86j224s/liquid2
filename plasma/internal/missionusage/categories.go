package missionusage

import "strings"

var categoryOrder = []string{
	"conversation",
	"investigation",
	"report_planning",
	"report_writing",
	"report_finishing",
	"session_maintenance",
	"other",
}

var categoryLabels = map[string]string{
	"conversation":        "대화",
	"investigation":       "자율 조사",
	"report_planning":     "보고서 계획",
	"report_writing":      "보고서 작성과 편집",
	"report_finishing":    "보고서 마무리와 내보내기",
	"session_maintenance": "세션 관리",
	"other":               "기타",
}

func categoryForSurface(surface string) string {
	switch strings.TrimSpace(surface) {
	case "turn":
		return "conversation"
	case "workflow_step":
		return "investigation"
	case "report_plan", "report_plan_repair", "report_requirements", "report_part_plan":
		return "report_planning"
	case "report_section", "report_part", "report_part_edit", "report_frame", "report_markdown", "report_one_take":
		return "report_writing"
	case "report_final_write", "report_reader_edit", "report_style_edit", "report_corrective_gate",
		"report_style_semantic_validation", "report_evidence_gate", "report_design", "report_humanize_h5":
		return "report_finishing"
	case "compaction":
		return "session_maintenance"
	default:
		return "other"
	}
}
