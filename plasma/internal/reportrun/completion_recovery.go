package reportrun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
)

func RecoverMission(ctx context.Context, store ReportCompletionStore, missionID string) (int, error) {
	events, err := store.ListEvents(ctx, missionID)
	if err != nil {
		return 0, err
	}
	registration, err := BuildRegistration(reportRunEvents(events), RegistrationBackfilled, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	changed := 0
	var errs []error
	seen := map[string]bool{}
	for _, run := range registration.Runs {
		if seen[run.RunID] || run.LifecycleState == LifecycleAmbiguous || run.FinalArtifactID == "" {
			continue
		}
		seen[run.RunID] = true
		completionID := completionEventID(run.RunID)
		existing, exists := eventByID(events, completionID)
		canonicalID := ""
		if exists && existing.EventType == ReportRunCompletedEventType {
			var payload struct {
				CanonicalEventID string `json:"canonical_event_id"`
			}
			if json.Unmarshal(existing.Payload, &payload) == nil {
				canonicalID = strings.TrimSpace(payload.CanonicalEventID)
			}
			if canonicalID == "" {
				errs = append(errs, fmt.Errorf("recover report completion for %s: invalid existing report completion", run.RunID))
				continue
			}
		} else {
			var finals []ledger.Event
			for _, member := range registration.Events {
				if member.RunID != run.RunID || member.EventRole != "final" {
					continue
				}
				if event, found := eventByID(events, member.EventID); found && event.EventType == "report.artifact.created" {
					finals = append(finals, event)
				}
			}
			sort.Slice(finals, func(i, j int) bool {
				if finals[i].Sequence != finals[j].Sequence {
					return finals[i].Sequence < finals[j].Sequence
				}
				if !finals[i].CreatedAt.Equal(finals[j].CreatedAt) {
					return finals[i].CreatedAt.Before(finals[j].CreatedAt)
				}
				return finals[i].EventID < finals[j].EventID
			})
			if len(finals) > 0 {
				canonicalID = finals[len(finals)-1].EventID
			}
		}
		if canonicalID == "" {
			continue
		}
		_, created, runErr := completeReportRun(ctx, store, ReportCompletionRequest{MissionID: missionID, CanonicalEventID: canonicalID})
		if runErr != nil {
			errs = append(errs, fmt.Errorf("recover report completion for %s: %w", run.RunID, runErr))
			continue
		}
		if created {
			changed++
		}
	}
	return changed, errors.Join(errs...)
}

func RecoverAll(ctx context.Context, store ReportCompletionMissionStore) (int, error) {
	missions, err := store.ListMissionsWithState(ctx, mission.ListRequest{IncludeArchived: true})
	if err != nil {
		return 0, err
	}
	changed := 0
	var errs []error
	for _, current := range missions {
		count, runErr := RecoverMission(ctx, store, current.MissionID)
		changed += count
		if runErr != nil {
			errs = append(errs, fmt.Errorf("recover report completion for %s: %w", current.MissionID, runErr))
		}
	}
	return changed, errors.Join(errs...)
}
