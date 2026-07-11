package httptransport

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
	feedrefresh "github.com/c86j224s/liquid2/internal/feeds"
)

func TestFeedRoutes(t *testing.T) {
	service := app.NewService(app.WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	router := NewRouter(service)

	response := serveJSON(router, http.MethodPost, "/api/v1/feeds", `{"url":"https://example.com/feed.xml","title":"Example"}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"url":"https://example.com/feed.xml"`) {
		t.Fatalf("expected feed body, got %s", response.Body.String())
	}
	feedID := responseID(t, response.Body.String())

	response = serveJSON(router, http.MethodGet, "/api/v1/feeds", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"items"`) {
		t.Fatalf("expected feed list, got %d: %s", response.Code, response.Body.String())
	}
	response = serveJSON(router, http.MethodPatch, "/api/v1/feeds/"+feedID, `{"enabled":false}`)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"enabled":false`) {
		t.Fatalf("expected disabled feed, got %d: %s", response.Code, response.Body.String())
	}
	response = serveJSON(router, http.MethodDelete, "/api/v1/feeds/"+feedID, "")
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected delete status 204, got %d: %s", response.Code, response.Body.String())
	}
}

func TestJobRoutes(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	router := NewRouter(service)

	response := serveJSON(router, http.MethodGet, "/api/v1/jobs", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"items":[]`) {
		t.Fatalf("expected empty jobs, got %d: %s", response.Code, response.Body.String())
	}
	response = serveJSON(router, http.MethodGet, "/api/v1/jobs/missing", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected missing job status 404, got %d: %s", response.Code, response.Body.String())
	}
}

type fakeFeedRefresher struct {
	job app.Job
	err error
	id  string
}

func (refresher *fakeFeedRefresher) RefreshFeed(_ context.Context, feedID string) (app.Job, error) {
	refresher.id = feedID
	return refresher.job, refresher.err
}

func TestFeedRefreshRoute(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	refresher := &fakeFeedRefresher{job: app.Job{
		ID: "job_1", Kind: app.JobKindPollFeed, Status: app.JobStatusQueued,
	}}
	router := NewRouter(service, WithFeedRefresher(refresher))

	response := serveJSON(router, http.MethodPost, "/api/v1/feeds/feed_1/refresh", "")
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected refresh status 202, got %d: %s", response.Code, response.Body.String())
	}
	if refresher.id != "feed_1" || !strings.Contains(response.Body.String(), `"job":{"id":"job_1"`) {
		t.Fatalf("unexpected refresh response id=%q body=%s", refresher.id, response.Body.String())
	}
}

func TestFeedRefreshRouteErrors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
	}{
		{name: "missing", err: app.ErrNotFound, status: http.StatusNotFound},
		{name: "disabled", err: app.ErrConflict, status: http.StatusConflict},
		{name: "duplicate", err: feedrefresh.ErrRefreshAlreadyQueued, status: http.StatusConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service := app.NewService()
			t.Cleanup(func() { _ = service.Close() })
			router := NewRouter(service, WithFeedRefresher(&fakeFeedRefresher{err: tc.err}))

			response := serveJSON(router, http.MethodPost, "/api/v1/feeds/feed_1/refresh", "")
			if response.Code != tc.status {
				t.Fatalf("expected status %d, got %d: %s", tc.status, response.Code, response.Body.String())
			}
		})
	}
}

func TestFeedRefreshRouteUnavailable(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	router := NewRouter(service)

	response := serveJSON(router, http.MethodPost, "/api/v1/feeds/feed_1/refresh", "")
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", response.Code, response.Body.String())
	}
}

func responseID(t *testing.T, body string) string {
	t.Helper()
	const marker = `"id":"`
	index := strings.Index(body, marker)
	if index < 0 {
		t.Fatalf("response id missing from %s", body)
	}
	rest := body[index+len(marker):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		t.Fatalf("unterminated response id in %s", body)
	}
	return rest[:end]
}
