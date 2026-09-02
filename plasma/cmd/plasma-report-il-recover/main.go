package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reportilphase0"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func main() {
	var databasePath string
	var missionID string
	var pendingID string
	var write bool
	var readerSession string
	flag.StringVar(&databasePath, "db", "", "Plasma SQLite database")
	flag.StringVar(&missionID, "mission", "", "mission ID")
	flag.StringVar(&pendingID, "pending", "", "failed report pending event ID")
	flag.BoolVar(&write, "write", false, "append the verified checkpoint")
	flag.StringVar(&readerSession, "reader-session", "", "promote a finalized reader session")
	flag.Parse()
	if strings.TrimSpace(databasePath) == "" || !strings.HasPrefix(missionID, "mis_") || !strings.HasPrefix(pendingID, "evt_") {
		fmt.Fprintln(os.Stderr, "usage: plasma-report-il-recover -db PATH -mission mis_... -pending evt_... [-write]")
		os.Exit(2)
	}
	ctx := context.Background()
	store, err := sqlite.Open(ctx, databasePath)
	if err != nil {
		fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	events, err := service.ListEvents(ctx, missionID)
	if err != nil {
		fatal(err)
	}
	resume, err := reportilphase0.RecoverLegacyCheckpoint(
		ctx, missionID, pendingID, events,
		service.ReportILSourceReader(), service,
	)
	if err != nil {
		fatal(err)
	}
	if strings.TrimSpace(readerSession) != "" {
		resume, err = reportilphase0.PromoteReaderArtifact(ctx, "", *resume, strings.TrimSpace(readerSession), service)
		if err != nil {
			fatal(err)
		}
	}
	if write {
		if err := service.AppendReportILCheckpoint(ctx, missionID, resume.ProductCheckpoint); err != nil {
			fatal(err)
		}
		if err := service.RebuildReportRuns(ctx, missionID); err != nil {
			fatal(err)
		}
		fmt.Printf("recovered %s %s %s\n", missionID, pendingID, resume.ArtifactID)
		return
	}
	fmt.Printf("verified %s %s %s\n", missionID, pendingID, resume.ArtifactID)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
