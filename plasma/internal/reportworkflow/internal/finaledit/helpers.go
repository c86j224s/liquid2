package finaledit

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

// ID는 stage package가 기존 ID prefix 순서를 보존해 새 durable identity를 만들 때 쓴다.
func (runner Runner) ID(prefix string) string {
	if runner.NewID != nil {
		return runner.NewID(prefix)
	}
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s_%s_%s", prefix, time.Now().UTC().Format("20060102150405"), hex.EncodeToString(b[:]))
}

// ReportEventString은 plan event payload에서 legacy binding에 복사할 문자열 값을 읽는다.
func ReportEventString(event ledger.Event, key string) string {
	var payload map[string]any
	if json.Unmarshal(event.Payload, &payload) != nil {
		return ""
	}
	value, _ := payload[key].(string)
	return strings.TrimSpace(value)
}

// FirstNonEmpty는 session fallback chain에서 첫 non-empty 값을 고른다.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

// LongFormFinalEditContractReasoningEffort는 final edit pipeline의 기존 default effort 계약이다.
func LongFormFinalEditContractReasoningEffort(value string) string {
	return FirstNonEmpty(value, "default")
}

// RetryNote는 두 번째 final edit stage 시도 prompt에 붙는 기존 기술 retry 문구다.
func RetryNote(attempt int) string {
	if attempt <= 1 {
		return ""
	}
	return "\n\nThis is the one allowed technical retry. Reopen the durable stage and complete the same bound workflow without changing the contract."
}
