package reporting

import (
	"strings"
	"testing"
)

func TestFormatDirectionHint(t *testing.T) {
	if got := FormatDirectionHint(" \n "); got != "" {
		t.Fatalf("empty hint block = %q", got)
	}
	got := FormatDirectionHint("  compare risks\ncarefully  ")
	if !strings.HasPrefix(got, DirectionAdvisory+"\n\n<request_direction>\n") || !strings.HasSuffix(got, "compare risks\ncarefully\n</request_direction>") {
		t.Fatalf("unexpected block: %q", got)
	}
}
