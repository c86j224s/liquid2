package feeds

import (
	"context"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/jobs"
)

type FetchStage struct {
	fetcher Fetcher
}

func NewFetchStage(fetcher Fetcher) FetchStage {
	return FetchStage{fetcher: fetcher}
}

func (stage FetchStage) Name() string { return "fetch" }

func (stage FetchStage) Contract() jobs.StageContract {
	return jobs.StageContract{
		Input: "app.Feed", Output: "feeds.fetchedContext",
		Idempotency: "read-only remote fetch", Retry: "runner retry",
	}
}

func (stage FetchStage) Run(ctx context.Context, input jobs.StageInput) (jobs.StageOutput, error) {
	feed, ok := input.Data.(app.Feed)
	if !ok {
		return jobs.StageOutput{}, fetchFailed("fetch stage input is invalid")
	}
	fetched, err := stage.fetcher.Fetch(ctx, feed.URL)
	if err != nil {
		return jobs.StageOutput{}, err
	}
	return jobs.StageOutput{Data: fetchedContext{Feed: feed, Fetched: fetched}}, nil
}

type ParseStage struct {
	parser Parser
}

func NewParseStage(parser Parser) ParseStage {
	return ParseStage{parser: parser}
}

func (stage ParseStage) Name() string { return "parse" }

func (stage ParseStage) Contract() jobs.StageContract {
	return jobs.StageContract{
		Input: "feeds.fetchedContext", Output: "feeds.parsedContext",
		Idempotency: "pure parse", Retry: "runner retry",
	}
}

func (stage ParseStage) Run(ctx context.Context, input jobs.StageInput) (jobs.StageOutput, error) {
	fetched, ok := input.Data.(fetchedContext)
	if !ok {
		return jobs.StageOutput{}, parseFailed("parse stage input is invalid")
	}
	parsed, err := stage.parser.Parse(ctx, fetched.Fetched)
	if err != nil {
		return jobs.StageOutput{}, err
	}
	return jobs.StageOutput{Data: parsedContext{
		Feed: fetched.Feed, Fetched: fetched.Fetched, Parsed: parsed,
	}}, nil
}
