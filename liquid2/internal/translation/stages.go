package translation

import (
	"context"
	"strings"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/jobs"
)

type LoadSourceStage struct {
	documents *app.Service
}

func NewLoadSourceStage(documents *app.Service) LoadSourceStage {
	return LoadSourceStage{documents: documents}
}

func (stage LoadSourceStage) Name() string { return "load_source" }

func (stage LoadSourceStage) Contract() jobs.StageContract {
	return jobs.StageContract{
		Input: "translation.TranslateDocumentPayload", Output: "translation.sourceContext",
		Idempotency: "read-only document lookup", Retry: "runner retry",
	}
}

func (stage LoadSourceStage) Run(ctx context.Context, input jobs.StageInput) (jobs.StageOutput, error) {
	payload, ok := input.Data.(TranslateDocumentPayload)
	if !ok {
		return jobs.StageOutput{}, invalidPayload("load source stage input is invalid")
	}
	prepared, err := stage.documents.PrepareTranslation(ctx, payload.DocumentID, app.PrepareTranslationInput{
		SourceContentID: payload.SourceContentID,
		TargetLanguage:  payload.TargetLanguage,
	})
	if err != nil {
		return jobs.StageOutput{}, err
	}
	payload.TargetLanguage = prepared.TargetLanguage
	return jobs.StageOutput{Data: sourceContext{
		Payload: payload, Source: prepared.SourceContent,
	}}, nil
}

type TranslateStage struct {
	provider Provider
}

func NewTranslateStage(provider Provider) TranslateStage {
	return TranslateStage{provider: provider}
}

func (stage TranslateStage) Name() string { return "translate" }

func (stage TranslateStage) Contract() jobs.StageContract {
	return jobs.StageContract{
		Input: "translation.sourceContext", Output: "translation.translatedContext",
		Idempotency: "provider-dependent remote call", Retry: "runner retry",
	}
}

func (stage TranslateStage) Run(ctx context.Context, input jobs.StageInput) (jobs.StageOutput, error) {
	source, ok := input.Data.(sourceContext)
	if !ok {
		return jobs.StageOutput{}, invalidPayload("translate stage input is invalid")
	}
	result, err := stage.translate(ctx, Request{
		DocumentID: source.Payload.DocumentID, SourceContentID: source.Payload.SourceContentID,
		SourceLanguage: contentLanguage(source.Source), TargetLanguage: source.Payload.TargetLanguage,
		Format: source.Source.Format, Text: source.Source.Content,
	})
	if err != nil {
		return jobs.StageOutput{}, err
	}
	result.Format = strings.TrimSpace(result.Format)
	if result.Format == "" {
		result.Format = source.Source.Format
	}
	return jobs.StageOutput{Data: translatedContext{Source: source, Result: result}}, nil
}

func (stage TranslateStage) translate(ctx context.Context, request Request) (result Result, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = providerFailed("translate content panicked")
		}
	}()
	result, err = stage.provider.Translate(ctx, request)
	if err != nil {
		return Result{}, providerFailed("translate content", err)
	}
	return result, nil
}

type PersistStage struct {
	documents *app.Service
}

func NewPersistStage(documents *app.Service) PersistStage {
	return PersistStage{documents: documents}
}

func (stage PersistStage) Name() string { return "persist_translation" }

func (stage PersistStage) Contract() jobs.StageContract {
	return jobs.StageContract{
		Input: "translation.translatedContext", Output: "app.DocumentDetail",
		Idempotency: "app rejects duplicate source-language translations", Retry: "runner retry",
	}
}

func (stage PersistStage) Run(ctx context.Context, input jobs.StageInput) (jobs.StageOutput, error) {
	translated, ok := input.Data.(translatedContext)
	if !ok {
		return jobs.StageOutput{}, invalidPayload("persist stage input is invalid")
	}
	detail, err := stage.documents.AppendTranslatedContent(ctx, translated.Source.Payload.DocumentID, app.AppendTranslationInput{
		SourceContentID: translated.Source.Payload.SourceContentID,
		TargetLanguage:  translated.Source.Payload.TargetLanguage,
		Content:         translated.Result.Content,
		Format:          translated.Result.Format,
	})
	if err != nil {
		return jobs.StageOutput{}, err
	}
	return jobs.StageOutput{Data: detail}, nil
}

func contentLanguage(content app.DocumentContent) string {
	if content.Language == nil {
		return ""
	}
	return *content.Language
}
