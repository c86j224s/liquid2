package app

import (
	"errors"
	"fmt"
)

var (
	ErrConflict   = errors.New("conflict")
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("validation failed")
)

func conflict(message string) error {
	return fmt.Errorf("%w: %s", ErrConflict, message)
}

func notFound(resource string) error {
	return fmt.Errorf("%w: %s not found", ErrNotFound, resource)
}

func validation(message string) error {
	return fmt.Errorf("%w: %s", ErrValidation, message)
}
