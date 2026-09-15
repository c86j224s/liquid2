package reportilphase0

import "testing"

// CompileReportFirstAuthoredPDFFixture keeps the authored draft and compiler
// private while exposing only a test fixture to the external PDF integration test.
func CompileReportFirstAuthoredPDFFixture(t *testing.T) (Document, []string) {
	t.Helper()
	catalog := testSourceCatalog(t, "mis_report_first_pdf")
	receipt := testSourceReadReceipt(catalog)
	draft := reportFirstAuthorDraft{
		Title:    "Takeda Castle and Ikuno Silver Mine",
		Language: "en",
		Sections: []reportFirstSectionDraft{
			{
				Title: "A mountain position in regional history",
				Blocks: []reportFirstBlockDraft{
					{Kind: "prose", Prose: "Takeda Castle matters because defense, routes, and silver production met in one regional setting.", EvidenceSourceKeys: []string{"source_001"}},
					{Kind: "list", Items: []string{"The traditions name 1433 and 1443.", "Stone walls reshaped the site after 1585."}, EvidenceSourceKeys: []string{"source_001"}},
					{Kind: "table", Table: &documentTableDraft{
						Caption: stringPointer("Chronology at a glance"), Column1: "Date", Column2: "Recorded development",
						Rows: []documentTableRowDraft{{Cell1: "1585", Cell2: "A new phase of stone construction"}},
					}, EvidenceSourceKeys: []string{"source_001"}},
				},
			},
			{
				Title: "The evidence boundary and final judgment",
				Blocks: []reportFirstBlockDraft{
					{Kind: "prose", Prose: "The surviving evidence does not prove direct mine administration by the castle.", EvidenceSourceKeys: []string{"source_001"}},
					{Kind: "prose", Prose: "Its stronger significance is the visible convergence of military position, movement, and production.", EvidenceSourceKeys: []string{"source_001"}},
				},
			},
		},
	}
	_, document, err := compileReportFirstAuthorDraft(
		draft,
		"narrative_pdf",
		"doc_pdf",
		"Explain the relationship between Takeda Castle and Ikuno Silver Mine.",
		catalog,
		receipt,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	document.RevisionID = "rev_pdf"
	return document, []string{
		draft.Title,
		draft.Sections[0].Title,
		draft.Sections[0].Blocks[0].Prose,
		draft.Sections[0].Blocks[1].Items[0],
		*draft.Sections[0].Blocks[2].Table.Caption,
		draft.Sections[0].Blocks[2].Table.Rows[0].Cell2,
		draft.Sections[1].Blocks[0].Prose,
		draft.Sections[1].Blocks[1].Prose,
	}
}
