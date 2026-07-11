package jobs

import "context"

type fakeQueue struct {
	claimJob       Job
	hasClaim       bool
	completedID    string
	failedID       string
	failedMessage  string
	requeuedID     string
	requeueMessage string
	recoverMessage string
}

func (queue *fakeQueue) Enqueue(context.Context, EnqueueRequest) (Job, error) {
	return Job{}, nil
}

func (queue *fakeQueue) Claim(_ context.Context, kinds []string) (Job, bool, error) {
	if !queue.hasClaim {
		return Job{}, false, nil
	}
	if !containsKind(kinds, queue.claimJob.Kind) {
		return Job{}, false, nil
	}
	queue.hasClaim = false
	return queue.claimJob, true, nil
}

func (queue *fakeQueue) Complete(_ context.Context, id string) (Job, error) {
	queue.completedID = id
	return Job{ID: id, Status: StatusCompleted}, nil
}

func (queue *fakeQueue) Fail(_ context.Context, id string, message string) (Job, error) {
	queue.failedID = id
	queue.failedMessage = message
	return Job{ID: id, Status: StatusFailed}, nil
}

func (queue *fakeQueue) Requeue(_ context.Context, id string, message string) (Job, error) {
	queue.requeuedID = id
	queue.requeueMessage = message
	return Job{ID: id, Status: StatusQueued}, nil
}

func (queue *fakeQueue) RecoverRunning(_ context.Context, message string) error {
	queue.recoverMessage = message
	return nil
}

func (queue *fakeQueue) Job(context.Context, string) (Job, bool, error) {
	return Job{}, false, nil
}

func (queue *fakeQueue) List(context.Context, Filters) ([]Job, error) {
	return nil, nil
}

func containsKind(kinds []string, kind string) bool {
	for _, item := range kinds {
		if item == kind {
			return true
		}
	}
	return false
}
