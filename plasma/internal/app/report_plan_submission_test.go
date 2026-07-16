package app

import (
	"context"
	"errors"
	"testing"
)

func TestReportPlanSubmissionRequiresConditionalLedgerStore(t *testing.T) {
	svc := NewService(fakeStore{})
	_, err := svc.SubmitReportPlan(context.Background(), ReportPlanSubmissionRequest{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected conditional capability error, got %v", err)
	}
	_, err = svc.PromoteReportPlan(context.Background(), PromoteReportPlanRequest{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected conditional capability error, got %v", err)
	}
}
