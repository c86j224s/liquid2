package sourcecandidates

import "testing"

func TestNormalizeSourceCandidateProposalsDerivesConfluenceTitleFromURL(t *testing.T) {
	candidates, err := NormalizeSourceCandidateProposals([]SourceCandidateProposalInput{{
		URL:    "https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Product+Roadmap#details",
		Reason: "Confluence page should keep its document name when proposed from a connector result.",
	}})
	if err != nil {
		t.Fatalf("NormalizeSourceCandidateProposals returned error: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected one candidate, got %#v", candidates)
	}
	if candidates[0].Title != "Product Roadmap" {
		t.Fatalf("expected Confluence page title fallback, got %#v", candidates[0])
	}
}
