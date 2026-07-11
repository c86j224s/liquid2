package feeds

import "time"

type timeSchedulerTicker struct {
	ticker *time.Ticker
}

func newSchedulerTicker(interval time.Duration) schedulerTicker {
	return timeSchedulerTicker{ticker: time.NewTicker(interval)}
}

func (ticker timeSchedulerTicker) C() <-chan time.Time {
	return ticker.ticker.C
}

func (ticker timeSchedulerTicker) Stop() {
	ticker.ticker.Stop()
}
