package web

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentcapability"
	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/conversation"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func agentExecutorWithCapabilityProfile(executor AgentExecutor, profile agentcapability.Profile) AgentExecutor {
	return agentexec.WithCapabilityProfile(executor, profile)
}

func (server *Server) agentCapabilityProfileForSession(
	ctx context.Context,
	missionID string,
	executorName string,
	sessionID string,
) (agentcapability.Profile, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return agentcapability.Resolve("", "")
	}
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return agentcapability.Profile{}, err
	}
	session, ok := conversation.AgentSessionByID(events, executorName, sessionID)
	if !ok {
		return agentcapability.Resolve("", "")
	}
	profile, err := agentcapability.Resolve(session.ProfileID, session.ProfileRevision)
	if err != nil {
		return agentcapability.Profile{}, fmt.Errorf(
			"%w: persisted agent capability profile is invalid: %v",
			producterror.ErrConflict,
			err,
		)
	}
	return profile, nil
}
