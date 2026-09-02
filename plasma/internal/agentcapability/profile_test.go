package agentcapability

import "testing"

func TestResolveDefaultsToLegacyProfile(t *testing.T) {
	profile, err := Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if profile.ID != ProfileLegacyV1 || profile.Revision != RevisionV1 {
		t.Fatalf("unexpected default profile: %#v", profile)
	}
}

func TestResolveRejectsUnknownProfileAndRevision(t *testing.T) {
	for _, test := range []struct {
		name     string
		id       ProfileID
		revision string
	}{
		{name: "profile", id: "unknown.v1", revision: RevisionV1},
		{name: "revision", id: ProfileResearchV1, revision: "2"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Resolve(test.id, test.revision); err == nil {
				t.Fatal("expected invalid profile to be rejected")
			}
		})
	}
}
