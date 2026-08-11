package reportworkflow

import (
	"context"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestFinalTailForPipelineExactMapping(t *testing.T) {
	tests := []struct {
		name     string
		pipeline string
		want     FinalTail
	}{
		{name: "legacy", pipeline: "", want: FinalTailLegacy},
		{name: "v1", pipeline: reporting.FinalEditPipelineReaderStyleGateV1, want: FinalTailV1},
		{name: "v2", pipeline: reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2, want: FinalTailV2},
		{name: "v3", pipeline: reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3, want: FinalTailV3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := finalTailForPipeline(tc.pipeline)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("tail = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFinalTailForPipelineRejectsUnsupportedLiteral(t *testing.T) {
	_, err := finalTailForPipeline("reader_style_gate_future")
	if !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("err = %v, want conflict", err)
	}
}

func TestFinalizeLongFormPrefixRejectsUnknownTail(t *testing.T) {
	_, err := (Runner{}).FinalizeLongFormPrefix(context.Background(), PrefixOutput{FinalTail: FinalTail("future")})
	if !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("err = %v, want conflict", err)
	}
}
