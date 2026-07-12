package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/config"
	"github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
	"github.com/c86j224s/liquid2/plasma/internal/web"
	workflowruntime "github.com/c86j224s/liquid2/plasma/internal/workflow"
)

func TestCLIReportDirectionPromptAllowlist(t *testing.T) {
	const hint = "CLI_DIRECTION_SENTINEL"
	for name, prompt := range map[string]string{
		"plan":   cliPromptWithDirection(cliReportPlanPrompt("t", "mis_1", "ses_1"), hint),
		"writer": cliPromptWithDirection(cliReportPrompt("t", "mis_1", "ses_1", "planned", "evt_1", ""), hint),
	} {
		if !strings.Contains(prompt, hint) || !strings.Contains(prompt, "weak editorial axis") {
			t.Fatalf("%s prompt missing direction: %q", name, prompt)
		}
	}
	if strings.Contains(cliReportPrompt("t", "mis_1", "ses_1", "one_take", "", ""), hint) {
		t.Fatal("no-hint report inherited direction")
	}
}

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "plasma-cmd-test-*")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("HOME", dir)
	_ = os.Setenv(config.RuntimeModeEnv, "")
	if err := os.Chdir(dir); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestResolveServeStaticDirStrictValues(t *testing.T) {
	expected := filepath.Join("internal", "web", "static")
	for _, value := range []string{"auto", " Auto "} {
		if got, err := resolveServeStaticDir(value); err != nil || got != expected {
			t.Fatalf("resolveServeStaticDir(%q) = %q, want %q", value, got, expected)
		}
	}
	if got, err := resolveServeStaticDir(""); err != nil || got != "" {
		t.Fatalf("expected empty static dir to use embedded assets, got %q error %v", got, err)
	}
	if got, err := resolveServeStaticDir("/tmp/static"); err != nil || got != "/tmp/static" {
		t.Fatalf("expected explicit static dir to pass through, got %q", got)
	}
	for _, value := range []string{"1", "on", "default", "off", "none", "embedded"} {
		if got, err := resolveServeStaticDir(value); err == nil {
			t.Fatalf("expected %q to fail, got %q", value, got)
		}
	}
}

func TestRunVersion(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run returned %d, stderr %q", code, errOut.String())
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Fatal("expected version output")
	}
}

func TestRunStatusReportsResolvedServeConfig(t *testing.T) {
	t.Setenv(config.RuntimeModeEnv, "dev")
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"status"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run returned %d, stderr %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Plasma development") ||
		!strings.Contains(out.String(), "URL     http://127.0.0.1:6002") ||
		!strings.Contains(out.String(), "Mode    dev") {
		t.Fatalf("unexpected status output %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"status", "-url"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run returned %d, stderr %q", code, errOut.String())
	}
	if strings.TrimSpace(out.String()) != "http://127.0.0.1:6002" {
		t.Fatalf("unexpected status url %q", out.String())
	}
}

func TestRunHealthCreatesSeparatePlasmaDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"health", "-db", dbPath}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run returned %d, stderr %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "plasma ok") {
		t.Fatalf("expected ok health output, got %q", out.String())
	}
	if !strings.Contains(out.String(), "db="+dbPath) {
		t.Fatalf("expected db path in output, got %q", out.String())
	}
}

func TestRunMCPListsPlasmaResearchTools(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"test"}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
	}, "\n")
	var out, errOut bytes.Buffer
	code := runMCP(context.Background(), []string{"-db", dbPath, "-mission-id", "mis_1", "-agent-session-id", "ses_1", "-agent-executor", "codex"}, strings.NewReader(input), &out, &errOut)
	if code != 0 {
		t.Fatalf("runMCP returned %d, stderr %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"plasma.research.outline"`) ||
		!strings.Contains(out.String(), `"plasma.research.references"`) ||
		!strings.Contains(out.String(), `"plasma.sources.list"`) ||
		!strings.Contains(out.String(), `"plasma.sources.read"`) ||
		!strings.Contains(out.String(), `"plasma.sources.search"`) ||
		!strings.Contains(out.String(), `"plasma.sources.candidates.propose"`) ||
		!strings.Contains(out.String(), `"inputSchema"`) {
		t.Fatalf("expected MCP tool list output, got %q", out.String())
	}
	if strings.Contains(out.String(), `"plasma.evidence.propose"`) ||
		strings.Contains(out.String(), `"plasma.claims.propose"`) ||
		strings.Contains(out.String(), `"plasma.claims.confidence.update"`) ||
		strings.Contains(out.String(), `"plasma.local_path.attach"`) ||
		strings.Contains(out.String(), `"plasma.experiment.report.create"`) {
		t.Fatalf("default MCP tool list must not include mutation tools, got %q", out.String())
	}
}

func TestRunMCPLegacyResearchLoopFlagExposesLegacyTools(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
	}, "\n")
	var out, errOut bytes.Buffer
	code := runMCP(context.Background(), []string{"-db", dbPath, "-mission-id", "mis_1", "-agent-session-id", "ses_1", "-agent-executor", "codex", "-legacy-research-loop"}, strings.NewReader(input), &out, &errOut)
	if code != 0 {
		t.Fatalf("runMCP returned %d, stderr %q", code, errOut.String())
	}
	for _, expected := range []string{`"plasma.evidence.propose"`, `"plasma.claims.propose"`, `"plasma.claims.confidence.update"`} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("expected legacy MCP tool %s, got %q", expected, out.String())
		}
	}
}

func TestRunMCPExperimentalReportCompositionFlagExposesOnlyExperimentToolsWhenRequested(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
	}, "\n")
	var out, errOut bytes.Buffer
	code := runMCP(context.Background(), []string{"-db", dbPath, "-mission-id", "mis_1", "-agent-session-id", "ses_1", "-agent-executor", "codex", "-experimental-report-composition"}, strings.NewReader(input), &out, &errOut)
	if code != 0 {
		t.Fatalf("runMCP returned %d, stderr %q", code, errOut.String())
	}
	for _, expected := range []string{`"plasma.experiment.report.create"`, `"plasma.experiment.report.append"`, `"plasma.experiment.report.read"`, `"plasma.experiment.report.finalize"`} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("expected experimental MCP tool %s, got %q", expected, out.String())
		}
	}
}

func TestRunMCPRequiresMissionAndAgentSessionBinding(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	var out, errOut bytes.Buffer
	code := runMCP(context.Background(), []string{"-db", dbPath}, strings.NewReader(""), &out, &errOut)
	if code != 2 {
		t.Fatalf("runMCP returned %d, stderr %q", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "mission-id") {
		t.Fatalf("expected binding error, got %q", errOut.String())
	}
}

func TestRunServeCodexRequiresFileBackedDB(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"serve", "-agent", "codex", "-db", ":memory:"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("run returned %d, stdout %q stderr %q", code, out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "file-backed Plasma database") {
		t.Fatalf("expected file-backed database error, got %q", errOut.String())
	}
}

func TestRunServeConfigArgsOnlyIncludeExplicitFlags(t *testing.T) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", "", "")
	agent := fs.String("agent", "", "")
	codexCommand := fs.String("codex-command", "", "")
	timeout := fs.Duration("agent-timeout", 0, "")
	roots := repeatedStringFlag{}
	fs.Var(&roots, "local-source-root", "")
	if err := fs.Parse([]string{"-agent", "codex", "-local-source-root", "workspace=/tmp/work"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	args := config.Args{
		Addr:             stringFlagArg(fs, "addr", *addr),
		Agent:            stringFlagArg(fs, "agent", *agent),
		CodexCommand:     stringFlagArg(fs, "codex-command", *codexCommand),
		AgentTimeout:     durationFlagArg(fs, "agent-timeout", *timeout),
		LocalSourceRoots: listFlagArg(fs, "local-source-root", []string(roots)),
	}
	if args.Addr != "" || args.CodexCommand != "" || args.AgentTimeout != "" {
		t.Fatalf("unspecified serve defaults must not override config: %#v", args)
	}
	if args.Agent != "codex" {
		t.Fatalf("expected explicit agent flag, got %#v", args)
	}
	if len(args.LocalSourceRoots) != 1 || args.LocalSourceRoots[0] != "workspace=/tmp/work" {
		t.Fatalf("expected explicit local root, got %#v", args.LocalSourceRoots)
	}
	cfg := config.Config{}
	applyServeDefaults(&cfg)
	if cfg.Addr != "127.0.0.1:3002" || cfg.Agent != "codex" {
		t.Fatalf("unexpected serve defaults: %#v", cfg)
	}
}

func TestApplyServeDefaultsUsesRuntimeMode(t *testing.T) {
	t.Setenv(config.RuntimeModeEnv, "dev")
	cfg := config.Config{}
	applyServeDefaults(&cfg)
	if cfg.Addr != "127.0.0.1:6002" ||
		cfg.DBPath != filepath.Join(os.Getenv("HOME"), "research-artifacts", "liquid2", "plasma", "runtime", "dev-6002", "plasma-ui-user.db") ||
		cfg.Liquid2URL != "http://127.0.0.1:6011" ||
		cfg.Agent != "codex" ||
		cfg.AgentWorkDir != filepath.Join(os.TempDir(), "plasma-agent-workdir") ||
		cfg.WorkflowGoalReasoningEffort != "low" {
		t.Fatalf("unexpected dev serve defaults: %#v", cfg)
	}

	t.Setenv(config.RuntimeModeEnv, "release")
	cfg = config.Config{}
	applyServeDefaults(&cfg)
	if cfg.Addr != "127.0.0.1:3002" ||
		cfg.DBPath != filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Plasma", "plasma.db") ||
		cfg.Liquid2URL != "http://127.0.0.1:3011" ||
		cfg.Agent != "codex" ||
		cfg.AgentWorkDir != filepath.Join(os.TempDir(), "plasma-release-agent-workdir") ||
		cfg.WorkflowGoalReasoningEffort != "low" {
		t.Fatalf("unexpected release serve defaults: %#v", cfg)
	}
}

func TestAgentTimeoutZeroFlagOverridesConfigTimeout(t *testing.T) {
	t.Setenv(config.RuntimeModeEnv, "dev")
	if err := os.WriteFile("config.toml", []byte("[plasma-agents]\ntimeout = \"5m\"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	defer os.Remove("config.toml")

	fs := flag.NewFlagSet("agent timeout", flag.ContinueOnError)
	timeoutFlag := fs.Duration("agent-timeout", 0, "")
	if err := fs.Parse([]string{"-agent-timeout", "0"}); err != nil {
		t.Fatalf("parse explicit zero timeout: %v", err)
	}
	cfg, timeout, err := loadAgentConfig(config.Args{
		AgentTimeout: durationFlagArg(fs, "agent-timeout", *timeoutFlag),
	})
	if err != nil {
		t.Fatalf("load explicit zero timeout config: %v", err)
	}
	if cfg.AgentTimeout != "0s" || timeout != 0 {
		t.Fatalf("expected explicit zero timeout to override config, cfg=%#v timeout=%v", cfg, timeout)
	}

	defaultFS := flag.NewFlagSet("agent timeout default", flag.ContinueOnError)
	defaultTimeoutFlag := defaultFS.Duration("agent-timeout", 0, "")
	if err := defaultFS.Parse(nil); err != nil {
		t.Fatalf("parse default timeout: %v", err)
	}
	cfg, timeout, err = loadAgentConfig(config.Args{
		AgentTimeout: durationFlagArg(defaultFS, "agent-timeout", *defaultTimeoutFlag),
	})
	if err != nil {
		t.Fatalf("load default timeout config: %v", err)
	}
	if cfg.AgentTimeout != "5m" || timeout != 5*time.Minute {
		t.Fatalf("expected unspecified timeout to preserve config, cfg=%#v timeout=%v", cfg, timeout)
	}
}

func TestRunMissionCommandsCreateListShow(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"missions", "create", "-db", dbPath, "-title", "CLI mission", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("missions create returned %d stderr=%q", code, errOut.String())
	}
	created := decodeCLIJSON(t, out.String())
	missionID := nestedCLIString(t, created, "projection", "mission_id")
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"missions", "list", "-db", dbPath}, &out, &errOut)
	if code != 0 || !strings.Contains(out.String(), missionID) {
		t.Fatalf("missions list returned %d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"missions", "show", missionID, "-db", dbPath, "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("missions show returned %d stderr=%q", code, errOut.String())
	}
	shown := decodeCLIJSON(t, out.String())
	if got := nestedCLIString(t, shown, "projection", "mission_id"); got != missionID {
		t.Fatalf("expected mission %q, got %q", missionID, got)
	}
}

func TestRunMissionUpdatePreservesOmittedFieldsAndReplacesScope(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"missions", "create", "-db", dbPath, "-title", "Original", "-objective", "Keep", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("missions create returned %d stderr=%q", code, errOut.String())
	}
	missionID := nestedCLIString(t, decodeCLIJSON(t, out.String()), "projection", "mission_id")
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{
		"missions", "update", missionID, "-db", dbPath, "-title", " Updated ",
		"-scope-included", " A ", "-scope-included", "B", "-scope-excluded", " X ", "-json",
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("missions update returned %d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	updated := decodeCLIJSON(t, out.String())
	if got := nestedCLIString(t, updated, "projection", "title"); got != "Updated" {
		t.Fatalf("updated title = %q", got)
	}
	if got := nestedCLIString(t, updated, "projection", "objective"); got != "Keep" {
		t.Fatalf("omitted objective changed to %q", got)
	}
	projection := updated["projection"].(map[string]any)
	scope := projection["scope"].(map[string]any)
	if got := fmt.Sprint(scope["included"]); got != "[A B]" {
		t.Fatalf("included scope = %s", got)
	}
	if got := fmt.Sprint(scope["excluded"]); got != "[X]" {
		t.Fatalf("excluded scope = %s", got)
	}

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"missions", "update", missionID, "-db", dbPath, "-clear-scope"}, &out, &errOut)
	if code != 0 || !strings.Contains(out.String(), "updated mission "+missionID) {
		t.Fatalf("clear scope returned %d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"missions", "update", missionID, "-db", dbPath, "-clear-scope", "-scope-included", "A"}, &out, &errOut)
	if code != 2 || !strings.Contains(errOut.String(), "cannot be combined") {
		t.Fatalf("conflicting scope flags returned %d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
}

func TestRunSourcesRootsUsesEnv(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	rootDir := t.TempDir()
	t.Setenv(config.LocalSourceRootsEnv, "workspace="+rootDir)
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"sources", "roots", "-db", dbPath, "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("sources roots returned %d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"root_id": "workspace"`) {
		t.Fatalf("expected workspace root, got %q", out.String())
	}
	assertCLINoRootPath(t, rootDir, out.String(), errOut.String())
}

func TestRunSourcesUploadCreatesReadableSource(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	missionID := createCLITestMission(t, dbPath)
	sourcePath := filepath.Join(t.TempDir(), "upload.md")
	if err := os.WriteFile(sourcePath, []byte("# CLI Upload\n\nBody."), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"sources", "upload", missionID, sourcePath, "-db", dbPath, "-title", "CLI Upload", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("sources upload returned %d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	uploaded := decodeCLIJSON(t, out.String())
	if got := nestedCLIString(t, uploaded, "snapshot", "Connector", "ConnectorType"); got != app.SourceConnectorTypeFileUpload {
		t.Fatalf("expected file_upload connector, got %q", got)
	}
	artifact, ok := uploaded["artifact"].(map[string]any)
	if !ok {
		t.Fatalf("expected artifact object in upload response, got %#v", uploaded["artifact"])
	}
	if _, leaked := artifact["Content"]; leaked {
		t.Fatalf("upload JSON response must not include artifact content: %#v", artifact)
	}
	snapshotID := nestedCLIString(t, uploaded, "snapshot", "SnapshotID")
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"sources", "read", missionID, snapshotID, "-db", dbPath, "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("sources read returned %d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	if got := nestedCLIString(t, decodeCLIJSON(t, out.String()), "content"); got != "# CLI Upload\n\nBody." {
		t.Fatalf("expected uploaded source content, got %q", got)
	}
}

func TestRunSourcesUploadRejectsOversizeBeforeRead(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	missionID := createCLITestMission(t, dbPath)
	sourcePath := filepath.Join(t.TempDir(), "too-large.txt")
	file, err := os.Create(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(app.UploadedFileMaxBytes + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"sources", "upload", missionID, sourcePath, "-db", dbPath}, &out, &errOut)
	if code != 2 || !strings.Contains(errOut.String(), "100 MiB") {
		t.Fatalf("expected oversize rejection, code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
}

func TestRunSourcesLocalPathWorkflow(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	missionID := createCLITestMission(t, dbPath)
	rootDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(rootDir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "docs", "notes.txt"), []byte("alpha beta\nsecond beta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, ".env"), []byte("SECRET=value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rootSpec := "workspace=" + rootDir

	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"sources", "roots", "-db", dbPath, "-local-source-root", rootSpec, "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("sources roots returned %d stderr=%q", code, errOut.String())
	}
	assertCLINoRootPath(t, rootDir, out.String(), errOut.String())

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"sources", "tree", missionID, "-db", dbPath, "-local-source-root", rootSpec, "-root", "workspace", "-path", "docs", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("sources tree returned %d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"relative_path": "docs/notes.txt"`) {
		t.Fatalf("expected notes.txt tree entry, got %q", out.String())
	}
	assertCLINoRootPath(t, rootDir, out.String(), errOut.String())

	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "absolute path", args: []string{"sources", "tree", missionID, "-db", dbPath, "-local-source-root", rootSpec, "-root", "workspace", "-path", rootDir, "-json"}},
		{name: "unknown root", args: []string{"sources", "tree", missionID, "-db", dbPath, "-local-source-root", rootSpec, "-root", "missing", "-path", "docs", "-json"}},
		{name: "denied path", args: []string{"sources", "attach-local", missionID, "-db", dbPath, "-local-source-root", rootSpec, "-root", "workspace", "-path", ".env", "-json"}},
	} {
		out.Reset()
		errOut.Reset()
		code = run(context.Background(), tc.args, &out, &errOut)
		if code == 0 {
			t.Fatalf("%s should fail, stdout=%q", tc.name, out.String())
		}
		assertCLINoRootPath(t, rootDir, out.String(), errOut.String())
	}

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"sources", "attach-local", missionID, "-db", dbPath, "-local-source-root", rootSpec, "-root", "workspace", "-path", "docs/notes.txt", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("sources attach-local returned %d stderr=%q", code, errOut.String())
	}
	attached := decodeCLIJSON(t, out.String())
	sourceID := nestedCLIString(t, attached, "snapshot", "SnapshotID")
	if got := nestedCLIString(t, attached, "snapshot", "Access", "RetrievalPolicy"); got != app.SourceRetrievalPolicyLiveReference {
		t.Fatalf("expected live reference, got %q", got)
	}
	assertCLINoRootPath(t, rootDir, out.String(), errOut.String())

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"sources", "read", missionID, sourceID, "-db", dbPath, "-local-source-root", rootSpec, "-max-bytes", "5", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("sources read returned %d stderr=%q", code, errOut.String())
	}
	read := decodeCLIJSON(t, out.String())
	if got := nestedCLIString(t, read, "content"); got != "alpha" {
		t.Fatalf("expected bounded content %q, got %q", "alpha", got)
	}
	if nestedCLIString(t, read, "observation_event_id") == "" {
		t.Fatalf("expected observation event id in read response")
	}
	assertCLINoRootPath(t, rootDir, out.String(), errOut.String())

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"sources", "grep", missionID, sourceID, "-db", dbPath, "-local-source-root", rootSpec, "-query", "second", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("sources grep returned %d stderr=%q", code, errOut.String())
	}
	grep := decodeCLIJSON(t, out.String())
	if nestedCLIString(t, grep, "observation_event_id") == "" || !strings.Contains(out.String(), `"relative_path": "docs/notes.txt"`) {
		t.Fatalf("expected grep observation and relative match, got %q", out.String())
	}
	assertCLINoRootPath(t, rootDir, out.String(), errOut.String())

	store, err := sqlite.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	svc := app.NewService(store)
	artifact, err := svc.CreateRawArtifact(context.Background(), app.CreateRawArtifactRequest{
		ArtifactID: "art_cli_pinned",
		MissionID:  missionID,
		MediaType:  "text/plain; charset=utf-8",
		Filename:   "pinned.txt",
		Producer:   app.Producer{Type: "user", ID: "test"},
		Content:    []byte("pinned body"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateSourceSnapshot(context.Background(), app.CreateSourceSnapshotRequest{
		SnapshotID: "src_cli_pinned",
		MissionID:  missionID,
		Connector: app.ConnectorRef{
			ConnectorID:      "manual",
			ConnectorType:    "manual",
			ExternalSourceID: "pinned.txt",
		},
		Title:       "Pinned",
		ArtifactIDs: []string{artifact.ArtifactID},
		ContentHash: app.ContentHash{Algorithm: "sha256", Value: artifact.SHA256},
		Access:      app.SourceAccess{RetrievalPolicy: app.SourceRetrievalPolicySnapshotOnly},
	}); err != nil {
		t.Fatal(err)
	}
	events, err := svc.ListEvents(context.Background(), missionID)
	if closeErr := store.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if err != nil {
		t.Fatal(err)
	}
	if got := cliCountEventType(events, app.SourceObservedEvent); got != 2 {
		t.Fatalf("expected live read+grep observation events, got %d", got)
	}

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"sources", "read", missionID, "src_cli_pinned", "-db", dbPath, "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("pinned sources read returned %d stderr=%q", code, errOut.String())
	}
	if got := nestedCLIString(t, decodeCLIJSON(t, out.String()), "content"); got != "pinned body" {
		t.Fatalf("expected pinned content, got %q", got)
	}
	store, err = sqlite.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	events, err = app.NewService(store).ListEvents(context.Background(), missionID)
	if closeErr := store.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if err != nil {
		t.Fatal(err)
	}
	if got := cliCountEventType(events, app.SourceObservedEvent); got != 2 {
		t.Fatalf("pinned read should not create observation events, got %d", got)
	}

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"sources", "remove", missionID, sourceID, "-db", dbPath, "-reason", "wrong file", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("sources remove returned %d stderr=%q", code, errOut.String())
	}
	assertCLINoRootPath(t, rootDir, out.String(), errOut.String())

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"sources", "list", missionID, "-db", dbPath, "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("sources list returned %d stderr=%q", code, errOut.String())
	}
	if strings.Contains(out.String(), sourceID) {
		t.Fatalf("default source list should hide removed source, got %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"sources", "list", missionID, "-db", dbPath, "-include-removed", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("sources list --include-removed returned %d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), sourceID) || !strings.Contains(out.String(), `"removed": true`) {
		t.Fatalf("include removed should show removed source, got %q", out.String())
	}

	store, err = sqlite.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	svc = app.NewService(store)
	artifact, err = svc.CreateRawArtifact(context.Background(), app.CreateRawArtifactRequest{
		ArtifactID: "art_cli_pinned_new",
		MissionID:  missionID,
		MediaType:  "text/plain; charset=utf-8",
		Filename:   "pinned-new.txt",
		Producer:   app.Producer{Type: "user", ID: "test"},
		Content:    []byte("new pinned body"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateSourceSnapshot(context.Background(), app.CreateSourceSnapshotRequest{
		SnapshotID: "src_cli_pinned_new",
		MissionID:  missionID,
		Connector: app.ConnectorRef{
			ConnectorID:      "manual",
			ConnectorType:    "manual",
			ExternalSourceID: "pinned-new.txt",
		},
		Title:       "Pinned new",
		ArtifactIDs: []string{artifact.ArtifactID},
		ContentHash: app.ContentHash{Algorithm: "sha256", Value: artifact.SHA256},
		Access:      app.SourceAccess{RetrievalPolicy: app.SourceRetrievalPolicySnapshotOnly},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(context.Background(), app.AppendEventRequest{
		EventID:   "evt_cli_source_superseded",
		MissionID: missionID,
		EventType: app.ConfluenceUpdatedEvent,
		Producer:  app.Producer{Type: "user", ID: "test"},
		Payload: cliMustJSON(t, map[string]any{
			"old_snapshot_id": "src_cli_pinned",
			"new_snapshot_id": "src_cli_pinned_new",
		}),
	}); err != nil {
		t.Fatal(err)
	}
	if closeErr := store.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"sources", "list", missionID, "-db", dbPath, "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("sources list after supersede returned %d stderr=%q", code, errOut.String())
	}
	if strings.Contains(out.String(), "src_cli_pinned\"") || !strings.Contains(out.String(), "src_cli_pinned_new") {
		t.Fatalf("default source list should hide superseded source and show current source, got %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"sources", "list", missionID, "-db", dbPath, "-include-superseded", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("sources list --include-superseded returned %d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "src_cli_pinned") || !strings.Contains(out.String(), `"superseded": true`) {
		t.Fatalf("include superseded should show superseded source, got %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"sources", "read", missionID, sourceID, "-db", dbPath, "-local-source-root", rootSpec, "-json"}, &out, &errOut)
	if code == 0 || !strings.Contains(errOut.String(), "source is removed") {
		t.Fatalf("removed source read should fail, code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	assertCLINoRootPath(t, rootDir, out.String(), errOut.String())

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"sources", "attach-local", missionID, "-db", dbPath, "-local-source-root", rootSpec, "-root", "workspace", "-path", "docs/notes.txt", "-json"}, &out, &errOut)
	if code == 0 || !strings.Contains(errOut.String(), "restore is required") {
		t.Fatalf("removed duplicate attach should require restore, code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	assertCLINoRootPath(t, rootDir, out.String(), errOut.String())

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"sources", "attach-local", missionID, "-db", dbPath, "-local-source-root", rootSpec, "-root", "workspace", "-path", "docs/notes.txt", "-restore", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("attach-local --restore returned %d stderr=%q", code, errOut.String())
	}
	if !nestedCLIBool(t, decodeCLIJSON(t, out.String()), "restored") {
		t.Fatalf("expected restored attach response, got %q", out.String())
	}
	assertCLINoRootPath(t, rootDir, out.String(), errOut.String())

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"sources", "remove", missionID, sourceID, "-db", dbPath, "-reason", "restore command check", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("second sources remove returned %d stderr=%q", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"sources", "restore", missionID, sourceID, "-db", dbPath, "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("sources restore returned %d stderr=%q", code, errOut.String())
	}
	assertCLINoRootPath(t, rootDir, out.String(), errOut.String())
}

func TestRunSourcesConfluenceWorkflow(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	missionID := createCLITestMission(t, dbPath)
	var authHeaders []string
	var metadataQuery string
	pageVersion := 7
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/oauth/token/accessible-resources":
			_, _ = w.Write([]byte(`[{"id":"cloud_1","name":"Docs","url":"https://docs.atlassian.net","scopes":["read:page:confluence"]}]`))
		case "/wiki/rest/api/search":
			_, _ = w.Write([]byte(`{
				"results": [{
					"content": {
						"id": "123",
						"title": "Roadmap",
						"space": {"id": "987", "key": "ENG"},
						"version": {"when": "2026-07-02T05:10:00.000Z", "number": 7},
						"_links": {"webui": "/spaces/ENG/pages/123/Roadmap"}
					},
					"excerpt": "<p>roadmap result</p>",
					"url": "/spaces/ENG/pages/123/Roadmap"
				}],
				"_links": {"base": "https://docs.atlassian.net/wiki"}
			}`))
		case "/wiki/api/v2/pages/123":
			if r.URL.Query().Get("body-format") == "" {
				metadataQuery = r.URL.RawQuery
			}
			_, _ = fmt.Fprintf(w, `{
				"id": "123",
				"title": "Roadmap",
				"spaceId": "987",
				"version": {"createdAt": "2026-07-03T02:00:00.000Z", "number": %d},
				"body": {"storage": {"value": "<p>Hello version %d</p>", "representation": "storage"}},
				"_links": {"base": "https://docs.atlassian.net/wiki", "webui": "/spaces/ENG/pages/123/Roadmap"}
			}`, pageVersion, pageVersion)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{
		"sources", "confluence", "connect-token",
		"-db", dbPath,
		"-connection", "cnf_cli",
		"-name", "Docs",
		"-email", "person@example.com",
		"-api-token", "secret-api-token",
		"-json",
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("connect-token returned %d stderr=%q", code, errOut.String())
	}
	if strings.Contains(out.String(), "secret-api-token") || strings.Contains(errOut.String(), "secret-api-token") {
		t.Fatalf("connect-token leaked token stdout=%q stderr=%q", out.String(), errOut.String())
	}
	cloudID := "site_docs.atlassian.net"

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{
		"sources", "confluence", "search", missionID,
		"-db", dbPath,
		"-connection", "cnf_cli",
		"-cloud-id", cloudID,
		"-site-url", "https://docs.atlassian.net/wiki",
		"-query", "roadmap",
		"-api-base-url", server.URL + "/wiki",
		"-json",
	}, &out, &errOut)
	if code != 2 || !strings.Contains(errOut.String(), "--unsafe-allow-oauth-overrides") {
		t.Fatalf("expected api token api base override rejection, got code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	if strings.Contains(out.String(), "secret-api-token") || strings.Contains(errOut.String(), "secret-api-token") {
		t.Fatalf("api token override rejection leaked token stdout=%q stderr=%q", out.String(), errOut.String())
	}

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{
		"sources", "confluence", "search", missionID,
		"-db", dbPath,
		"-connection", "cnf_cli",
		"-cloud-id", cloudID,
		"-site-url", "https://docs.atlassian.net/wiki",
		"-query", "roadmap",
		"-api-base-url", server.URL + "/wiki",
		"-unsafe-allow-oauth-overrides",
		"-json",
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("search returned %d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"Title": "Roadmap"`) {
		t.Fatalf("expected Roadmap candidate, got %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{
		"sources", "confluence", "snapshot", missionID,
		"-db", dbPath,
		"-connection", "cnf_cli",
		"-cloud-id", cloudID,
		"-site-url", "https://docs.atlassian.net/wiki",
		"-page-id", "123",
		"-api-base-url", server.URL + "/wiki",
		"-json",
	}, &out, &errOut)
	if code != 2 || !strings.Contains(errOut.String(), "--version") {
		t.Fatalf("expected snapshot without version to fail usage, got code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{
		"sources", "confluence", "snapshot", missionID,
		"-db", dbPath,
		"-connection", "cnf_cli",
		"-cloud-id", cloudID,
		"-site-url", "https://docs.atlassian.net/wiki",
		"-page-id", "123",
		"-version", "7",
		"-api-base-url", server.URL + "/wiki",
		"-unsafe-allow-oauth-overrides",
		"-json",
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("snapshot returned %d stderr=%q", code, errOut.String())
	}
	snapshotID := nestedCLIString(t, decodeCLIJSON(t, out.String()), "Snapshot", "SnapshotID")
	if snapshotID == "" {
		t.Fatalf("expected snapshot id, got %q", out.String())
	}

	pageVersion = 8
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{
		"sources", "confluence", "check-update", missionID, snapshotID,
		"-db", dbPath,
		"-connection", "cnf_cli",
		"-site-url", "https://docs.atlassian.net/wiki",
		"-api-base-url", server.URL + "/wiki",
		"-unsafe-allow-oauth-overrides",
		"-json",
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("check-update returned %d stderr=%q", code, errOut.String())
	}
	if metadataQuery != "" {
		t.Fatalf("metadata request should not request body, got query %q", metadataQuery)
	}
	if !strings.Contains(out.String(), `"UpdateAvailable": true`) {
		t.Fatalf("expected update available, got %q", out.String())
	}
	if strings.Contains(out.String(), "Hello version 8") {
		t.Fatalf("check-update leaked page body: %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{
		"sources", "confluence", "update", missionID, snapshotID,
		"-db", dbPath,
		"-connection", "cnf_cli",
		"-site-url", "https://docs.atlassian.net/wiki",
		"-api-base-url", server.URL + "/wiki",
		"-json",
	}, &out, &errOut)
	if code != 2 || !strings.Contains(errOut.String(), "--version") {
		t.Fatalf("expected update without version to fail usage, got code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}

	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{
		"sources", "confluence", "update", missionID, snapshotID,
		"-db", dbPath,
		"-connection", "cnf_cli",
		"-version", "8",
		"-site-url", "https://docs.atlassian.net/wiki",
		"-api-base-url", server.URL + "/wiki",
		"-unsafe-allow-oauth-overrides",
		"-json",
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("update returned %d stderr=%q", code, errOut.String())
	}
	updated := decodeCLIJSON(t, out.String())
	newSnapshotID := nestedCLIString(t, updated, "Snapshot", "SnapshotID")
	if newSnapshotID == "" || newSnapshotID == snapshotID {
		t.Fatalf("expected new snapshot id, got %q", out.String())
	}
	if strings.Contains(out.String(), "secret-api-token") || strings.Contains(errOut.String(), "secret-api-token") {
		t.Fatalf("confluence workflow leaked token stdout=%q stderr=%q", out.String(), errOut.String())
	}
	for _, auth := range authHeaders {
		if auth != "" && !strings.HasPrefix(auth, "Basic ") {
			t.Fatalf("unexpected auth header %q", auth)
		}
	}
}

func TestRunSourcesConfluenceAPITokenRejectsUnsafeAPIBaseURL(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	missionID := createCLITestMission(t, dbPath)
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{
		"sources", "confluence", "connect-token",
		"-db", dbPath,
		"-connection", "cnf_api",
		"-name", "Docs API",
		"-email", "person@example.com",
		"-api-token", "secret-api-token",
		"-json",
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("connect-token returned %d stderr=%q", code, errOut.String())
	}
	for _, apiBaseURL := range []string{
		"https://attacker.example/wiki",
		"http://docs.atlassian.net/wiki",
		"https://other.atlassian.net/wiki",
	} {
		out.Reset()
		errOut.Reset()
		code = run(context.Background(), []string{
			"sources", "confluence", "search", missionID,
			"-db", dbPath,
			"-connection", "cnf_api",
			"-cloud-id", "cloud_1",
			"-query", "roadmap",
			"-site-url", "https://docs.atlassian.net/wiki",
			"-api-base-url", apiBaseURL,
			"-json",
		}, &out, &errOut)
		if code != 2 || !strings.Contains(errOut.String(), "API base URL") {
			t.Fatalf("expected unsafe API base URL %q rejection, got code=%d stdout=%q stderr=%q", apiBaseURL, code, out.String(), errOut.String())
		}
		if strings.Contains(out.String(), "secret-api-token") || strings.Contains(errOut.String(), "secret-api-token") {
			t.Fatalf("unsafe API base rejection leaked token stdout=%q stderr=%q", out.String(), errOut.String())
		}
	}
}

func TestRunSourcesConfluenceOAuthCommandsDisabled(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")

	for _, tc := range []struct {
		name string
		args []string
	}{
		{
			name: "oauth-url",
			args: []string{
				"sources", "confluence", "oauth-url",
				"-client-id", "client-id",
				"-redirect-uri", "http://127.0.0.1/callback",
				"-state", "state-1",
				"-json",
			},
		},
		{
			name: "oauth-exchange",
			args: []string{
				"sources", "confluence", "oauth-exchange",
				"-db", dbPath,
				"-connection", "cnf_cli_oauth",
				"-name", "Docs OAuth",
				"-client-id", "client-id",
				"-client-secret", "client-secret",
				"-redirect-uri", "http://127.0.0.1/callback",
				"-code", "code-1",
				"-json",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			code := run(context.Background(), tc.args, &out, &errOut)
			if code != 2 || !strings.Contains(errOut.String(), "API token") {
				t.Fatalf("expected OAuth command to be disabled, got code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
			}
			for _, leaked := range []string{"client-secret", "code-1", "state-1"} {
				if strings.Contains(out.String(), leaked) {
					t.Fatalf("disabled OAuth command leaked %q stdout=%q stderr=%q", leaked, out.String(), errOut.String())
				}
			}
		})
	}
}

func TestRunWorkflowStartStatusStopJSON(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	missionID := createCLITestMission(t, dbPath)
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"workflow", "start", missionID, "-db", dbPath, "-instruction", "Make bounded progress", "-json"}, &out, &errOut)
	if code == 0 || !strings.Contains(errOut.String(), "requires --wait") {
		t.Fatalf("workflow start without --wait should fail, code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	store, err := sqlite.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	workflowRun, err := svc.RequestWorkflowRun(context.Background(), app.RequestWorkflowRunRequest{
		WorkflowRunID:      "wfr_cli_status",
		MissionID:          missionID,
		RequestedBySurface: app.WorkflowSurfaceCLI,
		AgentExecutor:      "codex",
		MCPMode:            "auto",
		Instruction:        "Make bounded progress",
		MaxSteps:           1,
		MaxDurationMS:      60000,
	})
	if err != nil {
		t.Fatalf("RequestWorkflowRun returned error: %v", err)
	}
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"workflow", "status", missionID, workflowRun.WorkflowRunID, "-db", dbPath, "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("workflow status returned %d stderr=%q", code, errOut.String())
	}
	if got := nestedCLIString(t, decodeCLIJSON(t, out.String()), "workflow_run", "workflow_run_id"); got != workflowRun.WorkflowRunID {
		t.Fatalf("expected run %q, got %q", workflowRun.WorkflowRunID, got)
	}
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"workflow", "stop", missionID, workflowRun.WorkflowRunID, "-db", dbPath, "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("workflow stop returned %d stderr=%q", code, errOut.String())
	}
	if status := nestedCLIString(t, decodeCLIJSON(t, out.String()), "workflow_run", "status"); status != "stopped" {
		t.Fatalf("expected stopped run, got %q", status)
	}
}

func TestRunProviderCommandsRequireWaitWithoutBackgroundWorker(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	missionID := createCLITestMission(t, dbPath)
	cases := [][]string{
		{"turns", "send", missionID, "-db", dbPath, "-text", "hello"},
		{"reports", "draft", missionID, "-db", dbPath, "-title", "Report"},
	}
	for _, args := range cases {
		var out, errOut bytes.Buffer
		code := run(context.Background(), args, &out, &errOut)
		if code == 0 || !strings.Contains(errOut.String(), "requires --wait") {
			t.Fatalf("%v should require --wait, code=%d stdout=%q stderr=%q", args, code, out.String(), errOut.String())
		}
	}
}

func TestBuildCLIAgentExecutorPassesLocalRootsToMCP(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	rootSpec := "workspace=" + t.TempDir()
	executor, err := buildCLIAgentExecutor(context.Background(), cliAgentConfig{
		AgentName:    "codex",
		DBPath:       dbPath,
		CodexCommand: "codex",
		LocalRoots:   []string{rootSpec},
	})
	if err != nil {
		t.Fatalf("buildCLIAgentExecutor returned error: %v", err)
	}
	codex, ok := executor.(web.CodexExecutor)
	if !ok {
		t.Fatalf("expected CodexExecutor, got %T", executor)
	}
	if !hasCLIArgPair(codex.MCPServer.Args, "-local-source-root", rootSpec) {
		t.Fatalf("expected local source root in MCP args, got %#v", codex.MCPServer.Args)
	}
	if !hasCLIArgPair(codex.MCPServer.Args, "-enabled-tool", mcp.ToolResearchOutline) {
		t.Fatalf("expected enabled tool in MCP args, got %#v", codex.MCPServer.Args)
	}
}

func TestBuildCLIClaudeAgentExecutorPassesLocalRootsToMCP(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	rootSpec := "workspace=" + t.TempDir()
	executor, err := buildCLIAgentExecutor(context.Background(), cliAgentConfig{
		AgentName:     "claude",
		DBPath:        dbPath,
		ClaudeCommand: "claude",
		ClaudeModel:   "haiku",
		LocalRoots:    []string{rootSpec},
	})
	if err != nil {
		t.Fatalf("buildCLIAgentExecutor returned error: %v", err)
	}
	claude, ok := executor.(web.ClaudeExecutor)
	if !ok {
		t.Fatalf("expected ClaudeExecutor, got %T", executor)
	}
	if claude.Model != "haiku" {
		t.Fatalf("expected haiku model, got %q", claude.Model)
	}
	if !hasCLIArgPair(claude.MCPServer.Args, "-local-source-root", rootSpec) {
		t.Fatalf("expected local source root in MCP args, got %#v", claude.MCPServer.Args)
	}
	if !hasCLIArgPair(claude.MCPServer.Args, "-enabled-tool", mcp.ToolResearchOutline) {
		t.Fatalf("expected enabled tool in MCP args, got %#v", claude.MCPServer.Args)
	}
}

func TestValidateMCPBindingRejectsUnsupportedAgentExecutor(t *testing.T) {
	err := validateMCPBinding(mcp.Binding{
		MissionID:      "mis_1",
		AgentSessionID: "ses_1",
		AgentExecutor:  "unknown",
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported agent executor") {
		t.Fatalf("expected unsupported agent executor rejection, got %v", err)
	}
}

func TestRunProviderCommandsUseCLILocalRootsBeforeEnv(t *testing.T) {
	for _, tc := range []struct {
		name      string
		args      func(string, string, string) []string
		responses []web.AgentResult
	}{
		{
			name: "turns send",
			args: func(missionID, dbPath, rootSpec string) []string {
				return []string{"turns", "send", missionID, "-db", dbPath, "-text", "hello", "-wait", "-local-source-root", rootSpec, "-json"}
			},
			responses: []web.AgentResult{{Text: "answer", SessionID: "agent-session-1"}},
		},
		{
			name: "workflow start",
			args: func(missionID, dbPath, rootSpec string) []string {
				return []string{"workflow", "start", missionID, "-db", dbPath, "-instruction", "continue", "-max-steps", "1", "-wait", "-local-source-root", rootSpec, "-json"}
			},
			responses: []web.AgentResult{{Text: "workflow answer\nPLASMA_WORKFLOW_CONTROL: {\"decision\":\"stop\",\"reason\":\"done\"}", SessionID: "agent-session-1"}},
		},
		{
			name: "reports draft",
			args: func(missionID, dbPath, rootSpec string) []string {
				return []string{"reports", "draft", missionID, "-db", dbPath, "-title", "Report", "-wait", "-local-source-root", rootSpec, "-json"}
			},
			responses: []web.AgentResult{
				{Text: "- Plan report.", SessionID: "agent-session-1"},
				{Text: "# Report\n\nBody.", SessionID: "agent-session-1"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dbPath := filepath.Join(t.TempDir(), "plasma.db")
			missionID := createCLITestMission(t, dbPath)
			rootSpec := "workspace=" + t.TempDir()
			envRootSpec := "envspace=" + t.TempDir()
			t.Setenv(config.LocalSourceRootsEnv, envRootSpec)
			t.Setenv("PLASMA_AGENT", "claude")
			t.Setenv("PLASMA_CLAUDE_COMMAND", "/opt/test/claude")
			t.Setenv("PLASMA_CLAUDE_MODEL", "sonnet")
			fake := &cliFakeAgent{responses: tc.responses}
			oldFactory := newCLIAgentExecutor
			newCLIAgentExecutor = func(_ context.Context, cfg cliAgentConfig) (web.AgentExecutor, error) {
				if cfg.AgentName != "claude" || cfg.ClaudeCommand != "/opt/test/claude" || cfg.ClaudeModel != "sonnet" {
					t.Fatalf("expected env agent config to survive CLI defaults, got %#v", cfg)
				}
				if len(cfg.LocalRoots) != 1 || cfg.LocalRoots[0] != rootSpec {
					t.Fatalf("expected CLI local roots to replace env roots %q in agent config, got %#v", envRootSpec, cfg.LocalRoots)
				}
				return fake, nil
			}
			t.Cleanup(func() { newCLIAgentExecutor = oldFactory })

			var out, errOut bytes.Buffer
			code := run(context.Background(), tc.args(missionID, dbPath, rootSpec), &out, &errOut)
			if code != 0 {
				t.Fatalf("%s returned %d stdout=%q stderr=%q", tc.name, code, out.String(), errOut.String())
			}
		})
	}
}

func TestRunProviderCommandsDoNotRecordPendingWhenExecutorBuildFails(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	oldFactory := newCLIAgentExecutor
	newCLIAgentExecutor = func(context.Context, cliAgentConfig) (web.AgentExecutor, error) {
		return nil, errors.New("executor unavailable")
	}
	t.Cleanup(func() { newCLIAgentExecutor = oldFactory })

	cases := []struct {
		args      []string
		forbidden []string
	}{
		{
			args:      []string{"turns", "send", "-text", "hello", "-wait"},
			forbidden: []string{"turn.user", "turn.agent.pending"},
		},
		{
			args:      []string{"workflow", "start", "-instruction", "continue", "-wait"},
			forbidden: []string{app.WorkflowRunRequestedEvent},
		},
		{
			args:      []string{"reports", "draft", "-title", "Report", "-wait"},
			forbidden: []string{"report.draft.pending"},
		},
	}
	for _, tc := range cases {
		missionID := createCLITestMission(t, dbPath)
		args := append([]string{}, tc.args[:2]...)
		args = append(args, missionID, "-db", dbPath)
		args = append(args, tc.args[2:]...)
		var out, errOut bytes.Buffer
		code := run(context.Background(), args, &out, &errOut)
		if code == 0 || !strings.Contains(errOut.String(), "executor unavailable") {
			t.Fatalf("expected executor failure for %v, code=%d stdout=%q stderr=%q", args, code, out.String(), errOut.String())
		}
		store, err := sqlite.Open(context.Background(), dbPath)
		if err != nil {
			t.Fatal(err)
		}
		events, err := app.NewService(store).ListEvents(context.Background(), missionID)
		if closeErr := store.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
		if err != nil {
			t.Fatal(err)
		}
		for _, eventType := range tc.forbidden {
			if cliCountEventType(events, eventType) != 0 {
				t.Fatalf("executor build failure should not record %s for %v, got %#v", eventType, args, events)
			}
		}
	}
}

func TestRunTurnsWaitDrainsWorkflowRequestedByMCPContext(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	missionID := createCLITestMission(t, dbPath)
	store, err := sqlite.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	fake := &cliFakeAgent{responses: []web.AgentResult{
		{Text: "first answer", SessionID: "agent-session-1"},
		{Text: "workflow answer\nPLASMA_WORKFLOW_CONTROL: {\"decision\":\"stop\",\"reason\":\"done\"}", SessionID: "agent-session-1"},
	}}
	fake.onRun = func(req web.AgentRequest) {
		if req.UserEventID == "" {
			t.Fatalf("expected CLI agent request to include user event id: %#v", req)
		}
		if _, err := svc.RequestWorkflowRun(context.Background(), app.RequestWorkflowRunRequest{
			MissionID:                req.MissionID,
			RequestedBySurface:       app.WorkflowSurfaceMCP,
			RequestedByToolSessionID: req.ToolSessionID,
			AgentExecutor:            "codex",
			MCPMode:                  "auto",
			Instruction:              "continue after this turn",
			MaxSteps:                 1,
			MaxDurationMS:            60000,
			StartAfterEventID:        req.UserEventID,
		}); err != nil {
			t.Fatalf("RequestWorkflowRun from fake MCP context returned error: %v", err)
		}
	}
	oldFactory := newCLIAgentExecutor
	newCLIAgentExecutor = func(context.Context, cliAgentConfig) (web.AgentExecutor, error) {
		return fake, nil
	}
	t.Cleanup(func() { newCLIAgentExecutor = oldFactory })

	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"turns", "send", missionID, "-db", dbPath, "-text", "start autonomous follow-up", "-wait", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("turns send returned %d stderr=%q", code, errOut.String())
	}
	if len(fake.requests) != 2 {
		t.Fatalf("expected main turn and drained workflow request, got %#v", fake.requests)
	}
	events, err := svc.ListEvents(context.Background(), missionID)
	if err != nil {
		t.Fatal(err)
	}
	if cliCountEventType(events, app.WorkflowRunCompletedEvent) != 1 {
		t.Fatalf("expected drained workflow completion event, got %#v", events)
	}
}

func TestRunWorkflowWaitResumesSameSessionAfterTurn(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	missionID := createCLITestMission(t, dbPath)
	fake := &cliFakeAgent{responses: []web.AgentResult{
		{Text: "first answer", SessionID: "agent-session-1"},
		{Text: "workflow answer\nPLASMA_WORKFLOW_CONTROL: {\"decision\":\"stop\",\"reason\":\"done\"}", SessionID: "agent-session-1"},
	}}
	oldFactory := newCLIAgentExecutor
	newCLIAgentExecutor = func(context.Context, cliAgentConfig) (web.AgentExecutor, error) {
		return fake, nil
	}
	t.Cleanup(func() { newCLIAgentExecutor = oldFactory })

	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"turns", "send", missionID, "-db", dbPath, "-text", "first", "-wait", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("turns send returned %d stderr=%q", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"workflow", "start", missionID, "-db", dbPath, "-instruction", "continue", "-max-steps", "1", "-wait", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("workflow start --wait returned %d stderr=%q", code, errOut.String())
	}
	result := decodeCLIJSON(t, out.String())
	if status := nestedCLIString(t, result, "workflow_run", "status"); status != "completed" {
		t.Fatalf("expected completed workflow, got %q", status)
	}
	if len(fake.requests) != 2 || fake.requests[1].PreviousSessionID != "agent-session-1" {
		t.Fatalf("expected workflow request to resume same provider session, got %#v", fake.requests)
	}
}

func TestCLIWorkflowAgentAdapterForwardsCompactionAndUsage(t *testing.T) {
	usage := agentusage.New("codex", "codex", "gpt-5.5", "high", "workflow prompt").
		WithProviderUsage(agentusage.ProviderUsage{
			InputTokens:       120,
			CachedInputTokens: 80,
			OutputTokens:      30,
		}, "test")
	fake := &cliFakeAgent{responses: []web.AgentResult{{
		Text:      "workflow answer",
		SessionID: "agent-session-1",
		Usage:     usage,
	}}}
	result, err := (cliWorkflowAgentAdapter{executor: fake}).Run(context.Background(), workflowruntime.AgentRequest{
		UserText:          "user request",
		Prompt:            "workflow prompt",
		Model:             "gpt-5.5",
		ReasoningEffort:   "high",
		MissionID:         "mis_1",
		ToolSessionID:     "ses_1",
		UserEventID:       "evt_1",
		PreviousSessionID: "agent-session-0",
		AgentExecutor:     "codex",
		MCPMode:           "workflow",
		Compaction:        true,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(fake.requests) != 1 {
		t.Fatalf("expected one forwarded request, got %#v", fake.requests)
	}
	if !fake.requests[0].Compaction {
		t.Fatalf("expected compaction flag to be forwarded, got %#v", fake.requests[0])
	}
	if fake.requests[0].MCPMode != "workflow" || fake.requests[0].PreviousSessionID != "agent-session-0" {
		t.Fatalf("expected workflow request metadata to be forwarded, got %#v", fake.requests[0])
	}
	if result.Usage.ProviderUsage == nil {
		t.Fatal("expected usage to be returned from CLI workflow adapter")
	}
	if result.Usage.ProviderUsage.InputTokens != 120 || result.Usage.ProviderUsage.CachedInputTokens != 80 || result.Usage.ProviderUsage.OutputTokens != 30 {
		t.Fatalf("unexpected forwarded usage: %#v", result.Usage.ProviderUsage)
	}
}

func TestRunReportsDraftWaitUsesSameSessionAndMarkdownArtifact(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	missionID := createCLITestMission(t, dbPath)
	fake := &cliFakeAgent{responses: []web.AgentResult{
		{Text: "first answer", SessionID: "agent-session-1"},
		{Text: "- Plan CLI report.", SessionID: "agent-session-1"},
		{Text: "# CLI Report\n\n보고서가 작성되어야 한다.", SessionID: "agent-session-1"},
		{Text: "H5 patch finalized.", SessionID: "agent-session-1"},
	}}
	fake.onEveryRun = cliHumanizePatchFinalizer(t, dbPath, "# CLI Report\n\n보고서를 작성해야 한다.")
	oldFactory := newCLIAgentExecutor
	newCLIAgentExecutor = func(context.Context, cliAgentConfig) (web.AgentExecutor, error) {
		return fake, nil
	}
	t.Cleanup(func() { newCLIAgentExecutor = oldFactory })

	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"turns", "send", missionID, "-db", dbPath, "-text", "first", "-wait", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("turns send returned %d stderr=%q", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"reports", "draft", missionID, "-db", dbPath, "-title", "CLI Report", "-wait", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("reports draft returned %d stderr=%q", code, errOut.String())
	}
	result := decodeCLIJSON(t, out.String())
	if got := nestedCLIString(t, result, "event", "EventType"); got != "report.artifact.created" {
		t.Fatalf("expected report.artifact.created, got %q", got)
	}
	if len(fake.requests) != 3 ||
		fake.requests[1].PreviousSessionID != "agent-session-1" ||
		fake.requests[2].PreviousSessionID != "agent-session-1" {
		t.Fatalf("expected report request to resume same provider session, got %#v", fake.requests)
	}

	store, err := sqlite.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	events, err := app.NewService(store).ListEvents(context.Background(), missionID)
	if err != nil {
		t.Fatal(err)
	}
	if cliCountEventType(events, "report.artifact.created") != 1 {
		t.Fatalf("expected one markdown report artifact event, got %#v", events)
	}
	if cliCountEventType(events, "report.humanize.pending") != 0 || cliCountEventType(events, "report.artifact.exported") != 0 {
		t.Fatalf("default CLI report must not create a humanized export, got %#v", events)
	}
	for _, legacy := range []string{"report.drafted"} {
		if cliCountEventType(events, legacy) != 0 {
			t.Fatalf("default CLI report path should not create %s", legacy)
		}
	}
	if cliCountEventType(events, "report.plan.created") != 1 {
		t.Fatalf("default CLI report path should create one report.plan.created event, got %#v", events)
	}
	pendingPayload := cliLatestEventPayload(t, events, "report.draft.pending")
	if pendingPayload["report_session_policy"] != reporting.SessionPolicySameSession ||
		pendingPayload["report_session_policy_selection"] != reporting.SessionPolicySelectionAutoSameSessionNoForker {
		t.Fatalf("expected non-forking CLI report to record same-session fallback, got %#v", pendingPayload)
	}
	artifactPayload := cliLatestEventPayload(t, events, "report.artifact.created")
	if artifactPayload["report_session_policy"] != reporting.SessionPolicySameSession ||
		artifactPayload["report_session_policy_selection"] != reporting.SessionPolicySelectionAutoSameSessionNoForker ||
		artifactPayload["pre_report_research_session_id"] != "agent-session-1" ||
		artifactPayload["report_plan_session_id"] != "agent-session-1" ||
		artifactPayload["report_session_id"] != "agent-session-1" {
		t.Fatalf("expected same-session report metadata, got %#v", artifactPayload)
	}
	if artifactPayload["post_report_humanize"] != "disabled" ||
		artifactPayload["humanize_enabled"] != false ||
		artifactPayload["generation_guidance_profile"] != "g2" ||
		artifactPayload["generation_guidance_sha256"] == "" {
		t.Fatalf("expected default CLI report to record G2 guidance and disabled H5, got %#v", artifactPayload)
	}
}

func TestRunReportsDraftExperimentalGuidanceCanSkipHumanize(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	missionID := createCLITestMission(t, dbPath)
	fake := &cliFakeAgent{responses: []web.AgentResult{
		{Text: "- Plan CLI report.", SessionID: "report-session-1"},
		{Text: "# CLI Report\n\n구체적인 내용을 보존한 보고서입니다.", SessionID: "report-session-1"},
	}}
	oldFactory := newCLIAgentExecutor
	newCLIAgentExecutor = func(context.Context, cliAgentConfig) (web.AgentExecutor, error) {
		return fake, nil
	}
	t.Cleanup(func() { newCLIAgentExecutor = oldFactory })

	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{
		"reports", "draft", missionID,
		"-db", dbPath,
		"-title", "CLI Report",
		"-wait",
		"-json",
		"-humanize=false",
		"-experimental-generation-guidance", "g2",
		"-report-session-policy", "same_session",
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("reports draft returned %d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	if len(fake.requests) != 2 {
		t.Fatalf("expected plan and report requests only, got %#v", fake.requests)
	}
	if strings.Contains(fake.requests[0].Prompt, "Generation guidance:") {
		t.Fatalf("plan prompt must not receive experimental generation guidance: %s", fake.requests[0].Prompt)
	}
	if !strings.Contains(fake.requests[1].Prompt, "Report writing guidance:") ||
		!strings.Contains(fake.requests[1].Prompt, "never improve fluency by dropping concrete source details") {
		t.Fatalf("report prompt did not receive G2 guidance: %s", fake.requests[1].Prompt)
	}
	if fake.requests[1].ReportPatch != nil {
		t.Fatalf("humanize=false should not create an H5 report patch request")
	}

	store, err := sqlite.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	events, err := app.NewService(store).ListEvents(context.Background(), missionID)
	if err != nil {
		t.Fatal(err)
	}
	if cliCountEventType(events, "report.humanize.pending") != 0 ||
		cliCountEventType(events, "report.artifact.exported") != 0 {
		t.Fatalf("humanize=false should not create H5 events, got %#v", events)
	}
	pendingPayload := cliLatestEventPayload(t, events, "report.draft.pending")
	if pendingPayload["post_report_humanize"] != "disabled" ||
		pendingPayload["humanize_enabled"] != false ||
		pendingPayload["generation_guidance_profile"] != "g2" ||
		pendingPayload["generation_guidance_sha256"] == "" {
		t.Fatalf("expected pending event to record experimental guidance and disabled H5, got %#v", pendingPayload)
	}
	artifactPayload := cliLatestEventPayload(t, events, "report.artifact.created")
	if artifactPayload["post_report_humanize"] != "disabled" ||
		artifactPayload["humanize_enabled"] != false ||
		artifactPayload["generation_guidance_profile"] != "g2" ||
		artifactPayload["generation_guidance_sha256"] != pendingPayload["generation_guidance_sha256"] {
		t.Fatalf("expected artifact event to carry matching experimental metadata, got %#v", artifactPayload)
	}
}

func TestRunReportsDraftRejectsUnsupportedExperimentalGenerationGuidance(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	missionID := createCLITestMission(t, dbPath)

	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{
		"reports", "draft", missionID,
		"-db", dbPath,
		"-title", "CLI Report",
		"-wait",
		"-experimental-generation-guidance", "unknown",
	}, &out, &errOut)
	if code != 2 {
		t.Fatalf("expected unsupported guidance to return code 2, got %d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "unsupported report generation guidance profile") {
		t.Fatalf("expected unsupported guidance error, got %q", errOut.String())
	}
}

func TestRunReportsDraftWaitUsesIsolatedForkWhenAvailable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	missionID := createCLITestMission(t, dbPath)
	fake := &cliForkingFakeAgent{
		cliFakeAgent: cliFakeAgent{responses: []web.AgentResult{
			{Text: "first answer", SessionID: "research-session-1"},
			{Text: "- Plan CLI report.", SessionID: "report-fork-1"},
			{Text: "# CLI Report\n\n보고서가 작성되어야 한다.", SessionID: "report-fork-1"},
			{Text: "H5 patch finalized.", SessionID: "report-fork-1"},
		}},
		forkSessionID: "report-fork-1",
	}
	fake.onEveryRun = cliHumanizePatchFinalizer(t, dbPath, "# CLI Report\n\n보고서를 작성해야 한다.")
	oldFactory := newCLIAgentExecutor
	newCLIAgentExecutor = func(context.Context, cliAgentConfig) (web.AgentExecutor, error) {
		return fake, nil
	}
	t.Cleanup(func() { newCLIAgentExecutor = oldFactory })

	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"turns", "send", missionID, "-db", dbPath, "-text", "first", "-wait", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("turns send returned %d stderr=%q", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"reports", "draft", missionID, "-db", dbPath, "-title", "CLI Report", "-wait", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("reports draft returned %d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	if len(fake.forkSources) != 1 || fake.forkSources[0] != "research-session-1" {
		t.Fatalf("expected report fork from research session, got %#v", fake.forkSources)
	}
	if len(fake.requests) != 3 ||
		fake.requests[0].PreviousSessionID != "" ||
		fake.requests[1].PreviousSessionID != "report-fork-1" ||
		fake.requests[2].PreviousSessionID != "report-fork-1" {
		t.Fatalf("expected report plan/body to run on isolated fork, got %#v", fake.requests)
	}

	store, err := sqlite.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	events, err := app.NewService(store).ListEvents(context.Background(), missionID)
	if err != nil {
		t.Fatal(err)
	}
	pendingPayload := cliLatestEventPayload(t, events, "report.draft.pending")
	if pendingPayload["report_session_policy"] != reporting.SessionPolicyIsolatedFork ||
		pendingPayload["report_session_policy_selection"] != reporting.SessionPolicySelectionAutoIsolatedFork {
		t.Fatalf("expected pending event to record automatic isolated fork, got %#v", pendingPayload)
	}
	planPayload := cliLatestEventPayload(t, events, "report.plan.created")
	if planPayload["previous_agent_session_id"] != "report-fork-1" ||
		planPayload["pre_report_research_session_id"] != "research-session-1" ||
		planPayload["report_plan_session_id"] != "report-fork-1" {
		t.Fatalf("expected isolated plan metadata, got %#v", planPayload)
	}
	artifactPayload := cliLatestEventPayload(t, events, "report.artifact.created")
	if artifactPayload["report_session_policy"] != reporting.SessionPolicyIsolatedFork ||
		artifactPayload["report_session_policy_selection"] != reporting.SessionPolicySelectionAutoIsolatedFork ||
		artifactPayload["previous_agent_session_id"] != "report-fork-1" ||
		artifactPayload["pre_report_research_session_id"] != "research-session-1" ||
		artifactPayload["report_plan_session_id"] != "report-fork-1" ||
		artifactPayload["report_session_id"] != "report-fork-1" ||
		artifactPayload["fork_source_agent_session_id"] != "research-session-1" {
		t.Fatalf("expected isolated report metadata, got %#v", artifactPayload)
	}
	if got := workflowruntime.LatestAgentSessionID(events, "codex"); got != "research-session-1" {
		t.Fatalf("expected later turns to resume research session, got %q", got)
	}
}

func TestRunReportsDraftRejectsLongFormUntilCLISectionRunnerExists(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	missionID := createCLITestMission(t, dbPath)

	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"reports", "draft", missionID, "-db", dbPath, "-title", "CLI Long Form", "-mode", "long_form", "-wait", "-json"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("expected long_form CLI rejection code 2, got %d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "CLI reports currently support planned or one_take") {
		t.Fatalf("expected long_form rejection explanation, got %q", errOut.String())
	}
}

func TestRunReportsDraftWaitRecordsFailureOnSessionMismatch(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	missionID := createCLITestMission(t, dbPath)
	fake := &cliFakeAgent{responses: []web.AgentResult{
		{Text: "first answer", SessionID: "agent-session-1"},
		{Text: "- Plan CLI report.", SessionID: "agent-session-1"},
		{Text: "# CLI Report\n\nWrong session body.", SessionID: "agent-session-2"},
	}}
	oldFactory := newCLIAgentExecutor
	newCLIAgentExecutor = func(context.Context, cliAgentConfig) (web.AgentExecutor, error) {
		return fake, nil
	}
	t.Cleanup(func() { newCLIAgentExecutor = oldFactory })

	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"turns", "send", missionID, "-db", dbPath, "-text", "first", "-wait", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("turns send returned %d stderr=%q", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"reports", "draft", missionID, "-db", dbPath, "-title", "CLI Report", "-wait", "-json"}, &out, &errOut)
	if code == 0 {
		t.Fatalf("expected reports draft to fail on session mismatch, stdout=%q", out.String())
	}

	store, err := sqlite.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	events, err := app.NewService(store).ListEvents(context.Background(), missionID)
	if err != nil {
		t.Fatal(err)
	}
	if cliCountEventType(events, "report.draft.failed") != 1 {
		t.Fatalf("expected report.draft.failed event, got %#v", events)
	}
}

func TestRunEndToEndWorkflowThenConversationThenReportUsesSameLedgerAndSession(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	missionID := createCLITestMission(t, dbPath)
	fake := &cliFakeAgent{responses: []web.AgentResult{
		{Text: "first answer", SessionID: "agent-session-1"},
		{Text: "workflow answer\nPLASMA_WORKFLOW_CONTROL: {\"decision\":\"stop\",\"reason\":\"done\"}", SessionID: "agent-session-1"},
		{Text: "resumed answer", SessionID: "agent-session-1"},
		{Text: "- Plan final report.", SessionID: "agent-session-1"},
		{Text: "# Final Report\n\n보고서가 작성되어야 합니다.", SessionID: "agent-session-1"},
		{Text: "H5 patch finalized.", SessionID: "agent-session-1"},
	}}
	fake.onEveryRun = cliHumanizePatchFinalizer(t, dbPath, "# Final Report\n\n보고서를 작성해야 합니다.")
	oldFactory := newCLIAgentExecutor
	newCLIAgentExecutor = func(context.Context, cliAgentConfig) (web.AgentExecutor, error) {
		return fake, nil
	}
	t.Cleanup(func() { newCLIAgentExecutor = oldFactory })

	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"turns", "send", missionID, "-db", dbPath, "-text", "first", "-wait", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("first turns send returned %d stderr=%q", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"workflow", "start", missionID, "-db", dbPath, "-instruction", "continue", "-max-steps", "2", "-wait", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("workflow start --wait returned %d stderr=%q", code, errOut.String())
	}
	if status := nestedCLIString(t, decodeCLIJSON(t, out.String()), "workflow_run", "status"); status != "completed" {
		t.Fatalf("expected completed workflow, got %q", status)
	}
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"turns", "send", missionID, "-db", dbPath, "-text", "resume after workflow", "-wait", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("second turns send returned %d stderr=%q", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"reports", "draft", missionID, "-db", dbPath, "-title", "Final Report", "-wait", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("reports draft returned %d stderr=%q", code, errOut.String())
	}
	if got := nestedCLIString(t, decodeCLIJSON(t, out.String()), "event", "EventType"); got != "report.artifact.created" {
		t.Fatalf("expected report.artifact.created, got %q", got)
	}

	if len(fake.requests) != 5 {
		t.Fatalf("expected first turn, workflow, resumed turn, report requests, got %#v", fake.requests)
	}
	if fake.requests[0].PreviousSessionID != "" ||
		fake.requests[1].PreviousSessionID != "agent-session-1" ||
		fake.requests[2].PreviousSessionID != "agent-session-1" ||
		fake.requests[3].PreviousSessionID != "agent-session-1" ||
		fake.requests[4].PreviousSessionID != "agent-session-1" {
		t.Fatalf("expected workflow and non-forking report fallback to resume the provider session, got %#v", fake.requests)
	}

	store, err := sqlite.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	events, err := app.NewService(store).ListEvents(context.Background(), missionID)
	if err != nil {
		t.Fatal(err)
	}
	assertCLIEventTypeOrder(t, cliEventTypes(events), []string{
		"mission.created",
		"turn.user",
		"turn.agent.pending",
		"turn.agent.response",
		"workflow.run.requested",
		"workflow.run.started",
		"workflow.step.started",
		"turn.user",
		"turn.agent.pending",
		"turn.agent.response",
		"workflow.step.completed",
		"workflow.run.completed",
		"turn.user",
		"turn.agent.pending",
		"turn.agent.response",
		"report.draft.pending",
		"report.plan.created",
		"report.artifact.created",
	})
	if cliCountWorkflowSteeringTurns(t, events) != 1 {
		t.Fatalf("expected one workflow_steering turn, got events %#v", events)
	}
	for _, legacy := range []string{"report.drafted", "evidence.proposed", "claim.proposed", "proposal.submitted"} {
		if cliCountEventType(events, legacy) != 0 {
			t.Fatalf("default end-to-end flow should not create %s", legacy)
		}
	}
}

func TestCodexEnabledToolsExposeResearchSurface(t *testing.T) {
	tools := codexEnabledTools()
	for _, expected := range []string{
		mcp.ToolResearchOutline,
		mcp.ToolResearchList,
		mcp.ToolResearchGrep,
		mcp.ToolResearchRead,
		mcp.ToolResearchRefs,
		mcp.ToolSourcesList,
		mcp.ToolSourcesRead,
		mcp.ToolSourcesTree,
		mcp.ToolSourcesGrep,
		mcp.ToolSourcesSearch,
		mcp.ToolSourceCandidatesPropose,
		mcp.ToolSourceCandidatesRead,
		mcp.ToolWorkflowStart,
		mcp.ToolWorkflowStatus,
		mcp.ToolWorkflowStop,
	} {
		if !containsString(tools, expected) {
			t.Fatalf("expected Codex enabled tools to include %q: %#v", expected, tools)
		}
	}
	for _, legacy := range []string{mcp.ToolEvidencePropose, mcp.ToolQuestionsPropose, mcp.ToolClaimsPropose, mcp.ToolClaimConfidence, mcp.ToolProposalsSubmit} {
		if containsString(tools, legacy) {
			t.Fatalf("Codex enabled tools should not include legacy mutation tool %q: %#v", legacy, tools)
		}
	}
	for _, mutation := range []string{mcp.ToolLocalPathAttach, mcp.ToolSourcesRemove, mcp.ToolSourcesRestore} {
		if containsString(tools, mutation) {
			t.Fatalf("Codex enabled tools should not include source mutation tool %q: %#v", mutation, tools)
		}
	}
	for _, rootBrowse := range []string{mcp.ToolLocalPathRoots, mcp.ToolLocalPathTree} {
		if containsString(tools, rootBrowse) {
			t.Fatalf("Codex enabled tools should not include root-scoped local path browse tool %q: %#v", rootBrowse, tools)
		}
	}
	for _, experimental := range []string{mcp.ToolExperimentReportCreate, mcp.ToolExperimentReportAppend, mcp.ToolExperimentReportRead, mcp.ToolExperimentReportFinalize} {
		if containsString(tools, experimental) {
			t.Fatalf("Codex enabled tools should not include experimental report tool %q: %#v", experimental, tools)
		}
	}
	if containsString(tools, mcp.ToolMissionGet) {
		t.Fatalf("Codex enabled tools should not include legacy mission.get: %#v", tools)
	}
	if containsString(tools, mcp.ToolMissionUpdate) {
		t.Fatalf("Codex enabled tools should not let an agent impersonate a user metadata edit: %#v", tools)
	}
}

func TestCodexSharedDBPathRequiresFileBackedDatabase(t *testing.T) {
	for _, value := range []string{"", "   ", ":memory:"} {
		if _, err := codexSharedDBPath(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
	abs, err := codexSharedDBPath(filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatalf("codexSharedDBPath returned error: %v", err)
	}
	if !filepath.IsAbs(abs) {
		t.Fatalf("expected absolute path, got %q", abs)
	}
}

func TestCodexWorkDirDefaultsToTempDir(t *testing.T) {
	workDir, err := codexWorkDir("")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(workDir) {
		t.Fatalf("expected absolute workdir, got %q", workDir)
	}
	if !strings.Contains(workDir, "plasma-agent-workdir") {
		t.Fatalf("expected plasma temp workdir, got %q", workDir)
	}
	info, err := os.Stat(workDir)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatalf("expected directory, got %q", workDir)
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func createCLITestMission(t *testing.T, dbPath string) string {
	t.Helper()
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"missions", "create", "-db", dbPath, "-title", "CLI test mission", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("missions create returned %d stderr=%q", code, errOut.String())
	}
	return nestedCLIString(t, decodeCLIJSON(t, out.String()), "projection", "mission_id")
}

func decodeCLIJSON(t *testing.T, text string) map[string]any {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal([]byte(text), &decoded); err != nil {
		t.Fatalf("decode JSON %q: %v", text, err)
	}
	return decoded
}

func nestedCLIString(t *testing.T, value map[string]any, path ...string) string {
	t.Helper()
	var current any = value
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("expected object at %s in %#v", key, current)
		}
		current = object[key]
	}
	text, ok := current.(string)
	if !ok || text == "" {
		t.Fatalf("expected string at %v in %#v", path, current)
	}
	return text
}

func nestedCLIBool(t *testing.T, value map[string]any, path ...string) bool {
	t.Helper()
	var current any = value
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("expected object at %s in %#v", key, current)
		}
		current = object[key]
	}
	boolean, ok := current.(bool)
	if !ok {
		t.Fatalf("expected bool at %v in %#v", path, current)
	}
	return boolean
}

func cliLatestEventPayload(t *testing.T, events []app.LedgerEvent, eventType string) map[string]any {
	t.Helper()
	for index := len(events) - 1; index >= 0; index-- {
		if events[index].EventType != eventType {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(events[index].Payload, &payload); err != nil {
			t.Fatalf("decode %s payload: %v", eventType, err)
		}
		return payload
	}
	t.Fatalf("expected event type %s in %#v", eventType, events)
	return nil
}

func cliHumanizePatchFinalizer(t *testing.T, dbPath string, content string) func(web.AgentRequest) {
	t.Helper()
	return func(req web.AgentRequest) {
		if req.ReportPatch == nil {
			return
		}
		ctx := context.Background()
		store, err := sqlite.Open(ctx, dbPath)
		if err != nil {
			t.Errorf("open db for H5 patch finalize: %v", err)
			return
		}
		defer store.Close()
		svc := app.NewService(store)
		artifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
			ArtifactID: cliNewID("art"),
			MissionID:  req.MissionID,
			MediaType:  "text/markdown; charset=utf-8",
			Filename:   "humanized.md",
			Producer:   app.Producer{Type: "mcp_tool", ID: mcp.ToolReportPatchFinalize},
			Content:    []byte(content),
		})
		if err != nil {
			t.Errorf("create CLI H5 patch artifact: %v", err)
			return
		}
		if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
			EventID:       cliNewID("evt"),
			MissionID:     req.MissionID,
			EventType:     "report.patch.finalized",
			Producer:      app.Producer{Type: "mcp_tool", ID: mcp.ToolReportPatchFinalize},
			CorrelationID: req.ToolSessionID,
			Payload: cliMustJSON(t, map[string]any{
				"kind":                            "markdown_report_patch_finalized",
				"pending_event_id":                req.ReportPatch.PendingEventID,
				"title":                           "Humanized report",
				"artifact_id":                     artifact.ArtifactID,
				"media_type":                      artifact.MediaType,
				"base_artifact_id":                req.ReportPatch.BaseArtifactID,
				"agent_executor":                  req.ReportPatch.AgentExecutor,
				"agent_model":                     req.ReportPatch.AgentModel,
				"agent_reasoning_effort":          req.ReportPatch.AgentReasoningEffort,
				"agent_session_id":                req.ReportPatch.AgentSessionID,
				"previous_agent_session_id":       req.ReportPatch.PreviousAgentSessionID,
				"returned_agent_session_id":       req.ReportPatch.ReportSessionID,
				"report_session_id":               req.ReportPatch.ReportSessionID,
				"report_session_policy":           req.ReportPatch.ReportSessionPolicy,
				"report_session_policy_selection": req.ReportPatch.ReportSessionPolicySelection,
				"tool_session_id":                 req.ToolSessionID,
				"mcp_mode":                        req.ReportPatch.MCPMode,
				"composition_strategy":            "mcp_patch_markdown",
				"session_chain_kind":              req.ReportPatch.SessionChainKind,
			}),
		}); err != nil {
			t.Errorf("append CLI H5 patch finalized event: %v", err)
		}
	}
}

func assertCLINoRootPath(t *testing.T, rootPath string, values ...string) {
	t.Helper()
	for _, value := range values {
		if strings.Contains(value, rootPath) {
			t.Fatalf("CLI output leaked configured root path %q in %q", rootPath, value)
		}
	}
}

func hasCLIArgPair(args []string, flagName string, value string) bool {
	for index := 0; index+1 < len(args); index++ {
		if args[index] == flagName && args[index+1] == value {
			return true
		}
	}
	return false
}

func TestRunReportsDraftFreezesExplicitSelectionAndRejectsInvalidPair(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	missionID := createCLITestMission(t, dbPath)
	fake := &cliFakeAgent{responses: []web.AgentResult{{Text: "- Plan", SessionID: "report-session"}, {Text: "# Report", SessionID: "report-session"}}}
	oldFactory := newCLIAgentExecutor
	newCLIAgentExecutor = func(context.Context, cliAgentConfig) (web.AgentExecutor, error) { return fake, nil }
	t.Cleanup(func() { newCLIAgentExecutor = oldFactory })

	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"reports", "draft", missionID, "-db", dbPath, "-wait", "-agent-model", "gpt-5.5", "-agent-reasoning-effort", "high"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("explicit draft returned %d: %s", code, errOut.String())
	}
	if len(fake.requests) != 2 {
		t.Fatalf("expected plan/body requests, got %#v", fake.requests)
	}
	for _, req := range fake.requests {
		if req.Model != "gpt-5.5" || req.ReasoningEffort != "high" {
			t.Fatalf("selection not propagated: %#v", req)
		}
	}
	store, err := sqlite.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	events, err := app.NewService(store).ListEvents(context.Background(), missionID)
	if err != nil {
		t.Fatal(err)
	}
	for _, eventType := range []string{"report.draft.pending", "report.plan.created", "report.artifact.created"} {
		payload := cliLatestEventPayload(t, events, eventType)
		if payload["agent_model"] != "gpt-5.5" || payload["agent_reasoning_effort"] != "high" || payload["agent_selection_source"] != reporting.AgentSelectionSourceExplicitRequest {
			t.Fatalf("%s selection mismatch: %#v", eventType, payload)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	defaultMissionID := createCLITestMission(t, dbPath)
	defaultAgent := &cliFakeAgent{responses: []web.AgentResult{{Text: "- Plan", SessionID: "default-session"}, {Text: "# Report", SessionID: "default-session"}}}
	newCLIAgentExecutor = func(context.Context, cliAgentConfig) (web.AgentExecutor, error) { return defaultAgent, nil }
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"reports", "draft", defaultMissionID, "-db", dbPath, "-wait", "-agent-model", "gpt-5.5"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("model-default draft returned %d: %s", code, errOut.String())
	}
	for _, req := range defaultAgent.requests {
		if req.Model != "gpt-5.5" || req.ReasoningEffort != "medium" {
			t.Fatalf("model default mismatch: %#v", req)
		}
	}

	invalidMissionID := createCLITestMission(t, dbPath)
	invalidAgent := &cliForkingFakeAgent{forkSessionID: "fork"}
	newCLIAgentExecutor = func(context.Context, cliAgentConfig) (web.AgentExecutor, error) { return invalidAgent, nil }
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"reports", "draft", invalidMissionID, "-db", dbPath, "-wait", "-agent-model", "gpt-5.6-luna", "-agent-reasoning-effort", "ultra"}, &out, &errOut)
	if code != 2 || len(invalidAgent.requests) != 0 || len(invalidAgent.forkSources) != 0 {
		t.Fatalf("invalid pair side effects: code=%d requests=%#v forks=%#v stderr=%q", code, invalidAgent.requests, invalidAgent.forkSources, errOut.String())
	}
	store, err = sqlite.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	events, err = app.NewService(store).ListEvents(context.Background(), invalidMissionID)
	if err != nil {
		t.Fatal(err)
	}
	if cliCountEventType(events, "report.draft.pending") != 0 {
		t.Fatalf("invalid pair appended pending: %#v", events)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRunReportsDraftClaudeEmptyConfigFreezesHaiku(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	missionID := createCLITestMission(t, dbPath)
	fake := &cliFakeAgent{responses: []web.AgentResult{{Text: "- Plan", SessionID: "claude-session"}, {Text: "# Report", SessionID: "claude-session"}}}
	oldFactory := newCLIAgentExecutor
	newCLIAgentExecutor = func(_ context.Context, cfg cliAgentConfig) (web.AgentExecutor, error) {
		if cfg.AgentName != "claude" || cfg.ClaudeModel != "" {
			t.Fatalf("unexpected config: %#v", cfg)
		}
		return fake, nil
	}
	t.Cleanup(func() { newCLIAgentExecutor = oldFactory })
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"reports", "draft", missionID, "-db", dbPath, "-wait", "-agent", "claude"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("claude draft returned %d: %s", code, errOut.String())
	}
	for _, req := range fake.requests {
		if req.Model != "haiku" || req.ReasoningEffort != "" {
			t.Fatalf("claude request mismatch: %#v", req)
		}
	}
	store, err := sqlite.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	events, err := app.NewService(store).ListEvents(context.Background(), missionID)
	if err != nil {
		t.Fatal(err)
	}
	for _, eventType := range []string{"report.draft.pending", "report.plan.created", "report.artifact.created"} {
		payload := cliLatestEventPayload(t, events, eventType)
		if payload["agent_model"] != "haiku" || payload["agent_selection_source"] != reporting.AgentSelectionSourceProviderDefault {
			t.Fatalf("%s mismatch: %#v", eventType, payload)
		}
	}
}

type cliFakeAgent struct {
	requests   []web.AgentRequest
	responses  []web.AgentResult
	onRun      func(web.AgentRequest)
	onEveryRun func(web.AgentRequest)
}

func (agent *cliFakeAgent) Run(_ context.Context, req web.AgentRequest) (web.AgentResult, error) {
	agent.requests = append(agent.requests, req)
	if agent.onEveryRun != nil {
		agent.onEveryRun(req)
	}
	if agent.onRun != nil {
		onRun := agent.onRun
		agent.onRun = nil
		onRun(req)
	}
	if len(agent.responses) == 0 {
		return web.AgentResult{Text: "fake answer", SessionID: req.PreviousSessionID, Resumed: req.PreviousSessionID != ""}, nil
	}
	response := agent.responses[0]
	agent.responses = agent.responses[1:]
	if response.SessionID == "" {
		response.SessionID = req.PreviousSessionID
	}
	response.Resumed = req.PreviousSessionID != ""
	return response, nil
}

type cliForkingFakeAgent struct {
	cliFakeAgent
	forkSessionID string
	forkSources   []string
}

func (agent *cliForkingFakeAgent) ForkSession(_ context.Context, sourceSessionID string) (web.AgentSessionForkResult, error) {
	agent.forkSources = append(agent.forkSources, sourceSessionID)
	sessionID := strings.TrimSpace(agent.forkSessionID)
	if sessionID == "" {
		sessionID = "forked-session"
	}
	return web.AgentSessionForkResult{
		SessionID:       sessionID,
		SourceSessionID: strings.TrimSpace(sourceSessionID),
	}, nil
}

func (agent *cliForkingFakeAgent) CheckForkSession(_ context.Context, sourceSessionID string) error {
	if strings.TrimSpace(sourceSessionID) == "" {
		return errors.New("source session id is required")
	}
	return nil
}

func cliCountEventType(events []app.LedgerEvent, eventType string) int {
	count := 0
	for _, event := range events {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}

func cliMustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal JSON: %v", err)
	}
	return encoded
}

func cliEventTypes(events []app.LedgerEvent) []string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.EventType)
	}
	return types
}

func assertCLIEventTypeOrder(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected event types %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected event types %v, got %v", want, got)
		}
	}
}

func cliCountWorkflowSteeringTurns(t *testing.T, events []app.LedgerEvent) int {
	t.Helper()
	count := 0
	for _, event := range events {
		if event.EventType != "turn.user" {
			continue
		}
		var payload struct {
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("decode turn.user payload: %v", err)
		}
		if payload.Kind == "workflow_steering" {
			count++
		}
	}
	return count
}
