package articlepilot

import "testing"

func TestValidateAcceptedMarkdownRejectsActiveContent(t *testing.T) {
	if err := validateAcceptedMarkdown([]byte("# Title\n\nSafe prose.")); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"plain", "# Title\n<script>alert(1)</script>", "# Title\n[j](javascript:bad)"} {
		if err := validateAcceptedMarkdown([]byte(value)); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
}
