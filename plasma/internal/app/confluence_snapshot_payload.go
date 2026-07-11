package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type confluenceSnapshotArtifact struct {
	SchemaVersion string                   `json:"schema_version"`
	Connector     ConnectorRef             `json:"connector"`
	Page          confluenceSnapshotPage   `json:"page"`
	Contents      []confluenceSnapshotBody `json:"contents"`
	Reason        string                   `json:"reason,omitempty"`
	Metadata      json.RawMessage          `json:"metadata"`
}

type confluenceSnapshotPage struct {
	CloudID         string `json:"cloud_id"`
	SiteURL         string `json:"site_url,omitempty"`
	PageID          string `json:"page_id"`
	SpaceID         string `json:"space_id,omitempty"`
	SpaceKey        string `json:"space_key,omitempty"`
	Title           string `json:"title"`
	WebURL          string `json:"web_url,omitempty"`
	Version         int    `json:"version,omitempty"`
	UpdatedAt       string `json:"updated_at,omitempty"`
	Partial         bool   `json:"partial,omitempty"`
	StorageSHA256   string `json:"storage_sha256"`
	PlainTextSHA256 string `json:"plain_text_sha256"`
}

type confluenceSnapshotBody struct {
	ContentID string `json:"content_id"`
	Role      string `json:"role"`
	Format    string `json:"format"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
	Content   string `json:"content"`
}

type confluenceSnapshotLocator struct {
	LocatorType string `json:"locator_type"`
	ArtifactID  string `json:"artifact_id"`
	CloudID     string `json:"cloud_id"`
	SiteURL     string `json:"site_url,omitempty"`
	PageID      string `json:"page_id"`
	ContentID   string `json:"content_id"`
	Format      string `json:"format"`
	Start       int    `json:"start"`
	End         int    `json:"end"`
	Partial     bool   `json:"partial,omitempty"`
}

func buildConfluenceSnapshotPayload(
	page ConfluenceSourcePage,
	artifactID string,
	reason string,
	selection ConfluenceRangeSelection,
) ([]byte, json.RawMessage, error) {
	if strings.TrimSpace(page.BodyStorage) == "" || strings.TrimSpace(page.PlainText) == "" {
		return nil, nil, fmt.Errorf("%w: confluence storage body and plain text are required", ErrInvalidInput)
	}
	partial := confluenceRangeSelected(selection)
	contents := []confluenceSnapshotBody{}
	if partial {
		body, err := confluenceRangeBody(page.PlainText, selection)
		if err != nil {
			return nil, nil, err
		}
		contents = append(contents, body)
	} else {
		storage := confluenceBody("storage", "source", "confluence_storage", page.BodyStorage)
		plainText := confluenceBody("plain_text", "plain_text", "text", page.PlainText)
		contents = append(contents, storage, plainText)
	}
	updatedAt := ""
	if !page.UpdatedAt.IsZero() {
		updatedAt = page.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	payload := confluenceSnapshotArtifact{
		SchemaVersion: ConfluenceSnapshotSchemaV1,
		Connector:     page.Connector,
		Page: confluenceSnapshotPage{
			CloudID:         page.CloudID,
			SiteURL:         page.SiteURL,
			PageID:          page.PageID,
			SpaceID:         page.SpaceID,
			SpaceKey:        page.SpaceKey,
			Title:           page.Title,
			WebURL:          page.WebURL,
			Version:         page.Version,
			UpdatedAt:       updatedAt,
			Partial:         partial,
			StorageSHA256:   sha256Hex(page.BodyStorage),
			PlainTextSHA256: sha256Hex(page.PlainText),
		},
		Contents: contents,
		Reason:   reason,
		Metadata: append(json.RawMessage(nil), page.Metadata...),
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	locators := make([]confluenceSnapshotLocator, 0, len(contents))
	for _, body := range contents {
		locator := confluenceLocator(page, artifactID, body)
		if partial {
			locator.LocatorType = "confluence_page_range"
			locator.Partial = true
		}
		locators = append(locators, locator)
	}
	locatorJSON, err := json.Marshal(locators)
	if err != nil {
		return nil, nil, err
	}
	return content, locatorJSON, nil
}

func confluenceBody(contentID string, role string, format string, content string) confluenceSnapshotBody {
	return confluenceSnapshotBody{
		ContentID: contentID,
		Role:      role,
		Format:    format,
		Start:     0,
		End:       len([]rune(content)),
		Content:   content,
	}
}

func confluenceLocator(
	page ConfluenceSourcePage,
	artifactID string,
	content confluenceSnapshotBody,
) confluenceSnapshotLocator {
	return confluenceSnapshotLocator{
		LocatorType: "confluence_page_body",
		ArtifactID:  artifactID,
		CloudID:     page.CloudID,
		SiteURL:     page.SiteURL,
		PageID:      page.PageID,
		ContentID:   content.ContentID,
		Format:      content.Format,
		Start:       content.Start,
		End:         content.End,
	}
}

func confluenceRangeBody(plainText string, selection ConfluenceRangeSelection) (confluenceSnapshotBody, error) {
	selection.ContentID = strings.TrimSpace(selection.ContentID)
	if selection.ContentID == "" {
		selection.ContentID = "plain_text"
	}
	if selection.ContentID != "plain_text" {
		return confluenceSnapshotBody{}, NewConfluenceValidationError(
			ConfluenceErrorCodeTooLarge,
			"현재 Confluence 범위 스냅샷은 plain_text content id만 지원합니다.",
		)
	}
	runes := []rune(plainText)
	if selection.Start < 0 || selection.End <= selection.Start || selection.End > len(runes) {
		return confluenceSnapshotBody{}, NewConfluenceValidationError(
			ConfluenceErrorCodeTooLarge,
			"Confluence 범위가 원문 길이와 맞지 않습니다. 다시 미리보기한 뒤 범위를 선택하세요.",
		)
	}
	content := string(runes[selection.Start:selection.End])
	return confluenceSnapshotBody{
		ContentID: selection.ContentID,
		Role:      "source",
		Format:    "text",
		Start:     selection.Start,
		End:       selection.End,
		Content:   content,
	}, nil
}

func confluenceRangeSelected(selection ConfluenceRangeSelection) bool {
	return strings.TrimSpace(selection.ContentID) != "" || selection.Start > 0 || selection.End > 0
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
