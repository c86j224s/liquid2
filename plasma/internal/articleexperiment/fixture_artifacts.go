package articleexperiment

const (
	SourceCatalogSchemaVersion   = "plasma.article_experiment.source_catalog.v1"
	DossierSchemaVersion         = "plasma.article_experiment.dossier.v1"
	ClaimInventorySchemaVersion  = "plasma.article_experiment.claim_inventory.v1"
	MemoryQuestionsSchemaVersion = "plasma.article_experiment.memory_questions.v1"
	TransferTaskSchemaVersion    = "plasma.article_experiment.transfer_task.v1"
)

// FixtureSourceCatalog binds curated text to exact official-page snapshots.
type FixtureSourceCatalog struct {
	SchemaVersion string              `json:"schema_version"`
	FixtureID     string              `json:"fixture_id"`
	Sources       []FixtureSource     `json:"sources"`
	Spans         []FixtureSourceSpan `json:"spans"`
}

type FixtureSource struct {
	SourceKey       string             `json:"source_key"`
	Content         FileRef            `json:"content"`
	MediaType       string             `json:"media_type"`
	RetrievalPolicy string             `json:"retrieval_policy"`
	Extraction      string             `json:"extraction"`
	Derivation      []SourceDerivation `json:"derivation"`
	MediaExcluded   bool               `json:"media_excluded"`
	Rights          FixtureRights      `json:"rights"`
	Origins         []FixtureOrigin    `json:"origins"`
}

type SourceDerivation struct {
	OriginID   string `json:"origin_id"`
	OriginSpan string `json:"origin_span"`
	SHA256     string `json:"sha256"`
}

type FixtureRights struct {
	Agency    string `json:"agency"`
	RightsURL string `json:"rights_url"`
	AIUse     string `json:"ai_use"`
}

type FixtureOrigin struct {
	OriginID string  `json:"origin_id"`
	URL      string  `json:"url"`
	Raw      FileRef `json:"raw"`
	Headers  FileRef `json:"headers"`
	FinalURL FileRef `json:"final_url"`
	Rights   FileRef `json:"rights"`
}

type FixtureSourceSpan struct {
	SpanID    string `json:"span_id"`
	SourceKey string `json:"source_key"`
	StartByte int    `json:"start_byte"`
	EndByte   int    `json:"end_byte"`
	SHA256    string `json:"sha256"`
}

// FixtureDossier freezes source-grounded accounts, connections, and caveats.
type FixtureDossier struct {
	SchemaVersion string              `json:"schema_version"`
	FixtureID     string              `json:"fixture_id"`
	Sensitive     bool                `json:"sensitive"`
	Accounts      []FixtureAccount    `json:"accounts"`
	Connections   []FixtureConnection `json:"connections"`
	Caveats       []FixtureCaveat     `json:"caveats"`
	Forbidden     []string            `json:"forbidden"`
}

type FixtureAccount struct {
	AccountID string   `json:"account_id"`
	Kind      string   `json:"kind"`
	ClaimIDs  []string `json:"claim_ids"`
	SpanIDs   []string `json:"span_ids"`
}

type FixtureConnection struct {
	ConnectionID string   `json:"connection_id"`
	Kind         string   `json:"kind"`
	FromClaimIDs []string `json:"from_claim_ids"`
	ToClaimIDs   []string `json:"to_claim_ids"`
	SummaryKO    string   `json:"summary_ko"`
}

type FixtureCaveat struct {
	CaveatID  string   `json:"caveat_id"`
	SummaryKO string   `json:"summary_ko"`
	SpanIDs   []string `json:"span_ids"`
}

// FixtureClaimInventory is the pre-prose material-claim contract.
type FixtureClaimInventory struct {
	SchemaVersion        string         `json:"schema_version"`
	FixtureID            string         `json:"fixture_id"`
	Claims               []FixtureClaim `json:"claims"`
	RequiredLeafCoverage string         `json:"required_leaf_coverage"`
	Verdicts             []string       `json:"verdicts"`
}

type FixtureClaim struct {
	ClaimID         string   `json:"claim_id"`
	SummaryKO       string   `json:"summary_ko"`
	SupportRelation string   `json:"support_relation"`
	Uncertainty     string   `json:"uncertainty"`
	ClaimStrength   string   `json:"claim_strength"`
	SpanIDs         []string `json:"span_ids"`
}

// FixtureMemoryQuestions freezes answerable post-reading checks.
type FixtureMemoryQuestions struct {
	SchemaVersion string                  `json:"schema_version"`
	FixtureID     string                  `json:"fixture_id"`
	Questions     []FixtureMemoryQuestion `json:"questions"`
}

type FixtureMemoryQuestion struct {
	QuestionID       string   `json:"question_id"`
	QuestionKO       string   `json:"question_ko"`
	RequiredClaimIDs []string `json:"required_claim_ids"`
	Scoring          string   `json:"scoring"`
}

// FixtureTransferTask freezes one bounded application test and its evidence keys.
type FixtureTransferTask struct {
	SchemaVersion    string   `json:"schema_version"`
	FixtureID        string   `json:"fixture_id"`
	TaskKO           string   `json:"task_ko"`
	RequiredClaimIDs []string `json:"required_claim_ids"`
	Scoring          string   `json:"scoring"`
}
