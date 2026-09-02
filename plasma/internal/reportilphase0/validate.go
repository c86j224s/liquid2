package reportilphase0

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"net/url"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// LoadBundle decodes and validates a complete Phase 0 input before any output
// directory is created or compiler side effect occurs.
func LoadBundle(raw []byte) (Bundle, error) {
	bundle, err := decodeStrictJSON[Bundle](raw)
	if err != nil {
		return Bundle{}, fmt.Errorf("decode Phase 0 bundle: %w", err)
	}
	if err := ValidateBundle(bundle); err != nil {
		return Bundle{}, err
	}
	return bundle, nil
}

// ValidateBundle applies JSON Schema and cross-object semantic invariants.
func ValidateBundle(bundle Bundle) error {
	if bundle.SchemaVersion != BundleSchemaVersion {
		return fmt.Errorf("unsupported bundle schema %q", bundle.SchemaVersion)
	}
	if bundle.Arm != "T0" && bundle.Arm != "T1" && bundle.Arm != "T2" {
		return fmt.Errorf("unsupported experiment arm %q", bundle.Arm)
	}
	compiled, err := loadSchemas()
	if err != nil {
		return err
	}
	if err := validateDocumentSchema(bundle.Document, compiled.document); err != nil {
		return err
	}
	if bundle.Narrative != nil {
		narrativeValue, err := schemaValue(bundle.Narrative)
		if err != nil {
			return err
		}
		if err := compiled.narrative.Validate(narrativeValue); err != nil {
			return fmt.Errorf("narrative contract schema: %w", err)
		}
	}
	if bundle.Attestation != nil {
		attestationValue, err := schemaValue(bundle.Attestation)
		if err != nil {
			return err
		}
		if err := compiled.attestation.Validate(attestationValue); err != nil {
			return fmt.Errorf("flow attestation schema: %w", err)
		}
	}
	if err := validateArm(bundle); err != nil {
		return err
	}
	if err := validateNarrative(bundle.Narrative, bundle.Document); err != nil {
		return err
	}
	if err := validateDocument(bundle.Document); err != nil {
		return err
	}
	if err := validateAttestation(bundle.Attestation, bundle.Document); err != nil {
		return err
	}
	return validateEvidenceLineage(bundle.Narrative, bundle.Attestation, bundle.Document)
}

func validateCanonicalDocumentSchema(document Document) error {
	compiled, err := loadSchemas()
	if err != nil {
		return err
	}
	return validateDocumentSchema(document, compiled.document)
}

func validateDocumentSchema(document Document, compiled *jsonschema.Schema) error {
	documentValue, err := schemaValue(document)
	if err != nil {
		return err
	}
	if err := compiled.Validate(documentValue); err != nil {
		return fmt.Errorf("semantic IL schema: %w", err)
	}
	return nil
}

func validateArm(bundle Bundle) error {
	switch bundle.Arm {
	case "T0":
		if bundle.Narrative != nil || bundle.Attestation != nil || bundle.Document.NarrativeContractID != "" {
			return fmt.Errorf("T0 must not bind a narrative contract or flow attestation")
		}
	case "T1":
		if bundle.Narrative == nil || bundle.Attestation != nil {
			return fmt.Errorf("T1 requires a narrative contract and no flow attestation")
		}
	case "T2":
		if bundle.Narrative == nil || bundle.Attestation == nil {
			return fmt.Errorf("T2 requires a narrative contract and flow attestation")
		}
	}
	return nil
}

func validateNarrativeContract(narrative Narrative) error {
	document := Document{
		DocumentID:          narrative.DocumentID,
		NarrativeContractID: narrative.ContractID,
		Blocks:              make([]Block, 0, len(narrative.SectionRoles)),
	}
	for _, role := range narrative.SectionRoles {
		document.Blocks = append(document.Blocks, Block{NodeID: role.SectionID, Kind: "section"})
	}
	return validateNarrative(&narrative, document)
}

func validateNarrative(narrative *Narrative, document Document) error {
	if narrative == nil {
		return nil
	}
	if narrative.DocumentID != document.DocumentID || narrative.ContractID != document.NarrativeContractID {
		return fmt.Errorf("narrative contract binding does not match semantic IL")
	}
	sections := map[string]bool{}
	for _, block := range document.Blocks {
		if block.Kind == "section" && !(block.Level == 2 && documentSectionHasChild(document, block.NodeID)) {
			sections[block.NodeID] = true
		}
	}
	roleSections := map[string]bool{}
	roleLoopIntroductions := map[string]string{}
	roleCallbackResolutions := map[string]string{}
	for _, role := range narrative.SectionRoles {
		if !sections[role.SectionID] {
			return fmt.Errorf("narrative Section %q does not exist in semantic IL", role.SectionID)
		}
		if roleSections[role.SectionID] {
			return fmt.Errorf("duplicate narrative Section role %q", role.SectionID)
		}
		roleSections[role.SectionID] = true
		for _, prerequisite := range role.PrerequisiteSections {
			if !sections[prerequisite] || prerequisite == role.SectionID {
				return fmt.Errorf("invalid narrative prerequisite %q for %q", prerequisite, role.SectionID)
			}
		}
		for _, loopID := range role.OpenLoopsIntroduced {
			if previous := roleLoopIntroductions[loopID]; previous != "" {
				return fmt.Errorf("open loop %q is introduced by both %q and %q", loopID, previous, role.SectionID)
			}
			roleLoopIntroductions[loopID] = role.SectionID
		}
		for _, loopID := range role.CallbacksResolved {
			if previous := roleCallbackResolutions[loopID]; previous != "" {
				return fmt.Errorf("callback %q is resolved by both %q and %q", loopID, previous, role.SectionID)
			}
			roleCallbackResolutions[loopID] = role.SectionID
		}
	}
	if len(roleSections) != len(sections) {
		return fmt.Errorf("narrative contract requires exactly one role for every semantic IL Section")
	}
	for _, transition := range narrative.TransitionObligations {
		if !sections[transition.FromSectionID] || !sections[transition.ToSectionID] || transition.FromSectionID == transition.ToSectionID {
			return fmt.Errorf("invalid narrative transition %q -> %q", transition.FromSectionID, transition.ToSectionID)
		}
	}
	for _, edge := range narrative.DependencyEdges {
		if !sections[edge.FromSectionID] || !sections[edge.ToSectionID] || edge.FromSectionID == edge.ToSectionID {
			return fmt.Errorf("invalid narrative dependency %q -> %q", edge.FromSectionID, edge.ToSectionID)
		}
	}
	if err := validateDependencyAcyclic(narrative.DependencyEdges); err != nil {
		return err
	}
	loops := map[string]string{}
	for _, loop := range narrative.OpenLoops {
		if !sections[loop.IntroducedSectionID] || loops[loop.LoopID] != "" {
			return fmt.Errorf("invalid or duplicate open loop %q", loop.LoopID)
		}
		loops[loop.LoopID] = loop.IntroducedSectionID
	}
	callbacks := map[string]string{}
	for _, callback := range narrative.Callbacks {
		if loops[callback.LoopID] == "" || !sections[callback.ResolvedSectionID] || callbacks[callback.LoopID] != "" {
			return fmt.Errorf("invalid or duplicate callback for %q", callback.LoopID)
		}
		callbacks[callback.LoopID] = callback.ResolvedSectionID
	}
	for loopID, introducedSectionID := range loops {
		if callbacks[loopID] == "" {
			return fmt.Errorf("open loop %q has no callback obligation", loopID)
		}
		if roleLoopIntroductions[loopID] != introducedSectionID {
			return fmt.Errorf("open loop %q is not bound to its introducing Section role", loopID)
		}
		if roleCallbackResolutions[loopID] == "" {
			return fmt.Errorf("callback %q is not bound to a resolving Section role", loopID)
		}
		if roleCallbackResolutions[loopID] != callbacks[loopID] {
			return fmt.Errorf("callback %q resolving Section differs between callback and Section role", loopID)
		}
	}
	for loopID := range roleLoopIntroductions {
		if loops[loopID] == "" {
			return fmt.Errorf("Section role introduces undefined open loop %q", loopID)
		}
	}
	for loopID := range roleCallbackResolutions {
		if loops[loopID] == "" {
			return fmt.Errorf("Section role resolves undefined callback %q", loopID)
		}
	}
	seenEvidence := map[string]bool{}
	seenPacketClaims := map[string]bool{}
	coveredBindings := map[string]bool{}
	for _, packet := range narrative.EvidencePackets {
		block := documentBlockByID(document, packet.AuthorNodeID)
		bindingKey := fmt.Sprintf("%s\x00%d", packet.AuthorNodeID, packet.AcceptedOrdinal)
		packetKey := bindingKey + "\x00" + packet.Claim
		claimMustRemain := packet.AuthorNodeSHA == readerBlockSHA256(block) || packet.Preserve
		if packet.EvidenceID == "" || seenEvidence[packet.EvidenceID] || seenPacketClaims[packetKey] ||
			packet.Claim == "" || packet.ClaimSHA != SHA256([]byte(packet.Claim)) ||
			packet.AcceptedOrdinal < 1 || packet.SourceOffset < 0 || packet.SourceByteSize < 1 ||
			packet.SourceByteSize > maxSourceExcerptBytes || len(packet.SourceExcerptSHA) != 64 ||
			len(packet.AuthorNodeSHA) != 64 ||
			!sections[documentSectionForNode(document, packet.AuthorNodeID)] ||
			claimMustRemain && !blockContainsExactQuote(block, packet.Claim) ||
			!documentNodeHasEvidenceOrdinals(document, packet.AuthorNodeID, []int{packet.AcceptedOrdinal}) {
			return fmt.Errorf("narrative evidence packet is invalid")
		}
		seenEvidence[packet.EvidenceID] = true
		seenPacketClaims[packetKey] = true
		coveredBindings[bindingKey] = true
	}
	if len(narrative.EvidencePackets) > 0 {
		for _, block := range document.Blocks {
			for _, ordinal := range documentNodeEvidenceOrdinals(document, block.NodeID) {
				if !coveredBindings[fmt.Sprintf("%s\x00%d", block.NodeID, ordinal)] {
					return fmt.Errorf("narrative evidence packet does not cover every manuscript source binding")
				}
			}
		}
	}
	return nil
}

func documentBlockByID(document Document, nodeID string) Block {
	for _, block := range document.Blocks {
		if block.NodeID == nodeID {
			return block
		}
	}
	return Block{}
}

func documentSectionForNode(document Document, nodeID string) string {
	for _, block := range document.Blocks {
		if block.NodeID == nodeID {
			return block.ParentNodeID
		}
	}
	return ""
}

func validateDependencyAcyclic(edges []DependencyEdge) error {
	adjacency := map[string][]string{}
	for _, edge := range edges {
		adjacency[edge.FromSectionID] = append(adjacency[edge.FromSectionID], edge.ToSectionID)
	}
	state := map[string]uint8{}
	var visit func(string) bool
	visit = func(node string) bool {
		if state[node] == 1 {
			return false
		}
		if state[node] == 2 {
			return true
		}
		state[node] = 1
		for _, next := range adjacency[node] {
			if !visit(next) {
				return false
			}
		}
		state[node] = 2
		return true
	}
	for node := range adjacency {
		if !visit(node) {
			return fmt.Errorf("narrative dependency graph contains a cycle")
		}
	}
	return nil
}

func validateDocument(document Document) error {
	if document.PipelineFamily != PipelineFamily {
		return fmt.Errorf("semantic IL pipeline family must be %q", PipelineFamily)
	}
	nodes := map[string]Block{}
	for _, block := range document.Blocks {
		if _, exists := nodes[block.NodeID]; exists {
			return fmt.Errorf("duplicate node ID %q", block.NodeID)
		}
		nodes[block.NodeID] = block
	}
	assets := map[string]Asset{}
	for _, asset := range document.Assets {
		if _, exists := assets[asset.AssetID]; exists {
			return fmt.Errorf("duplicate asset ID %q", asset.AssetID)
		}
		if _, _, err := mime.ParseMediaType(asset.MediaType); err != nil {
			return fmt.Errorf("asset %q has invalid media type: %w", asset.AssetID, err)
		}
		switch asset.LicenseStatus {
		case "allowed":
			if asset.ArtifactID != "" || asset.SourceSnapshotReceipt != "" || asset.SourcePageURL != "" || asset.SourceImageURL != "" {
				return fmt.Errorf("asset %q mixes standalone and source-backed provenance", asset.AssetID)
			}
		case "private_source":
			if strings.TrimSpace(asset.ArtifactID) == "" || len(asset.SourceSnapshotReceipt) != 64 || publicCitationURL(asset.SourcePageURL) != asset.SourcePageURL || publicCitationURL(asset.SourceImageURL) != asset.SourceImageURL {
				return fmt.Errorf("asset %q has invalid private-source provenance", asset.AssetID)
			}
		default:
			return fmt.Errorf("asset %q is not allowed for embedding", asset.AssetID)
		}
		data, err := base64.StdEncoding.DecodeString(asset.DataBase64)
		if err != nil {
			return fmt.Errorf("asset %q data is not base64: %w", asset.AssetID, err)
		}
		if asset.DataBase64 == "" || SHA256(data) != asset.SHA256 {
			return fmt.Errorf("asset %q hash does not match embedded bytes", asset.AssetID)
		}
		if !asset.Decorative && strings.TrimSpace(asset.Alt) == "" {
			return fmt.Errorf("asset %q requires alt text", asset.AssetID)
		}
		if strings.EqualFold(asset.MediaType, "image/svg+xml") && !safePhase0SVG(data) {
			return fmt.Errorf("asset %q SVG contains active or external content", asset.AssetID)
		}
		assets[asset.AssetID] = asset
	}
	refs := map[string]Reference{}
	for _, ref := range document.References {
		if _, exists := refs[ref.RefID]; exists {
			return fmt.Errorf("duplicate reference ID %q", ref.RefID)
		}
		switch ref.Kind {
		case "cross_reference":
			if _, ok := nodes[ref.Target]; !ok {
				return fmt.Errorf("cross-reference %q targets missing node %q", ref.RefID, ref.Target)
			}
		case "citation":
			parsed, err := url.Parse(ref.Target)
			if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
				return fmt.Errorf("reference %q has unsupported target %q", ref.RefID, ref.Target)
			}
		case "footnote":
			if strings.TrimSpace(ref.Target) == "" {
				return fmt.Errorf("footnote %q has an empty target", ref.RefID)
			}
		default:
			return fmt.Errorf("reference %q has unsupported kind %q", ref.RefID, ref.Kind)
		}
		refs[ref.RefID] = ref
	}
	usedAssets := map[string]bool{}
	usedRefs := map[string]bool{}
	for index, block := range document.Blocks {
		if block.ParentNodeID != "" {
			parent, ok := nodes[block.ParentNodeID]
			if !ok || parent.Kind != "section" || block.Kind == "section" && (parent.Level != 2 || block.Level != 3) {
				return fmt.Errorf("node %q has invalid parent %q", block.NodeID, block.ParentNodeID)
			}
		} else if block.Kind == "section" && block.Level == 3 {
			return fmt.Errorf("node %q has invalid root section level", block.NodeID)
		}
		if err := validateBlockShape(index, block, assets); err != nil {
			return err
		}
		if block.Figure != nil {
			usedAssets[block.Figure.AssetID] = true
		}
		for _, target := range append(append(append(append([]string{}, block.Supports...), block.Qualifies...), block.ContrastsWith...), block.Elaborates...) {
			if _, ok := nodes[target]; !ok || target == block.NodeID {
				return fmt.Errorf("node %q has invalid rhetorical target %q", block.NodeID, target)
			}
		}
		for _, target := range block.RefersTo {
			if _, nodeOK := nodes[target]; !nodeOK {
				if _, refOK := refs[target]; !refOK {
					return fmt.Errorf("node %q refers to missing target %q", block.NodeID, target)
				}
				usedRefs[target] = true
			}
		}
		for _, refID := range block.EvidenceRefs {
			ref, ok := refs[refID]
			if !ok || ref.Kind == "cross_reference" {
				return fmt.Errorf("node %q has invalid evidence reference %q", block.NodeID, refID)
			}
			usedRefs[refID] = true
		}
	}
	for assetID := range assets {
		if !usedAssets[assetID] {
			return fmt.Errorf("asset %q is not used by any figure", assetID)
		}
	}
	for refID := range refs {
		if !usedRefs[refID] {
			return fmt.Errorf("reference %q is not used by any block", refID)
		}
	}
	coveredRequirements := map[string]bool{}
	for _, coverage := range document.Coverage {
		if coveredRequirements[coverage.RequirementID] {
			return fmt.Errorf("duplicate coverage requirement %q", coverage.RequirementID)
		}
		coveredRequirements[coverage.RequirementID] = true
		for _, nodeID := range coverage.NodeIDs {
			if _, ok := nodes[nodeID]; !ok {
				return fmt.Errorf("coverage %q refers to missing node %q", coverage.RequirementID, nodeID)
			}
		}
	}
	for _, block := range document.Blocks {
		for _, requirementID := range block.RequirementRefs {
			if !coveredRequirements[requirementID] {
				return fmt.Errorf("node %q has invalid requirement reference %q", block.NodeID, requirementID)
			}
		}
	}
	if len(document.Extensions) > 0 {
		return fmt.Errorf("document extensions require a registered Phase 0 extension profile")
	}
	return nil
}

func validateBlockShape(index int, block Block, assets map[string]Asset) error {
	missing := func(field string) error { return fmt.Errorf("block %d %q requires %s", index, block.NodeID, field) }
	present := []string{}
	if block.Title != "" || block.Level != 0 {
		present = append(present, "section")
	}
	if block.Prose != "" {
		present = append(present, "prose")
	}
	if len(block.Items) > 0 {
		present = append(present, "items")
	}
	if block.Code != "" || block.Language != "" {
		present = append(present, "code")
	}
	if block.Table != nil {
		present = append(present, "table")
	}
	if block.Equation != nil {
		present = append(present, "equation")
	}
	if block.Figure != nil {
		present = append(present, "figure")
	}
	if len(block.Extension) > 0 {
		present = append(present, "extension")
	}
	allowedPayload := map[string]string{
		"section": "section", "prose": "prose", "quote": "prose", "callout": "prose",
		"list": "items", "code": "code", "table": "table", "equation": "equation", "figure": "figure", "raw": "extension",
	}[block.Kind]
	for _, payload := range present {
		if payload != allowedPayload {
			return fmt.Errorf("block %q kind %q contains incompatible %s payload", block.NodeID, block.Kind, payload)
		}
	}
	switch block.Kind {
	case "section":
		if strings.TrimSpace(block.Title) == "" || block.Level == 0 {
			return missing("title and level")
		}
	case "prose", "quote", "callout":
		if strings.TrimSpace(block.Prose) == "" {
			return missing("authored prose")
		}
	case "list":
		if len(block.Items) == 0 {
			return missing("items")
		}
	case "code":
		if block.Code == "" {
			return missing("code")
		}
	case "table":
		if block.Table == nil || len(block.Table.Columns) == 0 {
			return missing("table")
		}
		for rowIndex, row := range block.Table.Rows {
			if len(row) != len(block.Table.Columns) {
				return fmt.Errorf("table %q row %d has %d cells, want %d", block.NodeID, rowIndex, len(row), len(block.Table.Columns))
			}
		}
	case "equation":
		if block.Equation == nil || strings.TrimSpace(block.Equation.Expression) == "" || block.Equation.Notation != "latex" {
			return missing("LaTeX equation")
		}
	case "figure":
		if block.Figure == nil {
			return missing("figure")
		}
		asset, ok := assets[block.Figure.AssetID]
		if !ok {
			return fmt.Errorf("figure %q refers to missing asset %q", block.NodeID, block.Figure.AssetID)
		}
		if block.Figure.Decorative != asset.Decorative || block.Figure.Alt != asset.Alt || (!block.Figure.Decorative && strings.TrimSpace(block.Figure.Alt) == "") {
			return fmt.Errorf("figure %q accessibility metadata does not match asset", block.NodeID)
		}
	case "raw":
		if len(block.Extension) == 0 {
			return missing("extension payload")
		}
	default:
		return fmt.Errorf("unsupported block kind %q", block.Kind)
	}
	return nil
}

func validateEvidenceLineage(
	narrative *Narrative,
	attestation *FlowAttestation,
	document Document,
) error {
	if narrative == nil || attestation == nil {
		return nil
	}
	packets := map[string]EvidencePacket{}
	preserved := map[string]EvidencePacket{}
	for _, packet := range narrative.EvidencePackets {
		packets[packet.EvidenceID] = packet
		if packet.Preserve {
			preserved[packet.EvidenceID] = packet
		}
	}
	if len(attestation.EvidenceSupportCoverage) != len(packets) {
		return fmt.Errorf("flow evidence support coverage does not match narrative evidence")
	}
	seenSupport := map[string]bool{}
	for _, coverage := range attestation.EvidenceSupportCoverage {
		packet, ok := packets[coverage.EvidenceID]
		block := documentBlockByID(document, coverage.CoveredNodeID)
		claimPresent := blockContainsExactQuote(block, packet.Claim)
		supportValid := coverage.SupportLevel == "direct_source_statement" ||
			coverage.SupportLevel == "bounded_inference"
		removedValid := coverage.SupportLevel == "unsupported_removed" && !packet.Preserve
		quoteValid := supportValid && blockQuoteRangeValid(
			block, coverage.CoverageValueIndex, coverage.CoverageByteOffset,
			coverage.CoverageByteSize, coverage.CoverageQuoteSHA,
		)
		removedQuoteValid := removedValid && coverage.CoverageQuoteSHA == SHA256(nil) &&
			coverage.CoverageValueIndex == 0 && coverage.CoverageByteOffset == 0 &&
			coverage.CoverageByteSize == 0
		if !ok || seenSupport[coverage.EvidenceID] ||
			coverage.EvidenceReceipt != evidenceReceipt(packet) ||
			coverage.SourceExcerptSHA != packet.SourceExcerptSHA ||
			coverage.ClaimSHA != packet.ClaimSHA ||
			coverage.CoveredNodeID != packet.AuthorNodeID ||
			!documentNodeHasEvidenceOrdinals(document, coverage.CoveredNodeID, []int{packet.AcceptedOrdinal}) ||
			supportValid && !quoteValid || removedValid && (claimPresent || !removedQuoteValid) ||
			!supportValid && !removedValid {
			return fmt.Errorf("flow evidence support coverage is stale or invalid")
		}
		seenSupport[coverage.EvidenceID] = true
	}
	if len(packets) > 0 {
		for _, block := range document.Blocks {
			if block.Kind != "section" && len(documentNodeEvidenceOrdinals(document, block.NodeID)) > 0 &&
				!blockSupportCoverageComplete(block, attestation.EvidenceSupportCoverage) {
				return fmt.Errorf("flow evidence support coverage is stale or invalid")
			}
		}
	}
	if len(attestation.PreservationCoverage) != len(preserved) {
		return fmt.Errorf("flow preservation coverage does not match narrative evidence")
	}
	seenPreserved := map[string]bool{}
	for _, coverage := range attestation.PreservationCoverage {
		packet, ok := preserved[coverage.EvidenceID]
		if !ok || seenPreserved[coverage.EvidenceID] ||
			coverage.EvidenceReceipt != evidenceReceipt(packet) ||
			coverage.SourceExcerptSHA != packet.SourceExcerptSHA ||
			coverage.CoverageQuoteSHA != packet.ClaimSHA ||
			coverage.CoveredNodeID != packet.AuthorNodeID ||
			!blockContainsExactQuote(documentBlockByID(document, coverage.CoveredNodeID), packet.Claim) ||
			!documentNodeHasEvidenceOrdinals(document, coverage.CoveredNodeID, []int{packet.AcceptedOrdinal}) {
			return fmt.Errorf("flow preservation coverage is stale or invalid")
		}
		seenPreserved[coverage.EvidenceID] = true
	}
	return nil
}

func validateAttestation(attestation *FlowAttestation, document Document) error {
	if attestation == nil {
		return nil
	}
	if attestation.DocumentID != document.DocumentID || attestation.RevisionID != document.RevisionID {
		return fmt.Errorf("flow attestation document revision does not match semantic IL")
	}
	projection, _, err := RenderMarkdown(document)
	if err != nil {
		return err
	}
	if attestation.LinearProjectionSHA256 != SHA256(projection) {
		return fmt.Errorf("flow attestation is stale for the linear manuscript projection")
	}
	if attestation.Verdict == "accept" && (!attestation.CentralThreadPreserved || !attestation.SectionHandoffsResolved || !attestation.OpenLoopsResolved || !attestation.TerminologyContinuous || len(attestation.AccidentalRepetitionFindings) > 0 || len(attestation.AbruptTransitionFindings) > 0 || len(attestation.UnsupportedAdditionFindings) > 0) {
		return fmt.Errorf("accepted flow attestation contains unresolved required findings")
	}
	nodes := map[string]bool{}
	for _, block := range document.Blocks {
		nodes[block.NodeID] = true
	}
	findings := append(append(append([]FlowFinding{}, attestation.AccidentalRepetitionFindings...), attestation.AbruptTransitionFindings...), attestation.UnsupportedAdditionFindings...)
	for _, finding := range findings {
		if !nodes[finding.NodeID] {
			return fmt.Errorf("flow finding refers to missing node %q", finding.NodeID)
		}
	}
	seenSupport := map[string]bool{}
	for _, coverage := range attestation.EvidenceSupportCoverage {
		if coverage.EvidenceID == "" || seenSupport[coverage.EvidenceID] || !nodes[coverage.CoveredNodeID] ||
			len(coverage.EvidenceReceipt) != 64 || len(coverage.SourceExcerptSHA) != 64 ||
			len(coverage.ClaimSHA) != 64 || len(coverage.CoverageQuoteSHA) != 64 ||
			coverage.CoverageValueIndex < 0 || coverage.CoverageByteOffset < 0 ||
			coverage.CoverageByteSize < 0 || coverage.SupportLevel == "" {
			return fmt.Errorf("flow evidence support coverage receipt is invalid")
		}
		seenSupport[coverage.EvidenceID] = true
	}
	seenPreserved := map[string]bool{}
	for _, coverage := range attestation.PreservationCoverage {
		if coverage.EvidenceID == "" || seenPreserved[coverage.EvidenceID] || !nodes[coverage.CoveredNodeID] ||
			len(coverage.EvidenceReceipt) != 64 || len(coverage.SourceExcerptSHA) != 64 || len(coverage.CoverageQuoteSHA) != 64 {
			return fmt.Errorf("flow preservation coverage receipt is invalid")
		}
		seenPreserved[coverage.EvidenceID] = true
	}
	return nil
}

// SHA256 returns a lowercase hexadecimal SHA-256 receipt.
func SHA256(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func safePhase0SVG(data []byte) bool {
	allowedElements := map[string]bool{
		"svg": true, "g": true, "path": true, "rect": true, "circle": true,
		"ellipse": true, "line": true, "polyline": true, "polygon": true, "text": true,
	}
	allowedAttributes := map[string]bool{
		"width": true, "height": true, "viewbox": true, "x": true, "y": true,
		"x1": true, "y1": true, "x2": true, "y2": true, "cx": true, "cy": true, "r": true,
		"rx": true, "ry": true, "d": true, "points": true, "fill": true, "stroke": true,
		"stroke-width": true, "opacity": true, "transform": true, "text-anchor": true,
		"font-family": true, "font-size": true, "font-weight": true,
	}
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	rootSeen := false
	rootClosed := false
	depth := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false
		}
		switch typed := token.(type) {
		case xml.StartElement:
			if rootClosed {
				return false
			}
			name := strings.ToLower(typed.Name.Local)
			if !allowedElements[name] || (!rootSeen && name != "svg") || (rootSeen && name == "svg") {
				return false
			}
			if typed.Name.Space != "http://www.w3.org/2000/svg" {
				return false
			}
			rootSeen = true
			depth++
			for _, attribute := range typed.Attr {
				attributeName := strings.ToLower(attribute.Name.Local)
				if attribute.Name.Space == "" && attributeName == "xmlns" {
					if depth != 1 || attribute.Value != "http://www.w3.org/2000/svg" {
						return false
					}
					continue
				}
				normalizedValue := strings.ToLower(strings.Join(strings.Fields(attribute.Value), ""))
				if attribute.Name.Space != "" || !allowedAttributes[attributeName] || strings.Contains(normalizedValue, "url(") || strings.Contains(attribute.Value, `\`) {
					return false
				}
			}
		case xml.EndElement:
			depth--
			if depth < 0 {
				return false
			}
			if depth == 0 {
				rootClosed = true
			}
		case xml.CharData:
			if depth == 0 && strings.TrimSpace(string(typed)) != "" {
				return false
			}
		case xml.Directive, xml.ProcInst:
			return false
		}
	}
	return rootSeen && rootClosed && depth == 0
}
