package reportrun

// MergeArtifactMembership preserves the strongest known role and ownership for
// a run/artifact membership without changing the table identity.
func MergeArtifactMembership(existing ArtifactMembership, incoming ArtifactMembership) (ArtifactMembership, bool) {
	merged := existing
	changed := false
	lineageChanged := false
	if artifactRoleRank(incoming.ArtifactRole) > artifactRoleRank(existing.ArtifactRole) {
		merged.ArtifactRole = incoming.ArtifactRole
		changed = true
		lineageChanged = true
	}
	if existing.Ownership != OwnershipCreated && incoming.Ownership == OwnershipCreated {
		merged.Ownership = OwnershipCreated
		changed = true
		lineageChanged = true
	}
	if lineageChanged {
		merged.AttemptEventID = incoming.AttemptEventID
		merged.SourceEventID = incoming.SourceEventID
		return merged, changed
	}
	if merged.AttemptEventID == "" && incoming.AttemptEventID != "" {
		merged.AttemptEventID = incoming.AttemptEventID
		changed = true
	}
	if merged.SourceEventID == "" && incoming.SourceEventID != "" {
		merged.SourceEventID = incoming.SourceEventID
		changed = true
	}
	return merged, changed
}

// IsReportEventType reports whether an event belongs to Plasma report lineage.
func IsReportEventType(eventType string) bool {
	return isReportEvent(eventType)
}
