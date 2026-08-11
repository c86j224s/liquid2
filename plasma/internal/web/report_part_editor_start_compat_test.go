package web

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partedit"
)

func (server *Server) loadCurrentPartEditStart(ctx context.Context, req reportPartEditorRequest, expectedProviderSessionID string) (reporting.PartEditBinding, bool, error) {
	return (partedit.Runner{Service: server.service}).CurrentStart(ctx, partEditInput(req, false, ""), expectedProviderSessionID)
}

func (server *Server) currentPartEditStart(ctx context.Context, req reportPartEditorRequest, expectedProviderSessionID string) (reporting.PartEditBinding, bool, error) {
	return server.loadCurrentPartEditStart(ctx, req, expectedProviderSessionID)
}

func (server *Server) partEditStartContract(ctx context.Context, req reportPartEditorRequest, expectedProviderSessionID string) (reporting.PartEditStartContract, error) {
	return (partedit.Runner{Service: server.service}).StartContract(ctx, partEditInput(req, false, ""), expectedProviderSessionID)
}
