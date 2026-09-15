package confluencesource

// ConfluenceRangeSelection는 Confluence page 본문 중 source로 삼을 범위를 나타낸다.
type ConfluenceRangeSelection struct {
	ContentID string `json:"content_id,omitempty"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
}

// ConfluenceRangeOption는 애플리케이션 서비스 계층 실행 옵션이다. 0 값과 누락 값의 의미는 생성자나 Normalize 경계가 정한다.
type ConfluenceRangeOption struct {
	ContentID string `json:"content_id"`
	Label     string `json:"label"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
	RuneCount int    `json:"rune_count"`
}

const DefaultConfluenceMaxBodyBytes = int64(1024 * 1024)
