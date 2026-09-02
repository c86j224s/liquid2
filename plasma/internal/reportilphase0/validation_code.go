package reportilphase0

import (
	"errors"
	"regexp"

	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
)

var (
	providerValidationTermAliasPattern    = regexp.MustCompile(`^term_[0-9]{2}$`)
	providerValidationPacketAliasPattern  = regexp.MustCompile(`^evidence_[0-9]{3}$`)
	providerValidationSectionAliasPattern = regexp.MustCompile(`^section_[0-9]{2}$`)
	providerValidationBlockAliasPattern   = regexp.MustCompile(`^block_[0-9]{2}$`)
	providerValidationSourceKeyPattern    = regexp.MustCompile(`^source_[0-9]{3}$`)
)

type codedValidationError struct {
	code          reportexecution.ProviderValidationCode
	termAlias     string
	packetAliases []string
	cause         error
}

func (err *codedValidationError) Error() string { return err.cause.Error() }
func (err *codedValidationError) Unwrap() error { return err.cause }

func withValidationCode(code reportexecution.ProviderValidationCode, cause error) error {
	return withValidationCodeAndTerm(code, "", cause)
}

func withValidationCodeAndTerm(code reportexecution.ProviderValidationCode, termAlias string, cause error) error {
	if cause == nil {
		return nil
	}
	var coded *codedValidationError
	if errors.As(cause, &coded) {
		if coded.termAlias != "" || termAlias == "" {
			return cause
		}
		return &codedValidationError{
			code: coded.code, termAlias: termAlias,
			packetAliases: append([]string(nil), coded.packetAliases...), cause: cause,
		}
	}
	return &codedValidationError{code: code, termAlias: termAlias, cause: cause}
}

func withValidationCodeAndPackets(
	code reportexecution.ProviderValidationCode,
	packetAliases []string,
	cause error,
) error {
	if cause == nil {
		return nil
	}
	aliases := make([]string, 0, len(packetAliases))
	seen := map[string]bool{}
	for _, alias := range packetAliases {
		if providerValidationPacketAliasPattern.MatchString(alias) && !seen[alias] {
			aliases = append(aliases, alias)
			seen[alias] = true
		}
	}
	if len(aliases) == 0 {
		return withValidationCode(code, cause)
	}
	var coded *codedValidationError
	if errors.As(cause, &coded) {
		if len(coded.packetAliases) > 0 {
			return cause
		}
		return &codedValidationError{
			code: coded.code, termAlias: coded.termAlias,
			packetAliases: aliases, cause: cause,
		}
	}
	return &codedValidationError{code: code, packetAliases: aliases, cause: cause}
}

func providerValidationCode(cause error) reportexecution.ProviderValidationCode {
	var coded *codedValidationError
	if errors.As(cause, &coded) && coded.code.Valid() {
		return coded.code
	}
	return reportexecution.ProviderValidationCodeSemanticContract
}

func providerValidationTermAlias(cause error) string {
	var coded *codedValidationError
	if errors.As(cause, &coded) && providerValidationTermAliasPattern.MatchString(coded.termAlias) {
		return coded.termAlias
	}
	return ""
}

func providerValidationPacketAliases(cause error) []string {
	var coded *codedValidationError
	if !errors.As(cause, &coded) {
		return nil
	}
	aliases := make([]string, 0, len(coded.packetAliases))
	for _, alias := range coded.packetAliases {
		if providerValidationPacketAliasPattern.MatchString(alias) {
			aliases = append(aliases, alias)
		}
	}
	return aliases
}
