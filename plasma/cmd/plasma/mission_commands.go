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
		fmt.Fprintln(stderr, "usage: plasma missions <create|list|show|update|archive|restore> [options]")
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
		includeArchived := fs.Bool("include-archived", false, "include archived missions")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		svc, closeStore, _, err := openCLIService(ctx, *dbPath)
		if err != nil {
			fmt.Fprintf(stderr, "open storage: %v\n", err)
			return 1
		}
		defer closeStore()
		missions, err := svc.ListMissionsWithState(ctx, app.ListMissionsRequest{IncludeArchived: *includeArchived})
		if err != nil {
			fmt.Fprintf(stderr, "list missions: %v\n", err)
			return 1
		}
		if *jsonOut {
			writeCLIJSON(stdout, map[string]any{"missions": missions})
			return 0
		}
		for _, mission := range missions {
			stateSuffix := ""
			if mission.LifecycleState == app.MissionLifecycleArchived {
				stateSuffix = "\tarchived"
			}
			fmt.Fprintf(stdout, "%s\t%s%s\n", mission.MissionID, mission.Title, stateSuffix)
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
	case "update":
		return runMissionUpdate(ctx, args[1:], stdout, stderr)
	case "archive":
		return runMissionLifecycleCommand(ctx, args[1:], stdout, stderr, app.MissionLifecycleArchived)
	case "restore":
		return runMissionLifecycleCommand(ctx, args[1:], stdout, stderr, app.MissionLifecycleActive)
	default:
		fmt.Fprintf(stderr, "unknown missions command %q\n", args[0])
		return 2
	}
}

type presentString struct {
	value string
	set   bool
}

func (v *presentString) String() string         { return v.value }
func (v *presentString) Set(value string) error { v.value, v.set = value, true; return nil }

type stringListFlag struct {
	values []string
	set    bool
}

func (v *stringListFlag) String() string { return strings.Join(v.values, ",") }
func (v *stringListFlag) Set(value string) error {
	v.values, v.set = append(v.values, strings.TrimSpace(value)), true
	return nil
}

func runMissionUpdate(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("missions update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var title, objective presentString
	var included, excluded stringListFlag
	fs.Var(&title, "title", "mission title")
	fs.Var(&objective, "objective", "mission objective")
	fs.Var(&included, "scope-included", "included scope item (repeatable)")
	fs.Var(&excluded, "scope-excluded", "excluded scope item (repeatable)")
	clearScope := fs.Bool("clear-scope", false, "clear included and excluded scope")
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma missions update <mission_id> [options]")
		return 2
	}
	if *clearScope && (included.set || excluded.set) {
		fmt.Fprintln(stderr, "--clear-scope cannot be combined with scope flags")
		return 2
	}
	if !title.set && !objective.set && !included.set && !excluded.set && !*clearScope {
		fmt.Fprintln(stderr, "at least one update flag is required")
		return 2
	}
	var titlePtr, objectivePtr *string
	if title.set {
		titlePtr = &title.value
	}
	if objective.set {
		objectivePtr = &objective.value
	}
	var scope *app.MissionScope
	if *clearScope || included.set || excluded.set {
		scope = &app.MissionScope{Included: included.values, Excluded: excluded.values}
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	result, err := svc.UpdateMissionMetadata(ctx, app.UpdateMissionMetadataRequest{EventID: cliNewID("evt"), MissionID: positionals[0], Producer: app.Producer{Type: "user", ID: "plasma-cli"}, Title: titlePtr, Objective: objectivePtr, Scope: scope})
	if err != nil {
		fmt.Fprintf(stderr, "update mission: %v\n", err)
		return 1
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	fmt.Fprintf(stdout, "updated mission %s title=%q\n", result.Projection.MissionID, result.Projection.Title)
	return 0
}
