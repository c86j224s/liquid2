package plan

import (
	"slices"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestLongFormSectionRepairPromptKeepsBoundedPlanContract(t *testing.T) {
	input := LongFormSectionRepairInput{
		Request: LongFormInput{Input: Input{
			MissionID: "mis_1", Title: "Reader Report",
			DirectionHint: "Compare the supported mechanism with the user's stated constraint.",
		}},
		Plan: LongFormOutput{Plan: reporting.SectionalReportPlan{
			Summary: "Explain the subject.",
			Parts: []reporting.ReportPlanPart{{
				Title: "Core", Purpose: "Explain the core.",
				Sections: []reporting.ReportPlanSection{{
					Title: "Unsupported", Purpose: "Explain a claim that lacked evidence.",
				}},
			}},
		}},
		RequirementMap: reporting.ReportRequirementMap{Requirements: []reporting.ReportRequirement{{
			RequirementID: "req_1", Instruction: "Include the requested comparison.",
			Owner: &reporting.ReportRequirementOwner{PartIndex: 1, SectionIndex: 1},
		}}},
		Gaps: []reporting.ReportSectionCoordinate{{PartIndex: 1, SectionIndex: 1}},
	}

	prompt := LongFormSectionRepairPrompt(input)
	for _, fragment := range []string{
		"single allowed plan-repair round",
		"Do not add, remove, merge, split, reorder, or move Sections",
		"Preserve every assigned user requirement at its current coordinate",
		"Include the requested comparison.",
		input.Request.DirectionHint,
		SectionPlanUnrepairableControlToken,
	} {
		if !strings.Contains(prompt, fragment) {
			t.Fatalf("repair prompt is missing %q:\n%s", fragment, prompt)
		}
	}
}

func TestDecodeSectionPlanRepairResponseIsStrict(t *testing.T) {
	valid := `{"replacements":[{"part_index":1,"section_index":2,"section":{"title":"Supported","purpose":"Explain supported facts."}}]}`
	response, err := decodeSectionPlanRepairResponse(valid)
	if err != nil || len(response.Replacements) != 1 {
		t.Fatalf("valid response rejected: response=%#v err=%v", response, err)
	}
	for name, value := range map[string]string{
		"empty":         "",
		"fenced":        "```json\n" + valid + "\n```",
		"unknown field": `{"replacements":[],"commentary":"no"}`,
		"trailing data": valid + ` {}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeSectionPlanRepairResponse(value); err == nil {
				t.Fatalf("invalid response was accepted: %s", value)
			}
		})
	}
}

func TestSectionPlanRepairToolsAreReadOnly(t *testing.T) {
	if slices.Contains(ResearchMCPTools(), mcptools.ToolReportPlanSubmit) {
		t.Fatal("Section plan repair must not receive the canonical plan submit tool")
	}
	if !slices.Contains(MCPTools(), mcptools.ToolReportPlanSubmit) {
		t.Fatal("canonical planning must retain the plan submit tool")
	}
}
