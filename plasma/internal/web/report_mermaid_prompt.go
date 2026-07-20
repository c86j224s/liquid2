package web

const reportMermaidValidationRule = "If you include a Mermaid diagram, keep it simple and stable. Before finalizing that Markdown, call plasma.mermaid.validate with the Mermaid source, revise it if ok is false, and remember that ok true is only a static preflight pass, not a full browser-render guarantee."

func ReportMermaidValidationRule() string {
	return reportMermaidValidationRule
}
