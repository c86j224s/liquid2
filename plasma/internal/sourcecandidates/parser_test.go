package sourcecandidates

import (
	"strings"
	"testing"
)

func TestParseRequiresExplicitCandidateLabelAndReason(t *testing.T) {
	for _, text := range []string{
		"참고: https://example.com/report",
		"참고: https://example.com/report\n채택 의견: 원문 대조에 필요합니다.",
		"소스 후보: https://example.com/report",
		"Use https://example.com/report because it looks useful.",
	} {
		if got := Parse(text); len(got) != 0 {
			t.Fatalf("expected no candidates from %q, got %#v", text, got)
		}
	}
}

func TestParseExtractsKoreanCandidateVariants(t *testing.T) {
	got := Parse(strings.Join([]string{
		"후보 소스: https://Example.com/report#section",
		"제목: Example report",
		"검토 이유: 원문 대조와 추가 조사 방향 확인에 필요한 자료입니다.",
	}, "\n"))
	if len(got) != 1 {
		t.Fatalf("expected one candidate, got %#v", got)
	}
	if got[0].URL != "https://example.com/report" {
		t.Fatalf("expected normalized URL, got %#v", got[0])
	}
	if !strings.Contains(got[0].Reason, "추가 조사 방향") {
		t.Fatalf("expected explicit review reason, got %#v", got[0])
	}
}

func TestParseStopsReasonAtNextCandidateBoundary(t *testing.T) {
	got := Parse(strings.Join([]string{
		"소스 후보: https://example.com/a",
		"소스 후보: https://example.com/b",
		"채택 의견: 두 번째 후보의 이유입니다.",
	}, "\n"))
	if len(got) != 1 || got[0].URL != "https://example.com/b" {
		t.Fatalf("expected only the second candidate to receive the reason, got %#v", got)
	}
}

func TestParseDeduplicatesCandidates(t *testing.T) {
	got := Parse(strings.Join([]string{
		"source candidate: https://example.com/report#one",
		"acceptance opinion: primary source candidate.",
		"소스 후보: https://example.com/report#two",
		"채택 의견: 중복 후보입니다.",
	}, "\n"))
	if len(got) != 1 {
		t.Fatalf("expected duplicate URLs to collapse, got %#v", got)
	}
}

func TestParseRejectsCredentialBearingURLs(t *testing.T) {
	got := Parse(strings.Join([]string{
		"source candidate: https://user:secret@example.com/report",
		"acceptance opinion: this must not be recorded with credentials.",
	}, "\n"))
	if len(got) != 0 {
		t.Fatalf("credential-bearing URL must not become a source candidate, got %#v", got)
	}
}
