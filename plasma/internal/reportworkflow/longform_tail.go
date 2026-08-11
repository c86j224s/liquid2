package reportworkflow

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

// finalTailForPipeline은 canonical plan event에 얼어 있는 final_edit_pipeline을
// reportworkflow topology tail 이름으로 변환한다. 새 plan 생성은 legacy 또는 V3만
// 쓰지만, V1/V2는 이미 저장된 durable event 재개 호환을 위해 계속 인정한다.
func finalTailForPipeline(pipeline string) (FinalTail, error) {
	switch strings.TrimSpace(pipeline) {
	case "":
		return FinalTailLegacy, nil
	case reporting.FinalEditPipelineReaderStyleGateV1:
		return FinalTailV1, nil
	case reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2:
		return FinalTailV2, nil
	case reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3:
		return FinalTailV3, nil
	default:
		return "", fmt.Errorf("%w: unsupported final edit pipeline", producterror.ErrConflict)
	}
}
