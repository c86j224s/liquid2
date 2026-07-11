package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func runMissions(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: plasma missions <create|list|show> [options]")
		return 2
	}
	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("missions create", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dbPath := fs.String("db", "", "Plasma SQLite database path")
		title := fs.String("title", "", "mission title")
		objective := fs.String("objective", "", "mission objective")
		jsonOut := fs.Bool("json", false, "write JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		svc, closeStore, displayDB, err := openCLIService(ctx, *dbPath)
		if err != nil {
			fmt.Fprintf(stderr, "open storage: %v\n", err)
			return 1
		}
		defer closeStore()
		missionTitle := strings.TrimSpace(*title)
		if missionTitle == "" && fs.NArg() > 0 {
			missionTitle = strings.TrimSpace(strings.Join(fs.Args(), " "))
		}
		if missionTitle == "" {
			fmt.Fprintln(stderr, "title is required")
			return 2
		}
		missionID := cliNewID("mis")
		mission, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: missionTitle})
		if err != nil {
			fmt.Fprintf(stderr, "create mission: %v\n", err)
			return 1
		}
		missionObjective := strings.TrimSpace(*objective)
		if missionObjective == "" {
			missionObjective = missionTitle
		}
		if _, err := svc.AppendEvent(ctx, app.BuildMissionCreatedAppendRequest(app.MissionCreatedEventRequest{
			EventID:   cliNewID("evt"),
			MissionID: missionID,
			Title:     missionTitle,
			Objective: missionObjective,
			Scope:     app.MissionScope{Included: []string{}, Excluded: []string{}},
			Producer:  app.Producer{Type: "user", ID: "plasma-cli"},
		})); err != nil {
			fmt.Fprintf(stderr, "append mission.created: %v\n", err)
			return 1
		}
		projection, err := svc.RebuildProjection(ctx, missionID)
		if err != nil {
			fmt.Fprintf(stderr, "rebuild projection: %v\n", err)
			return 1
		}
		if *jsonOut {
			writeCLIJSON(stdout, map[string]any{"mission": mission, "projection": projection, "db": displayDB})
			return 0
		}
		fmt.Fprintf(stdout, "created mission %s title=%q db=%s\n", mission.MissionID, mission.Title, displayDB)
		return 0
	case "list":
		fs := flag.NewFlagSet("missions list", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dbPath := fs.String("db", "", "Plasma SQLite database path")
		jsonOut := fs.Bool("json", false, "write JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		svc, closeStore, _, err := openCLIService(ctx, *dbPath)
		if err != nil {
			fmt.Fprintf(stderr, "open storage: %v\n", err)
			return 1
		}
		defer closeStore()
		missions, err := svc.ListMissions(ctx)
		if err != nil {
			fmt.Fprintf(stderr, "list missions: %v\n", err)
			return 1
		}
		if *jsonOut {
			writeCLIJSON(stdout, map[string]any{"missions": missions})
			return 0
		}
		for _, mission := range missions {
			fmt.Fprintf(stdout, "%s\t%s\n", mission.MissionID, mission.Title)
		}
		return 0
	case "show":
		fs := flag.NewFlagSet("missions show", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dbPath := fs.String("db", "", "Plasma SQLite database path")
		jsonOut := fs.Bool("json", false, "write JSON")
		positionals, parseArgs := leadingPositionals(args[1:], 1)
		if err := fs.Parse(parseArgs); err != nil {
			return 2
		}
		positionals = append(positionals, fs.Args()...)
		if len(positionals) != 1 {
			fmt.Fprintln(stderr, "usage: plasma missions show <mission_id> [options]")
			return 2
		}
		svc, closeStore, _, err := openCLIService(ctx, *dbPath)
		if err != nil {
			fmt.Fprintf(stderr, "open storage: %v\n", err)
			return 1
		}
		defer closeStore()
		missionID := positionals[0]
		projection, err := svc.GetProjection(ctx, missionID)
		if err != nil {
			fmt.Fprintf(stderr, "show mission: %v\n", err)
			return 1
		}
		workflowRuns, _ := svc.ListWorkflowRuns(ctx, missionID)
		if *jsonOut {
			writeCLIJSON(stdout, map[string]any{"projection": projection, "workflow_runs": workflowRuns})
			return 0
		}
		fmt.Fprintf(stdout, "%s\t%s\tworkflow_runs=%d\n", projection.MissionID, projection.Title, len(workflowRuns))
		return 0
	default:
		fmt.Fprintf(stderr, "unknown missions command %q\n", args[0])
		return 2
	}
}
