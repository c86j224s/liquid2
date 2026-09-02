package main

import (
	"bytes"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/startuprecovery"
)

func TestWriteStartupRecoveryResultsLogsAllResults(t *testing.T) {
	var stderr bytes.Buffer
	writeStartupRecoveryResults(&stderr, []startuprecovery.Result{
		{Name: "source_candidate_staging", Err: errors.New("source failed")},
		{Name: "future_failure", Err: errors.New("future failed")},
		{Name: "source_candidate_staging", Changed: 2},
		{Name: "future_changed", Changed: 3},
		{Name: "future_zero"},
	})
	want := "source candidate recovery: source failed\n" +
		"startup recovery future_failure: future failed\n" +
		"source candidate recovery: marked 2 interrupted fetches as failed\n" +
		"startup recovery future_changed: changed 3\n"
	if stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
}
