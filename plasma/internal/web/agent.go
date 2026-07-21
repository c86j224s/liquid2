package web

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentmodels"
	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type AgentExecutor interface {
	Run(context.Context, AgentRequest) (AgentResult, error)
}

type AgentRequest struct {
	UserText          string
	Prompt            string
	Model             string
	ReasoningEffort   string
	MissionID         string
	ToolSessionID     string
	UserEventID       string
	PreviousSessionID string
	AgentExecutor     string
	MCPMode           string
	Compaction        bool
	DisableTools      bool
	ExtraMCPTools     []string
	ReplaceMCPTools   bool
	ReportPatch       *AgentReportPatchContext
	ReportPlan        *AgentReportPlanContext
	PartAssembly      *reporting.PartAssemblyBinding
	LongFormFinalize  *reporting.LongFormFinalizeBinding
}

type AgentReportPlanContext struct {
	PendingEventID            string
	ReportMode                string
	IdempotencyKey            string
	PreviousProviderSessionID string
	AgentModel                string
	AgentReasoningEffort      string
}

type AgentReportPatchContext struct {
	BaseArtifactID               string
	PendingEventID               string
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	MCPMode                      string
	AgentSessionID               string
	PreviousAgentSessionID       string
	ReturnedAgentSessionID       string
	ReportSessionID              string
	ForkSourceAgentSessionID     string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	SessionChainKind             string
}

type AgentResult struct {
	Text      string
	SessionID string
	Resumed   bool
	Log       string
	Usage     agentusage.AgentUsage
}

type AgentSessionForkResult struct {
	SessionID       string
	SourceSessionID string
	SourceHash      string
	CloneHash       string
	SourceSizeBytes int64
	CloneSizeBytes  int64
}

type AgentSessionForker interface {
	ForkSession(context.Context, string) (AgentSessionForkResult, error)
}

type AgentSessionForkReadiness interface {
	CheckForkSession(context.Context, string) error
}

type CodexExecutor struct {
	Command   string
	WorkDir   string
	Timeout   time.Duration
	Env       []string
	MCPServer CodexMCPServer
}

type CodexMCPServer struct {
	Name              string
	Command           string
	Args              []string
	Required          bool
	StartupTimeoutSec int
	ToolTimeoutSec    int
	EnabledTools      []string
}

func (executor CodexExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	model, effort, err := agentmodels.ResolveForSession(req.Model, req.ReasoningEffort, req.PreviousSessionID)
	if err != nil {
		return AgentResult{}, fmt.Errorf("invalid Codex model settings: %w", err)
	}
	req.Model = model
	req.ReasoningEffort = effort
	command := strings.TrimSpace(executor.Command)
	if command == "" {
		command = "codex"
	}
	command = resolveAgentCommand(command)
	workDir := strings.TrimSpace(executor.WorkDir)
	if workDir == "" {
		workDir = "."
	}
	if err := ensureAgentWorkDir(workDir); err != nil {
		return AgentResult{}, err
	}
	tmp, err := os.CreateTemp("", "plasma-codex-last-*.txt")
	if err != nil {
		return AgentResult{}, err
	}
	lastPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(lastPath)

	timeout := executor.Timeout
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	args := []string{"exec"}
	mcpArgs := codexMCPConfigArgs(executor.MCPServer, req)
	if model := strings.TrimSpace(req.Model); model != "" {
		args = append(args, "--model", model)
	}
	if effort := strings.TrimSpace(req.ReasoningEffort); effort != "" {
		args = append(args, "-c", "model_reasoning_effort="+strconv.Quote(effort))
	}
	args = append(args, "--json")
	resumed := strings.TrimSpace(req.PreviousSessionID) != ""
	if resumed {
		args = append(args, "resume")
		args = append(args, mcpArgs...)
		args = append(args,
			"-c", `sandbox_mode="read-only"`,
			"--skip-git-repo-check",
			"--ignore-rules",
			"--output-last-message", lastPath,
			strings.TrimSpace(req.PreviousSessionID),
			"-",
		)
	} else {
		args = append(args, mcpArgs...)
		args = append(args,
			"--sandbox", "read-only",
			"--skip-git-repo-check",
			"--ignore-rules",
			"-C", workDir,
			"--output-last-message", lastPath,
			"-",
		)
	}

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = workDir
	cmd.Env = codexEnvironment(executor.Env)
	prompt := req.Prompt
	if req.Compaction && resumed {
		prompt = "/compact"
	}
	usage := agentusage.New("codex", codexUsageExecutorName(req.AgentExecutor), req.Model, req.ReasoningEffort, prompt).
		WithSession(req.PreviousSessionID, "", resumed, req.Compaction)
	cmd.Stdin = strings.NewReader(prompt)
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined

	err = cmd.Run()
	log := combined.String()
	sessionID := codexSessionID(log)
	usage = codexUsageFromLog(usage, log, req.PreviousSessionID, sessionID, resumed, req.Compaction)
	if ctx.Err() == context.Canceled {
		return AgentResult{Log: log, Resumed: resumed, SessionID: sessionID, Usage: usage}, context.Canceled
	}
	if ctx.Err() == context.DeadlineExceeded {
		if timeout <= 0 {
			return AgentResult{Log: log, Resumed: resumed, SessionID: sessionID, Usage: usage}, context.DeadlineExceeded
		}
		return AgentResult{Log: log, Resumed: resumed, SessionID: sessionID, Usage: usage}, fmt.Errorf("agent timed out after %s", timeout)
	}
	if err != nil {
		return AgentResult{Log: log, Resumed: resumed, SessionID: sessionID, Usage: usage}, fmt.Errorf("agent command failed: %w", err)
	}
	content, err := os.ReadFile(lastPath)
	if err != nil {
		return AgentResult{Log: log, Resumed: resumed, SessionID: sessionID, Usage: usage}, err
	}
	text := strings.TrimSpace(string(content))
	if text == "" {
		return AgentResult{Log: log, Resumed: resumed, SessionID: sessionID, Usage: usage}, fmt.Errorf("agent returned an empty response")
	}
	return AgentResult{
		Text:      text,
		SessionID: sessionID,
		Resumed:   resumed,
		Log:       log,
		Usage:     usage,
	}, nil
}

func (executor CodexExecutor) CheckForkSession(_ context.Context, sourceSessionID string) error {
	sessionFile, _, err := executor.codexSessionFileContent(sourceSessionID)
	if err != nil {
		return err
	}
	checkFile, err := os.CreateTemp(filepath.Dir(sessionFile), ".plasma-fork-check-*")
	if err != nil {
		return fmt.Errorf("Codex session directory is not writable: %w", err)
	}
	checkName := checkFile.Name()
	closeErr := checkFile.Close()
	removeErr := os.Remove(checkName)
	if closeErr != nil {
		return closeErr
	}
	if removeErr != nil {
		return removeErr
	}
	return nil
}

func ensureAgentWorkDir(workDir string) error {
	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		return nil
	}
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return fmt.Errorf("ensure agent workdir %q: %w", workDir, err)
	}
	return nil
}

func (executor CodexExecutor) ForkSession(_ context.Context, sourceSessionID string) (AgentSessionForkResult, error) {
	sessionFile, content, err := executor.codexSessionFileContent(sourceSessionID)
	if err != nil {
		return AgentSessionForkResult{}, err
	}
	sourceSessionID = strings.TrimSpace(sourceSessionID)
	cloneID, err := newUUID()
	if err != nil {
		return AgentSessionForkResult{}, err
	}
	cloneContent := bytes.ReplaceAll(content, []byte(sourceSessionID), []byte(cloneID))
	cloneFile := filepath.Join(filepath.Dir(sessionFile), fmt.Sprintf("rollout-%s-%s.jsonl", time.Now().UTC().Format("2006-01-02T15-04-05"), cloneID))
	if err := os.WriteFile(cloneFile, cloneContent, 0o600); err != nil {
		return AgentSessionForkResult{}, err
	}
	sourceInfo, err := os.Stat(sessionFile)
	if err != nil {
		return AgentSessionForkResult{}, err
	}
	cloneInfo, err := os.Stat(cloneFile)
	if err != nil {
		return AgentSessionForkResult{}, err
	}
	return AgentSessionForkResult{
		SessionID:       cloneID,
		SourceSessionID: sourceSessionID,
		SourceHash:      sha256Hex(content),
		CloneHash:       sha256Hex(cloneContent),
		SourceSizeBytes: sourceInfo.Size(),
		CloneSizeBytes:  cloneInfo.Size(),
	}, nil
}

func (executor CodexExecutor) codexSessionFileContent(sourceSessionID string) (string, []byte, error) {
	sourceSessionID = strings.TrimSpace(sourceSessionID)
	if sourceSessionID == "" {
		return "", nil, fmt.Errorf("source session id is required")
	}
	codexHome := codexHomeFromEnv(executor.Env)
	if codexHome == "" {
		return "", nil, fmt.Errorf("Codex home is required for Codex session fork")
	}
	sessionFile, err := findCodexSessionFile(codexHome, sourceSessionID)
	if err != nil {
		return "", nil, err
	}
	content, err := os.ReadFile(sessionFile)
	if err != nil {
		return "", nil, err
	}
	if !bytes.Contains(content, []byte(sourceSessionID)) {
		return "", nil, fmt.Errorf("source session id %q was not present in Codex session file", sourceSessionID)
	}
	return sessionFile, content, nil
}

func codexHomeFromEnv(explicit []string) string {
	home := ""
	for _, item := range explicit {
		if strings.HasPrefix(item, "CODEX_HOME=") {
			if codexHome := strings.TrimSpace(strings.TrimPrefix(item, "CODEX_HOME=")); codexHome != "" {
				return codexHome
			}
		}
		if strings.HasPrefix(item, "HOME=") {
			home = strings.TrimSpace(strings.TrimPrefix(item, "HOME="))
		}
	}
	if codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME")); codexHome != "" {
		return codexHome
	}
	if home == "" {
		home = strings.TrimSpace(os.Getenv("HOME"))
	}
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".codex")
}

func findCodexSessionFile(codexHome string, sessionID string) (string, error) {
	sessionsRoot := filepath.Join(codexHome, "sessions")
	var matches []string
	err := filepath.WalkDir(sessionsRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			return nil
		}
		if strings.Contains(entry.Name(), sessionID) {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("Codex session file not found for session %q under %s", sessionID, sessionsRoot)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("Codex session file is ambiguous for session %q", sessionID)
	}
	return matches[0], nil
}

func newUUID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		bytes[0:4],
		bytes[4:6],
		bytes[6:8],
		bytes[8:10],
		bytes[10:16],
	), nil
}

func resolveAgentCommand(command string) string {
	if strings.ContainsRune(command, os.PathSeparator) {
		return command
	}
	if resolved, err := exec.LookPath(command); err == nil {
		return resolved
	}
	for _, dir := range []string{"/opt/homebrew/bin", "/usr/local/bin"} {
		candidate := filepath.Join(dir, command)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate
		}
	}
	return command
}

func codexEnvironment(explicit []string) []string {
	if len(explicit) > 0 {
		return append([]string(nil), explicit...)
	}
	allowedKeys := []string{
		"HOME",
		"PATH",
		"SHELL",
		"TERM",
		"USER",
		"LOGNAME",
		"TMPDIR",
		"LANG",
		"LC_ALL",
		"LC_CTYPE",
		"CODEX_HOME",
		"PLASMA_RUNTIME_MODE",
		"XDG_CONFIG_HOME",
		"XDG_DATA_HOME",
		"XDG_CACHE_HOME",
	}
	env := make([]string, 0, len(allowedKeys)+1)
	seen := map[string]struct{}{}
	for _, key := range allowedKeys {
		value, ok := os.LookupEnv(key)
		if !ok {
			continue
		}
		env = append(env, key+"="+value)
		seen[key] = struct{}{}
	}
	if _, ok := seen["PATH"]; !ok {
		env = append(env, "PATH="+agentPATH(""))
	} else {
		for i, value := range env {
			if strings.HasPrefix(value, "PATH=") {
				env[i] = "PATH=" + agentPATH(strings.TrimPrefix(value, "PATH="))
				break
			}
		}
	}
	env = append(env, "PLASMA_AGENT=1")
	return env
}

func agentPATH(current string) string {
	values := []string{}
	addPath := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		for _, existing := range values {
			if existing == value {
				return
			}
		}
		values = append(values, value)
	}
	for _, value := range []string{"/opt/homebrew/bin", "/usr/local/bin"} {
		addPath(value)
	}
	for _, value := range filepath.SplitList(current) {
		addPath(value)
	}
	for _, value := range []string{"/usr/bin", "/bin", "/usr/sbin", "/sbin"} {
		addPath(value)
	}
	return strings.Join(values, string(os.PathListSeparator))
}

func codexSessionID(log string) string {
	return agentusage.ParseCodexSessionID(log)
}

func codexUsageFromLog(usage agentusage.AgentUsage, log string, previousSessionID string, sessionID string, resumed bool, compaction bool) agentusage.AgentUsage {
	if providerUsage, ok := agentusage.ParseCodexProviderUsage(log); ok {
		usage = usage.WithProviderUsage(providerUsage, "codex_jsonl_turn_completed")
	} else {
		usage = usage.WithUnavailable("codex JSONL did not include turn.completed usage")
	}
	return usage.WithSession(previousSessionID, sessionID, resumed, compaction)
}

func codexUsageExecutorName(executorName string) string {
	executorName = strings.TrimSpace(executorName)
	if executorName == "" {
		return "codex"
	}
	return executorName
}

func codexMCPConfigArgs(server CodexMCPServer, req AgentRequest) []string {
	if req.DisableTools {
		return nil
	}
	command := strings.TrimSpace(server.Command)
	if command == "" {
		return nil
	}
	server.Args = codexMCPArgsForRequest(server.Args, req)
	name := sanitizeMCPServerName(server.Name)
	args := []string{
		"-c", "mcp_servers." + name + ".command=" + tomlString(command),
		"-c", "mcp_servers." + name + ".args=" + tomlStringArray(server.Args),
		"-c", "mcp_servers." + name + ".enabled=true",
		"-c", "mcp_servers." + name + ".default_tools_approval_mode=" + tomlString("approve"),
	}
	if server.Required {
		args = append(args, "-c", "mcp_servers."+name+".required=true")
	}
	if server.StartupTimeoutSec > 0 {
		args = append(args, "-c", "mcp_servers."+name+".startup_timeout_sec="+strconv.Itoa(server.StartupTimeoutSec))
	}
	if server.ToolTimeoutSec > 0 {
		args = append(args, "-c", "mcp_servers."+name+".tool_timeout_sec="+strconv.Itoa(server.ToolTimeoutSec))
	}
	if enabledTools := effectiveMCPEnabledTools(server.EnabledTools, req.ExtraMCPTools, req.ReplaceMCPTools); len(enabledTools) > 0 {
		args = append(args, "-c", "mcp_servers."+name+".enabled_tools="+tomlStringArray(enabledTools))
	}
	return args
}

func codexMCPArgsForRequest(base []string, req AgentRequest) []string {
	args := append([]string(nil), base...)
	if req.ReplaceMCPTools {
		args = stripMCPEnabledToolArgs(args)
	}
	if hasReportPatchTool(req.ExtraMCPTools) || req.ReportPatch != nil {
		args = append(args, "-report-patch")
	}
	if req.ReportPlan != nil {
		args = appendReportPlanMCPArgs(args, req.ToolSessionID, *req.ReportPlan)
	}
	if req.PartAssembly != nil {
		if encoded, err := json.Marshal(req.PartAssembly); err == nil {
			args = append(args, "-report-part-assembly-binding-json", string(encoded))
		}
	}
	if req.LongFormFinalize != nil {
		if encoded, err := json.Marshal(req.LongFormFinalize); err == nil {
			args = append(args, "-report-long-form-finalize-binding-json", string(encoded))
		}
	}
	if req.ReportPatch != nil {
		args = appendReportPatchMCPArgs(args, *req.ReportPatch)
	}
	for _, tool := range req.ExtraMCPTools {
		if tool = strings.TrimSpace(tool); tool != "" {
			args = append(args, "-enabled-tool", tool)
		}
	}
	if missionID := strings.TrimSpace(req.MissionID); missionID != "" {
		args = append(args, "-mission-id", missionID)
	}
	if toolSessionID := strings.TrimSpace(req.ToolSessionID); toolSessionID != "" {
		args = append(args, "-agent-session-id", toolSessionID)
	}
	if userEventID := strings.TrimSpace(req.UserEventID); userEventID != "" {
		args = append(args, "-current-user-event-id", userEventID)
	}
	if agentExecutor := strings.TrimSpace(strings.ToLower(req.AgentExecutor)); agentExecutor != "" {
		args = append(args, "-agent-executor", agentExecutor)
	}
	return args
}

func stripMCPEnabledToolArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		if args[index] == "-enabled-tool" {
			index++
			continue
		}
		out = append(out, args[index])
	}
	return out
}

func appendReportPatchMCPArgs(args []string, patch AgentReportPatchContext) []string {
	values := []struct {
		flag  string
		value string
	}{
		{"-report-patch-base-artifact-id", patch.BaseArtifactID},
		{"-report-patch-pending-event-id", patch.PendingEventID},
		{"-report-patch-agent-executor", patch.AgentExecutor},
		{"-report-patch-agent-model", patch.AgentModel},
		{"-report-patch-agent-reasoning-effort", patch.AgentReasoningEffort},
		{"-report-patch-mcp-mode", patch.MCPMode},
		{"-report-patch-agent-session-id", patch.AgentSessionID},
		{"-report-patch-previous-agent-session-id", patch.PreviousAgentSessionID},
		{"-report-patch-returned-agent-session-id", patch.ReturnedAgentSessionID},
		{"-report-patch-report-session-id", patch.ReportSessionID},
		{"-report-patch-fork-source-agent-session-id", patch.ForkSourceAgentSessionID},
		{"-report-patch-report-session-policy", patch.ReportSessionPolicy},
		{"-report-patch-report-session-policy-selection", patch.ReportSessionPolicySelection},
		{"-report-patch-session-chain-kind", patch.SessionChainKind},
	}
	for _, item := range values {
		if value := strings.TrimSpace(item.value); value != "" {
			args = append(args, item.flag, value)
		}
	}
	return args
}

func appendReportPlanMCPArgs(args []string, toolSessionID string, plan AgentReportPlanContext) []string {
	values := []struct{ flag, value string }{
		{"-report-plan-pending-event-id", plan.PendingEventID}, {"-report-plan-mode", plan.ReportMode},
		{"-report-plan-idempotency-key", plan.IdempotencyKey}, {"-report-plan-tool-session-id", toolSessionID},
		{"-report-plan-previous-provider-session-id", plan.PreviousProviderSessionID},
		{"-report-plan-agent-model", plan.AgentModel}, {"-report-plan-agent-reasoning-effort", plan.AgentReasoningEffort},
	}
	for _, item := range values {
		if value := strings.TrimSpace(item.value); value != "" {
			args = append(args, item.flag, value)
		}
	}
	return args
}

func effectiveMCPEnabledTools(base []string, extra []string, replace bool) []string {
	seen := map[string]struct{}{}
	out := []string{}
	tools := append([]string(nil), extra...)
	if !replace {
		tools = append(append([]string{}, base...), extra...)
	}
	for _, tool := range tools {
		tool = strings.TrimSpace(tool)
		if tool == "" {
			continue
		}
		if _, ok := seen[tool]; ok {
			continue
		}
		seen[tool] = struct{}{}
		out = append(out, tool)
	}
	return out
}

func hasReportPatchTool(tools []string) bool {
	for _, tool := range tools {
		if strings.HasPrefix(strings.TrimSpace(tool), "plasma.report.patch.") {
			return true
		}
	}
	return false
}

func sanitizeMCPServerName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "plasma"
	}
	if regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(name) {
		return name
	}
	return "plasma"
}

func tomlString(value string) string {
	return strconv.Quote(value)
}

func tomlStringArray(values []string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, tomlString(value))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func agentPrompt(userText string, recall recallPreview, mcpMode string, resumed bool, toolSessionID string, controller controllerStrategyDecision) string {
	intro := "You are the Plasma research agent."
	if resumed {
		intro = "Continue the existing Plasma research agent session."
	}
	toolPolicy := "Use Plasma read tools when the user's request would benefit from mission ledger inspection or source inspection. Start with plasma.research.outline, narrow with plasma.research.list or plasma.research.grep, confirm source or result content with plasma.research.read, plasma.sources.read, plasma.sources.tree, plasma.sources.grep, and use plasma.research.references when you need relationships between sources, observations, results, and report artifacts. If a read is truncated, continue with next_offset when more content is relevant. Sources may be snapshot_only pinned artifacts or live_reference local_path sources; live reads create source.observed events. PDF sources are original documents; read them through Plasma tools, which return extracted text and metadata rather than raw PDF bytes."
	if mcpMode == "auto" {
		toolPolicy = "For research, investigation, comparison, purchase, or recommendation requests, actively investigate within the mission. First call plasma.research.outline, then inspect mission material with plasma.research.list, plasma.research.grep, plasma.research.read, plasma.sources.read, plasma.sources.tree, plasma.sources.grep, and plasma.research.references. If a read is truncated, continue with next_offset when more content is relevant; do not treat the first chunk of a long source as the whole source. Sources may be snapshot_only pinned artifacts, PDF documents, or live_reference local_path sources. PDF reads return extracted text and metadata, not raw PDF bytes. For live_reference local_path material, use plasma.sources.read, plasma.sources.tree, or plasma.sources.grep to create source.observed metadata and cite the observation_event_id, observed_at, relative_path, sha256, and git metadata when those details support the answer. If more original materials are useful, search mounted source connectors such as Liquid2 or Confluence with plasma.sources.search without asking for separate pre-approval. If a connector fails, is unavailable, or returns insufficient material, treat that as a route failure, report it briefly, and continue with another available route such as web search when the provider offers it. When you find new original material worth user review, call plasma.sources.candidates.propose with the URL, the source title from search results when available, and a concrete acceptance opinion. When proposing a plasma.sources.search result, copy source_uri into url and title into title, especially for Confluence pages. If that tool is unavailable, include the candidate in the visible answer with exactly this two-line shape:\n소스 후보: https://example.com/original-material\n채택 의견: why this original material should be reviewed and possibly attached as a source. Source candidates are not sources and are not saved source snapshots; Plasma only records them for user review. Do not create evidence, claims, confidence updates, or proposal bundles in the default C1 loop."
	}
	toolPolicy += " Before showing Mermaid diagrams to the user, call plasma.mermaid.validate and revise the source if ok is false. Treat ok true as a static preflight pass, not as a full browser-render guarantee."
	return fmt.Sprintf(`%s

Answer the user's latest turn directly and use Korean unless the user asks otherwise.
Do not modify files, run project commands, create commits, or treat your own answer as a source.
Your answer is a result, not a source. Plasma stores it as a conversation result unless the user later creates a report artifact.
Plasma source policy: source content is stored in Plasma, not pasted into this prompt. Do not claim that you inspected stored source content unless you actually accessed it through an available tool in this turn.
Local path policy: do not paste local file content into prompts. Read live_reference local_path sources through Plasma tools and treat source.observed events as the cited observation of mutable source state.
Tool policy: %s
Plasma tool binding: use mission_id %s. If a read tool requires session_id or producer, use session_id %s and producer {"type":"agent_session","id":"%s"}.
C1 boundary: do not call evidence, claim, confidence, or proposal mutation tools in the default loop. You may call plasma.sources.candidates.propose to create review-only source candidates; this does not create source snapshots or saved knowledge. If your answer includes the exact 소스 후보 / 채택 의견 lines above, Plasma may also record those links as review candidates only. Links or claims in your answer remain part of the result; they do not become sources or saved knowledge automatically.

Mission reminder:
%s

%s

Latest user turn:
%s
`, intro, toolPolicy, strings.TrimSpace(recall.Mission.MissionID), strings.TrimSpace(toolSessionID), strings.TrimSpace(toolSessionID), missionReminder(recall), controllerStrategyPromptBlock(controller), userText)
}

func agentCompactPrompt(recall recallPreview) string {
	return fmt.Sprintf(`You are continuing the same Plasma agent session.

Compact the useful session context for future turns. Do not answer the user's research question in this turn.
Keep only mission-critical information: current objective, user steering decisions, unresolved questions, constraints, and next actions.
Do not modify files, run project commands, create commits, or treat your own summary as a source.
Do not ask Plasma to paste source bodies into this prompt.

Mission reminder:
%s
`, missionReminder(recall))
}

func agentProposalPrompt(recall recallPreview, answerText string, toolSessionID string) string {
	return fmt.Sprintf(`You are continuing the same Plasma research agent session.

Create review proposals for the latest answer. Do not answer the user in this turn.
Use Korean for proposal summaries.
Use available Plasma tools only.
Use mission_id %s. For proposal tools, use session_id %s and producer {"type":"agent_session","id":"%s"}.

Workflow:
1. Inspect the mission ledger with plasma.research.outline, plasma.research.list, plasma.research.grep, plasma.research.read, plasma.sources.read, plasma.sources.tree, plasma.sources.grep, and plasma.research.references when source-backed facts may exist. If a read is truncated, continue with next_offset when more source or record content is relevant. If this turn already created some evidence or claim proposals, add missing non-duplicate proposals instead of stopping.
2. Build a useful evidence slate, not only a minimal proof set. Propose focused evidence records for distinct source-backed facts, direct quotes/statistics/table rows, interpretations/evaluations, reactions, rumors or unconfirmed circulating claims, controversies, market signals, code/API usage, formulas, benchmarks, and open questions when they would help future review or reporting.
3. Use the most specific evidence_type and honest confidence/risk language. Weak signals are useful when clearly labeled; do not upgrade them to facts.
4. As a default, aim for several focused evidence proposals for a source-backed research answer when the material supports it; fewer is fine when sources are thin, repetitive, or not actually inspected. Do not invent evidence and do not split duplicates just to increase count.
5. For each main conclusion or recommendation, call plasma.claims.propose backed by the proposed evidence ids.
6. Leave all records proposed/pending for user review. Do not approve anything.
7. Do not call plasma.proposals.submit for records you just created with plasma.evidence.propose, plasma.claims.propose, or plasma.questions.propose; those tools already submit review proposals.
8. If there is no source-backed content to propose, reply exactly: NO_PROPOSALS.

Generate unique ids with the required prefixes: evd_, clm_, prp_, evt_. Use stable idempotency_key values for this proposal extraction turn.

Mission reminder:
%s

Latest answer to convert into proposals:
%s
`, strings.TrimSpace(recall.Mission.MissionID), strings.TrimSpace(toolSessionID), strings.TrimSpace(toolSessionID), missionReminder(recall), strings.TrimSpace(answerText))
}

func missionReminder(recall recallPreview) string {
	lines := []string{
		"- mission_id: " + strings.TrimSpace(recall.Mission.MissionID),
		"- title: " + strings.TrimSpace(recall.Mission.Title),
	}
	if objective := strings.TrimSpace(recall.Mission.Objective); objective != "" {
		lines = append(lines, "- objective: "+objective)
	}
	if included := cleanList(recall.Mission.Scope.Included); len(included) > 0 {
		lines = append(lines, "- included scope: "+strings.Join(included, "; "))
	}
	if excluded := cleanList(recall.Mission.Scope.Excluded); len(excluded) > 0 {
		lines = append(lines, "- excluded scope: "+strings.Join(excluded, "; "))
	}
	if len(recall.OpenQuestionIDs) > 0 {
		lines = append(lines, fmt.Sprintf("- open questions: %d", len(recall.OpenQuestionIDs)))
	}
	lines = append(lines, "- source discovery: allowed when useful; accepted sources still require user review")
	return strings.Join(lines, "\n")
}

func cleanList(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func agentWorkDir(defaultDir string) string {
	if strings.TrimSpace(defaultDir) != "" {
		return defaultDir
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return filepath.Clean(wd)
}
