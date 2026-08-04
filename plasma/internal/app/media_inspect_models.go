package app

import (
	"context"
	"time"
)

// VisionEngine은 이미지 source artifact를 시각 모델로 관찰하는 adapter port다.
type VisionEngine interface {
	InspectImage(context.Context, InspectImageRequest) (InspectImageResult, error)
}

// InspectImageRequest는 승인된 image source artifact를 관찰하기 위한 입력이다.
type InspectImageRequest struct {
	MissionID  string
	SnapshotID string
	ArtifactID string
	Prompt     string
}

// InspectImageResult는 vision model이 반환한 image observation이다.
//
// 이 결과는 source 자체가 아니라 agent-produced result 성격의 관찰값이다.
type InspectImageResult struct {
	MissionID        string    `json:"mission_id"`
	SnapshotID       string    `json:"snapshot_id"`
	ArtifactID       string    `json:"artifact_id,omitempty"`
	Description      string    `json:"description,omitempty"`
	DetectedObjects  []string  `json:"detected_objects,omitempty"`
	OCRText          string    `json:"ocr_text,omitempty"`
	ModelID          string    `json:"model_id"`
	ModelVersion     string    `json:"model_version,omitempty"`
	ObservedAt       time.Time `json:"observed_at"`
	SourceSnapshotID string    `json:"source_snapshot_id"`
}
