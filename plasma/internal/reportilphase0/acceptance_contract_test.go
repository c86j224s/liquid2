package reportilphase0

import "testing"

func TestContinuityListRepairDoesNotRelaxReaderOrStructure(t *testing.T) {
	original := Document{Blocks: []Block{{NodeID: "list", Kind: "list", Items: []string{"Original supported statement"}}}}
	edited := Document{Blocks: []Block{{NodeID: "list", Kind: "list", Items: []string{"Corrected supported statement"}}}}
	if err := validatePublicationDocumentStructure(original, edited); err != nil {
		t.Fatal(err)
	}
	if err := validateStructuredPayloads(original, edited, true); err != nil {
		t.Fatal(err)
	}
	if err := validatePublicationStructuredPayloads(original, edited); err == nil {
		t.Fatal("reader list mutation accepted")
	}
	edited.Blocks[0].Items = append(edited.Blocks[0].Items, "additional")
	if err := validatePublicationDocumentStructure(original, edited); err == nil {
		t.Fatal("continuity list cardinality changed")
	}
	a := Document{Blocks: []Block{{Kind: "code", Code: "original"}}}
	b := Document{Blocks: []Block{{Kind: "code", Code: "changed"}}}
	if validateStructuredPayloads(a, b, true) == nil {
		t.Fatal("continuity changed protected code")
	}
}

func TestReaderAllowsLocalUncertaintyWithoutSourceTour(t *testing.T) {
	for _, text := range []string{
		"이 우선순위는 재작업 원인과 대안의 역할을 연결한 판단이다. 효과는 아직 입증되지 않았으므로 시험 결과를 확인한 뒤 확대 여부를 정한다.",
		"현재 자료에서 확인되는 문제는 입력 누락이다. 표본의 재작업 비율을 기준선으로 삼는다. 실제 절감 효과는 시험 후 평가한다.",
		"비용은 투입시간으로만 제시되어 있다. 예상 절감 수치는 아직 없다. 결합 효과는 검증되지 않았다. 현재 확인된 시간과 시험 결과를 분리해 기록해야 한다.",
	} {
		if readerFacingAuditVoiceDominates(text) {
			t.Fatalf("local caveat rejected: %q", text)
		}
	}
	if !readerFacingAuditVoiceDominates("자료는 병력 배치를 설명하지 않는다.") {
		t.Fatal("source-tour predicate lost")
	}
}
