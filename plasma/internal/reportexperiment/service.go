package reportexperiment

import (
	"context"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow"
)

// Service는 fixed Part seed와 제품 finalization tail이 공유하는 consumer-side 포트다.
type Service interface {
	reportworkflow.Service
	reportexecution.Service
	CreateMission(context.Context, mission.CreateRequest) (mission.Mission, error)
}

// ServiceHandle은 실행별 DB service와 close 함수를 함께 보존한다.
type ServiceHandle struct {
	Service Service
	Close   func() error
}

// ServiceFactory는 run directory 안의 DB path를 받아 제품 service를 연다.
type ServiceFactory func(context.Context, string) (ServiceHandle, error)

// ExecutorContext는 archive-local DB와 run directory에 묶인 Codex adapter 생성 입력이다.
type ExecutorContext struct {
	Service Service
	RunDir  string
	DBPath  string
}

// ExecutorFactory는 실행별 archive 경로가 준비된 뒤 Codex executor를 조립한다.
type ExecutorFactory func(context.Context, ExecutorContext) (agentexec.AgentExecutor, error)

// Config는 fixed-Part finalization-only experiment 한 번의 실행 계약이다.
type Config struct {
	ArchiveRoot          string
	FixturePath          string
	RunID                string
	RepositoryRoot       string
	AgentModel           string
	AgentReasoningEffort string
	BinaryPair           BinaryPair
	ServiceFactory       ServiceFactory
	ExecutorFactory      ExecutorFactory
	StartedAt            time.Time
}

// Result는 archive에 기록된 run output 위치와 compact manifest다.
type Result struct {
	RunDir       string
	DBPath       string
	ReportPath   string
	LedgerPath   string
	ManifestPath string
	Manifest     Manifest
}
