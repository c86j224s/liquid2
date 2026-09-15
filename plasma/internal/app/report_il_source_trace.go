package app

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/reportilsource"
)

// VerifyReportILSourceRead delegates trace verification while retaining event access.
func (s *Service) VerifyReportILSourceRead(ctx context.Context, missionID, toolSessionID, expectedStage string, catalog reportilcontract.SourceCatalog) (reportilcontract.SourceReadReceipt, error) {
	return reportilsource.VerifySourceRead(ctx, s.ListEvents, missionID, toolSessionID, expectedStage, catalog)
}
