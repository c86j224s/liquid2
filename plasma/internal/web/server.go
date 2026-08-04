package web

import (
	"context"
	"embed"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	confluenceconnector "github.com/c86j224s/liquid2/plasma/internal/connectors/confluence"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	workflowruntime "github.com/c86j224s/liquid2/plasma/internal/workflow"
)

//go:embed static/*
var staticFiles embed.FS

// Server는 Plasma browser/API HTTP route를 묶는 adapter state다.
//
// mission/report/source/workflow 제품 규칙은 app/reporting service로 위임하고, 이
// 타입은 HTTP 요청 해석, agent executor wiring, in-flight UI lock, 정적 asset 제공에
// 필요한 process-local 상태만 보관한다.
type Server struct {
	service                     *app.Service
	liquid2                     app.Liquid2SourceConnector
	agent                       AgentExecutor
	agents                      map[string]AgentExecutor
	turns                       missionTurnLocks
	runningTurns                runningAgentTurns
	sources                     missionTurnLocks
	reports                     missionTurnLocks
	runningReports              reportexecution.InFlight
	workflowSupervisor          *workflowruntime.Supervisor
	workflowGoalModel           string
	workflowGoalReasoningEffort string
	confluenceOAuth             confluenceconnector.OAuthConfig
	confluenceOAuthDiscoveryURL string
	confluenceAPIBaseURL        string
	confluenceOAuthStates       confluenceOAuthStates
	fetchURLSource              urlSourceFetchFunc
	renderBrowserURLSource      browserURLSourceRenderFunc
	fetchMedia                  mediaSourceFetchFunc
	fetchPDF                    pdfSourceFetchFunc
	environmentLabel            string
	staticDir                   string
	activityServerID            string
}

// Options는 NewServer가 HTTP adapter를 구성할 때 받는 의존성이다.
//
// nil fetcher와 renderer는 제품 기본 구현으로 채워진다. Agent executor는 제공된
// 값만 등록하므로, agent 실행이 필요한 테스트나 embedding은 명시적으로 주입해야 한다.
type Options struct {
	Liquid2Connector            app.Liquid2SourceConnector
	AgentExecutor               AgentExecutor
	AgentExecutors              map[string]AgentExecutor
	urlFetcher                  urlSourceFetchFunc
	browserRenderer             browserURLSourceRenderFunc
	mediaFetcher                mediaSourceFetchFunc
	pdfFetcher                  pdfSourceFetchFunc
	WorkflowGoalModel           string
	WorkflowGoalReasoningEffort string
	ConfluenceOAuth             confluenceconnector.OAuthConfig
	ConfluenceOAuthDiscoveryURL string
	ConfluenceAPIBaseURL        string
	EnvironmentLabel            string
	// StaticDir은 설정되면 embedded copy 대신 디스크의 정적 asset을 제공한다.
	// 개발 중 edit + refresh를 위한 값이며, release 동작의 source of truth가 아니다.
	StaticDir string
}

type urlSourceFetchFunc func(context.Context, string) (fetchedURLSource, error)
type browserURLSourceRenderFunc func(context.Context, string) (fetchedURLSource, error)
type mediaSourceFetchFunc func(context.Context, string) (fetchedMediaSource, error)
type pdfSourceFetchFunc func(context.Context, string) (fetchedPDFSource, error)

// NewServer는 Plasma HTTP handler tree와 process-local runtime 상태를 구성한다.
//
// 이 함수는 서버를 listen하지 않는다. 호출자는 반환된 http.Handler를 원하는 runtime
// control script나 test server에 연결한다.
func NewServer(service *app.Service, options Options) http.Handler {
	urlFetcher := options.urlFetcher
	if urlFetcher == nil {
		urlFetcher = fetchURLSource
	}
	browserRenderer := options.browserRenderer
	if browserRenderer == nil {
		browserRenderer = renderBrowserURLSource
	}
	mediaFetcher := options.mediaFetcher
	if mediaFetcher == nil {
		mediaFetcher = fetchMediaSource
	}
	pdfFetcher := options.pdfFetcher
	if pdfFetcher == nil {
		pdfFetcher = fetchPDFSource
	}
	agents := map[string]AgentExecutor{}
	if options.AgentExecutor != nil {
		agents["codex"] = options.AgentExecutor
	}
	for name, executor := range options.AgentExecutors {
		normalized := strings.TrimSpace(strings.ToLower(name))
		if normalized != "" && executor != nil {
			agents[normalized] = executor
		}
	}
	server := &Server{
		service:                     service,
		liquid2:                     options.Liquid2Connector,
		agent:                       options.AgentExecutor,
		agents:                      agents,
		workflowGoalModel:           strings.TrimSpace(options.WorkflowGoalModel),
		workflowGoalReasoningEffort: strings.TrimSpace(options.WorkflowGoalReasoningEffort),
		confluenceOAuth:             options.ConfluenceOAuth,
		confluenceOAuthDiscoveryURL: strings.TrimSpace(options.ConfluenceOAuthDiscoveryURL),
		confluenceAPIBaseURL:        strings.TrimSpace(options.ConfluenceAPIBaseURL),
		fetchURLSource:              urlFetcher,
		renderBrowserURLSource:      browserRenderer,
		fetchMedia:                  mediaFetcher,
		fetchPDF:                    pdfFetcher,
		environmentLabel:            strings.TrimSpace(options.EnvironmentLabel),
		staticDir:                   strings.TrimSpace(options.StaticDir),
		activityServerID:            newID("act"),
	}
	server.runningReports.SetNewID(newID)
	server.workflowSupervisor = workflowruntime.NewSupervisor(workflowruntime.SupervisorOptions{
		Service:        service,
		RunnerFactory:  server.workflowRunner,
		AgentAvailable: func(name string) bool { return server.agentExecutor(name) != nil },
		NewID:          newID,
	})
	return server
}
