package app

import "sort"

func cloneJob(job Job) Job {
	job.Error = cloneString(job.Error)
	job.StartedAt = cloneInt64(job.StartedAt)
	job.FinishedAt = cloneInt64(job.FinishedAt)
	return job
}

func matchesJobFilters(job Job, filters JobFilters) bool {
	if filters.Status != "" && job.Status != filters.Status {
		return false
	}
	if filters.Kind != "" && job.Kind != filters.Kind {
		return false
	}
	return true
}

func limitJobs(jobs []Job, limit int) []Job {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if len(jobs) <= limit {
		return jobs
	}
	return jobs[:limit]
}

func sortJobs(jobs []Job) {
	sort.Slice(jobs, func(i int, j int) bool {
		if jobs[i].CreatedAt != jobs[j].CreatedAt {
			return jobs[i].CreatedAt > jobs[j].CreatedAt
		}
		return jobs[i].ID > jobs[j].ID
	})
}
