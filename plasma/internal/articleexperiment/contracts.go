package articleexperiment

import (
	"context"
	"time"
)

const (
	ProtocolSchemaVersion = "plasma.article_experiment.protocol.v1"
	FixtureSchemaVersion  = "plasma.article_experiment.fixture.v1"
	ArmSchemaVersion      = "plasma.article_experiment.arm.v1"
	PendingSchemaVersion  = "plasma.article_experiment.run_pending.v1"
	TerminalSchemaVersion = "plasma.article_experiment.run_terminal.v2"
	MatrixSchemaVersion   = "plasma.article_experiment.matrix_terminal.v3"

	ArmReport  = "R"
	ArmControl = "E"
	ArmArticle = "A"
)

var requiredFixtureIDs = []string{"M1", "M2", "M3"}
var requiredArmIDs = []string{ArmReport, ArmControl, ArmArticle}

// Protocol freezes the complete three-fixture, three-arm preflight matrix.
type Protocol struct {
	SchemaVersion string    `json:"schema_version"`
	ProtocolID    string    `json:"protocol_id"`
	CreatedAt     string    `json:"created_at"`
	Fixtures      []FileRef `json:"fixtures"`
	Arms          []FileRef `json:"arms"`
	Runs          []RunCell `json:"runs"`
}

// FileRef binds an archive-relative contract or input file to exact bytes.
type FileRef struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// RunCell assigns one immutable run identity to one fixture and one arm.
type RunCell struct {
	FixtureID string `json:"fixture_id"`
	ArmID     string `json:"arm_id"`
	RunID     string `json:"run_id"`
}

// Fixture freezes the reader purpose and arm-independent input files.
type Fixture struct {
	SchemaVersion string    `json:"schema_version"`
	FixtureID     string    `json:"fixture_id"`
	ValueType     string    `json:"value_type"`
	Audience      string    `json:"audience"`
	ReaderPromise string    `json:"reader_promise"`
	Emphasis      string    `json:"emphasis,omitempty"`
	Language      string    `json:"language"`
	InputFiles    []FileRef `json:"input_files"`
}

// Arm freezes the treatment role and executable contract digest.
type Arm struct {
	SchemaVersion  string `json:"schema_version"`
	ArmID          string `json:"arm_id"`
	Role           string `json:"role"`
	ContractPath   string `json:"contract_path"`
	ContractSHA256 string `json:"contract_sha256"`
	ContextualOnly bool   `json:"contextual_only"`
}

// Executor performs one preassigned synthetic cell without owning archive paths.
type Executor interface {
	Execute(context.Context, ExecutionInput) (ExecutionOutput, error)
}

// ExecutionInput contains only harness-validated immutable bytes and identities.
type ExecutionInput struct {
	ProtocolID     string
	ProtocolSHA256 string
	Fixture        ExecutionFixture
	Inputs         []InputArtifact
	Arm            ExecutionArm
	ArmContract    InputArtifact
	Cell           RunCell
}

// ExecutionFixture omits archive paths after harness-side validation.
type ExecutionFixture struct {
	FixtureID     string
	ValueType     string
	Audience      string
	ReaderPromise string
	Emphasis      string
	Language      string
}

// ExecutionArm omits the archive-local contract path after validation.
type ExecutionArm struct {
	ArmID          string
	Role           string
	ContractSHA256 string
	ContextualOnly bool
}

// InputArtifact is one verified immutable fixture input passed by value.
type InputArtifact struct {
	ID      string
	SHA256  string
	Content []byte
}

// ExecutionOutput returns attempt facts and candidate bytes to the harness.
type ExecutionOutput struct {
	Artifacts []OutputArtifact
	Attempts  []AttemptReceipt
}

// OutputArtifact is an unpersisted candidate whose filename is archive-local.
type OutputArtifact struct {
	Kind      string
	MediaType string
	Filename  string
	Content   []byte
}

// AttemptReceipt records the single semantic attempt and optional technical retry.
type AttemptReceipt struct {
	Attempt int    `json:"attempt"`
	Kind    string `json:"kind"`
	Outcome string `json:"outcome"`
}

// Config identifies one preassigned run cell and exact protocol bytes.
type Config struct {
	ArchiveRoot    string
	RepositoryRoot string
	ProtocolPath   string
	ProtocolSHA256 string
	RunID          string
	Executor       Executor
	StartedAt      time.Time
}

// Result names one run directory and its authoritative terminal receipt.
type Result struct {
	RunDir       string
	PendingPath  string
	TerminalPath string
	Terminal     TerminalManifest
}

// MatrixConfig identifies one complete synthetic nine-cell preflight.
type MatrixConfig struct {
	ArchiveRoot       string
	RepositoryRoot    string
	ProtocolPath      string
	ProtocolSHA256    string
	ExpectedFailedRun string
	Executor          Executor
	StartedAt         time.Time
}

// MatrixResult reports all terminal cells and the final matrix receipt.
type MatrixResult struct {
	Runs           []Result
	Completed      int
	Failed         int
	ManifestPath   string
	ManifestSHA256 string
	Manifest       MatrixManifest
}
