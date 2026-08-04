package research

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func validateProposalInputs(proposalID, eventID, proposalEventID string) error {
	if err := validateID("prp_", proposalID); err != nil {
		return err
	}
	if err := validateID("evt_", eventID); err != nil {
		return err
	}
	return validateID("evt_", proposalEventID)
}

func validateSnapshotRefs(refs []app.SnapshotRef) error {
	for _, ref := range refs {
		if err := validateID("src_", ref.SnapshotID); err != nil {
			return err
		}
		if err := validateID("art_", ref.ArtifactID); err != nil {
			return err
		}
		if len(ref.Locator) > 0 && !json.Valid(ref.Locator) {
			return fmt.Errorf("%w: evidence locator must be valid JSON", app.ErrInvalidInput)
		}
	}
	return nil
}

func validateIDList(prefix string, ids []string) error {
	seen := map[string]struct{}{}
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if err := validateID(prefix, trimmed); err != nil {
			return err
		}
		if _, ok := seen[trimmed]; ok {
			return fmt.Errorf("%w: duplicate id", app.ErrInvalidInput)
		}
		seen[trimmed] = struct{}{}
	}
	return nil
}

func validateConfidence(confidence app.Confidence) error {
	level := strings.TrimSpace(confidence.Level)
	if level == "" {
		return nil
	}
	if !toolConfidence[level] {
		return fmt.Errorf("%w: unsupported confidence level", app.ErrInvalidInput)
	}
	return nil
}

func validateObjectRef(ref app.ObjectRef) error {
	switch strings.TrimSpace(ref.ObjectKind) {
	case app.EvidenceRecordObjectKind:
		return validateID("evd_", ref.ObjectID)
	case app.ClaimRecordObjectKind:
		return validateID("clm_", ref.ObjectID)
	case app.QuestionRecordObjectKind:
		return validateID("qst_", ref.ObjectID)
	case app.OptionRecordObjectKind:
		return validateID("opt_", ref.ObjectID)
	default:
		return fmt.Errorf("%w: unsupported proposal object kind", app.ErrInvalidInput)
	}
}
