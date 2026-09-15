package confluencesource

import (
	"fmt"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"strconv"
	"strings"
)

func NormalizeVersion(version ConfluenceSourceVersion, cloudID string, pageID string) (ConfluenceSourceVersion, error) {
	version.CloudID = strings.TrimSpace(version.CloudID)
	if version.CloudID == "" {
		version.CloudID = cloudID
	} else if version.CloudID != cloudID {
		return ConfluenceSourceVersion{}, NewConfluenceValidationError(
			ConfluenceErrorCodeCloudMismatch,
			"Confluence cloud id가 저장된 소스와 일치하지 않습니다. 연결 site를 확인하세요.",
		)
	}
	version.PageID = strings.TrimSpace(version.PageID)
	if version.PageID == "" {
		version.PageID = pageID
	} else if version.PageID != pageID {
		return ConfluenceSourceVersion{}, NewConfluenceValidationError(
			ConfluenceErrorCodePageMismatch,
			"Confluence page id가 저장된 소스와 일치하지 않습니다. 페이지를 다시 확인하세요.",
		)
	}
	version.Title = strings.TrimSpace(version.Title)
	if version.Title == "" {
		version.Title = version.PageID
	}
	return version, nil
}

func SnapshotVersion(snapshot sourcecontract.Snapshot) int {
	if parsed, err := strconv.Atoi(strings.TrimSpace(snapshot.Connector.ExternalVersion)); err == nil {
		return parsed
	}
	return 0
}

func SnapshotNewerThan(candidate sourcecontract.Snapshot, current sourcecontract.Snapshot) bool {
	candidateVersion := SnapshotVersion(candidate)
	currentVersion := SnapshotVersion(current)
	if candidateVersion != currentVersion {
		return candidateVersion > currentVersion
	}
	if !candidate.ExternalUpdatedAt.Equal(current.ExternalUpdatedAt) {
		return candidate.ExternalUpdatedAt.After(current.ExternalUpdatedAt)
	}
	if !candidate.CapturedAt.Equal(current.CapturedAt) {
		return candidate.CapturedAt.After(current.CapturedAt)
	}
	return candidate.SnapshotID > current.SnapshotID
}

func ValidateSnapshotBodySize(page ConfluenceSourcePage, maxBytes int64) error {
	maxBytes = NormalizeMaxBodyBytes(maxBytes)
	bodyBytes := int64(len([]byte(page.BodyStorage)))
	if bodyBytes > maxBytes {
		return NewConfluenceValidationError(
			ConfluenceErrorCodeTooLarge,
			fmt.Sprintf("Confluence 페이지가 너무 큽니다. 전체 %d bytes가 한도 %d bytes를 넘었습니다. 미리보기에서 범위를 선택하세요.", bodyBytes, maxBytes),
		)
	}
	return nil
}

func ValidateSelectedRange(page ConfluenceSourcePage, selection ConfluenceRangeSelection, maxBytes int64) error {
	body, err := RangeBody(page.PlainText, selection)
	if err != nil {
		return err
	}
	maxBytes = NormalizeMaxBodyBytes(maxBytes)
	bodyBytes := int64(len([]byte(body.Content)))
	if bodyBytes > maxBytes {
		return NewConfluenceValidationError(
			ConfluenceErrorCodeTooLarge,
			fmt.Sprintf("선택한 Confluence 범위가 너무 큽니다. 선택 범위 %d bytes가 한도 %d bytes를 넘었습니다.", bodyBytes, maxBytes),
		)
	}
	return nil
}

func NormalizeMaxBodyBytes(maxBytes int64) int64 {
	if maxBytes <= 0 {
		return DefaultConfluenceMaxBodyBytes
	}
	return maxBytes
}

func PreviewText(plainText string, limit int) (string, bool) {
	if limit <= 0 {
		limit = 1200
	}
	runes := []rune(plainText)
	if len(runes) <= limit {
		return plainText, false
	}
	return string(runes[:limit]), true
}

func RangeOptions(plainText string, maxBytes int64) []ConfluenceRangeOption {
	runes := []rune(plainText)
	if len(runes) == 0 {
		return nil
	}
	maxBytes = NormalizeMaxBodyBytes(maxBytes)
	const maxRunesPerOption = 4000
	options := []ConfluenceRangeOption{}
	for start := 0; start < len(runes) && len(options) < 20; {
		end := start
		bytes := int64(0)
		for end < len(runes) && end-start < maxRunesPerOption {
			nextBytes := int64(len(string(runes[end])))
			if end > start && bytes+nextBytes > maxBytes {
				break
			}
			if end == start && nextBytes > maxBytes {
				start++
				break
			}
			bytes += nextBytes
			end++
		}
		if end <= start {
			continue
		}
		options = append(options, ConfluenceRangeOption{
			ContentID: "plain_text",
			Label:     fmt.Sprintf("문자 %d-%d", start, end),
			Start:     start,
			End:       end,
			RuneCount: end - start,
		})
		start = end
	}
	return options
}
