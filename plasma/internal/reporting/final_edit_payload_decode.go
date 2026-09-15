package reporting

import (
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func payloadBoolStrict(payload map[string]any, key string) (bool, bool) {
	value, ok := payload[key].(bool)
	return value, ok
}

func payloadIntStrict(payload map[string]any, key string) (int, bool) {
	number, ok := payload[key].(float64)
	if !ok || number < 0 || float64(int(number)) != number {
		return 0, false
	}
	return int(number), true
}

func stringSlicePayload(value any) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	values, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%w: string slice payload is invalid", producterror.ErrConflict)
	}
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		text, ok := value.(string)
		if !ok || text == "" || seen[text] {
			return nil, fmt.Errorf("%w: string slice payload is invalid", producterror.ErrConflict)
		}
		seen[text] = true
		out = append(out, text)
	}
	return out, nil
}
