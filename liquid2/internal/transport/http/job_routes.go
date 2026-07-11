package httptransport

import (
	"context"
	"net/http"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/danielgtaylor/huma/v2"
)

type jobIDInput struct {
	// ID is the target job ID.
	ID string `path:"id" doc:"Job ID"`
}

type listJobsInput struct {
	// Status filters jobs by lifecycle state.
	Status string `query:"status" enum:"queued,running,completed,failed" required:"false"`
	// Kind filters jobs by worker operation.
	Kind string `query:"kind" enum:"scrape_url,translate_document,poll_feed,extract_upload_text" required:"false"`
	// Limit caps the number of returned jobs.
	Limit int `query:"limit" minimum:"1" maximum:"100" required:"false"`
}

type jobListOutput struct {
	// Body is the job list response body.
	Body app.JobList
}

type jobOutput struct {
	// Body is the job response body.
	Body app.Job
}

func registerJobRoutes(api huma.API, service *app.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "list-jobs", Method: http.MethodGet, Path: "/api/v1/jobs",
		Summary: "List jobs", Tags: []string{"Jobs"}, Errors: []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *listJobsInput) (*jobListOutput, error) {
		jobs, err := service.ListJobs(ctx, app.JobFilters{
			Status: input.Status, Kind: input.Kind, Limit: input.Limit,
		})
		return &jobListOutput{Body: jobs}, mapError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-job", Method: http.MethodGet, Path: "/api/v1/jobs/{id}",
		Summary: "Get job", Tags: []string{"Jobs"}, Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, input *jobIDInput) (*jobOutput, error) {
		job, err := service.GetJob(ctx, input.ID)
		return &jobOutput{Body: job}, mapError(err)
	})
}
