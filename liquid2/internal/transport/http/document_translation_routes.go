package httptransport

import (
	"context"
	"net/http"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/translation"
	"github.com/danielgtaylor/huma/v2"
)

type translateDocumentInput struct {
	// ID is the target document ID.
	ID string `path:"id" doc:"Document ID"`
	// Body contains translation fields.
	Body struct {
		// SourceContentID identifies the document content variant to translate.
		SourceContentID string `json:"sourceContentId" minLength:"1"`
		// TargetLanguage is the requested target language tag.
		TargetLanguage string `json:"targetLanguage" minLength:"1"`
	}
}

type translateDocumentOutput struct {
	// Body is the translation enqueue response body.
	Body struct {
		// Job contains the enqueued translation job.
		Job app.Job `json:"job"`
	}
}

func registerDocumentTranslationRoutes(api huma.API, translator DocumentTranslator) {
	huma.Register(api, huma.Operation{
		OperationID: "translate-document", Method: http.MethodPost,
		Path: "/api/v1/documents/{id}/translate", Summary: "Translate document content",
		Tags: []string{"Documents"}, DefaultStatus: http.StatusAccepted,
		Errors: []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict, http.StatusServiceUnavailable},
	}, func(ctx context.Context, input *translateDocumentInput) (*translateDocumentOutput, error) {
		if translator == nil {
			return nil, mapError(translation.ErrTranslationUnavailable)
		}
		job, err := translator.TranslateDocument(ctx, translation.EnqueueDocumentInput{
			DocumentID: input.ID, SourceContentID: input.Body.SourceContentID,
			TargetLanguage: input.Body.TargetLanguage,
		})
		if err != nil {
			return nil, mapError(err)
		}
		output := &translateDocumentOutput{}
		output.Body.Job = job
		return output, nil
	})
}
