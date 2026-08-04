package reportexecution

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"
)

func TestLogTerminalWriteFailureUsesSafeStructuredFields(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })
	logTerminalWriteFailure("mis_1", "evt_pending", "draft", "report.draft.failed", errors.New("sqlite busy"))
	line := output.String()
	for _, want := range []string{"report_terminal_write_failed", `mission_id="mis_1"`, `pending_event_id="evt_pending"`, `report_type="draft"`, `intended_event_type="report.draft.failed"`, `err="sqlite busy"`} {
		if !strings.Contains(line, want) {
			t.Fatalf("missing safe structured log field %q: %s", want, line)
		}
	}
}
