package partassembly

import (
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestAssembleMarkdownUsesOnePartNumber(t *testing.T) {
	tests := []struct {
		name      string
		partIndex int
		title     string
		want      string
	}{
		{name: "Korean part prefix", partIndex: 0, title: "1부. 권위의 원형", want: "# Part 1. 권위의 원형\n"},
		{name: "English part prefix", partIndex: 1, title: "Part 2. 권위와 실권", want: "# Part 2. 권위와 실권\n"},
		{name: "semantic Korean number", partIndex: 0, title: "1부 리그 운영 방식", want: "# Part 1. 1부 리그 운영 방식\n"},
		{name: "semantic year", partIndex: 2, title: "2022 프로그래밍 언어 추세", want: "# Part 3. 2022 프로그래밍 언어 추세\n"},
		{name: "empty title", partIndex: 3, want: "# Part 4.\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			markdown := AssembleMarkdown(reporting.ReportPlanPart{Title: tt.title}, nil, reporting.PartAssembly{}, tt.partIndex)
			if markdown != tt.want {
				t.Fatalf("AssembleMarkdown() = %q, want %q", markdown, tt.want)
			}
		})
	}
}

func TestAssembleMarkdownUsesOneSectionNumber(t *testing.T) {
	part := reporting.ReportPlanPart{Title: "Async state"}
	drafts := []SectionDraft{
		{Title: "4. Future는 아직 완료되지 않았을 수 있는 값이다", Markdown: "# 4. Future는 아직 완료되지 않았을 수 있는 값이다\n\n첫 본문."},
		{Title: "2.2 8. Poll::Pending과 Poll::Ready는 상태 전이의 계약이다", Markdown: "## 2.2 8. Poll::Pending과 Poll::Ready는 상태 전이의 계약이다\n\n둘째 본문."},
	}

	markdown := AssembleMarkdown(part, drafts, reporting.PartAssembly{}, 1)
	for _, heading := range []string{
		"## 2.1 Future는 아직 완료되지 않았을 수 있는 값이다",
		"## 2.2 Poll::Pending과 Poll::Ready는 상태 전이의 계약이다",
	} {
		if !strings.Contains(markdown, heading) {
			t.Fatalf("assembled Markdown missing normalized heading %q:\n%s", heading, markdown)
		}
	}
	for _, duplicate := range []string{"## 2.1 4.", "## 2.2 8."} {
		if strings.Contains(markdown, duplicate) {
			t.Fatalf("assembled Markdown kept duplicate numbering %q:\n%s", duplicate, markdown)
		}
	}
	if strings.Count(markdown, "Future는 아직 완료되지 않았을 수 있는 값이다") != 1 ||
		strings.Count(markdown, "Poll::Pending과 Poll::Ready는 상태 전이의 계약이다") != 1 {
		t.Fatalf("assembled Markdown did not preserve each title body exactly once:\n%s", markdown)
	}
}

func TestDisplayHeadingTextPreservesSemanticLeadingNumbers(t *testing.T) {
	for _, title := range []string{
		"2022년 이후 언어 생태계의 변화",
		"2022 프로그래밍 언어 추세",
		"3.11의 예외 그룹",
		"3.11 릴리스의 예외 그룹",
		"HTTP/2의 스트림 우선순위",
	} {
		if got := displayHeadingText(title, 1, 1); got != title {
			t.Fatalf("displayHeadingText(%q) = %q", title, got)
		}
	}
}
