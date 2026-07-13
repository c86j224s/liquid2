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
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	service                     *app.Service
	liquid2                     app.Liquid2SourceConnector
	agent                       AgentExecutor
	agents                      map[string]AgentExecutor
	turns                       missionTurnLocks
	runningTurns                runningAgentTurns
	sources                     missionTurnLocks
	reports                     missionTurnLocks
	runningReports              reporting.InFlight
	workflows                   missionTurnLocks
	runningWorkflow             runningWorkflowRuns
	workflowGoalModel           string
	workflowGoalReasoningEffort string
	confluenceOAuth             confluenceconnector.OAuthConfig
	confluenceOAuthDiscoveryURL string
	confluenceAPIBaseURL        string
	confluenceOAuthStates       confluenceOAuthStates
	fetchURLSource              urlSourceFetchFunc
	fetchMedia                  mediaSourceFetchFunc
	fetchPDF                    pdfSourceFetchFunc
	environmentLabel            string
	staticDir                   string
	activityServerID            string
}

type Options struct {
	Liquid2Connector            app.Liquid2SourceConnector
	AgentExecutor               AgentExecutor
	AgentExecutors              map[string]AgentExecutor
	urlFetcher                  urlSourceFetchFunc
	mediaFetcher                mediaSourceFetchFunc
	pdfFetcher                  pdfSourceFetchFunc
	WorkflowGoalModel           string
	WorkflowGoalReasoningEffort string
	ConfluenceOAuth             confluenceconnector.OAuthConfig
	ConfluenceOAuthDiscoveryURL string
	ConfluenceAPIBaseURL        string
	EnvironmentLabel            string
	// StaticDir, when set, serves static assets from disk instead of the
	// embedded copy — for development (edit + refresh, no rebuild).
	StaticDir string
}

type urlSourceFetchFunc func(context.Context, string) (fetchedURLSource, error)
type mediaSourceFetchFunc func(context.Context, string) (fetchedMediaSource, error)
type pdfSourceFetchFunc func(context.Context, string) (fetchedPDFSource, error)

func NewServer(service *app.Service, options Options) http.Handler {
	urlFetcher := options.urlFetcher
	if urlFetcher == nil {
		urlFetcher = fetchURLSource
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
		fetchMedia:                  mediaFetcher,
		fetchPDF:                    pdfFetcher,
		environmentLabel:            strings.TrimSpace(options.EnvironmentLabel),
		staticDir:                   strings.TrimSpace(options.StaticDir),
		activityServerID:            newID("act"),
	}
	server.runningReports.SetNewID(newID)
	return server
}
