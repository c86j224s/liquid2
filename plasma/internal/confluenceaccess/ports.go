package confluenceaccess

import "context"

// ConnectionStore is the consumer-side persistence port for credentials and sites.
type ConnectionStore interface {
	UpsertConfluenceConnection(context.Context, Connection) error
	GetConfluenceConnection(context.Context, string) (Connection, error)
	ListConfluenceConnections(context.Context) ([]Connection, error)
	DeleteConfluenceConnection(context.Context, string) error
}

// SiteLister reads sites available through a concrete connector adapter.
type SiteLister interface {
	ListConfluenceSites(context.Context) (SiteListResult, error)
}
