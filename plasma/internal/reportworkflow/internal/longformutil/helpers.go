package longformutil

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// AnyJSON은 prompt에 포함되는 구조체를 기존 들여쓰기 bytes로 직렬화한다.
func AnyJSON(value any) string {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

// MustJSON은 ledger payload 조립 경계에서 실패할 수 없는 값을 raw JSON으로 바꾼다.
func MustJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

// ValidateSameSessionResult는 provider가 이어받은 session과 같은 session을 반환했는지 검증한다.
func ValidateSameSessionResult(result agentexec.AgentResult, previousSessionID string) (agentexec.AgentResult, error) {
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

// FallbackSessionID는 event producer에 빈 provider session을 남기지 않기 위한 기존 fallback 규칙이다.
func FallbackSessionID(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}

// SafeFilename은 기존 Web report artifact 파일명 정규화 bytes를 보존한다.
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

// WordCount는 기존 Markdown report word count와 같은 strings.Fields 기준을 쓴다.
func WordCount(markdown string) int {
	return len(strings.Fields(markdown))
}

// SHA256Hex는 테스트와 저장 검증에서 raw artifact content identity를 비교할 때 쓴다.
func SHA256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
