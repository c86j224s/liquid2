package app

type MissionScope struct {
	Included []string `json:"included"`
	Excluded []string `json:"excluded"`
}

type MissionProjection struct {
	MissionID             string       `json:"mission_id"`
	LastEventID           string       `json:"last_event_id"`
	LastSequence          int64        `json:"last_sequence"`
	Title                 string       `json:"title"`
	Objective             string       `json:"objective"`
	Scope                 MissionScope `json:"scope"`
	ActiveSessionIDs      []string     `json:"active_session_ids"`
	AcceptedClaimIDs      []string     `json:"accepted_claim_ids"`
	OpenQuestionIDs       []string     `json:"open_question_ids"`
	ActiveReportVersionID string       `json:"active_report_version_id"`
	LifecycleState        string       `json:"lifecycle_state"`
	NeedsReview           bool         `json:"needs_review"`
	NeedsReviewReasons    []string     `json:"needs_review_reasons"`
}
