package reportworkflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// LongFormWorkerLimit는 section_fanout prefix가 동시에 실행할 수 있는 최대 stage 수다.
// Web은 테스트 호환 alias만 둘 수 있고 실제 scheduling 정책은 root runner가 소유한다.
const LongFormWorkerLimit = 8

const longFormWorkerLimit = LongFormWorkerLimit

// SectionFanoutSectionUserText는 section_fanout Section writer의 provider user text 계약이다.
func SectionFanoutSectionUserText(partIndex, sectionIndex int) string {
	return fmt.Sprintf("draft section %d.%d for section-fanout long-form markdown report", partIndex+1, sectionIndex+1)
}

func (runner Runner) id(prefix string) string {
	if runner.newID != nil {
		return runner.newID(prefix)
	}
	return prefix + "_missing"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func forkLongFormSession(ctx context.Context, forker agentexec.AgentSessionForker, sourceSessionID string) (string, string, error) {
	sourceSessionID = strings.TrimSpace(sourceSessionID)
	if sourceSessionID == "" {
		return "", "", fmt.Errorf("%w: section fanout requires a report plan provider session", producterror.ErrConflict)
	}
	fork, err := forker.ForkSession(ctx, sourceSessionID)
	if err != nil {
		return "", "", fmt.Errorf("section fanout session fork failed: %w", err)
	}
	if strings.TrimSpace(fork.SessionID) == "" {
		return "", "", fmt.Errorf("%w: section fanout session fork returned an empty session", producterror.ErrConflict)
	}
	return strings.TrimSpace(fork.SessionID), strings.TrimSpace(fork.SourceSessionID), nil
}
