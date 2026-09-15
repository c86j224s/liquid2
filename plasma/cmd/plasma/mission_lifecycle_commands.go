package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"io"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

func runMissionLifecycleCommand(ctx context.Context, args []string, stdout, stderr io.Writer, targetState string) int {
	command := "archive"
	if targetState == mission.LifecycleActive {
		command = "restore"
	}
	fs := flag.NewFlagSet("missions "+command, flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	jsonOut := fs.Bool("json", false, "write JSON")
	reason := fs.String("reason", "", "lifecycle change reason")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintf(stderr, "usage: plasma missions %s <mission_id> [options]\n", command)
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	req := mission.MissionLifecycleChangeRequest{
		EventID:   cliNewID("evt"),
		MissionID: positionals[0],
		Producer:  ledger.Producer{Type: "user", ID: "plasma-cli"},
		Reason:    *reason,
	}
	var result mission.MissionLifecycleChangeResult
	if targetState == mission.LifecycleArchived {
		result, err = svc.ArchiveMission(ctx, req)
	} else {
		result, err = svc.RestoreMission(ctx, req)
	}
	if err != nil {
		fmt.Fprintf(stderr, "%s mission: %v\n", command, err)
		return 1
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	status := command + "d"
	if command == "restore" {
		status = "restored"
	}
	if result.Idempotent {
		fmt.Fprintf(stdout, "mission %s already %s\n", result.Projection.MissionID, result.Projection.LifecycleState)
		return 0
	}
	fmt.Fprintf(stdout, "%s mission %s\n", status, result.Projection.MissionID)
	return 0
}
