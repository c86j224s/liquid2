package jobs

import "context"

type StageInput struct {
	Job  Job
	Data any
}

type StageOutput struct {
	Data any
}

type StageContract struct {
	Input       string
	Output      string
	Idempotency string
	Retry       string
}

type Stage interface {
	Name() string
	Contract() StageContract
	Run(context.Context, StageInput) (StageOutput, error)
}
