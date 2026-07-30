package reporting

import "testing"

func TestNormalizeReportRequirementMapKeepsFixedOutlineOwnership(t *testing.T) {
	plan := SectionalReportPlan{Parts: []ReportPlanPart{{Title: "Part", Sections: []ReportPlanSection{{Title: "One"}, {Title: "Two"}}}}}
	value := ReportRequirementMap{
		ReviewedEventIDs: []string{" evt_user ", "evt_pending"},
		Requirements: []ReportRequirement{
			{RequirementID: " req_table ", Instruction: " include a comparison table ", SourceEventIDs: []string{"evt_pending"}, Owner: &ReportRequirementOwner{PartIndex: 1, SectionIndex: 2}},
			{RequirementID: "req_missing", Instruction: "cover an unsupported appendix", SourceEventIDs: []string{"evt_user"}, UnmappedReason: "no matching section in the fixed outline"},
		},
	}
	normalized, err := NormalizeReportRequirementMap(value, plan)
	if err != nil {
		t.Fatal(err)
	}
	if normalized.Requirements[0].Instruction != "include a comparison table" || len(ReportRequirementsForSection(normalized, 1, 2)) != 1 {
		t.Fatalf("unexpected normalized map: %#v", normalized)
	}
	if len(normalized.Requirements) != 2 || normalized.Requirements[1].UnmappedReason != "no matching section in the fixed outline" {
		t.Fatalf("durable normalization lost the unmapped requirement: %#v", normalized)
	}
	ownerBound := ReportOwnerBoundRequirements(normalized)
	if len(ownerBound) != 1 || ownerBound[0].RequirementID != "req_table" {
		t.Fatalf("owner-bound requirement selection leaked unmapped entries: %#v", ownerBound)
	}
	if _, _, err := ReportRequirementMapHash(normalized); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeReportRequirementMapRejectsOwnershipAndTraceGaps(t *testing.T) {
	plan := SectionalReportPlan{Parts: []ReportPlanPart{{Title: "Part", Sections: []ReportPlanSection{{Title: "One"}}}}}
	base := ReportRequirementMap{ReviewedEventIDs: []string{"evt_pending"}, Requirements: []ReportRequirement{{RequirementID: "req_one", Instruction: "keep one", SourceEventIDs: []string{"evt_pending"}, Owner: &ReportRequirementOwner{PartIndex: 1, SectionIndex: 1}}}}
	cases := map[string]func(*ReportRequirementMap){
		"outside outline":   func(value *ReportRequirementMap) { value.Requirements[0].Owner.SectionIndex = 2 },
		"unreviewed source": func(value *ReportRequirementMap) { value.Requirements[0].SourceEventIDs = []string{"evt_other"} },
		"two destinations":  func(value *ReportRequirementMap) { value.Requirements[0].UnmappedReason = "also unmapped" },
		"no destination":    func(value *ReportRequirementMap) { value.Requirements[0].Owner = nil },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := base
			value.ReviewedEventIDs = append([]string(nil), base.ReviewedEventIDs...)
			value.Requirements = append([]ReportRequirement(nil), base.Requirements...)
			owner := *base.Requirements[0].Owner
			value.Requirements[0].Owner = &owner
			mutate(&value)
			if _, err := NormalizeReportRequirementMap(value, plan); err == nil {
				t.Fatal("invalid requirement map was accepted")
			}
		})
	}
}
