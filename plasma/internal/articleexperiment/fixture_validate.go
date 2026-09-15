package articleexperiment

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// ValidateFixtureArtifacts validates typed source, dossier, claim, and reader-test
// contracts before their digests can enter a real pilot protocol.
func ValidateFixtureArtifacts(archiveRoot, repositoryRoot, baseDir string, fixture RealFixture) error {
	catalog, err := loadFixtureArtifact[FixtureSourceCatalog](archiveRoot, repositoryRoot, baseDir, fixture.SourceCatalog)
	if err != nil {
		return err
	}
	dossier, err := loadFixtureArtifact[FixtureDossier](archiveRoot, repositoryRoot, baseDir, fixture.Dossier)
	if err != nil {
		return err
	}
	inventory, err := loadFixtureArtifact[FixtureClaimInventory](archiveRoot, repositoryRoot, baseDir, fixture.ClaimInventory)
	if err != nil {
		return err
	}
	memory, err := loadFixtureArtifact[FixtureMemoryQuestions](archiveRoot, repositoryRoot, baseDir, fixture.MemoryQuestions)
	if err != nil {
		return err
	}
	transfer, err := loadFixtureArtifact[FixtureTransferTask](archiveRoot, repositoryRoot, baseDir, fixture.TransferTask)
	if err != nil {
		return err
	}
	if catalog.SchemaVersion != SourceCatalogSchemaVersion || dossier.SchemaVersion != DossierSchemaVersion || inventory.SchemaVersion != ClaimInventorySchemaVersion || memory.SchemaVersion != MemoryQuestionsSchemaVersion || transfer.SchemaVersion != TransferTaskSchemaVersion || catalog.FixtureID != fixture.FixtureID || dossier.FixtureID != fixture.FixtureID || inventory.FixtureID != fixture.FixtureID || memory.FixtureID != fixture.FixtureID || transfer.FixtureID != fixture.FixtureID {
		return fmt.Errorf("%w: fixture artifact identity is invalid", producterror.ErrInvalidInput)
	}
	spans, err := validateFixtureSourceCatalog(archiveRoot, repositoryRoot, baseDir, catalog)
	if err != nil {
		return err
	}
	claims, err := validateFixtureClaimInventory(inventory, spans, fixture.MaterialTruthClaims)
	if err != nil {
		return err
	}
	if err := validateFixtureDossier(dossier, claims, spans, fixture.MaterialCaveats, fixture.SupportedConnections); err != nil {
		return err
	}
	if err := validateFixtureEvaluation(memory, transfer, claims); err != nil {
		return err
	}
	return nil
}

func loadFixtureArtifact[T any](archiveRoot, repositoryRoot, baseDir string, ref FileRef) (T, error) {
	var zero T
	path, content, err := loadRealReference(archiveRoot, repositoryRoot, baseDir, ref)
	if err != nil {
		return zero, err
	}
	value, _, err := decodeJSONBytes[T](path, content)
	return value, err
}

func validateFixtureSourceCatalog(archiveRoot, repositoryRoot, baseDir string, catalog FixtureSourceCatalog) (map[string]FixtureSourceSpan, error) {
	if len(catalog.Sources) != 1 || len(catalog.Spans) < 8 {
		return nil, fmt.Errorf("%w: fixture source catalog is incomplete", producterror.ErrInvalidInput)
	}
	source := catalog.Sources[0]
	if source.SourceKey != "source_001" || source.MediaType != "text/markdown; charset=utf-8" || source.RetrievalPolicy != "snapshot_only" || strings.TrimSpace(source.Extraction) == "" || len(source.Derivation) == 0 || !source.MediaExcluded || strings.TrimSpace(source.Rights.Agency) == "" || strings.TrimSpace(source.Rights.RightsURL) == "" || strings.TrimSpace(source.Rights.AIUse) == "" || len(source.Origins) == 0 {
		return nil, fmt.Errorf("%w: fixture source contract is invalid", producterror.ErrInvalidInput)
	}
	_, body, err := loadRealReference(archiveRoot, repositoryRoot, baseDir, source.Content)
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(body) {
		return nil, fmt.Errorf("%w: fixture source content is not UTF-8", producterror.ErrInvalidInput)
	}
	originIDs := map[string]bool{}
	originRaw := map[string][]byte{}
	for _, origin := range source.Origins {
		if !safeSlugPattern.MatchString(origin.OriginID) || strings.TrimSpace(origin.URL) == "" || originIDs[origin.OriginID] {
			return nil, fmt.Errorf("%w: fixture source origin is invalid", producterror.ErrInvalidInput)
		}
		originIDs[origin.OriginID] = true
		_, raw, err := loadRealReference(archiveRoot, repositoryRoot, baseDir, origin.Raw)
		if err != nil {
			return nil, err
		}
		originRaw[origin.OriginID] = raw
		for _, ref := range []FileRef{origin.Headers, origin.FinalURL, origin.Rights} {
			if _, _, err := loadRealReference(archiveRoot, repositoryRoot, baseDir, ref); err != nil {
				return nil, err
			}
		}
	}
	seenDerivation := map[string]bool{}
	for _, derivation := range source.Derivation {
		if !originIDs[derivation.OriginID] || strings.TrimSpace(derivation.OriginSpan) == "" || !validSHA256(derivation.SHA256) || bytesSHA256([]byte(derivation.OriginSpan)) != derivation.SHA256 || !bytes.Contains(originRaw[derivation.OriginID], []byte(derivation.OriginSpan)) || seenDerivation[derivation.OriginID+"/"+derivation.SHA256] {
			return nil, fmt.Errorf("%w: fixture source derivation is invalid", producterror.ErrInvalidInput)
		}
		seenDerivation[derivation.OriginID+"/"+derivation.SHA256] = true
	}
	spans := map[string]FixtureSourceSpan{}
	for _, span := range catalog.Spans {
		if !safeSlugPattern.MatchString(span.SpanID) || span.SourceKey != source.SourceKey || span.StartByte < 0 || span.EndByte <= span.StartByte || span.EndByte > len(body) || !validSHA256(span.SHA256) || spans[span.SpanID].SpanID != "" {
			return nil, fmt.Errorf("%w: fixture source span is invalid", producterror.ErrInvalidInput)
		}
		exact := body[span.StartByte:span.EndByte]
		if bytesSHA256(exact) != span.SHA256 {
			return nil, fmt.Errorf("%w: fixture source span differs", producterror.ErrConflict)
		}
		spans[span.SpanID] = span
	}
	return spans, nil
}

func validateFixtureClaimInventory(inventory FixtureClaimInventory, spans map[string]FixtureSourceSpan, expected int) (map[string]FixtureClaim, error) {
	if len(inventory.Claims) != expected || inventory.RequiredLeafCoverage != "all authored strings" || !sameStrings(inventory.Verdicts, []string{"directly_supported", "bounded_inference", "overstated", "contradicted", "unsupported"}) {
		return nil, fmt.Errorf("%w: fixture claim inventory is invalid", producterror.ErrInvalidInput)
	}
	claims := map[string]FixtureClaim{}
	for _, claim := range inventory.Claims {
		if !safeSlugPattern.MatchString(claim.ClaimID) || strings.TrimSpace(claim.SummaryKO) == "" || claim.SupportRelation != "direct_or_bounded" || claim.Uncertainty != "preserve source qualifiers" || claim.ClaimStrength != "material" || len(claim.SpanIDs) == 0 || claims[claim.ClaimID].ClaimID != "" {
			return nil, fmt.Errorf("%w: fixture claim is invalid", producterror.ErrInvalidInput)
		}
		for _, spanID := range claim.SpanIDs {
			if spans[spanID].SpanID == "" {
				return nil, fmt.Errorf("%w: fixture claim span is missing", producterror.ErrInvalidInput)
			}
		}
		claims[claim.ClaimID] = claim
	}
	return claims, nil
}

func validateFixtureDossier(dossier FixtureDossier, claims map[string]FixtureClaim, spans map[string]FixtureSourceSpan, caveatCount, connectionCount int) error {
	if !dossier.Sensitive || len(dossier.Accounts) != len(claims) || len(dossier.Caveats) != caveatCount || len(dossier.Connections) < connectionCount || len(dossier.Forbidden) == 0 {
		return fmt.Errorf("%w: fixture dossier is incomplete", producterror.ErrInvalidInput)
	}
	seenClaims := map[string]bool{}
	seenAccounts := map[string]bool{}
	for _, account := range dossier.Accounts {
		if !safeSlugPattern.MatchString(account.AccountID) || seenAccounts[account.AccountID] || strings.TrimSpace(account.Kind) == "" || len(account.ClaimIDs) == 0 || len(account.SpanIDs) == 0 {
			return fmt.Errorf("%w: fixture dossier account is invalid", producterror.ErrInvalidInput)
		}
		seenAccounts[account.AccountID] = true
		accountSpanIDs := map[string]bool{}
		for _, spanID := range account.SpanIDs {
			if spans[spanID].SpanID == "" {
				return fmt.Errorf("%w: fixture dossier span is missing", producterror.ErrInvalidInput)
			}
			accountSpanIDs[spanID] = true
		}
		for _, claimID := range account.ClaimIDs {
			claim := claims[claimID]
			if claim.ClaimID == "" || seenClaims[claimID] {
				return fmt.Errorf("%w: fixture dossier claim coverage is invalid", producterror.ErrInvalidInput)
			}
			for _, spanID := range claim.SpanIDs {
				if !accountSpanIDs[spanID] {
					return fmt.Errorf("%w: fixture dossier account does not bind its claim spans", producterror.ErrInvalidInput)
				}
			}
			seenClaims[claimID] = true
		}
	}
	seenCaveats := map[string]bool{}
	for _, caveat := range dossier.Caveats {
		if !safeSlugPattern.MatchString(caveat.CaveatID) || seenCaveats[caveat.CaveatID] || strings.TrimSpace(caveat.SummaryKO) == "" || len(caveat.SpanIDs) == 0 {
			return fmt.Errorf("%w: fixture caveat is invalid", producterror.ErrInvalidInput)
		}
		seenCaveats[caveat.CaveatID] = true
		for _, spanID := range caveat.SpanIDs {
			if spans[spanID].SpanID == "" {
				return fmt.Errorf("%w: fixture caveat span is missing", producterror.ErrInvalidInput)
			}
		}
	}
	seenConnections := map[string]bool{}
	for _, connection := range dossier.Connections {
		if !safeSlugPattern.MatchString(connection.ConnectionID) || seenConnections[connection.ConnectionID] || strings.TrimSpace(connection.Kind) == "" || strings.TrimSpace(connection.SummaryKO) == "" || len(connection.FromClaimIDs) == 0 || len(connection.ToClaimIDs) == 0 || !allClaimsExist(claims, append(append([]string{}, connection.FromClaimIDs...), connection.ToClaimIDs...)) {
			return fmt.Errorf("%w: fixture connection is invalid", producterror.ErrInvalidInput)
		}
		seenConnections[connection.ConnectionID] = true
	}
	return nil
}

func validateFixtureEvaluation(memory FixtureMemoryQuestions, transfer FixtureTransferTask, claims map[string]FixtureClaim) error {
	if len(memory.Questions) < 1 || strings.TrimSpace(transfer.TaskKO) == "" || strings.TrimSpace(transfer.Scoring) == "" || len(transfer.RequiredClaimIDs) == 0 || !allClaimsExist(claims, transfer.RequiredClaimIDs) {
		return fmt.Errorf("%w: fixture evaluation contract is invalid", producterror.ErrInvalidInput)
	}
	seen := map[string]bool{}
	for _, question := range memory.Questions {
		if !safeSlugPattern.MatchString(question.QuestionID) || seen[question.QuestionID] || strings.TrimSpace(question.QuestionKO) == "" || strings.TrimSpace(question.Scoring) == "" || len(question.RequiredClaimIDs) == 0 || !allClaimsExist(claims, question.RequiredClaimIDs) {
			return fmt.Errorf("%w: fixture memory question is invalid", producterror.ErrInvalidInput)
		}
		seen[question.QuestionID] = true
	}
	return nil
}

func allClaimsExist(claims map[string]FixtureClaim, ids []string) bool {
	for _, id := range ids {
		if claims[id].ClaimID == "" {
			return false
		}
	}
	return true
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
