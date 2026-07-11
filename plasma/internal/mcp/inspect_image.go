package mcp

// ReservedInspectImageToolName is intentionally not registered in ListTools.
// Image inspection needs a real vision engine. Until one exists, media sources
// expose metadata only; source reads must not pretend to inspect image content.
const ReservedInspectImageToolName = "plasma.research.inspect_image"

type inspectImageInput struct {
	MissionID  string `json:"mission_id"`
	SnapshotID string `json:"snapshot_id"`
	ArtifactID string `json:"artifact_id"`
	Prompt     string `json:"prompt"`
}
