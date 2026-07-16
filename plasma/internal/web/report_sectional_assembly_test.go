package web

import (
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestAssembleSectionalPartMarkdownNormalizesSectionHeadings(t *testing.T) {
	part := agentReportPart{Title: "제품화 판단"}
	drafts := []sectionalReportDraft{{
		Title: "C4가 보존하는 것",
		Markdown: `## 1.1 C4가 보존하는 것

## C4가 보존하는 것

### 첫 소제목

첫 문장은 그대로 남아야 한다.

### 내부 소제목

두 번째 문장도 그대로 남아야 한다.`,
	}, {
		Title:    "코드 예시",
		Markdown: "### 코드 섹션 맥락\n\n코드 블록은 그대로 남아야 한다.\n\n```markdown\n### 코드 안의 heading 예시\n---\n```\n\n코드 뒤 문장입니다.",
	}}
	assembly := agentPartAssembly{
		Intro: "# 제품화 판단\n\n도입 문장입니다.",
		Transitions: []agentPartTransition{{
			AfterSectionIndex: 1,
			Markdown:          "### 전환 제목\n\n전환 문장입니다.",
		}},
		Closing: "## 마무리\n\n마무리 문장입니다.",
	}

	got := assembleSectionalPartMarkdown(part, drafts, assembly, 0)

	for _, expected := range []string{
		"# Part 1. 제품화 판단",
		"**제품화 판단**",
		"## 1.1 C4가 보존하는 것",
		"**첫 소제목**",
		"첫 문장은 그대로 남아야 한다.",
		"**내부 소제목**",
		"두 번째 문장도 그대로 남아야 한다.",
		"## 1.2 코드 예시",
		"**코드 섹션 맥락**",
		"```markdown\n### 코드 안의 heading 예시\n---\n```",
		"**전환 제목**",
		"**마무리**",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected assembled part to contain %q:\n%s", expected, got)
		}
	}
	for _, unexpected := range []string{
		"## C4가 보존하는 것",
		"**C4가 보존하는 것**",
		"**1.1 C4가 보존하는 것**",
		"### 첫 소제목",
		"### 내부 소제목",
		"### 코드 섹션 맥락",
		"### 전환 제목",
		"## 마무리",
	} {
		if strings.Contains(got, unexpected) {
			t.Fatalf("expected assembled part to normalize %q away:\n%s", unexpected, got)
		}
	}
}

func TestAssembleSectionalFinalMarkdownNormalizesFrameBoundaries(t *testing.T) {
	got := reporting.AssembleLongFormFinalMarkdown("보고서", "# 보고서\n\n읽기 안내입니다.\n\n---", "---\n\n# 결론\n\n## 결론\n\n### 다음 점검\n\n닫는 문장입니다.\n\n---", []string{"# Part 1. Part\n\n본문입니다."})

	for _, expected := range []string{
		"# 보고서",
		"# Part 1. Part",
		"## 결론",
		"## 다음 점검",
		"닫는 문장입니다.",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected final report to contain %q:\n%s", expected, got)
		}
	}
	if strings.Contains(got, "\n---\n\n---\n") {
		t.Fatalf("expected final report to collapse boundary rules:\n%s", got)
	}
	if count := strings.Count(got, "## 결론"); count != 1 {
		t.Fatalf("expected adjacent duplicate conclusion headings to collapse, got %d:\n%s", count, got)
	}
	if strings.Contains(got, "### 다음 점검") {
		t.Fatalf("expected h3+ frame heading to be normalized:\n%s", got)
	}
}
