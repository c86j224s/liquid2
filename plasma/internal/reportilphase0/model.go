package reportilphase0

import "encoding/json"

const (
	BundleSchemaVersion          = "plasma.report_il.phase0_bundle.experimental.v1"
	NarrativeSchemaVersion       = "plasma.report_narrative.experimental.v1"
	DocumentSchemaVersion        = "plasma.report_il.experimental.v2"
	AttestationSchemaVersion     = "plasma.report_flow_attestation.experimental.v2"
	ManifestSchemaVersion        = "plasma.report_il.phase0_manifest.v1"
	ProductManifestSchemaVersion = "plasma.report_il.product_manifest.experimental.v36"
	CompilerVersion              = "plasma.report_il.phase0_compiler.v68"
	PipelineFamily               = "report_il_experimental"
)

// Bundle is one frozen Phase 0 input. It binds an optional Narrative Contract,
// one accepted semantic document revision, and an optional human flow verdict.
type Bundle struct {
	SchemaVersion string           `json:"schema_version"`
	Arm           string           `json:"arm"`
	Narrative     *Narrative       `json:"narrative,omitempty"`
	Document      Document         `json:"document"`
	Attestation   *FlowAttestation `json:"flow_attestation,omitempty"`
}

// Narrative records authoring and editing obligations. It is not a sentence
// template and is never consumed by renderers to invent prose.
type Narrative struct {
	SchemaVersion         string             `json:"schema_version"`
	ContractID            string             `json:"contract_id"`
	DocumentID            string             `json:"document_id"`
	CentralQuestion       string             `json:"central_question"`
	ReaderTakeaway        string             `json:"reader_takeaway"`
	Throughline           string             `json:"throughline"`
	ReaderJourney         []string           `json:"reader_journey"`
	ArgumentArc           []string           `json:"argument_arc"`
	SectionRoles          []SectionRole      `json:"section_roles"`
	DependencyEdges       []DependencyEdge   `json:"dependency_edges,omitempty"`
	TransitionObligations []Transition       `json:"transition_obligations,omitempty"`
	ContinuityTerms       []ContinuityTerm   `json:"continuity_terms,omitempty"`
	OpenLoops             []OpenLoop         `json:"open_loops,omitempty"`
	Callbacks             []Callback         `json:"callbacks,omitempty"`
	RepetitionPolicies    []RepetitionPolicy `json:"repetition_policies,omitempty"`
	ConclusionObligations []string           `json:"conclusion_obligations"`
	EvidencePackets       []EvidencePacket   `json:"evidence_packets,omitempty"`
	VoiceAndTone          string             `json:"voice_and_tone"`
}

// SectionRole states a Section's job and reader-state handoff without
// constraining the sentences used to realize it.
type SectionRole struct {
	SectionID                string   `json:"section_id"`
	Role                     string   `json:"role"`
	QuestionAnswered         string   `json:"question_answered"`
	ReaderStateBefore        []string `json:"reader_state_before,omitempty"`
	MustEstablish            []string `json:"must_establish,omitempty"`
	PrerequisiteSections     []string `json:"prerequisite_sections,omitempty"`
	MustNotRepeat            []string `json:"must_not_repeat,omitempty"`
	OpenLoopsIntroduced      []string `json:"open_loops_introduced,omitempty"`
	CallbacksResolved        []string `json:"callbacks_resolved,omitempty"`
	TransitionFromPrevious   string   `json:"transition_from_previous,omitempty"`
	TransitionToNext         string   `json:"transition_to_next,omitempty"`
	ExpectedReaderStateAfter []string `json:"expected_reader_state_after,omitempty"`
}

type DependencyEdge struct {
	FromSectionID string `json:"from_section_id"`
	ToSectionID   string `json:"to_section_id"`
	Reason        string `json:"reason"`
}

type Transition struct {
	FromSectionID string `json:"from_section_id"`
	ToSectionID   string `json:"to_section_id"`
	Obligation    string `json:"obligation"`
}

type ContinuityTerm struct {
	Canonical string   `json:"canonical"`
	Aliases   []string `json:"aliases,omitempty"`
}

type OpenLoop struct {
	LoopID              string `json:"loop_id"`
	IntroducedSectionID string `json:"introduced_section_id"`
	Question            string `json:"question"`
}

type Callback struct {
	LoopID            string `json:"loop_id"`
	ResolvedSectionID string `json:"resolved_section_id"`
	Obligation        string `json:"obligation"`
}

type RepetitionPolicy struct {
	Concept string `json:"concept"`
	Policy  string `json:"policy"`
}

// Document is the publishable target-neutral meaning owned by the Phase 0 IL.
type Document struct {
	SchemaVersion       string                     `json:"schema_version"`
	PipelineFamily      string                     `json:"pipeline_family"`
	DocumentID          string                     `json:"document_id"`
	RevisionID          string                     `json:"revision_id"`
	NarrativeContractID string                     `json:"narrative_contract_id,omitempty"`
	Title               string                     `json:"title"`
	Language            string                     `json:"language"`
	Blocks              []Block                    `json:"blocks"`
	References          []Reference                `json:"references,omitempty"`
	Assets              []Asset                    `json:"assets,omitempty"`
	Coverage            []Coverage                 `json:"coverage,omitempty"`
	Provenance          map[string]string          `json:"provenance"`
	Extensions          map[string]json.RawMessage `json:"extensions,omitempty"`
}

// Block is a stable logical unit. Ordinary prose remains an authored leaf;
// structural and cross-target semantics are represented explicitly.
type Block struct {
	NodeID             string          `json:"node_id"`
	Kind               string          `json:"kind"`
	ParentNodeID       string          `json:"parent_node_id,omitempty"`
	Level              int             `json:"level,omitempty"`
	Title              string          `json:"title,omitempty"`
	Prose              string          `json:"prose,omitempty"`
	Items              []string        `json:"items,omitempty"`
	Code               string          `json:"code,omitempty"`
	Language           string          `json:"language,omitempty"`
	Table              *Table          `json:"table,omitempty"`
	Equation           *Equation       `json:"equation,omitempty"`
	Figure             *Figure         `json:"figure,omitempty"`
	SemanticRole       string          `json:"semantic_role,omitempty"`
	Supports           []string        `json:"supports,omitempty"`
	Qualifies          []string        `json:"qualifies,omitempty"`
	ContrastsWith      []string        `json:"contrasts_with,omitempty"`
	Elaborates         []string        `json:"elaborates,omitempty"`
	RefersTo           []string        `json:"refers_to,omitempty"`
	EvidenceRefs       []string        `json:"evidence_refs,omitempty"`
	RequirementRefs    []string        `json:"requirement_refs,omitempty"`
	PresentationIntent string          `json:"presentation_intent,omitempty"`
	Extension          json.RawMessage `json:"extension,omitempty"`
}

type Table struct {
	Caption string     `json:"caption,omitempty"`
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

type Equation struct {
	Expression string `json:"expression"`
	Notation   string `json:"notation"`
}

type Figure struct {
	AssetID    string `json:"asset_id"`
	Caption    string `json:"caption"`
	Alt        string `json:"alt"`
	Decorative bool   `json:"decorative,omitempty"`
}

type Reference struct {
	RefID        string `json:"ref_id"`
	Kind         string `json:"kind"`
	Target       string `json:"target"`
	VisibleLabel string `json:"visible_label,omitempty"`
	Locator      string `json:"locator,omitempty"`
}

type Asset struct {
	AssetID               string `json:"asset_id"`
	MediaType             string `json:"media_type"`
	SHA256                string `json:"sha256"`
	DataBase64            string `json:"data_base64,omitempty"`
	Alt                   string `json:"alt,omitempty"`
	Decorative            bool   `json:"decorative,omitempty"`
	LicenseStatus         string `json:"license_status"`
	ArtifactID            string `json:"artifact_id,omitempty"`
	SourceSnapshotReceipt string `json:"source_snapshot_receipt,omitempty"`
	SourcePageURL         string `json:"source_page_url,omitempty"`
	SourceImageURL        string `json:"source_image_url,omitempty"`
}

type Coverage struct {
	RequirementID string   `json:"requirement_id"`
	NodeIDs       []string `json:"node_ids"`
	Status        string   `json:"status"`
}

// FlowAttestation binds a full-reader verdict to exact accepted IL and linear
// manuscript bytes. The compiler verifies the binding but never creates the verdict.
type FlowAttestation struct {
	SchemaVersion                string                           `json:"schema_version"`
	DocumentID                   string                           `json:"document_id"`
	RevisionID                   string                           `json:"revision_id"`
	LinearProjectionSHA256       string                           `json:"linear_projection_sha256"`
	Reviewer                     string                           `json:"reviewer"`
	ReviewedFullManuscript       bool                             `json:"reviewed_full_manuscript"`
	CentralThreadPreserved       bool                             `json:"central_thread_preserved"`
	SectionHandoffsResolved      bool                             `json:"section_handoffs_resolved"`
	OpenLoopsResolved            bool                             `json:"open_loops_resolved"`
	TerminologyContinuous        bool                             `json:"terminology_continuous"`
	EvidenceSupportCoverage      []EvidenceSupportCoverageReceipt `json:"evidence_support_coverage,omitempty"`
	PreservationCoverage         []PreservationCoverageReceipt    `json:"preservation_coverage,omitempty"`
	AccidentalRepetitionFindings []FlowFinding                    `json:"accidental_repetition_findings,omitempty"`
	AbruptTransitionFindings     []FlowFinding                    `json:"abrupt_transition_findings,omitempty"`
	UnsupportedAdditionFindings  []FlowFinding                    `json:"unsupported_addition_findings,omitempty"`
	Verdict                      string                           `json:"verdict"`
}

type FlowFinding struct {
	NodeID string `json:"node_id"`
	Detail string `json:"detail"`
}

// DegradationReceipt records every target capability outcome that is not a
// direct, lossless representation.
type DegradationReceipt struct {
	Target     string `json:"target"`
	NodeID     string `json:"node_id,omitempty"`
	Capability string `json:"capability"`
	Outcome    string `json:"outcome"`
	Reason     string `json:"reason"`
}
