package translation

import (
	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/jobs"
)

func appJob(job jobs.Job) app.Job {
	return app.Job{
		ID: job.ID, Kind: job.Kind, Status: job.Status, PayloadJSON: job.PayloadJSON,
		Error: cloneString(job.Error), Attempts: job.Attempts, CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt, StartedAt: cloneInt64(job.StartedAt),
		FinishedAt: cloneInt64(job.FinishedAt),
	}
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
