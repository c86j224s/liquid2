package confluence

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func (client *Client) ListConfluenceSpaces(
	ctx context.Context,
	req app.ConfluenceSpaceListRequest,
) (app.ConfluenceSpaceListResult, error) {
	if err := client.validateCloudID(req.CloudID); err != nil {
		return app.ConfluenceSpaceListResult{}, err
	}
	query := browseQuery(req.Limit, req.Cursor)
	var response confluenceSpacesResponse
	if err := client.getJSON(ctx, "/api/v2/spaces", query, &response); err != nil {
		return app.ConfluenceSpaceListResult{}, err
	}
	spaces := make([]app.ConfluenceSpaceSummary, 0, len(response.Results))
	for _, item := range response.Results {
		spaces = append(spaces, app.ConfluenceSpaceSummary{
			CloudID:  client.cloudID,
			SpaceID:  item.ID,
			SpaceKey: item.Key,
			Name:     item.Name,
			Type:     item.Type,
			Status:   item.Status,
			WebURL:   client.absoluteURL(response.Links.Base, item.Links.WebUI),
		})
	}
	return app.ConfluenceSpaceListResult{
		MissionID:  req.MissionID,
		CloudID:    client.cloudID,
		Spaces:     spaces,
		NextCursor: cursorFromNextLink(response.Links.Next),
	}, nil
}

func (client *Client) ListConfluenceSpacePages(
	ctx context.Context,
	req app.ConfluenceSpacePagesRequest,
) (app.ConfluencePageListResult, error) {
	if err := client.validateCloudID(req.CloudID); err != nil {
		return app.ConfluencePageListResult{}, err
	}
	spaceID := strings.TrimSpace(req.SpaceID)
	query := browseQuery(req.Limit, req.Cursor)
	var response confluencePageListResponse
	if err := client.getJSON(ctx, "/api/v2/spaces/"+url.PathEscape(spaceID)+"/pages", query, &response); err != nil {
		return app.ConfluencePageListResult{}, err
	}
	return client.pageListResult(req.MissionID, response), nil
}

func (client *Client) ListConfluencePageChildren(
	ctx context.Context,
	req app.ConfluencePageChildrenRequest,
) (app.ConfluencePageListResult, error) {
	if err := client.validateCloudID(req.CloudID); err != nil {
		return app.ConfluencePageListResult{}, err
	}
	pageID := strings.TrimSpace(req.PageID)
	query := browseQuery(req.Limit, req.Cursor)
	var response confluencePageListResponse
	if err := client.getJSON(ctx, "/api/v2/pages/"+url.PathEscape(pageID)+"/children", query, &response); err != nil {
		return app.ConfluencePageListResult{}, err
	}
	result := client.pageListResult(req.MissionID, response)
	for i := range result.Pages {
		if strings.TrimSpace(result.Pages[i].ParentID) == "" {
			result.Pages[i].ParentID = pageID
		}
	}
	return result, nil
}

func browseQuery(limit int, cursor string) url.Values {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	if strings.TrimSpace(cursor) != "" {
		query.Set("cursor", strings.TrimSpace(cursor))
	}
	return query
}

func (client *Client) pageListResult(missionID string, response confluencePageListResponse) app.ConfluencePageListResult {
	pages := make([]app.ConfluencePageSummary, 0, len(response.Results))
	for _, item := range response.Results {
		pages = append(pages, app.ConfluencePageSummary{
			CloudID:     client.cloudID,
			PageID:      item.ID,
			SpaceID:     item.SpaceID,
			ParentID:    item.ParentID,
			Title:       item.Title,
			WebURL:      client.absoluteURL(response.Links.Base, item.Links.WebUI),
			Version:     item.Version.Number,
			UpdatedAt:   parseConfluenceTime(item.Version.CreatedAt),
			HasChildren: true,
		})
	}
	return app.ConfluencePageListResult{
		MissionID:  missionID,
		CloudID:    client.cloudID,
		Pages:      pages,
		NextCursor: cursorFromNextLink(response.Links.Next),
	}
}
