package reportilcontract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/source"
)

const (
	SourceCatalogSchemaVersion       = "plasma.report_il.source_catalog.experimental.v2"
	SourceAccessProfile              = "report_il_source.v1"
	SourceListTool                   = "plasma.report_il.sources.list"
	SourceReadTool                   = "plasma.report_il.sources.read"
	SourceQuoteRegisterTool          = "plasma.report_il.sources.quote"
	EditorialMemoryStartTool         = "plasma.report_il.memory.start"
	EditorialMemoryAppendTool        = "plasma.report_il.memory.append"
	EditorialMemoryReadTool          = "plasma.report_il.memory.read"
	EditorialMemoryFinalizeTool      = "plasma.report_il.memory.finalize"
	AuthorDocumentStartTool          = "plasma.report_il.document.start"
	AuthorDocumentOpenTool           = "plasma.report_il.document.open"
	AuthorDocumentAppendTool         = "plasma.report_il.document.append"
	AuthorDocumentAppendSourceTool   = "plasma.report_il.document.append_source"
	AuthorDocumentReadTool           = "plasma.report_il.document.read"
	AuthorDocumentReplaceTool        = "plasma.report_il.document.replace"
	AuthorDocumentEditTextTool       = "plasma.report_il.document.edit_text"
	AuthorDocumentReviseBlockTool    = "plasma.report_il.document.revise_block"
	AuthorDocumentFinalizeTool       = "plasma.report_il.document.finalize"
	LongFormPlanSubmitTool           = "plasma.report_il.long_form.plan.submit"
	LongFormPlanReadTool             = "plasma.report_il.long_form.plan.read"
	LongFormDocumentStartTool        = "plasma.report_il.long_form.document.start"
	LongFormDocumentAppendTool       = "plasma.report_il.long_form.document.append"
	LongFormDocumentReadTool         = "plasma.report_il.long_form.document.read"
	LongFormDocumentReplaceTool      = "plasma.report_il.long_form.document.replace"
	LongFormDocumentCorrectBlockTool = "plasma.report_il.long_form.document.correct_block"
	LongFormDocumentFinalizeTool     = "plasma.report_il.long_form.document.finalize"

	DefaultSourceReadMaxBytes     = 64 * 1024
	DefaultSourceAttemptReadBytes = 512 * 1024
	MaxSourceReadSpans            = 128
	MaxSourceReadSpanBytes        = 2 * 1024
	MaxSourceQuoteBytes           = 512
)

// SourceCatalog freezes the accepted source identities available to one IL run.
// It intentionally carries no source content, locator, filename, or URL.
type SourceCatalog struct {
	SchemaVersion string               `json:"schema_version"`
	MissionID     string               `json:"mission_id"`
	Sources       []SourceCatalogEntry `json:"sources"`
	SHA256        string               `json:"sha256"`
}

type SourceCatalogEntry struct {
	SourceKey          string                  `json:"source_key"`
	AcceptedOrdinal    int                     `json:"accepted_ordinal"`
	SnapshotID         string                  `json:"snapshot_id"`
	SnapshotReceipt    string                  `json:"snapshot_receipt"`
	ContentHash        string                  `json:"content_hash,omitempty"`
	RetrievalPolicy    string                  `json:"retrieval_policy"`
	Artifacts          []SourceCatalogArtifact `json:"artifacts,omitempty"`
	ReadableSHA256     string                  `json:"readable_sha256"`
	ReadableBytes      int                     `json:"readable_bytes"`
	Extraction         string                  `json:"extraction"`
	ObservationReceipt string                  `json:"observation_receipt,omitempty"`
}

type SourceCatalogArtifact struct {
	ArtifactID string `json:"artifact_id"`
	SHA256     string `json:"sha256"`
	ByteSize   int64  `json:"byte_size"`
	MediaType  string `json:"media_type"`
}

// SourceReadReceipt is the content-free inventory of frozen sources read by one
// exact provider attempt. It never carries source text or private identities.
type SourceReadReceipt struct {
	SourceKeys           []string
	FullyReadSourceKeys  []string
	ReturnedContentBytes int
	ReadBytesBySource    map[string]int
	ReadRanges           []SourceReadRange
	SourceQuotes         map[string]SourceQuoteReceipt
}

type SourceReadRange struct {
	SourceKey string
	Offset    int
	ByteSize  int
}

// SourceQuoteReceipt is request-local, content-free proof that one provider
// attempt registered an exact quote from a frozen canonical source. The quote
// bytes remain only in the frozen source and are reconstructed by the server.
type SourceQuoteReceipt struct {
	Receipt   string
	SourceKey string
	Offset    int
	ByteSize  int
	SHA256    string
}

func (receipt SourceReadReceipt) Validate(catalog SourceCatalog) error {
	if err := ValidateSourceCatalog(catalog); err != nil {
		return err
	}
	if receipt.ReturnedContentBytes < 1 || len(receipt.SourceKeys) == 0 {
		return fmt.Errorf("report IL provider did not read a frozen accepted source")
	}
	seen := make(map[string]bool, len(receipt.SourceKeys))
	readBytes := 0
	for _, sourceKey := range receipt.SourceKeys {
		if seen[sourceKey] {
			return fmt.Errorf("report IL source read receipt contains a duplicate source key")
		}
		seen[sourceKey] = true
		entry, ok := catalog.Entry(sourceKey)
		if !ok {
			return fmt.Errorf("report IL source read receipt contains an unavailable source key")
		}
		if receipt.ReadBytesBySource != nil {
			bytes := receipt.ReadBytesBySource[sourceKey]
			if bytes < 1 || bytes > entry.ReadableBytes {
				return fmt.Errorf("report IL source read receipt contains invalid source byte coverage")
			}
			readBytes += bytes
		}
	}
	for sourceKey := range receipt.ReadBytesBySource {
		if !seen[sourceKey] {
			return fmt.Errorf("report IL source read receipt contains bytes for an unread source")
		}
	}
	if receipt.ReadBytesBySource != nil && readBytes != receipt.ReturnedContentBytes {
		return fmt.Errorf("report IL source read receipt byte total does not match per-source coverage")
	}
	fullySeen := make(map[string]bool, len(receipt.FullyReadSourceKeys))
	for _, sourceKey := range receipt.FullyReadSourceKeys {
		if fullySeen[sourceKey] {
			return fmt.Errorf("report IL source read receipt contains a duplicate fully-read source key")
		}
		fullySeen[sourceKey] = true
		if !seen[sourceKey] {
			return fmt.Errorf("report IL source read receipt marks an unread source as fully read")
		}
	}
	if len(receipt.SourceQuotes) > MaxSourceReadSpans {
		return fmt.Errorf("report IL source quote receipt inventory exceeds the ceiling")
	}
	for key, quote := range receipt.SourceQuotes {
		entry, ok := catalog.Entry(quote.SourceKey)
		if key == "" || key != quote.Receipt || !validSourceQuoteReceipt(key) || !ok ||
			quote.Offset < 0 || quote.ByteSize < 1 || quote.ByteSize > MaxSourceQuoteBytes ||
			quote.Offset > entry.ReadableBytes-quote.ByteSize || !validSourceAccessSHA256(quote.SHA256) ||
			!seen[quote.SourceKey] {
			return fmt.Errorf("report IL source quote receipt is invalid")
		}
	}
	return nil
}

func (receipt SourceReadReceipt) Includes(sourceKey string) bool {
	for _, readKey := range receipt.SourceKeys {
		if readKey == sourceKey {
			return true
		}
	}
	return false
}

func (receipt SourceReadReceipt) IncludesEntireSource(sourceKey string) bool {
	for _, readKey := range receipt.FullyReadSourceKeys {
		if readKey == sourceKey {
			return true
		}
	}
	return false
}

// SourceAccessBinding is serialized only into the request-local Plasma MCP
// server configuration. Each fresh provider attempt receives its own budget.
type SourceReadSpan struct {
	SourceKey string `json:"source_key"`
	Offset    int    `json:"offset"`
	ByteSize  int    `json:"byte_size"`
	SHA256    string `json:"sha256"`
}

type SourceAccessBinding struct {
	PendingEventID             string           `json:"pending_event_id"`
	Stage                      string           `json:"stage"`
	Attempt                    int              `json:"attempt"`
	MaxReadBytes               int              `json:"max_read_bytes"`
	MaxCallBytes               int              `json:"max_call_bytes"`
	MaxSourceReadBytes         int              `json:"max_source_read_bytes,omitempty"`
	ReadSpans                  []SourceReadSpan `json:"read_spans,omitempty"`
	EditorialMemoryArtifactID  string           `json:"editorial_memory_artifact_id,omitempty"`
	EditorialMemorySHA256      string           `json:"editorial_memory_sha256,omitempty"`
	BaseAuthorArtifactID       string           `json:"base_author_artifact_id,omitempty"`
	BaseAuthorSHA256           string           `json:"base_author_sha256,omitempty"`
	LongFormInputArtifactIDs   []string         `json:"long_form_input_artifact_ids,omitempty"`
	LongFormInputSHA256s       []string         `json:"long_form_input_sha256s,omitempty"`
	LongFormPlanArtifactID     string           `json:"long_form_plan_artifact_id,omitempty"`
	LongFormPlanSHA256         string           `json:"long_form_plan_sha256,omitempty"`
	LongFormPartKey            string           `json:"long_form_part_key,omitempty"`
	LongFormSectionKey         string           `json:"long_form_section_key,omitempty"`
	LongFormTargetLanguage     string           `json:"long_form_target_language,omitempty"`
	LongFormSourceKeys         []string         `json:"long_form_source_keys,omitempty"`
	LongFormSectionAccountKeys []string         `json:"long_form_section_account_keys,omitempty"`
	Catalog                    SourceCatalog    `json:"catalog"`
}

func SealSourceCatalog(catalog SourceCatalog) (SourceCatalog, error) {
	catalog.SchemaVersion = SourceCatalogSchemaVersion
	catalog.MissionID = strings.TrimSpace(catalog.MissionID)
	catalog.SHA256 = ""
	for index := range catalog.Sources {
		if catalog.Sources[index].AcceptedOrdinal == 0 {
			catalog.Sources[index].AcceptedOrdinal = index + 1
		}
	}
	if err := validateSourceCatalogShape(catalog, false); err != nil {
		return SourceCatalog{}, err
	}
	raw, err := sourceCatalogPreimage(catalog)
	if err != nil {
		return SourceCatalog{}, err
	}
	catalog.SHA256 = sourceAccessSHA256(raw)
	return catalog, nil
}

func ValidateSourceCatalog(catalog SourceCatalog) error {
	if err := validateSourceCatalogShape(catalog, true); err != nil {
		return err
	}
	raw, err := sourceCatalogPreimage(catalog)
	if err != nil {
		return err
	}
	if !strings.EqualFold(catalog.SHA256, sourceAccessSHA256(raw)) {
		return fmt.Errorf("source catalog hash mismatch")
	}
	return nil
}

func ValidateSourceAccessBinding(binding SourceAccessBinding) error {
	if !strings.HasPrefix(strings.TrimSpace(binding.PendingEventID), "evt_") {
		return fmt.Errorf("source access pending event is invalid")
	}
	stage := strings.TrimSpace(binding.Stage)
	switch stage {
	case "il_source_selection", "il_editorial_memory", "il_narrative", "il_long_form_plan", "il_long_form_section", "il_long_form_part", "il_long_form_final", "il_continuity", "il_reader", "il_document", "il_flow":
	default:
		return fmt.Errorf("source access stage is invalid")
	}
	if binding.Attempt < 1 || binding.Attempt > 2 ||
		((stage == "il_editorial_memory" || stage == "il_continuity") && binding.Attempt != 1) {
		return fmt.Errorf("source access attempt is invalid")
	}
	if binding.MaxCallBytes < 0 || binding.MaxCallBytes > DefaultSourceReadMaxBytes ||
		binding.MaxReadBytes < 0 || binding.MaxReadBytes > DefaultSourceAttemptReadBytes ||
		(binding.MaxCallBytes == 0) != (binding.MaxReadBytes == 0) ||
		binding.MaxReadBytes > 0 && binding.MaxReadBytes < binding.MaxCallBytes {
		return fmt.Errorf("source access read budget is invalid for the stage")
	}
	if stage == "il_source_selection" {
		if binding.MaxSourceReadBytes < 1 || binding.MaxSourceReadBytes > binding.MaxCallBytes {
			return fmt.Errorf("source selection per-source budget is invalid")
		}
	} else if binding.MaxSourceReadBytes != 0 {
		return fmt.Errorf("source access per-source budget is invalid")
	}
	requiresMemory := stage == "il_continuity" || stage == "il_reader" ||
		stage == "il_narrative" && binding.MaxReadBytes == 0 ||
		(stage == "il_long_form_plan" || stage == "il_long_form_section") && binding.MaxReadBytes == 0
	allowsOptionalMemory := stage == "il_long_form_part" || stage == "il_long_form_final"
	hasMemory := binding.EditorialMemoryArtifactID != "" || binding.EditorialMemorySHA256 != ""
	if requiresMemory || allowsOptionalMemory && hasMemory {
		if !strings.HasPrefix(strings.TrimSpace(binding.EditorialMemoryArtifactID), "art_") ||
			!validSourceAccessSHA256(binding.EditorialMemorySHA256) {
			return fmt.Errorf("editorial memory binding is invalid")
		}
	} else if hasMemory {
		return fmt.Errorf("editorial memory binding is invalid for the stage")
	}
	if stage == "il_continuity" || stage == "il_reader" || stage == "il_long_form_part" || stage == "il_long_form_final" {
		if stage == "il_continuity" || stage == "il_reader" {
			if !strings.HasPrefix(strings.TrimSpace(binding.BaseAuthorArtifactID), "art_") ||
				!validSourceAccessSHA256(binding.BaseAuthorSHA256) {
				return fmt.Errorf("publication reader base document binding is invalid")
			}
		} else if binding.BaseAuthorArtifactID != "" || binding.BaseAuthorSHA256 != "" {
			return fmt.Errorf("author document binding is invalid for the long-form stage")
		}
	} else if binding.BaseAuthorArtifactID != "" || binding.BaseAuthorSHA256 != "" {
		return fmt.Errorf("author document binding is invalid for the stage")
	}
	if stage == "il_long_form_part" || stage == "il_long_form_final" {
		if len(binding.LongFormInputArtifactIDs) == 0 || len(binding.LongFormInputArtifactIDs) != len(binding.LongFormInputSHA256s) {
			return fmt.Errorf("long-form input artifact binding is invalid")
		}
		seenArtifacts := map[string]bool{}
		for index, artifactID := range binding.LongFormInputArtifactIDs {
			if !strings.HasPrefix(strings.TrimSpace(artifactID), "art_") || seenArtifacts[artifactID] ||
				!validSourceAccessSHA256(binding.LongFormInputSHA256s[index]) {
				return fmt.Errorf("long-form input artifact binding is invalid")
			}
			seenArtifacts[artifactID] = true
		}
	} else if len(binding.LongFormInputArtifactIDs) != 0 || len(binding.LongFormInputSHA256s) != 0 {
		return fmt.Errorf("long-form input artifact binding is invalid for the stage")
	}
	needsLongFormPlan := stage == "il_long_form_section" || stage == "il_long_form_part" || stage == "il_long_form_final"
	if needsLongFormPlan {
		if !strings.HasPrefix(strings.TrimSpace(binding.LongFormPlanArtifactID), "art_") ||
			!validSourceAccessSHA256(binding.LongFormPlanSHA256) {
			return fmt.Errorf("long-form plan binding is invalid")
		}
		switch stage {
		case "il_long_form_section":
			if !regexp.MustCompile(`^part_[0-9]{3}$`).MatchString(binding.LongFormPartKey) ||
				!regexp.MustCompile(`^part_[0-9]{3}\.section_[0-9]{3}$`).MatchString(binding.LongFormSectionKey) ||
				!strings.HasPrefix(binding.LongFormSectionKey, binding.LongFormPartKey+".") {
				return fmt.Errorf("long-form Section binding is invalid")
			}
			if binding.MaxReadBytes == 0 && len(binding.LongFormSectionAccountKeys) == 0 {
				return fmt.Errorf("memory-backed long-form Section account binding is invalid")
			}
		case "il_long_form_part":
			if !regexp.MustCompile(`^part_[0-9]{3}$`).MatchString(binding.LongFormPartKey) || binding.LongFormSectionKey != "" {
				return fmt.Errorf("long-form Part binding is invalid")
			}
		case "il_long_form_final":
			if binding.LongFormPartKey != "" || binding.LongFormSectionKey != "" {
				return fmt.Errorf("long-form final binding is invalid")
			}
		}
	} else if binding.LongFormPlanArtifactID != "" || binding.LongFormPlanSHA256 != "" || binding.LongFormPartKey != "" || binding.LongFormSectionKey != "" {
		return fmt.Errorf("long-form plan binding is invalid for the stage")
	}
	if err := ValidateSourceCatalog(binding.Catalog); err != nil {
		return err
	}
	longFormStage := stage == "il_long_form_plan" || stage == "il_long_form_section" || stage == "il_long_form_part" || stage == "il_long_form_final"
	if longFormStage {
		if !authorDocumentLanguagePattern.MatchString(binding.LongFormTargetLanguage) {
			return fmt.Errorf("long-form target language binding is invalid")
		}
	} else if binding.LongFormTargetLanguage != "" {
		return fmt.Errorf("long-form target language binding is invalid for the stage")
	}
	if stage == "il_long_form_plan" || stage == "il_long_form_section" {
		if binding.MaxReadBytes > 0 {
			if len(binding.LongFormSourceKeys) == 0 || len(binding.LongFormSectionAccountKeys) != 0 {
				return fmt.Errorf("direct long-form source binding is invalid")
			}
			seen := map[string]bool{}
			for _, sourceKey := range binding.LongFormSourceKeys {
				if seen[sourceKey] {
					return fmt.Errorf("direct long-form source binding contains duplicates")
				}
				if _, ok := binding.Catalog.Entry(sourceKey); !ok {
					return fmt.Errorf("direct long-form source binding is unavailable")
				}
				seen[sourceKey] = true
			}
		} else if len(binding.LongFormSourceKeys) != 0 {
			return fmt.Errorf("memory-backed long-form source binding is invalid")
		}
	} else if len(binding.LongFormSourceKeys) != 0 || len(binding.LongFormSectionAccountKeys) != 0 {
		return fmt.Errorf("long-form source binding is invalid for the stage")
	}
	if len(binding.LongFormSectionAccountKeys) > 0 {
		seenAccounts := map[string]bool{}
		for _, accountKey := range binding.LongFormSectionAccountKeys {
			if !regexp.MustCompile(`^account_[0-9]{3}$`).MatchString(accountKey) || seenAccounts[accountKey] {
				return fmt.Errorf("long-form Section account binding is invalid")
			}
			seenAccounts[accountKey] = true
		}
	}
	if binding.Stage == "il_flow" && len(binding.ReadSpans) > 0 {
		return validateSourceReadSpans(binding)
	}
	if len(binding.ReadSpans) != 0 {
		return fmt.Errorf("source access spans are invalid for the stage")
	}
	return nil
}

func validateSourceReadSpans(binding SourceAccessBinding) error {
	if len(binding.ReadSpans) < 1 || len(binding.ReadSpans) > MaxSourceReadSpans {
		return fmt.Errorf("flow source access spans are invalid")
	}
	total := 0
	previousKey := ""
	previousEnd := 0
	for _, span := range binding.ReadSpans {
		entry, ok := binding.Catalog.Entry(span.SourceKey)
		if !ok || span.Offset < 0 || span.ByteSize < 1 ||
			span.ByteSize > MaxSourceReadSpanBytes || !validSourceAccessSHA256(span.SHA256) ||
			span.Offset > entry.ReadableBytes-span.ByteSize {
			return fmt.Errorf("flow source access span is invalid")
		}
		if span.SourceKey < previousKey ||
			(span.SourceKey == previousKey && span.Offset < previousEnd) {
			return fmt.Errorf("flow source access spans are not ordered and non-overlapping")
		}
		previousKey = span.SourceKey
		previousEnd = span.Offset + span.ByteSize
		total += span.ByteSize
	}
	if total != binding.MaxReadBytes || binding.MaxCallBytes > binding.MaxReadBytes {
		return fmt.Errorf("flow source access span budget is invalid")
	}
	return nil
}

func SourceSnapshotReceipt(snapshotID, contentHash string) string {
	return sourceAccessSHA256([]byte(strings.TrimSpace(snapshotID) + "\x00" + strings.TrimSpace(contentHash)))
}

func SourceQuoteReceiptID(
	toolSessionID,
	catalogSHA256,
	stage string,
	ordinal int,
	sourceKey string,
	offset,
	byteSize int,
	quoteSHA256 string,
) string {
	return "quote_" + sourceAccessSHA256([]byte(fmt.Sprintf(
		"%s\x00%s\x00%s\x00%d\x00%s\x00%d\x00%d\x00%s",
		strings.TrimSpace(toolSessionID), strings.TrimSpace(catalogSHA256),
		strings.TrimSpace(stage), ordinal, strings.TrimSpace(sourceKey), offset,
		byteSize, strings.TrimSpace(quoteSHA256),
	)))
}

func (catalog SourceCatalog) Entry(sourceKey string) (SourceCatalogEntry, bool) {
	sourceKey = strings.TrimSpace(sourceKey)
	for _, entry := range catalog.Sources {
		if entry.SourceKey == sourceKey {
			return entry, true
		}
	}
	return SourceCatalogEntry{}, false
}

func validateSourceCatalogShape(catalog SourceCatalog, requireHash bool) error {
	if catalog.SchemaVersion != SourceCatalogSchemaVersion {
		return fmt.Errorf("source catalog schema version is invalid")
	}
	if !strings.HasPrefix(catalog.MissionID, "mis_") {
		return fmt.Errorf("source catalog mission is invalid")
	}
	if len(catalog.Sources) == 0 {
		return fmt.Errorf("source catalog requires at least one source")
	}
	seenKeys := map[string]struct{}{}
	seenOrdinals := map[int]struct{}{}
	seenSnapshots := map[string]struct{}{}
	for index, entry := range catalog.Sources {
		wantKey := fmt.Sprintf("source_%03d", index+1)
		if entry.SourceKey != wantKey {
			return fmt.Errorf("source catalog key order is invalid")
		}
		if entry.AcceptedOrdinal < 1 {
			return fmt.Errorf("source catalog accepted-source ordinal is invalid")
		}
		if _, ok := seenOrdinals[entry.AcceptedOrdinal]; ok {
			return fmt.Errorf("source catalog contains duplicate accepted-source ordinal")
		}
		seenOrdinals[entry.AcceptedOrdinal] = struct{}{}
		if _, ok := seenKeys[entry.SourceKey]; ok {
			return fmt.Errorf("source catalog contains duplicate source key")
		}
		seenKeys[entry.SourceKey] = struct{}{}
		if !strings.HasPrefix(entry.SnapshotID, "src_") || entry.SnapshotReceipt == "" {
			return fmt.Errorf("source catalog snapshot identity is invalid")
		}
		if _, ok := seenSnapshots[entry.SnapshotID]; ok {
			return fmt.Errorf("source catalog contains duplicate snapshot")
		}
		seenSnapshots[entry.SnapshotID] = struct{}{}
		if entry.ReadableBytes < 1 || len(entry.ReadableSHA256) != 64 || strings.TrimSpace(entry.Extraction) == "" {
			return fmt.Errorf("source catalog readable receipt is invalid")
		}
		switch entry.RetrievalPolicy {
		case source.RetrievalPolicySnapshotOnly:
			if len(entry.Artifacts) == 0 || entry.ObservationReceipt != "" {
				return fmt.Errorf("snapshot source catalog receipt is invalid")
			}
		case source.RetrievalPolicyLiveReference:
			if len(entry.Artifacts) != 0 || !strings.HasPrefix(strings.TrimSpace(entry.ObservationReceipt), "evt_") {
				return fmt.Errorf("live source catalog receipt is invalid")
			}
		default:
			return fmt.Errorf("source catalog retrieval policy is invalid")
		}
		for _, artifact := range entry.Artifacts {
			if !strings.HasPrefix(artifact.ArtifactID, "art_") || len(artifact.SHA256) != 64 || artifact.ByteSize < 1 || strings.TrimSpace(artifact.MediaType) == "" {
				return fmt.Errorf("source catalog artifact receipt is invalid")
			}
		}
	}
	if requireHash && len(catalog.SHA256) != 64 {
		return fmt.Errorf("source catalog hash is invalid")
	}
	return nil
}

func sourceCatalogPreimage(catalog SourceCatalog) ([]byte, error) {
	preimage := struct {
		SchemaVersion string               `json:"schema_version"`
		MissionID     string               `json:"mission_id"`
		Sources       []SourceCatalogEntry `json:"sources"`
	}{catalog.SchemaVersion, catalog.MissionID, catalog.Sources}
	return json.Marshal(preimage)
}

func sourceAccessSHA256(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func validSourceAccessSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			if character < 'a' || character > 'f' {
				return false
			}
		}
	}
	return true
}

func validSourceQuoteReceipt(value string) bool {
	const prefix = "quote_"
	return strings.HasPrefix(value, prefix) && validSourceAccessSHA256(strings.TrimPrefix(value, prefix))
}
