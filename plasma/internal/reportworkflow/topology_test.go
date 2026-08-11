package reportworkflow

import (
	"reflect"
	"testing"
)

func TestStaticTopologyNames(t *testing.T) {
	planned := StaticTopology(FamilyPlanned, FinalTailLegacy)
	if !reflect.DeepEqual(planned.Nodes, []string{NodePlan, NodeDirectDraft, NodeFinalStore}) {
		t.Fatalf("unexpected planned topology nodes: %#v", planned.Nodes)
	}
	if !reflect.DeepEqual(planned.Edges, [][2]string{{NodePlan, NodeDirectDraft}, {NodeDirectDraft, NodeFinalStore}}) {
		t.Fatalf("unexpected planned topology edges: %#v", planned.Edges)
	}
	if got, err := SelectFamily("long_form", "section_fanout"); err != nil || got != FamilyLongFormSectionFanout {
		t.Fatalf("expected section fanout family, got %q err=%v", got, err)
	}
	v3 := StaticTopology(FamilyLongFormSectionFanout, FinalTailV3)
	for _, node := range []string{NodePartPlan, NodeSectionDraft, NodeSemanticCheck, NodeEvidenceCheck} {
		if !containsTopologyNode(v3.Nodes, node) {
			t.Fatalf("v3 fanout topology missing %q in %#v", node, v3.Nodes)
		}
	}
}

func TestLongFormTopologyDeclaresOrderedEdgesAndOptionalNodes(t *testing.T) {
	serial := StaticTopology(FamilyLongFormSerial, FinalTailV2)
	wantSerialEdges := [][2]string{
		{NodePlan, NodeRequirements}, {NodeRequirements, NodeSectionDraft},
		{NodeSectionDraft, NodePartAssembly}, {NodePartAssembly, NodePartEdit},
		{NodePartEdit, NodeReportAssembly}, {NodePartAssembly, NodeReportAssembly},
		{NodeReportAssembly, NodeFinalWrite}, {NodeFinalWrite, NodeReaderEdit},
		{NodeReaderEdit, NodeStyleEdit}, {NodeStyleEdit, NodeEvidenceCheck},
		{NodeReaderEdit, NodeEvidenceCheck}, {NodeEvidenceCheck, NodeFinalStore},
	}
	if !reflect.DeepEqual(serial.Edges, wantSerialEdges) {
		t.Fatalf("unexpected serial edges:\n got %#v\nwant %#v", serial.Edges, wantSerialEdges)
	}
	wantSerialOptional := []OptionalNode{
		{NodeID: NodePartEdit, DecidedBy: OptionalPartEdit},
		{NodeID: NodeStyleEdit, DecidedBy: OptionalStyleEdit},
	}
	if !reflect.DeepEqual(serial.Optional, wantSerialOptional) {
		t.Fatalf("unexpected serial optional nodes: %#v", serial.Optional)
	}

	fanout := StaticTopology(FamilyLongFormSectionFanout, FinalTailV3)
	wantFanoutEdges := [][2]string{
		{NodePlan, NodeRequirements}, {NodeRequirements, NodePartPlan},
		{NodePartPlan, NodeSectionDraft}, {NodeRequirements, NodeSectionDraft},
		{NodeSectionDraft, NodePartAssembly}, {NodePartAssembly, NodePartEdit},
		{NodePartEdit, NodeReportAssembly}, {NodePartAssembly, NodeReportAssembly},
		{NodeReportAssembly, NodeFinalWrite}, {NodeFinalWrite, NodeReaderEdit},
		{NodeReaderEdit, NodeStyleEdit}, {NodeStyleEdit, NodeSemanticCheck},
		{NodeSemanticCheck, NodeEvidenceCheck}, {NodeReaderEdit, NodeEvidenceCheck},
		{NodeEvidenceCheck, NodeFinalStore},
	}
	if !reflect.DeepEqual(fanout.Edges, wantFanoutEdges) {
		t.Fatalf("unexpected fanout edges:\n got %#v\nwant %#v", fanout.Edges, wantFanoutEdges)
	}
	wantFanoutOptional := []OptionalNode{
		{NodeID: NodePartPlan, DecidedBy: OptionalPartPlanning},
		{NodeID: NodePartEdit, DecidedBy: OptionalPartEdit},
		{NodeID: NodeStyleEdit, DecidedBy: OptionalStyleEdit},
		{NodeID: NodeSemanticCheck, DecidedBy: OptionalStyleEdit},
	}
	if !reflect.DeepEqual(fanout.Optional, wantFanoutOptional) {
		t.Fatalf("unexpected fanout optional nodes: %#v", fanout.Optional)
	}

	legacy := StaticTopology(FamilyLongFormSerial, FinalTailLegacy)
	wantLegacyEdges := [][2]string{
		{NodePlan, NodeRequirements}, {NodeRequirements, NodeSectionDraft},
		{NodeSectionDraft, NodePartAssembly}, {NodePartAssembly, NodePartEdit},
		{NodePartEdit, NodeLegacyFinal}, {NodePartAssembly, NodeLegacyFinal},
		{NodeLegacyFinal, NodeHumanize},
	}
	if !reflect.DeepEqual(legacy.Edges, wantLegacyEdges) {
		t.Fatalf("unexpected legacy edges:\n got %#v\nwant %#v", legacy.Edges, wantLegacyEdges)
	}
	if containsTopologyNode(legacy.Nodes, NodeFinalStore) {
		t.Fatalf("legacy topology must not declare finalstore: %#v", legacy.Nodes)
	}
}

func TestLongFormTopologyRejectsUnknownTailShape(t *testing.T) {
	for _, family := range []Family{FamilyLongFormSerial, FamilyLongFormSectionFanout} {
		topology := StaticTopology(family, FinalTail("future"))
		if topology.Family != family || topology.Tail != FinalTail("future") {
			t.Fatalf("topology identity changed: %#v", topology)
		}
		if len(topology.Nodes) != 0 || len(topology.Edges) != 0 || len(topology.Optional) != 0 {
			t.Fatalf("unknown final tail must not look executable: %#v", topology)
		}
	}
}

func containsTopologyNode(nodes []string, want string) bool {
	for _, node := range nodes {
		if node == want {
			return true
		}
	}
	return false
}
