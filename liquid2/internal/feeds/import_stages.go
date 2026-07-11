package feeds

import (
	"context"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/jobs"
)

type NormalizeStage struct {
	normalizer *Normalizer
}

func NewNormalizeStage(normalizer *Normalizer) NormalizeStage {
	return NormalizeStage{normalizer: normalizer}
}

func (stage NormalizeStage) Name() string { return "normalize" }

func (stage NormalizeStage) Contract() jobs.StageContract {
	return jobs.StageContract{
		Input: "feeds.parsedContext", Output: "feeds.normalizedContext",
		Idempotency: "pure normalization", Retry: "runner retry",
	}
}

func (stage NormalizeStage) Run(_ context.Context, input jobs.StageInput) (jobs.StageOutput, error) {
	parsed, ok := input.Data.(parsedContext)
	if !ok {
		return jobs.StageOutput{}, parseFailed("normalize stage input is invalid")
	}
	items := stage.normalizer.Normalize(parsed.Fetched.URL, parsed.Parsed)
	return jobs.StageOutput{Data: normalizedContext{Feed: parsed.Feed, Items: items}}, nil
}

type ImportStage struct {
	documents *app.Service
}

func NewImportStage(documents *app.Service) ImportStage {
	return ImportStage{documents: documents}
}

func (stage ImportStage) Name() string { return "import" }

func (stage ImportStage) Contract() jobs.StageContract {
	return jobs.StageContract{
		Input: "feeds.normalizedContext", Output: "app.FeedImportResult",
		Idempotency: "app import de-duplicates by feed item keys", Retry: "runner retry",
	}
}

func (stage ImportStage) Run(ctx context.Context, input jobs.StageInput) (jobs.StageOutput, error) {
	normalized, ok := input.Data.(normalizedContext)
	if !ok {
		return jobs.StageOutput{}, parseFailed("import stage input is invalid")
	}
	result, err := stage.documents.ImportFeedItems(ctx, normalized.Feed.ID, normalized.Items)
	if err != nil {
		return jobs.StageOutput{}, err
	}
	return jobs.StageOutput{Data: result}, nil
}
