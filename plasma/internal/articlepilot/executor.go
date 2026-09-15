package articlepilot

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/articleexperiment"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportilphase0"
)

// Executor connects one validated E/A pipeline to the existing immutable outer
// run harness. It deliberately does not execute the contextual R arm.
type Executor struct {
	Bundle      articleexperiment.FixtureBundle
	Arm         articleexperiment.ArmBundle
	Provider    Provider
	PDFRenderer reportilphase0.PDFRenderer
}

func (executor Executor) Execute(ctx context.Context, input articleexperiment.ExecutionInput) (articleexperiment.ExecutionOutput, error) {
	failed := []articleexperiment.AttemptReceipt{{Attempt: 1, Kind: "semantic", Outcome: "failed"}}
	if executor.Bundle.Fixture.FixtureID != input.Fixture.FixtureID || executor.Bundle.Fixture.ValueType != input.Fixture.ValueType || executor.Bundle.Fixture.Audience != input.Fixture.Audience || executor.Bundle.Fixture.ReaderPromise != input.Fixture.ReaderPromise || executor.Bundle.Fixture.Emphasis != input.Fixture.Emphasis || executor.Bundle.Fixture.Language != input.Fixture.Language || executor.Arm.Arm.ArmID != input.Arm.ArmID || executor.Arm.Arm.Role != input.Arm.Role || executor.Arm.Arm.ContextualOnly != input.Arm.ContextualOnly || executor.Arm.Arm.Treatment.Contract.SHA256 != input.Arm.ContractSHA256 || input.ArmContract.SHA256 != input.Arm.ContractSHA256 || string(input.ArmContract.Content) != string(executor.Arm.Treatment) || !sameInputs(input.Inputs, executor.Bundle) || (input.Arm.ArmID != articleexperiment.ArmControl && input.Arm.ArmID != articleexperiment.ArmArticle) {
		return articleexperiment.ExecutionOutput{Attempts: failed}, fmt.Errorf("%w: Article executor identity differs", producterror.ErrConflict)
	}
	result, err := Run(ctx, Config{ProtocolSHA256: input.ProtocolSHA256, Bundle: executor.Bundle, Arm: executor.Arm.Arm, AuthorPrompt: string(executor.Arm.AuthorPrompt), ReaderPrompt: string(executor.Arm.ReaderPrompt), AuditorPrompt: string(executor.Arm.AuditorPrompt), RepairPrompt: string(executor.Arm.RepairPrompt), Treatment: executor.Arm.Treatment, Provider: executor.Provider})
	if err != nil {
		partial := []articleexperiment.OutputArtifact{}
		if len(result.Manuscript) > 0 {
			partial = append(partial, articleexperiment.OutputArtifact{Kind: "article_markdown", MediaType: "text/markdown; charset=utf-8", Filename: "partial-article.md", Content: result.Manuscript})
		}
		if usage, marshalErr := json.MarshalIndent(result.Usage, "", "  "); marshalErr == nil {
			partial = append(partial, articleexperiment.OutputArtifact{Kind: "usage_receipt", MediaType: "application/json", Filename: "partial-usage.json", Content: append(usage, '\n')})
		}
		return articleexperiment.ExecutionOutput{Artifacts: partial, Attempts: failed}, err
	}
	rendered, err := Render(ctx, result.Manuscript, executor.PDFRenderer)
	if err != nil {
		return articleexperiment.ExecutionOutput{Artifacts: []articleexperiment.OutputArtifact{{Kind: "article_markdown", MediaType: "text/markdown; charset=utf-8", Filename: "partial-article.md", Content: result.Manuscript}}, Attempts: failed}, err
	}
	document, err := json.MarshalIndent(result.Document, "", "  ")
	if err != nil {
		return articleexperiment.ExecutionOutput{Attempts: failed}, err
	}
	usage, err := json.MarshalIndent(result.Usage, "", "  ")
	if err != nil {
		return articleexperiment.ExecutionOutput{Attempts: failed}, err
	}
	audit, err := json.MarshalIndent(result.Audit, "", "  ")
	if err != nil {
		return articleexperiment.ExecutionOutput{Attempts: failed}, err
	}
	return articleexperiment.ExecutionOutput{Attempts: []articleexperiment.AttemptReceipt{{Attempt: 1, Kind: "semantic", Outcome: "completed"}}, Artifacts: []articleexperiment.OutputArtifact{
		{Kind: "article_il", MediaType: "application/json", Filename: "article-il.json", Content: append(document, '\n')},
		{Kind: "article_markdown", MediaType: "text/markdown; charset=utf-8", Filename: "article.md", Content: rendered.Markdown},
		{Kind: "article_html", MediaType: "text/html; charset=utf-8", Filename: "article.html", Content: rendered.HTML},
		{Kind: "article_pdf", MediaType: "application/pdf", Filename: "article.pdf", Content: rendered.PDF},
		{Kind: "usage_receipt", MediaType: "application/json", Filename: "usage.json", Content: append(usage, '\n')},
		{Kind: "audit_receipt", MediaType: "application/json", Filename: "audit.json", Content: append(audit, '\n')},
	}}, nil
}

func sameInputs(inputs []articleexperiment.InputArtifact, bundle articleexperiment.FixtureBundle) bool {
	expected := []struct {
		id      string
		content []byte
	}{{"source_body", bundle.SourceBody}, {"source_catalog", bundle.SourceCatalog}, {"dossier", bundle.Dossier}, {"claim_inventory", bundle.ClaimInventory}}
	if len(inputs) != len(expected) {
		return false
	}
	for index, value := range expected {
		if inputs[index].ID != value.id || inputs[index].SHA256 != digest(value.content) || string(inputs[index].Content) != string(value.content) {
			return false
		}
	}
	return true
}
