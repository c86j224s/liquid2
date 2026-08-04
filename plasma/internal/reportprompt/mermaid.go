package reportprompt

const reportMermaidValidationRule = "If you include a Mermaid diagram, keep it simple and stable. Before finalizing that Markdown, call plasma.mermaid.validate with the Mermaid source, revise it if ok is false, and remember that ok true is only a static preflight pass, not a full browser-render guarantee."

// ReportMermaidValidationRule는 리포트 작성 prompt에 붙일 Mermaid 검증 규칙 문장을 반환한다.
func ReportMermaidValidationRule() string {
	return reportMermaidValidationRule
}
