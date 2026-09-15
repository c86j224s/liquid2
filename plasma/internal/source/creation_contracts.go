package source

import (
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

// CreateSourceSnapshotWithEventRequest는 새 raw artifact와 source snapshot, event를
// 같은 transaction으로 생성하는 입력이다.
type CreateSourceSnapshotWithEventRequest struct {
	Artifact artifact.CreateRequest
	Snapshot CreateRequest
	Event    ledger.AppendRequest
}

// SourceSnapshotWithEventResult는 artifact-backed source snapshot 생성 결과다.
type SourceSnapshotWithEventResult struct {
	Artifact artifact.Raw
	Snapshot Snapshot
	Event    ledger.Event
}

// CreateExistingArtifactSourceSnapshotWithEventRequest는 이미 저장된 artifact를 source
// snapshot으로 연결하는 입력이다.
type CreateExistingArtifactSourceSnapshotWithEventRequest struct {
	Snapshot CreateRequest
	Event    ledger.AppendRequest
}

// ExistingArtifactSourceSnapshotWithEventResult는 existing artifact 기반 snapshot 생성 결과다.
type ExistingArtifactSourceSnapshotWithEventResult struct {
	Snapshot Snapshot
	Event    ledger.Event
}

// CreateLiveSourceSnapshotWithEventRequest는 live reference source snapshot과 event를
// 함께 생성하기 위한 입력이다.
type CreateLiveSourceSnapshotWithEventRequest struct {
	Snapshot CreateRequest
	Event    ledger.AppendRequest
}

// LiveSourceSnapshotWithEventResult는 live reference source snapshot 생성 결과다.
type LiveSourceSnapshotWithEventResult struct {
	Snapshot Snapshot
	Event    ledger.Event
}
