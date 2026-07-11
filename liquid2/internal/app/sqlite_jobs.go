package app

import sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"

func (tx *sqliteTx) Job(id string) (Job, bool) {
	row, err := tx.q.GetJob(tx.ctx, id)
	if tx.missing(err) {
		return Job{}, false
	}
	return sqliteJob(row), true
}

func (tx *sqliteTx) Jobs(filters JobFilters) []Job {
	rows, err := tx.q.ListJobs(tx.ctx, sqlitedb.ListJobsParams{
		Status: sqliteNullString(filterString(filters.Status)),
		Kind:   sqliteNullString(filterString(filters.Kind)),
		Limit:  int64(jobLimit(filters.Limit)),
	})
	tx.abort(err)
	jobs := make([]Job, 0, len(rows))
	for _, row := range rows {
		jobs = append(jobs, sqliteJob(row))
	}
	return jobs
}

func (tx *sqliteTx) PutJob(job Job) {
	_, err := tx.q.UpsertJob(tx.ctx, sqlitedb.UpsertJobParams{
		ID: job.ID, Kind: job.Kind, Status: job.Status, PayloadJson: job.PayloadJSON,
		Error: sqliteNullString(job.Error), Attempts: job.Attempts, CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt, StartedAt: sqliteNullInt64(job.StartedAt),
		FinishedAt: sqliteNullInt64(job.FinishedAt),
	})
	tx.abort(err)
}
