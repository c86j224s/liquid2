package app

import (
	"context"
	"sync"

	"github.com/c86j224s/liquid2/plasma/internal/sources/localpath"
	"github.com/c86j224s/liquid2/plasma/internal/version"
)

type Store interface {
	Health(context.Context) error
	MigrationVersions(context.Context) ([]string, error)
	MissionStore
	ProjectionStore
	ArtifactStore
	ResearchRecordStore
	ReportStore
}

type Service struct {
	store      Store
	workflowMu sync.Mutex
	localPaths *localpath.Engine
}

type Health struct {
	Status     string
	Version    string
	Migrations []string
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func NewServiceWithLocalPathEngine(store Store, engine *localpath.Engine) *Service {
	return &Service{store: store, localPaths: engine}
}

func (s *Service) SetLocalPathEngine(engine *localpath.Engine) {
	s.localPaths = engine
}

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
