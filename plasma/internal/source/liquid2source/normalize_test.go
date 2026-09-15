package liquid2source

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
)

func TestNormalizeSearchLimitKeepsDefaultAndMaximum(t *testing.T) {
	for _, test := range []struct {
		name  string
		input int
		want  int
	}{
		{name: "zero default", input: 0, want: 10},
		{name: "negative default", input: -1, want: 10},
		{name: "maximum", input: 100, want: 100},
		{name: "capped", input: 101, want: 100},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := NormalizeSearchLimit(test.input); got != test.want {
				t.Fatalf("NormalizeSearchLimit(%d) = %d, want %d", test.input, got, test.want)
			}
		})
	}
}

func TestNormalizeFiltersAndCandidateTrimWithoutChangingFlags(t *testing.T) {
	filters := NormalizeFilters(Liquid2SourceFilters{
		Status: " status ", Tag: " tag ", Kind: " kind ", RatingMin: 3,
		IncludeDeleted: true, IncludeTrash: true,
	})
	if filters.Status != "status" || filters.Tag != "tag" || filters.Kind != "kind" || filters.RatingMin != 3 || !filters.IncludeDeleted || !filters.IncludeTrash {
		t.Fatalf("NormalizeFilters() = %#v", filters)
	}

	candidate, err := NormalizeCandidate(Liquid2SourceCandidate{
		Connector: sourcecontract.ConnectorRef{ExternalSourceID: " doc/1 "},
		Title:     " title ", SourceURI: " uri ", Summary: " summary ", CanSnapshot: false,
	})
	if err != nil {
		t.Fatalf("NormalizeCandidate() error = %v", err)
	}
	if candidate.Title != "title" || candidate.SourceURI != "uri" || candidate.Summary != "summary" || !candidate.CanSnapshot {
		t.Fatalf("NormalizeCandidate() = %#v", candidate)
	}
	if candidate.Connector.ExternalSourceID != "doc/1" || candidate.Connector.ExternalURI != DocumentURI(candidate.Connector.ExternalSourceID) || candidate.Connector.ExternalURI != "liquid2://documents/doc/1" {
		t.Fatalf("candidate connector = %#v", candidate.Connector)
	}
}

func TestNormalizeDocumentMetadataFallbackAndValidation(t *testing.T) {
	document, err := NormalizeDocument(Liquid2SourceDocument{
		Connector: sourcecontract.ConnectorRef{},
		Title:     "  ",
		SourceURI: " source ",
	}, " doc/1 ")
	if err != nil {
		t.Fatalf("NormalizeDocument() fallback error = %v", err)
	}
	if document.Connector.ExternalSourceID != "doc/1" || document.Title != "doc/1" || document.SourceURI != "source" || string(document.Metadata) != `{}` {
		t.Fatalf("NormalizeDocument() fallback = %#v", document)
	}

	_, err = NormalizeDocument(Liquid2SourceDocument{Metadata: json.RawMessage(`{"bad":`)}, "doc_1")
	if !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("invalid metadata error = %v, want producterror.ErrInvalidInput", err)
	}
}

func TestNormalizeDocumentRejectsIDMismatchAndMissingConnectorID(t *testing.T) {
	_, err := NormalizeDocument(Liquid2SourceDocument{
		Connector: sourcecontract.ConnectorRef{ExternalSourceID: "other"},
	}, "requested")
	if !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("mismatched id error = %v, want producterror.ErrInvalidInput", err)
	}

	_, err = NormalizeConnector(sourcecontract.ConnectorRef{})
	if !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("missing connector id error = %v, want producterror.ErrInvalidInput", err)
	}
}

func TestDocumentURITrimsWithoutEscapingIdentifier(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "slash", input: "doc/1", want: "liquid2://documents/doc/1"},
		{name: "internal space", input: "doc 1", want: "liquid2://documents/doc 1"},
		{name: "percent", input: "doc%1", want: "liquid2://documents/doc%1"},
		{name: "unicode", input: "문서-1", want: "liquid2://documents/문서-1"},
		{name: "surrounding whitespace", input: "  doc/1  ", want: "liquid2://documents/doc/1"},
		{name: "empty", input: "  ", want: "liquid2://documents/"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := DocumentURI(test.input); got != test.want {
				t.Fatalf("DocumentURI(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}

	connector, err := NormalizeConnector(sourcecontract.ConnectorRef{ExternalSourceID: "  문서/1  "})
	if err != nil {
		t.Fatalf("NormalizeConnector() error = %v", err)
	}
	if connector.ExternalURI != DocumentURI(connector.ExternalSourceID) {
		t.Fatalf("NormalizeConnector() external URI = %q, want %q", connector.ExternalURI, DocumentURI(connector.ExternalSourceID))
	}
}
