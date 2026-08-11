package reportworkflow

// FinalTail은 long-form finalization tail의 제품 고정 이름이다.
type FinalTail string

const (
	FinalTailLegacy FinalTail = "legacy"
	FinalTailV1     FinalTail = "v1"
	FinalTailV2     FinalTail = "v2"
	FinalTailV3     FinalTail = "v3"
)

const (
	NodePlan           = "plan"
	NodeRequirements   = "requirements"
	NodePartPlan       = "partplan"
	NodeSectionDraft   = "sectiondraft"
	NodePartAssembly   = "partassembly"
	NodePartEdit       = "partedit"
	NodeLegacyFinal    = "legacyfinalize"
	NodeReportAssembly = "reportassembly"
	NodeFinalWrite     = "finalwrite"
	NodeReaderEdit     = "readeredit"
	NodeStyleEdit      = "styleedit"
	NodeSemanticCheck  = "semanticcheck"
	NodeEvidenceCheck  = "evidencecheck"
	NodeHumanize       = "humanize"
	NodeDirectDraft    = "directdraft"
	NodeFinalStore     = "finalstore"
)

// OptionalDecision은 canonical plan payload가 최종 활성 여부를 결정하는 선택 조건이다.
type OptionalDecision string

const (
	OptionalPartPlanning OptionalDecision = "canonical_plan.part_planning_enabled"
	OptionalPartEdit     OptionalDecision = "canonical_plan.part_edit_enabled"
	OptionalStyleEdit    OptionalDecision = "canonical_plan.final_edit_pipeline_and_post_report_humanize"
	OptionalHumanize     OptionalDecision = "canonical_plan.post_report_humanize"
)

// OptionalNode는 정적 graph 후보에는 있지만 canonical plan에 의해 실행 여부가 정해지는 node다.
type OptionalNode struct {
	NodeID    string
	DecidedBy OptionalDecision
}

// Topology는 제품 graph의 정적 node/edge 설명이다. 실행 payload나 stage callback은 담지 않는다.
//
// Optional에 포함된 node는 가능한 경로를 설명할 뿐 활성 상태를 확정하지 않는다. long-form
// 실행 graph는 canonical plan의 flag와 final tail 선택을 읽어 이 후보 graph에서 typed
// 경로를 고른다.
type Topology struct {
	Family   Family
	Tail     FinalTail
	Nodes    []string
	Edges    [][2]string
	Optional []OptionalNode
}

// StaticTopology는 이번 제품 wave가 알고 있는 report graph family를 내용 없이 설명한다.
func StaticTopology(family Family, tail FinalTail) Topology {
	switch family {
	case FamilyOneTake:
		return Topology{Family: family, Tail: tail, Nodes: []string{NodeDirectDraft, NodeFinalStore}, Edges: [][2]string{{NodeDirectDraft, NodeFinalStore}}}
	case FamilyPlanned:
		return Topology{Family: family, Tail: tail, Nodes: []string{NodePlan, NodeDirectDraft, NodeFinalStore}, Edges: [][2]string{{NodePlan, NodeDirectDraft}, {NodeDirectDraft, NodeFinalStore}}}
	case FamilyLongFormSerial:
		return longFormSerialTopology(tail)
	case FamilyLongFormSectionFanout:
		return longFormFanoutTopology(tail)
	default:
		return Topology{Family: family, Tail: tail}
	}
}

func longFormSerialTopology(tail FinalTail) Topology {
	if !knownFinalTail(tail) {
		return Topology{Family: FamilyLongFormSerial, Tail: tail}
	}
	nodes := []string{NodePlan, NodeRequirements, NodeSectionDraft, NodePartAssembly, NodePartEdit}
	nodes = append(nodes, finalTailNodes(tail)...)
	if tail != FinalTailLegacy {
		nodes = append(nodes, NodeFinalStore)
	}
	return Topology{
		Family: FamilyLongFormSerial, Tail: tail, Nodes: nodes,
		Edges:    longFormEdges(false, tail),
		Optional: optionalLongFormNodes(false, tail),
	}
}

func longFormFanoutTopology(tail FinalTail) Topology {
	if !knownFinalTail(tail) {
		return Topology{Family: FamilyLongFormSectionFanout, Tail: tail}
	}
	nodes := []string{NodePlan, NodeRequirements, NodePartPlan, NodeSectionDraft, NodePartAssembly, NodePartEdit}
	nodes = append(nodes, finalTailNodes(tail)...)
	if tail != FinalTailLegacy {
		nodes = append(nodes, NodeFinalStore)
	}
	return Topology{
		Family: FamilyLongFormSectionFanout, Tail: tail, Nodes: nodes,
		Edges:    longFormEdges(true, tail),
		Optional: optionalLongFormNodes(true, tail),
	}
}

func finalTailNodes(tail FinalTail) []string {
	switch tail {
	case FinalTailLegacy:
		return []string{NodeLegacyFinal, NodeHumanize}
	case FinalTailV1:
		return []string{NodeReportAssembly, NodeReaderEdit, NodeStyleEdit, NodeEvidenceCheck}
	case FinalTailV2:
		return []string{NodeReportAssembly, NodeFinalWrite, NodeReaderEdit, NodeStyleEdit, NodeEvidenceCheck}
	case FinalTailV3:
		return []string{NodeReportAssembly, NodeFinalWrite, NodeReaderEdit, NodeStyleEdit, NodeSemanticCheck, NodeEvidenceCheck}
	default:
		return nil
	}
}

func optionalLongFormNodes(fanout bool, tail FinalTail) []OptionalNode {
	nodes := []OptionalNode{{NodeID: NodePartEdit, DecidedBy: OptionalPartEdit}}
	if fanout {
		nodes = append([]OptionalNode{{NodeID: NodePartPlan, DecidedBy: OptionalPartPlanning}}, nodes...)
	}
	switch tail {
	case FinalTailV1, FinalTailV2:
		nodes = append(nodes, OptionalNode{NodeID: NodeStyleEdit, DecidedBy: OptionalStyleEdit})
	case FinalTailV3:
		nodes = append(nodes,
			OptionalNode{NodeID: NodeStyleEdit, DecidedBy: OptionalStyleEdit},
			OptionalNode{NodeID: NodeSemanticCheck, DecidedBy: OptionalStyleEdit},
		)
	case FinalTailLegacy:
		nodes = append(nodes, OptionalNode{NodeID: NodeHumanize, DecidedBy: OptionalHumanize})
	}
	return nodes
}

func longFormEdges(fanout bool, tail FinalTail) [][2]string {
	edges := [][2]string{{NodePlan, NodeRequirements}}
	if fanout {
		edges = append(edges, [2]string{NodeRequirements, NodePartPlan}, [2]string{NodePartPlan, NodeSectionDraft}, [2]string{NodeRequirements, NodeSectionDraft})
	} else {
		edges = append(edges, [2]string{NodeRequirements, NodeSectionDraft})
	}
	edges = append(edges, [2]string{NodeSectionDraft, NodePartAssembly})
	tailNodes := finalTailNodes(tail)
	if len(tailNodes) == 0 {
		return nil
	}
	firstTail := tailNodes[0]
	edges = append(edges, [2]string{NodePartAssembly, NodePartEdit}, [2]string{NodePartEdit, firstTail}, [2]string{NodePartAssembly, firstTail})
	edges = append(edges, finalTailEdges(tail)...)
	if tail == FinalTailLegacy {
		return edges
	}
	return append(edges, [2]string{tailNodes[len(tailNodes)-1], NodeFinalStore})
}

func finalTailEdges(tail FinalTail) [][2]string {
	switch tail {
	case FinalTailV1:
		return [][2]string{
			{NodeReportAssembly, NodeReaderEdit}, {NodeReaderEdit, NodeStyleEdit},
			{NodeStyleEdit, NodeEvidenceCheck}, {NodeReaderEdit, NodeEvidenceCheck},
		}
	case FinalTailV2:
		return [][2]string{
			{NodeReportAssembly, NodeFinalWrite}, {NodeFinalWrite, NodeReaderEdit},
			{NodeReaderEdit, NodeStyleEdit}, {NodeStyleEdit, NodeEvidenceCheck},
			{NodeReaderEdit, NodeEvidenceCheck},
		}
	case FinalTailV3:
		return [][2]string{
			{NodeReportAssembly, NodeFinalWrite}, {NodeFinalWrite, NodeReaderEdit},
			{NodeReaderEdit, NodeStyleEdit}, {NodeStyleEdit, NodeSemanticCheck},
			{NodeSemanticCheck, NodeEvidenceCheck}, {NodeReaderEdit, NodeEvidenceCheck},
		}
	case FinalTailLegacy:
		return [][2]string{{NodeLegacyFinal, NodeHumanize}}
	default:
		return nil
	}
}

func knownFinalTail(tail FinalTail) bool {
	switch tail {
	case FinalTailLegacy, FinalTailV1, FinalTailV2, FinalTailV3:
		return true
	default:
		return false
	}
}
