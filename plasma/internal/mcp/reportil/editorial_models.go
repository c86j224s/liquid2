package reportil

type ReportILEditorialMemoryStartInput struct {
	Language string `json:"language"`
}

type ReportILEditorialMemoryAppendInput struct {
	WorkspaceID   string   `json:"workspace_id"`
	Importance    string   `json:"importance"`
	Account       string   `json:"account"`
	SourceKeys    []string `json:"source_keys"`
	SourceAnchors []string `json:"source_anchors"`
}

type ReportILEditorialMemoryReadInput struct {
	WorkspaceID string `json:"workspace_id"`
	Offset      int    `json:"offset"`
	MaxBytes    int    `json:"max_bytes"`
}

type ReportILEditorialMemoryFinalizeInput struct {
	WorkspaceID string `json:"workspace_id"`
}

type ReportILEditorialMemoryStateOutput struct {
	WorkspaceID string `json:"workspace_id"`
	Revision    int    `json:"revision"`
	Accounts    int    `json:"accounts"`
	ByteSize    int    `json:"byte_size"`
	Finalized   bool   `json:"finalized"`
	ArtifactID  string `json:"artifact_id,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
}

type ReportILEditorialMemoryReadOutput struct {
	WorkspaceID   string `json:"workspace_id"`
	ReportILStage string `json:"report_il_stage"`
	Revision      int    `json:"revision"`
	Content       string `json:"content"`
	Offset        int    `json:"offset"`
	NextOffset    int    `json:"next_offset,omitempty"`
	ContentLength int    `json:"content_length"`
	Truncated     bool   `json:"truncated"`
}
