package reportexecution

import (
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func TestSelectSessionPolicyAutoFreshForPlannedAndLongForm(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode string
	}{
		{name: "planned", mode: ModePlanned},
		{name: "long form", mode: ModeLongForm},
	} {
		t.Run(tc.name, func(t *testing.T) {
			policy, selection, err := SelectSessionPolicy(SessionPolicySelectionInput{
				ReportMode:                  tc.mode,
				CanForkSession:              false,
				HasPreReportResearchSession: false,
				ForkReady:                   false,
			})
			if err != nil {
				t.Fatal(err)
			}
			if policy != SessionPolicyFreshSession || selection != SessionPolicySelectionAutoFreshSession {
				t.Fatalf("policy=%q selection=%q, want fresh auto", policy, selection)
			}
		})
	}
}

func TestSelectSessionPolicyKeepsOneTakeSameSession(t *testing.T) {
	policy, selection, err := SelectSessionPolicy(SessionPolicySelectionInput{
		ReportMode:                  ModeOneTake,
		CanForkSession:              true,
		HasPreReportResearchSession: true,
		ForkReady:                   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if policy != SessionPolicySameSession || selection != SessionPolicySelectionAutoSameSessionOneTake {
		t.Fatalf("policy=%q selection=%q, want one-take same-session", policy, selection)
	}
}

func TestSelectSessionPolicyRejectsExplicitFreshSession(t *testing.T) {
	_, _, err := SelectSessionPolicy(SessionPolicySelectionInput{
		RequestedPolicy:             SessionPolicyFreshSession,
		ReportMode:                  ModePlanned,
		CanForkSession:              true,
		HasPreReportResearchSession: true,
		ForkReady:                   true,
	})
	if !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("err=%v, want invalid input", err)
	}
}

func TestSessionPolicyCompatibilityNormalizationAndValidation(t *testing.T) {
	if policy, err := NormalizeSessionPolicy(SessionPolicyFreshSession); err != nil || policy != SessionPolicyFreshSession {
		t.Fatalf("fresh canonical normalization failed: policy=%q err=%v", policy, err)
	}
	for _, alias := range []string{"fresh", "fresh-session"} {
		if _, err := NormalizeSessionPolicy(alias); !errors.Is(err, producterror.ErrInvalidInput) {
			t.Fatalf("NormalizeSessionPolicy(%q) err=%v, want invalid input", alias, err)
		}
	}
	if err := ValidateSessionPolicy(SessionPolicyFreshSession, ModePlanned, false, false, false); err != nil {
		t.Fatalf("fresh planned validation must not require fork readiness: %v", err)
	}
	if err := ValidateSessionPolicy(SessionPolicyFreshSession, ModeOneTake, false, false, false); !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("fresh one-take validation err=%v, want invalid input", err)
	}
}

func TestSelectSessionPolicyKeepsExplicitLegacyOverrides(t *testing.T) {
	policy, selection, err := SelectSessionPolicy(SessionPolicySelectionInput{
		RequestedPolicy:             SessionPolicySameSession,
		ReportMode:                  ModePlanned,
		CanForkSession:              false,
		HasPreReportResearchSession: false,
		ForkReady:                   false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if policy != SessionPolicySameSession || selection != SessionPolicySelectionExplicitSameSession {
		t.Fatalf("policy=%q selection=%q, want explicit same-session", policy, selection)
	}

	policy, selection, err = SelectSessionPolicy(SessionPolicySelectionInput{
		RequestedPolicy:             SessionPolicyIsolatedFork,
		ReportMode:                  ModePlanned,
		CanForkSession:              true,
		HasPreReportResearchSession: true,
		ForkReady:                   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if policy != SessionPolicyIsolatedFork || selection != SessionPolicySelectionExplicitIsolatedFork {
		t.Fatalf("policy=%q selection=%q, want explicit isolated fork", policy, selection)
	}
}
