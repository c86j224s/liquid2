package reporting

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestNormalizeReportPlanPreservesPlannedWhitespaceSemantics(t *testing.T) {
	plan := ReportPlan{Summary: "  summary  ", Sections: []ReportPlanSection{{Title: "  title  ", Purpose: "  purpose  "}}, CoverageNotes: []string{"  note  ", ""}}
	got, err := NormalizeReportPlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, plan) {
		t.Fatalf("planned normalization changed existing data: %#v", got)
	}
	if _, err := NormalizeReportPlan(ReportPlan{Summary: " \n\t"}); err == nil {
		t.Fatal("expected whitespace-only empty planned plan to fail")
	}
}

func TestNormalizeSectionalReportPlanKeepsTwentyFourNotesAndTruncatesTwentyFifth(t *testing.T) {
	values := make([]string, 25)
	for index := range values {
		values[index] = fmt.Sprintf(" note %02d ", index+1)
	}
	for _, size := range []int{24, 25} {
		t.Run(fmt.Sprintf("input_%d", size), func(t *testing.T) {
			plan := SectionalReportPlan{
				Parts:            []ReportPlanPart{{Title: "part", Sections: []ReportPlanSection{{Title: "section"}}}},
				CoverageNotes:    append([]string(nil), values[:size]...),
				PlannedOmissions: append([]string(nil), values[:size]...),
			}
			got, err := NormalizeSectionalReportPlan(plan)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.CoverageNotes) != 24 || len(got.PlannedOmissions) != 24 || got.CoverageNotes[23] != "note 24" || got.PlannedOmissions[23] != "note 24" {
				t.Fatalf("24-item normalization contract changed: %#v %#v", got.CoverageNotes, got.PlannedOmissions)
			}
		})
	}
}

func TestNormalizeSectionalReportPlanPreservesNormalizationAndRejectsSynthesis(t *testing.T) {
	plan := SectionalReportPlan{
		Summary: " summary ",
		Parts: []ReportPlanPart{
			{},
			{Title: " part ", Purpose: " purpose ", Sections: []ReportPlanSection{{}, {Title: " section ", Purpose: " detail ", TargetRefs: app.ReportBlockSourceRefs{QuestionIDs: []string{"qst_1"}, OptionIDs: []string{"opt_1"}}}}},
		},
		CoverageNotes: []string{"", " coverage "}, PlannedOmissions: []string{" omission ", ""},
	}
	got, err := NormalizeSectionalReportPlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary != "summary" || len(got.Parts) != 1 || got.Parts[0].Title != "part" || len(got.Parts[0].Sections) != 1 || got.Parts[0].Sections[0].Title != "section" {
		t.Fatalf("unexpected normalized plan: %#v", got)
	}
	if !reflect.DeepEqual(got.CoverageNotes, []string{"coverage"}) || !reflect.DeepEqual(got.PlannedOmissions, []string{"omission"}) {
		t.Fatalf("unexpected note normalization: %#v", got)
	}
	for name, invalid := range map[string]SectionalReportPlan{
		"missing parts":         {Summary: "summary"},
		"missing part title":    {Parts: []ReportPlanPart{{Purpose: "purpose", Sections: []ReportPlanSection{{Title: "section"}}}}},
		"missing sections":      {Parts: []ReportPlanPart{{Title: "part", Purpose: "purpose"}}},
		"missing section title": {Parts: []ReportPlanPart{{Title: "part", Sections: []ReportPlanSection{{Purpose: "purpose"}}}}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NormalizeSectionalReportPlan(invalid); err == nil {
				t.Fatal("expected synthesis-dependent plan to fail")
			}
		})
	}
}

func TestReportPlanHashIsDeterministicAndIncludesAllRefKinds(t *testing.T) {
	plan := ReportPlan{Summary: "summary", Sections: []ReportPlanSection{{Title: "section", TargetRefs: app.ReportBlockSourceRefs{ClaimIDs: []string{"clm_1"}, EvidenceIDs: []string{"evd_1"}, SnapshotIDs: []string{"src_1"}, QuestionIDs: []string{"qst_1"}, OptionIDs: []string{"opt_1"}}}}}
	first, _, err := ReportPlanHash(plan)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := ReportPlanHash(plan)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || len(first) != 64 {
		t.Fatalf("unstable hash: %q %q", first, second)
	}
	refs := ReportPlanRefs(plan)
	if len(refs) != 1 || len(refs[0].QuestionIDs) != 1 || len(refs[0].OptionIDs) != 1 {
		t.Fatalf("missing refs: %#v", refs)
	}
}
