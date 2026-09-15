package articleexperiment

// PendingManifest binds a new run directory to frozen protocol, fixture, and arm
// bytes before the executor is called.
type PendingManifest struct {
	SchemaVersion  string `json:"schema_version"`
	ProtocolID     string `json:"protocol_id"`
	ProtocolSHA256 string `json:"protocol_sha256"`
	FixtureID      string `json:"fixture_id"`
	FixtureSHA256  string `json:"fixture_sha256"`
	ArmID          string `json:"arm_id"`
	ArmSHA256      string `json:"arm_sha256"`
	RunID          string `json:"run_id"`
	StartedAt      string `json:"started_at"`
}

// TerminalManifest is the authoritative completed or failed state for one cell.
type TerminalManifest struct {
	SchemaVersion    string            `json:"schema_version"`
	Status           string            `json:"status"`
	Pending          PendingManifest   `json:"pending"`
	PendingSHA256    string            `json:"pending_sha256"`
	Attempts         []AttemptReceipt  `json:"attempts"`
	Artifacts        []ArtifactReceipt `json:"artifacts,omitempty"`
	PartialArtifacts []ArtifactReceipt `json:"partial_artifacts,omitempty"`
	Failure          *FailureReceipt   `json:"failure,omitempty"`
	CompletedAt      string            `json:"completed_at"`
}

// ArtifactReceipt binds one output or preserved partial artifact to exact bytes.
type ArtifactReceipt struct {
	Kind      string `json:"kind"`
	MediaType string `json:"media_type"`
	Filename  string `json:"filename"`
	SHA256    string `json:"sha256"`
	ByteSize  int    `json:"byte_size"`
}

// FailureReceipt exposes only a closed class and stable safe message.
type FailureReceipt struct {
	Class   string `json:"class"`
	Message string `json:"message"`
}

// MatrixManifest binds the complete nine-cell terminal set to one protocol.
type MatrixManifest struct {
	SchemaVersion     string             `json:"schema_version"`
	ProtocolID        string             `json:"protocol_id"`
	ProtocolSHA256    string             `json:"protocol_sha256"`
	ExpectedFailedRun string             `json:"expected_failed_run"`
	Runs              []MatrixRunReceipt `json:"runs"`
	Completed         int                `json:"completed"`
	Failed            int                `json:"failed"`
	CompletedAt       string             `json:"completed_at"`
}

// MatrixRunReceipt binds one fixture/arm cell to its terminal manifest bytes.
type MatrixRunReceipt struct {
	FixtureID      string `json:"fixture_id"`
	ArmID          string `json:"arm_id"`
	RunID          string `json:"run_id"`
	Status         string `json:"status"`
	TerminalPath   string `json:"terminal_path"`
	TerminalSHA256 string `json:"terminal_sha256"`
}
