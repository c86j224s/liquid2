package web

import "github.com/c86j224s/liquid2/plasma/internal/reporting"

func finalEditStyleDiagnosesForWebTest(count int) []reporting.FinalEditStyleOperationDiagnosis {
	records := make([]reporting.FinalEditStyleOperationDiagnosis, 0, count)
	for index := 0; index < count; index++ {
		records = append(records, reporting.FinalEditStyleOperationDiagnosis{
			OperationOrdinal: index + 1,
			Category:         "unnatural_collocation",
			Reason:           "awkward local phrasing",
			MatchText:        "이 작업은 수행되어야 한다.",
			Replacement:      "이 작업은 수행해야 한다.",
			Occurrence:       1,
		})
	}
	return records
}
