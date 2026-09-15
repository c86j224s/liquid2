package liquid2source

import (
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"strings"
)

func selectLiquid2SnapshotContents(
	document Liquid2SourceDocument,
	artifactID string,
	ranges []Liquid2ContentRange,
) ([]liquid2SnapshotContent, []liquid2SnapshotLocator, error) {
	if len(document.Contents) == 0 {
		return nil, nil, fmt.Errorf("%w: liquid2 source content is required", producterror.ErrInvalidInput)
	}
	byID, err := liquid2ContentsByID(document.Contents)
	if err != nil {
		return nil, nil, err
	}
	if len(ranges) == 0 {
		selected := make([]liquid2SnapshotContent, 0, len(document.Contents))
		locators := make([]liquid2SnapshotLocator, 0, len(document.Contents))
		for _, content := range document.Contents {
			item, err := fullLiquid2SnapshotContent(content)
			if err != nil {
				return nil, nil, err
			}
			selected = append(selected, item)
			locators = append(locators, liquid2Locator(document, artifactID, item))
		}
		return selected, locators, nil
	}

	selected := make([]liquid2SnapshotContent, 0, len(ranges))
	locators := make([]liquid2SnapshotLocator, 0, len(ranges))
	for _, requestedRange := range ranges {
		contentID := strings.TrimSpace(requestedRange.ContentID)
		content, ok := byID[contentID]
		if !ok || contentID == "" {
			return nil, nil, fmt.Errorf("%w: liquid2 content range references unknown content", producterror.ErrInvalidInput)
		}
		item, err := rangedLiquid2SnapshotContent(content, requestedRange.Start, requestedRange.End)
		if err != nil {
			return nil, nil, err
		}
		selected = append(selected, item)
		locators = append(locators, liquid2Locator(document, artifactID, item))
	}
	return selected, locators, nil
}

func liquid2ContentsByID(contents []Liquid2SourceContent) (map[string]Liquid2SourceContent, error) {
	byID := make(map[string]Liquid2SourceContent, len(contents))
	for _, content := range contents {
		contentID := strings.TrimSpace(content.ContentID)
		if contentID == "" {
			return nil, fmt.Errorf("%w: liquid2 content id is required", producterror.ErrInvalidInput)
		}
		if strings.TrimSpace(content.Content) == "" {
			return nil, fmt.Errorf("%w: liquid2 content body is required", producterror.ErrInvalidInput)
		}
		if _, exists := byID[contentID]; exists {
			return nil, fmt.Errorf("%w: duplicate liquid2 content id", producterror.ErrInvalidInput)
		}
		byID[contentID] = content
	}
	return byID, nil
}

func fullLiquid2SnapshotContent(content Liquid2SourceContent) (liquid2SnapshotContent, error) {
	if strings.TrimSpace(content.ContentID) == "" || strings.TrimSpace(content.Content) == "" {
		return liquid2SnapshotContent{}, fmt.Errorf("%w: liquid2 content id and body are required", producterror.ErrInvalidInput)
	}
	return liquid2SnapshotContent{
		ContentID: strings.TrimSpace(content.ContentID),
		Role:      strings.TrimSpace(content.Role),
		Format:    strings.TrimSpace(content.Format),
		Language:  strings.TrimSpace(content.Language),
		Start:     0,
		End:       len([]rune(content.Content)),
		Content:   content.Content,
	}, nil
}

func rangedLiquid2SnapshotContent(content Liquid2SourceContent, start int, end int) (liquid2SnapshotContent, error) {
	if strings.TrimSpace(content.ContentID) == "" || strings.TrimSpace(content.Content) == "" {
		return liquid2SnapshotContent{}, fmt.Errorf("%w: liquid2 content id and body are required", producterror.ErrInvalidInput)
	}
	runes := []rune(content.Content)
	if start < 0 || end <= start || end > len(runes) {
		return liquid2SnapshotContent{}, fmt.Errorf("%w: invalid liquid2 content range", producterror.ErrInvalidInput)
	}
	return liquid2SnapshotContent{
		ContentID: strings.TrimSpace(content.ContentID),
		Role:      strings.TrimSpace(content.Role),
		Format:    strings.TrimSpace(content.Format),
		Language:  strings.TrimSpace(content.Language),
		Start:     start,
		End:       end,
		Content:   string(runes[start:end]),
	}, nil
}

func liquid2Locator(document Liquid2SourceDocument, artifactID string, content liquid2SnapshotContent) liquid2SnapshotLocator {
	return liquid2SnapshotLocator{
		LocatorType:      "liquid2_content_range",
		ArtifactID:       artifactID,
		ExternalSourceID: document.Connector.ExternalSourceID,
		SourceURI:        document.SourceURI,
		ContentID:        content.ContentID,
		Role:             content.Role,
		Format:           content.Format,
		Start:            content.Start,
		End:              content.End,
	}
}
