package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func writeWorkflowView(w io.Writer, view app.WorkflowRunView, jsonOut bool) {
	if jsonOut {
		writeCLIJSON(w, map[string]any{"workflow_run": view})
		return
	}
	fmt.Fprintf(w, "%s\tmission=%s\tstatus=%s\tlatest=%s\treason=%s\n",
		view.WorkflowRunID, view.MissionID, view.Status, view.LatestEventID, view.StopReason)
}

func writeCLIJSON(w io.Writer, value any) {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(value)
}

func leadingPositionals(args []string, max int) ([]string, []string) {
	var positionals []string
	index := 0
	for index < len(args) && len(positionals) < max {
		if strings.HasPrefix(args[index], "-") {
			break
		}
		positionals = append(positionals, args[index])
		index++
	}
	remaining := append([]string(nil), args[index:]...)
	return positionals, remaining
}

func cliJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func cliNewID(prefix string) string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s_%s_%s", strings.TrimSuffix(prefix, "_"), time.Now().UTC().Format("20060102150405"), hex.EncodeToString(b[:]))
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
