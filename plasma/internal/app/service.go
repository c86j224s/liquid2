package app

import (
	"context"
	"sync"

	"github.com/c86j224s/liquid2/plasma/internal/sources/localpath"
	"github.com/c86j224s/liquid2/plasma/internal/version"
)

// Store는 app.Service가 제품 상태 전이에 필요한 저장소 기능을 모은 포트다.
//
// 구체적인 SQL 스키마나 저장 최적화는 어댑터 책임이고, Service는 이 포트를
// 통해서만 제품 상태를 읽고 쓴다.
type Store interface {
	Health(context.Context) error
	MigrationVersions(context.Context) ([]string, error)
	MissionStore
	ProjectionStore
	ArtifactStore
	ResearchRecordStore
	ReportStore
}

// Service는 Plasma 애플리케이션 계층의 진입점이다.
//
// HTTP/MCP/CLI 어댑터는 Service 메서드를 통해서만 미션, source, report, workflow의
// 제품 상태를 바꿔야 한다. 프로세스 로컬 lock은 보조 안전장치이며 장부 이벤트가
// 지속 상태의 기준이다.
type Service struct {
	store      Store
	workflowMu sync.Mutex
	localPaths *localpath.Engine
}

// Health는 저장소 마이그레이션 상태와 빌드 버전을 포함한 서버 상태 view다.
type Health struct {
	Status     string
	Version    string
	Migrations []string
}

// NewService는 저장소 포트만 가진 기본 애플리케이션 서비스를 만든다.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// NewServiceWithLocalPathEngine은 local path source 기능까지 연결한 애플리케이션
// 서비스를 만든다.
func NewServiceWithLocalPathEngine(store Store, engine *localpath.Engine) *Service {
	return &Service{store: store, localPaths: engine}
}

// SetLocalPathEngine은 실행 중 service에 local path Engine을 주입한다.
//
// 일반 서버 구성에서는 생성 시점 주입을 선호하고, 이 메서드는 테스트나 embedding
// 조립용으로만 사용한다.
func (s *Service) SetLocalPathEngine(engine *localpath.Engine) {
	s.localPaths = engine
}

// Health는 저장소 연결과 마이그레이션 목록을 읽어 현재 애플리케이션 상태를 반환한다.
func (s *Service) Health(ctx context.Context) (Health, error) {
	if err := s.store.Health(ctx); err != nil {
		return Health{}, err
	}
	migrations, err := s.store.MigrationVersions(ctx)
	if err != nil {
		return Health{}, err
	}
	return Health{
		Status:     "ok",
		Version:    version.Version,
		Migrations: migrations,
	}, nil
}
