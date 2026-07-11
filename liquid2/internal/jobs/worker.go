package jobs

import (
	"context"
	"errors"
	"runtime/debug"
)

type Handler func(context.Context, Job) error

type workerEvent struct {
	err        error
	panicValue any
	stack      []byte
}

func runWorker(ctx context.Context, job Job, handler Handler) workerEvent {
	done := make(chan workerEvent, 1)
	go func() {
		var event workerEvent
		defer func() {
			if recovered := recover(); recovered != nil {
				event.panicValue = recovered
				event.stack = debug.Stack()
			}
			done <- event
		}()
		event.err = handler(ctx, job)
	}()
	return <-done
}

func (event workerEvent) failed() bool {
	return event.err != nil || event.panicValue != nil
}

func (event workerEvent) safeMessage() string {
	if event.panicValue != nil {
		return "job worker panicked"
	}
	if errors.Is(event.err, context.Canceled) {
		return "job canceled"
	}
	if event.err != nil {
		return "job failed"
	}
	return ""
}
