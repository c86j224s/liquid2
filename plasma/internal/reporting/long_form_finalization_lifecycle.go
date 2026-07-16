package reporting

import (
	"bytes"
	"encoding/json"
)

// LongFormFinalizationHint is a non-durable retry hint recovered from one
// narrowly malformed legacy final response.
type LongFormFinalizationHint struct {
	OpeningMarkdown string
	ClosingMarkdown string
	Available       bool
}

// RecoverLongFormFinalizationHint accepts only a single root object with
// exactly one trailing comma immediately before its closing brace.
func RecoverLongFormFinalizationHint(text string) LongFormFinalizationHint {
	raw := bytes.TrimSpace([]byte(text))
	comma, ok := rootTrailingComma(raw)
	if !ok {
		return LongFormFinalizationHint{}
	}
	repaired := make([]byte, 0, len(raw)-1)
	repaired = append(repaired, raw[:comma]...)
	repaired = append(repaired, raw[comma+1:]...)

	decoder := json.NewDecoder(bytes.NewReader(repaired))
	opening, closing, ok := decodeLegacyFinalObject(decoder)
	if !ok {
		return LongFormFinalizationHint{}
	}
	return LongFormFinalizationHint{OpeningMarkdown: opening, ClosingMarkdown: closing, Available: true}
}

func rootTrailingComma(raw []byte) (int, bool) {
	if len(raw) < 3 || raw[0] != '{' || raw[len(raw)-1] != '}' {
		return 0, false
	}
	inString, escaped, depth := false, false, 0
	lastRootToken, rootComma := byte(0), -1
	for index, value := range raw {
		if inString {
			if escaped {
				escaped = false
			} else if value == '\\' {
				escaped = true
			} else if value == '"' {
				inString = false
			}
			continue
		}
		switch value {
		case '"':
			inString = true
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth < 0 {
				return 0, false
			}
		case ',':
			if depth == 1 {
				rootComma = index
				lastRootToken = ','
			}
		default:
			if depth == 1 && value != ' ' && value != '\t' && value != '\r' && value != '\n' {
				lastRootToken = value
			}
		}
	}
	return rootComma, !inString && depth == 0 && lastRootToken == ',' && rootComma > 0
}

func decodeLegacyFinalObject(decoder *json.Decoder) (string, string, bool) {
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		return "", "", false
	}
	values := map[string]string{}
	for decoder.More() {
		keyToken, err := decoder.Token()
		key, keyOK := keyToken.(string)
		if err != nil || !keyOK || (key != "front_matter" && key != "closing") {
			return "", "", false
		}
		if _, duplicate := values[key]; duplicate {
			return "", "", false
		}
		var value string
		if err := decoder.Decode(&value); err != nil {
			return "", "", false
		}
		values[key] = value
	}
	end, err := decoder.Token()
	if err != nil || end != json.Delim('}') || len(values) != 2 {
		return "", "", false
	}
	if decoder.More() {
		return "", "", false
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return "", "", false
	}
	return values["front_matter"], values["closing"], true
}
