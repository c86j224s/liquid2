package web

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partedit"
)

// reportPartCreatedEventID는 Web recovery projection이 기존 part artifact의
// canonical 생성 이벤트를 좁게 조회하는 adapter다. Part edit 실행·재개 정책은
// reportworkflow/partedit stage가 소유하며, 이 함수는 새 이벤트를 쓰지 않는다.
func (server *Server) reportPartCreatedEventID(ctx context.Context, missionID string, planEventID string, partIndex int, artifactID string) (string, error) {
	return (partedit.Runner{Service: server.service}).SourcePartEventID(ctx, missionID, planEventID, partIndex, artifactID)
}
