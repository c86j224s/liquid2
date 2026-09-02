package startuprecovery

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestRunPreservesOrderAndResults(t *testing.T) {
	var order []string
	results, err := Run(context.Background(), []Step{
		{Name: "first", Run: func(context.Context) (int, error) { order = append(order, "first"); return 1, nil }},
		{Name: "second", Run: func(context.Context) (int, error) { order = append(order, "second"); return 2, nil }},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(order, []string{"first", "second"}) {
		t.Fatalf("order = %v", order)
	}
	if !reflect.DeepEqual(results, []Result{{Name: "first", Changed: 1}, {Name: "second", Changed: 2}}) {
		t.Fatalf("results = %#v", results)
	}
}

func TestRunContinuesAfterFailures(t *testing.T) {
	firstErr := errors.New("first failure")
	lastErr := errors.New("last failure")
	var order []string
	results, err := Run(context.Background(), []Step{
		{Name: "first", Run: func(context.Context) (int, error) { order = append(order, "first"); return 3, firstErr }},
		{Name: "middle", Run: func(context.Context) (int, error) { order = append(order, "middle"); return 4, nil }},
		{Name: "last", Run: func(context.Context) (int, error) { order = append(order, "last"); return 5, lastErr }},
	})
	if !reflect.DeepEqual(order, []string{"first", "middle", "last"}) || len(results) != 3 {
		t.Fatalf("order=%v results=%d", order, len(results))
	}
	if !errors.Is(err, firstErr) || !errors.Is(err, lastErr) {
		t.Fatalf("aggregate error does not preserve originals: %v", err)
	}
	if !strings.Contains(err.Error(), "first") || !strings.Contains(err.Error(), "last") {
		t.Fatalf("aggregate error does not name steps: %v", err)
	}
	if results[0].Changed != 3 || results[0].Err != firstErr || results[2].Changed != 5 || results[2].Err != lastErr {
		t.Fatalf("results = %#v", results)
	}
}

func TestRunInvalidStepsDoNotBlockLaterValidSteps(t *testing.T) {
	results, err := Run(context.Background(), []Step{
		{Name: "   "},
		{Name: "missing-run"},
		{Name: "valid", Run: func(context.Context) (int, error) { return 7, nil }},
	})
	if err == nil || len(results) != 3 || results[2].Changed != 7 {
		t.Fatalf("err=%v results=%#v", err, results)
	}
	if !strings.Contains(err.Error(), "step 0") || !strings.Contains(err.Error(), "missing-run") {
		t.Fatalf("validation error is not identifiable: %v", err)
	}
}
