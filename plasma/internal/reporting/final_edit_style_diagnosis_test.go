package reporting

func finalEditStyleDiagnosesForTest(count int) []FinalEditStyleOperationDiagnosis {
	records := make([]FinalEditStyleOperationDiagnosis, 0, count)
	for index := 0; index < count; index++ {
		records = append(records, FinalEditStyleOperationDiagnosis{
			OperationOrdinal: index + 1,
			Category:         "unnatural_collocation",
			Reason:           "awkward local phrasing",
			MatchText:        "Preserved body.",
			Replacement:      "Preserved body!",
			Occurrence:       1,
		})
	}
	return records
}
