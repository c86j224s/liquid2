package feeds

import (
	"encoding/json"
	"strings"
)

type PollFeedPayload struct {
	FeedID string `json:"feedId"`
}

func EncodePollFeedPayload(feedID string) (string, error) {
	payload := PollFeedPayload{FeedID: strings.TrimSpace(feedID)}
	if payload.FeedID == "" {
		return "", invalidPayload("feed id is required")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", invalidPayload("encode poll feed payload", err)
	}
	return string(data), nil
}

func DecodePollFeedPayload(raw string) (PollFeedPayload, error) {
	var payload PollFeedPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return PollFeedPayload{}, invalidPayload("decode poll feed payload", err)
	}
	payload.FeedID = strings.TrimSpace(payload.FeedID)
	if payload.FeedID == "" {
		return PollFeedPayload{}, invalidPayload("feed id is required")
	}
	return payload, nil
}
