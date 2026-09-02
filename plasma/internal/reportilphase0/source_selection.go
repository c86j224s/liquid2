package reportilphase0

import (
	"context"
	"fmt"
	"sort"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

const (
	sourceSelectionSampleBytes     = 8 * 1024
	sourceSelectionTargetReadBytes = 128 * 1024
	sourceSelectionMaxSources      = 64
)

type sourceSelectionDraft struct {
	RankedSourceKeys             []string `json:"ranked_source_keys"`
	LongFormSupplementSourceKeys []string `json:"long_form_supplement_source_keys,omitempty"`
}

type SourceSelectionResult struct {
	Catalog       reportilcontract.SourceCatalog
	Supplement    reportilcontract.SourceCatalog
	AuthorCatalog reportilcontract.SourceCatalog
}

type SourceSelectionDisposition struct {
	AcceptedOrdinal int    `json:"accepted_ordinal"`
	Status          string `json:"status"`
	Reason          string `json:"reason,omitempty"`
}

type SourceSelectionReceipt struct {
	Applied                   bool                         `json:"applied"`
	AcceptedSources           int                          `json:"accepted_sources"`
	UsableSources             int                          `json:"usable_sources"`
	SelectedSources           int                          `json:"selected_sources"`
	SupplementalSources       int                          `json:"supplemental_sources"`
	ExcludedUnusableSources   int                          `json:"excluded_unusable_sources"`
	ExcludedBudgetSources     int                          `json:"excluded_budget_sources"`
	SelectedReadableBytes     int                          `json:"selected_readable_bytes"`
	SupplementalReadableBytes int                          `json:"supplemental_readable_bytes"`
	CandidateCatalogSHA256    string                       `json:"candidate_catalog_sha256"`
	SelectedCatalogSHA256     string                       `json:"selected_catalog_sha256"`
	AuthorCatalogSHA256       string                       `json:"author_catalog_sha256"`
	DispositionSHA256         string                       `json:"disposition_sha256"`
	Dispositions              []SourceSelectionDisposition `json:"dispositions"`
}

func selectSourceCatalog(
	ctx context.Context,
	config ProductConfig,
	build SourceCatalogBuild,
) (SourceSelectionResult, SourceSelectionReceipt, []agentexec.AgentResult, error) {
	candidate := build.Catalog
	receipt := sourceSelectionReceipt(build, candidate)
	if !build.HasExclusions() && sourceCatalogFitsSelectionTarget(candidate) {
		receipt.SelectedSources = len(candidate.Sources)
		receipt.SelectedReadableBytes = sourceCatalogReadableBytes(candidate)
		receipt.SelectedCatalogSHA256 = candidate.SHA256
		receipt.AuthorCatalogSHA256 = candidate.SHA256
		for index := range receipt.Dispositions {
			receipt.Dispositions[index].Status = "selected"
		}
		receipt.DispositionSHA256 = sourceSelectionDispositionSHA(receipt.Dispositions)
		return SourceSelectionResult{Catalog: candidate, AuthorCatalog: candidate}, receipt, nil, nil
	}

	receipt.Applied = true
	if !sourceCatalogHasSelectableEntry(candidate) {
		return SourceSelectionResult{}, SourceSelectionReceipt{}, nil, fmt.Errorf("source selection cannot fit any usable source within the complete-read budget")
	}
	sampleBytes, err := sourceSelectionSampleLimit(candidate)
	if err != nil {
		return SourceSelectionResult{}, SourceSelectionReceipt{}, nil, err
	}
	longForm := normalizeAuthoringMode(config.AuthoringMode) == AuthoringModeLongForm
	supplementGuidance := "For standard authoring, return only ranked_source_keys."
	if longForm {
		supplementGuidance = "Also return long_form_supplement_source_keys. Use [] unless a source outside the compact packet uniquely supplies a materially different dimension, explanatory asset, operational example, comparison, calculation, benchmark, or caveat that a complete long-form answer would otherwise lose. Do not repeat a source already likely to fit the compact packet, add redundant corroboration, or fill a quota. This inventory is bounded source access for long-form editorial memory, not permission to expand the standard author packet."
	}
	schema := sourceSelectionSchemaBytes(candidate, longForm)
	prompt := stagePrompt("il_source_selection", fmt.Sprintf(`Rank every usable frozen accepted source for a compact, sufficient author packet. Return every source key exactly once, strongest first.

Rank by the value of fully reading the source for this exact report:
- direct relevance to the requested subject and direction
- authority, specificity, and support for central claims or necessary uncertainty
- unique information not already supplied by a stronger source
- complete readable byte cost: a very large source must rank below smaller direct sources unless its unique value is essential

First identify every distinct positively requested subject, relationship, time period, structure, example, and context in the report direction. Rank at least one direct source for each supported requirement ahead of redundant sources for an already-covered requirement. A central source does not displace a source that uniquely supplies a different requested dimension.

Rank sources for the author to read completely. This stage chooses a compact reading set; it does not write, summarize, or validate the report's facts.

%s

The server will take at most %d sources and target at most %d complete readable bytes for the compact packet. Do not rank a broad primary text or general corpus highly merely because it mentions the subject; prefer direct official, scholarly, or focused sources that can support the report efficiently.

Report title: %s
Report direction: %s
Candidate source catalog SHA-256: %s
%s`, supplementGuidance, sourceSelectionMaxSources, sourceSelectionTargetReadBytes, config.Title, config.Direction, candidate.SHA256, sourceSelectionToolContract(sampleBytes)))
	draft, results, err := runJSONStageWithSourceLimit(ctx, config, candidate, "il_source_selection", prompt, schema, sampleBytes, func(value sourceSelectionDraft, readReceipt reportilcontract.SourceReadReceipt) error {
		if err := validateSourceSelectionReads(candidate, readReceipt, sampleBytes); err != nil {
			return withValidationCode(reportexecution.ProviderValidationCodeSourceReadContract, err)
		}
		return validateSourceSelectionDraft(candidate, value)
	})
	if err != nil {
		return SourceSelectionResult{}, SourceSelectionReceipt{}, results, err
	}
	selected, selectedOrdinals, err := CompileSelectedSourceCatalog(
		candidate,
		draft.RankedSourceKeys,
	)
	if err != nil {
		return SourceSelectionResult{}, SourceSelectionReceipt{}, results, err
	}
	var supplementKeys []string
	if longForm {
		supplementKeys = draft.LongFormSupplementSourceKeys
	}
	supplement, err := compileLongFormSupplementCatalog(
		candidate,
		supplementKeys,
		selectedOrdinals,
		sourceCatalogReadableBytes(selected),
	)
	if err != nil {
		return SourceSelectionResult{}, SourceSelectionReceipt{}, results, err
	}
	authorCatalog, err := mergeSourceCatalogs(selected, supplement)
	if err != nil {
		return SourceSelectionResult{}, SourceSelectionReceipt{}, results, err
	}
	if err := validateCompleteSourceCatalogBudget(authorCatalog); err != nil {
		return SourceSelectionResult{}, SourceSelectionReceipt{}, results, err
	}
	selectedSet := make(map[int]bool, len(selectedOrdinals))
	for _, ordinal := range selectedOrdinals {
		selectedSet[ordinal] = true
	}
	supplementSet := make(map[int]bool, len(supplement.Sources))
	for _, entry := range supplement.Sources {
		supplementSet[entry.AcceptedOrdinal] = true
	}
	for index := range build.Dispositions {
		disposition := build.Dispositions[index]
		if disposition.Status == "image_only" {
			receipt.Dispositions[index].Status = "image_only"
			receipt.Dispositions[index].Reason = disposition.Reason
			continue
		}
		if disposition.Status != "available" {
			receipt.Dispositions[index].Status = "excluded"
			receipt.Dispositions[index].Reason = disposition.Reason
			receipt.ExcludedUnusableSources++
			continue
		}
		switch {
		case selectedSet[disposition.AcceptedOrdinal]:
			receipt.Dispositions[index].Status = "selected"
		case supplementSet[disposition.AcceptedOrdinal]:
			receipt.Dispositions[index].Status = "supplemental"
		default:
			receipt.Dispositions[index].Status = "excluded"
			receipt.Dispositions[index].Reason = "selection_budget"
			receipt.ExcludedBudgetSources++
		}
	}
	receipt.SelectedSources = len(selected.Sources)
	receipt.SupplementalSources = len(supplement.Sources)
	receipt.SelectedReadableBytes = sourceCatalogReadableBytes(selected)
	receipt.SupplementalReadableBytes = sourceCatalogReadableBytes(supplement)
	receipt.SelectedCatalogSHA256 = selected.SHA256
	receipt.AuthorCatalogSHA256 = authorCatalog.SHA256
	receipt.DispositionSHA256 = sourceSelectionDispositionSHA(receipt.Dispositions)
	return SourceSelectionResult{
		Catalog: selected, Supplement: supplement, AuthorCatalog: authorCatalog,
	}, receipt, results, nil
}

func sourceSelectionReceipt(build SourceCatalogBuild, candidate reportilcontract.SourceCatalog) SourceSelectionReceipt {
	receipt := SourceSelectionReceipt{
		AcceptedSources:        len(build.Dispositions),
		UsableSources:          len(candidate.Sources),
		CandidateCatalogSHA256: candidate.SHA256,
		Dispositions:           make([]SourceSelectionDisposition, len(build.Dispositions)),
	}
	for index, disposition := range build.Dispositions {
		receipt.Dispositions[index] = SourceSelectionDisposition{
			AcceptedOrdinal: disposition.AcceptedOrdinal,
			Status:          disposition.Status,
			Reason:          disposition.Reason,
		}
	}
	return receipt
}

func sourceSelectionSchemaBytes(catalog reportilcontract.SourceCatalog, longForm ...bool) []byte {
	keys := make([]any, 0, len(catalog.Sources))
	for _, entry := range catalog.Sources {
		keys = append(keys, entry.SourceKey)
	}
	properties := map[string]any{
		"ranked_source_keys": map[string]any{
			"type":     "array",
			"minItems": len(keys),
			"maxItems": len(keys),
			"items":    map[string]any{"type": "string", "enum": keys},
		},
	}
	required := []any{"ranked_source_keys"}
	if len(longForm) > 0 && longForm[0] {
		required = append(required, "long_form_supplement_source_keys")
		properties["long_form_supplement_source_keys"] = map[string]any{
			"type":     "array",
			"maxItems": sourceSelectionMaxSources,
			"items":    map[string]any{"type": "string", "enum": keys},
		}
	}
	return marshalProviderSchema(providerSchemaSourceSelection, map[string]any{
		"$defs": map[string]any{
			"source_selection": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             required,
				"properties":           properties,
			},
		},
		"$ref": "#/$defs/source_selection",
	})
}

func sourceSelectionSampleLimit(catalog reportilcontract.SourceCatalog) (int, error) {
	if len(catalog.Sources) == 0 {
		return 0, fmt.Errorf("source selection requires at least one usable source")
	}
	limit := reportilcontract.DefaultSourceReadMaxBytes / len(catalog.Sources)
	if limit < utf8.UTFMax {
		return 0, fmt.Errorf("source selection cannot sample every usable source on a UTF-8 boundary within one source-read call")
	}
	if limit > sourceSelectionSampleBytes {
		limit = sourceSelectionSampleBytes
	}
	return limit, nil
}

func sourceSelectionToolContract(sampleBytes int) string {
	return fmt.Sprintf(`SOURCE ACCESS CONTRACT:
- Source content is intentionally absent from this prompt.
- First call plasma.report_il.sources.list with {}.
- Then call plasma.report_il.sources.read with {} until remaining_sources is zero. The server returns catalog-ordered batches within the per-call ceiling and samples up to %d bytes per source; do not request individual keys, offsets, or sizes.
- Use only plasma.report_il.sources.list and plasma.report_il.sources.read. Do not use or request connectors, URLs, paths, ambient knowledge, or any other tool.
- Do not reproduce source identifiers except in ranked_source_keys.`, sampleBytes)
}

func validateSourceSelectionReads(catalog reportilcontract.SourceCatalog, receipt reportilcontract.SourceReadReceipt, sampleBytes int) error {
	if err := receipt.Validate(catalog); err != nil {
		return err
	}
	if receipt.ReadBytesBySource == nil {
		return fmt.Errorf("source selection read receipt lacks per-source coverage")
	}
	for _, entry := range catalog.Sources {
		required := sampleBytes
		if entry.ReadableBytes < required {
			required = entry.ReadableBytes
		}
		readBytes := receipt.ReadBytesBySource[entry.SourceKey]
		minimumRequired := required - (utf8.UTFMax - 1)
		if minimumRequired < 1 {
			minimumRequired = 1
		}
		if readBytes > required || readBytes < minimumRequired {
			return fmt.Errorf("source selection did not review the exact UTF-8-aligned sample for every usable accepted source")
		}
		if entry.ReadableBytes <= sampleBytes && !receipt.IncludesEntireSource(entry.SourceKey) {
			return fmt.Errorf("source selection did not read a short usable accepted source to EOF")
		}
	}
	return nil
}

func validateSourceSelectionDraft(catalog reportilcontract.SourceCatalog, draft sourceSelectionDraft) error {
	if len(draft.RankedSourceKeys) != len(catalog.Sources) {
		return fmt.Errorf("source selection ranking does not cover the complete usable catalog")
	}
	seen := make(map[string]bool, len(draft.RankedSourceKeys))
	for _, key := range draft.RankedSourceKeys {
		if seen[key] {
			return fmt.Errorf("source selection ranking contains a duplicate source")
		}
		seen[key] = true
		if _, ok := catalog.Entry(key); !ok {
			return fmt.Errorf("source selection ranking contains an unavailable source")
		}
	}
	if len(draft.LongFormSupplementSourceKeys) > sourceSelectionMaxSources {
		return fmt.Errorf("long-form source supplement exceeds the source-count ceiling")
	}
	supplementSeen := make(map[string]bool, len(draft.LongFormSupplementSourceKeys))
	for _, key := range draft.LongFormSupplementSourceKeys {
		if supplementSeen[key] {
			return fmt.Errorf("long-form source supplement contains a duplicate source")
		}
		supplementSeen[key] = true
		if _, ok := catalog.Entry(key); !ok {
			return fmt.Errorf("long-form source supplement contains an unavailable source")
		}
	}
	return nil
}

func sourceCatalogFitsSelectionTarget(catalog reportilcontract.SourceCatalog) bool {
	return len(catalog.Sources) <= sourceSelectionMaxSources &&
		sourceCatalogReadableBytes(catalog) <= sourceSelectionTargetReadBytes
}

func sourceCatalogHasSelectableEntry(catalog reportilcontract.SourceCatalog) bool {
	for _, entry := range catalog.Sources {
		if entry.ReadableBytes <= reportilcontract.DefaultSourceAttemptReadBytes {
			return true
		}
	}
	return false
}

func compileSelectedSourceCatalog(
	catalog reportilcontract.SourceCatalog,
	ranked []string,
) (reportilcontract.SourceCatalog, []int, error) {
	return CompileSelectedSourceCatalog(catalog, ranked)
}

// CompileSelectedSourceCatalog deterministically reconstructs the compact
// author catalog from a verified legacy ranking trace.
func CompileSelectedSourceCatalog(
	catalog reportilcontract.SourceCatalog,
	ranked []string,
) (reportilcontract.SourceCatalog, []int, error) {
	if err := validateSourceSelectionDraft(catalog, sourceSelectionDraft{
		RankedSourceKeys: ranked,
	}); err != nil {
		return reportilcontract.SourceCatalog{}, nil, err
	}
	remaining := sourceSelectionTargetReadBytes
	selectedEntries := make([]reportilcontract.SourceCatalogEntry, 0, min(len(ranked), sourceSelectionMaxSources))
	selectedKeys := make(map[string]bool, len(ranked))
	for _, key := range ranked {
		if len(selectedEntries) == sourceSelectionMaxSources {
			break
		}
		if selectedKeys[key] {
			continue
		}
		entry, _ := catalog.Entry(key)
		if entry.ReadableBytes > remaining {
			continue
		}
		selectedKeys[key] = true
		selectedEntries = append(selectedEntries, entry)
		remaining -= entry.ReadableBytes
	}
	selectedSet := make(map[string]reportilcontract.SourceCatalogEntry, len(selectedEntries))
	for _, entry := range selectedEntries {
		selectedSet[entry.SourceKey] = entry
	}
	selectedEntries = selectedEntries[:0]
	selectedOrdinals := make([]int, 0, len(selectedSet))
	for _, key := range ranked {
		entry, ok := selectedSet[key]
		if !ok {
			continue
		}
		selectedEntries = append(selectedEntries, entry)
		selectedOrdinals = append(selectedOrdinals, entry.AcceptedOrdinal)
	}
	if len(selectedEntries) == 0 {
		for _, key := range ranked {
			entry, _ := catalog.Entry(key)
			if entry.ReadableBytes <= reportilcontract.DefaultSourceAttemptReadBytes {
				selectedOrdinals = append(selectedOrdinals, entry.AcceptedOrdinal)
				selectedEntries = append(selectedEntries, entry)
				break
			}
		}
	}
	for index := range selectedEntries {
		selectedEntries[index].SourceKey = fmt.Sprintf("source_%03d", index+1)
	}
	if len(selectedEntries) == 0 {
		return reportilcontract.SourceCatalog{}, nil, fmt.Errorf("source selection could not fit any usable source within the complete-read budget")
	}
	selected, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{
		MissionID: catalog.MissionID,
		Sources:   selectedEntries,
	})
	if err != nil {
		return reportilcontract.SourceCatalog{}, nil, err
	}
	return selected, selectedOrdinals, nil
}

func compileLongFormSupplementCatalog(
	catalog reportilcontract.SourceCatalog,
	keys []string,
	selectedOrdinals []int,
	selectedReadableBytes int,
) (reportilcontract.SourceCatalog, error) {
	if len(keys) == 0 {
		return reportilcontract.SourceCatalog{}, nil
	}
	if selectedReadableBytes < 0 || selectedReadableBytes > reportilcontract.DefaultSourceAttemptReadBytes {
		return reportilcontract.SourceCatalog{}, fmt.Errorf("selected source catalog exceeds the complete-read attempt ceiling")
	}
	selected := make(map[int]bool, len(selectedOrdinals))
	for _, ordinal := range selectedOrdinals {
		selected[ordinal] = true
	}
	entries := make([]reportilcontract.SourceCatalogEntry, 0, len(keys))
	seen := make(map[string]bool, len(keys))
	remaining := reportilcontract.DefaultSourceAttemptReadBytes - selectedReadableBytes
	for _, key := range keys {
		if seen[key] {
			return reportilcontract.SourceCatalog{}, fmt.Errorf("long-form source supplement contains a duplicate source")
		}
		seen[key] = true
		entry, ok := catalog.Entry(key)
		if !ok {
			return reportilcontract.SourceCatalog{}, fmt.Errorf("long-form source supplement contains an unavailable source")
		}
		if selected[entry.AcceptedOrdinal] || entry.ReadableBytes > remaining {
			continue
		}
		entries = append(entries, entry)
		remaining -= entry.ReadableBytes
	}
	if len(entries) == 0 {
		return reportilcontract.SourceCatalog{}, nil
	}
	for index := range entries {
		entries[index].SourceKey = fmt.Sprintf("source_%03d", index+1)
	}
	return reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{
		MissionID: catalog.MissionID,
		Sources:   entries,
	})
}

func mergeSourceCatalogs(
	primary reportilcontract.SourceCatalog,
	supplement reportilcontract.SourceCatalog,
) (reportilcontract.SourceCatalog, error) {
	if len(supplement.Sources) == 0 {
		return primary, nil
	}
	if primary.MissionID != supplement.MissionID {
		return reportilcontract.SourceCatalog{}, fmt.Errorf("long-form source supplement belongs to a different mission")
	}
	entries := make([]reportilcontract.SourceCatalogEntry, 0, len(primary.Sources)+len(supplement.Sources))
	entries = append(entries, primary.Sources...)
	seenOrdinals := make(map[int]bool, len(entries))
	for _, entry := range entries {
		seenOrdinals[entry.AcceptedOrdinal] = true
	}
	for _, entry := range supplement.Sources {
		if seenOrdinals[entry.AcceptedOrdinal] {
			continue
		}
		entries = append(entries, entry)
		seenOrdinals[entry.AcceptedOrdinal] = true
	}
	for index := range entries {
		entries[index].SourceKey = fmt.Sprintf("source_%03d", index+1)
	}
	return reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{
		MissionID: primary.MissionID,
		Sources:   entries,
	})
}

func sourceCatalogReadableBytes(catalog reportilcontract.SourceCatalog) int {
	total := 0
	for _, entry := range catalog.Sources {
		total += entry.ReadableBytes
	}
	return total
}

func sourceSelectionDispositionSHA(dispositions []SourceSelectionDisposition) string {
	ordered := append([]SourceSelectionDisposition(nil), dispositions...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].AcceptedOrdinal < ordered[j].AcceptedOrdinal })
	return sha256Hex(mustMarshal(ordered))
}
