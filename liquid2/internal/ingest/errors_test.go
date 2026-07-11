package ingest

import (
	"errors"
	"testing"
)

func TestClassifiedErrorWrapsCause(t *testing.T) {
	cause := errors.New("connection refused")
	err := fetchFailed("request failed", cause)

	if !errors.Is(err, ErrFetchFailed) {
		t.Fatalf("expected fetch classification, got %v", err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("expected wrapped cause, got %v", err)
	}
}
