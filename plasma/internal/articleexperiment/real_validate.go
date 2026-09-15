package articleexperiment

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

const (
	maxRealContractBytes = 16 << 20
	pilotProvider        = "codex"
	pilotModel           = "gpt-5.6-luna"
	pilotReasoningEffort = "xhigh"
)

// LoadRealProtocol validates a complete provider-ready pilot bundle without
// executing any provider or writing product state. A separate live Issue-lock
// verifier must validate Protocol.IssueLock before generation begins.
func LoadRealProtocol(archiveRoot, repositoryRoot, protocolPath, expectedSHA string) (LoadedRealProtocol, error) {
	archiveRoot, repositoryRoot, err := prepareArchiveRoot(archiveRoot, repositoryRoot)
	if err != nil {
		return LoadedRealProtocol{}, err
	}
	archiveInfo, err := os.Stat(archiveRoot)
	if err != nil {
		return LoadedRealProtocol{}, err
	}
	if archiveInfo.Mode().Perm() != 0o700 {
		return LoadedRealProtocol{}, fmt.Errorf("%w: real archive mode must be 0700", producterror.ErrInvalidInput)
	}
	protocolPath, err = resolveInput(archiveRoot, repositoryRoot, archiveRoot, protocolPath)
	if err != nil {
		return LoadedRealProtocol{}, err
	}
	protocol, raw, err := loadJSONFile[RealProtocol](protocolPath)
	if err != nil {
		return LoadedRealProtocol{}, err
	}
	protocolInfo, err := os.Stat(protocolPath)
	if err != nil {
		return LoadedRealProtocol{}, err
	}
	if protocolInfo.Mode().Perm() != 0o600 {
		return LoadedRealProtocol{}, fmt.Errorf("%w: real protocol file mode must be 0600", producterror.ErrInvalidInput)
	}
	protocolSHA := bytesSHA256(raw)
	if !validSHA256(expectedSHA) || protocolSHA != expectedSHA {
		return LoadedRealProtocol{}, fmt.Errorf("%w: real protocol SHA-256 mismatch", producterror.ErrConflict)
	}
	if err := validateRealProtocolEnvelope(protocol); err != nil {
		return LoadedRealProtocol{}, err
	}
	loaded := LoadedRealProtocol{
		Protocol: protocol, ProtocolSHA256: protocolSHA,
		fixtures: map[string]RealFixture{}, fixtureSHA256: map[string]string{},
		arms: map[string]RealArm{}, armSHA256: map[string]string{},
	}
	baseDir := filepath.Dir(protocolPath)
	for _, ref := range protocol.Fixtures {
		path, content, err := loadRealReference(archiveRoot, repositoryRoot, baseDir, ref)
		if err != nil {
			return LoadedRealProtocol{}, err
		}
		fixture, _, err := decodeJSONBytes[RealFixture](filepath.Base(path), content)
		if err != nil {
			return LoadedRealProtocol{}, err
		}
		if fixture.FixtureID != ref.ID || loaded.fixtureSHA256[ref.ID] != "" {
			return LoadedRealProtocol{}, fmt.Errorf("%w: real fixture identity is invalid", producterror.ErrInvalidInput)
		}
		if err := validateRealFixture(archiveRoot, repositoryRoot, filepath.Dir(path), fixture); err != nil {
			return LoadedRealProtocol{}, err
		}
		loaded.fixtures[ref.ID] = fixture
		loaded.fixtureSHA256[ref.ID] = ref.SHA256
	}
	for _, ref := range protocol.Arms {
		path, content, err := loadRealReference(archiveRoot, repositoryRoot, baseDir, ref)
		if err != nil {
			return LoadedRealProtocol{}, err
		}
		arm, _, err := decodeJSONBytes[RealArm](filepath.Base(path), content)
		if err != nil {
			return LoadedRealProtocol{}, err
		}
		if arm.ArmID != ref.ID || loaded.armSHA256[ref.ID] != "" {
			return LoadedRealProtocol{}, fmt.Errorf("%w: real arm identity is invalid", producterror.ErrInvalidInput)
		}
		if err := validateRealArm(archiveRoot, repositoryRoot, filepath.Dir(path), arm); err != nil {
			return LoadedRealProtocol{}, err
		}
		loaded.arms[ref.ID] = arm
		loaded.armSHA256[ref.ID] = ref.SHA256
	}
	blindPath, blindContent, err := loadRealReference(archiveRoot, repositoryRoot, baseDir, protocol.Blind)
	if err != nil {
		return LoadedRealProtocol{}, err
	}
	blind, _, err := decodeJSONBytes[BlindContract](filepath.Base(blindPath), blindContent)
	if err != nil {
		return LoadedRealProtocol{}, err
	}
	if err := validateBlindContract(archiveRoot, repositoryRoot, filepath.Dir(blindPath), protocol, blind); err != nil {
		return LoadedRealProtocol{}, err
	}
	loaded.Blind = blind
	loaded.BlindSHA256 = protocol.Blind.SHA256
	if err := validateRealMatrix(loaded); err != nil {
		return LoadedRealProtocol{}, err
	}
	return loaded, nil
}

func validGitRevision(value string) bool {
	return len(value) == 40 && strings.Trim(value, "0123456789abcdef") == ""
}

func nonzeroSHA256(value string) bool {
	return validSHA256(value) && strings.Trim(value, "0") != ""
}

func validateRealProtocolEnvelope(protocol RealProtocol) error {
	_, createdErr := time.Parse(time.RFC3339Nano, protocol.CreatedAt)
	if protocol.SchemaVersion != RealProtocolSchemaVersion || !safeSlugPattern.MatchString(protocol.ProtocolID) || createdErr != nil || !validGitRevision(protocol.CodeRevision) || strings.Trim(protocol.CodeRevision, "0") == "" || len(protocol.Fixtures) != 3 || len(protocol.Arms) != 3 || len(protocol.Runs) != 9 {
		return fmt.Errorf("%w: real protocol envelope is invalid", producterror.ErrInvalidInput)
	}
	if protocol.IssueLock.Repository != "c86j224s/liquid2" || protocol.IssueLock.Issue != 461 || protocol.IssueLock.Owner != "c86j224s" || !protocol.IssueLock.Required {
		return fmt.Errorf("%w: Issue 461 owner lock is invalid", producterror.ErrInvalidInput)
	}
	if protocol.Isolation != (IsolationContract{
		ArchiveMode: "0700", FileMode: "0600", DedicatedArchive: true, SingleWriter: true,
		IsolatedHome: true, IsolatedProviderConfig: true, IsolatedWorkspace: true,
		IsolatedDatabase: true, EnvironmentAllowlist: true,
		RetainRawProviderLog: false, RetainProviderSession: false,
	}) {
		return fmt.Errorf("%w: real pilot isolation contract is invalid", producterror.ErrInvalidInput)
	}
	if protocol.FailurePolicy != (RealFailurePolicy{
		SemanticAttemptsPerCell: 1, AllowReplacementRuns: false, AllowMatrixResume: false,
		AnyUnusableCellOutcome: "INCONCLUSIVE", ProtocolChangeOutcome: "INVALID",
	}) {
		return fmt.Errorf("%w: real pilot failure policy is invalid", producterror.ErrInvalidInput)
	}
	if err := validateMatrixBudget(protocol.MatrixBudget); err != nil {
		return err
	}
	return nil
}

func validateRealFixture(archiveRoot, repositoryRoot, baseDir string, fixture RealFixture) error {
	if fixture.SchemaVersion != RealFixtureSchemaVersion || fixture.Language != "ko" || strings.TrimSpace(fixture.Audience) == "" || strings.TrimSpace(fixture.ReaderPromise) == "" || fixture.MinimumBodyCharacters != 7000 || fixture.MaximumBodyCharacters != 11000 || fixture.TargetBodyCharacters < fixture.MinimumBodyCharacters || fixture.TargetBodyCharacters > fixture.MaximumBodyCharacters || fixture.MaterialTruthClaims < 8 || fixture.MaterialTruthClaims > 12 || fixture.MaterialCaveats < 2 || fixture.MaterialCaveats > 3 || fixture.SupportedConnections < 1 {
		return fmt.Errorf("%w: real fixture contract is invalid", producterror.ErrInvalidInput)
	}
	switch fixture.FixtureID {
	case "M1":
		if fixture.ValueType != "method_how_to" {
			return fmt.Errorf("%w: M1 value type is invalid", producterror.ErrInvalidInput)
		}
	case "M2":
		if fixture.ValueType != "mechanism_insight" {
			return fmt.Errorf("%w: M2 value type is invalid", producterror.ErrInvalidInput)
		}
	case "M3":
		if fixture.ValueType != "discovery_context" {
			return fmt.Errorf("%w: M3 value type is invalid", producterror.ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: real fixture ID is invalid", producterror.ErrInvalidInput)
	}
	refs := []FileRef{fixture.SourceCatalog, fixture.Dossier, fixture.ClaimInventory, fixture.MemoryQuestions, fixture.TransferTask}
	seen := []os.FileInfo{}
	for _, ref := range refs {
		path, _, err := loadRealReference(archiveRoot, repositoryRoot, baseDir, ref)
		if err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		for _, prior := range seen {
			if os.SameFile(prior, info) {
				return fmt.Errorf("%w: real fixture file is duplicated", producterror.ErrInvalidInput)
			}
		}
		seen = append(seen, info)
	}
	return ValidateFixtureArtifacts(archiveRoot, repositoryRoot, baseDir, fixture)
}

func validateRealArm(archiveRoot, repositoryRoot, baseDir string, arm RealArm) error {
	if arm.SchemaVersion != RealArmSchemaVersion || len(arm.Stages) == 0 {
		return fmt.Errorf("%w: real arm envelope is invalid", producterror.ErrInvalidInput)
	}
	switch arm.ArmID {
	case ArmReport:
		if arm.Role != "contextual_report_il" || !arm.ContextualOnly || arm.Treatment.Kind != "report_product" {
			return fmt.Errorf("%w: R arm role is invalid", producterror.ErrInvalidInput)
		}
	case ArmControl:
		if arm.Role != "matched_control" || arm.ContextualOnly || arm.Treatment.Kind != "none" {
			return fmt.Errorf("%w: E arm treatment is invalid", producterror.ErrInvalidInput)
		}
	case ArmArticle:
		if arm.Role != "article_narrative" || arm.ContextualOnly || arm.Treatment.Kind != "article_narrative" {
			return fmt.Errorf("%w: A arm treatment is invalid", producterror.ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: real arm ID is invalid", producterror.ErrInvalidInput)
	}
	provider := arm.Provider
	if provider.Executor != pilotProvider || provider.Model != pilotModel || provider.ReasoningEffort != pilotReasoningEffort || !provider.FreshEphemeralSessions || !provider.IgnoreUserConfig || !provider.ReplaceMCPTools || provider.AllowAmbientWeb || provider.AllowAmbientFiles {
		return fmt.Errorf("%w: real arm provider contract is invalid", producterror.ErrInvalidInput)
	}
	if arm.ArmID == ArmReport {
		if err := validateReportStages(arm.Stages); err != nil {
			return err
		}
	} else {
		if len(arm.Stages) != len(requiredRealStages) {
			return fmt.Errorf("%w: real E/A stage count is invalid", producterror.ErrInvalidInput)
		}
		for index, stage := range arm.Stages {
			if stage.Stage != requiredRealStages[index] || stage.MaxAttempts < 1 || stage.MaxAttempts > 2 {
				return fmt.Errorf("%w: real arm stage contract is invalid", producterror.ErrInvalidInput)
			}
			wantAccess := "read_only"
			if stage.Stage == "author" || stage.Stage == "conditional_author_repair" || stage.Stage == "render" {
				wantAccess = "author_or_server"
			}
			if stage.Access != wantAccess {
				return fmt.Errorf("%w: real arm stage access is invalid", producterror.ErrInvalidInput)
			}
		}
	}
	if err := validateCellBudget(arm.Budget); err != nil {
		return err
	}
	refs := []FileRef{arm.Common.AuthorPrompt, arm.Common.ReaderPrompt, arm.Common.AuditorPrompt, arm.Common.RepairPrompt, arm.Common.ToolPolicy, arm.Common.OutputSchema, arm.Common.Renderer, arm.Treatment.Contract}
	seen := []os.FileInfo{}
	for _, ref := range refs {
		path, _, err := loadRealReference(archiveRoot, repositoryRoot, baseDir, ref)
		if err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		for _, prior := range seen {
			if os.SameFile(prior, info) {
				return fmt.Errorf("%w: real arm contract file is duplicated", producterror.ErrInvalidInput)
			}
		}
		seen = append(seen, info)
	}
	wantArtifacts := []ArtifactContract{
		{Kind: "article_il", MediaType: "application/json", Filename: "article-il.json", MaxByteSize: maxOutputBytes, RequireUTF8: true},
		{Kind: "article_markdown", MediaType: "text/markdown; charset=utf-8", Filename: "article.md", MaxByteSize: maxOutputBytes, RequireUTF8: true},
		{Kind: "article_html", MediaType: "text/html; charset=utf-8", Filename: "article.html", MaxByteSize: maxOutputBytes, RequireUTF8: true},
		{Kind: "article_pdf", MediaType: "application/pdf", Filename: "article.pdf", MaxByteSize: maxOutputBytes, RequireUTF8: false},
		{Kind: "usage_receipt", MediaType: "application/json", Filename: "usage.json", MaxByteSize: maxContractBytes, RequireUTF8: true},
		{Kind: "audit_receipt", MediaType: "application/json", Filename: "audit.json", MaxByteSize: maxContractBytes, RequireUTF8: true},
	}
	if arm.ArmID == ArmReport {
		wantArtifacts = []ArtifactContract{
			{Kind: "report_markdown", MediaType: "text/markdown; charset=utf-8", Filename: "report.md", MaxByteSize: maxOutputBytes, RequireUTF8: true},
			{Kind: "report_html", MediaType: "text/html; charset=utf-8", Filename: "report.html", MaxByteSize: maxOutputBytes, RequireUTF8: true},
			{Kind: "report_pdf", MediaType: "application/pdf", Filename: "report.pdf", MaxByteSize: maxOutputBytes, RequireUTF8: false},
			{Kind: "usage_receipt", MediaType: "application/json", Filename: "usage.json", MaxByteSize: maxContractBytes, RequireUTF8: true},
			{Kind: "audit_receipt", MediaType: "application/json", Filename: "audit.json", MaxByteSize: maxContractBytes, RequireUTF8: true},
		}
	}
	if !reflect.DeepEqual(arm.Artifacts, wantArtifacts) {
		return fmt.Errorf("%w: real arm artifact allowlist is invalid", producterror.ErrInvalidInput)
	}
	return nil
}

func validateReportStages(stages []StageContract) error {
	if len(stages) != 1 || stages[0] != (StageContract{Stage: "report_product", Access: "product_or_server", MaxAttempts: 1}) {
		return fmt.Errorf("%w: contextual R stage contract is invalid", producterror.ErrInvalidInput)
	}
	return nil
}

func validateCellBudget(budget CellBudget) error {
	if budget.MaxProviderCalls < 1 || budget.MaxProviderCalls > 40 || budget.MaxTechnicalRetries < 0 || budget.MaxTechnicalRetries > 1 || budget.MaxRepairBatches != 1 || budget.MaxDurationMilliseconds < 1 || budget.MaxDurationMilliseconds > 2*60*60*1000 || budget.MaxSourceBytesPerCall < 1 || budget.MaxSourceBytesPerCall > 64<<10 || budget.MaxSourceBytesPerAttempt < budget.MaxSourceBytesPerCall || budget.MaxSourceBytesPerAttempt > 512<<10 || budget.MaxTotalTokens < 1 || budget.MaxTotalTokens > 2_000_000 || budget.MaxInputTokens < 1 || budget.MaxOutputTokens < 1 || budget.MaxInputTokens > budget.MaxTotalTokens-budget.MaxOutputTokens || budget.UnknownUsagePolicy != "fail_closed" {
		return fmt.Errorf("%w: real cell budget is invalid", producterror.ErrInvalidInput)
	}
	return nil
}

func validateMatrixBudget(budget MatrixBudget) error {
	if budget.MaxCells != 9 || budget.MaxProviderCalls < 9 || budget.MaxProviderCalls > 360 || budget.MaxWallTimeMilliseconds < 1 || budget.MaxWallTimeMilliseconds > 12*60*60*1000 || budget.MaxTotalTokens < 1 || budget.MaxTotalTokens > 18_000_000 || budget.MaxInputTokens < 1 || budget.MaxOutputTokens < 1 || budget.MaxInputTokens > budget.MaxTotalTokens-budget.MaxOutputTokens || budget.OnCeiling != "stop_new_cells" || budget.UnknownUsagePolicy != "fail_closed" || budget.BudgetAccountingVersion != "agentusage.increment.v1" {
		return fmt.Errorf("%w: real matrix budget is invalid", producterror.ErrInvalidInput)
	}
	return nil
}

func validateBlindContract(archiveRoot, repositoryRoot, baseDir string, protocol RealProtocol, blind BlindContract) error {
	if blind.SchemaVersion != BlindSchemaVersion || !reflect.DeepEqual(blind.PrimaryArms, []string{ArmControl, ArmArticle}) || blind.ContextualArm != ArmReport || !nonzeroSHA256(blind.MappingCommitmentSHA256) || !nonzeroSHA256(blind.RandomizationSeedSHA256) || len(blind.GenerationOrder) != 9 || len(blind.Pairs) != 3 || strings.TrimSpace(blind.ArmGuessQuestion) == "" || !blind.RevealAfterUserJudgment || !blind.RevealAfterFactualAudits || !blind.NativeLaneAfterReveal {
		return fmt.Errorf("%w: blind contract is invalid", producterror.ErrInvalidInput)
	}
	runIDs := map[string]bool{}
	for _, cell := range protocol.Runs {
		runIDs[cell.RunID] = true
	}
	seenOrder := map[string]bool{}
	for _, runID := range blind.GenerationOrder {
		if !runIDs[runID] || seenOrder[runID] {
			return fmt.Errorf("%w: blind generation order is invalid", producterror.ErrInvalidInput)
		}
		seenOrder[runID] = true
	}
	seenFixtures := map[string]bool{}
	seenPackets := map[string]bool{}
	for _, pair := range blind.Pairs {
		if !containsString(requiredFixtureIDs, pair.FixtureID) || seenFixtures[pair.FixtureID] || !safeSlugPattern.MatchString(pair.PacketID) || seenPackets[pair.PacketID] || len(pair.Slots) != 2 || pair.Slots[0] == pair.Slots[1] || !safeSlugPattern.MatchString(pair.Slots[0]) || !safeSlugPattern.MatchString(pair.Slots[1]) {
			return fmt.Errorf("%w: blind pair is invalid", producterror.ErrInvalidInput)
		}
		seenFixtures[pair.FixtureID] = true
		seenPackets[pair.PacketID] = true
	}
	for _, ref := range []FileRef{blind.Rubric, blind.ReaderShell, blind.Administration} {
		if _, _, err := loadRealReference(archiveRoot, repositoryRoot, baseDir, ref); err != nil {
			return err
		}
	}
	return nil
}

func validateRealMatrix(loaded LoadedRealProtocol) error {
	for _, fixtureID := range requiredFixtureIDs {
		if loaded.fixtureSHA256[fixtureID] == "" {
			return fmt.Errorf("%w: real fixture matrix is incomplete", producterror.ErrInvalidInput)
		}
	}
	for _, armID := range requiredArmIDs {
		if loaded.armSHA256[armID] == "" {
			return fmt.Errorf("%w: real arm matrix is incomplete", producterror.ErrInvalidInput)
		}
	}
	seenRuns := map[string]bool{}
	seenCells := map[string]bool{}
	for _, cell := range loaded.Protocol.Runs {
		key := cell.FixtureID + "/" + cell.ArmID
		if loaded.fixtureSHA256[cell.FixtureID] == "" || loaded.armSHA256[cell.ArmID] == "" || !safeSlugPattern.MatchString(cell.RunID) || strings.Contains(cell.RunID, "..") || seenRuns[cell.RunID] || seenCells[key] {
			return fmt.Errorf("%w: real run matrix is invalid", producterror.ErrInvalidInput)
		}
		seenRuns[cell.RunID] = true
		seenCells[key] = true
	}
	for _, fixtureID := range requiredFixtureIDs {
		for _, armID := range requiredArmIDs {
			if !seenCells[fixtureID+"/"+armID] {
				return fmt.Errorf("%w: real run matrix is incomplete", producterror.ErrInvalidInput)
			}
		}
	}
	control := loaded.arms[ArmControl]
	article := loaded.arms[ArmArticle]
	if !reflect.DeepEqual(control.Provider, article.Provider) || !reflect.DeepEqual(control.Stages, article.Stages) || !reflect.DeepEqual(control.Common, article.Common) || !reflect.DeepEqual(control.Budget, article.Budget) || !reflect.DeepEqual(control.Artifacts, article.Artifacts) || control.Treatment.Contract.SHA256 == article.Treatment.Contract.SHA256 {
		return fmt.Errorf("%w: E/A matched contract differs outside treatment or treatment is duplicated", producterror.ErrInvalidInput)
	}
	return nil
}

func loadRealReference(archiveRoot, repositoryRoot, baseDir string, ref FileRef) (string, []byte, error) {
	if !safeSlugPattern.MatchString(ref.ID) || !validSHA256(ref.SHA256) {
		return "", nil, fmt.Errorf("%w: real file reference is invalid", producterror.ErrInvalidInput)
	}
	path, err := resolveReference(archiveRoot, repositoryRoot, baseDir, ref.Path)
	if err != nil {
		return "", nil, err
	}
	content, err := readRegularFile(path, maxRealContractBytes)
	if err != nil {
		return "", nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", nil, err
	}
	if info.Mode().Perm() != 0o600 {
		return "", nil, fmt.Errorf("%w: real contract file mode must be 0600", producterror.ErrInvalidInput)
	}
	if bytesSHA256(content) != ref.SHA256 {
		return "", nil, fmt.Errorf("%w: real file reference SHA-256 mismatch", producterror.ErrConflict)
	}
	return path, content, nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
