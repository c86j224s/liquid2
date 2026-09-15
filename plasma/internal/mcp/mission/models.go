package mission

import (
	"encoding/json"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	canonicalmission "github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
)

type getInput struct {
	MissionID string   `json:"mission_id"`
	Include   []string `json:"include"`
}

// UpdateInput is the complete mission.update wire payload used by both the
// feature handler and the root transport guard before binding or cache checks.
type UpdateInput struct {
	MissionID      string                  `json:"mission_id"`
	SessionID      string                  `json:"session_id"`
	IdempotencyKey string                  `json:"idempotency_key"`
	Producer       ledger.Producer         `json:"producer"`
	Title          *string                 `json:"title"`
	Objective      *string                 `json:"objective"`
	Scope          *canonicalmission.Scope `json:"scope"`
}

// GetOutput preserves the existing mission.get content wire shape.
type GetOutput struct {
	MissionProjection   canonicalmission.Projection      `json:"mission_projection"`
	Sources             []any                            `json:"sources,omitempty"`
	Evidence            []researchrecords.EvidenceRecord `json:"evidence,omitempty"`
	Claims              []researchrecords.ClaimRecord    `json:"claims,omitempty"`
	OpenQuestions       []researchrecords.QuestionRecord `json:"open_questions"`
	ActiveReportVersion any                              `json:"active_report_version"`
}

// SourceOutput maps the existing root-owned source projection at the boundary.
type SourceOutput = any

func errInvalidInput(format string, args ...any) error {
	return fmt.Errorf("%w: %s", producterror.ErrInvalidInput, fmt.Sprintf(format, args...))
}

func decodeArgs(args json.RawMessage, target any) error {
	if len(args) == 0 {
		args = json.RawMessage(`{}`)
	}
	if !json.Valid(args) {
		return errInvalidInput("tool arguments must be valid JSON")
	}
	if err := json.Unmarshal(args, target); err != nil {
		return errInvalidInput("decode tool arguments: %v", err)
	}
	return nil
}
