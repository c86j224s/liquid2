package finaledit

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
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

// AgentReportAnyJSON은 기존 prompt에 포함되는 pretty JSON 표현을 그대로 만든다.
func AgentReportAnyJSON(value any) string {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(encoded)
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

// SafeFilename은 기존 Web helper와 같은 report artifact filename canonicalization이다.
func SafeFilename(title string, ext string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "source"
	}
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ' || r == '.':
			b.WriteRune('-')
		}
	}
	name := strings.Trim(b.String(), "-_")
	if name == "" {
		name = "source"
	}
	if len(name) > 80 {
		name = name[:80]
	}
	return name + ext
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

// ValidatedSameSessionResult는 resumed provider session이 bound session과 같은지 검증한다.
func ValidatedSameSessionResult(result agentexec.AgentResult, previousSessionID string) (agentexec.AgentResult, error) {
	previousSessionID = strings.TrimSpace(previousSessionID)
	result.SessionID = strings.TrimSpace(result.SessionID)
	if previousSessionID == "" {
		if result.SessionID == "" {
			return result, fmt.Errorf("%w: agent did not return a session id", producterror.ErrInvalidInput)
		}
		return result, nil
	}
	if result.SessionID == "" {
		return result, fmt.Errorf("%w: agent did not return a session id for resumed session", producterror.ErrInvalidInput)
	}
	if result.SessionID != previousSessionID {
		result.SessionID = ""
		return result, fmt.Errorf("%w: agent returned a different session id", producterror.ErrInvalidInput)
	}
	return result, nil
}

// RetryNote는 두 번째 final edit stage 시도 prompt에 붙는 기존 기술 retry 문구다.
func RetryNote(attempt int) string {
	if attempt <= 1 {
		return ""
	}
	return "\n\nThis is the one allowed technical retry. Reopen the durable stage and complete the same bound workflow without changing the contract."
}
