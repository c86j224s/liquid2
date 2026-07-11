package app

import (
	"context"
	"time"
)

type VisionEngine interface {
	InspectImage(context.Context, InspectImageRequest) (InspectImageResult, error)
}

type InspectImageRequest struct {
	MissionID  string
	SnapshotID string
	ArtifactID string
	Prompt     string
}

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
