package agentexec

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/agentcapability"
)

type capabilityProfileExecutor struct {
	delegate AgentExecutor
	profile  agentcapability.Profile
}

func (executor capabilityProfileExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	req.CapabilityProfile = executor.profile.ID
	req.ProfileRevision = executor.profile.Revision
	return executor.delegate.Run(ctx, req)
}

type capabilityProfileForkExecutor struct {
	capabilityProfileExecutor
	AgentSessionForker
}

type capabilityProfileForkReadyExecutor struct {
	capabilityProfileExecutor
	AgentSessionForkReadiness
}

type capabilityProfileForkAndReadyExecutor struct {
	capabilityProfileExecutor
	AgentSessionForker
	AgentSessionForkReadiness
}

// WithCapabilityProfile binds every request through an executor to one validated
// provider-session profile while preserving optional session-fork capabilities.
func WithCapabilityProfile(executor AgentExecutor, profile agentcapability.Profile) AgentExecutor {
	profiled := capabilityProfileExecutor{delegate: executor, profile: profile}
	forker, canFork := executor.(AgentSessionForker)
	readiness, canCheckFork := executor.(AgentSessionForkReadiness)
	switch {
	case canFork && canCheckFork:
		return capabilityProfileForkAndReadyExecutor{
			capabilityProfileExecutor: profiled,
			AgentSessionForker:        forker,
			AgentSessionForkReadiness: readiness,
		}
	case canFork:
		return capabilityProfileForkExecutor{
			capabilityProfileExecutor: profiled,
			AgentSessionForker:        forker,
		}
	case canCheckFork:
		return capabilityProfileForkReadyExecutor{
			capabilityProfileExecutor: profiled,
			AgentSessionForkReadiness: readiness,
		}
	default:
		return profiled
	}
}
