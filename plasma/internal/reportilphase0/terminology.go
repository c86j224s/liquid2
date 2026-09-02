package reportilphase0

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

const maxAuthorTerminologyTerms = 96

var terminologyCategories = map[string]bool{
	"proper_name":          true,
	"official_designation": true,
	"specialist_term":      true,
	"foreign_unit":         true,
}

var terminologyHandling = map[string]bool{
	"preserve":             true,
	"preserve_and_explain": true,
	"convert_and_explain":  true,
}

var sourceScriptTermPattern = regexp.MustCompile(`[\p{Han}\p{Hiragana}\p{Katakana}]+`)

type authorTermDraft struct {
	Category      string  `json:"category"`
	SourceForm    string  `json:"source_form"`
	SourceReading *string `json:"source_reading"`
	ReaderForm    string  `json:"reader_form"`
	Handling      string  `json:"handling"`
}

// terminologyDecision is a request-local, source-safe handoff. It contains no
// source keys, locators, citation metadata, or source bodies and is never
// serialized as part of the semantic document.
type terminologyDecision struct {
	Category            string
	SourceForm          string
	SourceReading       string
	ReaderForm          string
	FirstUseExplanation string
	Handling            string
	HideSourceAliases   bool
	PreserveSourceForm  bool
}

func authorTerminologySchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required": []any{
			"category", "source_form", "source_reading", "reader_form", "handling",
		},
		"properties": map[string]any{
			"category": map[string]any{"type": "string", "enum": []any{
				"proper_name", "official_designation", "specialist_term", "foreign_unit",
			}},
			"source_form":    nonblankStringSchema(),
			"source_reading": nullableSchema(nonblankStringSchema()),
			"reader_form":    nonblankStringSchema(),
			"handling": map[string]any{"type": "string", "enum": []any{
				"preserve", "preserve_and_explain", "convert_and_explain",
			}},
		},
	}
}

func compileAuthorTerminology(
	drafts []authorTermDraft,
	document Document,
	catalog reportilcontract.SourceCatalog,
	receipt reportilcontract.SourceReadReceipt,
	readableByAcceptedOrdinal map[int]string,
) ([]terminologyDecision, error) {
	if len(drafts) > maxAuthorTerminologyTerms {
		return nil, withValidationCode(
			reportexecution.ProviderValidationCodeTerminologyInventory,
			fmt.Errorf("author terminology inventory is too large"),
		)
	}
	if len(drafts) > 0 && len(readableByAcceptedOrdinal) == 0 {
		return nil, withValidationCode(
			reportexecution.ProviderValidationCodeTerminologySourceGrounding,
			fmt.Errorf("author terminology lacks server source text for validation"),
		)
	}
	values := documentReaderValues(document)
	seenSourceForms := map[string]bool{}
	seenReaderForms := map[string]bool{}
	terms := make([]terminologyDecision, 0, len(drafts))
	for index, draft := range drafts {
		termAlias := fmt.Sprintf("term_%02d", index+1)
		term, err := validateAuthorTermDraft(draft, catalog, receipt, readableByAcceptedOrdinal)
		if err != nil {
			return nil, withValidationCodeAndTerm(
				reportexecution.ProviderValidationCodeTerminologyInventory,
				termAlias,
				fmt.Errorf("author terminology %s: %w", termAlias, err),
			)
		}
		if seenSourceForms[term.SourceForm] || seenReaderForms[term.ReaderForm] {
			return nil, withValidationCodeAndTerm(
				reportexecution.ProviderValidationCodeTerminologyInventory,
				termAlias,
				fmt.Errorf("author terminology contains a duplicate source or reader form"),
			)
		}
		seenSourceForms[term.SourceForm] = true
		seenReaderForms[term.ReaderForm] = true
		if !readerValuesContain(values, term.ReaderForm) {
			return nil, withValidationCodeAndTerm(
				reportexecution.ProviderValidationCodeTerminologyInventory,
				termAlias,
				fmt.Errorf("author terminology %s: reader form does not occur in the manuscript", termAlias),
			)
		}
		terms = append(terms, term)
	}
	if err := validateOriginalTerminologyAliases(terms); err != nil {
		return nil, withValidationCode(
			reportexecution.ProviderValidationCodeTerminologyAliasCollision,
			fmt.Errorf("author %w", err),
		)
	}
	if err := validateSourceScriptTerminologyCoverage(document, terms); err != nil {
		return nil, withValidationCode(
			reportexecution.ProviderValidationCodeTerminologyScriptCoverage,
			fmt.Errorf("author %w", err),
		)
	}
	return terms, nil
}

func validateSourceScriptTerminologyCoverage(document Document, terms []terminologyDecision) error {
	language := strings.ToLower(strings.TrimSpace(document.Language))
	if language != "ko" && !strings.HasPrefix(language, "ko-") {
		return nil
	}
	covered := map[string]bool{}
	for _, term := range terms {
		values := []string{term.ReaderForm}
		if !term.HideSourceAliases {
			values = append(values, term.SourceForm, term.SourceReading)
		}
		for _, value := range values {
			for _, token := range sourceScriptTermPattern.FindAllString(value, -1) {
				covered[token] = true
			}
		}
	}
	values := []string{document.Title}
	for _, block := range document.Blocks {
		values = append(values, authoredBlockText(block)...)
	}
	for _, value := range values {
		for _, token := range sourceScriptTermPattern.FindAllString(value, -1) {
			if !covered[token] {
				return fmt.Errorf("source-script manuscript term %q is missing from the terminology inventory", token)
			}
		}
	}
	return nil
}

func validateAuthorTermDraft(
	draft authorTermDraft,
	catalog reportilcontract.SourceCatalog,
	receipt reportilcontract.SourceReadReceipt,
	readableByAcceptedOrdinal map[int]string,
) (terminologyDecision, error) {
	term := terminologyDecision{
		Category:   strings.TrimSpace(draft.Category),
		SourceForm: strings.TrimSpace(draft.SourceForm),
		ReaderForm: strings.TrimSpace(draft.ReaderForm),
		Handling:   strings.TrimSpace(draft.Handling),
	}
	if draft.SourceReading != nil {
		term.SourceReading = strings.TrimSpace(*draft.SourceReading)
	}
	if !terminologyCategories[term.Category] || !terminologyHandling[term.Handling] {
		return terminologyDecision{}, fmt.Errorf("category or handling is invalid")
	}
	if term.Category == "foreign_unit" && term.Handling != "convert_and_explain" {
		return terminologyDecision{}, fmt.Errorf("foreign unit requires conversion and explanation")
	}
	if term.Category == "specialist_term" && term.Handling != "preserve_and_explain" {
		return terminologyDecision{}, fmt.Errorf("specialist term requires preservation and explanation")
	}
	if term.Category != "foreign_unit" && term.Handling == "convert_and_explain" {
		return terminologyDecision{}, fmt.Errorf("only a foreign unit may use conversion handling")
	}
	if term.SourceForm != term.ReaderForm && term.Handling == "preserve" {
		return terminologyDecision{}, fmt.Errorf("a changed source form requires an explanation")
	}
	if term.SourceReading != "" && term.Handling == "preserve" {
		return terminologyDecision{}, fmt.Errorf("a source-supplied reading requires an explanation")
	}
	for name, value := range map[string]string{
		"source form": term.SourceForm, "reader form": term.ReaderForm,
	} {
		if err := validateTerminologyValue(name, value, 256); err != nil {
			return terminologyDecision{}, err
		}
	}
	if term.SourceReading != "" {
		if err := validateTerminologyValue("source reading", term.SourceReading, 256); err != nil {
			return terminologyDecision{}, err
		}
	}
	formGrounded := false
	readingGrounded := term.SourceReading == ""
	for _, entry := range catalog.Sources {
		if !receipt.IncludesEntireSource(entry.SourceKey) {
			return terminologyDecision{}, withValidationCode(
				reportexecution.ProviderValidationCodeSourceReadContract,
				fmt.Errorf("selected source was not completely read"),
			)
		}
		readable, ok := readableByAcceptedOrdinal[entry.AcceptedOrdinal]
		if !ok {
			return terminologyDecision{}, withValidationCode(
				reportexecution.ProviderValidationCodeTerminologySourceGrounding,
				fmt.Errorf("server source text is unavailable for terminology grounding"),
			)
		}
		if !strings.Contains(readable, term.SourceForm) {
			continue
		}
		formGrounded = true
		if term.SourceReading != "" && sourceReadingNearForm(readable, term.SourceForm, term.SourceReading) {
			readingGrounded = true
		}
	}
	if !formGrounded {
		return terminologyDecision{}, withValidationCode(
			reportexecution.ProviderValidationCodeTerminologySourceFormGrounding,
			fmt.Errorf("source form is not grounded in a selected source"),
		)
	}
	if !readingGrounded {
		return terminologyDecision{}, withValidationCode(
			reportexecution.ProviderValidationCodeTerminologySourceReadingGrounding,
			fmt.Errorf("source reading is not grounded near its source form"),
		)
	}
	return term, nil
}

func sourceReadingNearForm(readable, sourceForm, sourceReading string) bool {
	for offset := 0; ; {
		index := strings.Index(readable[offset:], sourceForm)
		if index < 0 {
			return false
		}
		index += offset
		start := index - 128
		if start < 0 {
			start = 0
		}
		end := index + len(sourceForm) + 128
		if end > len(readable) {
			end = len(readable)
		}
		if strings.Contains(readable[start:end], sourceReading) {
			return true
		}
		offset = index + len(sourceForm)
		if offset >= len(readable) {
			return false
		}
	}
}

func validateTerminologyValue(name, value string, maxBytes int) error {
	if value == "" || len(value) > maxBytes || value != strings.TrimSpace(value) {
		return fmt.Errorf("%s is invalid", name)
	}
	if err := validateProviderContent(value); err != nil {
		return fmt.Errorf("%s contains restricted content", name)
	}
	return nil
}

func requiresSourceFormReaderForm(term terminologyDecision) bool {
	return term.Category == "specialist_term" &&
		term.SourceReading == "" &&
		sourceScriptTermPattern.MatchString(term.SourceForm)
}

func requiresSourceFormRemoval(language string, term terminologyDecision) bool {
	language = strings.ToLower(strings.TrimSpace(language))
	return (language == "ko" || strings.HasPrefix(language, "ko-")) &&
		requiresSourceFormReaderForm(term) &&
		!term.PreserveSourceForm
}

func termRequiresExplanation(term terminologyDecision) bool {
	return (!term.HideSourceAliases &&
		(term.SourceForm != term.ReaderForm || term.SourceReading != "")) ||
		term.Category == "specialist_term" ||
		term.Category == "foreign_unit" ||
		term.Handling != "preserve"
}

func validateTermPresentation(values []string, term terminologyDecision) error {
	if !readerValuesContain(values, term.ReaderForm) {
		return withValidationCode(
			reportexecution.ProviderValidationCodeTerminologyReaderForm,
			fmt.Errorf("reader form does not occur in the manuscript"),
		)
	}
	if termRequiresExplanation(term) && term.FirstUseExplanation == "" {
		return withValidationCode(
			reportexecution.ProviderValidationCodeTerminologyFirstUse,
			fmt.Errorf("specialist, unit, or explanatory handling lacks a first-use explanation"),
		)
	}
	if term.FirstUseExplanation != "" {
		firstBodyValue, ok := firstBodyTermValue(values, term.ReaderForm)
		if !ok || !strings.Contains(firstBodyValue, term.FirstUseExplanation) || !sameReaderSentence(firstBodyValue, term.ReaderForm, term.FirstUseExplanation) {
			return withValidationCode(
				reportexecution.ProviderValidationCodeTerminologyFirstUse,
				fmt.Errorf("first-use explanation is not colocated in the same sentence as the first body use"),
			)
		}
		if !term.HideSourceAliases &&
			term.SourceForm != term.ReaderForm &&
			!strings.Contains(term.FirstUseExplanation, term.SourceForm) {
			return withValidationCode(
				reportexecution.ProviderValidationCodeTerminologySourceForm,
				fmt.Errorf("first-use explanation does not identify the exact source form"),
			)
		}
		if !term.HideSourceAliases &&
			term.SourceReading != "" &&
			!strings.Contains(term.FirstUseExplanation, term.SourceReading) {
			return withValidationCode(
				reportexecution.ProviderValidationCodeTerminologySourceReading,
				fmt.Errorf("first-use explanation omits the source-supplied reading"),
			)
		}
	}
	return nil
}

func sameReaderSentence(value, readerForm, explanation string) bool {
	readerIndex := strings.Index(value, readerForm)
	explanationIndex := strings.Index(value, explanation)
	if readerIndex < 0 || explanationIndex < 0 {
		return false
	}
	readerStart, readerEnd := readerSentenceBounds(value, readerIndex)
	explanationStart, explanationEnd := readerSentenceBounds(value, explanationIndex)
	return readerStart == explanationStart && readerEnd == explanationEnd
}

const readerValueBodyPrefix = "\x00body\x00"

func documentReaderValues(document Document) []string {
	values := []string{strings.TrimSpace(document.Title)}
	for _, block := range document.Blocks {
		for _, value := range authoredBlockText(block) {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if block.Kind != "section" {
				value = readerValueBodyPrefix + value
			}
			values = append(values, value)
		}
	}
	result := values[:0]
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func firstBodyTermValue(values []string, term string) (string, bool) {
	for _, value := range values {
		if strings.HasPrefix(value, readerValueBodyPrefix) && strings.Contains(value, term) {
			return strings.TrimPrefix(value, readerValueBodyPrefix), true
		}
	}
	return "", false
}

func readerValueText(value string) string {
	return strings.TrimPrefix(value, readerValueBodyPrefix)
}

func compileReaderTerminology(document Document, edits map[string]readerTermDraft, original []terminologyDecision) ([]terminologyDecision, error) {
	if err := validateOriginalTerminologyAliases(original); err != nil {
		return nil, withValidationCode(
			reportexecution.ProviderValidationCodeTerminologyAliasCollision,
			err,
		)
	}
	if len(edits) != len(original) {
		return nil, withValidationCode(
			reportexecution.ProviderValidationCodeTerminologyInventory,
			fmt.Errorf("reader terminology inventory changed"),
		)
	}
	values := documentReaderValues(document)
	terms := make([]terminologyDecision, 0, len(original))
	seenReaderForms := map[string]bool{}
	for index, term := range original {
		edit, ok := edits[readerTermKey(index)]
		if !ok || edit.TermReceipt != readerTermReceipt(term) {
			return nil, withValidationCode(
				reportexecution.ProviderValidationCodeTerminologyInventory,
				fmt.Errorf("reader terminology inventory changed"),
			)
		}
		if edit.ReaderForm == nil {
			if edit.RetainAsTerminology || edit.PresentSourceAliases {
				return nil, withValidationCode(
					reportexecution.ProviderValidationCodeTerminologyInventory,
					fmt.Errorf("removed terminology has incompatible reader decisions"),
				)
			}
			for _, value := range termKnownForms(term) {
				if readerValuesContain(values, value) {
					return nil, withValidationCode(
						reportexecution.ProviderValidationCodeTerminologyRemoval,
						fmt.Errorf("removed terminology remains in the manuscript"),
					)
				}
			}
			continue
		}

		candidate := term
		candidate.ReaderForm = strings.TrimSpace(*edit.ReaderForm)
		candidate.FirstUseExplanation = ""
		candidate.HideSourceAliases = !edit.PresentSourceAliases
		if requiresSourceFormRemoval(document.Language, term) {
			return nil, withValidationCode(
				reportexecution.ProviderValidationCodeTerminologyReaderForm,
				fmt.Errorf("Korean reader manuscript must remove a source-script specialist term without a source-supplied reading"),
			)
		}
		if requiresSourceFormReaderForm(term) &&
			(!edit.RetainAsTerminology || candidate.ReaderForm != term.SourceForm) {
			return nil, withValidationCode(
				reportexecution.ProviderValidationCodeTerminologyReaderForm,
				fmt.Errorf("source-script specialist term without a source reading must retain its exact source form or be removed"),
			)
		}
		if !edit.RetainAsTerminology &&
			(term.Category == "specialist_term" ||
				term.Category == "foreign_unit" ||
				edit.PresentSourceAliases) {
			return nil, withValidationCode(
				reportexecution.ProviderValidationCodeTerminologyInventory,
				fmt.Errorf("unretained terminology may not be a specialist term, foreign unit, or source-alias presentation"),
			)
		}
		if err := validateTerminologyValue("reader form", candidate.ReaderForm, 256); err != nil {
			return nil, withValidationCode(
				reportexecution.ProviderValidationCodeTerminologyReaderForm,
				err,
			)
		}
		if seenReaderForms[candidate.ReaderForm] {
			return nil, withValidationCode(
				reportexecution.ProviderValidationCodeTerminologyAliasCollision,
				fmt.Errorf("reader terminology contains a duplicate reader form"),
			)
		}
		seenReaderForms[candidate.ReaderForm] = true
		if candidate.ReaderForm != term.ReaderForm && readerValuesContain(values, term.ReaderForm) {
			return nil, withValidationCode(
				reportexecution.ProviderValidationCodeTerminologyRename,
				fmt.Errorf("superseded reader form remains in the manuscript"),
			)
		}
		if candidate.HideSourceAliases {
			if candidate.ReaderForm == candidate.SourceForm ||
				(candidate.SourceReading != "" && candidate.ReaderForm == candidate.SourceReading) {
				return nil, withValidationCode(
					reportexecution.ProviderValidationCodeTerminologyAliasPlacement,
					fmt.Errorf("reader form is a source alias marked as hidden"),
				)
			}
			if err := validateSourceAliasesAbsent(values, candidate); err != nil {
				return nil, withValidationCode(
					reportexecution.ProviderValidationCodeTerminologyAliasPlacement,
					err,
				)
			}
		}
		if !edit.RetainAsTerminology {
			if !readerValuesContain(values, candidate.ReaderForm) {
				return nil, withValidationCode(
					reportexecution.ProviderValidationCodeTerminologyReaderForm,
					fmt.Errorf("reader form does not occur in the manuscript"),
				)
			}
			continue
		}
		if termRequiresExplanation(candidate) {
			candidate.FirstUseExplanation = firstBodyTermSentence(values, candidate.ReaderForm)
		}
		if err := validateTermPresentation(values, candidate); err != nil {
			return nil, err
		}
		if !candidate.HideSourceAliases {
			if err := validateSourceAliasPlacement(values, candidate); err != nil {
				return nil, withValidationCode(
					reportexecution.ProviderValidationCodeTerminologyAliasPlacement,
					err,
				)
			}
		}
		terms = append(terms, candidate)
	}
	if err := validateOriginalTerminologyAliases(terms); err != nil {
		return nil, withValidationCode(
			reportexecution.ProviderValidationCodeTerminologyAliasCollision,
			fmt.Errorf("reader %w", err),
		)
	}
	if err := validateSourceScriptTerminologyCoverage(document, terms); err != nil {
		return nil, withValidationCode(
			reportexecution.ProviderValidationCodeTerminologyScriptCoverage,
			fmt.Errorf("reader %w", err),
		)
	}
	return terms, nil
}

func firstBodyTermSentence(values []string, readerForm string) string {
	value, ok := firstBodyTermValue(values, readerForm)
	if !ok {
		return ""
	}
	index := strings.Index(value, readerForm)
	start, end := readerSentenceBounds(value, index)
	return strings.TrimSpace(value[start:end])
}

func validateSourceAliasesAbsent(values []string, term terminologyDecision) error {
	for _, alias := range []string{term.SourceForm, term.SourceReading} {
		if alias == "" || alias == term.ReaderForm {
			continue
		}
		if readerValuesContain(values, alias) {
			return fmt.Errorf("source form or reading remains after reader alias removal")
		}
	}
	return nil
}

func validateSourceAliasPlacement(values []string, term terminologyDecision) error {
	firstBodyIndex := -1
	firstSentenceStart, firstSentenceEnd := -1, -1
	for index, value := range values {
		if !strings.HasPrefix(value, readerValueBodyPrefix) {
			continue
		}
		text := readerValueText(value)
		readerIndex := strings.Index(text, term.ReaderForm)
		if readerIndex >= 0 {
			firstBodyIndex = index
			firstSentenceStart, firstSentenceEnd = readerSentenceBounds(text, readerIndex)
			break
		}
	}
	for _, alias := range []string{term.SourceForm, term.SourceReading} {
		if alias == "" || alias == term.ReaderForm {
			continue
		}
		for index, value := range values {
			text := readerValueText(value)
			for offset := 0; ; {
				aliasIndex := strings.Index(text[offset:], alias)
				if aliasIndex < 0 {
					break
				}
				aliasIndex += offset
				if index != firstBodyIndex || aliasIndex < firstSentenceStart || aliasIndex+len(alias) > firstSentenceEnd || !strings.Contains(text[firstSentenceStart:firstSentenceEnd], term.FirstUseExplanation) {
					return fmt.Errorf("source form or reading survives outside the explained first body sentence")
				}
				offset = aliasIndex + len(alias)
				if offset >= len(text) {
					break
				}
			}
		}
	}
	return nil
}

func readerSentenceBounds(value string, index int) (int, int) {
	start := 0
	for offset := 0; offset < index; {
		boundary := readerSentenceBoundaryAt(value, offset)
		if boundary > offset {
			start = boundary
			offset = boundary
			continue
		}
		offset++
	}
	end := len(value)
	for offset := index; offset < len(value); offset++ {
		if boundary := readerSentenceBoundaryAt(value, offset); boundary > offset {
			end = offset
			break
		}
	}
	return start, end
}

func readerSentenceBoundaryAt(value string, index int) int {
	if index < 0 || index >= len(value) {
		return index
	}
	if value[index] == '.' && index > 0 && index+1 < len(value) && value[index-1] >= '0' && value[index-1] <= '9' && value[index+1] >= '0' && value[index+1] <= '9' {
		return index
	}
	switch value[index] {
	case '.', '!', '?', '\n':
		return index + 1
	}
	for _, boundary := range []string{"。", "！", "？"} {
		if strings.HasPrefix(value[index:], boundary) {
			return index + len(boundary)
		}
	}
	return index
}

func readerValuesContain(values []string, needle string) bool {
	if needle == "" {
		return false
	}
	for _, value := range values {
		if strings.Contains(readerValueText(value), needle) {
			return true
		}
	}
	return false
}

func termKnownForms(term terminologyDecision) []string {
	forms := []string{term.ReaderForm, term.SourceForm, term.SourceReading}
	if term.FirstUseExplanation != "" {
		forms = append(forms, term.FirstUseExplanation)
	}
	return forms
}

func continuityAliases(term terminologyDecision) []string {
	seen := map[string]bool{}
	aliases := make([]string, 0, 3)
	values := []string{term.ReaderForm}
	if !term.HideSourceAliases {
		values = append(values, term.SourceForm, term.SourceReading)
	}
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			aliases = append(aliases, value)
		}
	}
	return aliases
}

func validateOriginalTerminologyAliases(terms []terminologyDecision) error {
	type ownedAlias struct {
		value string
		owner int
	}
	aliases := []ownedAlias{}
	for index, term := range terms {
		for _, alias := range continuityAliases(term) {
			for _, existing := range aliases {
				if existing.owner != index && (strings.Contains(alias, existing.value) || strings.Contains(existing.value, alias)) {
					return fmt.Errorf("terminology aliases overlap across different terms")
				}
			}
			aliases = append(aliases, ownedAlias{value: alias, owner: index})
		}
	}
	return nil
}

func continuityTerms(terms []terminologyDecision) []ContinuityTerm {
	result := make([]ContinuityTerm, 0, len(terms))
	for _, term := range terms {
		seen := map[string]bool{term.ReaderForm: true}
		aliases := make([]string, 0, 2)
		if !term.HideSourceAliases {
			for _, alias := range []string{term.SourceForm, term.SourceReading} {
				if alias != "" && !seen[alias] {
					seen[alias] = true
					aliases = append(aliases, alias)
				}
			}
		}
		result = append(result, ContinuityTerm{Canonical: term.ReaderForm, Aliases: aliases})
	}
	return result
}
