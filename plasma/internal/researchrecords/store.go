package researchrecords

import "context"

// ClaimStore persists and reads claim records.
type ClaimStore interface {
	CreateClaimRecord(context.Context, ClaimRecord) error
	GetClaimRecord(context.Context, string) (ClaimRecord, error)
}

// ClaimListStore lists claim records for one mission.
type ClaimListStore interface {
	ListClaimRecords(context.Context, string) ([]ClaimRecord, error)
}

// QuestionStore는 question record 저장과 단건 조회 계약이다.
type QuestionStore interface {
	CreateQuestionRecord(context.Context, QuestionRecord) error
	GetQuestionRecord(context.Context, string) (QuestionRecord, error)
}

// QuestionListStore는 mission별 question record 목록 조회 계약이다.
type QuestionListStore interface {
	ListQuestionRecords(context.Context, string) ([]QuestionRecord, error)
}

// OptionStore는 option record 저장과 단건 조회 계약이다.
type OptionStore interface {
	CreateOptionRecord(context.Context, OptionRecord) error
	GetOptionRecord(context.Context, string) (OptionRecord, error)
}

// OptionListStore는 mission별 option record 목록 조회 계약이다.
type OptionListStore interface {
	ListOptionRecords(context.Context, string) ([]OptionRecord, error)
}
