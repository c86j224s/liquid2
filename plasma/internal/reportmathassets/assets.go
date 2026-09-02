package reportmathassets

import _ "embed"

//go:embed katex.min.js
var katexRuntime []byte

// KaTeXRuntime returns a copy of the pinned KaTeX 0.17.0 browser runtime.
func KaTeXRuntime() []byte {
	return append([]byte(nil), katexRuntime...)
}
