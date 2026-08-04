package producterror

import "errors"

var (
	// ErrInvalidInput classifies requests that violate a stable product contract.
	ErrInvalidInput = errors.New("invalid input")
	// ErrConflict classifies valid requests that cannot apply to current state.
	ErrConflict = errors.New("conflict")
)
