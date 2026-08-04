package mcp

// ReservedInspectImageToolName은 의도적으로 ListTools에 등록하지 않는 예약 이름이다.
// 이미지 검사는 실제 vision engine이 필요하다. 그 전까지 media source는 metadata만
// 노출하며, source read가 이미지 내용을 검사한 척하면 안 된다.
const ReservedInspectImageToolName = "plasma.research.inspect_image"

type inspectImageInput struct {
	MissionID  string `json:"mission_id"`
	SnapshotID string `json:"snapshot_id"`
	ArtifactID string `json:"artifact_id"`
	Prompt     string `json:"prompt"`
}
