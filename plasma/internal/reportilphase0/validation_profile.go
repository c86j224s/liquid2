package reportilphase0

import "strings"

const (
	ValidationProfileUnverified  = "unverified"
	ValidationProfileExploratory = "exploratory"
	ValidationProfileStrict      = "strict"
)

type validationStagePlan struct {
	EditorialMemory bool
	PublicationRead bool
	Continuity      bool
}

func normalizeValidationProfile(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case ValidationProfileUnverified:
		return ValidationProfileUnverified
	case ValidationProfileExploratory:
		return ValidationProfileExploratory
	default:
		return ValidationProfileStrict
	}
}

func validationPlan(profile string) validationStagePlan {
	switch normalizeValidationProfile(profile) {
	case ValidationProfileUnverified:
		return validationStagePlan{}
	case ValidationProfileExploratory:
		return validationStagePlan{EditorialMemory: true, PublicationRead: true}
	default:
		return validationStagePlan{EditorialMemory: true, PublicationRead: true, Continuity: true}
	}
}
