package app

import (
	"encoding/json"
	"fmt"
	"strings"
)

func normalizeConfluenceSearchRequest(req ConfluenceSourceSearchRequest) (ConfluenceSourceSearchRequest, error) {
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return ConfluenceSourceSearchRequest{}, err
	}
	cloudID := strings.TrimSpace(req.CloudID)
	if cloudID == "" {
		return ConfluenceSourceSearchRequest{}, fmt.Errorf("%w: confluence cloud id is required", ErrInvalidInput)
	}
	req.MissionID = missionID
	req.CloudID = cloudID
	req.SiteURL = strings.TrimSpace(req.SiteURL)
	req.Query = strings.TrimSpace(req.Query)
	req.Cursor = strings.TrimSpace(req.Cursor)
	req.SpaceID = strings.TrimSpace(req.SpaceID)
	req.SpaceKey = strings.TrimSpace(req.SpaceKey)
	req.Limit = normalizeConfluenceSearchLimit(req.Limit)
	return req, nil
}

func normalizeConfluenceSearchLimit(limit int) int {
	if limit <= 0 {
		return defaultConfluenceSearchLimit
	}
	if limit > maxConfluenceSearchLimit {
		return maxConfluenceSearchLimit
	}
	return limit
}

func normalizeConfluenceCandidate(candidate ConfluenceSourceCandidate, requestedCloudID string) (ConfluenceSourceCandidate, error) {
	connector, err := normalizeConfluenceConnector(candidate.Connector, requestedCloudID, "")
	if err != nil {
		return ConfluenceSourceCandidate{}, err
	}
	candidate.Connector = connector
	candidate.CloudID = requestedCloudID
	candidate.SiteURL = strings.TrimSpace(candidate.SiteURL)
	candidate.SpaceID = strings.TrimSpace(candidate.SpaceID)
	candidate.SpaceKey = strings.TrimSpace(candidate.SpaceKey)
	candidate.Title = strings.TrimSpace(candidate.Title)
	candidate.SourceURI = strings.TrimSpace(candidate.SourceURI)
	candidate.Summary = ""
	candidate.CanSnapshot = true
	return candidate, nil
}

func normalizeConfluencePage(
	page ConfluenceSourcePage,
	cloudID string,
	pageID string,
	expectedVersion int,
) (ConfluenceSourcePage, error) {
	page.CloudID = strings.TrimSpace(page.CloudID)
	if page.CloudID == "" {
		page.CloudID = cloudID
	} else if page.CloudID != cloudID {
		return ConfluenceSourcePage{}, NewConfluenceValidationError(
			ConfluenceErrorCodeCloudMismatch,
			"Confluence cloud id가 선택한 site와 일치하지 않습니다. site와 페이지를 다시 선택하세요.",
		)
	}
	page.PageID = strings.TrimSpace(page.PageID)
	if page.PageID == "" {
		page.PageID = pageID
	} else if page.PageID != pageID {
		return ConfluenceSourcePage{}, NewConfluenceValidationError(
			ConfluenceErrorCodePageMismatch,
			"Confluence page id가 선택한 후보와 일치하지 않습니다. 페이지를 다시 선택하세요.",
		)
	}
	if expectedVersion > 0 && page.Version != expectedVersion {
		return ConfluenceSourcePage{}, NewConfluenceValidationError(
			ConfluenceErrorCodeVersionDrift,
			"Confluence 페이지 버전이 검토 이후 변경되었습니다. 다시 미리보기한 뒤 승인하세요.",
		)
	}
	connector, err := normalizeConfluenceConnector(page.Connector, page.CloudID, page.PageID)
	if err != nil {
		return ConfluenceSourcePage{}, err
	}
	page.Connector = connector
	page.SiteURL = strings.TrimSpace(page.SiteURL)
	page.SpaceID = strings.TrimSpace(page.SpaceID)
	page.SpaceKey = strings.TrimSpace(page.SpaceKey)
	page.Title = strings.TrimSpace(page.Title)
	if page.Title == "" {
		page.Title = page.PageID
	}
	page.WebURL = strings.TrimSpace(page.WebURL)
	page.BodyStorage = strings.TrimSpace(page.BodyStorage)
	page.PlainText = strings.TrimSpace(page.PlainText)
	if page.BodyStorage == "" || page.PlainText == "" {
		return ConfluenceSourcePage{}, fmt.Errorf("%w: confluence storage body and plain text are required", ErrInvalidInput)
	}
	if len(page.Metadata) > 0 && !json.Valid(page.Metadata) {
		return ConfluenceSourcePage{}, fmt.Errorf("%w: confluence metadata must be valid JSON", ErrInvalidInput)
	}
	if len(page.Metadata) == 0 {
		page.Metadata = json.RawMessage(`{}`)
	}
	return page, nil
}

func normalizeConfluenceConnector(connector ConnectorRef, cloudID string, pageID string) (ConnectorRef, error) {
	connector = normalizeConnector(connector)
	if connector.ConnectorID == "" {
		connector.ConnectorID = ConfluenceConnectorID
	}
	if connector.ConnectorType == "" {
		connector.ConnectorType = ConfluenceConnectorType
	}
	if connector.ConnectorVersion == "" {
		connector.ConnectorVersion = ConfluenceHTTPConnectorV1
	}
	if connector.ExternalSourceID == "" && pageID != "" {
		connector.ExternalSourceID = ConfluenceExternalSourceID(cloudID, pageID)
	}
	if connector.ExternalSourceID == "" {
		return ConnectorRef{}, fmt.Errorf("%w: confluence external source id is required", ErrInvalidInput)
	}
	if connector.ExternalURI == "" && pageID != "" {
		connector.ExternalURI = ConfluenceExternalURI(cloudID, pageID)
	}
	return connector, nil
}

func confluenceSnapshotProducer(producer Producer) (Producer, error) {
	producer.Type = strings.TrimSpace(producer.Type)
	producer.ID = strings.TrimSpace(producer.ID)
	if producer.Type == "" && producer.ID == "" {
		return Producer{Type: "connector", ID: ConfluenceConnectorID}, nil
	}
	if producer.Type != "connector" || producer.ID != ConfluenceConnectorID {
		return Producer{}, fmt.Errorf("%w: confluence snapshot producer must be connector/confluence", ErrInvalidInput)
	}
	return producer, nil
}

func confluenceSnapshotFilename(externalSourceID string) string {
	externalSourceID = strings.TrimSpace(externalSourceID)
	if externalSourceID == "" {
		return "confluence-source.json"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_")
	return "confluence-" + replacer.Replace(externalSourceID) + ".json"
}

func normalizeConfluenceBrowseLimit(limit int) int {
	return normalizeConfluenceSearchLimit(limit)
}

func normalizeConfluenceSpaceListRequest(req ConfluenceSpaceListRequest) (ConfluenceSpaceListRequest, error) {
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return ConfluenceSpaceListRequest{}, err
	}
	cloudID := strings.TrimSpace(req.CloudID)
	if cloudID == "" {
		return ConfluenceSpaceListRequest{}, fmt.Errorf("%w: confluence cloud id is required", ErrInvalidInput)
	}
	req.MissionID = missionID
	req.CloudID = cloudID
	req.Cursor = strings.TrimSpace(req.Cursor)
	req.Limit = normalizeConfluenceBrowseLimit(req.Limit)
	return req, nil
}

func normalizeConfluenceSpacePagesRequest(req ConfluenceSpacePagesRequest) (ConfluenceSpacePagesRequest, error) {
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return ConfluenceSpacePagesRequest{}, err
	}
	cloudID := strings.TrimSpace(req.CloudID)
	spaceID := strings.TrimSpace(req.SpaceID)
	if cloudID == "" || spaceID == "" {
		return ConfluenceSpacePagesRequest{}, fmt.Errorf("%w: confluence cloud id and space id are required", ErrInvalidInput)
	}
	req.MissionID = missionID
	req.CloudID = cloudID
	req.SpaceID = spaceID
	req.Cursor = strings.TrimSpace(req.Cursor)
	req.Limit = normalizeConfluenceBrowseLimit(req.Limit)
	return req, nil
}

func normalizeConfluencePageChildrenRequest(req ConfluencePageChildrenRequest) (ConfluencePageChildrenRequest, error) {
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return ConfluencePageChildrenRequest{}, err
	}
	cloudID := strings.TrimSpace(req.CloudID)
	pageID := strings.TrimSpace(req.PageID)
	if cloudID == "" || pageID == "" {
		return ConfluencePageChildrenRequest{}, fmt.Errorf("%w: confluence cloud id and page id are required", ErrInvalidInput)
	}
	req.MissionID = missionID
	req.CloudID = cloudID
	req.PageID = pageID
	req.Cursor = strings.TrimSpace(req.Cursor)
	req.Limit = normalizeConfluenceBrowseLimit(req.Limit)
	return req, nil
}

func normalizeConfluenceSpaceSummary(space ConfluenceSpaceSummary, cloudID string) ConfluenceSpaceSummary {
	space.CloudID = strings.TrimSpace(space.CloudID)
	if space.CloudID == "" {
		space.CloudID = cloudID
	}
	space.SpaceID = strings.TrimSpace(space.SpaceID)
	space.SpaceKey = strings.TrimSpace(space.SpaceKey)
	space.Name = strings.TrimSpace(space.Name)
	if space.Name == "" {
		space.Name = firstNonEmpty(space.SpaceKey, space.SpaceID)
	}
	space.Type = strings.TrimSpace(space.Type)
	space.Status = strings.TrimSpace(space.Status)
	space.WebURL = strings.TrimSpace(space.WebURL)
	return space
}

func normalizeConfluencePageSummary(page ConfluencePageSummary, cloudID string) ConfluencePageSummary {
	page.CloudID = strings.TrimSpace(page.CloudID)
	if page.CloudID == "" {
		page.CloudID = cloudID
	}
	page.PageID = strings.TrimSpace(page.PageID)
	page.SpaceID = strings.TrimSpace(page.SpaceID)
	page.ParentID = strings.TrimSpace(page.ParentID)
	page.Title = strings.TrimSpace(page.Title)
	if page.Title == "" {
		page.Title = page.PageID
	}
	page.WebURL = strings.TrimSpace(page.WebURL)
	return page
}
