package jobs

import "context"

type Queue interface {
	Enqueue(context.Context, EnqueueRequest) (Job, error)
	Claim(context.Context, []string) (Job, bool, error)
	Complete(context.Context, string) (Job, error)
	Fail(context.Context, string, string) (Job, error)
	Requeue(context.Context, string, string) (Job, error)
	RecoverRunning(context.Context, string) error
	Job(context.Context, string) (Job, bool, error)
	List(context.Context, Filters) ([]Job, error)
}
