package app

import (
	"context"
	"encoding/json"
	"fmt"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"strings"
)

// PreviewConfluenceSource는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) PreviewConfluenceSource(
	ctx context.Context,
	connector confluencesource.ConfluenceSourceConnector,
	req ConfluenceSourcePreviewRequest,
) (ConfluenceSourcePreviewResult, error) {
	if connector == nil {
		return ConfluenceSourcePreviewResult{}, fmt.Errorf("%w: confluence connector is required", ErrInvalidInput)
	}
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return ConfluenceSourcePreviewResult{}, err
	}
	cloudID := strings.TrimSpace(req.CloudID)
	pageID := strings.TrimSpace(req.PageID)
	if cloudID == "" || pageID == "" {
		return ConfluenceSourcePreviewResult{}, fmt.Errorf("%w: confluence cloud id and page id are required", ErrInvalidInput)
	}
	page, err := connector.ReadConfluenceSource(ctx, confluencesource.ConfluenceSourceReadRequest{CloudID: cloudID, PageID: pageID})
	if err != nil {
		return ConfluenceSourcePreviewResult{}, err
	}
	page, err = confluencesource.NormalizePage(page, cloudID, pageID, req.ExpectedVersion)
	if err != nil {
		return ConfluenceSourcePreviewResult{}, err
	}
	maxBytes := confluencesource.NormalizeMaxBodyBytes(req.MaxBodyBytes)
	bodyBytes := int64(len([]byte(page.BodyStorage)))
	preview, truncated := confluencesource.PreviewText(page.PlainText, req.PreviewRunes)
	result := ConfluenceSourcePreviewResult{
		MissionID:        missionID,
		CandidateKind:    "confluence_page_preview_result",
		Page:             confluencePreviewPage(page),
		PreviewText:      preview,
		PreviewTruncated: truncated,
		BodyBytes:        bodyBytes,
		MaxBodyBytes:     maxBytes,
		FullBodyTooLarge: bodyBytes > maxBytes,
		RangeOptions:     confluencesource.RangeOptions(page.PlainText, maxBytes),
	}
	if !result.FullBodyTooLarge && len(result.RangeOptions) > 4 {
		result.RangeOptions = result.RangeOptions[:4]
	}
	return result, nil
}

// PreviewConfluenceSourceUpdate는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) PreviewConfluenceSourceUpdate(
	ctx context.Context,
	connector confluencesource.ConfluenceSourceConnector,
	req ConfluenceUpdatePreviewRequest,
) (ConfluenceUpdatePreviewResult, error) {
	if connector == nil {
		return ConfluenceUpdatePreviewResult{}, fmt.Errorf("%w: confluence connector is required", ErrInvalidInput)
	}
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return ConfluenceUpdatePreviewResult{}, err
	}
	previous, identity, err := s.activeConfluenceSnapshotIdentity(ctx, missionID, req.SnapshotID)
	if err != nil {
		return ConfluenceUpdatePreviewResult{}, err
	}
	page, err := connector.ReadConfluenceSource(ctx, confluencesource.ConfluenceSourceReadRequest{CloudID: identity.CloudID, PageID: identity.PageID})
	if err != nil {
		return ConfluenceUpdatePreviewResult{}, err
	}
	page, err = confluencesource.NormalizePage(page, identity.CloudID, identity.PageID, req.ExpectedVersion)
	if err != nil {
		return ConfluenceUpdatePreviewResult{}, err
	}
	maxBytes := confluencesource.NormalizeMaxBodyBytes(req.MaxBodyBytes)
	bodyBytes := int64(len([]byte(page.BodyStorage)))
	preview, truncated := confluencesource.PreviewText(page.PlainText, req.PreviewRunes)
	previousRange, hasPreviousRange := confluenceRangeFromSnapshot(previous)
	return ConfluenceUpdatePreviewResult{
		Snapshot:               previous,
		OldPage:                confluencePreviewPageFromSnapshot(previous, identity),
		NewPage:                confluencePreviewPage(page),
		UpdateAvailable:        page.Version > confluencesource.SnapshotVersion(previous),
		PreviewText:            preview,
		PreviewTruncated:       truncated,
		BodyBytes:              bodyBytes,
		MaxBodyBytes:           maxBytes,
		FullBodyTooLarge:       bodyBytes > maxBytes,
		RangeOptions:           confluencesource.RangeOptions(page.PlainText, maxBytes),
		RequiresRangeReselect:  hasPreviousRange,
		PreviousRangeSelection: previousRange,
	}, nil
}

func confluencePreviewPage(page confluencesource.ConfluenceSourcePage) ConfluenceSourcePreviewPage {
	return ConfluenceSourcePreviewPage{
		CloudID:   page.CloudID,
		SiteURL:   page.SiteURL,
		PageID:    page.PageID,
		SpaceID:   page.SpaceID,
		SpaceKey:  page.SpaceKey,
		Title:     page.Title,
		WebURL:    page.WebURL,
		Version:   page.Version,
		UpdatedAt: page.UpdatedAt,
	}
}

func confluencePreviewPageFromSnapshot(snapshot sourcecontract.Snapshot, identity confluenceSourceIdentity) ConfluenceSourcePreviewPage {
	return ConfluenceSourcePreviewPage{
		CloudID:   identity.CloudID,
		PageID:    identity.PageID,
		Title:     snapshot.Title,
		Version:   confluencesource.SnapshotVersion(snapshot),
		UpdatedAt: snapshot.ExternalUpdatedAt,
	}
}

func confluenceRangeFromSnapshot(snapshot sourcecontract.Snapshot) (confluencesource.ConfluenceRangeSelection, bool) {
	var locators []struct {
		LocatorType string `json:"locator_type"`
		ContentID   string `json:"content_id"`
		Start       int    `json:"start"`
		End         int    `json:"end"`
		Partial     bool   `json:"partial"`
	}
	if len(snapshot.Locators) == 0 || json.Unmarshal(snapshot.Locators, &locators) != nil {
		return confluencesource.ConfluenceRangeSelection{}, false
	}
	for _, locator := range locators {
		if locator.LocatorType == "confluence_page_range" || locator.Partial {
			return confluencesource.ConfluenceRangeSelection{
				ContentID: strings.TrimSpace(locator.ContentID),
				Start:     locator.Start,
				End:       locator.End,
			}, true
		}
	}
	return confluencesource.ConfluenceRangeSelection{}, false
}
