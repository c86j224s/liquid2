package reportexperiment

import "encoding/json"

const (
	// ManifestSchemaVersion은 archive run manifest의 안정 JSON schema 이름이다.
	ManifestSchemaVersion = "plasma.reportexperiment.run_manifest.v1"
	// RunnerScope는 이 개발 실행기가 호출하는 제품 workflow entrypoint를 고정한다.
	RunnerScope = "reportworkflow.finalize_long_form_prefix"
	timeFormat  = "2006-01-02T15:04:05.000000000Z07:00"
)

// Manifest는 archive run directory에 남기는 compact 실행 receipt다.
type Manifest struct {
	SchemaVersion          string                    `json:"schema_version"`
	RunID                  string                    `json:"run_id"`
	CreatedAt              string                    `json:"created_at"`
	RunnerScope            string                    `json:"runner_scope"`
	ProductRevision        string                    `json:"product_revision,omitempty"`
	Fixture                ManifestFixture           `json:"fixture"`
	SourceProvenance       ManifestSourceProvenance  `json:"source_provenance"`
	Binaries               BinaryPair                `json:"binaries"`
	Agent                  ManifestAgent             `json:"agent"`
	Bootstrap              BootstrapReceipt          `json:"bootstrap"`
	Seed                   seedPlanSummary           `json:"seed"`
	Observations           []NodeObservation         `json:"observations"`
	AgentRequests          []AgentRequestObservation `json:"agent_requests"`
	FinalArtifact          ManifestFinalArtifact     `json:"final_artifact"`
	Outputs                ManifestOutputs           `json:"outputs"`
	IntentionalDifferences []string                  `json:"intentional_differences"`
}

// ManifestSourceProvenance는 free-form notes를 제외한 source provenance receipt다.
type ManifestSourceProvenance struct {
	ProvenanceID  string `json:"provenance_id"`
	ProductCommit string `json:"product_commit"`
	SourceID      string `json:"source_id,omitempty"`
}

// ManifestFixture는 fixture와 reviewed Part 파일 receipt만 담고 Part 본문은 담지 않는다.
type ManifestFixture struct {
	FixtureID string                `json:"fixture_id"`
	SHA256    string                `json:"sha256"`
	Parts     []ManifestPartReceipt `json:"parts"`
}

// ManifestPartReceipt는 실행 전에 검증한 Part order와 SHA-256 receipt다.
type ManifestPartReceipt struct {
	Index  int    `json:"index"`
	Title  string `json:"title"`
	SHA256 string `json:"sha256"`
}

// ManifestAgent는 실제 provider 요청에 사용한 Codex model 설정이다.
type ManifestAgent struct {
	Executor        string `json:"executor"`
	Model           string `json:"model"`
	ReasoningEffort string `json:"reasoning_effort"`
}

// ManifestFinalArtifact는 final report artifact와 archive export hash를 연결한다.
type ManifestFinalArtifact struct {
	ArtifactID     string `json:"artifact_id"`
	EventID        string `json:"event_id"`
	ArtifactSHA256 string `json:"artifact_sha256"`
	ReportSHA256   string `json:"report_sha256"`
	LedgerSHA256   string `json:"ledger_sha256"`
}

// ManifestOutputs는 run directory 안에 남긴 durable output paths다.
type ManifestOutputs struct {
	RunDirectory string `json:"run_directory"`
	DatabasePath string `json:"database_path"`
	ReportPath   string `json:"report_path"`
	LedgerPath   string `json:"ledger_path"`
	ManifestPath string `json:"manifest_path"`
}

func sourceProvenanceManifest(source SourceProvenance) ManifestSourceProvenance {
	return ManifestSourceProvenance{
		ProvenanceID:  source.ProvenanceID,
		ProductCommit: source.ProductCommit,
		SourceID:      source.SourceID,
	}
}

func fixtureManifest(loaded LoadedFixture) ManifestFixture {
	parts := make([]ManifestPartReceipt, 0, len(loaded.Parts))
	for _, part := range loaded.Parts {
		parts = append(parts, ManifestPartReceipt{
			Index: part.Spec.Index, Title: part.Spec.Title, SHA256: part.SHA256,
		})
	}
	return ManifestFixture{FixtureID: loaded.Spec.FixtureID, SHA256: loaded.SHA256, Parts: parts}
}

func manifestJSON(value Manifest) ([]byte, error) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}
