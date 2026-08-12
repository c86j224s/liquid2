package agentexec

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
)

const codexSessionTelemetryTailBytes int64 = 4 << 20

// codexTelemetrySessionID preserves the resumed session identity when Codex
// omits thread.started from a resumed run's JSONL output.
func codexTelemetrySessionID(returnedSessionID string, previousSessionID string) string {
	if returnedSessionID = strings.TrimSpace(returnedSessionID); returnedSessionID != "" {
		return returnedSessionID
	}
	return strings.TrimSpace(previousSessionID)
}

// codexContextWindowMetrics reads only the bounded tail of provider-owned
// session state. It returns telemetry, never session content or a product ID.
func (executor CodexExecutor) codexContextWindowMetrics(sessionID string) (agentusage.ContextWindowMetrics, bool) {
	codexHome := codexHomeFromEnv(executor.Env)
	if codexHome == "" || sessionID == "" {
		return agentusage.ContextWindowMetrics{}, false
	}
	path, err := findCodexSessionFile(codexHome, sessionID)
	if err != nil {
		return agentusage.ContextWindowMetrics{}, false
	}
	tail, err := readBoundedFileTail(path, codexSessionTelemetryTailBytes)
	if err != nil {
		return agentusage.ContextWindowMetrics{}, false
	}
	return agentusage.ParseCodexContextWindowMetrics(string(tail))
}

func readBoundedFileTail(path string, limit int64) ([]byte, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("tail limit must be positive")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	start := info.Size() - limit
	if start < 0 {
		start = 0
	}
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return nil, err
	}
	return io.ReadAll(io.LimitReader(file, limit))
}
