package app

import "context"

func (s *Service) ListJobs(ctx context.Context, filters JobFilters) (JobList, error) {
	return withView(ctx, s, func(tx RepositoryReader) (JobList, error) {
		if err := validateJobFilters(filters); err != nil {
			return JobList{}, err
		}
		jobs := tx.Jobs(filters)
		return JobList{Items: jobs}, nil
	})
}

func (s *Service) GetJob(ctx context.Context, id string) (Job, error) {
	return withView(ctx, s, func(tx RepositoryReader) (Job, error) {
		job, ok := tx.Job(id)
		if !ok {
			return Job{}, notFound("job")
		}
		return job, nil
	})
}

func validateJobFilters(filters JobFilters) error {
	if filters.Status != "" && !validJobStatus(filters.Status) {
		return validation("job status is invalid")
	}
	if filters.Kind != "" && !validJobKind(filters.Kind) {
		return validation("job kind is invalid")
	}
	return nil
}

func validJobStatus(status string) bool {
	switch status {
	case JobStatusQueued, JobStatusRunning, JobStatusCompleted, JobStatusFailed:
		return true
	default:
		return false
	}
}

func validJobKind(kind string) bool {
	switch kind {
	case JobKindScrapeURL, JobKindTranslateDocument, JobKindPollFeed, JobKindExtractUploadText:
		return true
	default:
		return false
	}
}

func filterString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func jobLimit(limit int) int {
	if limit <= 0 || limit > 100 {
		return 50
	}
	return limit
}
