package translation

import "context"

// PassthroughProvider is an explicit local provider for development and tests.
type PassthroughProvider struct{}

func (PassthroughProvider) Translate(_ context.Context, request Request) (Result, error) {
	return Result{Content: request.Text, Format: request.Format}, nil
}
