package reportil

type ReportILDocumentStartInput struct {
	Title    string `json:"title"`
	Language string `json:"language"`
}

type ReportILDocumentOpenInput struct{}

type ReportILDocumentAppendInput struct {
	WorkspaceID          string                      `json:"workspace_id"`
	SectionTitle         string                      `json:"section_title"`
	Kind                 string                      `json:"kind"`
	Prose                string                      `json:"prose"`
	Items                []string                    `json:"items"`
	Code                 string                      `json:"code"`
	Language             *string                     `json:"language"`
	Table                *ReportILDocumentTableInput `json:"table"`
	EditorialAccountKeys []string                    `json:"editorial_account_keys"`
}

type ReportILDocumentAppendSourceInput struct {
	WorkspaceID        string                      `json:"workspace_id"`
	SectionTitle       string                      `json:"section_title"`
	Kind               string                      `json:"kind"`
	Prose              string                      `json:"prose"`
	Items              []string                    `json:"items"`
	Code               string                      `json:"code"`
	Language           *string                     `json:"language"`
	Table              *ReportILDocumentTableInput `json:"table"`
	EvidenceSourceKeys []string                    `json:"evidence_source_keys"`
}

type ReportILDocumentTableInput struct {
	Caption *string                         `json:"caption"`
	Columns []string                        `json:"columns"`
	Rows    []ReportILDocumentTableRowInput `json:"rows"`
}

type ReportILDocumentTableRowInput struct {
	Cells []string `json:"cells"`
}

type ReportILDocumentReadInput struct {
	WorkspaceID string `json:"workspace_id"`
	Offset      int    `json:"offset"`
	MaxBytes    int    `json:"max_bytes"`
}

type ReportILDocumentReplaceInput struct {
	WorkspaceID string `json:"workspace_id"`
	OldText     string `json:"old_text"`
	NewText     string `json:"new_text"`
}

type ReportILDocumentEditTextInput struct {
	WorkspaceID string `json:"workspace_id"`
	TargetKind  string `json:"target_kind"`
	TargetKey   string `json:"target_key"`
	OldText     string `json:"old_text"`
	NewText     string `json:"new_text"`
}

type ReportILDocumentReviseBlockInput struct {
	WorkspaceID          string   `json:"workspace_id"`
	BlockKey             string   `json:"block_key"`
	OldText              string   `json:"old_text"`
	NewText              string   `json:"new_text"`
	EditorialAccountKeys []string `json:"editorial_account_keys"`
}

type ReportILDocumentFinalizeInput struct {
	WorkspaceID string `json:"workspace_id"`
}

type ReportILDocumentStateOutput struct {
	WorkspaceID   string `json:"workspace_id"`
	ReportILStage string `json:"report_il_stage"`
	Revision      int    `json:"revision"`
	Replacements  int    `json:"replacements"`
	Sections      int    `json:"sections"`
	Blocks        int    `json:"blocks"`
	ByteSize      int    `json:"byte_size"`
	Finalized     bool   `json:"finalized"`
	ArtifactID    string `json:"artifact_id,omitempty"`
	SHA256        string `json:"sha256,omitempty"`
}

type ReportILDocumentReadOutput struct {
	WorkspaceID   string `json:"workspace_id"`
	ReportILStage string `json:"report_il_stage"`
	Revision      int    `json:"revision"`
	Content       string `json:"content"`
	Offset        int    `json:"offset"`
	NextOffset    int    `json:"next_offset,omitempty"`
	ContentLength int    `json:"content_length"`
	Truncated     bool   `json:"truncated"`
}

type ReportILLongFormDocumentStartInput struct {
	Title    string `json:"title"`
	Language string `json:"language"`
}

type ReportILLongFormDocumentAppendInput struct {
	WorkspaceID          string                         `json:"workspace_id"`
	SectionKey           string                         `json:"section_key,omitempty"`
	SectionTitle         string                         `json:"section_title,omitempty"`
	Kind                 string                         `json:"kind"`
	Prose                string                         `json:"prose"`
	Items                []string                       `json:"items"`
	Code                 string                         `json:"code"`
	Language             *string                        `json:"language"`
	Table                *ReportILDocumentTableInput    `json:"table"`
	Equation             *ReportILDocumentEquationInput `json:"equation"`
	EditorialAccountKeys []string                       `json:"editorial_account_keys"`
	EvidenceSourceKeys   []string                       `json:"evidence_source_keys"`
}

type ReportILLongFormDocumentCorrectBlockInput struct {
	WorkspaceID string                                 `json:"workspace_id"`
	BlockKey    string                                 `json:"block_key"`
	Operation   string                                 `json:"operation"`
	Replacement *ReportILLongFormBlockReplacementInput `json:"replacement"`
}

type ReportILLongFormBlockReplacementInput struct {
	Kind                 string                         `json:"kind"`
	Prose                string                         `json:"prose"`
	Items                []string                       `json:"items"`
	Code                 string                         `json:"code"`
	Language             *string                        `json:"language"`
	Table                *ReportILDocumentTableInput    `json:"table"`
	Equation             *ReportILDocumentEquationInput `json:"equation"`
	EditorialAccountKeys []string                       `json:"editorial_account_keys"`
	EvidenceSourceKeys   []string                       `json:"evidence_source_keys"`
}

type ReportILDocumentEquationInput struct {
	Expression string `json:"expression"`
	Notation   string `json:"notation"`
}
