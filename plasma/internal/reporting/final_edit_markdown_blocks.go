package reporting

import ()

func markdownNonEmptyBlockSpans(text string) []markdownBlockSpan {
	blocks := []markdownBlockSpan{}
	start := -1
	lastNonSpace := -1
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			j := i + 1
			for j < len(text) && (text[j] == ' ' || text[j] == '\t' || text[j] == '\r') {
				j++
			}
			if j < len(text) && text[j] == '\n' {
				if start >= 0 {
					blocks = append(blocks, markdownBlockSpan{Start: start, End: lastNonSpace + 1, Text: text[start : lastNonSpace+1]})
					start = -1
					lastNonSpace = -1
				}
				i = j
				continue
			}
		}
		if !isASCIISpace(text[i]) {
			if start < 0 {
				start = i
			}
			lastNonSpace = i
		}
	}
	if start >= 0 {
		blocks = append(blocks, markdownBlockSpan{Start: start, End: lastNonSpace + 1, Text: text[start : lastNonSpace+1]})
	}
	return blocks
}

func isASCIISpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
