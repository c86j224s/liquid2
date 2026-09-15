package reportdocument

import "time"

// Approval records approval state for report blocks and other app-owned objects.
type Approval struct {
	State           string    `json:"state"`
	Required        bool      `json:"required"`
	ApprovalEventID string    `json:"approval_event_id,omitempty"`
	ApprovedAt      time.Time `json:"approved_at,omitempty"`
}
