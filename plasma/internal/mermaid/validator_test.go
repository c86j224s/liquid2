package mermaid

import "testing"

func TestValidateRequirementDiagramFindsKnownParseRisks(t *testing.T) {
	result := Validate(`requirementDiagram

requirement root_access {
  id: AUTH-ROOT
  text: Access decisions must combine identity, policy, and auditability
  risk: high
  verifymethod: inspection
}`)
	if result.OK {
		t.Fatalf("expected requirementDiagram risks, got ok result: %#v", result)
	}
	if result.DiagramType != "requirementDiagram" {
		t.Fatalf("expected requirementDiagram type, got %q", result.DiagramType)
	}
	if !hasIssue(result.Errors, "requirement_id_token") || !hasIssue(result.Errors, "requirement_text_needs_quotes") {
		t.Fatalf("expected id and text issues, got %#v", result.Errors)
	}
}

func TestValidateRequirementDiagramAcceptsQuotedText(t *testing.T) {
	result := Validate(`requirementDiagram

requirement root_access {
  id: AUTH_ROOT
  text: "Access decisions must combine identity, policy, and auditability"
  risk: high
  verifymethod: inspection
}`)
	if !result.OK {
		t.Fatalf("expected valid requirementDiagram, got %#v", result.Errors)
	}
}

func TestValidateStripsMarkdownFence(t *testing.T) {
	result := Validate("```mermaid\nflowchart TD\n  A[Start] --> B[End]\n```")
	if !result.OK || result.DiagramType != "flowchart" {
		t.Fatalf("expected fenced flowchart to validate, got %#v", result)
	}
	if !hasIssue(result.Warnings, "fence_stripped") {
		t.Fatalf("expected fence warning, got %#v", result.Warnings)
	}
}

func TestValidateWarnsForCompatibilitySensitiveTypes(t *testing.T) {
	result := Validate(`sankey-beta
User,Login,100`)
	if !result.OK {
		t.Fatalf("expected static preflight to pass, got %#v", result.Errors)
	}
	if !hasIssue(result.Warnings, "compatibility_sensitive") {
		t.Fatalf("expected compatibility warning, got %#v", result.Warnings)
	}
}

func hasIssue(issues []Issue, kind string) bool {
	for _, issue := range issues {
		if issue.Kind == kind {
			return true
		}
	}
	return false
}
