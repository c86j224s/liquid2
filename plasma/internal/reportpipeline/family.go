// Package reportpipeline defines the closed report pipeline family contract.
package reportpipeline

const (
	ExperimentalIL                        = "report_il_experimental"
	ExperimentalILValidationProfilesGraph = "report_il_validation_profiles_v7"
	ExperimentalILEditorialMemoryGraph    = "report_il_source_anchored_images_v6"
	ExperimentalILEditorialGraph          = "report_il_editorial_v2"
	ExperimentalILReaderGraph             = "report_il_reader_v1"
	ExperimentalILFlowGraph               = "report_il_flow_v1"
	Unverified                            = "report_unverified"
)

// Independent reports whether a family runs outside the classic report workflow.
func Independent(family string) bool {
	switch family {
	case ExperimentalIL, Unverified:
		return true
	default:
		return false
	}
}
