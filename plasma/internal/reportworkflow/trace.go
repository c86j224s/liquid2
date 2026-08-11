package reportworkflow

import "time"

// Observer는 ledger에 기록하지 않는 workflow node 관측 sink다.
type Observer interface {
	Observe(NodeObservation)
}

// NodeObservation은 content-free 실행 관측 값이다.
type NodeObservation struct {
	NodeID     string
	StartedAt  time.Time
	DurationMS int64
	Attempt    int
	Replay     bool
	Failed     bool
}

func (runner Runner) observeStart(nodeID string) func(error, bool) {
	if runner.observer == nil {
		return func(error, bool) {}
	}
	started := time.Now()
	runner.observer.Observe(NodeObservation{NodeID: nodeID, StartedAt: started, Attempt: 1})
	return func(err error, replay bool) {
		runner.observer.Observe(NodeObservation{
			NodeID: nodeID, StartedAt: started, Attempt: 1,
			Replay: replay, Failed: err != nil, DurationMS: time.Since(started).Milliseconds(),
		})
	}
}
