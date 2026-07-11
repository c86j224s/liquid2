package httptransport

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/ingest"
	"github.com/c86j224s/liquid2/internal/translation"
	"github.com/danielgtaylor/huma/v2"
)

type bookmarkDocumentInput struct {
	Body struct {
		URL      string   `json:"url" format:"uri" maxLength:"2048"`
		Title    string   `json:"title,omitempty" maxLength:"300"`
		FolderID string   `json:"folderId,omitempty"`
		TagIDs   []string `json:"tagIds,omitempty"`
	}
}

type scrapeDocumentInput struct {
	Body struct {
		URL      string   `json:"url" format:"uri" maxLength:"2048"`
		FolderID string   `json:"folderId,omitempty"`
		TagIDs   []string `json:"tagIds,omitempty"`
	}
}

type scrapeTranslateDocumentInput struct {
	Body struct {
		URL            string   `json:"url" format:"uri" maxLength:"2048"`
		TargetLanguage string   `json:"targetLanguage" minLength:"1"`
		FolderID       string   `json:"folderId,omitempty"`
		TagIDs         []string `json:"tagIds,omitempty"`
	}
}

type scrapeTranslateDocumentOutput struct {
	Body struct {
		Document app.DocumentDetail `json:"document"`
		Job      app.Job            `json:"job"`
	}
}

type uploadDocumentInput struct {
	RawBody huma.MultipartFormFiles[struct {
		File     huma.FormFile `form:"file" required:"true"`
		Title    string        `form:"title" required:"false"`
		FolderID string        `form:"folderId" required:"false"`
		TagIDs   []string      `form:"tagIds" required:"false"`
	}]
}

func registerIngestionRoutes(api huma.API, service *ingest.Service, translator DocumentTranslator) {
	registerRescrapeRoute(api, service)

	huma.Register(api, huma.Operation{
		OperationID: "bookmark-document", Method: http.MethodPost,
		Path: "/api/v1/documents/bookmark", Summary: "Bookmark URL",
		Tags: []string{"Ingestion"}, DefaultStatus: http.StatusCreated,
		Errors: []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *bookmarkDocumentInput) (*documentDetailOutput, error) {
		detail, err := service.Bookmark(ctx, ingest.BookmarkInput{
			URL: input.Body.URL, Title: input.Body.Title,
			FolderID: input.Body.FolderID, TagIDs: input.Body.TagIDs,
		})
		return &documentDetailOutput{Body: detail}, mapError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "scrape-document", Method: http.MethodPost,
		Path: "/api/v1/documents/scrape", Summary: "Scrape URL",
		Tags: []string{"Ingestion"}, DefaultStatus: http.StatusCreated,
		Errors: []int{http.StatusBadRequest, http.StatusRequestEntityTooLarge, http.StatusUnsupportedMediaType},
	}, func(ctx context.Context, input *scrapeDocumentInput) (*documentDetailOutput, error) {
		detail, err := service.Scrape(ctx, ingest.ScrapeInput{
			URL: input.Body.URL, FolderID: input.Body.FolderID, TagIDs: input.Body.TagIDs,
		})
		return &documentDetailOutput{Body: detail}, mapError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "scrape-translate-document", Method: http.MethodPost,
		Path: "/api/v1/documents/scrape-translate", Summary: "Scrape URL and translate",
		Tags: []string{"Ingestion"}, DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest, http.StatusNotFound, http.StatusConflict,
			http.StatusRequestEntityTooLarge, http.StatusUnsupportedMediaType,
			http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, input *scrapeTranslateDocumentInput) (*scrapeTranslateDocumentOutput, error) {
		result, err := translation.ScrapeTranslate(ctx, service, translator, translation.ScrapeTranslateInput{
			URL: input.Body.URL, TargetLanguage: input.Body.TargetLanguage,
			FolderID: input.Body.FolderID, TagIDs: input.Body.TagIDs,
		})
		if err != nil {
			return nil, mapError(err)
		}
		output := &scrapeTranslateDocumentOutput{}
		output.Body.Document = result.Document
		output.Body.Job = result.Job
		return output, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "upload-document", Method: http.MethodPost,
		Path: "/api/v1/documents/upload", Summary: "Upload document",
		Tags: []string{"Ingestion"}, DefaultStatus: http.StatusCreated,
		Errors: []int{http.StatusBadRequest, http.StatusRequestEntityTooLarge, http.StatusUnsupportedMediaType},
	}, func(ctx context.Context, input *uploadDocumentInput) (*documentDetailOutput, error) {
		file := input.RawBody.Data().File
		data, err := readUploadBytes(file)
		if err != nil {
			return nil, mapError(err)
		}
		detail, err := service.Upload(ctx, ingest.UploadDocumentInput{
			Title: input.RawBody.Data().Title, Filename: file.Filename,
			ContentType: file.ContentType, Data: data,
			FolderID: input.RawBody.Data().FolderID,
			TagIDs:   input.RawBody.Data().TagIDs,
		})
		return &documentDetailOutput{Body: detail}, mapError(err)
	})
}

func readUploadBytes(file huma.FormFile) ([]byte, error) {
	if !file.IsSet {
		return nil, fmt.Errorf("%w: file is required", ingest.ErrBadRequest)
	}
	data, err := io.ReadAll(io.LimitReader(file, ingest.MaxUploadBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > ingest.MaxUploadBytes {
		return nil, ingest.ErrPayloadTooLarge
	}
	return data, nil
}
