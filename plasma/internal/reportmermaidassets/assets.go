package reportmermaidassets

import _ "embed"

//go:embed mermaid.min.js
var mermaidRuntime []byte

//go:embed purify.min.js
var domPurifyRuntime []byte

//go:embed reports_mermaid.js
var renderer []byte

//go:embed reports_mermaid_legend.js
var legendRenderer []byte

//go:embed report_mermaid.css
var stylesheet []byte

//go:embed mermaid.LICENSE
var license []byte

// MermaidRuntime returns a copy of the pinned Mermaid 11.16.0 browser runtime.
func MermaidRuntime() []byte {
	return append([]byte(nil), mermaidRuntime...)
}

// License returns a copy of the pinned Mermaid runtime license notice.
func License() []byte {
	return append([]byte(nil), license...)
}

// DOMPurifyRuntime returns a copy of the sanitizer used by the shared renderer.
func DOMPurifyRuntime() []byte {
	return append([]byte(nil), domPurifyRuntime...)
}

// Renderer returns a copy of the shared Plasma Mermaid renderer.
func Renderer() []byte {
	return append([]byte(nil), renderer...)
}

// LegendRenderer returns a copy of the shared line-chart legend renderer.
func LegendRenderer() []byte {
	return append([]byte(nil), legendRenderer...)
}

// Stylesheet returns a copy of the shared Plasma Mermaid stylesheet.
func Stylesheet() []byte {
	return append([]byte(nil), stylesheet...)
}
