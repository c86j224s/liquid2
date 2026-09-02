package reportilphase0

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
)

var readerFacingInternalTermPattern = regexp.MustCompile(`(?i)(experimental\s+il|narrative\s+contract|semantic\s+il|flow\s+attestation|frozen\s+source\s+catalog|source[_ -]?selection|pipeline\s+family|structured\s+output|schema\s+validation|\bil[_ -]?(?:검증|파이프라인|단계|경로)\b|내러티브\s*계약|시맨틱\s*il|소스\s*(?:선별|완독|커버리지)|검증\s*파이프라인)`)

var readerFacingProcessPattern = regexp.MustCompile(`(?i)(이\s*보고서(?:에서는|는)|이제\s+.{0,40}(?:살펴보|검토하|알아보|넘어가)|다음\s*(?:절|장|섹션)(?:에서는|은|으로)|(?:이|본|해당)\s*(?:절|장|섹션)(?:에서는|은)\s+.{0,40}(?:살펴보|검토하|알아보|설명하))`)

var reportMachinerySubjectPattern = regexp.MustCompile(`(?i)(\bil\b|intermediate\s+(?:language|representation)|narrative\s+contract|semantic\s+il|flow\s+attestation|structured\s+output|schema\s+validation|중간\s*(?:언어|표현)|내러티브\s*계약|시맨틱\s*il|il\s*(?:검증|파이프라인|단계|경로))`)

var readerFacingSourceMetadataPattern = regexp.MustCompile(`(?i)(\bsource_[0-9]{3,}\b|accepted-source:|ref\.accepted_source\.|generic\s+source\s+label|(?:accepted\s+)?source\s+[0-9]+|승인\s*소스\s*[0-9]+|file://|/Users/|/home/|/private/|/var/|/tmp/|(?m)(?:^|[\s"'=(])(?:[A-Za-z]:[\\/]))`)

var repairContextLocatorPattern = regexp.MustCompile(`(?i)((?:https?|wss?|file)://|/Users/|/home/|/private/|/var/|/tmp/|~/|[A-Za-z]:[\\/])`)

var readerFacingSourceMetadataURLContextPattern = regexp.MustCompile(`(?i)(?:(?:출처|자료|사료|근거|증거|source|reference|evidence)(?:\s*(?:URL|링크|주소|locator))?\s*[:：]?\s*(?:https?|wss?)://|(?:https?|wss?)://[^\s<>()]+.{0,40}(?:출처|자료|사료|근거|증거|source|reference|evidence))`)

var readerFacingSourceMetadataHomeContextPattern = regexp.MustCompile(`(?i)(?:(?:출처|자료|사료|근거|증거|source|reference|evidence)(?:\s*(?:경로|path|locator))?\s*[:：]?\s*~/|~/(?:[^\s]+).{0,40}(?:출처|자료|사료|근거|증거|source|reference|evidence))`)

var readerFacingAuditVoicePattern = regexp.MustCompile(`(?i)((?:자료|사료|기록|근거|증거).{0,35}(?:보여|말해|설명|확인|입증|확립|단정|확정|복원)|(?:확인|입증|단정|확정|복원).{0,25}(?:할\s*수\s*없|되지\s*않|아니다)|(?:the\s+)?(?:source|record|evidence|material)s?.{0,35}(?:show|say|explain|confirm|establish|prove|support|reconstruct)|(?:cannot|does\s+not|do\s+not).{0,25}(?:confirm|establish|prove|support|reconstruct))`)

var readerFacingLowValueSIExplanationPattern = regexp.MustCompile(`(?i)(해발\s*\d+(?:[.,]\d+)?\s*(?:미터|m)\s*[,，]?\s*(?:곧|즉).{0,60}(?:바닷물|해수면|평균\s*해수면)|(?:elevation|altitude).{0,25}\d+(?:[.,]\d+)?\s*(?:meters?|m).{0,60}(?:means|measured|sea\s+level))`)

var repairContextCredentialPattern = regexp.MustCompile(`(?i)(password\s*[:=]|passwd\s*[:=]|-----BEGIN(?: [A-Z0-9]+)* ` + "PRIVATE" + ` KEY-----|-----BEGIN PGP ` + "PRIVATE" + ` KEY BLOCK-----)`)

func mixedTargetAndJapaneseScript(value string) bool {
	var token []rune
	check := func() bool {
		hangul, japanese := false, false
		for _, current := range token {
			hangul = hangul || unicode.Is(unicode.Hangul, current)
			japanese = japanese || unicode.In(current, unicode.Hiragana, unicode.Katakana)
		}
		return hangul && japanese
	}
	for _, current := range []rune(value) {
		if unicode.IsLetter(current) || unicode.IsMark(current) {
			token = append(token, current)
			continue
		}
		if check() {
			return true
		}
		token = token[:0]
	}
	return check()
}

func malformedKoreanReaderName(value string) bool {
	var token []rune
	check := func() bool {
		hangul, japanese := false, false
		hanCount := 0
		for _, current := range token {
			hangul = hangul || unicode.Is(unicode.Hangul, current)
			if unicode.Is(unicode.Han, current) {
				hanCount++
			}
			japanese = japanese || unicode.In(current, unicode.Hiragana, unicode.Katakana)
		}
		return japanese && (hangul || hanCount == 1)
	}
	for _, current := range []rune(value) {
		if unicode.IsLetter(current) || unicode.IsMark(current) {
			token = append(token, current)
			continue
		}
		if check() {
			return true
		}
		token = token[:0]
	}
	return check()
}

func validateAuthorFacingDocument(document Document, missionObjective string) error {
	machineryIsSubject := reportMachinerySubjectPattern.MatchString(missionObjective)
	if !machineryIsSubject && readerFacingInternalTermPattern.MatchString(document.Title) {
		return withValidationCode(
			reportexecution.ProviderValidationCodeReaderInternalMachinery,
			fmt.Errorf("reader-facing title exposes internal report machinery"),
		)
	}
	for _, block := range document.Blocks {
		for _, value := range authoredBlockText(block) {
			if !machineryIsSubject && readerFacingInternalTermPattern.MatchString(value) {
				return withValidationCode(
					reportexecution.ProviderValidationCodeReaderInternalMachinery,
					fmt.Errorf("reader-facing content exposes internal report machinery"),
				)
			}
		}
	}
	return nil
}

func validateReaderFacingDocument(document Document, missionObjective string) error {
	if err := validateAuthorFacingDocument(document, missionObjective); err != nil {
		return err
	}
	if err := validateReaderFacingDocumentExceptMixedScript(document); err != nil {
		return err
	}
	for _, value := range readerFacingValues(document) {
		if document.Language == "ko" && malformedKoreanReaderName(value) {
			return withValidationCode(
				reportexecution.ProviderValidationCodeReaderFacingContent,
				fmt.Errorf("Korean reader-facing content contains a malformed Japanese-script name"),
			)
		}
		if mixedTargetAndJapaneseScript(value) {
			return withValidationCode(
				reportexecution.ProviderValidationCodeReaderFacingContent,
				fmt.Errorf("reader-facing content contains a mixed-script lexical form"),
			)
		}
	}
	return nil
}

func readerFacingValues(document Document) []string {
	values := []string{document.Title}
	for _, block := range document.Blocks {
		values = append(values, authoredBlockText(block)...)
	}
	return values
}

func validateReaderFacingDocumentExceptMixedScript(document Document) error {
	values := readerFacingValues(document)
	contentPassages := 0
	opening := ""
	for _, block := range document.Blocks {
		blockValues := authoredBlockText(block)
		if block.Kind != "section" {
			contentPassages += len(blockValues)
			if opening == "" && len(blockValues) > 0 {
				opening = blockValues[0]
			}
		}
	}
	if readerFacingAuditVoicePattern.MatchString(opening) {
		return withValidationCode(
			reportexecution.ProviderValidationCodeReaderOpening,
			fmt.Errorf("reader-facing opening begins with source-audit narration"),
		)
	}
	auditVoicePassages := 0
	for _, value := range values {
		if readerFacingProcessPattern.MatchString(value) {
			return withValidationCode(
				reportexecution.ProviderValidationCodeReaderProcess,
				fmt.Errorf("reader-facing content narrates the report construction"),
			)
		}
		if readerFacingSourceMetadataPattern.MatchString(value) ||
			readerFacingSourceMetadataURLContextPattern.MatchString(value) ||
			readerFacingSourceMetadataHomeContextPattern.MatchString(value) {
			return withValidationCode(
				reportexecution.ProviderValidationCodeReaderMetadata,
				fmt.Errorf("reader-facing content exposes source metadata"),
			)
		}
		if readerFacingLowValueSIExplanationPattern.MatchString(value) {
			return withValidationCode(
				reportexecution.ProviderValidationCodeReaderOrdinarySI,
				fmt.Errorf("reader-facing content explains an ordinary SI measurement"),
			)
		}
		if readerFacingAuditVoicePattern.MatchString(value) {
			auditVoicePassages++
		}
	}
	if auditVoicePassages > 2 && auditVoicePassages*4 >= contentPassages {
		return withValidationCode(
			reportexecution.ProviderValidationCodeReaderAuditVoice,
			fmt.Errorf("reader-facing content repeats source-audit narration"),
		)
	}
	return nil
}

func authoredBlockText(block Block) []string {
	values := []string{block.Title, block.Prose, block.Code}
	values = append(values, block.Items...)
	if block.Equation != nil {
		values = append(values, block.Equation.Expression)
	}
	if block.Table != nil {
		values = append(values, block.Table.Caption)
		values = append(values, block.Table.Columns...)
		for _, row := range block.Table.Rows {
			values = append(values, row...)
		}
	}
	result := values[:0]
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, value)
		}
	}
	return result
}
