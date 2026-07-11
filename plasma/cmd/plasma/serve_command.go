package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/config"
	confluenceconnector "github.com/c86j224s/liquid2/plasma/internal/connectors/confluence"
	liquid2connector "github.com/c86j224s/liquid2/plasma/internal/connectors/liquid2"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
	"github.com/c86j224s/liquid2/plasma/internal/web"
)

func runServe(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	addr := fs.String("addr", "", "HTTP listen address")
	liquid2URL := fs.String("liquid2-url", "", "optional Liquid2 base URL")
	localRoots := repeatedStringFlag{}
	fs.Var(&localRoots, "local-source-root", "allowlisted local source root root_id=path; repeatable")
	agentMode := fs.String("agent", "", "agent executors: none, codex, claude, or comma-separated")
	codexCommand := fs.String("codex-command", "", "Codex CLI command")
	claudeCommand := fs.String("claude-command", "", "Claude Code CLI command")
	claudeModel := fs.String("claude-model", "", "Claude model alias for Plasma turns")
	claudeMaxBudgetUSD := fs.String("claude-max-budget-usd", "", "optional Claude max budget per turn")
	agentWorkDir := fs.String("agent-workdir", "", "agent working directory; defaults to a temp directory")
	agentTimeout := fs.Duration("agent-timeout", 0, "agent response timeout; 0 disables the limit")
	workflowGoalModel := fs.String("workflow-goal-model", "", "optional model override for workflow goal draft generation")
	workflowGoalReasoningEffort := fs.String("workflow-goal-reasoning-effort", "", "optional reasoning effort override for workflow goal draft generation")
	environmentLabel := fs.String("environment-label", "", "optional short runtime environment label shown in the browser UI")
	confluenceOAuthClientID := fs.String("confluence-oauth-client-id", "", "Atlassian OAuth 3LO client id")
	confluenceOAuthClientSecret := fs.String("confluence-oauth-client-secret", "", "Atlassian OAuth 3LO client secret")
	confluenceOAuthRedirectURI := fs.String("confluence-oauth-redirect-uri", "", "Atlassian OAuth 3LO callback URL")
	confluenceOAuthAuthorizeURL := fs.String("confluence-oauth-authorize-url", "", "override Atlassian OAuth authorize URL")
	confluenceOAuthTokenURL := fs.String("confluence-oauth-token-url", "", "override Atlassian OAuth token URL")
	confluenceOAuthDiscoveryURL := fs.String("confluence-oauth-discovery-url", "", "override Atlassian accessible-resources base URL")
	confluenceOAuthScopes := repeatedStringFlag{}
	fs.Var(&confluenceOAuthScopes, "confluence-oauth-scope", "Atlassian OAuth scope; repeatable")
	staticDir := fs.String("static-dir", "", "serve static web assets from this directory instead of the embedded copy (dev: edit + refresh, no rebuild)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, effectiveAgentTimeout, err := loadAgentConfig(config.Args{
		DBPath:                      stringFlagArg(fs, "db", *dbPath),
		Addr:                        stringFlagArg(fs, "addr", *addr),
		Liquid2URL:                  stringFlagArg(fs, "liquid2-url", *liquid2URL),
		Agent:                       stringFlagArg(fs, "agent", *agentMode),
		CodexCommand:                stringFlagArg(fs, "codex-command", *codexCommand),
		ClaudeCommand:               stringFlagArg(fs, "claude-command", *claudeCommand),
		ClaudeModel:                 stringFlagArg(fs, "claude-model", *claudeModel),
		ClaudeMaxBudgetUSD:          stringFlagArg(fs, "claude-max-budget-usd", *claudeMaxBudgetUSD),
		AgentWorkDir:                stringFlagArg(fs, "agent-workdir", *agentWorkDir),
		AgentTimeout:                durationFlagArg(fs, "agent-timeout", *agentTimeout),
		WorkflowGoalModel:           stringFlagArg(fs, "workflow-goal-model", *workflowGoalModel),
		WorkflowGoalReasoningEffort: stringFlagArg(fs, "workflow-goal-reasoning-effort", *workflowGoalReasoningEffort),
		EnvironmentLabel:            stringFlagArg(fs, "environment-label", *environmentLabel),
		StaticDir:                   stringFlagArg(fs, "static-dir", *staticDir),
		LocalSourceRoots:            listFlagArg(fs, "local-source-root", []string(localRoots)),
		ConfluenceOAuthClientID:     stringFlagArg(fs, "confluence-oauth-client-id", *confluenceOAuthClientID),
		ConfluenceOAuthClientSecret: stringFlagArg(fs, "confluence-oauth-client-secret", *confluenceOAuthClientSecret),
		ConfluenceOAuthRedirectURI:  stringFlagArg(fs, "confluence-oauth-redirect-uri", *confluenceOAuthRedirectURI),
		ConfluenceOAuthScopes:       listFlagArg(fs, "confluence-oauth-scope", []string(confluenceOAuthScopes)),
		ConfluenceOAuthAuthorizeURL: stringFlagArg(fs, "confluence-oauth-authorize-url", *confluenceOAuthAuthorizeURL),
		ConfluenceOAuthTokenURL:     stringFlagArg(fs, "confluence-oauth-token-url", *confluenceOAuthTokenURL),
		ConfluenceOAuthDiscoveryURL: stringFlagArg(fs, "confluence-oauth-discovery-url", *confluenceOAuthDiscoveryURL),
	})
	if err != nil {
		fmt.Fprintf(stderr, "config: %v\n", err)
		return 2
	}
	applyServeDefaults(&cfg)
	resolvedStaticDir, err := resolveServeStaticDir(cfg.StaticDir)
	if err != nil {
		fmt.Fprintf(stderr, "config: %v\n", err)
		return 2
	}
	effectiveDBPath := cfg.EffectiveDBPath()
	store, err := sqlite.Open(ctx, effectiveDBPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer store.Close()
	svc, err := newCLIService(store, cfg, nil)
	if err != nil {
		fmt.Fprintf(stderr, "local source roots: %v\n", err)
		return 2
	}
	effectiveLocalRoots := cfg.LocalSourceRoots

	var connector app.Liquid2SourceConnector
	if strings.TrimSpace(cfg.Liquid2URL) != "" {
		connector, err = liquid2connector.NewClient(cfg.Liquid2URL)
		if err != nil {
			fmt.Fprintf(stderr, "liquid2 connector: %v\n", err)
			return 2
		}
	}

	agents, err := buildAgentExecutorMap(ctx, cliAgentConfig{
		AgentName:          strings.TrimSpace(cfg.Agent),
		DBPath:             effectiveDBPath,
		Liquid2URL:         strings.TrimSpace(cfg.Liquid2URL),
		CodexCommand:       strings.TrimSpace(cfg.CodexCommand),
		ClaudeCommand:      strings.TrimSpace(cfg.ClaudeCommand),
		ClaudeModel:        strings.TrimSpace(cfg.ClaudeModel),
		ClaudeMaxBudgetUSD: strings.TrimSpace(cfg.ClaudeMaxBudgetUSD),
		AgentWorkDir:       strings.TrimSpace(cfg.AgentWorkDir),
		AgentTimeout:       effectiveAgentTimeout,
		LocalRoots:         effectiveLocalRoots,
	})
	if err != nil {
		fmt.Fprintf(stderr, "agent: %v\n", err)
		return 2
	}

	server := &http.Server{
		Addr: strings.TrimSpace(cfg.Addr),
		Handler: web.NewServer(svc, web.Options{
			Liquid2Connector:            connector,
			AgentExecutor:               agents["codex"],
			AgentExecutors:              agents,
			WorkflowGoalModel:           strings.TrimSpace(cfg.WorkflowGoalModel),
			WorkflowGoalReasoningEffort: strings.TrimSpace(cfg.WorkflowGoalReasoningEffort),
			EnvironmentLabel:            strings.TrimSpace(cfg.EnvironmentLabel),
			ConfluenceOAuth: confluenceconnector.OAuthConfig{
				ClientID:     strings.TrimSpace(cfg.ConfluenceOAuthClientID),
				ClientSecret: strings.TrimSpace(cfg.ConfluenceOAuthClientSecret),
				RedirectURI:  strings.TrimSpace(cfg.ConfluenceOAuthRedirectURI),
				Scopes:       cfg.ConfluenceOAuthScopes,
				AuthorizeURL: strings.TrimSpace(cfg.ConfluenceOAuthAuthorizeURL),
				TokenURL:     strings.TrimSpace(cfg.ConfluenceOAuthTokenURL),
			},
			ConfluenceOAuthDiscoveryURL: strings.TrimSpace(cfg.ConfluenceOAuthDiscoveryURL),
			StaticDir:                   resolvedStaticDir,
		}),
	}
	fmt.Fprintf(stdout, "plasma serving http://%s db=%s\n", server.Addr, cfg.DisplayDBPath())
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(stderr, "serve: %v\n", err)
		return 1
	}
	return 0
}

func runStatus(_ context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	urlOnly := fs.Bool("url", false, "print only the resolved browser URL")
	jsonOut := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.Load(config.Args{})
	if err != nil {
		fmt.Fprintf(stderr, "config: %v\n", err)
		return 2
	}
	applyServeDefaults(&cfg)
	mode := serveRuntimeMode()
	baseURL := "http://" + strings.TrimSpace(cfg.Addr)
	if *urlOnly {
		fmt.Fprintln(stdout, baseURL)
		return 0
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{
			"mode":        mode,
			"url":         baseURL,
			"addr":        strings.TrimSpace(cfg.Addr),
			"db":          cfg.DisplayDBPath(),
			"liquid2_url": strings.TrimSpace(cfg.Liquid2URL),
			"agent":       strings.TrimSpace(cfg.Agent),
		})
		return 0
	}

	label := "development"
	if mode == config.RuntimeModeRelease {
		label = "release"
	}
	fmt.Fprintf(stdout, "Plasma %s\n", label)
	fmt.Fprintf(stdout, "  URL     %s\n", baseURL)
	fmt.Fprintf(stdout, "  DB      %s\n", cfg.DisplayDBPath())
	fmt.Fprintf(stdout, "  Mode    %s\n", mode)
	fmt.Fprintf(stdout, "  Liquid2 %s\n", strings.TrimSpace(cfg.Liquid2URL))
	fmt.Fprintf(stdout, "  Agent   %s\n", strings.TrimSpace(cfg.Agent))
	return 0
}

func applyServeDefaults(cfg *config.Config) {
	mode := serveRuntimeMode()
	if strings.TrimSpace(cfg.Addr) == "" {
		if mode == config.RuntimeModeRelease {
			cfg.Addr = "127.0.0.1:3002"
		} else {
			cfg.Addr = "127.0.0.1:6002"
		}
	}
	if strings.TrimSpace(cfg.DBPath) == "" {
		cfg.DBPath = defaultServeDBPath(mode)
	}
	if strings.TrimSpace(cfg.Liquid2URL) == "" {
		if mode == config.RuntimeModeRelease {
			cfg.Liquid2URL = "http://127.0.0.1:3011"
		} else {
			cfg.Liquid2URL = "http://127.0.0.1:6011"
		}
	}
	if strings.TrimSpace(cfg.Agent) == "" {
		cfg.Agent = "codex"
	}
	if strings.TrimSpace(cfg.AgentWorkDir) == "" {
		if mode == config.RuntimeModeRelease {
			cfg.AgentWorkDir = filepath.Join(os.TempDir(), "plasma-release-agent-workdir")
		} else {
			cfg.AgentWorkDir = filepath.Join(os.TempDir(), "plasma-agent-workdir")
		}
	}
	if strings.TrimSpace(cfg.WorkflowGoalReasoningEffort) == "" {
		cfg.WorkflowGoalReasoningEffort = "low"
	}
}

func serveRuntimeMode() string {
	mode, err := config.RuntimeMode()
	if err != nil {
		return config.RuntimeModeRelease
	}
	return mode
}

func defaultServeDBPath(mode string) string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	if mode == config.RuntimeModeRelease {
		return filepath.Join(home, "Library", "Application Support", "Plasma", "plasma.db")
	}
	return filepath.Join(home, "research-artifacts", "liquid2", "plasma", "runtime", "dev-6002", "plasma-ui-user.db")
}

func resolveServeStaticDir(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if strings.EqualFold(value, "auto") {
		return filepath.Join("internal", "web", "static"), nil
	}
	switch strings.ToLower(value) {
	case "0", "1", "false", "true", "no", "yes", "off", "on", "none", "default", "embed", "embedded":
		return "", fmt.Errorf("invalid static_dir %q: omit static_dir for embedded assets, use %q for repo static assets, or provide an explicit directory path", value, "auto")
	}
	return value, nil
}
