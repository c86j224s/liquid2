package reportilphase0

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

const maxSourceExcerptBytes = reportilcontract.MaxSourceQuoteBytes

type authorEvidenceDraft struct {
	SectionKey    string `json:"section_key"`
	BlockKey      string `json:"block_key"`
	Claim         string `json:"claim"`
	SourceKey     string `json:"source_key"`
	SourceReceipt string `json:"source_receipt"`
	SourceExcerpt string `json:"-"`
	Preserve      bool   `json:"preserve"`
}

type EvidencePacket struct {
	EvidenceID       string `json:"evidence_id"`
	Claim            string `json:"claim"`
	ClaimSHA         string `json:"claim_sha256"`
	AcceptedOrdinal  int    `json:"accepted_ordinal"`
	SourceExcerptSHA string `json:"source_excerpt_sha256"`
	SourceOffset     int    `json:"source_offset"`
	SourceByteSize   int    `json:"source_byte_size"`
	AuthorNodeID     string `json:"author_node_id"`
	AuthorNodeSHA    string `json:"author_node_sha256"`
	Preserve         bool   `json:"preserve"`
	SourceExcerpt    string `json:"-"`
}

type EvidenceSupportCoverageReceipt struct {
	EvidenceID         string `json:"evidence_id"`
	EvidenceReceipt    string `json:"evidence_receipt"`
	SourceExcerptSHA   string `json:"source_excerpt_sha256"`
	ClaimSHA           string `json:"claim_sha256"`
	CoveredNodeID      string `json:"covered_node_id"`
	SupportLevel       string `json:"support_level"`
	CoverageQuoteSHA   string `json:"coverage_quote_sha256"`
	CoverageValueIndex int    `json:"coverage_value_index"`
	CoverageByteOffset int    `json:"coverage_byte_offset"`
	CoverageByteSize   int    `json:"coverage_byte_size"`
}

type PreservationCoverageReceipt struct {
	EvidenceID       string `json:"evidence_id"`
	EvidenceReceipt  string `json:"evidence_receipt"`
	SourceExcerptSHA string `json:"source_excerpt_sha256"`
	CoveredNodeID    string `json:"covered_node_id"`
	CoverageQuoteSHA string `json:"coverage_quote_sha256"`
}

func evidenceSourceReadSpans(
	catalog reportilcontract.SourceCatalog,
	packets []EvidencePacket,
) ([]reportilcontract.SourceReadSpan, error) {
	type sourceSpan struct {
		sourceKey string
		offset    int
		content   []byte
	}
	candidates := make([]sourceSpan, 0, len(packets))
	for _, packet := range packets {
		sourceKey := sourceKeyByAcceptedOrdinal(catalog, packet.AcceptedOrdinal)
		content := []byte(packet.SourceExcerpt)
		if sourceKey == "" || len(content) != packet.SourceByteSize ||
			SHA256(content) != packet.SourceExcerptSHA {
			return nil, fmt.Errorf("evidence packet source span is invalid")
		}
		candidates = append(candidates, sourceSpan{
			sourceKey: sourceKey,
			offset:    packet.SourceOffset,
			content:   append([]byte(nil), content...),
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].sourceKey != candidates[j].sourceKey {
			return candidates[i].sourceKey < candidates[j].sourceKey
		}
		return candidates[i].offset < candidates[j].offset
	})
	merged := make([]sourceSpan, 0, len(candidates))
	for _, candidate := range candidates {
		if len(merged) == 0 || merged[len(merged)-1].sourceKey != candidate.sourceKey ||
			candidate.offset > merged[len(merged)-1].offset+len(merged[len(merged)-1].content) {
			merged = append(merged, candidate)
			continue
		}
		previous := &merged[len(merged)-1]
		previousEnd := previous.offset + len(previous.content)
		candidateEnd := candidate.offset + len(candidate.content)
		overlapEnd := previousEnd
		if candidateEnd < overlapEnd {
			overlapEnd = candidateEnd
		}
		overlap := overlapEnd - candidate.offset
		previousOverlap := candidate.offset - previous.offset
		if overlap > 0 && string(previous.content[previousOverlap:previousOverlap+overlap]) != string(candidate.content[:overlap]) {
			return nil, fmt.Errorf("overlapping evidence packet source spans disagree")
		}
		if candidateEnd > previousEnd {
			previous.content = append(previous.content, candidate.content[overlap:]...)
		}
	}
	spans := make([]reportilcontract.SourceReadSpan, 0, len(merged))
	for _, span := range merged {
		for start := 0; start < len(span.content); {
			end := start + reportilcontract.MaxSourceReadSpanBytes
			if end > len(span.content) {
				end = len(span.content)
			}
			for end > start && !utf8.Valid(span.content[start:end]) {
				end--
			}
			if end == start {
				return nil, fmt.Errorf("merged evidence packet source span is not valid UTF-8")
			}
			content := span.content[start:end]
			spans = append(spans, reportilcontract.SourceReadSpan{
				SourceKey: span.sourceKey,
				Offset:    span.offset + start,
				ByteSize:  len(content),
				SHA256:    SHA256(content),
			})
			start = end
		}
	}
	if len(spans) > reportilcontract.MaxSourceReadSpans {
		return nil, fmt.Errorf("evidence packet source spans exceed the read ceiling")
	}
	return spans, nil
}

func validateEvidenceSourceReads(
	catalog reportilcontract.SourceCatalog,
	packets []EvidencePacket,
	receipt reportilcontract.SourceReadReceipt,
) error {
	if err := receipt.Validate(catalog); err != nil {
		return err
	}
	spans, err := evidenceSourceReadSpans(catalog, packets)
	if err != nil {
		return err
	}
	if len(receipt.ReadRanges) != len(spans) {
		return fmt.Errorf("final reader did not read every bound evidence span")
	}
	for index, span := range spans {
		read := receipt.ReadRanges[index]
		if read.SourceKey != span.SourceKey || read.Offset != span.Offset || read.ByteSize != span.ByteSize {
			return fmt.Errorf("final reader source read does not match the evidence packet")
		}
	}
	return nil
}

func evidenceReceipt(packet EvidencePacket) string {
	return SHA256(mustMarshal(struct {
		EvidenceID       string `json:"evidence_id"`
		Claim            string `json:"claim"`
		ClaimSHA         string `json:"claim_sha256"`
		AcceptedOrdinal  int    `json:"accepted_ordinal"`
		SourceExcerptSHA string `json:"source_excerpt_sha256"`
		SourceOffset     int    `json:"source_offset"`
		SourceByteSize   int    `json:"source_byte_size"`
		AuthorNodeID     string `json:"author_node_id"`
		AuthorNodeSHA    string `json:"author_node_sha256"`
		Preserve         bool   `json:"preserve"`
	}{
		EvidenceID: packet.EvidenceID, Claim: packet.Claim, ClaimSHA: packet.ClaimSHA,
		AcceptedOrdinal: packet.AcceptedOrdinal, SourceExcerptSHA: packet.SourceExcerptSHA,
		SourceOffset: packet.SourceOffset, SourceByteSize: packet.SourceByteSize,
		AuthorNodeID: packet.AuthorNodeID, AuthorNodeSHA: packet.AuthorNodeSHA,
		Preserve: packet.Preserve,
	}))
}

func compileAuthorEvidence(
	drafts []authorEvidenceDraft,
	document Document,
	catalog reportilcontract.SourceCatalog,
	receipt reportilcontract.SourceReadReceipt,
	readableByAcceptedOrdinal map[int]string,
) ([]EvidencePacket, error) {
	if len(drafts) == 0 {
		return nil, nil
	}
	if len(drafts) > reportilcontract.MaxSourceReadSpans {
		return nil, withValidationCode(
			reportexecution.ProviderValidationCodeEvidencePacketInventory,
			fmt.Errorf("author evidence packet exceeds the technical ceiling"),
		)
	}
	sections := readerSections(document)
	results := make([]EvidencePacket, 0, len(drafts))
	seenPacketIndexes := map[string]int{}
	seenBindings := map[string]bool{}
	seenPreservedClaims := map[string]bool{}
	seenSourceReceipts := map[string]bool{}
	invalidSourceAliases := []string{}
	for index, draft := range drafts {
		sectionIndex, blockIndex, ok := readerAliasIndexes(draft.SectionKey, draft.BlockKey)
		if !ok || sectionIndex >= len(sections) || blockIndex >= len(sections[sectionIndex].Blocks) {
			return nil, withValidationCode(
				reportexecution.ProviderValidationCodeEvidencePacketTarget,
				fmt.Errorf("author evidence packet target is invalid"),
			)
		}
		block := sections[sectionIndex].Blocks[blockIndex]
		claim := strings.TrimSpace(draft.Claim)
		entry, ok := catalog.Entry(draft.SourceKey)
		if !ok || claim == "" || validateProviderContent(claim) != nil ||
			!blockContainsExactQuote(block, claim) ||
			!documentNodeHasEvidenceOrdinals(document, block.NodeID, []int{entry.AcceptedOrdinal}) {
			return nil, withValidationCode(
				reportexecution.ProviderValidationCodeEvidencePacketBinding,
				fmt.Errorf("author evidence packet does not match manuscript evidence"),
			)
		}
		readable := readableByAcceptedOrdinal[entry.AcceptedOrdinal]
		sourceExcerpt := strings.TrimSpace(draft.SourceExcerpt)
		sourceOffset := -1
		if sourceReceipt := strings.TrimSpace(draft.SourceReceipt); sourceReceipt != "" {
			if seenSourceReceipts[sourceReceipt] {
				return nil, withValidationCode(
					reportexecution.ProviderValidationCodeEvidencePacketInventory,
					fmt.Errorf("author evidence packet reuses a source receipt"),
				)
			}
			seenSourceReceipts[sourceReceipt] = true
			quote, registered := receipt.SourceQuotes[sourceReceipt]
			if !registered || quote.SourceKey != draft.SourceKey ||
				quote.Offset < 0 || quote.ByteSize < 1 || quote.ByteSize > maxSourceExcerptBytes ||
				quote.Offset > len([]byte(readable))-quote.ByteSize {
				invalidSourceAliases = append(
					invalidSourceAliases, fmt.Sprintf("evidence_%03d", index+1),
				)
				continue
			}
			sourceBytes := []byte(readable)[quote.Offset : quote.Offset+quote.ByteSize]
			sourceExcerpt = string(sourceBytes)
			sourceOffset = quote.Offset
			if !utf8.Valid(sourceBytes) || strings.TrimSpace(sourceExcerpt) == "" ||
				SHA256(sourceBytes) != quote.SHA256 {
				invalidSourceAliases = append(
					invalidSourceAliases, fmt.Sprintf("evidence_%03d", index+1),
				)
				continue
			}
		} else if sourceExcerpt != "" {
			// Legacy in-process fixtures may still supply an excerpt directly. The
			// provider wire schema has no source_excerpt field.
			sourceOffset = strings.Index(readable, sourceExcerpt)
		}
		if sourceExcerpt == "" || !utf8.ValidString(sourceExcerpt) ||
			len([]byte(sourceExcerpt)) > maxSourceExcerptBytes || sourceOffset < 0 {
			invalidSourceAliases = append(
				invalidSourceAliases, fmt.Sprintf("evidence_%03d", index+1),
			)
			continue
		}
		packetKey := block.NodeID + "\x00" + draft.SourceKey + "\x00" + claim
		// Quote calls are bounded candidate selection, while evidence packets are the
		// final inventory. Canonicalize repeated valid packet meanings only after each
		// referenced receipt has independently passed source, range, hash, and reuse checks.
		if existingIndex, duplicate := seenPacketIndexes[packetKey]; duplicate {
			if draft.Preserve && !results[existingIndex].Preserve &&
				!seenPreservedClaims[strings.ToLower(claim)] {
				results[existingIndex].Preserve = true
				seenPreservedClaims[strings.ToLower(claim)] = true
			}
			continue
		}
		if draft.Preserve && seenPreservedClaims[strings.ToLower(claim)] {
			draft.Preserve = false
		}
		preimage := fmt.Sprintf("%s\x00%d\x00%d\x00%s", catalog.SHA256, index, entry.AcceptedOrdinal, claim)
		sum := sha256.Sum256([]byte(preimage))
		packet := EvidencePacket{
			EvidenceID: "evidence." + hex.EncodeToString(sum[:8]), Claim: claim,
			ClaimSHA: SHA256([]byte(claim)), AcceptedOrdinal: entry.AcceptedOrdinal,
			SourceExcerptSHA: SHA256([]byte(sourceExcerpt)),
			SourceOffset:     sourceOffset, SourceByteSize: len([]byte(sourceExcerpt)),
			AuthorNodeID: block.NodeID, AuthorNodeSHA: readerBlockSHA256(block),
			Preserve: draft.Preserve, SourceExcerpt: sourceExcerpt,
		}
		results = append(results, packet)
		seenPacketIndexes[packetKey] = len(results) - 1
		seenBindings[block.NodeID+"\x00"+draft.SourceKey] = true
		if draft.Preserve {
			seenPreservedClaims[strings.ToLower(claim)] = true
		}
	}
	if len(invalidSourceAliases) > 0 {
		return nil, withValidationCodeAndPackets(
			reportexecution.ProviderValidationCodeEvidencePacketSource,
			invalidSourceAliases,
			fmt.Errorf("author evidence packet is not source-grounded"),
		)
	}
	for _, block := range document.Blocks {
		if block.Kind == "section" {
			continue
		}
		for _, ordinal := range documentNodeEvidenceOrdinals(document, block.NodeID) {
			bindingKey := block.NodeID + "\x00" + sourceKeyByAcceptedOrdinal(catalog, ordinal)
			if !seenBindings[bindingKey] {
				return nil, withValidationCode(
					reportexecution.ProviderValidationCodeEvidencePacketBinding,
					fmt.Errorf("author evidence packet does not cover every manuscript source binding"),
				)
			}
		}
		if len(documentNodeEvidenceOrdinals(document, block.NodeID)) > 0 &&
			!blockClaimsCoverReaderContent(block, results) {
			return nil, withValidationCode(
				reportexecution.ProviderValidationCodeEvidencePacketCoverage,
				fmt.Errorf("author evidence packets do not cover the complete factual leaf"),
			)
		}
	}
	if _, err := evidenceSourceReadSpans(catalog, results); err != nil {
		return nil, withValidationCode(reportexecution.ProviderValidationCodeEvidencePacketInventory, err)
	}
	return results, nil
}

func blockReaderValues(block Block) []string {
	switch block.Kind {
	case "prose", "quote", "callout":
		return []string{block.Prose}
	case "list":
		return block.Items
	case "code":
		return []string{block.Code}
	case "equation":
		if block.Equation == nil {
			return nil
		}
		return []string{block.Equation.Expression}
	case "table":
		if block.Table == nil {
			return nil
		}
		values := []string{block.Table.Caption}
		values = append(values, block.Table.Columns...)
		for _, row := range block.Table.Rows {
			values = append(values, row...)
		}
		return values
	default:
		return nil
	}
}

func blockFactualReaderValues(block Block) []string {
	if block.Kind != "table" || block.Table == nil {
		return blockReaderValues(block)
	}
	values := []string{}
	for _, row := range block.Table.Rows {
		values = append(values, row...)
	}
	return values
}

func blockClaimsCoverReaderContent(block Block, packets []EvidencePacket) bool {
	claims := []string{}
	for _, packet := range packets {
		if packet.AuthorNodeID == block.NodeID {
			claims = append(claims, packet.Claim)
		}
	}
	if len(claims) == 0 {
		return false
	}
	for _, value := range blockFactualReaderValues(block) {
		covered := make([]bool, len(value))
		for _, claim := range claims {
			for offset := 0; offset <= len(value)-len(claim); {
				index := strings.Index(value[offset:], claim)
				if index < 0 {
					break
				}
				start := offset + index
				for position := start; position < start+len(claim); position++ {
					covered[position] = true
				}
				offset = start + len(claim)
			}
		}
		for offset, runeValue := range value {
			if !unicode.IsSpace(runeValue) && !unicode.IsPunct(runeValue) && !covered[offset] {
				return false
			}
		}
	}
	return true
}

func compileEvidenceSupportCoverage(
	document Document,
	drafts map[string]readerEvidenceSupportCoverageDraft,
	packets []EvidencePacket,
) ([]EvidenceSupportCoverageReceipt, error) {
	if len(drafts) != len(packets) {
		return nil, withValidationCodeAndPackets(
			reportexecution.ProviderValidationCodeEvidenceSupportInventory,
			missingEvidenceSupportAliases(drafts, packets),
			fmt.Errorf("reader evidence support inventory changed"),
		)
	}
	sections := readerSections(document)
	result := make([]EvidenceSupportCoverageReceipt, 0, len(packets))
	for index, packet := range packets {
		key := fmt.Sprintf("evidence_%03d", index+1)
		draft, ok := drafts[key]
		if !ok || draft.EvidenceReceipt != evidenceReceipt(packet) {
			return nil, withValidationCodeAndPackets(
				reportexecution.ProviderValidationCodeEvidenceSupportReceipt,
				[]string{key},
				fmt.Errorf("reader evidence support receipt is invalid"),
			)
		}
		sectionIndex, blockIndex, ok := readerAliasIndexes(draft.SectionKey, draft.BlockKey)
		if !ok || sectionIndex >= len(sections) || blockIndex >= len(sections[sectionIndex].Blocks) {
			return nil, withValidationCodeAndPackets(
				reportexecution.ProviderValidationCodeEvidenceSupportTarget,
				[]string{key},
				fmt.Errorf("reader evidence support target is invalid"),
			)
		}
		block := sections[sectionIndex].Blocks[blockIndex]
		if block.NodeID != packet.AuthorNodeID ||
			!documentNodeHasEvidenceOrdinals(document, block.NodeID, []int{packet.AcceptedOrdinal}) {
			return nil, withValidationCodeAndPackets(
				reportexecution.ProviderValidationCodeEvidenceSupportBinding,
				[]string{key},
				fmt.Errorf("reader evidence support changed source binding"),
			)
		}
		claimPresent := blockContainsExactQuote(block, packet.Claim)
		coverageQuote := optionalProviderString(draft.CoverageQuote)
		coverageValueIndex, coverageByteOffset, coverageByteSize := 0, 0, 0
		switch draft.SupportLevel {
		case "direct_source_statement", "bounded_inference":
			var ok bool
			coverageValueIndex, coverageByteOffset, coverageByteSize, ok = blockExactQuoteRange(block, coverageQuote)
			if !ok || packet.Preserve && coverageQuote != packet.Claim {
				return nil, withValidationCodeAndPackets(
					reportexecution.ProviderValidationCodeEvidenceSupportQuote,
					[]string{key},
					fmt.Errorf("reader supported evidence quote is missing from the final block"),
				)
			}
		case "unsupported_removed":
			if packet.Preserve || claimPresent || draft.CoverageQuote != nil {
				return nil, withValidationCodeAndPackets(
					reportexecution.ProviderValidationCodeEvidenceSupportUnsupported,
					[]string{key},
					fmt.Errorf("reader did not remove unsupported evidence claim"),
				)
			}
		default:
			return nil, withValidationCodeAndPackets(
				reportexecution.ProviderValidationCodeEvidenceSupportLevel,
				[]string{key},
				fmt.Errorf("reader evidence support level is invalid"),
			)
		}
		result = append(result, EvidenceSupportCoverageReceipt{
			EvidenceID: packet.EvidenceID, EvidenceReceipt: evidenceReceipt(packet),
			SourceExcerptSHA: packet.SourceExcerptSHA, ClaimSHA: packet.ClaimSHA,
			CoveredNodeID: block.NodeID, SupportLevel: draft.SupportLevel,
			CoverageQuoteSHA: SHA256([]byte(coverageQuote)), CoverageValueIndex: coverageValueIndex,
			CoverageByteOffset: coverageByteOffset, CoverageByteSize: coverageByteSize,
		})
	}
	if len(packets) > 0 {
		for _, block := range document.Blocks {
			if block.Kind == "section" || len(documentNodeEvidenceOrdinals(document, block.NodeID)) == 0 {
				continue
			}
			if !blockSupportCoverageComplete(block, result) {
				return nil, withValidationCodeAndPackets(
					reportexecution.ProviderValidationCodeEvidenceSupportCoverage,
					evidencePacketAliasesForNode(packets, block.NodeID),
					fmt.Errorf("reader evidence support does not cover the complete final factual leaf"),
				)
			}
		}
	}
	return result, nil
}

func missingEvidenceSupportAliases(
	drafts map[string]readerEvidenceSupportCoverageDraft,
	packets []EvidencePacket,
) []string {
	aliases := make([]string, 0)
	for index := range packets {
		alias := fmt.Sprintf("evidence_%03d", index+1)
		if _, ok := drafts[alias]; !ok {
			aliases = append(aliases, alias)
		}
	}
	return aliases
}

func evidencePacketAliasesForNode(packets []EvidencePacket, nodeID string) []string {
	aliases := make([]string, 0)
	for index, packet := range packets {
		if packet.AuthorNodeID == nodeID {
			aliases = append(aliases, fmt.Sprintf("evidence_%03d", index+1))
		}
	}
	return aliases
}

func compilePreservationCoverage(
	document Document,
	drafts map[string]readerPreservationCoverageDraft,
	packets []EvidencePacket,
) ([]PreservationCoverageReceipt, error) {
	preserved := make([]EvidencePacket, 0, len(packets))
	for _, packet := range packets {
		if packet.Preserve {
			preserved = append(preserved, packet)
		}
	}
	if len(drafts) != len(preserved) {
		return nil, withValidationCode(
			reportexecution.ProviderValidationCodeSupportedDetail,
			fmt.Errorf("reader preservation coverage inventory changed"),
		)
	}
	sections := readerSections(document)
	result := make([]PreservationCoverageReceipt, 0, len(preserved))
	for index, packet := range preserved {
		key := fmt.Sprintf("preserve_%02d", index+1)
		draft, ok := drafts[key]
		if !ok || draft.EvidenceReceipt != evidenceReceipt(packet) {
			return nil, withValidationCode(
				reportexecution.ProviderValidationCodeSupportedDetail,
				fmt.Errorf("reader preservation coverage receipt is invalid"),
			)
		}
		sectionIndex, blockIndex, ok := readerAliasIndexes(draft.SectionKey, draft.BlockKey)
		if !ok || sectionIndex >= len(sections) || blockIndex >= len(sections[sectionIndex].Blocks) {
			return nil, withValidationCode(
				reportexecution.ProviderValidationCodeSupportedDetail,
				fmt.Errorf("reader preservation coverage target is invalid"),
			)
		}
		block := sections[sectionIndex].Blocks[blockIndex]
		if block.NodeID != packet.AuthorNodeID ||
			!blockContainsExactQuote(block, packet.Claim) ||
			!documentNodeHasEvidenceOrdinals(document, block.NodeID, []int{packet.AcceptedOrdinal}) {
			return nil, withValidationCode(
				reportexecution.ProviderValidationCodeSupportedDetail,
				fmt.Errorf("reader removed or reassigned required source-backed detail"),
			)
		}
		result = append(result, PreservationCoverageReceipt{
			EvidenceID: packet.EvidenceID, EvidenceReceipt: evidenceReceipt(packet),
			SourceExcerptSHA: packet.SourceExcerptSHA, CoveredNodeID: block.NodeID,
			CoverageQuoteSHA: SHA256([]byte(packet.Claim)),
		})
	}
	return result, nil
}

func readerAliasIndexes(sectionKey, blockKey string) (int, int, bool) {
	var sectionNumber, blockNumber int
	if _, err := fmt.Sscanf(sectionKey, "section_%02d", &sectionNumber); err != nil ||
		sectionNumber < 1 {
		return 0, 0, false
	}
	if _, err := fmt.Sscanf(blockKey, "block_%02d", &blockNumber); err != nil || blockNumber < 1 {
		return 0, 0, false
	}
	return sectionNumber - 1, blockNumber - 1, true
}

func blockContainsExactQuote(block Block, quote string) bool {
	_, _, _, ok := blockExactQuoteRange(block, quote)
	return ok
}

func blockExactQuoteRange(block Block, quote string) (int, int, int, bool) {
	if quote == "" {
		return 0, 0, 0, false
	}
	for valueIndex, value := range blockFactualReaderValues(block) {
		if byteOffset := strings.Index(value, quote); byteOffset >= 0 {
			return valueIndex, byteOffset, len([]byte(quote)), true
		}
	}
	return 0, 0, 0, false
}

func blockQuoteRangeValid(block Block, valueIndex, byteOffset, byteSize int, quoteSHA string) bool {
	values := blockFactualReaderValues(block)
	if valueIndex < 0 || valueIndex >= len(values) || byteOffset < 0 || byteSize < 1 {
		return false
	}
	value := []byte(values[valueIndex])
	if byteOffset > len(value)-byteSize {
		return false
	}
	quote := value[byteOffset : byteOffset+byteSize]
	return utf8.Valid(quote) && SHA256(quote) == quoteSHA
}

func blockSupportCoverageComplete(
	block Block,
	coverage []EvidenceSupportCoverageReceipt,
) bool {
	values := blockFactualReaderValues(block)
	coveredByValue := make([][]bool, len(values))
	for valueIndex, value := range values {
		coveredByValue[valueIndex] = make([]bool, len([]byte(value)))
	}
	for _, item := range coverage {
		if item.CoveredNodeID != block.NodeID || item.SupportLevel == "unsupported_removed" ||
			item.CoverageByteSize < 1 {
			continue
		}
		for valueIndex, value := range values {
			valueBytes := []byte(value)
			for offset := 0; offset <= len(valueBytes)-item.CoverageByteSize; offset++ {
				end := offset + item.CoverageByteSize
				if SHA256(valueBytes[offset:end]) != item.CoverageQuoteSHA {
					continue
				}
				for position := offset; position < end; position++ {
					coveredByValue[valueIndex][position] = true
				}
			}
		}
	}
	for valueIndex, value := range values {
		for byteOffset, runeValue := range value {
			if !unicode.IsSpace(runeValue) && !unicode.IsPunct(runeValue) &&
				!coveredByValue[valueIndex][byteOffset] {
				return false
			}
		}
	}
	return true
}

func documentNodeEvidenceOrdinals(document Document, nodeID string) []int {
	refs := map[string]int{}
	for _, ref := range document.References {
		if ref.Kind != "footnote" || !strings.HasPrefix(ref.Target, "accepted-source:") {
			continue
		}
		var ordinal int
		if _, err := fmt.Sscanf(ref.Target, "accepted-source:%d", &ordinal); err == nil {
			refs[ref.RefID] = ordinal
		}
	}
	seen := map[int]bool{}
	ordinals := []int{}
	for _, block := range document.Blocks {
		if block.NodeID != nodeID {
			continue
		}
		for _, refID := range block.EvidenceRefs {
			if ordinal := refs[refID]; ordinal > 0 && !seen[ordinal] {
				seen[ordinal] = true
				ordinals = append(ordinals, ordinal)
			}
		}
	}
	sort.Ints(ordinals)
	return ordinals
}

func documentNodeHasEvidenceOrdinals(document Document, nodeID string, required []int) bool {
	available := map[int]bool{}
	for _, ordinal := range documentNodeEvidenceOrdinals(document, nodeID) {
		available[ordinal] = true
	}
	for _, ordinal := range required {
		if !available[ordinal] {
			return false
		}
	}
	return len(required) > 0
}
