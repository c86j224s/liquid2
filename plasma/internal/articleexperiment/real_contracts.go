package articleexperiment

const (
	RealProtocolSchemaVersion = "plasma.article_experiment.real_protocol.v1"
	RealFixtureSchemaVersion  = "plasma.article_experiment.real_fixture.v1"
	RealArmSchemaVersion      = "plasma.article_experiment.real_arm.v1"
	BlindSchemaVersion        = "plasma.article_experiment.blind.v1"
)

var requiredRealStages = []string{
	"author",
	"reader_diagnosis",
	"factual_audit",
	"conditional_author_repair",
	"confirmation",
	"render",
}

// RealProtocol freezes every input and operating limit for one real nine-cell pilot.
type RealProtocol struct {
	SchemaVersion string            `json:"schema_version"`
	ProtocolID    string            `json:"protocol_id"`
	CreatedAt     string            `json:"created_at"`
	CodeRevision  string            `json:"code_revision"`
	IssueLock     IssueLockContract `json:"issue_lock"`
	Fixtures      []FileRef         `json:"fixtures"`
	Arms          []FileRef         `json:"arms"`
	Runs          []RunCell         `json:"runs"`
	Blind         FileRef           `json:"blind"`
	Isolation     IsolationContract `json:"isolation"`
	MatrixBudget  MatrixBudget      `json:"matrix_budget"`
	FailurePolicy RealFailurePolicy `json:"failure_policy"`
}

// IssueLockContract identifies the owner-authored pre-run digest record required by the pilot.
type IssueLockContract struct {
	Repository string `json:"repository"`
	Issue      int    `json:"issue"`
	Owner      string `json:"owner"`
	Required   bool   `json:"required"`
}

// LiveIssueComment is fetched from GitHub by the command adapter.
type LiveIssueComment struct {
	URL       string
	ID        string
	Author    string
	CreatedAt string
	Body      []byte
}

// IssueLockReceipt is written only after the protocol digest is posted, avoiding
// a circular protocol-to-comment hash dependency.
type IssueLockReceipt struct {
	Repository        string            `json:"repository"`
	Issue             int               `json:"issue"`
	Owner             string            `json:"owner"`
	CommentURL        string            `json:"comment_url"`
	CommentID         string            `json:"comment_id"`
	CommentCreatedAt  string            `json:"comment_created_at"`
	CommentBodySHA256 string            `json:"comment_body_sha256"`
	ProtocolSHA256    string            `json:"protocol_sha256"`
	FixtureSHA256     map[string]string `json:"fixture_sha256"`
	ArmSHA256         map[string]string `json:"arm_sha256"`
	BlindSHA256       string            `json:"blind_sha256"`
}

// RealFixture freezes one source-grounded reader-value test case.
type RealFixture struct {
	SchemaVersion         string  `json:"schema_version"`
	FixtureID             string  `json:"fixture_id"`
	ValueType             string  `json:"value_type"`
	Audience              string  `json:"audience"`
	ReaderPromise         string  `json:"reader_promise"`
	Emphasis              string  `json:"emphasis,omitempty"`
	Language              string  `json:"language"`
	TargetBodyCharacters  int     `json:"target_body_characters"`
	MinimumBodyCharacters int     `json:"minimum_body_characters"`
	MaximumBodyCharacters int     `json:"maximum_body_characters"`
	MaterialTruthClaims   int     `json:"material_truth_claims"`
	MaterialCaveats       int     `json:"material_caveats"`
	SupportedConnections  int     `json:"supported_connections"`
	SourceCatalog         FileRef `json:"source_catalog"`
	Dossier               FileRef `json:"dossier"`
	ClaimInventory        FileRef `json:"claim_inventory"`
	MemoryQuestions       FileRef `json:"memory_questions"`
	TransferTask          FileRef `json:"transfer_task"`
}

// RealArm freezes one provider topology and its only allowed treatment payload.
type RealArm struct {
	SchemaVersion  string             `json:"schema_version"`
	ArmID          string             `json:"arm_id"`
	Role           string             `json:"role"`
	ContextualOnly bool               `json:"contextual_only"`
	Provider       ProviderContract   `json:"provider"`
	Stages         []StageContract    `json:"stages"`
	Common         ArmCommonContract  `json:"common"`
	Treatment      TreatmentContract  `json:"treatment"`
	Budget         CellBudget         `json:"budget"`
	Artifacts      []ArtifactContract `json:"artifacts"`
}

// ProviderContract fixes provider identity and fresh-session behavior.
type ProviderContract struct {
	Executor               string `json:"executor"`
	Model                  string `json:"model"`
	ReasoningEffort        string `json:"reasoning_effort"`
	FreshEphemeralSessions bool   `json:"fresh_ephemeral_sessions"`
	IgnoreUserConfig       bool   `json:"ignore_user_config"`
	ReplaceMCPTools        bool   `json:"replace_mcp_tools"`
	AllowAmbientWeb        bool   `json:"allow_ambient_web"`
	AllowAmbientFiles      bool   `json:"allow_ambient_files"`
}

// StageContract fixes provider role order and mutation authority.
type StageContract struct {
	Stage       string `json:"stage"`
	Access      string `json:"access"`
	MaxAttempts int    `json:"max_attempts"`
}

// ArmCommonContract contains all E/A bytes that must remain identical.
type ArmCommonContract struct {
	AuthorPrompt  FileRef `json:"author_prompt"`
	ReaderPrompt  FileRef `json:"reader_prompt"`
	AuditorPrompt FileRef `json:"auditor_prompt"`
	RepairPrompt  FileRef `json:"repair_prompt"`
	ToolPolicy    FileRef `json:"tool_policy"`
	OutputSchema  FileRef `json:"output_schema"`
	Renderer      FileRef `json:"renderer"`
}

// TreatmentContract is the sole planned semantic difference between E and A.
type TreatmentContract struct {
	Kind     string  `json:"kind"`
	Contract FileRef `json:"contract"`
}

// CellBudget bounds one cell before and after provider execution.
type CellBudget struct {
	MaxProviderCalls         int    `json:"max_provider_calls"`
	MaxTechnicalRetries      int    `json:"max_technical_retries"`
	MaxRepairBatches         int    `json:"max_repair_batches"`
	MaxDurationMilliseconds  int64  `json:"max_duration_milliseconds"`
	MaxSourceBytesPerCall    int64  `json:"max_source_bytes_per_call"`
	MaxSourceBytesPerAttempt int64  `json:"max_source_bytes_per_attempt"`
	MaxTotalTokens           int64  `json:"max_total_tokens"`
	MaxInputTokens           int64  `json:"max_input_tokens"`
	MaxOutputTokens          int64  `json:"max_output_tokens"`
	UnknownUsagePolicy       string `json:"unknown_usage_policy"`
}

// MatrixBudget prevents new cells from starting after the frozen aggregate ceiling.
type MatrixBudget struct {
	MaxCells                int    `json:"max_cells"`
	MaxProviderCalls        int    `json:"max_provider_calls"`
	MaxWallTimeMilliseconds int64  `json:"max_wall_time_milliseconds"`
	MaxTotalTokens          int64  `json:"max_total_tokens"`
	MaxInputTokens          int64  `json:"max_input_tokens"`
	MaxOutputTokens         int64  `json:"max_output_tokens"`
	OnCeiling               string `json:"on_ceiling"`
	UnknownUsagePolicy      string `json:"unknown_usage_policy"`
	BudgetAccountingVersion string `json:"budget_accounting_version"`
}

// IsolationContract fixes the private workspace and tool boundary.
type IsolationContract struct {
	ArchiveMode            string `json:"archive_mode"`
	FileMode               string `json:"file_mode"`
	DedicatedArchive       bool   `json:"dedicated_archive"`
	SingleWriter           bool   `json:"single_writer"`
	IsolatedHome           bool   `json:"isolated_home"`
	IsolatedProviderConfig bool   `json:"isolated_provider_config"`
	IsolatedWorkspace      bool   `json:"isolated_workspace"`
	IsolatedDatabase       bool   `json:"isolated_database"`
	EnvironmentAllowlist   bool   `json:"environment_allowlist"`
	RetainRawProviderLog   bool   `json:"retain_raw_provider_log"`
	RetainProviderSession  bool   `json:"retain_provider_session"`
}

// ArtifactContract is the exact output allowlist for one arm.
type ArtifactContract struct {
	Kind        string `json:"kind"`
	MediaType   string `json:"media_type"`
	Filename    string `json:"filename"`
	MaxByteSize int64  `json:"max_byte_size"`
	RequireUTF8 bool   `json:"require_utf8"`
}

// RealFailurePolicy prevents favorable replacement runs.
type RealFailurePolicy struct {
	SemanticAttemptsPerCell int    `json:"semantic_attempts_per_cell"`
	AllowReplacementRuns    bool   `json:"allow_replacement_runs"`
	AllowMatrixResume       bool   `json:"allow_matrix_resume"`
	AnyUnusableCellOutcome  string `json:"any_unusable_cell_outcome"`
	ProtocolChangeOutcome   string `json:"protocol_change_outcome"`
}

// BlindContract freezes the primary A/E lane before generation.
type BlindContract struct {
	SchemaVersion            string      `json:"schema_version"`
	PrimaryArms              []string    `json:"primary_arms"`
	ContextualArm            string      `json:"contextual_arm"`
	MappingCommitmentSHA256  string      `json:"mapping_commitment_sha256"`
	RandomizationSeedSHA256  string      `json:"randomization_seed_sha256"`
	GenerationOrder          []string    `json:"generation_order"`
	Pairs                    []BlindPair `json:"pairs"`
	Rubric                   FileRef     `json:"rubric"`
	ReaderShell              FileRef     `json:"reader_shell"`
	Administration           FileRef     `json:"administration"`
	ArmGuessQuestion         string      `json:"arm_guess_question"`
	RevealAfterUserJudgment  bool        `json:"reveal_after_user_judgment"`
	RevealAfterFactualAudits bool        `json:"reveal_after_factual_audits"`
	NativeLaneAfterReveal    bool        `json:"native_lane_after_reveal"`
}

// BlindPair names one anonymous packet pair without exposing its private mapping.
type BlindPair struct {
	FixtureID string   `json:"fixture_id"`
	PacketID  string   `json:"packet_id"`
	Slots     []string `json:"slots"`
}

// LoadedRealProtocol is the validated immutable pre-run bundle.
type LoadedRealProtocol struct {
	Protocol       RealProtocol
	ProtocolSHA256 string
	Blind          BlindContract
	BlindSHA256    string
	fixtures       map[string]RealFixture
	fixtureSHA256  map[string]string
	arms           map[string]RealArm
	armSHA256      map[string]string
}

func (loaded LoadedRealProtocol) FixtureCount() int {
	return len(loaded.fixtures)
}

func (loaded LoadedRealProtocol) ArmCount() int {
	return len(loaded.arms)
}

func (loaded LoadedRealProtocol) FixtureDigests() map[string]string {
	return copyStringMap(loaded.fixtureSHA256)
}

func (loaded LoadedRealProtocol) ArmDigests() map[string]string {
	return copyStringMap(loaded.armSHA256)
}

// Fixture returns one validated fixture value by ID.
func (loaded LoadedRealProtocol) Fixture(id string) (RealFixture, bool) {
	fixture, ok := loaded.fixtures[id]
	return fixture, ok
}

// Arm returns one validated arm value by ID.
func (loaded LoadedRealProtocol) Arm(id string) (RealArm, bool) {
	arm, ok := loaded.arms[id]
	arm.Stages = append([]StageContract(nil), arm.Stages...)
	arm.Artifacts = append([]ArtifactContract(nil), arm.Artifacts...)
	return arm, ok
}

func copyStringMap(source map[string]string) map[string]string {
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
